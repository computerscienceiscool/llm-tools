package sandbox

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/strslice"
	"github.com/docker/docker/client"
)

// ContainerConfig holds configuration for running a container
type ContainerConfig struct {
	Image       string
	Command     string
	RepoRoot    string
	MemoryLimit string
	CPULimit    int
	Timeout     time.Duration
	Stdin       string // NEW: stdin content to pass to container
}

// ContainerResult holds the result of container execution
type ContainerResult struct {
	ExitCode int
	Stdout   string
	Stderr   string
	Duration time.Duration
}

// RunContainer executes a command in a Docker container with security restrictions
func RunContainer(cfg ContainerConfig) (ContainerResult, error) {
	startTime := time.Now()
	result := ContainerResult{}

	// Create temporary directory for container writes
	tempDir, err := os.MkdirTemp("", "llm-exec-")
	if err != nil {
		return result, fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(tempDir)

	// Create Docker client
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return result, fmt.Errorf("failed to create Docker client: %w", err)
	}
	defer cli.Close()

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), cfg.Timeout)
	defer cancel()

	// Configure container
	containerConfig := &container.Config{
		Image:      cfg.Image,
		Cmd:        strslice.StrSlice{"sh", "-c", cfg.Command},
		WorkingDir: "/workspace",
		User:       "1000:1000",
	}

	// Enable stdin if provided
	if cfg.Stdin != "" {
		containerConfig.OpenStdin = true
		containerConfig.AttachStdin = true
		containerConfig.StdinOnce = true
	}

	// Configure host (mounts, resources, security)
	hostConfig := &container.HostConfig{
		NetworkMode: "none",
		Resources: container.Resources{
			Memory:   parseMemoryLimit(cfg.MemoryLimit),
			NanoCPUs: int64(cfg.CPULimit) * 1000000000,
		},
		Mounts: []mount.Mount{
			{
				Type:     mount.TypeBind,
				Source:   cfg.RepoRoot,
				Target:   "/workspace",
				ReadOnly: true,
			},
			{
				Type:   mount.TypeBind,
				Source: tempDir,
				Target: "/tmp/workspace",
			},
		},
		CapDrop:        strslice.StrSlice{"ALL"},
		SecurityOpt:    []string{"no-new-privileges"},
		ReadonlyRootfs: true,
		Tmpfs: map[string]string{
			"/tmp":    "exec",
			"/.cache": "",
			"/go":     "",
		},
	}

	// Create container
	resp, err := cli.ContainerCreate(ctx, containerConfig, hostConfig, nil, nil, "")
	if err != nil {
		return result, fmt.Errorf("failed to create container: %w", err)
	}
	defer cli.ContainerRemove(ctx, resp.ID, types.ContainerRemoveOptions{Force: true})

	// Attach stdin if provided
	var hijackedResp types.HijackedResponse
	if cfg.Stdin != "" {
		attachOpts := types.ContainerAttachOptions{
			Stream: true,
			Stdin:  true,
			Stdout: true,
			Stderr: true,
		}
		hijackedResp, err = cli.ContainerAttach(ctx, resp.ID, attachOpts)
		if err != nil {
			return result, fmt.Errorf("failed to attach to container: %w", err)
		}
		defer hijackedResp.Close()
	}

	// Start container
	if err := cli.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{}); err != nil {
		return result, fmt.Errorf("failed to start container: %w", err)
	}

	// Write stdin if provided
	if cfg.Stdin != "" {
		_, err = hijackedResp.Conn.Write([]byte(cfg.Stdin))
		if err != nil {
			return result, fmt.Errorf("failed to write stdin: %w", err)
		}
		hijackedResp.CloseWrite()
	}

	// Wait for container to finish
	statusCh, errCh := cli.ContainerWait(ctx, resp.ID, container.WaitConditionNotRunning)
	select {
	case err := <-errCh:
		if err != nil {
			return result, fmt.Errorf("error waiting for container: %w", err)
		}
	case status := <-statusCh:
		result.ExitCode = int(status.StatusCode)
	case <-ctx.Done():
		result.ExitCode = 124 // Standard timeout exit code
		return result, fmt.Errorf("command timed out after %v", cfg.Timeout)
	}

	// Get container logs
	logReader, err := cli.ContainerLogs(ctx, resp.ID, types.ContainerLogsOptions{
		ShowStdout: true,
		ShowStderr: true,
	})
	if err != nil {
		return result, fmt.Errorf("failed to get container logs: %w", err)
	}
	defer logReader.Close()

	// Read stdout and stderr
	var stdout, stderr strings.Builder
	if err := demuxLogs(logReader, &stdout, &stderr); err != nil {
		return result, fmt.Errorf("failed to read container logs: %w", err)
	}

	result.Stdout = stdout.String()
	result.Stderr = stderr.String()
	result.Duration = time.Since(startTime)

	if result.ExitCode != 0 {
		return result, fmt.Errorf("command exited with code %d", result.ExitCode)
	}

	return result, nil
}

// parseMemoryLimit converts memory limit string (e.g., "512m") to bytes
func parseMemoryLimit(limit string) int64 {
	if limit == "" {
		return 0
	}
	// Simple parser for common formats
	if strings.HasSuffix(limit, "m") || strings.HasSuffix(limit, "M") {
		var mb int64
		fmt.Sscanf(limit, "%d", &mb)
		return mb * 1024 * 1024
	}
	if strings.HasSuffix(limit, "g") || strings.HasSuffix(limit, "G") {
		var gb int64
		fmt.Sscanf(limit, "%d", &gb)
		return gb * 1024 * 1024 * 1024
	}
	return 0
}

// demuxLogs separates stdout and stderr from Docker logs stream
func demuxLogs(reader io.Reader, stdout, stderr io.Writer) error {
	// Docker multiplexes stdout/stderr with 8-byte headers
	// Header format: [stream_type, 0, 0, 0, size1, size2, size3, size4]
	// stream_type: 1=stdout, 2=stderr
	buf := make([]byte, 8)
	for {
		_, err := io.ReadFull(reader, buf)
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}

		streamType := buf[0]
		size := int(buf[4])<<24 | int(buf[5])<<16 | int(buf[6])<<8 | int(buf[7])

		payload := make([]byte, size)
		_, err = io.ReadFull(reader, payload)
		if err != nil {
			return err
		}

		switch streamType {
		case 1: // stdout
			stdout.Write(payload)
		case 2: // stderr
			stderr.Write(payload)
		}
	}
}

package sandbox

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/strslice"
	"github.com/docker/docker/client"
)

// RunIOContainer executes a containerized I/O operation
func RunIOContainer(repoRoot, containerImage, command string, timeout time.Duration, memLimit string, cpuLimit int) (string, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return "", fmt.Errorf("failed to create Docker client: %w", err)
	}
	defer cli.Close()

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// Configure container
	containerConfig := &container.Config{
		Image:      containerImage,
		Cmd:        strslice.StrSlice{"/bin/sh", "-c", command},
		WorkingDir: "/workspace",
		User:       "1000:1000",
	}

	// Configure host
	hostConfig := &container.HostConfig{
		NetworkMode: "none",
		Resources: container.Resources{
			Memory:   parseMemoryLimitIO(memLimit),
			NanoCPUs: int64(cpuLimit) * 1000000000,
		},
		Mounts: []mount.Mount{
			{
				Type:     mount.TypeBind,
				Source:   repoRoot,
				Target:   "/workspace",
				ReadOnly: true,
			},
		},
		CapDrop:     strslice.StrSlice{"ALL"},
		SecurityOpt: []string{"no-new-privileges"},
	}

	// Create container
	resp, err := cli.ContainerCreate(ctx, containerConfig, hostConfig, nil, nil, "")
	if err != nil {
		return "", fmt.Errorf("failed to create container: %w", err)
	}
	defer cli.ContainerRemove(ctx, resp.ID, types.ContainerRemoveOptions{Force: true})

	// Start container
	if err := cli.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{}); err != nil {
		return "", fmt.Errorf("failed to start container: %w", err)
	}

	// Wait for completion
	statusCh, errCh := cli.ContainerWait(ctx, resp.ID, container.WaitConditionNotRunning)
	select {
	case err := <-errCh:
		if err != nil {
			return "", fmt.Errorf("container execution failed: %w", err)
		}
	case <-statusCh:
		// Container finished
	case <-ctx.Done():
		return "", fmt.Errorf("I/O operation timed out after %v", timeout)
	}

	// Get logs
	logReader, err := cli.ContainerLogs(ctx, resp.ID, types.ContainerLogsOptions{
		ShowStdout: true,
		ShowStderr: true,
	})
	if err != nil {
		return "", fmt.Errorf("failed to get container logs: %w", err)
	}
	defer logReader.Close()

	// Read output
	var stdout strings.Builder
	if err := readDockerLogs(logReader, &stdout); err != nil {
		return "", fmt.Errorf("failed to read container output: %w", err)
	}

	return stdout.String(), nil
}

// ReadFileInContainer reads a file using the I/O container
func ReadFileInContainer(filePath, repoRoot, containerImage string, timeout time.Duration, memLimit string, cpuLimit int) (string, error) {
	// Make path relative to repo root for container
	relPath, err := filepath.Rel(repoRoot, filePath)
	if err != nil {
		return "", fmt.Errorf("failed to get relative path: %w", err)
	}

	command := fmt.Sprintf("cat /workspace/%s", relPath)
	return RunIOContainer(repoRoot, containerImage, command, timeout, memLimit, cpuLimit)
}

// WriteFileInContainer writes a file using the I/O container
func WriteFileInContainer(filePath, content, repoRoot, containerImage string, timeout time.Duration, memLimit string, cpuLimit int) error {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return fmt.Errorf("failed to create Docker client: %w", err)
	}
	defer cli.Close()

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	relPath, err := filepath.Rel(repoRoot, filePath)
	if err != nil {
		return fmt.Errorf("failed to get relative path: %w", err)
	}

	// Write to temp file first, then move (atomic)
	// Directory creation happens inside container
	command := fmt.Sprintf("mkdir -p $(dirname /workspace/%s) && printf '%%s' %q > /workspace/%s.tmp && mv /workspace/%s.tmp /workspace/%s",
		relPath, content, relPath, relPath, relPath)

	// Configure container with read-write mount
	containerConfig := &container.Config{
		Image:      containerImage,
		Cmd:        strslice.StrSlice{"/bin/sh", "-c", command},
		WorkingDir: "/workspace",
		User:       "1000:1000",
	}

	hostConfig := &container.HostConfig{
		NetworkMode: "none",
		Resources: container.Resources{
			Memory:   parseMemoryLimitIO(memLimit),
			NanoCPUs: int64(cpuLimit) * 1000000000,
		},
		Mounts: []mount.Mount{
			{
				Type:     mount.TypeBind,
				Source:   repoRoot,
				Target:   "/workspace",
				ReadOnly: false, // Read-write for writes
			},
		},
		CapDrop:     strslice.StrSlice{"ALL"},
		SecurityOpt: []string{"no-new-privileges"},
	}

	// Create container
	resp, err := cli.ContainerCreate(ctx, containerConfig, hostConfig, nil, nil, "")
	if err != nil {
		return fmt.Errorf("failed to create container: %w", err)
	}
	defer cli.ContainerRemove(ctx, resp.ID, types.ContainerRemoveOptions{Force: true})

	// Start container
	if err := cli.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{}); err != nil {
		return fmt.Errorf("failed to start container: %w", err)
	}

	// Wait for completion
	statusCh, errCh := cli.ContainerWait(ctx, resp.ID, container.WaitConditionNotRunning)
	select {
	case err := <-errCh:
		if err != nil {
			return fmt.Errorf("container write failed: %w", err)
		}
	case <-statusCh:
		// Container finished
	case <-ctx.Done():
		return fmt.Errorf("write operation timed out after %v", timeout)
	}

	return nil
}

// EnsureIOContainerImage verifies the I/O container image exists
func EnsureIOContainerImage(imageName string) error {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return fmt.Errorf("failed to create Docker client: %w", err)
	}
	defer cli.Close()

	ctx := context.Background()
	_, _, err = cli.ImageInspectWithRaw(ctx, imageName)
	if err != nil {
		return fmt.Errorf("I/O container image not found: %s\nRun: docker build -f Dockerfile.io -t %s .", imageName, imageName)
	}
	return nil
}

// ValidateIOContainer runs pre-flight checks for containerized I/O
func ValidateIOContainer(repoRoot, containerImage string) error {
	// Check Docker is available
	if err := CheckDockerAvailability(); err != nil {
		return fmt.Errorf("Docker not available: %w", err)
	}

	// Check image exists
	if err := EnsureIOContainerImage(containerImage); err != nil {
		return err
	}

	// Check repo root exists and is readable
	if _, err := os.Stat(repoRoot); err != nil {
		return fmt.Errorf("repository root not accessible: %w", err)
	}

	return nil
}

// parseMemoryLimitIO converts memory limit string to bytes (for io_container)
func parseMemoryLimitIO(limit string) int64 {
	if limit == "" {
		return 0
	}
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

// readDockerLogs reads Docker logs and extracts stdout
func readDockerLogs(reader io.Reader, stdout io.Writer) error {
	buf := make([]byte, 8)
	for {
		_, err := io.ReadFull(reader, buf)
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}

		size := int(buf[4])<<24 | int(buf[5])<<16 | int(buf[6])<<8 | int(buf[7])
		payload := make([]byte, size)
		_, err = io.ReadFull(reader, payload)
		if err != nil {
			return err
		}

		stdout.Write(payload)
	}
}

// ReadFileInContainerPooled reads a file using a pooled container
func ReadFileInContainerPooled(ctx context.Context, pool *ContainerPool, filePath, repoRoot string) (string, error) {
	if pool == nil {
		// Fallback to non-pooled version
		return ReadFileInContainer(filePath, repoRoot, "llm-runtime-io:latest", 60*time.Second, "256m", 1)
	}

	relPath, err := filepath.Rel(repoRoot, filePath)
	if err != nil {
		return "", fmt.Errorf("failed to get relative path: %w", err)
	}

	command := fmt.Sprintf("cat /workspace/%s", relPath)
	return ExecuteInPooledContainer(ctx, pool, command, repoRoot)
}

// WriteFileInContainerPooled writes a file using a pooled container
func WriteFileInContainerPooled(ctx context.Context, pool *ContainerPool, filePath, content, repoRoot string) error {
	if pool == nil {
		// Fallback to non-pooled version
		return WriteFileInContainer(filePath, content, repoRoot, "llm-runtime-io:latest", 60*time.Second, "256m", 1)
	}

	relPath, err := filepath.Rel(repoRoot, filePath)
	if err != nil {
		return fmt.Errorf("failed to get relative path: %w", err)
	}

	// Atomic write: write to temp file then move
	command := fmt.Sprintf("mkdir -p $(dirname /workspace/%s) && printf '%%s' %q > /workspace/%s.tmp && mv /workspace/%s.tmp /workspace/%s",
		relPath, content, relPath, relPath, relPath)

	_, err = ExecuteInPooledContainer(ctx, pool, command, repoRoot)
	return err
}

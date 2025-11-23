package infrastructure

import (
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// DefaultDockerClient implements DockerClient
type DefaultDockerClient struct{}

// NewDockerClient creates a new Docker client
func NewDockerClient() DockerClient {
	return &DefaultDockerClient{}
}

// CheckAvailability verifies that Docker is installed and accessible
func (d *DefaultDockerClient) CheckAvailability() error {
	cmd := exec.Command("docker", "version")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("Docker not available: %w", err)
	}
	return nil
}

// PullImage ensures the required image is available
func (d *DefaultDockerClient) PullImage(image string) error {
	// Check if image exists locally first
	cmd := exec.Command("docker", "image", "inspect", image)
	if err := cmd.Run(); err == nil {
		return nil // Image exists
	}

	// Pull the image
	cmd = exec.Command("docker", "pull", image)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to pull Docker image: %w\n%s", err, output)
	}

	return nil
}

// ExecuteInContainer executes a command in a Docker container
func (d *DefaultDockerClient) ExecuteInContainer(ctx context.Context, config ContainerConfig) (ContainerResult, error) {
	startTime := time.Now()
	
	result := ContainerResult{
		Duration: time.Since(startTime),
	}

	// Pull image if needed
	if err := d.PullImage(config.Image); err != nil {
		return result, fmt.Errorf("failed to pull image: %w", err)
	}

	// Prepare Docker command
	dockerArgs := []string{
		"run",
		"--rm", // Remove container when done
	}

	// Add network configuration
	if config.NetworkMode != "" {
		dockerArgs = append(dockerArgs, "--network", config.NetworkMode)
	}

	// Add working directory
	if config.WorkDir != "" {
		dockerArgs = append(dockerArgs, "--workdir", config.WorkDir)
	}

	// Add memory limit
	if config.Memory != "" {
		dockerArgs = append(dockerArgs, "--memory", config.Memory)
	}

	// Add CPU limit
	if config.CPULimit > 0 {
		dockerArgs = append(dockerArgs, "--cpus", strconv.Itoa(config.CPULimit))
	}

	// Add mounts
	for _, mount := range config.Mounts {
		mountStr := fmt.Sprintf("%s:%s", mount.Source, mount.Target)
		if mount.ReadOnly {
			mountStr += ":ro"
		}
		dockerArgs = append(dockerArgs, "-v", mountStr)
	}

	// Add security options
	dockerArgs = append(dockerArgs,
		"--user", "1000:1000", // Run as non-root
		"--cap-drop", "ALL",   // Drop all capabilities
		"--security-opt", "no-new-privileges", // Prevent privilege escalation
		"--read-only",     // Make root filesystem read-only
		"--tmpfs", "/tmp", // Temporary filesystem for /tmp
	)

	// Add image and command
	dockerArgs = append(dockerArgs, config.Image)
	dockerArgs = append(dockerArgs, config.Command...)

	// Create command with context for timeout
	cmd := exec.CommandContext(ctx, "docker", dockerArgs...)

	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	result.Duration = time.Since(startTime)
	result.Stdout = stdout.String()
	result.Stderr = stderr.String()

	if ctx.Err() == context.DeadlineExceeded {
		result.ExitCode = 124 // Standard timeout exit code
		return result, fmt.Errorf("command timed out after %v", config.Timeout)
	}

	if err != nil {
		// Try to get exit code
		if exitError, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitError.ExitCode()
		} else {
			result.ExitCode = 1
		}
		return result, fmt.Errorf("docker execution failed: %w", err)
	}

	result.ExitCode = 0
	return result, nil
}

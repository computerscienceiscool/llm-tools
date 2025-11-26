package docker

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

// ContainerConfig holds configuration for running a container
type ContainerConfig struct {
	Image       string
	Command     string
	RepoRoot    string
	MemoryLimit string
	CPULimit    int
	Timeout     time.Duration
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

	// Prepare Docker command
	dockerArgs := []string{
		"run",
		"--rm",              // Remove container when done
		"--network", "none", // No network access
		"--workdir", "/workspace", // Set working directory
		"--memory", cfg.MemoryLimit, // Memory limit
		"--cpus", fmt.Sprintf("%d", cfg.CPULimit), // CPU limit
		"-v", fmt.Sprintf("%s:/workspace:ro", cfg.RepoRoot), // Mount repo read-only
		"-v", fmt.Sprintf("%s:/tmp/workspace:rw", tempDir), // Mount temp for writes
		"--user", "1000:1000", // Run as non-root
	}

	// Add security options
	dockerArgs = append(dockerArgs,
		"--cap-drop", "ALL", // Drop all capabilities
		"--security-opt", "no-new-privileges", // Prevent privilege escalation
		"--read-only",     // Make root filesystem read-only
		"--tmpfs", "/tmp", // Temporary filesystem for /tmp
	)

	// Add image and command
	dockerArgs = append(dockerArgs, cfg.Image)
	dockerArgs = append(dockerArgs, "sh", "-c", cfg.Command)

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), cfg.Timeout)
	defer cancel()

	// Execute command
	cmd := exec.CommandContext(ctx, "docker", dockerArgs...)

	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()

	result.Stdout = stdout.String()
	result.Stderr = stderr.String()
	result.Duration = time.Since(startTime)

	if ctx.Err() == context.DeadlineExceeded {
		result.ExitCode = 124 // Standard timeout exit code
		return result, fmt.Errorf("command timed out after %v", cfg.Timeout)
	}

	if err != nil {
		// Try to get exit code
		if exitError, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitError.ExitCode()
			return result, fmt.Errorf("command exited with code %d", result.ExitCode)
		}
		return result, fmt.Errorf("execution error: %w", err)
	}

	result.ExitCode = 0
	return result, nil
}

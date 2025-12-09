package sandbox

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// RunIOContainer executes a containerized I/O operation
func RunIOContainer(repoRoot, containerImage, command string, timeout time.Duration, memLimit string, cpuLimit int) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	dockerArgs := []string{
		"run",
		"--rm",
		"--network", "none",
		"--memory", memLimit,
		"--cpus", fmt.Sprintf("%d", cpuLimit),
		"--cap-drop", "ALL",
		"--security-opt", "no-new-privileges",
		"--user", "1000:1000",
		"-v", fmt.Sprintf("%s:/workspace:ro", repoRoot),
		containerImage,
		"sh", "-c", command,
	}

	cmd := exec.CommandContext(ctx, "docker", dockerArgs...)
	output, err := cmd.CombinedOutput()

	if ctx.Err() == context.DeadlineExceeded {
		return "", fmt.Errorf("I/O operation timed out after %v", timeout)
	}

	if err != nil {
		return "", fmt.Errorf("container execution failed: %w\nOutput: %s", err, string(output))
	}

	return string(output), nil
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
	// For writes, we need read-write mount
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	relPath, err := filepath.Rel(repoRoot, filePath)
	if err != nil {
		return fmt.Errorf("failed to get relative path: %w", err)
	}

	// Create directory if needed (on host, before container)
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Write to temp file first, then move (atomic)
	command := fmt.Sprintf("printf '%%s' %q > /workspace/%s && mv /workspace/%s /workspace/%s",
		content, relPath+".tmp", relPath+".tmp", relPath)

	dockerArgs := []string{
		"run",
		"--rm",
		"--network", "none",
		"--memory", memLimit,
		"--cpus", fmt.Sprintf("%d", cpuLimit),
		"--cap-drop", "ALL",
		"--security-opt", "no-new-privileges",
		"--user", "1000:1000",
		"-v", fmt.Sprintf("%s:/workspace:rw", repoRoot), // Read-write for writes
		containerImage,
		"sh", "-c", command,
	}

	cmd := exec.CommandContext(ctx, "docker", dockerArgs...)
	output, err := cmd.CombinedOutput()

	if ctx.Err() == context.DeadlineExceeded {
		return fmt.Errorf("write operation timed out after %v", timeout)
	}

	if err != nil {
		return fmt.Errorf("container write failed: %w\nOutput: %s", err, string(output))
	}

	return nil
}

// EnsureIOContainerImage verifies the I/O container image exists
func EnsureIOContainerImage(imageName string) error {
	cmd := exec.Command("docker", "image", "inspect", imageName)
	if err := cmd.Run(); err != nil {
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

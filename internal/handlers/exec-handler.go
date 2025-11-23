package handlers

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/computerscienceiscool/llm-tools/internal/infrastructure"
)

// DefaultExecHandler implements ExecHandler
type DefaultExecHandler struct {
	dockerClient infrastructure.DockerClient
}

// NewExecHandler creates a new exec handler
func NewExecHandler(dockerClient infrastructure.DockerClient) ExecHandler {
	return &DefaultExecHandler{
		dockerClient: dockerClient,
	}
}

// ExecuteCommand executes a command in a Docker container
func (h *DefaultExecHandler) ExecuteCommand(command string, config ExecConfig) (ExecResult, error) {
	result := ExecResult{}

	// Validate command
	if err := h.validateExecCommand(command, config); err != nil {
		return result, fmt.Errorf("EXEC_VALIDATION: %w", err)
	}

	// Check Docker availability
	if err := h.dockerClient.CheckAvailability(); err != nil {
		return result, fmt.Errorf("DOCKER_UNAVAILABLE: %w", err)
	}

	// Create temporary directory for container writes
	tempDir, err := os.MkdirTemp("", "llm-exec-")
	if err != nil {
		return result, fmt.Errorf("TEMP_DIR: %w", err)
	}
	defer os.RemoveAll(tempDir)

	// Prepare container configuration
	containerConfig := infrastructure.ContainerConfig{
		Image:   config.ContainerImage,
		Command: []string{"sh", "-c", command},
		WorkDir: "/workspace",
		Mounts: []infrastructure.Mount{
			{
				Source:   config.RepoRoot,
				Target:   "/workspace",
				ReadOnly: true,
			},
			{
				Source:   tempDir,
				Target:   "/tmp/workspace",
				ReadOnly: false,
			},
		},
		Memory:      config.MemoryLimit,
		CPULimit:    config.CPULimit,
		Timeout:     config.Timeout,
		NetworkMode: "none", // No network access
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), config.Timeout)
	defer cancel()

	// Execute command
	containerResult, err := h.dockerClient.ExecuteInContainer(ctx, containerConfig)

	if ctx.Err() == context.DeadlineExceeded {
		return result, fmt.Errorf("EXEC_TIMEOUT: command timed out after %v", config.Timeout)
	}

	if err != nil {
		return result, fmt.Errorf("EXEC_ERROR: %w", err)
	}

	result.ExitCode = containerResult.ExitCode
	result.Stdout = containerResult.Stdout
	result.Stderr = containerResult.Stderr
	result.Duration = containerResult.Duration

	if containerResult.ExitCode != 0 {
		return result, fmt.Errorf("EXEC_FAILED: command exited with code %d", containerResult.ExitCode)
	}

	return result, nil
}

// validateExecCommand checks if the command is whitelisted
func (h *DefaultExecHandler) validateExecCommand(command string, config ExecConfig) error {
	if !config.Enabled {
		return fmt.Errorf("exec command is disabled")
	}

	if len(config.Whitelist) == 0 {
		return fmt.Errorf("no commands are whitelisted")
	}

	commandParts := strings.Fields(command)
	if len(commandParts) == 0 {
		return fmt.Errorf("empty command")
	}

	baseCommand := commandParts[0]

	// Check against whitelist
	for _, allowed := range config.Whitelist {
		if allowed == baseCommand || strings.HasPrefix(command, allowed) {
			return nil
		}
	}

	return fmt.Errorf("command not in whitelist: %s", baseCommand)
}

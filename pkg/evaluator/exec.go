package evaluator

import (
	"fmt"
	"time"

	"github.com/computerscienceiscool/llm-runtime/internal/config"
	"github.com/computerscienceiscool/llm-runtime/pkg/sandbox"
	"github.com/computerscienceiscool/llm-runtime/pkg/scanner"
)

// ExecuteExec handles the "exec" command
func ExecuteExec(cmd scanner.Command, cfg *config.Config, auditLog func(cmdType, arg string, success bool, errMsg string)) scanner.ExecutionResult {
	startTime := time.Now()
	result := scanner.ExecutionResult{
		Command: cmd,
	}

	// Validate command
	if err := sandbox.ValidateExecCommand(cmd.Argument, cfg.ExecEnabled, cfg.ExecWhitelist); err != nil {
		result.Success = false
		result.Error = fmt.Errorf("EXEC_VALIDATION: %w", err)
		result.ExecutionTime = time.Since(startTime)
		if auditLog != nil {
			auditLog("exec", cmd.Argument, false, result.Error.Error())
		}
		return result
	}

	// Check Docker availability
	if err := sandbox.CheckDockerAvailability(); err != nil {
		result.Success = false
		result.Error = fmt.Errorf("DOCKER_UNAVAILABLE: %w", err)
		result.ExecutionTime = time.Since(startTime)
		if auditLog != nil {
			auditLog("exec", cmd.Argument, false, result.Error.Error())
		}
		return result
	}

	// Pull Docker image if needed
	if err := sandbox.PullDockerImage(cfg.ExecContainerImage, cfg.Verbose); err != nil {
		result.Success = false
		result.Error = fmt.Errorf("DOCKER_IMAGE: %w", err)
		result.ExecutionTime = time.Since(startTime)
		if auditLog != nil {
			auditLog("exec", cmd.Argument, false, result.Error.Error())
		}
		return result
	}

	// Configure and run container
	containerCfg := sandbox.ContainerConfig{
		Image:       cfg.ExecContainerImage,
		Command:     cmd.Argument,
		RepoRoot:    cfg.RepositoryRoot,
		MemoryLimit: cfg.ExecMemoryLimit,
		CPULimit:    cfg.ExecCPULimit,
		Timeout:     cfg.ExecTimeout,
		Stdin:       cmd.Content, // NEW: Pass stdin content if present
	}

	containerResult, err := sandbox.RunContainer(containerCfg)

	result.Stdout = containerResult.Stdout
	result.Stderr = containerResult.Stderr
	result.ExitCode = containerResult.ExitCode
	result.ExecutionTime = time.Since(startTime)

	if err != nil {
		result.Success = false
		if containerResult.ExitCode == 124 {
			result.Error = fmt.Errorf("EXEC_TIMEOUT: command timed out after %v", cfg.ExecTimeout)
		} else if containerResult.ExitCode != 0 {
			result.Error = fmt.Errorf("EXEC_FAILED: command exited with code %d", containerResult.ExitCode)
		} else {
			result.Error = fmt.Errorf("EXEC_ERROR: %w", err)
		}
	} else {
		result.Success = true
	}

	// Combine stdout and stderr for result
	if result.Stdout != "" && result.Stderr != "" {
		result.Result = fmt.Sprintf("STDOUT:\n%s\n\nSTDERR:\n%s", result.Stdout, result.Stderr)
	} else if result.Stdout != "" {
		result.Result = result.Stdout
	} else if result.Stderr != "" {
		result.Result = result.Stderr
	}

	// Enhanced audit logging for exec commands
	auditMsg := fmt.Sprintf("exit_code:%d,duration:%.3fs", result.ExitCode, result.ExecutionTime.Seconds())
	if result.Success {
		auditMsg += ",status:completed"
	} else {
		auditMsg += ",status:failed"
	}
	if cmd.Content != "" {
		auditMsg += ",stdin:provided"
	}

	if auditLog != nil {
		auditLog("exec", cmd.Argument, result.Success, auditMsg)
	}

	return result
}

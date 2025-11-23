package handlers

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// ExecOperations defines command execution methods
type ExecOperations interface {
	ValidateCommand(command string, whitelist []string) error
	PrepareCommand(command string, workDir string) (*exec.Cmd, error)
	ExecuteWithTimeout(cmd *exec.Cmd, timeout time.Duration) (ExecOutput, error)
	CheckCommandExists(command string) bool
}

// ExecOutput holds command execution results
type ExecOutput struct {
	Stdout   string
	Stderr   string
	ExitCode int
	Duration time.Duration
	Error    error
}

// DefaultExecOperations implements ExecOperations
type DefaultExecOperations struct{}

// NewExecOperations creates a new exec operations handler
func NewExecOperations() ExecOperations {
	return &DefaultExecOperations{}
}

func (eo *DefaultExecOperations) ValidateCommand(command string, whitelist []string) error {
	if len(whitelist) == 0 {
		return fmt.Errorf("no commands are whitelisted")
	}

	commandParts := strings.Fields(command)
	if len(commandParts) == 0 {
		return fmt.Errorf("empty command")
	}

	baseCommand := commandParts[0]

	// Check against whitelist
	for _, allowed := range whitelist {
		if allowed == baseCommand || strings.HasPrefix(command, allowed) {
			return nil
		}
	}

	return fmt.Errorf("command not in whitelist: %s", baseCommand)
}

func (eo *DefaultExecOperations) PrepareCommand(command string, workDir string) (*exec.Cmd, error) {
	cmd := exec.Command("sh", "-c", command)
	if workDir != "" {
		cmd.Dir = workDir
	}
	return cmd, nil
}

func (eo *DefaultExecOperations) ExecuteWithTimeout(cmd *exec.Cmd, timeout time.Duration) (ExecOutput, error) {
	var output ExecOutput
	startTime := time.Now()

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// Replace the command with a context-aware version
	cmdWithContext := exec.CommandContext(ctx, cmd.Path, cmd.Args[1:]...)
	cmdWithContext.Dir = cmd.Dir
	cmdWithContext.Env = cmd.Env

	stdout, err := cmdWithContext.Output()
	output.Duration = time.Since(startTime)

	if ctx.Err() == context.DeadlineExceeded {
		output.Error = fmt.Errorf("command timed out after %v", timeout)
		output.ExitCode = 124 // Standard timeout exit code
		return output, output.Error
	}

	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			output.ExitCode = exitError.ExitCode()
			output.Stderr = string(exitError.Stderr)
		} else {
			output.ExitCode = 1
			output.Error = err
		}
		return output, err
	}

	output.Stdout = string(stdout)
	output.ExitCode = 0
	return output, nil
}

func (eo *DefaultExecOperations) CheckCommandExists(command string) bool {
	_, err := exec.LookPath(command)
	return err == nil
}

// SanitizeCommand removes potentially dangerous characters
func SanitizeCommand(command string) string {
	// Remove dangerous characters and sequences
	dangerous := []string{";", "&", "|", "`", "$", "(", ")", "<", ">", "\\", "\"", "'"}

	sanitized := command
	for _, char := range dangerous {
		sanitized = strings.ReplaceAll(sanitized, char, "")
	}

	return strings.TrimSpace(sanitized)
}

// IsCommandSafe performs basic safety checks on commands
func IsCommandSafe(command string) bool {
	dangerous := []string{"rm -rf", "del /f", "format", "fdisk", "dd if=", ":(){ :|:& };:"}

	commandLower := strings.ToLower(command)
	for _, pattern := range dangerous {
		if strings.Contains(commandLower, pattern) {
			return false
		}
	}

	return true
}

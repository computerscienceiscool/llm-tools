package security

import (
	"fmt"
	"strings"
)

// ValidateExecCommand checks if the command is allowed to execute
func ValidateExecCommand(command string, execEnabled bool, whitelist []string) error {
	if !execEnabled {
		return fmt.Errorf("exec command is disabled")
	}

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

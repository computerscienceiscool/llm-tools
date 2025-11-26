package cli

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/computerscienceiscool/llm-tools/internal/command"
	"github.com/computerscienceiscool/llm-tools/internal/executor"
)

// ProcessText processes LLM output and executes commands
func ProcessText(text string, exec *executor.Executor, startTime time.Time) string {
	var output strings.Builder

	// Parse commands from the text
	commands := command.ParseCommands(text)

	if len(commands) == 0 {
		// No commands found, return original text
		return text
	}

	output.WriteString("=== LLM TOOL START ===\n")

	// Process text and commands in order
	lastPos := 0
	for _, cmd := range commands {
		// Write text before command
		if cmd.StartPos > lastPos {
			output.WriteString(text[lastPos:cmd.StartPos])
		}

		// Write original command
		output.WriteString(cmd.Original)
		output.WriteString("\n")

		// Execute command
		result := exec.Execute(cmd)

		// Format and write result
		output.WriteString(fmt.Sprintf("=== COMMAND: %s ===\n", cmd.Original))
		if result.Success {
			switch cmd.Type {
			case "open":
				output.WriteString(fmt.Sprintf("=== FILE: %s ===\n", cmd.Argument))
				output.WriteString(result.Result)
				if !strings.HasSuffix(result.Result, "\n") {
					output.WriteString("\n")
				}
				output.WriteString("=== END FILE ===\n")
			case "write":
				output.WriteString(fmt.Sprintf("=== WRITE SUCCESSFUL: %s ===\n", cmd.Argument))
				output.WriteString(fmt.Sprintf("Action: %s\n", result.Action))
				output.WriteString(fmt.Sprintf("Bytes written: %d\n", result.BytesWritten))
				if result.BackupFile != "" {
					output.WriteString(fmt.Sprintf("Backup: %s\n", filepath.Base(result.BackupFile)))
				}
				output.WriteString("=== END WRITE ===\n")
			case "exec":
				output.WriteString(fmt.Sprintf("=== EXEC SUCCESSFUL: %s ===\n", cmd.Argument))
				output.WriteString(fmt.Sprintf("Exit code: %d\n", result.ExitCode))
				output.WriteString(fmt.Sprintf("Duration: %.3fs\n", result.ExecutionTime.Seconds()))
				if result.Result != "" {
					output.WriteString("Output:\n")
					output.WriteString(result.Result)
					if !strings.HasSuffix(result.Result, "\n") {
						output.WriteString("\n")
					}
				}
				output.WriteString("=== END EXEC ===\n")
			case "search":
				output.WriteString(result.Result)
			}
		} else {
			errParts := strings.Split(result.Error.Error(), ":")
			errType := errParts[0]
			output.WriteString(fmt.Sprintf("=== ERROR: %s ===\n", errType))
			output.WriteString(fmt.Sprintf("Message: %s\n", result.Error.Error()))
			output.WriteString(fmt.Sprintf("Command: %s\n", cmd.Original))
			if cmd.Type == "exec" && result.ExitCode != 0 {
				output.WriteString(fmt.Sprintf("Exit code: %d\n", result.ExitCode))
				if result.Stderr != "" {
					output.WriteString(fmt.Sprintf("Stderr: %s\n", result.Stderr))
				}
			}
			output.WriteString("=== END ERROR ===\n")
		}
		output.WriteString("=== END COMMAND ===\n")

		lastPos = cmd.EndPos
	}

	// Write remaining text
	if lastPos < len(text) {
		output.WriteString(text[lastPos:])
	}

	// Write summary
	output.WriteString("\n=== LLM TOOL COMPLETE ===\n")
	output.WriteString(fmt.Sprintf("Commands executed: %d\n", exec.GetCommandsRun()))
	output.WriteString(fmt.Sprintf("Time elapsed: %.2fs\n", time.Since(startTime).Seconds()))
	output.WriteString("=== END ===\n")

	return output.String()
}

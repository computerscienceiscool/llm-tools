package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/computerscienceiscool/llm-runtime/internal/command"
	"github.com/computerscienceiscool/llm-runtime/internal/executor"
)

// isCommandStart checks if a line starts with a command (ignoring leading whitespace)
func isCommandStart(line string) bool {
	trimmed := strings.TrimLeft(line, " \t")
	return strings.HasPrefix(trimmed, "<open") ||
		strings.HasPrefix(trimmed, "<write") ||
		strings.HasPrefix(trimmed, "<exec") ||
		strings.HasPrefix(trimmed, "<search")
}



// ScanInput handles continuous input/output using state machine scanner
func ScanInput(exec *executor.Executor, startTime time.Time, showPrompts bool) {
	reader := bufio.NewReader(os.Stdin)
	scanner := NewScanner(reader, showPrompts)

	if showPrompts {
		fmt.Fprintln(os.Stderr, "LLM Tool - Interactive Mode")
		fmt.Fprintln(os.Stderr, "Waiting for input (send EOF with Ctrl+D to process)...")
		fmt.Fprintln(os.Stderr, "Supports commands: <open filepath>, <write filepath>content</write>, <exec command args>, <search query>")
	}

	for {
		cmd := scanner.Scan()
		if cmd == nil {
			break
		}

		// Execute the command
		result := exec.Execute(*cmd)

		// Format and print result
		output := formatCommandResult(*cmd, result, exec, startTime)
		fmt.Print(output)

		if showPrompts {
			fmt.Fprintln(os.Stderr, "\nWaiting for more input...")
		}
	}
}

// formatCommandResult formats a single command result
func formatCommandResult(cmd command.Command, result command.ExecutionResult, exec *executor.Executor, startTime time.Time) string {
	var output strings.Builder

	output.WriteString("=== LLM TOOL START ===\n")
	output.WriteString(fmt.Sprintf("=== COMMAND: <%s %s> ===\n", cmd.Type, cmd.Argument))

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
				output.WriteString(fmt.Sprintf("Backup: %s\n", result.BackupFile))
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
		output.WriteString(fmt.Sprintf("Command: <%s %s>\n", cmd.Type, cmd.Argument))
		if cmd.Type == "exec" && result.ExitCode != 0 {
			output.WriteString(fmt.Sprintf("Exit code: %d\n", result.ExitCode))
			if result.Stderr != "" {
				output.WriteString(fmt.Sprintf("Stderr: %s\n", result.Stderr))
			}
		}
		output.WriteString("=== END ERROR ===\n")
	}

	output.WriteString("=== END COMMAND ===\n")
	output.WriteString("=== LLM TOOL COMPLETE ===\n")
	output.WriteString(fmt.Sprintf("Commands executed: %d\n", exec.GetCommandsRun()))
	output.WriteString(fmt.Sprintf("Time elapsed: %.2fs\n", time.Since(startTime).Seconds()))
	output.WriteString("=== END ===\n")

	return output.String()
}

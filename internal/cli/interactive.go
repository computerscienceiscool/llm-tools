package cli

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/computerscienceiscool/llm-runtime/pkg/evaluator"
	"github.com/computerscienceiscool/llm-runtime/pkg/scanner"
)



// ScanInput handles continuous input/output using state machine scanner
func ScanInput(exec *evaluator.Executor, startTime time.Time, showPrompts bool, input io.Reader, output io.Writer) {
	reader := bufio.NewReader(input)
	sc := scanner.NewScanner(reader, showPrompts)

	if showPrompts {
		fmt.Fprintln(os.Stderr, "LLM Tool - Interactive Mode")
		fmt.Fprintln(os.Stderr, "Waiting for input (send EOF with Ctrl+D to process)...")
		fmt.Fprintln(os.Stderr, "Supports commands: <open filepath>, <write filepath>content</write>, <exec command args>, <search query>")
	}

	for {
		cmd := sc.Scan()
		if cmd == nil {
			break
		}

		// Execute the command
		result := exec.Execute(*cmd)

		// Format and print result
		cmdOutput := formatCommandResult(*cmd, result, exec, startTime)
		fmt.Fprint(output, cmdOutput)

		if showPrompts {
			fmt.Fprintln(os.Stderr, "\nWaiting for more input...")
		}
	}
}

// formatCommandResult formats a single command result
func formatCommandResult(cmd scanner.Command, result scanner.ExecutionResult, exec *evaluator.Executor, startTime time.Time) string {
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

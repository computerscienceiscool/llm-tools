package cmd

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/computerscienceiscool/llm-tools/internal/config"
	"github.com/computerscienceiscool/llm-tools/internal/core"
	"github.com/computerscienceiscool/llm-tools/internal/parser"
)

// CLIApp represents the main CLI application
type CLIApp struct {
	configLoader config.ConfigLoader
	executor     core.CommandExecutor
	parser       parser.CommandParser
}

// NewCLIApp creates a new CLI application
func NewCLIApp(configLoader config.ConfigLoader, executor core.CommandExecutor, parser parser.CommandParser) *CLIApp {
	return &CLIApp{
		configLoader: configLoader,
		executor:     executor,
		parser:       parser,
	}
}

// Execute runs the CLI application with the given arguments
func (app *CLIApp) Execute(args []string) error {
	// Handle command line arguments and execute
	config, err := app.configLoader.LoadConfig("")
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if config.Interactive {
		return app.runInteractive(config)
	}

	return app.runPipeMode(config)
}

// runPipeMode handles non-interactive mode (pipe/file input)
func (app *CLIApp) runPipeMode(config *core.Config) error {
	var input []byte
	var err error

	// Read input
	if config.InputFile != "" {
		input, err = os.ReadFile(config.InputFile)
		if err != nil {
			return fmt.Errorf("cannot read input file: %w", err)
		}
	} else {
		input, err = io.ReadAll(os.Stdin)
		if err != nil {
			return fmt.Errorf("cannot read stdin: %w", err)
		}
	}

	// Process text
	result := app.processText(string(input), config)

	// Write output
	if config.OutputFile != "" {
		err := os.WriteFile(config.OutputFile, []byte(result), 0644)
		if err != nil {
			return fmt.Errorf("cannot write output file: %w", err)
		}
	} else {
		fmt.Print(result)
	}

	return nil
}

// runInteractive handles interactive mode
func (app *CLIApp) runInteractive(config *core.Config) error {
	scanner := bufio.NewScanner(os.Stdin)
	var buffer strings.Builder

	fmt.Fprintln(os.Stderr, "LLM Tool - Interactive Mode")
	fmt.Fprintln(os.Stderr, "Waiting for input (send EOF with Ctrl+D to process)...")
	fmt.Fprintln(os.Stderr, "Supports commands: <open filepath>, <write filepath>content</write>, <exec command args>")

	for scanner.Scan() {
		line := scanner.Text()
		buffer.WriteString(line)
		buffer.WriteString("\n")

		// Check if line contains a command
		if strings.Contains(line, "<open") || strings.Contains(line, "<write") || strings.Contains(line, "<exec") {
			// Process accumulated text
			result := app.processText(buffer.String(), config)
			fmt.Print(result)
			buffer.Reset()

			fmt.Fprintln(os.Stderr, "\nWaiting for more input...")
		}
	}

	// Process any remaining text
	if buffer.Len() > 0 {
		result := app.processText(buffer.String(), config)
		fmt.Print(result)
	}

	return nil
}

// processText processes LLM output and executes commands
func (app *CLIApp) processText(text string, config *core.Config) string {
	var output strings.Builder

	// Parse commands from the text
	commands := app.parser.ParseCommands(text)

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

		// Execute command based on type
		var result core.ExecutionResult
		switch cmd.Type {
		case "open":
			result = app.executor.ExecuteOpen(cmd.Argument)
		case "write":
			result = app.executor.ExecuteWrite(cmd.Argument, cmd.Content)
		case "exec":
			result = app.executor.ExecuteExec(cmd.Argument)
		case "search":
			result = app.executor.ExecuteSearch(cmd.Argument)
		default:
			result = core.ExecutionResult{
				Command: core.Command{
					Type:     cmd.Type,
					Argument: cmd.Argument,
					Content:  cmd.Content,
				},
				Success: false,
				Error:   fmt.Errorf("UNKNOWN_COMMAND: %s", cmd.Type),
			}
		}

		// Format and write result
		app.formatResult(&output, result)

		lastPos = cmd.EndPos
	}

	// Write remaining text
	if lastPos < len(text) {
		output.WriteString(text[lastPos:])
	}

	// Write summary
	output.WriteString("\n=== LLM TOOL COMPLETE ===\n")
	output.WriteString(fmt.Sprintf("Commands executed: %d\n", len(commands)))
	output.WriteString(fmt.Sprintf("Time elapsed: %.2fs\n", time.Since(time.Now()).Seconds()))
	output.WriteString("=== END ===\n")

	return output.String()
}

// formatResult formats an execution result for output
func (app *CLIApp) formatResult(output *strings.Builder, result core.ExecutionResult) {
	output.WriteString(fmt.Sprintf("=== COMMAND: %s ===\n", result.Command.Type))

	if result.Success {
		switch result.Command.Type {
		case "open":
			output.WriteString(fmt.Sprintf("=== FILE: %s ===\n", result.Command.Argument))
			output.WriteString(result.Result)
			if !strings.HasSuffix(result.Result, "\n") {
				output.WriteString("\n")
			}
			output.WriteString("=== END FILE ===\n")
		case "write":
			output.WriteString(fmt.Sprintf("=== WRITE SUCCESSFUL: %s ===\n", result.Command.Argument))
			output.WriteString(fmt.Sprintf("Action: %s\n", result.Action))
			output.WriteString(fmt.Sprintf("Bytes written: %d\n", result.BytesWritten))
			if result.BackupFile != "" {
				output.WriteString(fmt.Sprintf("Backup: %s\n", result.BackupFile))
			}
			output.WriteString("=== END WRITE ===\n")
		case "exec":
			output.WriteString(fmt.Sprintf("=== EXEC SUCCESSFUL: %s ===\n", result.Command.Argument))
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
		output.WriteString(fmt.Sprintf("=== ERROR: %s ===\n", strings.Split(result.Error.Error(), ":")[0]))
		output.WriteString(fmt.Sprintf("Message: %s\n", result.Error.Error()))
		output.WriteString(fmt.Sprintf("Command: %s\n", result.Command.Argument))
		if result.Command.Type == "exec" && result.ExitCode != 0 {
			output.WriteString(fmt.Sprintf("Exit code: %d\n", result.ExitCode))
			if result.Stderr != "" {
				output.WriteString(fmt.Sprintf("Stderr: %s\n", result.Stderr))
			}
		}
		output.WriteString("=== END ERROR ===\n")
	}
	output.WriteString("=== END COMMAND ===\n")
}

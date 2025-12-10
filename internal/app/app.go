package app

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/computerscienceiscool/llm-runtime/internal/config"
	"github.com/computerscienceiscool/llm-runtime/internal/search"
	"github.com/computerscienceiscool/llm-runtime/internal/session"
	"github.com/computerscienceiscool/llm-runtime/pkg/evaluator"
	"github.com/computerscienceiscool/llm-runtime/pkg/scanner"
)

// App represents the main application
type App struct {
	config    *config.Config
	session   *session.Session
	executor  *evaluator.Executor
	searchCfg *search.SearchConfig
}

// Run executes the application based on configuration
func (a *App) Run() error {
	if a.config.Verbose {
		a.printVerboseInfo()
	}

	// Set up input source
	input := os.Stdin
	if a.config.InputFile != "" {
		file, err := os.Open(a.config.InputFile)
		if err != nil {
			return fmt.Errorf("cannot read input file: %w", err)
		}
		defer file.Close()
		input = file
	}

	// Set up output destination
	output := os.Stdout
	if a.config.OutputFile != "" {
		file, err := os.Create(a.config.OutputFile)
		if err != nil {
			return fmt.Errorf("cannot write output file: %w", err)
		}
		defer file.Close()
		output = file
	}

	a.scanInput(a.executor, a.session.StartTime, a.config.Interactive, input, output)
	return nil
}

// scanInput handles continuous input/output using state machine scanner
func (a *App) scanInput(exec *evaluator.Executor, startTime time.Time, showPrompts bool, input io.Reader, output io.Writer) {
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
		cmdOutput := a.formatCommandResult(*cmd, result, exec, startTime)
		fmt.Fprint(output, cmdOutput)

		if showPrompts {
			fmt.Fprintln(os.Stderr, "\nWaiting for more input...")
		}
	}
}

// formatCommandResult formats a single command result
func (a *App) formatCommandResult(cmd scanner.Command, result scanner.ExecutionResult, exec *evaluator.Executor, startTime time.Time) string {
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

// printVerboseInfo prints verbose configuration information
func (a *App) printVerboseInfo() {
	fmt.Fprintf(os.Stderr, "Repository root: %s\n", a.config.RepositoryRoot)
	fmt.Fprintf(os.Stderr, "Max file size: %d bytes\n", a.config.MaxFileSize)
	fmt.Fprintf(os.Stderr, "Max write file size: %d bytes\n", a.config.MaxWriteSize)
	fmt.Fprintf(os.Stderr, "Allowed extensions: %v\n", a.config.AllowedExtensions)
	fmt.Fprintf(os.Stderr, "Excluded paths: %v\n", a.config.ExcludedPaths)
	fmt.Fprintf(os.Stderr, "Backup enabled: %v\n", a.config.BackupBeforeWrite)
	fmt.Fprintf(os.Stderr, "Exec enabled: %v\n", a.config.ExecEnabled)
	if a.config.ExecEnabled {
		fmt.Fprintf(os.Stderr, "Exec whitelist: %v\n", a.config.ExecWhitelist)
		fmt.Fprintf(os.Stderr, "Exec timeout: %v\n", a.config.ExecTimeout)
		fmt.Fprintf(os.Stderr, "Exec image: %s\n", a.config.ExecContainerImage)
	}
}

// GetSession returns the app's session
func (a *App) GetSession() *session.Session {
	return a.session
}

// GetExecutor returns the app's executor
func (a *App) GetExecutor() *evaluator.Executor {
	return a.executor
}

// GetConfig returns the app's configuration
func (a *App) GetConfig() *config.Config {
	return a.config
}

// GetSearchConfig returns the app's search configuration
func (a *App) GetSearchConfig() *search.SearchConfig {
	return a.searchCfg
}

package app

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/computerscienceiscool/llm-runtime/pkg/config"
	"github.com/computerscienceiscool/llm-runtime/pkg/evaluator"
	"github.com/computerscienceiscool/llm-runtime/pkg/scanner"
	"github.com/computerscienceiscool/llm-runtime/pkg/search"
	"github.com/computerscienceiscool/llm-runtime/pkg/session"
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

		// Print result directly - no intermediate formatting function
		fmt.Fprint(output, "=== LLM TOOL START ===\n")
		fmt.Fprintf(output, "=== COMMAND: <%s %s> ===\n", cmd.Type, cmd.Argument)

		if result.Success {
			switch cmd.Type {
			case "open":
				fmt.Fprintf(output, "=== FILE: %s ===\n", cmd.Argument)
				fmt.Fprint(output, result.Result)
				if !strings.HasSuffix(result.Result, "\n") {
					fmt.Fprint(output, "\n")
				}
				fmt.Fprint(output, "=== END FILE ===\n")

			case "write":
				fmt.Fprintf(output, "=== WRITE SUCCESSFUL: %s ===\n", cmd.Argument)
				fmt.Fprintf(output, "Action: %s\n", result.Action)
				fmt.Fprintf(output, "Bytes written: %d\n", result.BytesWritten)
				if result.BackupFile != "" {
					fmt.Fprintf(output, "Backup: %s\n", result.BackupFile)
				}
				fmt.Fprint(output, "=== END WRITE ===\n")

			case "exec":
				fmt.Fprintf(output, "=== EXEC SUCCESSFUL: %s ===\n", cmd.Argument)
				fmt.Fprintf(output, "Exit code: %d\n", result.ExitCode)
				fmt.Fprintf(output, "Duration: %.3fs\n", result.ExecutionTime.Seconds())
				if result.Result != "" {
					fmt.Fprint(output, "Output:\n")
					fmt.Fprint(output, result.Result)
					if !strings.HasSuffix(result.Result, "\n") {
						fmt.Fprint(output, "\n")
					}
				}
				fmt.Fprint(output, "=== END EXEC ===\n")

			case "search":
				fmt.Fprint(output, result.Result)
			}
		} else {
			errParts := strings.Split(result.Error.Error(), ":")
			errType := errParts[0]
			fmt.Fprintf(output, "=== ERROR: %s ===\n", errType)
			fmt.Fprintf(output, "Message: %s\n", result.Error.Error())
			fmt.Fprintf(output, "Command: <%s %s>\n", cmd.Type, cmd.Argument)
			if cmd.Type == "exec" && result.ExitCode != 0 {
				fmt.Fprintf(output, "Exit code: %d\n", result.ExitCode)
				if result.Stderr != "" {
					fmt.Fprintf(output, "Stderr: %s\n", result.Stderr)
				}
			}
			fmt.Fprint(output, "=== END ERROR ===\n")
		}

		fmt.Fprint(output, "=== END COMMAND ===\n")
		fmt.Fprint(output, "=== LLM TOOL COMPLETE ===\n")
		fmt.Fprintf(output, "Commands executed: %d\n", exec.GetCommandsRun())
		fmt.Fprintf(output, "Time elapsed: %.2fs\n", time.Since(startTime).Seconds())
		fmt.Fprint(output, "=== END ===\n")

		if showPrompts {
			fmt.Fprintln(os.Stderr, "\nWaiting for more input...")
		}
	}
}

// printVerboseInfo prints verbose configuration information
func (a *App) printVerboseInfo() {
	fmt.Fprintf(os.Stderr, "Repository root: %s\n", a.config.RepositoryRoot)
	fmt.Fprintf(os.Stderr, "Max file size: %d bytes\n", a.config.MaxFileSize)
	fmt.Fprintf(os.Stderr, "Max write file size: %d bytes\n", a.config.MaxWriteSize)
	fmt.Fprintf(os.Stderr, "Allowed extensions: %v\n", a.config.AllowedExtensions)
	fmt.Fprintf(os.Stderr, "Excluded paths: %v\n", a.config.ExcludedPaths)
	fmt.Fprintf(os.Stderr, "Backup enabled: %v\n", a.config.BackupBeforeWrite)

	// Exec is always enabled in container mode - controlled by whitelist only
	fmt.Fprintf(os.Stderr, "Exec enabled: true (container mode)\n")
	if len(a.config.ExecWhitelist) > 0 {
		fmt.Fprintf(os.Stderr, "Exec whitelist: %v\n", a.config.ExecWhitelist)
	}
	if a.config.ExecContainerImage != "" {
		fmt.Fprintf(os.Stderr, "Exec image: %s\n", a.config.ExecContainerImage)
	}
	if a.config.ExecTimeout > 0 {
		fmt.Fprintf(os.Stderr, "Exec timeout: %v\n", a.config.ExecTimeout)
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

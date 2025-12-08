package app

import (
	"fmt"
	"os"

	"github.com/computerscienceiscool/llm-runtime/internal/cli"
	"github.com/computerscienceiscool/llm-runtime/internal/config"
	"github.com/computerscienceiscool/llm-runtime/pkg/evaluator"
	"github.com/computerscienceiscool/llm-runtime/internal/infrastructure"
	"github.com/computerscienceiscool/llm-runtime/internal/search"
	"github.com/computerscienceiscool/llm-runtime/internal/session"
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

	cli.ScanInput(a.executor, a.session.StartTime, a.config.Interactive)

	return nil
}

// RunSearchCommand handles search-related CLI commands
func (a *App) RunSearchCommand(flags *cli.CLIFlags) error {
	if flags.CheckPythonSetup {
		return a.checkPythonSetup()
	}

	// Initialize search commands
	if a.searchCfg == nil || !a.searchCfg.Enabled {
		return fmt.Errorf("search is not enabled in configuration")
	}

	searchCmds, err := search.NewSearchCommands(a.searchCfg, a.config.RepositoryRoot)
	if err != nil {
		return fmt.Errorf("search not available: %w", err)
	}
	defer searchCmds.Close()

	if flags.Reindex {
		return searchCmds.HandleReindex(a.config.ExcludedPaths, a.config.Verbose)
	}
	if flags.SearchStatus {
		return searchCmds.HandleSearchStatus()
	}
	if flags.SearchValidate {
		return searchCmds.HandleSearchValidate()
	}
	if flags.SearchCleanup {
		return searchCmds.HandleSearchCleanup()
	}
	if flags.SearchUpdate {
		return searchCmds.HandleSearchUpdate(a.config.ExcludedPaths)
	}

	return nil
}

// checkPythonSetup verifies Python dependencies
func (a *App) checkPythonSetup() error {
	fmt.Fprintf(os.Stderr, "Checking Python setup for search functionality...\n")

	pythonPaths := []string{a.searchCfg.PythonPath, "python3", "python"}
	var workingPython string

	for _, pythonPath := range pythonPaths {
		if err := infrastructure.CheckPythonDependencies(pythonPath); err == nil {
			workingPython = pythonPath
			break
		}
	}

	if workingPython == "" {
		fmt.Fprintf(os.Stderr, "\nPython setup incomplete\n")
		fmt.Fprintf(os.Stderr, "Please install Python and sentence-transformers:\n")
		fmt.Fprintf(os.Stderr, "  pip install sentence-transformers\n")
		fmt.Fprintf(os.Stderr, "\nFor more details, see: https://pypi.org/project/sentence-transformers/\n")
		return fmt.Errorf("Python dependencies not available")
	}

	fmt.Fprintf(os.Stderr, "Python setup complete (%s)\n", workingPython)
	fmt.Fprintf(os.Stderr, "sentence-transformers library available\n")
	fmt.Fprintf(os.Stderr, "\nSearch functionality is ready to use!\n")
	fmt.Fprintf(os.Stderr, "Enable search in llm-runtime.config.yaml by setting search.enabled: true\n")

	return nil
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

package app

import (
	"fmt"
	"net/http"
	"os"

	"github.com/computerscienceiscool/llm-runtime/internal/cli"
	"github.com/computerscienceiscool/llm-runtime/internal/config"
	"github.com/computerscienceiscool/llm-runtime/internal/search"
	"github.com/computerscienceiscool/llm-runtime/internal/session"
	"github.com/computerscienceiscool/llm-runtime/pkg/evaluator"
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
	if flags.CheckOllamaSetup {
		return a.checkOllamaSetup()
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

// checkOllamaSetup verifies Ollama is available
func (a *App) checkOllamaSetup() error {
	fmt.Fprintf(os.Stderr, "Checking Ollama setup for search functionality...\n")

	if err := checkOllamaAvailability(a.searchCfg.OllamaURL); err != nil {
		fmt.Fprintf(os.Stderr, "\nOllama not available at %s\n", a.searchCfg.OllamaURL)
		fmt.Fprintf(os.Stderr, "Please install and start Ollama:\n")
		fmt.Fprintf(os.Stderr, "  curl -fsSL https://ollama.com/install.sh | sh\n")
		fmt.Fprintf(os.Stderr, "  ollama pull nomic-embed-text\n")
		fmt.Fprintf(os.Stderr, "\nFor more details, see: https://ollama.com\n")
		return fmt.Errorf("Ollama not available: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Ollama is running at %s\n", a.searchCfg.OllamaURL)
	fmt.Fprintf(os.Stderr, "\nSearch functionality is ready to use!\n")

	return nil
}

// checkOllamaAvailability verifies Ollama is running and accessible
func checkOllamaAvailability(ollamaURL string) error {
	resp, err := http.Get(ollamaURL + "/api/tags")
	if err != nil {
		return fmt.Errorf("cannot connect to Ollama: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("Ollama responded with status %d", resp.StatusCode)
	}

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

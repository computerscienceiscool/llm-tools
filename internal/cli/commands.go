package cli

import (
	"fmt"
	"net/http"
	"os"

	"github.com/computerscienceiscool/llm-runtime/internal/config"
	"github.com/computerscienceiscool/llm-runtime/internal/search"
	"github.com/spf13/cobra"
)

var reindexCmd = &cobra.Command{
	Use:   "reindex",
	Short: "Rebuild search index from scratch",
	Long:  "Rebuilds the entire search index by scanning all files in the repository and generating embeddings.",
	RunE:  runReindex,
}

var searchStatusCmd = &cobra.Command{
	Use:   "search-status",
	Short: "Show search index status",
	Long:  "Displays information about the current search index including file count and last update time.",
	RunE:  runSearchStatus,
}

var searchValidateCmd = &cobra.Command{
	Use:   "search-validate",
	Short: "Validate search index",
	Long:  "Checks the search index for consistency and reports any issues.",
	RunE:  runSearchValidate,
}

var searchCleanupCmd = &cobra.Command{
	Use:   "search-cleanup",
	Short: "Clean up search index",
	Long:  "Removes stale entries from the search index for files that no longer exist.",
	RunE:  runSearchCleanup,
}

var searchUpdateCmd = &cobra.Command{
	Use:   "search-update",
	Short: "Update search index incrementally",
	Long:  "Updates the search index by only processing new or modified files.",
	RunE:  runSearchUpdate,
}

var checkOllamaCmd = &cobra.Command{
	Use:   "check-ollama",
	Short: "Check Ollama setup for search",
	Long:  "Verifies that Ollama is installed, running, and accessible for search functionality.",
	RunE:  runCheckOllama,
}

func init() {
	// Add subcommands to root
	rootCmd.AddCommand(reindexCmd)
	rootCmd.AddCommand(searchStatusCmd)
	rootCmd.AddCommand(searchValidateCmd)
	rootCmd.AddCommand(searchCleanupCmd)
	rootCmd.AddCommand(searchUpdateCmd)
	rootCmd.AddCommand(checkOllamaCmd)
}

func runReindex(cmd *cobra.Command, args []string) error {
	cfg, err := buildConfig()
	if err != nil {
		return err
	}

	searchCfg := config.LoadSearchConfig()
	if !searchCfg.Enabled {
		return fmt.Errorf("search is not enabled in configuration")
	}

	searchCmds, err := search.NewSearchCommands(searchCfg, cfg.RepositoryRoot)
	if err != nil {
		return fmt.Errorf("search not available: %w", err)
	}
	defer searchCmds.Close()

	return searchCmds.HandleReindex(cfg.ExcludedPaths, cfg.Verbose)
}

func runSearchStatus(cmd *cobra.Command, args []string) error {
	cfg, err := buildConfig()
	if err != nil {
		return err
	}

	searchCfg := config.LoadSearchConfig()
	if !searchCfg.Enabled {
		return fmt.Errorf("search is not enabled in configuration")
	}

	searchCmds, err := search.NewSearchCommands(searchCfg, cfg.RepositoryRoot)
	if err != nil {
		return fmt.Errorf("search not available: %w", err)
	}
	defer searchCmds.Close()

	return searchCmds.HandleSearchStatus()
}

func runSearchValidate(cmd *cobra.Command, args []string) error {
	cfg, err := buildConfig()
	if err != nil {
		return err
	}

	searchCfg := config.LoadSearchConfig()
	if !searchCfg.Enabled {
		return fmt.Errorf("search is not enabled in configuration")
	}

	searchCmds, err := search.NewSearchCommands(searchCfg, cfg.RepositoryRoot)
	if err != nil {
		return fmt.Errorf("search not available: %w", err)
	}
	defer searchCmds.Close()

	return searchCmds.HandleSearchValidate()
}

func runSearchCleanup(cmd *cobra.Command, args []string) error {
	cfg, err := buildConfig()
	if err != nil {
		return err
	}

	searchCfg := config.LoadSearchConfig()
	if !searchCfg.Enabled {
		return fmt.Errorf("search is not enabled in configuration")
	}

	searchCmds, err := search.NewSearchCommands(searchCfg, cfg.RepositoryRoot)
	if err != nil {
		return fmt.Errorf("search not available: %w", err)
	}
	defer searchCmds.Close()

	return searchCmds.HandleSearchCleanup()
}

func runSearchUpdate(cmd *cobra.Command, args []string) error {
	cfg, err := buildConfig()
	if err != nil {
		return err
	}

	searchCfg := config.LoadSearchConfig()
	if !searchCfg.Enabled {
		return fmt.Errorf("search is not enabled in configuration")
	}

	searchCmds, err := search.NewSearchCommands(searchCfg, cfg.RepositoryRoot)
	if err != nil {
		return fmt.Errorf("search not available: %w", err)
	}
	defer searchCmds.Close()

	return searchCmds.HandleSearchUpdate(cfg.ExcludedPaths)
}

func runCheckOllama(cmd *cobra.Command, args []string) error {
	searchCfg := config.LoadSearchConfig()

	fmt.Fprintf(os.Stderr, "Checking Ollama setup for search functionality...\n")

	if err := checkOllamaAvailability(searchCfg.OllamaURL); err != nil {
		fmt.Fprintf(os.Stderr, "\nOllama not available at %s\n", searchCfg.OllamaURL)
		fmt.Fprintf(os.Stderr, "Please install and start Ollama:\n")
		fmt.Fprintf(os.Stderr, "  curl -fsSL https://ollama.com/install.sh | sh\n")
		fmt.Fprintf(os.Stderr, "  ollama pull nomic-embed-text\n")
		fmt.Fprintf(os.Stderr, "\nFor more details, see: https://ollama.com\n")
		return fmt.Errorf("Ollama not available: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Ollama is running at %s\n", searchCfg.OllamaURL)
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

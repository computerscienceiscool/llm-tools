package search

import (
	"fmt"
	"os"

	"github.com/computerscienceiscool/llm-tools/internal/infrastructure"
)

// SearchCommands handles search-related CLI commands
type SearchCommands struct {
	engine *SearchEngine
}

// NewSearchCommands creates a new search commands handler
func NewSearchCommands(cfg *SearchConfig, repoRoot string) (*SearchCommands, error) {
	engine, err := NewSearchEngine(cfg, repoRoot)
	if err != nil {
		return nil, err
	}

	return &SearchCommands{
		engine: engine,
	}, nil
}

// Close closes the search commands handler
func (sc *SearchCommands) Close() error {
	if sc.engine != nil {
		return sc.engine.Close()
	}
	return nil
}

// HandleReindex handles the reindex command
func (sc *SearchCommands) HandleReindex(excludedPaths []string, showProgress bool) error {
	fmt.Fprintf(os.Stderr, "Reindexing repository...\n")

	_, err := IndexRepository(
		sc.engine.GetDB(),
		sc.engine.GetConfig(),
		sc.engine.GetRepoRoot(),
		excludedPaths,
		showProgress,
		true, // force reindex
	)

	return err
}

// HandleSearchStatus handles the search status command
func (sc *SearchCommands) HandleSearchStatus() error {
	stats, err := getIndexStats(sc.engine.GetDB())
	if err != nil {
		return err
	}

	fmt.Printf("Search Index Status\n")
	fmt.Printf("==================\n")
	fmt.Printf("Total files indexed: %v\n", stats["total_files"])
	fmt.Printf("Total size: %v bytes\n", stats["total_size"])
	fmt.Printf("Oldest index: %s\n", stats["oldest_index"])
	fmt.Printf("Newest index: %s\n", stats["newest_index"])

	return nil
}

// HandleSearchValidate handles the search validate command
func (sc *SearchCommands) HandleSearchValidate() error {
	return ValidateIndex(sc.engine.GetDB(), sc.engine.GetRepoRoot())
}

// HandleSearchCleanup handles the search cleanup command
func (sc *SearchCommands) HandleSearchCleanup() error {
	fmt.Fprintf(os.Stderr, "Cleaning up search index...\n")
	return CleanupIndex(sc.engine.GetDB(), sc.engine.GetRepoRoot())
}

// HandleSearchUpdate handles the search update command
func (sc *SearchCommands) HandleSearchUpdate(excludedPaths []string) error {
	fmt.Fprintf(os.Stderr, "Updating search index...\n")
	return UpdateIndex(
		sc.engine.GetDB(),
		sc.engine.GetConfig(),
		sc.engine.GetRepoRoot(),
		excludedPaths,
	)
}

// CheckPythonSetup verifies Python environment is correctly configured
func CheckPythonSetup(pythonPath string) error {
	return infrastructure.CheckPythonDependencies(pythonPath)
}

// PrintSearchHelp prints help information for search commands
func PrintSearchHelp() {
	fmt.Println(`Search Commands:
  [[search:<query>]]     - Search for files matching the query
  
Search Management:
  --search-reindex       - Rebuild the entire search index
  --search-update        - Update index with new/modified files
  --search-status        - Show index statistics
  --search-validate      - Validate index integrity
  --search-cleanup       - Remove entries for deleted files

Configuration:
  Search settings can be configured in .llm-tool.yaml under the 'search' section.
  
Requirements:
  - Python 3 with sentence-transformers package
  - Run: pip install sentence-transformers`)
}

// InitializeSearchIndex creates initial index if needed
func (sc *SearchCommands) InitializeSearchIndex(excludedPaths []string, showProgress bool) error {
	// Check if index exists and has entries
	stats, err := getIndexStats(sc.engine.GetDB())
	if err != nil {
		return err
	}

	if stats["total_files"] == "0" {
		fmt.Fprintf(os.Stderr, "No search index found. Building initial index...\n")
		_, err = IndexRepository(
			sc.engine.GetDB(),
			sc.engine.GetConfig(),
			sc.engine.GetRepoRoot(),
			excludedPaths,
			showProgress,
			false,
		)
		return err
	}

	return nil
}

// Search performs a search and returns formatted results
func (sc *SearchCommands) Search(query string) (string, error) {
	results, err := sc.engine.Search(query)
	if err != nil {
		return "", err
	}

	return FormatSearchResults(results, query, sc.engine.GetConfig().MaxResults), nil
}

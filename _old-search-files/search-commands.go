package main

import (
	"fmt"
	"os"
	"time"
)

// SearchCommands handles search-related CLI commands
type SearchCommands struct {
	session *Session
	engine  *SearchEngine
}

// NewSearchCommands creates a new search commands handler
func NewSearchCommands(session *Session) (*SearchCommands, error) {
	searchConfig := session.GetSearchConfig()
	if !searchConfig.Enabled {
		return nil, fmt.Errorf("search is disabled")
	}

	engine, err := NewSearchEngine(searchConfig, session)
	if err != nil {
		return nil, err
	}

	return &SearchCommands{
		session: session,
		engine:  engine,
	}, nil
}

// Close closes the search commands handler
func (sc *SearchCommands) Close() error {
	if sc.engine != nil {
		return sc.engine.Close()
	}
	return nil
}

// HandleReindex handles the --reindex command line flag
func (sc *SearchCommands) HandleReindex(verbose bool) error {
	fmt.Fprintf(os.Stderr, "Starting full reindex of repository...\n")

	stats, err := sc.engine.IndexRepository(verbose, true)
	if err != nil {
		return fmt.Errorf("reindex failed: %w", err)
	}

	fmt.Fprintf(os.Stderr, "\nReindex completed successfully!\n")
	fmt.Fprintf(os.Stderr, "Files indexed: %d\n", stats.IndexedFiles)
	fmt.Fprintf(os.Stderr, "Total time: %.2fs\n", stats.EndTime.Sub(stats.StartTime).Seconds())

	return nil
}

// HandleSearchStatus shows current search index status
func (sc *SearchCommands) HandleSearchStatus() error {
	stats, err := sc.engine.GetIndexStats()
	if err != nil {
		return fmt.Errorf("failed to get index stats: %w", err)
	}

	fmt.Printf("=== Search Index Status ===\n")
	fmt.Printf("Database path: %s\n", stats["database_path"])
	fmt.Printf("Embedding model: %s\n", stats["model"])
	fmt.Printf("Total files indexed: %d\n", stats["total_files"])
	fmt.Printf("Total size indexed: %s\n", formatFileSize(stats["total_size"].(int64)))

	if stats["total_files"].(int64) > 0 {
		fmt.Printf("Oldest index: %s\n", stats["oldest_index"].(time.Time).Format("2006-01-02 15:04:05"))
		fmt.Printf("Newest index: %s\n", stats["newest_index"].(time.Time).Format("2006-01-02 15:04:05"))
	}

	return nil
}

// HandleSearchValidate validates the search index
func (sc *SearchCommands) HandleSearchValidate() error {
	fmt.Fprintf(os.Stderr, "Validating search index...\n")

	if err := sc.engine.ValidateIndex(); err != nil {
		return fmt.Errorf("index validation failed: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Index validation completed successfully!\n")
	return nil
}

// HandleSearchCleanup cleans up the search index
func (sc *SearchCommands) HandleSearchCleanup() error {
	fmt.Fprintf(os.Stderr, "Cleaning up search index...\n")

	if err := sc.engine.CleanupIndex(); err != nil {
		return fmt.Errorf("index cleanup failed: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Index cleanup completed successfully!\n")
	return nil
}

// HandleSearchUpdate performs incremental index update
func (sc *SearchCommands) HandleSearchUpdate(verbose bool) error {
	if verbose {
		fmt.Fprintf(os.Stderr, "Updating search index...\n")
	}

	if err := sc.engine.UpdateIndex(); err != nil {
		return fmt.Errorf("index update failed: %w", err)
	}

	if verbose {
		fmt.Fprintf(os.Stderr, "Index update completed successfully!\n")
	}
	return nil
}

// CheckPythonSetup checks if Python and dependencies are properly set up
func CheckPythonSetup() error {
	fmt.Fprintf(os.Stderr, "Checking Python setup for search functionality...\n")

	// Test Python availability
	pythonPaths := []string{"python3", "python"}
	var workingPython string

	for _, pythonPath := range pythonPaths {
		if err := checkPythonDependencies(pythonPath); err == nil {
			workingPython = pythonPath
			break
		}
	}

	if workingPython == "" {
		fmt.Fprintf(os.Stderr, "\n❌ Python setup incomplete\n")
		fmt.Fprintf(os.Stderr, "Please install Python and sentence-transformers:\n")
		fmt.Fprintf(os.Stderr, "  pip install sentence-transformers\n")
		fmt.Fprintf(os.Stderr, "\nFor more details, see: https://pypi.org/project/sentence-transformers/\n")
		return fmt.Errorf("Python dependencies not available")
	}

	fmt.Fprintf(os.Stderr, "✅ Python setup complete (%s)\n", workingPython)
	fmt.Fprintf(os.Stderr, "✅ sentence-transformers library available\n")
	fmt.Fprintf(os.Stderr, "\nSearch functionality is ready to use!\n")
	fmt.Fprintf(os.Stderr, "Enable search in llm-tool.config.yaml by setting search.enabled: true\n")

	return nil
}

// PrintSearchHelp prints help information for search functionality
func PrintSearchHelp() {
	fmt.Printf("Search functionality allows semantic search through your codebase.\n\n")

	fmt.Printf("Setup:\n")
	fmt.Printf("  1. Install Python dependencies: pip install sentence-transformers\n")
	fmt.Printf("  2. Enable search in config: search.enabled: true\n")
	fmt.Printf("  3. Build initial index: ./llm-tool --reindex\n\n")

	fmt.Printf("Usage in LLM prompts:\n")
	fmt.Printf("  <search authentication logic>     - Find files related to authentication\n")
	fmt.Printf("  <search error handling>          - Find error handling code\n")
	fmt.Printf("  <search database connection>     - Find database-related code\n\n")

	fmt.Printf("Command-line options:\n")
	fmt.Printf("  --reindex              - Rebuild search index from scratch\n")
	fmt.Printf("  --search-status        - Show current index statistics\n")
	fmt.Printf("  --search-validate      - Validate index integrity\n")
	fmt.Printf("  --search-cleanup       - Remove entries for deleted files\n")
	fmt.Printf("  --search-update        - Update index incrementally\n")
	fmt.Printf("  --check-python-setup   - Check if Python dependencies are installed\n\n")

	fmt.Printf("Configuration (llm-tool.config.yaml):\n")
	fmt.Printf("  search:\n")
	fmt.Printf("    enabled: true\n")
	fmt.Printf("    vector_db_path: \"./embeddings.db\"\n")
	fmt.Printf("    max_results: 10\n")
	fmt.Printf("    min_similarity_score: 0.5\n")
	fmt.Printf("    index_extensions: [\".go\", \".py\", \".js\", \".md\"]\n")
}

// InitializeSearchIndex performs initial indexing if database doesn't exist
func (sc *SearchCommands) InitializeSearchIndex() error {
	// Check if database exists
	if _, err := os.Stat(sc.engine.config.VectorDBPath); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "No search index found. Creating initial index...\n")

		stats, err := sc.engine.IndexRepository(true, true)
		if err != nil {
			return fmt.Errorf("initial indexing failed: %w", err)
		}

		fmt.Fprintf(os.Stderr, "Initial index created successfully!\n")
		fmt.Fprintf(os.Stderr, "Files indexed: %d\n", stats.IndexedFiles)

		return nil
	}

	// Database exists, just do incremental update
	return sc.engine.UpdateIndex()
}

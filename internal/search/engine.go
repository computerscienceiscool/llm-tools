package search

import (
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/computerscienceiscool/llm-runtime/internal/infrastructure"
)

// SearchEngine provides semantic search functionality
type SearchEngine struct {
	db       *sql.DB
	config   *SearchConfig
	repoRoot string
}

// NewSearchEngine creates a new search engine instance
func NewSearchEngine(cfg *SearchConfig, repoRoot string) (*SearchEngine, error) {
	if !cfg.Enabled {
		return nil, fmt.Errorf("search is not enabled in configuration")
	}

	// Ensure database directory exists
	dbDir := filepath.Dir(cfg.VectorDBPath)
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}

	// Initialize database
	db, err := infrastructure.InitSearchDB(cfg.VectorDBPath)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize search database: %w", err)
	}

	return &SearchEngine{
		db:       db,
		config:   cfg,
		repoRoot: repoRoot,
	}, nil
}

// Close closes the search engine and its resources
func (se *SearchEngine) Close() error {
	if se.db != nil {
		return se.db.Close()
	}
	return nil
}

// Search performs a semantic search for the given query
func (se *SearchEngine) Search(query string) ([]SearchResult, error) {
	// Check Ollama availability
	if err := checkOllamaAvailability(se.config.OllamaURL); err != nil {
		return nil, fmt.Errorf("Ollama not available: %w", err)
	}

	// Generate embedding for query
	queryEmbedding, err := generateEmbedding(se.config.OllamaURL, query)
	if err != nil {
		return nil, fmt.Errorf("failed to generate query embedding: %w", err)
	}

	// Query all embeddings from database
	rows, err := se.db.Query("SELECT filepath, embedding, file_size FROM embeddings")
	if err != nil {
		return nil, fmt.Errorf("failed to query embeddings: %w", err)
	}
	defer rows.Close()

	var results []SearchResult

	for rows.Next() {
		var filePath string
		var embeddingBytes []byte
		var fileSize int64

		if err := rows.Scan(&filePath, &embeddingBytes, &fileSize); err != nil {
			continue
		}

		// Deserialize embedding
		fileEmbedding := deserializeEmbedding(embeddingBytes)
		if len(fileEmbedding) != embeddingDimensions {
			continue
		}

		// Calculate similarity
		score := cosineSimilarity(queryEmbedding, fileEmbedding)

		// Filter by minimum score
		if score < float32(se.config.MinSimilarityScore) {
			continue
		}

		// Create result
		result := SearchResult{
			FilePath:  filePath,
			Score:     score,
			FileSize:  fileSize,
			LineCount: countLines(filepath.Join(se.repoRoot, filePath)),
			Relevance: GetRelevanceLabel(score),
		}

		// Generate preview if needed
		if se.config.MaxPreviewLength > 0 {
			result.Preview = generatePreview(se.repoRoot, filePath, se.config.MaxPreviewLength)
		}

		results = append(results, result)
	}

	// Rank results by score
	rankSearchResults(results)

	// Limit results
	if se.config.MaxResults > 0 && len(results) > se.config.MaxResults {
		results = results[:se.config.MaxResults]
	}

	return results, nil
}

// GetDB returns the underlying database connection
func (se *SearchEngine) GetDB() *sql.DB {
	return se.db
}

// GetConfig returns the search configuration
func (se *SearchEngine) GetConfig() *SearchConfig {
	return se.config
}

// GetRepoRoot returns the repository root path
func (se *SearchEngine) GetRepoRoot() string {
	return se.repoRoot
}

// checkOllamaAvailability verifies Ollama is running and accessible
func checkOllamaAvailability(ollamaURL string) error {
	resp, err := http.Get(ollamaURL + "/api/tags")
	if err != nil {
		return fmt.Errorf("cannot connect to Ollama at %s: %w", ollamaURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Ollama responded with status %d", resp.StatusCode)
	}

	return nil
}

package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// ExecuteSearch handles the "search" command
func (s *Session) ExecuteSearch(query string) ExecutionResult {
	startTime := time.Now()
	result := ExecutionResult{
		Command: Command{Type: "search", Argument: query},
	}

	// Check if search is enabled
	searchConfig := s.GetSearchConfig()
	if !searchConfig.Enabled {
		result.Success = false
		result.Error = fmt.Errorf("SEARCH_DISABLED: search feature is not enabled")
		result.ExecutionTime = time.Since(startTime)
		s.LogAudit("search", query, false, result.Error.Error())
		return result
	}

	// Initialize search engine
	searchEngine, err := NewSearchEngine(searchConfig, s)
	if err != nil {
		result.Success = false
		result.Error = fmt.Errorf("SEARCH_INIT_FAILED: %w", err)
		result.ExecutionTime = time.Since(startTime)
		s.LogAudit("search", query, false, result.Error.Error())
		return result
	}
	defer searchEngine.Close()

	// Perform incremental index update
	/*	if err := searchEngine.UpdateIndex(); err != nil {
		// Don't fail on update errors, just log them
		if s.Config.Verbose {
			fmt.Fprintf(os.Stderr, "Warning: Failed to update index: %v\n", err)
		}
	}*/

	// Execute search
	searchResults, err := searchEngine.Search(query)
	if err != nil {
		result.Success = false
		result.Error = fmt.Errorf("SEARCH_FAILED: %w", err)
		result.ExecutionTime = time.Since(startTime)
		s.LogAudit("search", query, false, result.Error.Error())
		return result
	}

	// Format results
	result.Success = true
	result.Result = searchEngine.FormatSearchResults(query, searchResults, time.Since(startTime))
	result.ExecutionTime = time.Since(startTime)

	// Log successful search
	s.LogAudit("search", query, true, fmt.Sprintf("results:%d,duration:%.3fs",
		len(searchResults), result.ExecutionTime.Seconds()))
	s.CommandsRun++

	return result
}

// Search performs semantic search on indexed files
func (se *SearchEngine) Search(query string) ([]SearchResult, error) {
	if strings.TrimSpace(query) == "" {
		return nil, fmt.Errorf("empty search query")
	}

	// Generate embedding for query
	queryEmbedding, err := se.generateEmbedding(query)
	if err != nil {
		return nil, fmt.Errorf("failed to generate query embedding: %w", err)
	}

	// Get all embeddings from database
	rows, err := se.db.Query(`
		SELECT filepath, embedding, last_modified, file_size 
		FROM embeddings
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to query embeddings: %w", err)
	}
	defer rows.Close()

	var candidates []SearchResult

	// Calculate similarities
	for rows.Next() {
		var filePath string
		var embeddingData []byte
		var lastModified, fileSize int64

		if err := rows.Scan(&filePath, &embeddingData, &lastModified, &fileSize); err != nil {
			continue // Skip problematic entries
		}

		// Deserialize embedding
		embedding := deserializeEmbedding(embeddingData)
		if embedding == nil {
			continue
		}

		// Calculate cosine similarity
		similarity := cosineSimilarity(queryEmbedding, embedding)

		// Apply minimum score filter
		if similarity < float32(se.config.MinSimilarityScore) {
			continue
		}

		// Get file info
		fullPath := filepath.Join(se.session.Config.RepositoryRoot, filePath)
		//	info, err := os.Stat(fullPath)
		if err != nil {
			continue // File might have been deleted
		}

		// Count lines
		lines := se.countLines(fullPath)

		// Generate preview
		preview := se.generatePreview(fullPath, se.config.MaxPreviewLength)

		candidates = append(candidates, SearchResult{
			FilePath: filePath,
			Score:    float64(similarity),
			Lines:    lines,
			Size:     fileSize,
			Preview:  preview,
			ModTime:  time.Unix(lastModified, 0),
		})
	}

	// Apply search ranking algorithm
	se.rankSearchResults(candidates, query)

	// Sort by score (highest first)
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].Score > candidates[j].Score
	})

	// Limit results
	if len(candidates) > se.config.MaxResults {
		candidates = candidates[:se.config.MaxResults]
	}

	return candidates, nil
}

// rankSearchResults applies ranking algorithm to boost certain results
func (se *SearchEngine) rankSearchResults(results []SearchResult, query string) {
	queryLower := strings.ToLower(query)

	for i := range results {
		result := &results[i]

		// Boost for exact filename matches
		fileName := strings.ToLower(filepath.Base(result.FilePath))
		if strings.Contains(fileName, queryLower) {
			result.Score += 0.1
		}

		// Boost for files in commonly important directories
		if strings.Contains(result.FilePath, "src/") ||
			strings.Contains(result.FilePath, "lib/") ||
			strings.Contains(result.FilePath, "main") {
			result.Score += 0.05
		}

		// Penalize very large files slightly
		if result.Size > 50000 {
			result.Score -= 0.05
		}

		// Boost recent files slightly
		if time.Since(result.ModTime) < 24*time.Hour {
			result.Score += 0.02
		}
	}
}

// countLines counts the number of lines in a file
func (se *SearchEngine) countLines(filePath string) int {
	file, err := os.Open(filePath)
	if err != nil {
		return 0
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lines := 0
	for scanner.Scan() {
		lines++
	}
	return lines
}

// FormatSearchResults formats search results for display
func (se *SearchEngine) FormatSearchResults(query string, results []SearchResult, duration time.Duration) string {
	var output strings.Builder

	output.WriteString(fmt.Sprintf("=== SEARCH: %s ===\n", query))
	output.WriteString(fmt.Sprintf("=== SEARCH RESULTS (%.2fs) ===\n", duration.Seconds()))

	if len(results) == 0 {
		output.WriteString("No files found matching query.\n")
		output.WriteString("Try broader search terms or check if files are indexed.\n")
		output.WriteString("=== END SEARCH ===\n")
		return output.String()
	}

	for i, result := range results {
		output.WriteString(fmt.Sprintf("%d. %s (score: %.2f)\n",
			i+1, result.FilePath, result.Score))

		// File metadata
		output.WriteString(fmt.Sprintf("   Lines: %d | Size: %s",
			result.Lines, formatFileSize(result.Size)))

		if !result.ModTime.IsZero() {
			output.WriteString(fmt.Sprintf(" | Modified: %s",
				result.ModTime.Format("2006-01-02")))
		}
		output.WriteString("\n")

		// Preview
		if result.Preview != "" {
			output.WriteString(fmt.Sprintf("   Preview: \"%s\"\n", result.Preview))
		}

		output.WriteString("\n")
	}

	// Show additional results count
	totalResults := len(results)
	if totalResults >= se.config.MaxResults {
		output.WriteString(fmt.Sprintf("[Showing top %d results]\n", se.config.MaxResults))
	}

	output.WriteString("=== END SEARCH ===\n")
	return output.String()
}

// formatFileSize formats file size in human-readable format
func formatFileSize(size int64) string {
	const unit = 1024
	if size < unit {
		return fmt.Sprintf("%d B", size)
	}

	div, exp := int64(unit), 0
	for n := size / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}

	return fmt.Sprintf("%.1f %cB", float64(size)/float64(div), "KMGTPE"[exp])
}

/*
// GetSearchConfig returns search configuration (to be implemented in main.go)
func (s *Session) GetSearchConfig() *SearchConfig {
	// This will need to be implemented in main.go to read from config
	return &SearchConfig{
		Enabled:            false, // Default to disabled
		VectorDBPath:       "./embeddings.db",
		EmbeddingModel:     "all-MiniLM-L6-v2",
		MaxResults:         10,
		MinSimilarityScore: 0.5,
		MaxPreviewLength:   100,
		ChunkSize:          1000,
		PythonPath:         "python3",
		IndexExtensions:    []string{".go", ".py", ".js", ".md", ".txt", ".yaml", ".json"},
		MaxFileSize:        1048576, // 1MB
	}
}*/

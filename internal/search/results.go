package search

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// SearchResult represents a single search result
type SearchResult struct {
	FilePath   string
	Score      float32
	Preview    string
	LineCount  int
	FileSize   int64
	Relevance  string
}

// FormatSearchResults formats search results for display
func FormatSearchResults(results []SearchResult, query string, maxResults int) string {
	if len(results) == 0 {
		return fmt.Sprintf("No results found for query: %s", query)
	}

	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("Search results for: %s\n", query))
	sb.WriteString(fmt.Sprintf("Found %d matching files\n\n", len(results)))

	displayCount := len(results)
	if maxResults > 0 && displayCount > maxResults {
		displayCount = maxResults
	}

	for i := 0; i < displayCount; i++ {
		result := results[i]

		sb.WriteString(fmt.Sprintf("─── %d. %s ───\n", i+1, result.FilePath))
		sb.WriteString(fmt.Sprintf("Score: %.2f%% | Size: %s | Lines: %d\n",
			result.Score*100, formatFileSize(result.FileSize), result.LineCount))

		if result.Preview != "" {
			sb.WriteString("Preview:\n")
			sb.WriteString(result.Preview)
			sb.WriteString("\n")
		}

		sb.WriteString("\n")
	}

	if len(results) > displayCount {
		sb.WriteString(fmt.Sprintf("... and %d more results\n", len(results)-displayCount))
	}

	return sb.String()
}

// formatFileSize formats file size in human readable format
func formatFileSize(size int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)

	switch {
	case size >= GB:
		return fmt.Sprintf("%.2f GB", float64(size)/float64(GB))
	case size >= MB:
		return fmt.Sprintf("%.2f MB", float64(size)/float64(MB))
	case size >= KB:
		return fmt.Sprintf("%.2f KB", float64(size)/float64(KB))
	default:
		return fmt.Sprintf("%d B", size)
	}
}

// rankSearchResults sorts results by score descending
func rankSearchResults(results []SearchResult) {
	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})
}

// countLines counts the number of lines in a file
func countLines(filePath string) int {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return 0
	}

	lines := 1
	for _, b := range content {
		if b == '\n' {
			lines++
		}
	}
	return lines
}

// generatePreview generates a preview snippet from file content
func generatePreview(repoRoot string, filePath string, maxLength int) string {
	fullPath := filepath.Join(repoRoot, filePath)

	content, err := os.ReadFile(fullPath)
	if err != nil {
		return ""
	}

	text := string(content)

	// Trim whitespace
	text = strings.TrimSpace(text)

	// Truncate if necessary
	if len(text) > maxLength {
		// Try to break at a line boundary
		truncated := text[:maxLength]
		lastNewline := strings.LastIndex(truncated, "\n")
		if lastNewline > maxLength/2 {
			truncated = truncated[:lastNewline]
		}
		text = truncated + "\n..."
	}

	// Indent preview lines
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		lines[i] = "  " + line
	}

	return strings.Join(lines, "\n")
}

// GetRelevanceLabel returns a human-readable relevance label based on score
func GetRelevanceLabel(score float32) string {
	switch {
	case score >= 0.9:
		return "Excellent"
	case score >= 0.8:
		return "Very Good"
	case score >= 0.7:
		return "Good"
	case score >= 0.6:
		return "Fair"
	case score >= 0.5:
		return "Marginal"
	default:
		return "Low"
	}
}

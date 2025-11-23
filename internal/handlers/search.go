package handlers

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

// SearchOperations defines search operation methods
type SearchOperations interface {
	IndexFile(filePath string, content string) error
	Search(query string, maxResults int) ([]SearchMatch, error)
	UpdateIndex(filePath string, content string) error
	RemoveFromIndex(filePath string) error
	GetIndexStats() (IndexStats, error)
}

// SearchMatch represents a search result match
type SearchMatch struct {
	FilePath   string
	Score      float64
	LineNumber int
	Line       string
	Context    []string
	Snippet    string
}

// IndexStats holds statistics about the search index
type IndexStats struct {
	TotalFiles   int
	TotalLines   int
	LastUpdated  time.Time
	IndexSize    int64
	AverageScore float64
}

// DefaultSearchOperations implements SearchOperations
type DefaultSearchOperations struct {
	index map[string][]IndexedLine
}

// IndexedLine represents a line in the search index
type IndexedLine struct {
	Number  int
	Content string
	Words   []string
}

// NewSearchOperations creates a new search operations handler
func NewSearchOperations() SearchOperations {
	return &DefaultSearchOperations{
		index: make(map[string][]IndexedLine),
	}
}

func (so *DefaultSearchOperations) IndexFile(filePath string, content string) error {
	lines := strings.Split(content, "\n")
	indexedLines := make([]IndexedLine, len(lines))

	for i, line := range lines {
		words := strings.Fields(strings.ToLower(line))
		indexedLines[i] = IndexedLine{
			Number:  i + 1,
			Content: line,
			Words:   words,
		}
	}

	so.index[filePath] = indexedLines
	return nil
}

func (so *DefaultSearchOperations) Search(query string, maxResults int) ([]SearchMatch, error) {
	queryWords := strings.Fields(strings.ToLower(query))
	if len(queryWords) == 0 {
		return nil, fmt.Errorf("empty search query")
	}

	var matches []SearchMatch

	for filePath, lines := range so.index {
		for _, line := range lines {
			score := so.calculateScore(queryWords, line.Words)
			if score > 0 {
				match := SearchMatch{
					FilePath:   filePath,
					Score:      score,
					LineNumber: line.Number,
					Line:       line.Content,
					Snippet:    so.createSnippet(line.Content, queryWords),
				}
				matches = append(matches, match)
			}
		}
	}

	// Sort by score (highest first)
	sort.Slice(matches, func(i, j int) bool {
		return matches[i].Score > matches[j].Score
	})

	// Limit results
	if len(matches) > maxResults {
		matches = matches[:maxResults]
	}

	return matches, nil
}

func (so *DefaultSearchOperations) UpdateIndex(filePath string, content string) error {
	return so.IndexFile(filePath, content)
}

func (so *DefaultSearchOperations) RemoveFromIndex(filePath string) error {
	delete(so.index, filePath)
	return nil
}

func (so *DefaultSearchOperations) GetIndexStats() (IndexStats, error) {
	totalFiles := len(so.index)
	totalLines := 0

	for _, lines := range so.index {
		totalLines += len(lines)
	}

	return IndexStats{
		TotalFiles:  totalFiles,
		TotalLines:  totalLines,
		LastUpdated: time.Now(),
		IndexSize:   int64(totalLines * 50), // Rough estimate
	}, nil
}

func (so *DefaultSearchOperations) calculateScore(queryWords []string, lineWords []string) float64 {
	if len(queryWords) == 0 || len(lineWords) == 0 {
		return 0
	}

	matches := 0
	for _, queryWord := range queryWords {
		for _, lineWord := range lineWords {
			if strings.Contains(lineWord, queryWord) {
				matches++
				break
			}
		}
	}

	// Score is the percentage of query words found
	return float64(matches) / float64(len(queryWords))
}

func (so *DefaultSearchOperations) createSnippet(line string, queryWords []string) string {
	const maxSnippetLength = 100

	if len(line) <= maxSnippetLength {
		return line
	}

	// Find the first occurrence of any query word
	firstIndex := len(line)
	for _, word := range queryWords {
		if index := strings.Index(strings.ToLower(line), word); index != -1 && index < firstIndex {
			firstIndex = index
		}
	}

	// Create snippet around the first match
	start := firstIndex - 30
	if start < 0 {
		start = 0
	}

	end := start + maxSnippetLength
	if end > len(line) {
		end = len(line)
	}

	snippet := line[start:end]
	if start > 0 {
		snippet = "..." + snippet
	}
	if end < len(line) {
		snippet = snippet + "..."
	}

	return snippet
}

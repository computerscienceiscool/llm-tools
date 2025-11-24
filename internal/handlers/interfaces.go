package handlers

import "time"

// FileHandler handles file operations
type FileHandler interface {
	OpenFile(filePath string, maxSize int64, repoRoot string) (string, error)
	WriteFile(filePath, content string, maxSize int64, repoRoot string, allowedExts []string, backup bool) (WriteResult, error)
}

// WriteResult contains write operation results
type WriteResult struct {
	Action       string
	BytesWritten int64
	BackupFile   string
}

// SearchHandler handles search operations
type SearchHandler interface {
	Search(query string) ([]SearchResult, error)
}

// SearchResult represents a search result
type SearchResult struct {
	FilePath string
	Score    float64
	Lines    int
	Size     int64
	Preview  string
	ModTime  time.Time
}

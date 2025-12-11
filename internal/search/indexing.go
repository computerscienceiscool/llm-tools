package search

import (
	"database/sql"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// IndexStats holds statistics about indexing operation
type IndexStats struct {
	TotalFiles   int
	IndexedFiles int
	SkippedFiles int
	ErrorFiles   int
	StartTime    time.Time
	EndTime      time.Time
	BytesIndexed int64
}

// IndexRepository walks through repository and indexes files
func IndexRepository(db *sql.DB, cfg *SearchConfig, repoRoot string, excludedPaths []string, showProgress bool, reindexAll bool) (*IndexStats, error) {
	stats := &IndexStats{
		StartTime: time.Now(),
	}

	if showProgress {
		fmt.Fprintf(os.Stderr, "Starting repository indexing...\n")
	}

	// Walk through repository
	err := filepath.Walk(repoRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		stats.TotalFiles++

		// Get relative path
		relPath, err := filepath.Rel(repoRoot, path)
		if err != nil {
			relPath = path
		}

		// Check if file should be indexed
		if !shouldIndexFile(relPath, cfg.IndexExtensions, excludedPaths) {
			stats.SkippedFiles++
			return nil
		}

		// Check file size limit
		if info.Size() > cfg.MaxFileSize {
			stats.SkippedFiles++
			return nil
		}

		// Check if file is text
		if !isTextFile(path) {
			stats.SkippedFiles++
			return nil
		}

		// Show progress
		if showProgress && stats.TotalFiles > 0 {
			fmt.Fprintf(os.Stderr, "\rIndexing: %d files processed, %d indexed - %s",
				stats.TotalFiles, stats.IndexedFiles, filepath.Base(relPath))
		}

		// Check if file needs indexing
		needsIndexing, err := fileNeedsIndexing(db, relPath, info, reindexAll)
		if err != nil {
			stats.ErrorFiles++
			if showProgress {
				fmt.Fprintf(os.Stderr, "\nError checking %s: %v\n", relPath, err)
			}
			return nil
		}

		if !needsIndexing {
			stats.SkippedFiles++
			return nil
		}

		// Index the file
		if err := indexFile(db, cfg, repoRoot, relPath, info); err != nil {
			stats.ErrorFiles++
			if showProgress {
				fmt.Fprintf(os.Stderr, "\nError indexing %s: %v\n", relPath, err)
			}
			return nil
		}

		stats.IndexedFiles++
		stats.BytesIndexed += info.Size()

		return nil
	})

	stats.EndTime = time.Now()

	if showProgress {
		fmt.Fprintf(os.Stderr, "\n")
		printIndexStats(stats)
	}

	return stats, err
}

// shouldIndexFile determines if a file should be indexed based on extension and other criteria
func shouldIndexFile(filePath string, indexExtensions []string, excludedPaths []string) bool {
	// Check file extension
	ext := strings.ToLower(filepath.Ext(filePath))
	if ext == "" {
		return false
	}

	found := false
	for _, allowedExt := range indexExtensions {
		if strings.ToLower(allowedExt) == ext {
			found = true
			break
		}
	}
	if !found {
		return false
	}

	// Check if path is excluded
	for _, excluded := range excludedPaths {
		if matched, _ := filepath.Match(excluded, filepath.Base(filePath)); matched {
			return false
		}
		if strings.HasPrefix(filePath, excluded+string(filepath.Separator)) {
			return false
		}
	}

	return true
}

// fileNeedsIndexing checks if a file needs to be indexed or re-indexed
func fileNeedsIndexing(db *sql.DB, filePath string, info os.FileInfo, forceReindex bool) (bool, error) {
	if forceReindex {
		return true, nil
	}

	// Check if file exists in database
	existingInfo, err := getFileInfo(db, filePath)
	if err != nil {
		// File not in database, needs indexing
		return true, nil
	}

	// Check if file has been modified
	if existingInfo.LastModified != info.ModTime().Unix() {
		return true, nil
	}

	// Check if file size changed
	if existingInfo.FileSize != info.Size() {
		return true, nil
	}

	return false, nil
}

// indexFile indexes a single file
func indexFile(db *sql.DB, cfg *SearchConfig, repoRoot string, filePath string, info os.FileInfo) error {
	// Read file content
	fullPath := filepath.Join(repoRoot, filePath)
	content, err := os.ReadFile(fullPath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	// Calculate content hash
	contentHash := fmt.Sprintf("%x", content)

	// Generate embedding
	truncated := truncateText(string(content), 200)
	embedding, err := generateEmbedding(cfg.OllamaURL, truncated)
	if err != nil {
		return fmt.Errorf("failed to generate embedding: %w", err)
	}

	// Create file info
	fileInfo := &FileInfo{
		FilePath:     filePath,
		ContentHash:  contentHash,
		Embedding:    embedding,
		LastModified: info.ModTime().Unix(),
		FileSize:     info.Size(),
		IndexedAt:    time.Now().Unix(),
	}

	// Store in database
	if err := storeFileInfo(db, fileInfo); err != nil {
		return fmt.Errorf("failed to store file info: %w", err)
	}

	return nil
}

// UpdateIndex performs incremental update of the index
func UpdateIndex(db *sql.DB, cfg *SearchConfig, repoRoot string, excludedPaths []string) error {
	// Get all files currently in database
	dbFiles, err := getAllIndexedFiles(db)
	if err != nil {
		return err
	}

	// Track files that still exist
	existingFiles := make(map[string]bool)

	// Walk through repository to find changed/new files
	err = filepath.Walk(repoRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		relPath, err := filepath.Rel(repoRoot, path)
		if err != nil {
			relPath = path
		}

		existingFiles[relPath] = true

		// Check if file should be indexed
		if !shouldIndexFile(relPath, cfg.IndexExtensions, excludedPaths) {
			return nil
		}

		if info.Size() > cfg.MaxFileSize {
			return nil
		}

		if !isTextFile(path) {
			return nil
		}

		// Check if needs indexing
		needsIndexing, err := fileNeedsIndexing(db, relPath, info, false)
		if err != nil {
			return err
		}

		if needsIndexing {
			if err := indexFile(db, cfg, repoRoot, relPath, info); err != nil {
				return err
			}
		}

		return nil
	})

	if err != nil {
		return err
	}

	// Remove files that no longer exist
	for _, dbFile := range dbFiles {
		if !existingFiles[dbFile] {
			removeFileInfo(db, dbFile)
		}
	}

	return nil
}

// CleanupIndex removes entries for non-existent files
func CleanupIndex(db *sql.DB, repoRoot string) error {
	files, err := getAllIndexedFiles(db)
	if err != nil {
		return err
	}

	for _, filePath := range files {
		fullPath := filepath.Join(repoRoot, filePath)
		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			removeFileInfo(db, filePath)
		}
	}

	return nil
}

// ValidateIndex checks the integrity of the search index
func ValidateIndex(db *sql.DB, repoRoot string) error {
	// Check if all indexed files still exist and have correct hashes
	rows, err := db.Query("SELECT filepath, content_hash, last_modified FROM embeddings")
	if err != nil {
		return err
	}
	defer rows.Close()

	issues := 0
	for rows.Next() {
		var filePath, storedHash string
		var storedModTime int64

		if err := rows.Scan(&filePath, &storedHash, &storedModTime); err != nil {
			return err
		}

		fullPath := filepath.Join(repoRoot, filePath)

		// Check if file exists
		info, err := os.Stat(fullPath)
		if os.IsNotExist(err) {
			fmt.Fprintf(os.Stderr, "Missing file: %s\n", filePath)
			issues++
			continue
		}

		// Check modification time
		if info.ModTime().Unix() != storedModTime {
			fmt.Fprintf(os.Stderr, "Modified file: %s\n", filePath)
			issues++
			continue
		}
	}

	if issues > 0 {
		return fmt.Errorf("found %d issues in index", issues)
	}

	fmt.Fprintf(os.Stderr, "Index validation passed\n")
	return nil
}

// printIndexStats prints indexing statistics
func printIndexStats(stats *IndexStats) {
	duration := stats.EndTime.Sub(stats.StartTime)

	fmt.Fprintf(os.Stderr, "\n=== Indexing Complete ===\n")
	fmt.Fprintf(os.Stderr, "Duration: %.2fs\n", duration.Seconds())
	fmt.Fprintf(os.Stderr, "Total files found: %d\n", stats.TotalFiles)
	fmt.Fprintf(os.Stderr, "Files indexed: %d\n", stats.IndexedFiles)
	fmt.Fprintf(os.Stderr, "Files skipped: %d\n", stats.SkippedFiles)
	fmt.Fprintf(os.Stderr, "Files with errors: %d\n", stats.ErrorFiles)
	fmt.Fprintf(os.Stderr, "Data indexed: %.2f KB\n", float64(stats.BytesIndexed)/1024)

	if stats.IndexedFiles > 0 {
		avgTime := duration.Seconds() / float64(stats.IndexedFiles)
		fmt.Fprintf(os.Stderr, "Average time per file: %.3fs\n", avgTime)
	}
}

// isTextFile checks if a file is text-based by looking for null bytes in the first 8KB
func isTextFile(filePath string) bool {
	file, err := os.Open(filePath)
	if err != nil {
		return false
	}
	defer file.Close()

	buffer := make([]byte, 8192)
	n, err := file.Read(buffer)
	if err != nil && err != io.EOF {
		return false
	}

	// Check for null bytes (indicates binary)
	for i := 0; i < n; i++ {
		if buffer[i] == 0 {
			return false
		}
	}

	return true
}

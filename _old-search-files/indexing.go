package main

import (
	"fmt"
	"os"
	"path/filepath"
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
func (se *SearchEngine) IndexRepository(showProgress bool, reindexAll bool) (*IndexStats, error) {
	stats := &IndexStats{
		StartTime: time.Now(),
	}

	if showProgress {
		fmt.Fprintf(os.Stderr, "Starting repository indexing...\n")
	}

	// Walk through repository
	err := filepath.Walk(se.session.Config.RepositoryRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		stats.TotalFiles++

		// Get relative path
		relPath, err := filepath.Rel(se.session.Config.RepositoryRoot, path)
		if err != nil {
			relPath = path
		}

		// Check if file should be indexed
		if !se.shouldIndexFile(relPath) {
			stats.SkippedFiles++
			return nil
		}

		// Check file size limit
		if info.Size() > se.config.MaxFileSize {
			stats.SkippedFiles++
			return nil
		}

		// Check if file is text
		if !isTextFile(path) {
			stats.SkippedFiles++
			return nil
		}

		// Show progress
		if showProgress {
			fmt.Fprintf(os.Stderr, "\rIndexing: %d/%d files (%d%%) - %s",
				stats.IndexedFiles+stats.SkippedFiles, stats.TotalFiles,
				((stats.IndexedFiles+stats.SkippedFiles)*100)/stats.TotalFiles,
				filepath.Base(relPath))
		}

		// Check if file needs indexing
		needsIndexing, err := se.fileNeedsIndexing(relPath, info, reindexAll)
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
		if err := se.indexFile(relPath, info); err != nil {
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
		se.printIndexStats(stats)
	}

	return stats, err
}

// fileNeedsIndexing checks if a file needs to be indexed or re-indexed
func (se *SearchEngine) fileNeedsIndexing(filePath string, info os.FileInfo, forceReindex bool) (bool, error) {
	if forceReindex {
		return true, nil
	}

	// Check if file exists in database
	existingInfo, err := se.getFileInfo(filePath)
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
func (se *SearchEngine) indexFile(filePath string, info os.FileInfo) error {
	// Read file content
	fullPath := filepath.Join(se.session.Config.RepositoryRoot, filePath)
	content, err := os.ReadFile(fullPath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	// Calculate content hash
	contentHash := fmt.Sprintf("%x", content)

	// Generate embedding
	embedding, err := se.generateEmbedding(string(content))
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
	if err := se.storeFileInfo(fileInfo); err != nil {
		return fmt.Errorf("failed to store file info: %w", err)
	}

	return nil
}

// UpdateIndex performs incremental update of the index
func (se *SearchEngine) UpdateIndex() error {
	// Get all files currently in database
	dbFiles, err := se.getAllIndexedFiles()
	if err != nil {
		return err
	}

	// Track files that still exist
	existingFiles := make(map[string]bool)

	// Walk through repository to find changed/new files
	err = filepath.Walk(se.session.Config.RepositoryRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		relPath, err := filepath.Rel(se.session.Config.RepositoryRoot, path)
		if err != nil {
			relPath = path
		}

		existingFiles[relPath] = true

		// Check if file should be indexed
		if !se.shouldIndexFile(relPath) {
			return nil
		}

		if info.Size() > se.config.MaxFileSize {
			return nil
		}

		if !isTextFile(path) {
			return nil
		}

		// Check if needs indexing
		needsIndexing, err := se.fileNeedsIndexing(relPath, info, false)
		if err != nil {
			return err
		}

		if needsIndexing {
			if err := se.indexFile(relPath, info); err != nil {
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
			se.removeFileInfo(dbFile)
		}
	}

	return nil
}

// getAllIndexedFiles returns all file paths currently in the database
func (se *SearchEngine) getAllIndexedFiles() ([]string, error) {
	rows, err := se.db.Query("SELECT filepath FROM embeddings")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var files []string
	for rows.Next() {
		var filepath string
		if err := rows.Scan(&filepath); err != nil {
			return nil, err
		}
		files = append(files, filepath)
	}

	return files, rows.Err()
}

// CleanupIndex removes entries for non-existent files
func (se *SearchEngine) CleanupIndex() error {
	files, err := se.getAllIndexedFiles()
	if err != nil {
		return err
	}

	for _, filePath := range files {
		fullPath := filepath.Join(se.session.Config.RepositoryRoot, filePath)
		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			se.removeFileInfo(filePath)
		}
	}

	return nil
}

// printIndexStats prints indexing statistics
func (se *SearchEngine) printIndexStats(stats *IndexStats) {
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

// GetIndexStats returns current index statistics
func (se *SearchEngine) GetIndexStats() (map[string]interface{}, error) {
	var totalFiles, totalSize int64
	var oldestIndex, newestIndex int64

	err := se.db.QueryRow(`
		SELECT COUNT(*), COALESCE(SUM(file_size), 0), 
		       COALESCE(MIN(indexed_at), 0), COALESCE(MAX(indexed_at), 0)
		FROM embeddings
	`).Scan(&totalFiles, &totalSize, &oldestIndex, &newestIndex)

	if err != nil {
		return nil, err
	}

	stats := map[string]interface{}{
		"total_files":   totalFiles,
		"total_size":    totalSize,
		"oldest_index":  time.Unix(oldestIndex, 0),
		"newest_index":  time.Unix(newestIndex, 0),
		"database_path": se.config.VectorDBPath,
		"model":         se.config.EmbeddingModel,
	}

	return stats, nil
}

// ValidateIndex checks the integrity of the search index
func (se *SearchEngine) ValidateIndex() error {
	// Check if all indexed files still exist and have correct hashes
	rows, err := se.db.Query("SELECT filepath, content_hash, last_modified FROM embeddings")
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

		fullPath := filepath.Join(se.session.Config.RepositoryRoot, filePath)

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

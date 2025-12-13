package search

import (
	"database/sql"
	"os"
	"path/filepath"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// FileInfo holds metadata about indexed files
type FileInfo struct {
	FilePath     string
	ContentHash  string
	Embedding    []float32
	LastModified int64
	FileSize     int64
	IndexedAt    int64
}

// InitSearchDB initializes the SQLite database for storing embeddings
func InitSearchDB(dbPath string) (*sql.DB, error) {
	// Create directory if it doesn't exist
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}

	// Create embeddings table
	schema := `
	CREATE TABLE IF NOT EXISTS embeddings (
		filepath TEXT PRIMARY KEY,
		content_hash TEXT NOT NULL,
		embedding BLOB NOT NULL,
		last_modified INTEGER NOT NULL,
		file_size INTEGER NOT NULL,
		indexed_at INTEGER NOT NULL
	);
	CREATE INDEX IF NOT EXISTS idx_hash ON embeddings(content_hash);
	CREATE INDEX IF NOT EXISTS idx_modified ON embeddings(last_modified);
	`

	if _, err := db.Exec(schema); err != nil {
		db.Close()
		return nil, err
	}

	return db, nil
}

// getFileInfo retrieves file metadata and embedding from database
func getFileInfo(db *sql.DB, filePath string) (*FileInfo, error) {
	var info FileInfo
	var embeddingData []byte

	err := db.QueryRow(`
		SELECT filepath, content_hash, embedding, last_modified, file_size, indexed_at 
		FROM embeddings WHERE filepath = ?
	`, filePath).Scan(
		&info.FilePath, &info.ContentHash, &embeddingData,
		&info.LastModified, &info.FileSize, &info.IndexedAt,
	)

	if err != nil {
		return nil, err
	}

	info.Embedding = deserializeEmbedding(embeddingData)
	return &info, nil
}

// storeFileInfo stores file metadata and embedding in database
func storeFileInfo(db *sql.DB, info *FileInfo) error {
	embeddingData := serializeEmbedding(info.Embedding)

	_, err := db.Exec(`
		INSERT OR REPLACE INTO embeddings 
		(filepath, content_hash, embedding, last_modified, file_size, indexed_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`, info.FilePath, info.ContentHash, embeddingData,
		info.LastModified, info.FileSize, info.IndexedAt)

	return err
}

// removeFileInfo removes file info from database (for deleted files)
func removeFileInfo(db *sql.DB, filePath string) error {
	_, err := db.Exec("DELETE FROM embeddings WHERE filepath = ?", filePath)
	return err
}

// getAllIndexedFiles returns all file paths currently in the database
func getAllIndexedFiles(db *sql.DB) ([]string, error) {
	rows, err := db.Query("SELECT filepath FROM embeddings")
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

// getIndexStats returns current index statistics
func getIndexStats(db *sql.DB) (map[string]interface{}, error) {
	var totalFiles, totalSize int64
	var oldestIndex, newestIndex int64

	err := db.QueryRow(`
		SELECT COUNT(*), COALESCE(SUM(file_size), 0), 
		       COALESCE(MIN(indexed_at), 0), COALESCE(MAX(indexed_at), 0)
		FROM embeddings
	`).Scan(&totalFiles, &totalSize, &oldestIndex, &newestIndex)

	if err != nil {
		return nil, err
	}

	stats := map[string]interface{}{
		"total_files":  totalFiles,
		"total_size":   totalSize,
		"oldest_index": time.Unix(oldestIndex, 0),
		"newest_index": time.Unix(newestIndex, 0),
	}

	return stats, nil
}

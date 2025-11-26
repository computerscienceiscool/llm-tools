package infrastructure

import (
	"database/sql"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
)

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

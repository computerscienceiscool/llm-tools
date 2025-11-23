package infrastructure

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
)

// Database handles database operations
type Database interface {
	Connect(dbPath string) error
	Close() error
	Execute(query string, args ...interface{}) error
	Query(query string, args ...interface{}) (*sql.Rows, error)
	QueryRow(query string, args ...interface{}) *sql.Row
}

// SQLiteDatabase implements Database for SQLite
type SQLiteDatabase struct {
	db *sql.DB
}

// NewDatabase creates a new database handler
func NewDatabase() Database {
	return &SQLiteDatabase{}
}

func (d *SQLiteDatabase) Connect(dbPath string) error {
	// Create directory if it doesn't exist
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create database directory: %w", err)
	}

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	d.db = db
	return nil
}

func (d *SQLiteDatabase) Close() error {
	if d.db != nil {
		return d.db.Close()
	}
	return nil
}

func (d *SQLiteDatabase) Execute(query string, args ...interface{}) error {
	_, err := d.db.Exec(query, args...)
	return err
}

func (d *SQLiteDatabase) Query(query string, args ...interface{}) (*sql.Rows, error) {
	return d.db.Query(query, args...)
}

func (d *SQLiteDatabase) QueryRow(query string, args ...interface{}) *sql.Row {
	return d.db.QueryRow(query, args...)
}

// InitializeSchema sets up database tables
func (d *SQLiteDatabase) InitializeSchema() error {
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

	return d.Execute(schema)
}

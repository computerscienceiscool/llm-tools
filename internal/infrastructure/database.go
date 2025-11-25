package infrastructure

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// Database handles database operations
type Database interface {
	Connect(dbPath string) error
	Close() error
	Execute(query string, args ...interface{}) error
	Query(query string, args ...interface{}) (*sql.Rows, error)
	QueryRow(query string, args ...interface{}) *sql.Row
	Initialize() error
	LogAuditEvent(sessionID, command, argument string, success bool, errorMsg string) error
	GetAuditLogs(sessionID string, limit int) ([]AuditLog, error)
}

// AuditLog represents an audit log entry
type AuditLog struct {
	ID        int64
	Timestamp time.Time
	SessionID string
	Command   string
	Argument  string
	Success   bool
	ErrorMsg  string
}

// SQLiteDatabase implements Database for SQLite
type SQLiteDatabase struct {
	db     *sql.DB
	dbPath string
}

// NewDatabase creates a new database handler
func NewDatabase() Database {
	return &SQLiteDatabase{}
}

// NewSQLiteDatabase creates a new SQLite database with a specific path
func NewSQLiteDatabase(dbPath string) Database {
	return &SQLiteDatabase{dbPath: dbPath}
}

func (d *SQLiteDatabase) Connect(dbPath string) error {
	// Create directory if it doesn't exist
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create database directory: %w", err)
	}

	db, err := sql.Open("sqlite3", dbPath+"?cache=shared&_journal_mode=WAL")
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	// Test the connection
	if err := db.Ping(); err != nil {
		db.Close()
		return fmt.Errorf("failed to ping database: %w", err)
	}

	d.db = db
	d.dbPath = dbPath
	return nil
}

func (d *SQLiteDatabase) Close() error {
	if d.db != nil {
		return d.db.Close()
	}
	return nil
}

func (d *SQLiteDatabase) Execute(query string, args ...interface{}) error {
	if d.db == nil {
		return fmt.Errorf("database not connected")
	}
	_, err := d.db.Exec(query, args...)
	return err
}

func (d *SQLiteDatabase) Query(query string, args ...interface{}) (*sql.Rows, error) {
	if d.db == nil {
		return nil, fmt.Errorf("database not connected")
	}
	return d.db.Query(query, args...)
}

func (d *SQLiteDatabase) QueryRow(query string, args ...interface{}) *sql.Row {
	if d.db == nil {
		// Return a row that will error when scanned
		return (&sql.DB{}).QueryRow("SELECT 1 WHERE 0=1")
	}
	return d.db.QueryRow(query, args...)
}

// Initialize sets up database tables
func (d *SQLiteDatabase) Initialize() error {
	// Auto-connect if we have a path but no connection
	if d.db == nil && d.dbPath != "" {
		if err := d.Connect(d.dbPath); err != nil {
			return fmt.Errorf("failed to auto-connect: %w", err)
		}
	}

	if d.db == nil {
		return fmt.Errorf("no database connection available")
	}

	schema := `
	CREATE TABLE IF NOT EXISTS audit_logs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		timestamp DATETIME NOT NULL,
		session_id TEXT NOT NULL,
		command TEXT NOT NULL,
		argument TEXT NOT NULL,
		success BOOLEAN NOT NULL,
		error_msg TEXT
	);
	CREATE INDEX IF NOT EXISTS idx_session ON audit_logs(session_id);
	CREATE INDEX IF NOT EXISTS idx_timestamp ON audit_logs(timestamp);
	`

	return d.Execute(schema)
}

// LogAuditEvent logs an audit event
func (d *SQLiteDatabase) LogAuditEvent(sessionID, command, argument string, success bool, errorMsg string) error {
	query := `
		INSERT INTO audit_logs (timestamp, session_id, command, argument, success, error_msg)
		VALUES (?, ?, ?, ?, ?, ?)
	`
	return d.Execute(query, time.Now(), sessionID, command, argument, success, errorMsg)
}

// GetAuditLogs retrieves audit logs for a session
func (d *SQLiteDatabase) GetAuditLogs(sessionID string, limit int) ([]AuditLog, error) {
	query := `
		SELECT id, timestamp, session_id, command, argument, success, error_msg
		FROM audit_logs
		WHERE session_id = ?
		ORDER BY timestamp DESC
		LIMIT ?
	`

	rows, err := d.Query(query, sessionID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []AuditLog
	for rows.Next() {
		var log AuditLog
		err := rows.Scan(&log.ID, &log.Timestamp, &log.SessionID, &log.Command, &log.Argument, &log.Success, &log.ErrorMsg)
		if err != nil {
			return nil, err
		}
		logs = append(logs, log)
	}

	return logs, nil
}

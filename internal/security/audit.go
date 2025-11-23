package security

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"
)

// AuditEntry represents a single audit log entry
type AuditEntry struct {
	Timestamp string `json:"timestamp"`
	SessionID string `json:"session_id"`
	Command   string `json:"command"`
	Argument  string `json:"argument"`
	Status    string `json:"status"`
	Message   string `json:"message"`
	UserAgent string `json:"user_agent,omitempty"`
	IPAddress string `json:"ip_address,omitempty"`
}

// AuditConfig holds audit logging configuration
type AuditConfig struct {
	Enabled    bool   `json:"enabled"`
	LogPath    string `json:"log_path"`
	Format     string `json:"format"` // "json" or "text"
	MaxSize    int64  `json:"max_size"`
	MaxBackups int    `json:"max_backups"`
}

// AuditManager manages audit logging operations
type AuditManager struct {
	config *AuditConfig
	logger *log.Logger
	file   *os.File
}

// NewAuditManager creates a new audit manager
func NewAuditManager(config *AuditConfig) (*AuditManager, error) {
	if !config.Enabled {
		return &AuditManager{config: config}, nil
	}

	file, err := os.OpenFile(config.LogPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open audit log: %w", err)
	}

	logger := log.New(file, "", 0)

	return &AuditManager{
		config: config,
		logger: logger,
		file:   file,
	}, nil
}

// LogEntry logs an audit entry
func (am *AuditManager) LogEntry(entry *AuditEntry) error {
	if !am.config.Enabled || am.logger == nil {
		return nil
	}

	entry.Timestamp = time.Now().Format(time.RFC3339)

	switch am.config.Format {
	case "json":
		data, err := json.Marshal(entry)
		if err != nil {
			return fmt.Errorf("failed to marshal audit entry: %w", err)
		}
		am.logger.Println(string(data))
	default:
		// Text format
		logLine := fmt.Sprintf("%s|session:%s|%s|%s|%s|%s",
			entry.Timestamp,
			entry.SessionID,
			entry.Command,
			entry.Argument,
			entry.Status,
			entry.Message)
		am.logger.Println(logLine)
	}

	return nil
}

// Close closes the audit manager
func (am *AuditManager) Close() error {
	if am.file != nil {
		return am.file.Close()
	}
	return nil
}

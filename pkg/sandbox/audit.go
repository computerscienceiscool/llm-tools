package sandbox

import (
	"fmt"
	"log"
	"os"
	"time"
)

// AuditLogger handles audit logging operations
type AuditLogger struct {
	logger *log.Logger
	file   *os.File
}

// NewAuditLogger creates a new audit logger
func NewAuditLogger(logPath string) (*AuditLogger, error) {
	file, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("could not open audit log: %w", err)
	}

	logger := log.New(file, "", 0)

	return &AuditLogger{
		logger: logger,
		file:   file,
	}, nil
}

// Log writes an audit log entry
func (a *AuditLogger) Log(sessionID, command, argument string, success bool, errorMsg string) {
	if a.logger == nil {
		return
	}

	status := "success"
	if !success {
		status = "failed"
	}

	logEntry := fmt.Sprintf("%s|session:%s|%s|%s|%s|%s",
		time.Now().Format(time.RFC3339),
		sessionID,
		command,
		argument,
		status,
		errorMsg,
	)

	a.logger.Println(logEntry)
}

// Close closes the audit log file
func (a *AuditLogger) Close() error {
	if a.file != nil {
		return a.file.Close()
	}
	return nil
}

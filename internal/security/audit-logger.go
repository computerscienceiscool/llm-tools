package security

import (
	"fmt"
	"log"
	"os"
	"time"
)

// DefaultAuditLogger implements AuditLogger
type DefaultAuditLogger struct {
	logger *log.Logger
}

// NewAuditLogger creates a new audit logger
func NewAuditLogger() AuditLogger {
	// Setup audit logging
	auditFile, err := os.OpenFile("audit.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		log.Printf("Warning: Could not open audit log: %v", err)
		// Return a logger that writes to stderr as fallback
		return &DefaultAuditLogger{
			logger: log.New(os.Stderr, "", 0),
		}
	}

	return &DefaultAuditLogger{
		logger: log.New(auditFile, "", 0),
	}
}

// LogOperation logs an operation for security audit
func (a *DefaultAuditLogger) LogOperation(sessionID, command, argument string, success bool, errorMsg string) {
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

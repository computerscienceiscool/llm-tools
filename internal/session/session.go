package session

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/computerscienceiscool/llm-runtime/internal/config"
)

// Session manages a tool execution session
type Session struct {
	ID          string
	Config      *config.Config
	CommandsRun int
	StartTime   time.Time
	AuditLogger *log.Logger
}

// NewSession creates a new execution session
func NewSession(cfg *config.Config) *Session {
	sessionID := fmt.Sprintf("%d", time.Now().UnixNano())

	// Setup audit logging
	auditFile, err := os.OpenFile("audit.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		log.Printf("Warning: Could not open audit log: %v", err)
	}

	auditLogger := log.New(auditFile, "", 0)

	return &Session{
		ID:          sessionID,
		Config:      cfg,
		StartTime:   time.Now(),
		AuditLogger: auditLogger,
	}
}

// LogAudit writes an audit log entry
func (s *Session) LogAudit(command, argument string, success bool, errorMsg string) {
	if s.AuditLogger == nil {
		return
	}

	status := "success"
	if !success {
		status = "failed"
	}

	logEntry := fmt.Sprintf("%s|session:%s|%s|%s|%s|%s",
		time.Now().Format(time.RFC3339),
		s.ID,
		command,
		argument,
		status,
		errorMsg,
	)

	s.AuditLogger.Println(logEntry)
}

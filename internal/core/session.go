package core

import (
	"fmt"
	"time"

	"github.com/computerscienceiscool/llm-tools/internal/security"
)

// DefaultSession implements Session interface
type DefaultSession struct {
	id           string
	config       *Config
	commandsRun  int
	startTime    time.Time
	auditLogger  security.AuditLogger
}

// NewSession creates a new execution session
func NewSession(config *Config) (Session, error) {
	sessionID := fmt.Sprintf("%d", time.Now().UnixNano())

	auditLogger := security.NewAuditLogger()

	return &DefaultSession{
		id:          sessionID,
		config:      config,
		startTime:   time.Now(),
		auditLogger: auditLogger,
	}, nil
}

// GetConfig returns the session configuration
func (s *DefaultSession) GetConfig() *Config {
	return s.config
}

// GetID returns the session ID
func (s *DefaultSession) GetID() string {
	return s.id
}

// LogAudit logs an audit entry
func (s *DefaultSession) LogAudit(command, argument string, success bool, errorMsg string) {
	if s.auditLogger != nil {
		s.auditLogger.LogOperation(s.id, command, argument, success, errorMsg)
	}
}

// IncrementCommandsRun increments the commands run counter
func (s *DefaultSession) IncrementCommandsRun() {
	s.commandsRun++
}

// GetCommandsRun returns the number of commands run
func (s *DefaultSession) GetCommandsRun() int {
	return s.commandsRun
}

// GetStartTime returns the session start time
func (s *DefaultSession) GetStartTime() time.Time {
	return s.startTime
}

package session

import (
	"log"
	"time"

	"github.com/computerscienceiscool/llm-tools/internal/config"
)

// Session manages a tool execution session
type Session struct {
	ID          string
	Config      *config.Config
	CommandsRun int
	StartTime   time.Time
	AuditLogger *log.Logger
}

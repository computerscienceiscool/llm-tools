package scanner

import (
	"time"
)

// Command represents a parsed command from LLM output
type Command struct {
	Type     string
	Argument string
	Content  string
	StartPos int
	EndPos   int
	Original string
}

// ExecutionResult holds the result of a command execution
type ExecutionResult struct {
	Command       Command
	Success       bool
	Result        string
	Error         error
	ExecutionTime time.Duration
	BytesWritten  int64
	BackupFile    string
	Action        string
	ExitCode      int
	Stdout        string
	Stderr        string
	ContainerID   string
}

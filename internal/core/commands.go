package core

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

// ExecutionResult holds the result of command execution
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

// CommandExecutor executes commands
type CommandExecutor interface {
	ExecuteOpen(filepath string) ExecutionResult
	ExecuteWrite(filepath, content string) ExecutionResult
	ExecuteExec(command string) ExecutionResult
	ExecuteSearch(query string) ExecutionResult
}

// Session holds runtime state
type Session interface {
	GetConfig() *Config
	GetID() string
	LogAudit(command, argument string, success bool, errorMsg string)
	IncrementCommandsRun()
	GetCommandsRun() int
	GetStartTime() time.Time
}

// Config represents the complete application configuration
type Config struct {
	RepositoryRoot      string
	MaxFileSize         int64
	MaxWriteSize        int64
	ExcludedPaths       []string
	Interactive         bool
	InputFile           string
	OutputFile          string
	JSONOutput          bool
	Verbose             bool
	RequireConfirmation bool
	BackupBeforeWrite   bool
	AllowedExtensions   []string
	ForceWrite          bool
	ExecEnabled         bool
	ExecWhitelist       []string
	ExecTimeout         time.Duration
	ExecMemoryLimit     string
	ExecCPULimit        int
	ExecContainerImage  string
	ExecNetworkEnabled  bool
}

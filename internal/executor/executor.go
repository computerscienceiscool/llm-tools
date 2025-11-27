package executor

import (
	"fmt"
	"sync"

	"github.com/computerscienceiscool/llm-tools/internal/command"
	"github.com/computerscienceiscool/llm-tools/internal/config"
	"github.com/computerscienceiscool/llm-tools/internal/search"
)

// Executor handles command execution
type Executor struct {
	config      *config.Config
	searchCfg   *search.SearchConfig
	auditLog    func(cmd, arg string, success bool, errMsg string)
	commandsRun int
	mu          sync.Mutex
}

// NewExecutor creates a new executor instance
func NewExecutor(cfg *config.Config, searchCfg *search.SearchConfig, auditLog func(cmd, arg string, success bool, errMsg string)) *Executor {
	return &Executor{
		config:    cfg,
		searchCfg: searchCfg,
		auditLog:  auditLog,
	}
}

// Execute dispatches command execution based on type
func (e *Executor) Execute(cmd command.Command) command.ExecutionResult {
	var result command.ExecutionResult

	switch cmd.Type {
	case "open":
		result = ExecuteOpen(cmd.Argument, e.config, e.auditLog)
	case "write":
		result = ExecuteWrite(cmd.Argument, cmd.Content, e.config, e.auditLog)
	case "exec":
		result = ExecuteExec(cmd.Argument, e.config, e.auditLog)
	case "search":
		result = ExecuteSearch(cmd.Argument, e.config, e.searchCfg, e.auditLog)
	default:
		result = command.ExecutionResult{
			Command: cmd,
			Success: false,
			Error:   fmt.Errorf("UNKNOWN_COMMAND: %s", cmd.Type),
		}
	}

	if result.Success {
		e.mu.Lock()
		e.commandsRun++
		e.mu.Unlock()
	}

	return result
}

// GetCommandsRun returns the number of successfully executed commands
func (e *Executor) GetCommandsRun() int {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.commandsRun
}

// GetConfig returns the executor's configuration
func (e *Executor) GetConfig() *config.Config {
	return e.config
}

// GetSearchConfig returns the executor's search configuration
func (e *Executor) GetSearchConfig() *search.SearchConfig {
	return e.searchCfg
}

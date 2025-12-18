package evaluator

import (
	"fmt"
	"sync"

	"github.com/computerscienceiscool/llm-runtime/pkg/config"
	"github.com/computerscienceiscool/llm-runtime/pkg/sandbox"
	"github.com/computerscienceiscool/llm-runtime/pkg/search"
	"github.com/computerscienceiscool/llm-runtime/pkg/scanner"
)

// Executor handles command execution
//
// Security Model:
// - Container is the security boundary (all mounted repo is sacrificial on compromise)
// - No path-level validation; LLM has unrestricted filesystem access within container
// - Symlinks are treated as normal filesystem objects
// - All operations audited to audit.log
// - Host protected by container namespace isolation and read-only mounts where appropriate
type Executor struct {
	config      *config.Config
	searchCfg   *search.SearchConfig
	auditLog    func(cmd, arg string, success bool, errMsg string)
	commandsRun int
	mu          sync.Mutex
	pool        *sandbox.ContainerPool
}

// NewExecutor creates a new executor instance
func NewExecutor(cfg *config.Config, searchCfg *search.SearchConfig, auditLog func(cmd, arg string, success bool, errMsg string), pool *sandbox.ContainerPool) *Executor {
	return &Executor{
		config:    cfg,
		searchCfg: searchCfg,
		auditLog:  auditLog,
		pool:      pool,
	}
}

// Execute dispatches command execution based on type
func (e *Executor) Execute(cmd scanner.Command) scanner.ExecutionResult {
	var result scanner.ExecutionResult

	switch cmd.Type {
	case "open":
		result = ExecuteOpen(cmd.Argument, e.config, e.auditLog, e.pool)
	case "write":
		result = ExecuteWrite(cmd.Argument, cmd.Content, e.config, e.auditLog, e.pool)
	case "exec":
		result = ExecuteExec(cmd, e.config, e.auditLog, e.pool)
	case "search":
		result = ExecuteSearch(cmd.Argument, e.config, e.searchCfg, e.auditLog, e.pool)
	default:
		result = scanner.ExecutionResult{
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

// GetPool returns the executor's container pool
func (e *Executor) GetPool() *sandbox.ContainerPool {
	return e.pool
}

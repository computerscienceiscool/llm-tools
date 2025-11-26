package app

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/computerscienceiscool/llm-tools/internal/config"
	"github.com/computerscienceiscool/llm-tools/internal/executor"
	"github.com/computerscienceiscool/llm-tools/internal/session"
)

// Bootstrap initializes and returns a configured App
func Bootstrap(cfg *config.Config) (*App, error) {
	// Resolve repository root to absolute path
	absRoot, err := filepath.Abs(cfg.RepositoryRoot)
	if err != nil {
		return nil, fmt.Errorf("cannot resolve repository root: %w", err)
	}
	cfg.RepositoryRoot = absRoot

	// Verify repository root exists
	if _, err := os.Stat(cfg.RepositoryRoot); err != nil {
		return nil, fmt.Errorf("repository root does not exist: %w", err)
	}

	// Create session
	sess := session.NewSession(cfg)

	// Load search configuration
	searchCfg := config.LoadSearchConfig()

	// Create executor with audit logging
	exec := executor.NewExecutor(cfg, searchCfg, sess.LogAudit)

	return &App{
		config:    cfg,
		session:   sess,
		executor:  exec,
		searchCfg: searchCfg,
	}, nil
}

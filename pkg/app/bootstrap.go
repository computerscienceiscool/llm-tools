package app

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/computerscienceiscool/llm-runtime/pkg/config"
	"github.com/computerscienceiscool/llm-runtime/pkg/sandbox"
	"github.com/computerscienceiscool/llm-runtime/pkg/evaluator"
	"github.com/computerscienceiscool/llm-runtime/pkg/session"
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

	// Create container pool if enabled
	var pool *sandbox.ContainerPool
	if cfg.ContainerPool.Enabled {
		poolConfig := sandbox.PoolConfig{
			Size:                cfg.ContainerPool.Size,
			MaxUsesPerContainer: cfg.ContainerPool.MaxUsesPerContainer,
			IdleTimeout:         cfg.ContainerPool.IdleTimeout,
			HealthCheckInterval: cfg.ContainerPool.HealthCheckInterval,
			StartupContainers:   cfg.ContainerPool.StartupContainers,
			Image:               cfg.IOContainerImage,
			MemoryLimit:         cfg.IOMemoryLimit,
			CPULimit:            cfg.IOCPULimit,
			RepoRoot:            cfg.RepositoryRoot,
		}
		var err error
		pool, err = sandbox.NewContainerPool(context.Background(), poolConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to create container pool: %w", err)
		}
	}

	
	// Create executor with audit logging
	exec := evaluator.NewExecutor(cfg, searchCfg, sess.LogAudit, pool)

	return &App{
		config:    cfg,
		session:   sess,
		executor:  exec,
		searchCfg: searchCfg,
	}, nil
}

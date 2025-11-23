package config

import (
	"github.com/computerscienceiscool/llm-tools/internal/core"
)

// ConfigLoader loads and validates configuration
type ConfigLoader interface {
	LoadConfig(configPath string) (*core.Config, error)
}

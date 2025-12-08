package config

import (
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/computerscienceiscool/llm-runtime/internal/search"
	"gopkg.in/yaml.v2"
)

// LoadConfig loads configuration from YAML file
func LoadConfig(configPath string) (*FullConfig, error) {
	// Set defaults
	config := &FullConfig{}
	SetFullConfigDefaults(config)

	// Load config file if it exists
	if _, err := os.Stat(configPath); err == nil {
		data, err := ioutil.ReadFile(configPath)
		if err != nil {
			return nil, err
		}

		if err := yaml.Unmarshal(data, config); err != nil {
			return nil, err
		}
	}

	return config, nil
}

// GetConfigPath returns the path to the configuration file
func GetConfigPath() string {
	// Check for config file in current directory first
	if _, err := os.Stat("llm-runtime.config.yaml"); err == nil {
		return "llm-runtime.config.yaml"
	}

	// Check home directory
	if home, err := os.UserHomeDir(); err == nil {
		homePath := filepath.Join(home, ".llm-runtime.config.yaml")
		if _, err := os.Stat(homePath); err == nil {
			return homePath
		}
	}

	// Default to current directory
	return "llm-runtime.config.yaml"
}

// GetSearchConfigFromFull extracts search configuration from FullConfig
func GetSearchConfigFromFull(fullConfig *FullConfig) *search.SearchConfig {
	return &search.SearchConfig{
		Enabled:            fullConfig.Commands.Search.Enabled,
		VectorDBPath:       fullConfig.Commands.Search.VectorDBPath,
		EmbeddingModel:     fullConfig.Commands.Search.EmbeddingModel,
		MaxResults:         fullConfig.Commands.Search.MaxResults,
		MinSimilarityScore: fullConfig.Commands.Search.MinSimilarityScore,
		MaxPreviewLength:   fullConfig.Commands.Search.MaxPreviewLength,
		ChunkSize:          fullConfig.Commands.Search.ChunkSize,
		OllamaURL:         fullConfig.Commands.Search.OllamaURL,
		IndexExtensions:    fullConfig.Commands.Search.IndexExtensions,
		MaxFileSize:        fullConfig.Commands.Search.MaxFileSize,
	}
}

// LoadSearchConfig loads search configuration from the default config file
func LoadSearchConfig() *search.SearchConfig {
	configPath := GetConfigPath()
	fullConfig, err := LoadConfig(configPath)
	if err != nil {
		// Return default config if file loading fails
		return GetDefaultSearchConfig()
	}

	return GetSearchConfigFromFull(fullConfig)
}

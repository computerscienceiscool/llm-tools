package config

import (
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/computerscienceiscool/llm-runtime/internal/search"
	"gopkg.in/yaml.v2"
)

// SaveConfig saves configuration to YAML file
func SaveConfig(config *FullConfig, configPath string) error {
	data, err := yaml.Marshal(config)
	if err != nil {
		return err
	}

	// Create directory if it doesn't exist
	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	return ioutil.WriteFile(configPath, data, 0644)
}

// UpdateSearchConfigInFile updates search configuration in existing config file
func UpdateSearchConfigInFile(configPath string, searchConfig *search.SearchConfig) error {
	fullConfig, err := LoadConfig(configPath)
	if err != nil {
		// Create new config if file doesn't exist
		fullConfig = &FullConfig{}
		SetFullConfigDefaults(fullConfig)
	}

	// Update search section
	fullConfig.Commands.Search.Enabled = searchConfig.Enabled
	fullConfig.Commands.Search.VectorDBPath = searchConfig.VectorDBPath
	fullConfig.Commands.Search.EmbeddingModel = searchConfig.EmbeddingModel
	fullConfig.Commands.Search.MaxResults = searchConfig.MaxResults
	fullConfig.Commands.Search.MinSimilarityScore = searchConfig.MinSimilarityScore
	fullConfig.Commands.Search.MaxPreviewLength = searchConfig.MaxPreviewLength
	fullConfig.Commands.Search.ChunkSize = searchConfig.ChunkSize
	fullConfig.Commands.Search.OllamaURL = searchConfig.OllamaURL
	fullConfig.Commands.Search.IndexExtensions = searchConfig.IndexExtensions
	fullConfig.Commands.Search.MaxFileSize = searchConfig.MaxFileSize

	return SaveConfig(fullConfig, configPath)
}

// EnableSearchInConfig enables search functionality in config file
func EnableSearchInConfig(configPath string) error {
	searchConfig := GetDefaultSearchConfig()
	searchConfig.Enabled = true

	return UpdateSearchConfigInFile(configPath, searchConfig)
}

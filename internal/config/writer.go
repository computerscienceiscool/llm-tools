package config

import (
	"os"
	"path/filepath"

	"github.com/computerscienceiscool/llm-runtime/internal/search"
	"github.com/spf13/viper"
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

	return os.WriteFile(configPath, data, 0644)
}

// UpdateSearchConfigInFile updates search configuration in existing config file
func UpdateSearchConfigInFile(configPath string, searchConfig *search.SearchConfig) error {
	// Try to read existing config
	v := viper.New()
	v.SetConfigFile(configPath)

	fullConfig := &FullConfig{}

	// If file exists, read it
	if _, err := os.Stat(configPath); err == nil {
		if err := v.ReadInConfig(); err != nil {
			return err
		}
		if err := v.Unmarshal(fullConfig); err != nil {
			return err
		}
	} else {
		// File doesn't exist, set defaults
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

// LoadSearchConfig loads search configuration from the default config file
func LoadSearchConfig() *search.SearchConfig {
	// Try to get config path
	configPath := GetConfigPath()

	v := viper.New()
	v.SetConfigFile(configPath)

	// Set defaults first
	SetViperDefaults()

	// Try to read config file
	if err := v.ReadInConfig(); err != nil {
		// If can't read, just return defaults
		return GetDefaultSearchConfig()
	}

	// Extract search config from viper
	return &search.SearchConfig{
		Enabled:            v.GetBool("commands.search.enabled"),
		VectorDBPath:       v.GetString("commands.search.vector_db_path"),
		EmbeddingModel:     v.GetString("commands.search.embedding_model"),
		MaxResults:         v.GetInt("commands.search.max_results"),
		MinSimilarityScore: v.GetFloat64("commands.search.min_similarity_score"),
		MaxPreviewLength:   v.GetInt("commands.search.max_preview_length"),
		ChunkSize:          v.GetInt("commands.search.chunk_size"),
		OllamaURL:          v.GetString("commands.search.ollama_url"),
		IndexExtensions:    v.GetStringSlice("commands.search.index_extensions"),
		MaxFileSize:        v.GetInt64("commands.search.max_file_size"),
	}
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

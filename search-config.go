package main

import (
	"io/ioutil"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v2"
)

// FullConfig represents the complete configuration structure including search
type FullConfig struct {
	Repository struct {
		Root          string   `yaml:"root"`
		ExcludedPaths []string `yaml:"excluded_paths"`
	} `yaml:"repository"`

	Commands struct {
		Open struct {
			Enabled           bool     `yaml:"enabled"`
			MaxFileSize       int64    `yaml:"max_file_size"`
			AllowedExtensions []string `yaml:"allowed_extensions"`
		} `yaml:"open"`

		Write struct {
			Enabled           bool  `yaml:"enabled"`
			MaxFileSize       int64 `yaml:"max_file_size"`
			BackupBeforeWrite bool  `yaml:"backup_before_write"`
		} `yaml:"write"`

		Exec struct {
			Enabled        bool     `yaml:"enabled"`
			ContainerImage string   `yaml:"container_image"`
			TimeoutSeconds int      `yaml:"timeout_seconds"`
			MemoryLimit    string   `yaml:"memory_limit"`
			CPULimit       int      `yaml:"cpu_limit"`
			Whitelist      []string `yaml:"whitelist"`
		} `yaml:"exec"`

		Search struct {
			Enabled            bool     `yaml:"enabled"`
			VectorDBPath       string   `yaml:"vector_db_path"`
			EmbeddingModel     string   `yaml:"embedding_model"`
			MaxResults         int      `yaml:"max_results"`
			MinSimilarityScore float64  `yaml:"min_similarity_score"`
			MaxPreviewLength   int      `yaml:"max_preview_length"`
			ChunkSize          int      `yaml:"chunk_size"`
			PythonPath         string   `yaml:"python_path"`
			IndexExtensions    []string `yaml:"index_extensions"`
			MaxFileSize        int64    `yaml:"max_file_size"`
		} `yaml:"search"`
	} `yaml:"commands"`

	Security struct {
		RateLimitPerMinute int    `yaml:"rate_limit_per_minute"`
		LogAllOperations   bool   `yaml:"log_all_operations"`
		AuditLogPath       string `yaml:"audit_log_path"`
	} `yaml:"security"`

	Output struct {
		ShowSummaries        bool `yaml:"show_summaries"`
		ShowExecutionTime    bool `yaml:"show_execution_time"`
		TruncateLargeOutputs bool `yaml:"truncate_large_outputs"`
		MaxOutputLines       int  `yaml:"max_output_lines"`
	} `yaml:"output"`

	Logging struct {
		Level  string `yaml:"level"`
		File   string `yaml:"file"`
		Format string `yaml:"format"`
	} `yaml:"logging"`
}

// LoadConfig loads configuration from YAML file
func LoadConfig(configPath string) (*FullConfig, error) {
	// Set defaults
	config := &FullConfig{}

	// Default repository settings
	config.Repository.Root = "."
	config.Repository.ExcludedPaths = []string{".git", ".env", "*.key", "*.pem"}

	// Default command settings
	config.Commands.Open.Enabled = true
	config.Commands.Open.MaxFileSize = 1048576 // 1MB
	config.Commands.Open.AllowedExtensions = []string{".go", ".py", ".js", ".md", ".txt", ".json", ".yaml"}

	config.Commands.Write.Enabled = true
	config.Commands.Write.MaxFileSize = 102400 // 100KB
	config.Commands.Write.BackupBeforeWrite = true

	config.Commands.Exec.Enabled = false
	config.Commands.Exec.ContainerImage = "ubuntu:22.04"
	config.Commands.Exec.TimeoutSeconds = 30
	config.Commands.Exec.MemoryLimit = "512m"
	config.Commands.Exec.CPULimit = 2
	config.Commands.Exec.Whitelist = []string{"go test", "go build", "npm test", "make"}

	// Default search settings
	config.Commands.Search.Enabled = false
	config.Commands.Search.VectorDBPath = "./embeddings.db"
	config.Commands.Search.EmbeddingModel = "all-MiniLM-L6-v2"
	config.Commands.Search.MaxResults = 10
	config.Commands.Search.MinSimilarityScore = 0.5
	config.Commands.Search.MaxPreviewLength = 100
	config.Commands.Search.ChunkSize = 1000
	config.Commands.Search.PythonPath = "python3"
	config.Commands.Search.IndexExtensions = []string{".go", ".py", ".js", ".md", ".txt", ".yaml", ".json"}
	config.Commands.Search.MaxFileSize = 1048576 // 1MB

	// Default security settings
	config.Security.RateLimitPerMinute = 100
	config.Security.LogAllOperations = true
	config.Security.AuditLogPath = "./audit.log"

	// Default output settings
	config.Output.ShowSummaries = true
	config.Output.ShowExecutionTime = true
	config.Output.TruncateLargeOutputs = true
	config.Output.MaxOutputLines = 1000

	// Default logging settings
	config.Logging.Level = "info"
	config.Logging.File = "./llm-tool.log"
	config.Logging.Format = "json"

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

// GetSearchConfig extracts search configuration from session
func (s *Session) GetSearchConfig() *SearchConfig {
	// Try to load from config file first
	configPath := "llm-tool.config.yaml"
	fullConfig, err := LoadConfig(configPath)
	if err != nil {
		// Return default config if file loading fails
		return getDefaultSearchConfig()
	}

	// Convert to SearchConfig
	return &SearchConfig{
		Enabled:            fullConfig.Commands.Search.Enabled,
		VectorDBPath:       fullConfig.Commands.Search.VectorDBPath,
		EmbeddingModel:     fullConfig.Commands.Search.EmbeddingModel,
		MaxResults:         fullConfig.Commands.Search.MaxResults,
		MinSimilarityScore: fullConfig.Commands.Search.MinSimilarityScore,
		MaxPreviewLength:   fullConfig.Commands.Search.MaxPreviewLength,
		ChunkSize:          fullConfig.Commands.Search.ChunkSize,
		PythonPath:         fullConfig.Commands.Search.PythonPath,
		IndexExtensions:    fullConfig.Commands.Search.IndexExtensions,
		MaxFileSize:        fullConfig.Commands.Search.MaxFileSize,
	}
}

// getDefaultSearchConfig returns default search configuration
func getDefaultSearchConfig() *SearchConfig {
	return &SearchConfig{
		Enabled:            false,
		VectorDBPath:       "./embeddings.db",
		EmbeddingModel:     "all-MiniLM-L6-v2",
		MaxResults:         10,
		MinSimilarityScore: 0.5,
		MaxPreviewLength:   100,
		ChunkSize:          1000,
		PythonPath:         "python3",
		IndexExtensions:    []string{".go", ".py", ".js", ".md", ".txt", ".yaml", ".json"},
		MaxFileSize:        1048576,
	}
}

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
func UpdateSearchConfigInFile(configPath string, searchConfig *SearchConfig) error {
	fullConfig, err := LoadConfig(configPath)
	if err != nil {
		// Create new config if file doesn't exist
		fullConfig = &FullConfig{}
		LoadConfig("") // Load defaults
	}

	// Update search section
	fullConfig.Commands.Search.Enabled = searchConfig.Enabled
	fullConfig.Commands.Search.VectorDBPath = searchConfig.VectorDBPath
	fullConfig.Commands.Search.EmbeddingModel = searchConfig.EmbeddingModel
	fullConfig.Commands.Search.MaxResults = searchConfig.MaxResults
	fullConfig.Commands.Search.MinSimilarityScore = searchConfig.MinSimilarityScore
	fullConfig.Commands.Search.MaxPreviewLength = searchConfig.MaxPreviewLength
	fullConfig.Commands.Search.ChunkSize = searchConfig.ChunkSize
	fullConfig.Commands.Search.PythonPath = searchConfig.PythonPath
	fullConfig.Commands.Search.IndexExtensions = searchConfig.IndexExtensions
	fullConfig.Commands.Search.MaxFileSize = searchConfig.MaxFileSize

	return SaveConfig(fullConfig, configPath)
}

// EnableSearchInConfig enables search functionality in config file
func EnableSearchInConfig(configPath string) error {
	searchConfig := getDefaultSearchConfig()
	searchConfig.Enabled = true

	return UpdateSearchConfigInFile(configPath, searchConfig)
}

// GetConfigPath returns the path to the configuration file
func GetConfigPath() string {
	// Check for config file in current directory first
	if _, err := os.Stat("llm-tool.config.yaml"); err == nil {
		return "llm-tool.config.yaml"
	}

	// Check home directory
	if home, err := os.UserHomeDir(); err == nil {
		homePath := filepath.Join(home, ".llm-tool.config.yaml")
		if _, err := os.Stat(homePath); err == nil {
			return homePath
		}
	}

	// Default to current directory
	return "llm-tool.config.yaml"
}

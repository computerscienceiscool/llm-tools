package config

import (
	"github.com/computerscienceiscool/llm-runtime/pkg/search"
	"github.com/spf13/viper"
)

// GetDefaultSearchConfig returns default search configuration
func getDefaultSearchConfig() *search.SearchConfig {
	return &search.SearchConfig{
		Enabled:             false,
		VectorDBPath:        "./embeddings.db",
		EmbeddingModel:      "all-MiniLM-L6-v2",
		EmbeddingDimensions: DefaultEmbeddingDims,
		MaxResults:          DefaultMaxSearchResults,
		MinSimilarityScore:  DefaultMinSimilarity,
		MaxPreviewLength:    100,
		ChunkSize:           1000,
		OllamaURL:           "http://localhost:11434",
		IndexExtensions:     []string{".go", ".py", ".js", ".md", ".txt", ".yaml", ".json"},
		MaxFileSize:         int64(DefaultMaxFileSize),
	}
}

// SetViperDefaults sets all default configuration values in Viper
func SetViperDefaults() {
	// Repository defaults
	viper.SetDefault("repository.root", ".")
	viper.SetDefault("repository.excluded_paths", []string{".git", ".env", "*.key", "*.pem"})

	// Command defaults - Open
	viper.SetDefault("commands.open.enabled", true)
	viper.SetDefault("commands.open.max_file_size", DefaultMaxFileSize)
	viper.SetDefault("commands.open.allowed_extensions", []string{".go", ".py", ".js", ".md", ".txt", ".json", ".yaml"})

	// Command defaults - Write
	viper.SetDefault("commands.write.enabled", true)
	viper.SetDefault("commands.write.max_file_size", DefaultMaxWriteSize)
	viper.SetDefault("commands.write.backup_before_write", true)

	// Command defaults - Exec
	viper.SetDefault("commands.exec.enabled", false)
	viper.SetDefault("commands.exec.container_image", "ubuntu:22.04")
	viper.SetDefault("commands.exec.timeout_seconds", int(DefaultExecTimeout.Seconds()))
	viper.SetDefault("commands.exec.memory_limit", DefaultContainerMemory)
	viper.SetDefault("commands.exec.cpu_limit", 2)
	viper.SetDefault("commands.exec.whitelist", []string{"go test", "go build", "npm test", "make"})

	// Command defaults - Search
	viper.SetDefault("commands.search.enabled", false)
	viper.SetDefault("commands.search.vector_db_path", "./embeddings.db")
	viper.SetDefault("commands.search.embedding_model", "all-MiniLM-L6-v2")
	viper.SetDefault("commands.search.embedding_dimensions", DefaultEmbeddingDims)
	viper.SetDefault("commands.search.max_results", DefaultMaxSearchResults)
	viper.SetDefault("commands.search.min_similarity_score", DefaultMinSimilarity)
	viper.SetDefault("commands.search.max_preview_length", 100)
	viper.SetDefault("commands.search.chunk_size", 1000)
	viper.SetDefault("commands.search.ollama_url", "http://localhost:11434")
	viper.SetDefault("commands.search.index_extensions", []string{".go", ".py", ".js", ".md", ".txt", ".yaml", ".json"})
	viper.SetDefault("commands.search.max_file_size", DefaultMaxFileSize)

	// Security defaults
	viper.SetDefault("security.rate_limit_per_minute", 100)
	viper.SetDefault("security.log_all_operations", true)
	viper.SetDefault("security.audit_log_path", DefaultAuditLogPath)

	// Output defaults
	viper.SetDefault("output.show_summaries", true)
	viper.SetDefault("output.show_execution_time", true)
	viper.SetDefault("output.truncate_large_outputs", true)
	viper.SetDefault("output.max_output_lines", 1000)

	// Logging defaults
	viper.SetDefault("logging.level", "info")
	viper.SetDefault("logging.file", "./llm-runtime.log")
	viper.SetDefault("logging.format", "json")
}

// SetFullConfigDefaults sets default values on a FullConfig struct (deprecated, use SetViperDefaults)
func setFullConfigDefaults(config *fullConfig) {
	// Default repository settings
	config.Repository.Root = "."
	config.Repository.ExcludedPaths = []string{".git", ".env", "*.key", "*.pem"}

	// Default command settings
	config.Commands.Open.Enabled = true
	config.Commands.Open.MaxFileSize = DefaultMaxFileSize
	config.Commands.Open.AllowedExtensions = []string{".go", ".py", ".js", ".md", ".txt", ".json", ".yaml"}

	config.Commands.Write.Enabled = true
	config.Commands.Write.MaxFileSize = DefaultMaxWriteSize
	config.Commands.Write.BackupBeforeWrite = true

	config.Commands.Exec.Enabled = false
	config.Commands.Exec.ContainerImage = "ubuntu:22.04"
	config.Commands.Exec.TimeoutSeconds = int(DefaultExecTimeout.Seconds())
	config.Commands.Exec.MemoryLimit = DefaultContainerMemory
	config.Commands.Exec.CPULimit = 2
	config.Commands.Exec.Whitelist = []string{"go test", "go build", "npm test", "make"}

	// Default search settings
	config.Commands.Search.Enabled = false
	config.Commands.Search.VectorDBPath = "./embeddings.db"
	config.Commands.Search.EmbeddingModel = "all-MiniLM-L6-v2"
	config.Commands.Search.MaxResults = DefaultMaxSearchResults
	config.Commands.Search.MinSimilarityScore = DefaultMinSimilarity
	config.Commands.Search.MaxPreviewLength = 100
	config.Commands.Search.ChunkSize = 1000
	config.Commands.Search.OllamaURL = "http://localhost:11434"
	config.Commands.Search.IndexExtensions = []string{".go", ".py", ".js", ".md", ".txt", ".yaml", ".json"}
	config.Commands.Search.MaxFileSize = int64(DefaultMaxFileSize)

	// Default security settings
	config.Security.RateLimitPerMinute = 100
	config.Security.LogAllOperations = true
	config.Security.AuditLogPath = DefaultAuditLogPath

	// Default output settings
	config.Output.ShowSummaries = true
	config.Output.ShowExecutionTime = true
	config.Output.TruncateLargeOutputs = true
	config.Output.MaxOutputLines = 1000

	// Default logging settings
	config.Logging.Level = "info"
	config.Logging.File = "./llm-runtime.log"
	config.Logging.Format = "json"
}

// LoadSearchConfig loads search configuration from viper (config file + defaults)
func LoadSearchConfig() *search.SearchConfig {
	cfg := getDefaultSearchConfig()

	// Override with values from config file if present
	if viper.IsSet("commands.search.enabled") {
		cfg.Enabled = viper.GetBool("commands.search.enabled")
	}
	if viper.IsSet("commands.search.vector_db_path") {
		cfg.VectorDBPath = viper.GetString("commands.search.vector_db_path")
	}
	if viper.IsSet("commands.search.embedding_model") {
		cfg.EmbeddingModel = viper.GetString("commands.search.embedding_model")
	}
	if viper.IsSet("commands.search.embedding_dimensions") {
		cfg.EmbeddingDimensions = viper.GetInt("commands.search.embedding_dimensions")
	}
	if viper.IsSet("commands.search.max_results") {
		cfg.MaxResults = viper.GetInt("commands.search.max_results")
	}
	if viper.IsSet("commands.search.min_similarity_score") {
		cfg.MinSimilarityScore = viper.GetFloat64("commands.search.min_similarity_score")
	}
	if viper.IsSet("commands.search.max_preview_length") {
		cfg.MaxPreviewLength = viper.GetInt("commands.search.max_preview_length")
	}
	if viper.IsSet("commands.search.chunk_size") {
		cfg.ChunkSize = viper.GetInt("commands.search.chunk_size")
	}
	if viper.IsSet("commands.search.ollama_url") {
		cfg.OllamaURL = viper.GetString("commands.search.ollama_url")
	}
	if viper.IsSet("commands.search.index_extensions") {
		cfg.IndexExtensions = viper.GetStringSlice("commands.search.index_extensions")
	}
	if viper.IsSet("commands.search.max_file_size") {
		cfg.MaxFileSize = viper.GetInt64("commands.search.max_file_size")
	}

	return cfg
}

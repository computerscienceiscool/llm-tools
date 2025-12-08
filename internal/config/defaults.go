package config

import (
	"github.com/computerscienceiscool/llm-runtime/internal/search"
)

// GetDefaultSearchConfig returns default search configuration
func GetDefaultSearchConfig() *search.SearchConfig {
	return &search.SearchConfig{
		Enabled:            false,
		VectorDBPath:       "./embeddings.db",
		EmbeddingModel:     "all-MiniLM-L6-v2",
		MaxResults:         10,
		MinSimilarityScore: 0.5,
		MaxPreviewLength:   100,
		ChunkSize:          1000,
		OllamaURL:         "python3",
		IndexExtensions:    []string{".go", ".py", ".js", ".md", ".txt", ".yaml", ".json"},
		MaxFileSize:        1048576,
	}
}

// SetFullConfigDefaults sets default values on a FullConfig struct
func SetFullConfigDefaults(config *FullConfig) {
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
	config.Commands.Search.OllamaURL = "python3"
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
	config.Logging.File = "./llm-runtime.log"
	config.Logging.Format = "json"
}

package config

import (
	"time"
)

// Config holds the tool configuration
type Config struct {
	RepositoryRoot      string
	MaxFileSize         int64
	MaxWriteSize        int64
	ExcludedPaths       []string
	Interactive         bool
	InputFile           string
	OutputFile          string
	JSONOutput          bool
	Verbose             bool
	RequireConfirmation bool
	BackupBeforeWrite   bool
	AllowedExtensions   []string
	ForceWrite          bool
	ExecEnabled         bool
	ExecWhitelist       []string
	ExecTimeout         time.Duration
	ExecMemoryLimit     string
	ExecCPULimit        int
	ExecContainerImage  string
	ExecNetworkEnabled  bool
	IOContainerized     bool
	IOContainerImage    string
	IOTimeout           time.Duration
	IOMemoryLimit       string
	IOCPULimit          int
}

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
			OllamaURL          string   `yaml:"ollama_url"`
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

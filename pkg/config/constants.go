package config

import "time"

// Default values and limits for the LLM runtime
const (
	// File size limits
	DefaultMaxFileSize    = 1 * 1024 * 1024  // 1MB - maximum file size for read operations
	DefaultMaxWriteSize   = 100 * 1024       // 100KB - maximum write content size
	DefaultScanBufferSize = 10 * 1024 * 1024 // 10MB - maximum scanner buffer size

	// Timeout values
	DefaultIOTimeout   = 30 * time.Second // Timeout for I/O container operations
	DefaultExecTimeout = 30 * time.Second // Timeout for exec container operations

	// Container resource limits
	DefaultContainerMemory = "512m" // Memory limit per container
	DefaultContainerCPUs   = "1.0"  // CPU limit per container

	// Search configuration
	DefaultMaxSearchResults = 10  // Maximum number of search results to return
	DefaultMinSimilarity    = 0.7 // Minimum similarity score for search results
	DefaultEmbeddingDims    = 768 // Default embedding dimensions (nomic-embed-text)

	// Validation limits
	MaxCommandLength = 1000 // Maximum length for exec commands
	MaxPathLength    = 4096 // Maximum path length

	// Backup configuration
	BackupExtension = ".bak" // Extension for backup files
	MaxBackups      = 5      // Maximum number of backups to keep per file

	// Audit log configuration
	DefaultAuditLogPath = "audit.log"
	AuditLogMaxSize     = 100 // MB
	AuditLogMaxBackups  = 5
	AuditLogMaxAge      = 30 // days

	// Session configuration
	DefaultSessionTimeout = 24 * time.Hour // Session timeout duration
	MaxSessionsPerUser    = 10             // Maximum concurrent sessions per user
)

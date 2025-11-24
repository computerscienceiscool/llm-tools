package config

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/computerscienceiscool/llm-tools/internal/core"
)

// DefaultConfigLoader implements ConfigLoader
type DefaultConfigLoader struct{}

// NewConfigLoader creates a new configuration loader
func NewConfigLoader() ConfigLoader {
	return &DefaultConfigLoader{}
}

// LoadConfig loads configuration from command line flags and config files
func (l *DefaultConfigLoader) LoadConfig(configPath string) (*core.Config, error) {
	config := &core.Config{}

	// Parse command-line flags only if not already parsed
	if !flag.Parsed() {
		if err := l.parseFlags(config); err != nil {
			return nil, err
		}
	} else {
		// Set default values when flags already parsed
		config.RepositoryRoot = "."
		config.MaxFileSize = 1048576
		config.MaxWriteSize = 102400
		config.ExecTimeout = 30 * time.Second
		config.ExecMemoryLimit = "512m"
		config.ExecCPULimit = 2
		config.BackupBeforeWrite = true
		config.AllowedExtensions = []string{".go", ".py", ".js", ".md", ".txt", ".json", ".yaml", ".yml", ".toml"}
		config.ExecWhitelist = []string{"go test", "go build", "go run", "npm test", "npm run build", "python -m pytest", "make", "cargo build", "cargo test"}
		config.ExcludedPaths = []string{".git", ".env", "*.key", "*.pem"}
	}

	// Resolve repository root to absolute path
	absRoot, err := filepath.Abs(config.RepositoryRoot)
	if err != nil {
		return nil, fmt.Errorf("cannot resolve repository root: %w", err)
	}
	config.RepositoryRoot = absRoot

	// Verify repository root exists
	if _, err := os.Stat(config.RepositoryRoot); err != nil {
		return nil, fmt.Errorf("repository root does not exist: %w", err)
	}

	return config, nil
}

// parseFlags parses command line flags into configuration
func (l *DefaultConfigLoader) parseFlags(config *core.Config) error {
	// Define flags
	flag.StringVar(&config.RepositoryRoot, "root", ".", "Repository root directory")
	flag.Int64Var(&config.MaxFileSize, "max-size", 1048576, "Maximum file size in bytes (default 1MB)")
	flag.BoolVar(&config.Interactive, "interactive", false, "Run in interactive mode")
	flag.StringVar(&config.InputFile, "input", "", "Input file (default: stdin)")
	flag.StringVar(&config.OutputFile, "output", "", "Output file (default: stdout)")
	flag.BoolVar(&config.JSONOutput, "json", false, "Output in JSON format")
	flag.BoolVar(&config.Verbose, "verbose", false, "Verbose output")
	flag.Int64Var(&config.MaxWriteSize, "max-write-size", 102400, "Maximum file size in bytes for writing (default 100KB)")
	flag.BoolVar(&config.RequireConfirmation, "require-confirmation", false, "Require confirmation for write operations")
	flag.BoolVar(&config.BackupBeforeWrite, "backup", true, "Create backup before overwriting files")
	flag.BoolVar(&config.ForceWrite, "force", false, "Force write even if conflicts exist")

	// Exec flags
	flag.BoolVar(&config.ExecEnabled, "exec-enabled", false, "Enable exec command")
	execTimeoutStr := flag.String("exec-timeout", "30s", "Timeout for exec commands")
	flag.StringVar(&config.ExecMemoryLimit, "exec-memory", "512m", "Memory limit for containers")
	flag.IntVar(&config.ExecCPULimit, "exec-cpu", 2, "CPU limit for containers")
	flag.StringVar(&config.ExecContainerImage, "exec-image", "ubuntu:22.04", "Docker image for exec commands")
	flag.BoolVar(&config.ExecNetworkEnabled, "exec-network", false, "Enable network access in containers")

	// Extension and whitelist flags
	allowedExts := flag.String("allowed-extensions", ".go,.py,.js,.md,.txt,.json,.yaml,.yml,.toml",
		"Comma-separated list of allowed file extensions for writing")
	execWhitelistStr := flag.String("exec-whitelist", "go test,go build,go run,npm test,npm run build,python -m pytest,make,cargo build,cargo test",
		"Comma-separated list of allowed exec commands")

	// Excluded paths
	excludedPaths := flag.String("exclude", ".git,.env,*.key,*.pem", "Comma-separated list of excluded paths")

	// Parse flags
	flag.Parse()

	// Parse timeout
	var err error
	config.ExecTimeout, err = time.ParseDuration(*execTimeoutStr)
	if err != nil {
		return fmt.Errorf("invalid exec timeout: %w", err)
	}

	// Set up allowed extensions with proper empty handling
	config.AllowedExtensions = parseStringList(*allowedExts)

	// Set up exec whitelist with proper empty handling
	config.ExecWhitelist = parseStringList(*execWhitelistStr)

	// Set up excluded paths with proper empty handling
	config.ExcludedPaths = parseStringList(*excludedPaths)

	return nil
}

// parseStringList parses a comma-separated string into a slice, handling empty strings correctly
func parseStringList(input string) []string {
	if input == "" {
		return []string{""}
	}

	parts := strings.Split(input, ",")
	result := make([]string, 0, len(parts))

	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		result = append(result, trimmed)
	}

	// If all parts were empty after trimming and we started with an empty string
	if len(result) == 1 && result[0] == "" && input == "" {
		return []string{""}
	}

	return result
}

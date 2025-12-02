package cli

import (
	"flag"
	"log"
	"strings"
	"time"

	"github.com/computerscienceiscool/llm-runtime/internal/config"
)

// CLIFlags holds parsed command-line flags
type CLIFlags struct {
	Config           *config.Config
	Reindex          bool
	SearchStatus     bool
	SearchValidate   bool
	SearchCleanup    bool
	SearchUpdate     bool
	CheckPythonSetup bool
}

// ParseFlags parses command-line flags and returns configuration
func ParseFlags() *CLIFlags {
	cfg := &config.Config{}
	flags := &CLIFlags{Config: cfg}

	flag.StringVar(&cfg.RepositoryRoot, "root", ".", "Repository root directory")
	flag.Int64Var(&cfg.MaxFileSize, "max-size", 1048576, "Maximum file size in bytes (default 1MB)")
	flag.BoolVar(&cfg.Interactive, "interactive", false, "Run in interactive mode")
	flag.StringVar(&cfg.InputFile, "input", "", "Input file (default: stdin)")
	flag.StringVar(&cfg.OutputFile, "output", "", "Output file (default: stdout)")
	flag.BoolVar(&cfg.JSONOutput, "json", false, "Output in JSON format")
	flag.BoolVar(&cfg.Verbose, "verbose", false, "Verbose output")
	flag.Int64Var(&cfg.MaxWriteSize, "max-write-size", 102400, "Maximum file size in bytes for writing (default 100KB)")
	flag.BoolVar(&cfg.RequireConfirmation, "require-confirmation", false, "Require confirmation for write operations")
	flag.BoolVar(&cfg.BackupBeforeWrite, "backup", true, "Create backup before overwriting files")
	flag.BoolVar(&cfg.ForceWrite, "force", false, "Force write even if conflicts exist")

	flag.BoolVar(&cfg.ExecEnabled, "exec-enabled", false, "Enable exec command")
	execTimeoutStr := flag.String("exec-timeout", "30s", "Timeout for exec commands")
	flag.StringVar(&cfg.ExecMemoryLimit, "exec-memory", "512m", "Memory limit for containers")
	flag.IntVar(&cfg.ExecCPULimit, "exec-cpu", 2, "CPU limit for containers")
	flag.StringVar(&cfg.ExecContainerImage, "exec-image", "ubuntu:22.04", "Docker image for exec commands")
	flag.BoolVar(&cfg.ExecNetworkEnabled, "exec-network", false, "Enable network access in containers")

	allowedExts := flag.String("allowed-extensions", ".go,.py,.js,.md,.txt,.json,.yaml,.yml,.toml",
		"Comma-separated list of allowed file extensions for writing")
	execWhitelistStr := flag.String("exec-whitelist", "go test,go build,go run,npm test,npm run build,python -m pytest,make,cargo build,cargo test",
		"Comma-separated list of allowed exec commands")

	// Parse excluded paths
	excludedPaths := flag.String("exclude", ".git,.env,*.key,*.pem", "Comma-separated list of excluded paths")

	// Search-related flags
	flag.BoolVar(&flags.Reindex, "reindex", false, "Rebuild search index from scratch")
	flag.BoolVar(&flags.SearchStatus, "search-status", false, "Show search index status")
	flag.BoolVar(&flags.SearchValidate, "search-validate", false, "Validate search index")
	flag.BoolVar(&flags.SearchCleanup, "search-cleanup", false, "Clean up search index")
	flag.BoolVar(&flags.SearchUpdate, "search-update", false, "Update search index incrementally")
	flag.BoolVar(&flags.CheckPythonSetup, "check-python-setup", false, "Check Python dependencies for search")

	flag.Parse()

	// Parse timeout
	var err error
	cfg.ExecTimeout, err = time.ParseDuration(*execTimeoutStr)
	if err != nil {
		log.Fatalf("Invalid exec timeout: %v", err)
	}

	// Set up allowed extensions
	if *allowedExts != "" {
		cfg.AllowedExtensions = strings.Split(*allowedExts, ",")
		for i := range cfg.AllowedExtensions {
			cfg.AllowedExtensions[i] = strings.TrimSpace(cfg.AllowedExtensions[i])
		}
	}

	// Set up exec whitelist
	if *execWhitelistStr != "" {
		cfg.ExecWhitelist = strings.Split(*execWhitelistStr, ",")
		for i := range cfg.ExecWhitelist {
			cfg.ExecWhitelist[i] = strings.TrimSpace(cfg.ExecWhitelist[i])
		}
	}

	// Set up excluded paths
	cfg.ExcludedPaths = strings.Split(*excludedPaths, ",")
	for i := range cfg.ExcludedPaths {
		cfg.ExcludedPaths[i] = strings.TrimSpace(cfg.ExcludedPaths[i])
	}

	return flags
}

// HasSearchCommand returns true if any search-related flag is set
func (f *CLIFlags) HasSearchCommand() bool {
	return f.Reindex || f.SearchStatus || f.SearchValidate || f.SearchCleanup || f.SearchUpdate || f.CheckPythonSetup
}

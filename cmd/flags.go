package cmd

import (
	"flag"
	"strings"
	"time"

	"github.com/computerscienceiscool/llm-tools/internal/core"
)

func setupFlags() *core.Config {
	config := &core.Config{}

	flag.StringVar(&config.RepositoryRoot, "root", ".", "Repository root directory")
	flag.Int64Var(&config.MaxFileSize, "max-size", 1048576, "Maximum file size in bytes")
	flag.BoolVar(&config.Interactive, "interactive", false, "Run in interactive mode")
	flag.StringVar(&config.InputFile, "input", "", "Input file")
	flag.StringVar(&config.OutputFile, "output", "", "Output file")
	flag.BoolVar(&config.JSONOutput, "json", false, "JSON output")
	flag.BoolVar(&config.Verbose, "verbose", false, "Verbose output")
	flag.BoolVar(&config.ExecEnabled, "exec-enabled", false, "Enable exec commands")

	execTimeoutStr := flag.String("exec-timeout", "30s", "Exec timeout")
	flag.StringVar(&config.ExecMemoryLimit, "exec-memory", "512m", "Memory limit")
	flag.IntVar(&config.ExecCPULimit, "exec-cpu", 2, "CPU limit")

	allowedExts := flag.String("allowed-extensions", ".go,.py,.js,.md,.txt", "Allowed extensions")
	execWhitelist := flag.String("exec-whitelist", "go test,go build,make", "Exec whitelist")
	excludedPaths := flag.String("exclude", ".git,.env,*.key", "Excluded paths")

	flag.Parse()

	// Parse duration
	if timeout, err := time.ParseDuration(*execTimeoutStr); err == nil {
		config.ExecTimeout = timeout
	}

	// Parse lists with proper trimming
	if *allowedExts != "" {
		parts := strings.Split(*allowedExts, ",")
		config.AllowedExtensions = make([]string, 0, len(parts))
		for _, part := range parts {
			trimmed := strings.TrimSpace(part)
			if trimmed != "" {
				config.AllowedExtensions = append(config.AllowedExtensions, trimmed)
			}
		}
		// Handle empty string case
		if *allowedExts == "" {
			config.AllowedExtensions = []string{""}
		}
	} else {
		config.AllowedExtensions = []string{""}
	}

	if *execWhitelist != "" {
		parts := strings.Split(*execWhitelist, ",")
		config.ExecWhitelist = make([]string, 0, len(parts))
		for _, part := range parts {
			trimmed := strings.TrimSpace(part)
			if trimmed != "" {
				config.ExecWhitelist = append(config.ExecWhitelist, trimmed)
			}
		}
		// Handle empty string case
		if *execWhitelist == "" {
			config.ExecWhitelist = []string{""}
		}
	} else {
		config.ExecWhitelist = []string{""}
	}

	if *excludedPaths != "" {
		parts := strings.Split(*excludedPaths, ",")
		config.ExcludedPaths = make([]string, 0, len(parts))
		for _, part := range parts {
			trimmed := strings.TrimSpace(part)
			if trimmed != "" {
				config.ExcludedPaths = append(config.ExcludedPaths, trimmed)
			}
		}
		// Handle empty string case
		if *excludedPaths == "" {
			config.ExcludedPaths = []string{""}
		}
	} else {
		config.ExcludedPaths = []string{""}
	}

	return config
}

package cmd

import (
	"flag"
	"os"
	"testing"
	"time"

	"github.com/computerscienceiscool/llm-tools/internal/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSetupFlags tests the flag parsing functionality
func TestSetupFlags(t *testing.T) {
	// Save original command line args and restore after test
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	tests := []struct {
		name     string
		args     []string
		validate func(t *testing.T, config *core.Config)
	}{
		{
			name: "default values",
			args: []string{"llm-tool"},
			validate: func(t *testing.T, config *core.Config) {
				assert.Equal(t, ".", config.RepositoryRoot)
				assert.Equal(t, int64(1048576), config.MaxFileSize)
				assert.False(t, config.Interactive)
				assert.False(t, config.ExecEnabled)
				assert.Equal(t, "30s", config.ExecTimeout.String())
				assert.Equal(t, "512m", config.ExecMemoryLimit)
				assert.Equal(t, 2, config.ExecCPULimit)
			},
		},
		{
			name: "custom values",
			args: []string{
				"llm-tool",
				"-root", "/custom/path",
				"-max-size", "2097152",
				"-interactive",
				"-exec-enabled",
				"-exec-timeout", "60s",
				"-exec-memory", "1g",
				"-exec-cpu", "4",
				"-verbose",
			},
			validate: func(t *testing.T, config *core.Config) {
				assert.Equal(t, "/custom/path", config.RepositoryRoot)
				assert.Equal(t, int64(2097152), config.MaxFileSize)
				assert.True(t, config.Interactive)
				assert.True(t, config.ExecEnabled)
				assert.True(t, config.Verbose)
				assert.Equal(t, time.Minute, config.ExecTimeout)
				assert.Equal(t, "1g", config.ExecMemoryLimit)
				assert.Equal(t, 4, config.ExecCPULimit)
			},
		},
		{
			name: "input and output files",
			args: []string{
				"llm-tool",
				"-input", "input.txt",
				"-output", "output.txt",
				"-json",
			},
			validate: func(t *testing.T, config *core.Config) {
				assert.Equal(t, "input.txt", config.InputFile)
				assert.Equal(t, "output.txt", config.OutputFile)
				assert.True(t, config.JSONOutput)
			},
		},
		{
			name: "extension lists",
			args: []string{
				"llm-tool",
				"-allowed-extensions", ".go,.py,.js",
				"-exec-whitelist", "go test,npm test",
				"-exclude", ".git,.env",
			},
			validate: func(t *testing.T, config *core.Config) {
				assert.Equal(t, []string{".go", ".py", ".js"}, config.AllowedExtensions)
				assert.Equal(t, []string{"go test", "npm test"}, config.ExecWhitelist)
				assert.Equal(t, []string{".git", ".env"}, config.ExcludedPaths)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset flag package state
			flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)

			// Set test args
			os.Args = tt.args

			// Call setupFlags
			config := setupFlags()

			// Validate results
			require.NotNil(t, config)
			tt.validate(t, config)
		})
	}
}

// TestInvalidDurations tests invalid duration parsing
func TestInvalidDurations(t *testing.T) {
	// Save original command line args and restore after test
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	// Reset flag package state
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)

	// Set invalid duration
	os.Args = []string{"llm-tool", "-exec-timeout", "invalid"}

	config := setupFlags()

	// Should fall back to zero value when parsing fails
	assert.Equal(t, time.Duration(0), config.ExecTimeout)
}

// TestEmptyLists tests behavior with empty list parameters
func TestEmptyLists(t *testing.T) {
	// Save original command line args and restore after test
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	// Reset flag package state
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)

	os.Args = []string{
		"llm-tool",
		"-allowed-extensions", "",
		"-exec-whitelist", "",
		"-exclude", "",
	}

	config := setupFlags()

	// Empty strings should result in empty slices after splitting
	assert.Equal(t, []string{""}, config.AllowedExtensions)
	assert.Equal(t, []string{""}, config.ExecWhitelist)
	assert.Equal(t, []string{""}, config.ExcludedPaths)
}

// TestListTrimming tests that list items are properly trimmed
func TestListTrimming(t *testing.T) {
	// Save original command line args and restore after test
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	// Reset flag package state
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)

	os.Args = []string{
		"llm-tool",
		"-allowed-extensions", " .go , .py , .js ",
		"-exec-whitelist", " go test , npm test , make ",
	}

	config := setupFlags()

	// Items should be trimmed of whitespace
	assert.Equal(t, []string{".go", ".py", ".js"}, config.AllowedExtensions)
	assert.Equal(t, []string{"go test", "npm test", "make"}, config.ExecWhitelist)
}

// TestFlagDefaults tests that all flags have reasonable defaults
func TestFlagDefaults(t *testing.T) {
	// Save original command line args and restore after test
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	// Reset flag package state
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)

	// Use only program name, no flags
	os.Args = []string{"llm-tool"}

	config := setupFlags()

	// Test all default values
	assert.Equal(t, ".", config.RepositoryRoot)
	assert.Equal(t, int64(1048576), config.MaxFileSize)
	assert.False(t, config.Interactive)
	assert.Equal(t, "", config.InputFile)
	assert.Equal(t, "", config.OutputFile)
	assert.False(t, config.JSONOutput)
	assert.False(t, config.Verbose)
	assert.False(t, config.ExecEnabled)
	assert.Equal(t, "512m", config.ExecMemoryLimit)
	assert.Equal(t, 2, config.ExecCPULimit)

	// Test default lists are set
	assert.NotEmpty(t, config.AllowedExtensions)
	assert.NotEmpty(t, config.ExecWhitelist)
	assert.NotEmpty(t, config.ExcludedPaths)
}

// TestBooleanFlags tests boolean flag behavior
func TestBooleanFlags(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		expected map[string]bool
	}{
		{
			name: "all boolean flags true",
			args: []string{
				"llm-tool",
				"-interactive",
				"-json",
				"-verbose",
				"-exec-enabled",
			},
			expected: map[string]bool{
				"interactive":  true,
				"json":         true,
				"verbose":      true,
				"exec-enabled": true,
			},
		},
		{
			name: "no boolean flags",
			args: []string{"llm-tool"},
			expected: map[string]bool{
				"interactive":  false,
				"json":         false,
				"verbose":      false,
				"exec-enabled": false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save original command line args and restore after test
			oldArgs := os.Args
			defer func() { os.Args = oldArgs }()

			// Reset flag package state
			flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)

			os.Args = tt.args
			config := setupFlags()

			assert.Equal(t, tt.expected["interactive"], config.Interactive)
			assert.Equal(t, tt.expected["json"], config.JSONOutput)
			assert.Equal(t, tt.expected["verbose"], config.Verbose)
			assert.Equal(t, tt.expected["exec-enabled"], config.ExecEnabled)
		})
	}
}

// TestNumericFlags tests numeric flag parsing
func TestNumericFlags(t *testing.T) {
	tests := []struct {
		name      string
		args      []string
		expectErr bool
		validate  func(t *testing.T, config *core.Config)
	}{
		{
			name: "valid numeric flags",
			args: []string{
				"llm-tool",
				"-max-size", "5242880",
				"-exec-cpu", "8",
			},
			expectErr: false,
			validate: func(t *testing.T, config *core.Config) {
				assert.Equal(t, int64(5242880), config.MaxFileSize)
				assert.Equal(t, 8, config.ExecCPULimit)
			},
		},
		{
			name: "zero values",
			args: []string{
				"llm-tool",
				"-max-size", "0",
				"-exec-cpu", "0",
			},
			expectErr: false,
			validate: func(t *testing.T, config *core.Config) {
				assert.Equal(t, int64(0), config.MaxFileSize)
				assert.Equal(t, 0, config.ExecCPULimit)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save original command line args and restore after test
			oldArgs := os.Args
			defer func() { os.Args = oldArgs }()

			// Reset flag package state
			flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)

			os.Args = tt.args
			config := setupFlags()

			if !tt.expectErr {
				require.NotNil(t, config)
				tt.validate(t, config)
			}
		})
	}
}

// TestDurationParsing tests duration flag parsing
func TestDurationParsing(t *testing.T) {
	tests := []struct {
		name            string
		timeoutValue    string
		expectedTimeout time.Duration
	}{
		{
			name:            "seconds",
			timeoutValue:    "45s",
			expectedTimeout: 45 * time.Second,
		},
		{
			name:            "minutes",
			timeoutValue:    "5m",
			expectedTimeout: 5 * time.Minute,
		},
		{
			name:            "hours",
			timeoutValue:    "2h",
			expectedTimeout: 2 * time.Hour,
		},
		{
			name:            "mixed units",
			timeoutValue:    "1m30s",
			expectedTimeout: 90 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save original command line args and restore after test
			oldArgs := os.Args
			defer func() { os.Args = oldArgs }()

			// Reset flag package state
			flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)

			os.Args = []string{"llm-tool", "-exec-timeout", tt.timeoutValue}
			config := setupFlags()

			assert.Equal(t, tt.expectedTimeout, config.ExecTimeout)
		})
	}
}

// BenchmarkSetupFlags benchmarks flag parsing performance
func BenchmarkSetupFlags(b *testing.B) {
	// Save original command line args and restore after benchmark
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	os.Args = []string{
		"llm-tool",
		"-root", "/test/path",
		"-max-size", "1048576",
		"-interactive",
		"-exec-enabled",
		"-allowed-extensions", ".go,.py,.js,.md,.txt",
		"-exec-whitelist", "go test,go build,npm test",
		"-exclude", ".git,.env,*.key",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Reset flag package state for each iteration
		flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
		_ = setupFlags()
	}
}

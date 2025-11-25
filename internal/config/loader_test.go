package config

import (
	"flag"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/computerscienceiscool/llm-tools/internal/core"
)

// TestNewConfigLoader tests config loader creation
func TestNewConfigLoader(t *testing.T) {
	loader := NewConfigLoader()
	assert.NotNil(t, loader)
	assert.IsType(t, &DefaultConfigLoader{}, loader)
}

// TestDefaultConfigLoaderLoadConfig tests the main LoadConfig functionality
func TestDefaultConfigLoaderLoadConfig(t *testing.T) {
	// Create temporary directory for test
	tempDir, err := os.MkdirTemp("", "config-test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	tests := []struct {
		name       string
		setupArgs  func()
		configPath string
		expectErr  bool
		validate   func(t *testing.T, config *core.Config)
	}{
		{
			name: "load with default flags",
			setupArgs: func() {
				// Reset flag package and set minimal args
				flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
				os.Args = []string{"test", "-root", tempDir}
			},
			configPath: "",
			expectErr:  false,
			validate: func(t *testing.T, config *core.Config) {
				assert.Equal(t, tempDir, config.RepositoryRoot)
				assert.Equal(t, int64(1048576), config.MaxFileSize)
				assert.False(t, config.Interactive)
			},
		},
		{
			name: "load with custom flags",
			setupArgs: func() {
				flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
				os.Args = []string{
					"test",
					"-root", tempDir,
					"-max-size", "2097152",
					"-interactive",
					"-verbose",
					"-exec-enabled",
					"-exec-timeout", "60s",
				}
			},
			configPath: "",
			expectErr:  false,
			validate: func(t *testing.T, config *core.Config) {
				assert.Equal(t, tempDir, config.RepositoryRoot)
				assert.Equal(t, int64(2097152), config.MaxFileSize)
				assert.True(t, config.Interactive)
				assert.True(t, config.Verbose)
				assert.True(t, config.ExecEnabled)
				assert.Equal(t, time.Minute, config.ExecTimeout)
			},
		},
		{
			name: "repository root does not exist",
			setupArgs: func() {
				flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
				os.Args = []string{"test", "-root", "/nonexistent/path"}
			},
			configPath: "",
			expectErr:  true,
			validate:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save original args
			oldArgs := os.Args
			defer func() { os.Args = oldArgs }()

			tt.setupArgs()

			loader := NewConfigLoader()
			config, err := loader.LoadConfig(tt.configPath)

			if tt.expectErr {
				assert.Error(t, err)
				assert.Nil(t, config)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, config)
				if tt.validate != nil {
					tt.validate(t, config)
				}
			}
		})
	}
}

// TestParseFlags tests flag parsing functionality
func TestParseFlags(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "config-test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	tests := []struct {
		name      string
		args      []string
		expectErr bool
		validate  func(t *testing.T, config *core.Config)
	}{
		{
			name: "basic flags",
			args: []string{
				"test",
				"-root", tempDir,
				"-max-size", "1048576",
				"-interactive",
			},
			expectErr: false,
			validate: func(t *testing.T, config *core.Config) {
				assert.Equal(t, tempDir, config.RepositoryRoot)
				assert.Equal(t, int64(1048576), config.MaxFileSize)
				assert.True(t, config.Interactive)
			},
		},
		{
			name: "input/output flags",
			args: []string{
				"test",
				"-root", tempDir,
				"-input", "input.txt",
				"-output", "output.txt",
				"-json",
			},
			expectErr: false,
			validate: func(t *testing.T, config *core.Config) {
				assert.Equal(t, "input.txt", config.InputFile)
				assert.Equal(t, "output.txt", config.OutputFile)
				assert.True(t, config.JSONOutput)
			},
		},
		{
			name: "write operation flags",
			args: []string{
				"test",
				"-root", tempDir,
				"-max-write-size", "51200",
				"-require-confirmation",
				"-backup",
				"-force",
			},
			expectErr: false,
			validate: func(t *testing.T, config *core.Config) {
				assert.Equal(t, int64(51200), config.MaxWriteSize)
				assert.True(t, config.RequireConfirmation)
				assert.True(t, config.BackupBeforeWrite)
				assert.True(t, config.ForceWrite)
			},
		},
		{
			name: "exec flags",
			args: []string{
				"test",
				"-root", tempDir,
				"-exec-enabled",
				"-exec-timeout", "45s",
				"-exec-memory", "1g",
				"-exec-cpu", "4",
				"-exec-image", "alpine:latest",
				"-exec-network",
			},
			expectErr: false,
			validate: func(t *testing.T, config *core.Config) {
				assert.True(t, config.ExecEnabled)
				assert.Equal(t, 45*time.Second, config.ExecTimeout)
				assert.Equal(t, "1g", config.ExecMemoryLimit)
				assert.Equal(t, 4, config.ExecCPULimit)
				assert.Equal(t, "alpine:latest", config.ExecContainerImage)
				assert.True(t, config.ExecNetworkEnabled)
			},
		},
		{
			name: "list flags",
			args: []string{
				"test",
				"-root", tempDir,
				"-allowed-extensions", ".go,.py,.js",
				"-exec-whitelist", "go test,npm test,make",
				"-exclude", ".git,.env,*.key",
			},
			expectErr: false,
			validate: func(t *testing.T, config *core.Config) {
				assert.Equal(t, []string{".go", ".py", ".js"}, config.AllowedExtensions)
				assert.Equal(t, []string{"go test", "npm test", "make"}, config.ExecWhitelist)
				assert.Equal(t, []string{".git", ".env", "*.key"}, config.ExcludedPaths)
			},
		},

		{
			name: "invalid timeout",
			args: []string{
				"test",
				"-root", tempDir,
				"-exec-timeout", "invalid-duration",
			},
			expectErr: true, // Should error on invalid duration
			validate:  nil,
		},
		{
			name: "zero timeout allowed",
			args: []string{
				"test",
				"-root", tempDir,
				"-exec-timeout", "0",
			},
			expectErr: false, // Should not error; zero is valid
			validate: func(t *testing.T, config *core.Config) {
				assert.Equal(t, time.Duration(0), config.ExecTimeout)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save original args
			oldArgs := os.Args
			defer func() { os.Args = oldArgs }()

			// Reset flag package
			flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
			os.Args = tt.args

			loader := &DefaultConfigLoader{}
			config := &core.Config{}
			err := loader.parseFlags(config)

			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				if tt.validate != nil {
					tt.validate(t, config)
				}
			}
		})
	}
}

// TestRepositoryRootResolution tests repository root path resolution
func TestRepositoryRootResolution(t *testing.T) {
	// Create temporary directory structure
	tempDir, err := os.MkdirTemp("", "repo-test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	subDir := filepath.Join(tempDir, "subdir")
	err = os.MkdirAll(subDir, 0755)
	require.NoError(t, err)

	tests := []struct {
		name         string
		root         string
		expectError  bool
		validatePath func(t *testing.T, resolvedPath string)
	}{
		{
			name:        "absolute path",
			root:        tempDir,
			expectError: false,
			validatePath: func(t *testing.T, resolvedPath string) {
				assert.Equal(t, tempDir, resolvedPath)
				assert.True(t, filepath.IsAbs(resolvedPath))
			},
		},
		{
			name:        "relative path",
			root:        ".",
			expectError: false,
			validatePath: func(t *testing.T, resolvedPath string) {
				assert.True(t, filepath.IsAbs(resolvedPath))
			},
		},
		{
			name:         "relative path with subdirectory",
			root:         filepath.Base(tempDir),
			expectError:  true, // Won't exist relative to current dir
			validatePath: nil,
		},
		{
			name:         "nonexistent path",
			root:         "/completely/nonexistent/path",
			expectError:  true,
			validatePath: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save original args
			oldArgs := os.Args
			defer func() { os.Args = oldArgs }()

			// Setup flags
			flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
			os.Args = []string{"test", "-root", tt.root}

			loader := NewConfigLoader()
			config, err := loader.LoadConfig("")

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, config)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, config)
				if tt.validatePath != nil {
					tt.validatePath(t, config.RepositoryRoot)
				}
			}
		})
	}
}

// TestListParsing tests parsing of comma-separated list flags
func TestListParsing(t *testing.T) {
	tests := []struct {
		name            string
		allowedExts     string
		execWhitelist   string
		excludedPaths   string
		expectedExts    []string
		expectedExec    []string
		expectedExclude []string
	}{
		{
			name:            "normal lists",
			allowedExts:     ".go,.py,.js",
			execWhitelist:   "go test,npm test",
			excludedPaths:   ".git,.env",
			expectedExts:    []string{".go", ".py", ".js"},
			expectedExec:    []string{"go test", "npm test"},
			expectedExclude: []string{".git", ".env"},
		},
		{
			name:            "lists with spaces",
			allowedExts:     " .go , .py , .js ",
			execWhitelist:   " go test , npm test ",
			excludedPaths:   " .git , .env ",
			expectedExts:    []string{".go", ".py", ".js"},
			expectedExec:    []string{"go test", "npm test"},
			expectedExclude: []string{".git", ".env"},
		},
		{
			name:            "empty lists",
			allowedExts:     "",
			execWhitelist:   "",
			excludedPaths:   "",
			expectedExts:    []string{""},
			expectedExec:    []string{""},
			expectedExclude: []string{""},
		},
		{
			name:            "single items",
			allowedExts:     ".go",
			execWhitelist:   "go test",
			excludedPaths:   ".git",
			expectedExts:    []string{".go"},
			expectedExec:    []string{"go test"},
			expectedExclude: []string{".git"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp dir for valid root
			tempDir, err := os.MkdirTemp("", "list-test")
			require.NoError(t, err)
			defer os.RemoveAll(tempDir)

			// Save original args
			oldArgs := os.Args
			defer func() { os.Args = oldArgs }()

			// Setup flags
			flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
			os.Args = []string{
				"test",
				"-root", tempDir,
				"-allowed-extensions", tt.allowedExts,
				"-exec-whitelist", tt.execWhitelist,
				"-exclude", tt.excludedPaths,
			}

			loader := NewConfigLoader()
			config, err := loader.LoadConfig("")

			require.NoError(t, err)
			assert.Equal(t, tt.expectedExts, config.AllowedExtensions)
			assert.Equal(t, tt.expectedExec, config.ExecWhitelist)
			assert.Equal(t, tt.expectedExclude, config.ExcludedPaths)
		})
	}
}

// TestDefaultValues tests that appropriate defaults are set
func TestDefaultValues(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "defaults-test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Save original args
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	// Setup with minimal flags
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	os.Args = []string{"test", "-root", tempDir}

	loader := NewConfigLoader()
	config, err := loader.LoadConfig("")

	require.NoError(t, err)

	// Test default values
	assert.Equal(t, int64(1048576), config.MaxFileSize)
	assert.Equal(t, int64(102400), config.MaxWriteSize)
	assert.False(t, config.Interactive)
	assert.False(t, config.JSONOutput)
	assert.False(t, config.Verbose)
	assert.True(t, config.BackupBeforeWrite) // Default true
	assert.False(t, config.RequireConfirmation)
	assert.False(t, config.ForceWrite)
	assert.False(t, config.ExecEnabled)
	assert.Equal(t, "512m", config.ExecMemoryLimit)
	assert.Equal(t, 2, config.ExecCPULimit)
	assert.Equal(t, "ubuntu:22.04", config.ExecContainerImage)
	assert.False(t, config.ExecNetworkEnabled)

	// Test default lists are not empty
	assert.NotEmpty(t, config.AllowedExtensions)
	assert.NotEmpty(t, config.ExecWhitelist)
	assert.NotEmpty(t, config.ExcludedPaths)

	// Test specific default values
	assert.Contains(t, config.AllowedExtensions, ".go")
	assert.Contains(t, config.AllowedExtensions, ".py")
	assert.Contains(t, config.ExecWhitelist, "go test")
	assert.Contains(t, config.ExcludedPaths, ".git")
	assert.Contains(t, config.ExcludedPaths, ".env")
}

// TestFlagAlreadyParsed tests behavior when flags are already parsed
func TestFlagAlreadyParsed(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "parsed-test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Save original args
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	// Simulate flags already being parsed
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	os.Args = []string{"test", "-root", tempDir}

	// Parse flags to set the parsed state
	loader := &DefaultConfigLoader{}
	config := &core.Config{}
	err = loader.parseFlags(config)
	require.NoError(t, err)

	// Now try to load config again - should not parse flags again
	fullLoader := NewConfigLoader()
	finalConfig, err := fullLoader.LoadConfig("")

	assert.NoError(t, err)
	assert.NotNil(t, finalConfig)

	// The repository root should be resolved to absolute path
	cwd, err := os.Getwd()
	require.NoError(t, err)
	assert.Equal(t, cwd, finalConfig.RepositoryRoot)
}

// TestConcurrentConfigLoad tests concurrent config loading
func TestConcurrentConfigLoad(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "concurrent-test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Save original args
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	os.Args = []string{"test", "-root", tempDir}

	loader := NewConfigLoader()

	// Load config concurrently
	results := make(chan error, 10)
	for i := 0; i < 10; i++ {
		go func() {
			_, err := loader.LoadConfig("")
			results <- err
		}()
	}

	// Check all succeeded
	for i := 0; i < 10; i++ {
		err := <-results
		assert.NoError(t, err)
	}
}

// BenchmarkConfigLoad benchmarks config loading performance
func BenchmarkConfigLoaderLoad(b *testing.B) {
	tempDir, err := os.MkdirTemp("", "benchmark-test")
	if err != nil {
		b.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Save original args
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	os.Args = []string{"test", "-root", tempDir}

	loader := NewConfigLoader()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := loader.LoadConfig("")
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkFlagParsing benchmarks just the flag parsing portion
func BenchmarkFlagParsing(b *testing.B) {
	tempDir, err := os.MkdirTemp("", "flag-benchmark")
	if err != nil {
		b.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Save original args
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	os.Args = []string{
		"test",
		"-root", tempDir,
		"-max-size", "1048576",
		"-interactive",
		"-exec-enabled",
		"-allowed-extensions", ".go,.py,.js,.md,.txt",
	}

	loader := &DefaultConfigLoader{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
		config := &core.Config{}
		err := loader.parseFlags(config)
		if err != nil {
			b.Fatal(err)
		}
	}
}

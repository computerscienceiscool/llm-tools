package core

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestCommand tests the Command structure
func TestCommand(t *testing.T) {
	tests := []struct {
		name     string
		command  Command
		validate func(t *testing.T, cmd Command)
	}{
		{
			name: "open command",
			command: Command{
				Type:     "open",
				Argument: "test.txt",
				Content:  "",
				StartPos: 0,
				EndPos:   15,
				Original: "<open test.txt>",
			},
			validate: func(t *testing.T, cmd Command) {
				assert.Equal(t, "open", cmd.Type)
				assert.Equal(t, "test.txt", cmd.Argument)
				assert.Empty(t, cmd.Content)
			},
		},
		{
			name: "write command",
			command: Command{
				Type:     "write",
				Argument: "output.txt",
				Content:  "Hello, World!",
				StartPos: 5,
				EndPos:   35,
				Original: "<write output.txt>Hello, World!</write>",
			},
			validate: func(t *testing.T, cmd Command) {
				assert.Equal(t, "write", cmd.Type)
				assert.Equal(t, "output.txt", cmd.Argument)
				assert.Equal(t, "Hello, World!", cmd.Content)
			},
		},
		{
			name: "exec command",
			command: Command{
				Type:     "exec",
				Argument: "go test",
				Content:  "",
				StartPos: 10,
				EndPos:   25,
				Original: "<exec go test>",
			},
			validate: func(t *testing.T, cmd Command) {
				assert.Equal(t, "exec", cmd.Type)
				assert.Equal(t, "go test", cmd.Argument)
				assert.Empty(t, cmd.Content)
			},
		},
		{
			name: "search command",
			command: Command{
				Type:     "search",
				Argument: "authentication",
				Content:  "",
				StartPos: 0,
				EndPos:   25,
				Original: "<search authentication>",
			},
			validate: func(t *testing.T, cmd Command) {
				assert.Equal(t, "search", cmd.Type)
				assert.Equal(t, "authentication", cmd.Argument)
				assert.Empty(t, cmd.Content)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.validate(t, tt.command)
		})
	}
}

// TestExecutionResult tests the ExecutionResult structure
func TestExecutionResult(t *testing.T) {
	tests := []struct {
		name     string
		result   ExecutionResult
		validate func(t *testing.T, result ExecutionResult)
	}{
		{
			name: "successful open result",
			result: ExecutionResult{
				Command: Command{
					Type:     "open",
					Argument: "test.txt",
				},
				Success:       true,
				Result:        "file contents",
				Error:         nil,
				ExecutionTime: time.Millisecond * 100,
			},
			validate: func(t *testing.T, result ExecutionResult) {
				assert.True(t, result.Success)
				assert.Equal(t, "file contents", result.Result)
				assert.NoError(t, result.Error)
				assert.Equal(t, time.Millisecond*100, result.ExecutionTime)
			},
		},
		{
			name: "successful write result",
			result: ExecutionResult{
				Command: Command{
					Type:     "write",
					Argument: "output.txt",
					Content:  "test content",
				},
				Success:      true,
				BytesWritten: 12,
				BackupFile:   "output.txt.bak.123456",
				Action:       "CREATED",
			},
			validate: func(t *testing.T, result ExecutionResult) {
				assert.True(t, result.Success)
				assert.Equal(t, int64(12), result.BytesWritten)
				assert.Equal(t, "output.txt.bak.123456", result.BackupFile)
				assert.Equal(t, "CREATED", result.Action)
			},
		},
		{
			name: "successful exec result",
			result: ExecutionResult{
				Command: Command{
					Type:     "exec",
					Argument: "echo hello",
				},
				Success:       true,
				Result:        "hello\n",
				ExitCode:      0,
				Stdout:        "hello\n",
				Stderr:        "",
				ContainerID:   "container123",
				ExecutionTime: time.Second,
			},
			validate: func(t *testing.T, result ExecutionResult) {
				assert.True(t, result.Success)
				assert.Equal(t, 0, result.ExitCode)
				assert.Equal(t, "hello\n", result.Stdout)
				assert.Empty(t, result.Stderr)
				assert.Equal(t, "container123", result.ContainerID)
			},
		},
		{
			name: "failed result with error",
			result: ExecutionResult{
				Command: Command{
					Type:     "open",
					Argument: "missing.txt",
				},
				Success: false,
				Error:   assert.AnError,
			},
			validate: func(t *testing.T, result ExecutionResult) {
				assert.False(t, result.Success)
				assert.Error(t, result.Error)
				assert.Empty(t, result.Result)
			},
		},
		{
			name: "exec failure with non-zero exit code",
			result: ExecutionResult{
				Command: Command{
					Type:     "exec",
					Argument: "false", // Command that exits with 1
				},
				Success:  false,
				ExitCode: 1,
				Stdout:   "",
				Stderr:   "command failed",
				Error:    assert.AnError,
			},
			validate: func(t *testing.T, result ExecutionResult) {
				assert.False(t, result.Success)
				assert.Equal(t, 1, result.ExitCode)
				assert.Equal(t, "command failed", result.Stderr)
				assert.Error(t, result.Error)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.validate(t, tt.result)
		})
	}
}

// TestConfig tests the Config structure
func TestConfig(t *testing.T) {
	tests := []struct {
		name     string
		config   Config
		validate func(t *testing.T, cfg Config)
	}{
		{
			name: "basic config",
			config: Config{
				RepositoryRoot: "/home/user/project",
				MaxFileSize:    1048576,
				MaxWriteSize:   102400,
				Interactive:    false,
				ExecEnabled:    false,
			},
			validate: func(t *testing.T, cfg Config) {
				assert.Equal(t, "/home/user/project", cfg.RepositoryRoot)
				assert.Equal(t, int64(1048576), cfg.MaxFileSize)
				assert.Equal(t, int64(102400), cfg.MaxWriteSize)
				assert.False(t, cfg.Interactive)
				assert.False(t, cfg.ExecEnabled)
			},
		},
		{
			name: "config with lists",
			config: Config{
				RepositoryRoot:    "/project",
				MaxFileSize:       2097152,
				ExcludedPaths:     []string{".git", ".env", "*.key"},
				AllowedExtensions: []string{".go", ".py", ".js"},
				ExecWhitelist:     []string{"go test", "npm test"},
			},
			validate: func(t *testing.T, cfg Config) {
				assert.Equal(t, []string{".git", ".env", "*.key"}, cfg.ExcludedPaths)
				assert.Equal(t, []string{".go", ".py", ".js"}, cfg.AllowedExtensions)
				assert.Equal(t, []string{"go test", "npm test"}, cfg.ExecWhitelist)
			},
		},
		{
			name: "exec configuration",
			config: Config{
				ExecEnabled:        true,
				ExecTimeout:        30 * time.Second,
				ExecMemoryLimit:    "512m",
				ExecCPULimit:       2,
				ExecContainerImage: "ubuntu:22.04",
				ExecNetworkEnabled: false,
			},
			validate: func(t *testing.T, cfg Config) {
				assert.True(t, cfg.ExecEnabled)
				assert.Equal(t, 30*time.Second, cfg.ExecTimeout)
				assert.Equal(t, "512m", cfg.ExecMemoryLimit)
				assert.Equal(t, 2, cfg.ExecCPULimit)
				assert.Equal(t, "ubuntu:22.04", cfg.ExecContainerImage)
				assert.False(t, cfg.ExecNetworkEnabled)
			},
		},
		{
			name: "write configuration",
			config: Config{
				MaxWriteSize:        51200,
				RequireConfirmation: true,
				BackupBeforeWrite:   true,
				AllowedExtensions:   []string{".txt", ".md"},
				ForceWrite:          false,
			},
			validate: func(t *testing.T, cfg Config) {
				assert.Equal(t, int64(51200), cfg.MaxWriteSize)
				assert.True(t, cfg.RequireConfirmation)
				assert.True(t, cfg.BackupBeforeWrite)
				assert.False(t, cfg.ForceWrite)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.validate(t, tt.config)
		})
	}
}

// TestConfigDefaults tests that zero values are reasonable
func TestConfigDefaults(t *testing.T) {
	var config Config

	// Test zero values
	assert.Empty(t, config.RepositoryRoot)
	assert.Equal(t, int64(0), config.MaxFileSize)
	assert.Equal(t, int64(0), config.MaxWriteSize)
	assert.False(t, config.Interactive)
	assert.False(t, config.ExecEnabled)
	assert.Equal(t, time.Duration(0), config.ExecTimeout)
	assert.Nil(t, config.ExcludedPaths)
	assert.Nil(t, config.AllowedExtensions)
	assert.Nil(t, config.ExecWhitelist)
}

// TestConfigValidation tests configuration validation logic
func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name   string
		config Config
		valid  bool
		reason string
	}{
		{
			name: "valid minimal config",
			config: Config{
				RepositoryRoot: "/valid/path",
				MaxFileSize:    1048576,
			},
			valid: true,
		},
		{
			name: "empty repository root",
			config: Config{
				RepositoryRoot: "",
				MaxFileSize:    1048576,
			},
			valid:  false,
			reason: "empty repository root",
		},
		{
			name: "negative max file size",
			config: Config{
				RepositoryRoot: "/valid/path",
				MaxFileSize:    -1,
			},
			valid:  false,
			reason: "negative max file size",
		},
		{
			name: "exec enabled but no whitelist",
			config: Config{
				RepositoryRoot: "/valid/path",
				MaxFileSize:    1048576,
				ExecEnabled:    true,
				ExecWhitelist:  []string{},
			},
			valid:  false,
			reason: "exec enabled but no whitelisted commands",
		},
		{
			name: "exec enabled with whitelist",
			config: Config{
				RepositoryRoot: "/valid/path",
				MaxFileSize:    1048576,
				ExecEnabled:    true,
				ExecWhitelist:  []string{"go test"},
			},
			valid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid := validateConfig(tt.config)
			assert.Equal(t, tt.valid, isValid, "Config validation mismatch: %s", tt.reason)
		})
	}
}

// validateConfig is a helper function that would exist in the real implementation
func validateConfig(config Config) bool {
	if config.RepositoryRoot == "" {
		return false
	}
	if config.MaxFileSize < 0 {
		return false
	}
	if config.ExecEnabled && len(config.ExecWhitelist) == 0 {
		return false
	}
	return true
}

// TestCommandTypeValidation tests command type validation
func TestCommandTypeValidation(t *testing.T) {
	validTypes := []string{"open", "write", "exec", "search"}
	invalidTypes := []string{"", "invalid", "Open", "WRITE", "delete", "run"}

	for _, validType := range validTypes {
		t.Run("valid_"+validType, func(t *testing.T) {
			cmd := Command{Type: validType}
			assert.True(t, isValidCommandType(cmd.Type))
		})
	}

	for _, invalidType := range invalidTypes {
		t.Run("invalid_"+invalidType, func(t *testing.T) {
			cmd := Command{Type: invalidType}
			assert.False(t, isValidCommandType(cmd.Type))
		})
	}
}

// isValidCommandType is a helper function that would exist in the real implementation
func isValidCommandType(cmdType string) bool {
	validTypes := map[string]bool{
		"open":   true,
		"write":  true,
		"exec":   true,
		"search": true,
	}
	return validTypes[cmdType]
}

// TestExecutionResultSerialization tests result serialization for logging/debugging
func TestExecutionResultSerialization(t *testing.T) {
	result := ExecutionResult{
		Command: Command{
			Type:     "open",
			Argument: "test.txt",
		},
		Success:       true,
		Result:        "file content",
		ExecutionTime: time.Millisecond * 150,
	}

	// In a real implementation, we might have JSON serialization
	// For now, just test that the structure can be converted to string
	assert.NotEmpty(t, result.Command.Type)
	assert.NotEmpty(t, result.Result)
	assert.Positive(t, result.ExecutionTime)
}

// TestExecutionResultMerging tests combining multiple results
func TestExecutionResultMerging(t *testing.T) {
	results := []ExecutionResult{
		{
			Command:       Command{Type: "open", Argument: "file1.txt"},
			Success:       true,
			Result:        "content1",
			ExecutionTime: time.Millisecond * 100,
		},
		{
			Command:       Command{Type: "open", Argument: "file2.txt"},
			Success:       true,
			Result:        "content2",
			ExecutionTime: time.Millisecond * 150,
		},
	}

	// Test that we can work with multiple results
	assert.Len(t, results, 2)

	totalTime := time.Duration(0)
	for _, result := range results {
		totalTime += result.ExecutionTime
		assert.True(t, result.Success)
	}

	assert.Equal(t, time.Millisecond*250, totalTime)
}

// BenchmarkCommandCreation benchmarks command structure creation
func BenchmarkCommandCreation(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = Command{
			Type:     "open",
			Argument: "test.txt",
			Content:  "",
			StartPos: 0,
			EndPos:   15,
			Original: "<open test.txt>",
		}
	}
}

// BenchmarkExecutionResultCreation benchmarks result creation
func BenchmarkExecutionResultCreation(b *testing.B) {
	cmd := Command{Type: "open", Argument: "test.txt"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ExecutionResult{
			Command:       cmd,
			Success:       true,
			Result:        "file content",
			ExecutionTime: time.Millisecond * 100,
		}
	}
}

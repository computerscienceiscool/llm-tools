package handlers

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockDockerClient for testing exec handler
type MockDockerClient struct {
	mock.Mock
}

func (m *MockDockerClient) RunCommand(ctx context.Context, config ExecConfig, command string) (ExecResult, error) {
	args := m.Called(ctx, config, command)
	return args.Get(0).(ExecResult), args.Error(1)
}

func (m *MockDockerClient) IsAvailable(ctx context.Context) bool {
	args := m.Called(ctx)
	return args.Bool(0)
}

// MockCommandValidator for testing
type MockCommandValidator struct {
	mock.Mock
}

func (m *MockCommandValidator) ValidateCommand(command string, whitelist []string) error {
	args := m.Called(command, whitelist)
	return args.Error(0)
}

// TestDefaultExecHandler tests the default exec handler implementation
func TestDefaultExecHandler(t *testing.T) {
	mockDocker := &MockDockerClient{}
	mockValidator := &MockCommandValidator{}

	handler := NewExecHandler(mockDocker, mockValidator)
	require.NotNil(t, handler)

	ctx := context.Background()
	config := ExecConfig{
		Enabled:        true,
		Whitelist:      []string{"go", "npm", "python"},
		Timeout:        30 * time.Second,
		MemoryLimit:    "512m",
		CPULimit:       2,
		ContainerImage: "ubuntu:22.04",
		RepoRoot:       "/repo",
	}

	t.Run("execute whitelisted command successfully", func(t *testing.T) {
		command := "go test ./..."
		expectedResult := ExecResult{
			ExitCode: 0,
			Stdout:   "ok  \ttest\t0.123s",
			Stderr:   "",
			Duration: 2 * time.Second,
		}

		mockValidator.On("ValidateCommand", command, config.Whitelist).Return(nil)
		mockDocker.On("IsAvailable", ctx).Return(true)
		mockDocker.On("RunCommand", ctx, config, command).Return(expectedResult, nil)

		result, err := handler.ExecuteCommand(ctx, command, config)
		assert.NoError(t, err)
		assert.Equal(t, 0, result.ExitCode)
		assert.Contains(t, result.Stdout, "ok")
		assert.Equal(t, 2*time.Second, result.Duration)

		mockValidator.AssertExpectations(t)
		mockDocker.AssertExpectations(t)
	})

	t.Run("reject non-whitelisted command", func(t *testing.T) {
		command := "rm -rf /"

		mockValidator.On("ValidateCommand", command, config.Whitelist).
			Return(fmt.Errorf("EXEC_VALIDATION: command not whitelisted"))

		_, err := handler.ExecuteCommand(ctx, command, config)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "EXEC_VALIDATION")

		mockValidator.AssertExpectations(t)
	})

	t.Run("docker unavailable", func(t *testing.T) {
		mockDocker = &MockDockerClient{}
		mockValidator = &MockCommandValidator{}
		handler = NewExecHandler(mockDocker, mockValidator)

		command := "go version"

		mockValidator.On("ValidateCommand", command, config.Whitelist).Return(nil)
		mockDocker.On("IsAvailable", ctx).Return(false)

		_, err := handler.ExecuteCommand(ctx, command, config)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Docker")

		mockValidator.AssertExpectations(t)
		mockDocker.AssertExpectations(t)
	})

	t.Run("execution disabled", func(t *testing.T) {
		disabledConfig := config
		disabledConfig.Enabled = false

		_, err := handler.ExecuteCommand(ctx, "go test", disabledConfig)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "disabled")
	})
}

// TestExecHandlerErrors tests error handling scenarios
func TestExecHandlerErrors(t *testing.T) {
	mockDocker := &MockDockerClient{}
	mockValidator := &MockCommandValidator{}
	handler := NewExecHandler(mockDocker, mockValidator)

	ctx := context.Background()
	config := ExecConfig{
		Enabled:   true,
		Whitelist: []string{"go"},
		Timeout:   10 * time.Second,
		RepoRoot:  "/repo",
	}

	t.Run("command execution fails", func(t *testing.T) {
		command := "go test ./nonexistent"
		failedResult := ExecResult{
			ExitCode: 1,
			Stdout:   "",
			Stderr:   "no packages to test",
			Duration: time.Second,
		}

		mockValidator.On("ValidateCommand", command, config.Whitelist).Return(nil)
		mockDocker.On("IsAvailable", ctx).Return(true)
		mockDocker.On("RunCommand", ctx, config, command).Return(failedResult, nil)

		result, err := handler.ExecuteCommand(ctx, command, config)
		assert.NoError(t, err) // Handler doesn't error, but command failed
		assert.Equal(t, 1, result.ExitCode)
		assert.Contains(t, result.Stderr, "no packages")

		mockValidator.AssertExpectations(t)
		mockDocker.AssertExpectations(t)
	})

	t.Run("docker execution error", func(t *testing.T) {
		mockDocker = &MockDockerClient{}
		mockValidator = &MockCommandValidator{}
		handler = NewExecHandler(mockDocker, mockValidator)

		command := "go build"

		mockValidator.On("ValidateCommand", command, config.Whitelist).Return(nil)
		mockDocker.On("IsAvailable", ctx).Return(true)
		mockDocker.On("RunCommand", ctx, config, command).
			Return(ExecResult{}, fmt.Errorf("docker: container failed to start"))

		_, err := handler.ExecuteCommand(ctx, command, config)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "docker")

		mockValidator.AssertExpectations(t)
		mockDocker.AssertExpectations(t)
	})

	t.Run("timeout handling", func(t *testing.T) {
		mockDocker = &MockDockerClient{}
		mockValidator = &MockCommandValidator{}
		handler = NewExecHandler(mockDocker, mockValidator)

		command := "sleep 30"
		shortConfig := config
		shortConfig.Timeout = 100 * time.Millisecond

		mockValidator.On("ValidateCommand", command, shortConfig.Whitelist).Return(nil)
		mockDocker.On("IsAvailable", ctx).Return(true)
		mockDocker.On("RunCommand", mock.Anything, shortConfig, command).
			Return(ExecResult{ExitCode: 124}, fmt.Errorf("execution timed out"))

		_, err := handler.ExecuteCommand(ctx, command, shortConfig)
		assert.Error(t, err)
		assert.Contains(t, strings.ToLower(err.Error()), "timeout")

		mockValidator.AssertExpectations(t)
		mockDocker.AssertExpectations(t)
	})
}

// TestExecHandlerValidation tests command validation
func TestExecHandlerValidation(t *testing.T) {
	mockDocker := &MockDockerClient{}
	mockValidator := &MockCommandValidator{}
	handler := NewExecHandler(mockDocker, mockValidator)

	ctx := context.Background()

	tests := []struct {
		name      string
		command   string
		whitelist []string
		shouldErr bool
		errorMsg  string
	}{
		{
			name:      "allowed go command",
			command:   "go test",
			whitelist: []string{"go", "npm"},
			shouldErr: false,
		},
		{
			name:      "allowed npm command",
			command:   "npm test",
			whitelist: []string{"go", "npm"},
			shouldErr: false,
		},
		{
			name:      "disallowed rm command",
			command:   "rm -rf /",
			whitelist: []string{"go", "npm"},
			shouldErr: true,
			errorMsg:  "not whitelisted",
		},
		{
			name:      "command injection attempt",
			command:   "go test; rm -rf /",
			whitelist: []string{"go"},
			shouldErr: true,
			errorMsg:  "injection",
		},
		{
			name:      "empty command",
			command:   "",
			whitelist: []string{"go"},
			shouldErr: true,
			errorMsg:  "empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := ExecConfig{
				Enabled:   true,
				Whitelist: tt.whitelist,
				Timeout:   30 * time.Second,
				RepoRoot:  "/repo",
			}

			if tt.shouldErr {
				mockValidator.On("ValidateCommand", tt.command, tt.whitelist).
					Return(fmt.Errorf("EXEC_VALIDATION: %s", tt.errorMsg))

				_, err := handler.ExecuteCommand(ctx, tt.command, config)
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "EXEC_VALIDATION")
			} else {
				expectedResult := ExecResult{ExitCode: 0, Stdout: "success"}

				mockValidator.On("ValidateCommand", tt.command, tt.whitelist).Return(nil)
				mockDocker.On("IsAvailable", ctx).Return(true)
				mockDocker.On("RunCommand", ctx, config, tt.command).Return(expectedResult, nil)

				result, err := handler.ExecuteCommand(ctx, tt.command, config)
				assert.NoError(t, err)
				assert.Equal(t, 0, result.ExitCode)
			}

			mockValidator.AssertExpectations(t)
			mockDocker.AssertExpectations(t)

			// Reset mocks for next test
			mockValidator = &MockCommandValidator{}
			mockDocker = &MockDockerClient{}
			handler = NewExecHandler(mockDocker, mockValidator)
		})
	}
}

// TestExecConfig tests the configuration structure
func TestExecConfig(t *testing.T) {
	config := ExecConfig{
		Enabled:        true,
		Whitelist:      []string{"go", "npm", "python3"},
		Timeout:        45 * time.Second,
		MemoryLimit:    "1g",
		CPULimit:       4,
		ContainerImage: "golang:1.21",
		RepoRoot:       "/workspace",
		EnvVars: map[string]string{
			"GO_ENV": "test",
			"DEBUG":  "true",
		},
	}

	assert.True(t, config.Enabled)
	assert.Contains(t, config.Whitelist, "go")
	assert.Equal(t, 45*time.Second, config.Timeout)
	assert.Equal(t, "1g", config.MemoryLimit)
	assert.Equal(t, 4, config.CPULimit)
	assert.Equal(t, "golang:1.21", config.ContainerImage)
	assert.Equal(t, "/workspace", config.RepoRoot)
	assert.Equal(t, "test", config.EnvVars["GO_ENV"])
}

// TestExecResult tests the result structure
func TestExecResult(t *testing.T) {
	result := ExecResult{
		ExitCode:    2,
		Stdout:      "Build successful",
		Stderr:      "Warning: deprecated function",
		Duration:    5 * time.Second,
		ContainerID: "abc123def456",
		StartTime:   time.Now(),
	}

	assert.Equal(t, 2, result.ExitCode)
	assert.Equal(t, "Build successful", result.Stdout)
	assert.Equal(t, "Warning: deprecated function", result.Stderr)
	assert.Equal(t, 5*time.Second, result.Duration)
	assert.Equal(t, "abc123def456", result.ContainerID)
	assert.False(t, result.StartTime.IsZero())

	// Test success determination
	assert.False(t, result.IsSuccess())

	successResult := ExecResult{ExitCode: 0}
	assert.True(t, successResult.IsSuccess())
}

// TestExecHandlerConcurrency tests concurrent command execution
func TestExecHandlerConcurrency(t *testing.T) {
	mockDocker := &MockDockerClient{}
	mockValidator := &MockCommandValidator{}
	handler := NewExecHandler(mockDocker, mockValidator)

	ctx := context.Background()
	config := ExecConfig{
		Enabled:   true,
		Whitelist: []string{"echo"},
		Timeout:   10 * time.Second,
		RepoRoot:  "/repo",
	}

	const numGoroutines = 5
	done := make(chan bool, numGoroutines)
	results := make([]ExecResult, numGoroutines)
	errors := make([]error, numGoroutines)

	// Setup mock expectations for concurrent calls
	mockValidator.On("ValidateCommand", mock.AnythingOfType("string"), config.Whitelist).Return(nil)
	mockDocker.On("IsAvailable", ctx).Return(true)
	mockDocker.On("RunCommand", ctx, config, mock.AnythingOfType("string")).
		Return(ExecResult{ExitCode: 0, Stdout: "hello"}, nil)

	// Run concurrent executions
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer func() { done <- true }()

			command := fmt.Sprintf("echo 'test %d'", id)
			result, err := handler.ExecuteCommand(ctx, command, config)

			results[id] = result
			errors[id] = err
		}(i)
	}

	// Wait for all to complete
	for i := 0; i < numGoroutines; i++ {
		<-done
	}

	// Verify all succeeded
	for i := 0; i < numGoroutines; i++ {
		assert.NoError(t, errors[i], "Goroutine %d failed", i)
		assert.Equal(t, 0, results[i].ExitCode, "Goroutine %d had non-zero exit", i)
	}

	mockValidator.AssertExpectations(t)
	mockDocker.AssertExpectations(t)
}

// BenchmarkExecHandler benchmarks exec handler performance
func BenchmarkExecHandler(b *testing.B) {
	mockDocker := &MockDockerClient{}
	mockValidator := &MockCommandValidator{}
	handler := NewExecHandler(mockDocker, mockValidator)

	ctx := context.Background()
	config := ExecConfig{
		Enabled:   true,
		Whitelist: []string{"echo"},
		Timeout:   10 * time.Second,
		RepoRoot:  "/repo",
	}

	mockValidator.On("ValidateCommand", "echo test", config.Whitelist).Return(nil)
	mockDocker.On("IsAvailable", ctx).Return(true)
	mockDocker.On("RunCommand", ctx, config, "echo test").
		Return(ExecResult{ExitCode: 0}, nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := handler.ExecuteCommand(ctx, "echo test", config)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Placeholder implementations and interfaces
type ExecHandler interface {
	ExecuteCommand(ctx context.Context, command string, config ExecConfig) (ExecResult, error)
}

type ExecConfig struct {
	Enabled        bool
	Whitelist      []string
	Timeout        time.Duration
	MemoryLimit    string
	CPULimit       int
	ContainerImage string
	RepoRoot       string
	EnvVars        map[string]string
}

type ExecResult struct {
	ExitCode    int
	Stdout      string
	Stderr      string
	Duration    time.Duration
	ContainerID string
	StartTime   time.Time
}

func (r ExecResult) IsSuccess() bool {
	return r.ExitCode == 0
}

type DockerClient interface {
	RunCommand(ctx context.Context, config ExecConfig, command string) (ExecResult, error)
	IsAvailable(ctx context.Context) bool
}

type CommandValidator interface {
	ValidateCommand(command string, whitelist []string) error
}

// Mock implementation
func NewExecHandler(docker DockerClient, validator CommandValidator) ExecHandler {
	return &defaultExecHandler{
		docker:    docker,
		validator: validator,
	}
}

type defaultExecHandler struct {
	docker    DockerClient
	validator CommandValidator
}

func (h *defaultExecHandler) ExecuteCommand(ctx context.Context, command string, config ExecConfig) (ExecResult, error) {
	if !config.Enabled {
		return ExecResult{}, fmt.Errorf("command execution is disabled")
	}

	// Validate command
	err := h.validator.ValidateCommand(command, config.Whitelist)
	if err != nil {
		return ExecResult{}, err
	}

	// Check Docker availability
	if !h.docker.IsAvailable(ctx) {
		return ExecResult{}, fmt.Errorf("Docker is not available")
	}

	// Execute command
	result, err := h.docker.RunCommand(ctx, config, command)
	if err != nil {
		return ExecResult{}, fmt.Errorf("execution failed: %w", err)
	}

	return result, nil
}

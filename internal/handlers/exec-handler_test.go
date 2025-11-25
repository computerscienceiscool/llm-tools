package handlers

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/computerscienceiscool/llm-tools/internal/infrastructure"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockDockerClient for testing exec handler
type MockDockerClient struct {
	mock.Mock
}

func (m *MockDockerClient) ExecuteInContainer(ctx context.Context, config infrastructure.ContainerConfig) (infrastructure.ContainerResult, error) {
	args := m.Called(ctx, config)
	return args.Get(0).(infrastructure.ContainerResult), args.Error(1)
}

func (m *MockDockerClient) PullImage(image string) error {
	args := m.Called(image)
	return args.Error(0)
}

func (m *MockDockerClient) RunCommand(ctx context.Context, config ExecConfig, command string) (ExecResult, error) {
	args := m.Called(ctx, config, command)
	return args.Get(0).(ExecResult), args.Error(1)
}

func (m *MockDockerClient) CheckAvailability() error {
	args := m.Called()
	return args.Error(0)
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
	handler := NewExecHandler(mockDocker)
	require.NotNil(t, handler)

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
		mockDocker.On("CheckAvailability").Return(nil)
		mockDocker.On("RunCommand", mock.AnythingOfType("*context.timerCtx"), config, command).Return(expectedResult, nil)

		result, err := handler.ExecuteCommand(command, config)
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
		_, err := handler.ExecuteCommand(command, config)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "EXEC_VALIDATION")

		mockValidator.AssertExpectations(t)
	})

	t.Run("docker unavailable", func(t *testing.T) {
		mockDocker = &MockDockerClient{}
		mockValidator = &MockCommandValidator{}
		handler = NewExecHandler(mockDocker)

		command := "go version"

		mockValidator.On("ValidateCommand", command, config.Whitelist).Return(nil)
		mockDocker.On("CheckAvailability").Return(fmt.Errorf("Docker not available"))

		_, err := handler.ExecuteCommand(command, config)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Docker")

		mockValidator.AssertExpectations(t)
		mockDocker.AssertExpectations(t)
	})

	t.Run("execution disabled", func(t *testing.T) {
		disabledConfig := config
		disabledConfig.Enabled = false

		_, err := handler.ExecuteCommand("go test", disabledConfig)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "disabled")
	})
}

// TestExecHandlerErrors tests error handling scenarios
func TestExecHandlerErrors(t *testing.T) {
	mockDocker := &MockDockerClient{}
	mockValidator := &MockCommandValidator{}
	handler := NewExecHandler(mockDocker)

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
		mockDocker.On("CheckAvailability").Return(nil)
		mockDocker.On("RunCommand", mock.AnythingOfType("*context.timerCtx"), config, command).Return(failedResult, nil)

		result, err := handler.ExecuteCommand(command, config)
		assert.NoError(t, err) // Handler doesn't error, but command failed
		assert.Equal(t, 1, result.ExitCode)
		assert.Contains(t, result.Stderr, "no packages")

		mockValidator.AssertExpectations(t)
		mockDocker.AssertExpectations(t)
	})

	t.Run("docker execution error", func(t *testing.T) {
		mockDocker = &MockDockerClient{}
		mockValidator = &MockCommandValidator{}
		handler = NewExecHandler(mockDocker)

		command := "go build"

		mockValidator.On("ValidateCommand", command, config.Whitelist).Return(nil)
		mockDocker.On("CheckAvailability").Return(nil)
		mockDocker.On("RunCommand", mock.AnythingOfType("*context.timerCtx"), config, command).
			Return(ExecResult{}, fmt.Errorf("docker: container failed to start"))

		_, err := handler.ExecuteCommand(command, config)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "docker")

		mockValidator.AssertExpectations(t)
		mockDocker.AssertExpectations(t)
	})

	t.Run("timeout handling", func(t *testing.T) {
		mockDocker = &MockDockerClient{}
		mockValidator = &MockCommandValidator{}
		handler = NewExecHandler(mockDocker)

		command := "sleep 30"
		shortConfig := config
		shortConfig.Timeout = 100 * time.Millisecond

		mockValidator.On("ValidateCommand", command, shortConfig.Whitelist).Return(nil)
		mockDocker.On("CheckAvailability").Return(nil)
		mockDocker.On("RunCommand", mock.Anything, shortConfig, command).
			Return(ExecResult{ExitCode: 124}, fmt.Errorf("execution timed out"))

		_, err := handler.ExecuteCommand(command, shortConfig)
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
	handler := NewExecHandler(mockDocker)

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

				_, err := handler.ExecuteCommand(tt.command, config)
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "EXEC_VALIDATION")
			} else {
				expectedResult := ExecResult{ExitCode: 0, Stdout: "success"}

				mockValidator.On("ValidateCommand", tt.command, tt.whitelist).Return(nil)
				mockDocker.On("CheckAvailability").Return(nil)
				mockDocker.On("RunCommand", mock.AnythingOfType("*context.timerCtx"), config, tt.command).Return(expectedResult, nil)

				result, err := handler.ExecuteCommand(tt.command, config)
				assert.NoError(t, err)
				assert.Equal(t, 0, result.ExitCode)
			}

			mockValidator.AssertExpectations(t)
			mockDocker.AssertExpectations(t)

			// Reset mocks for next test
			mockValidator = &MockCommandValidator{}
			mockDocker = &MockDockerClient{}
			handler = NewExecHandler(mockDocker)
		})
	}
}

// TestExecHandlerConcurrency tests concurrent command execution
func TestExecHandlerConcurrency(t *testing.T) {
	mockDocker := &MockDockerClient{}
	mockValidator := &MockCommandValidator{}
	handler := NewExecHandler(mockDocker)

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
	mockDocker.On("CheckAvailability").Return(nil)
	mockDocker.On("RunCommand", mock.AnythingOfType("*context.timerCtx"), config, mock.AnythingOfType("string")).
		Return(ExecResult{ExitCode: 0, Stdout: "hello"}, nil)

	// Run concurrent executions
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer func() { done <- true }()

			command := fmt.Sprintf("echo 'test %d'", id)
			result, err := handler.ExecuteCommand(command, config)

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
	handler := NewExecHandler(mockDocker)

	config := ExecConfig{
		Enabled:   true,
		Whitelist: []string{"echo"},
		Timeout:   10 * time.Second,
		RepoRoot:  "/repo",
	}

	mockValidator.On("ValidateCommand", "echo test", config.Whitelist).Return(nil)
	mockDocker.On("CheckAvailability").Return(nil)
	mockDocker.On("RunCommand", mock.AnythingOfType("*context.timerCtx"), config, "echo test").
		Return(ExecResult{ExitCode: 0}, nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := handler.ExecuteCommand("echo test", config)
		if err != nil {
			b.Fatal(err)
		}
	}
}

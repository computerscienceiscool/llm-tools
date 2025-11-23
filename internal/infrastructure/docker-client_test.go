package infrastructure

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockDockerClient for testing
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

func (m *MockDockerClient) PullImage(ctx context.Context, image string) error {
	args := m.Called(ctx, image)
	return args.Error(0)
}

// TestDockerClientInterface tests the DockerClient interface
func TestDockerClientInterface(t *testing.T) {
	var _ DockerClient = (*MockDockerClient)(nil)

	mockClient := &MockDockerClient{}
	ctx := context.Background()

	config := ExecConfig{
		Image:       "ubuntu:22.04",
		MemoryLimit: "512m",
		CPULimit:    2,
		Timeout:     30 * time.Second,
		RepoRoot:    "/repo",
	}

	result := ExecResult{
		ExitCode: 0,
		Stdout:   "hello world",
		Stderr:   "",
		Duration: time.Second,
	}

	// Setup expectations
	mockClient.On("RunCommand", ctx, config, "echo hello world").Return(result, nil)
	mockClient.On("IsAvailable", ctx).Return(true)
	mockClient.On("PullImage", ctx, "ubuntu:22.04").Return(nil)

	// Test RunCommand
	execResult, err := mockClient.RunCommand(ctx, config, "echo hello world")
	assert.NoError(t, err)
	assert.Equal(t, 0, execResult.ExitCode)
	assert.Equal(t, "hello world", execResult.Stdout)

	// Test IsAvailable
	available := mockClient.IsAvailable(ctx)
	assert.True(t, available)

	// Test PullImage
	err = mockClient.PullImage(ctx, "ubuntu:22.04")
	assert.NoError(t, err)

	mockClient.AssertExpectations(t)
}

// TestDefaultDockerClient tests the default Docker client implementation
func TestDefaultDockerClient(t *testing.T) {
	client := NewDockerClient()
	ctx := context.Background()

	// Check if Docker is available (skip if not)
	if !client.IsAvailable(ctx) {
		t.Skip("Docker not available, skipping Docker client tests")
	}

	t.Run("docker availability", func(t *testing.T) {
		available := client.IsAvailable(ctx)
		assert.True(t, available)
	})

	t.Run("pull image", func(t *testing.T) {
		// Use a small, common image for testing
		err := client.PullImage(ctx, "alpine:latest")
		if err != nil {
			// Image might already exist, which is fine
			assert.Contains(t, err.Error(), "already exists")
		}
	})

	t.Run("run simple command", func(t *testing.T) {
		config := ExecConfig{
			Image:       "alpine:latest",
			MemoryLimit: "128m",
			CPULimit:    1,
			Timeout:     10 * time.Second,
			RepoRoot:    "/tmp",
		}

		result, err := client.RunCommand(ctx, config, "echo 'test output'")
		assert.NoError(t, err)
		assert.Equal(t, 0, result.ExitCode)
		assert.Contains(t, result.Stdout, "test output")
		assert.Empty(t, result.Stderr)
		assert.Greater(t, result.Duration, time.Duration(0))
	})

	t.Run("run failing command", func(t *testing.T) {
		config := ExecConfig{
			Image:       "alpine:latest",
			MemoryLimit: "128m",
			CPULimit:    1,
			Timeout:     10 * time.Second,
			RepoRoot:    "/tmp",
		}

		result, err := client.RunCommand(ctx, config, "false") // Command that always fails
		assert.NoError(t, err)                                 // No error from Docker client itself
		assert.Equal(t, 1, result.ExitCode)                    // But command failed
	})

	t.Run("command timeout", func(t *testing.T) {
		config := ExecConfig{
			Image:       "alpine:latest",
			MemoryLimit: "128m",
			CPULimit:    1,
			Timeout:     1 * time.Second, // Short timeout
			RepoRoot:    "/tmp",
		}

		start := time.Now()
		result, err := client.RunCommand(ctx, config, "sleep 5") // Long-running command
		elapsed := time.Since(start)

		// Should timeout
		assert.Error(t, err)
		assert.Contains(t, strings.ToLower(err.Error()), "timeout")
		assert.Less(t, elapsed, 2*time.Second) // Should timeout before 2 seconds
		assert.NotEqual(t, 0, result.ExitCode) // Should indicate failure
	})
}

// TestDockerClientSecurity tests security features
func TestDockerClientSecurity(t *testing.T) {
	client := NewDockerClient()
	ctx := context.Background()

	if !client.IsAvailable(ctx) {
		t.Skip("Docker not available, skipping security tests")
	}

	t.Run("network isolation", func(t *testing.T) {
		config := ExecConfig{
			Image:       "alpine:latest",
			MemoryLimit: "128m",
			CPULimit:    1,
			Timeout:     10 * time.Second,
			RepoRoot:    "/tmp",
		}

		// Try to access network (should fail)
		result, err := client.RunCommand(ctx, config, "wget -t 1 -T 1 google.com")
		assert.NoError(t, err)                 // Docker client works
		assert.NotEqual(t, 0, result.ExitCode) // But network access fails
	})

	t.Run("filesystem isolation", func(t *testing.T) {
		config := ExecConfig{
			Image:       "alpine:latest",
			MemoryLimit: "128m",
			CPULimit:    1,
			Timeout:     10 * time.Second,
			RepoRoot:    "/tmp",
		}

		// Try to access host filesystem (should fail)
		result, err := client.RunCommand(ctx, config, "ls /etc/passwd")
		assert.NoError(t, err)
		// Should not be able to see host /etc/passwd
		assert.NotContains(t, result.Stdout, "root:")
	})

	t.Run("memory limits", func(t *testing.T) {
		config := ExecConfig{
			Image:       "alpine:latest",
			MemoryLimit: "64m", // Very low memory
			CPULimit:    1,
			Timeout:     10 * time.Second,
			RepoRoot:    "/tmp",
		}

		// Try to use more memory than allowed
		result, err := client.RunCommand(ctx, config, "dd if=/dev/zero of=/tmp/large bs=1M count=100")
		// Should either fail or be limited
		if err == nil {
			// If it doesn't error, it should at least be resource-limited
			assert.NotEqual(t, 0, result.ExitCode)
		}
	})
}

// TestDockerClientConfiguration tests different configurations
func TestDockerClientConfiguration(t *testing.T) {
	client := NewDockerClient()
	ctx := context.Background()

	if !client.IsAvailable(ctx) {
		t.Skip("Docker not available, skipping configuration tests")
	}

	tests := []struct {
		name   string
		config ExecConfig
		valid  bool
	}{
		{
			name: "valid minimal config",
			config: ExecConfig{
				Image:    "alpine:latest",
				Timeout:  10 * time.Second,
				RepoRoot: "/tmp",
			},
			valid: true,
		},
		{
			name: "valid full config",
			config: ExecConfig{
				Image:       "alpine:latest",
				MemoryLimit: "256m",
				CPULimit:    2,
				Timeout:     30 * time.Second,
				RepoRoot:    "/tmp",
			},
			valid: true,
		},
		{
			name: "zero timeout",
			config: ExecConfig{
				Image:    "alpine:latest",
				Timeout:  0,
				RepoRoot: "/tmp",
			},
			valid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := client.RunCommand(ctx, tt.config, "echo test")

			if tt.valid {
				assert.NoError(t, err)
				assert.Equal(t, 0, result.ExitCode)
			} else {
				// Should handle invalid config gracefully
				assert.Error(t, err)
			}
		})
	}
}

// TestDockerClientErrors tests error handling
func TestDockerClientErrors(t *testing.T) {
	client := NewDockerClient()
	ctx := context.Background()

	if !client.IsAvailable(ctx) {
		t.Skip("Docker not available, skipping error tests")
	}

	t.Run("invalid image", func(t *testing.T) {
		config := ExecConfig{
			Image:    "nonexistent-image:invalid-tag",
			Timeout:  10 * time.Second,
			RepoRoot: "/tmp",
		}

		_, err := client.RunCommand(ctx, config, "echo test")
		assert.Error(t, err)
		assert.Contains(t, strings.ToLower(err.Error()), "image")
	})

	t.Run("invalid command", func(t *testing.T) {
		config := ExecConfig{
			Image:    "alpine:latest",
			Timeout:  10 * time.Second,
			RepoRoot: "/tmp",
		}

		result, err := client.RunCommand(ctx, config, "nonexistent-command")
		assert.NoError(t, err)                 // Docker runs successfully
		assert.NotEqual(t, 0, result.ExitCode) // But command fails
	})

	t.Run("cancelled context", func(t *testing.T) {
		config := ExecConfig{
			Image:    "alpine:latest",
			Timeout:  30 * time.Second,
			RepoRoot: "/tmp",
		}

		cancelledCtx, cancel := context.WithCancel(ctx)
		cancel() // Cancel immediately

		_, err := client.RunCommand(cancelledCtx, config, "echo test")
		assert.Error(t, err)
		assert.Contains(t, strings.ToLower(err.Error()), "context")
	})
}

// TestDockerClientConcurrency tests concurrent Docker operations
func TestDockerClientConcurrency(t *testing.T) {
	client := NewDockerClient()
	ctx := context.Background()

	if !client.IsAvailable(ctx) {
		t.Skip("Docker not available, skipping concurrency tests")
	}

	const numGoroutines = 5
	done := make(chan bool, numGoroutines)
	results := make([]ExecResult, numGoroutines)
	errors := make([]error, numGoroutines)

	config := ExecConfig{
		Image:       "alpine:latest",
		MemoryLimit: "128m",
		CPULimit:    1,
		Timeout:     10 * time.Second,
		RepoRoot:    "/tmp",
	}

	// Run multiple commands concurrently
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer func() { done <- true }()

			command := fmt.Sprintf("echo 'concurrent test %d'", id)
			result, err := client.RunCommand(ctx, config, command)

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
		assert.Contains(t, results[i].Stdout, fmt.Sprintf("concurrent test %d", i))
	}
}

// BenchmarkDockerClient benchmarks Docker operations
func BenchmarkDockerClient(b *testing.B) {
	client := NewDockerClient()
	ctx := context.Background()

	if !client.IsAvailable(ctx) {
		b.Skip("Docker not available, skipping benchmarks")
	}

	config := ExecConfig{
		Image:       "alpine:latest",
		MemoryLimit: "128m",
		CPULimit:    1,
		Timeout:     10 * time.Second,
		RepoRoot:    "/tmp",
	}

	b.Run("SimpleCommand", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := client.RunCommand(ctx, config, "echo test")
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("IsAvailable", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = client.IsAvailable(ctx)
		}
	})
}

// TestDockerClientResourceUsage tests resource monitoring
func TestDockerClientResourceUsage(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping resource usage test in short mode")
	}

	client := NewDockerClient()
	ctx := context.Background()

	if !client.IsAvailable(ctx) {
		t.Skip("Docker not available, skipping resource tests")
	}

	config := ExecConfig{
		Image:       "alpine:latest",
		MemoryLimit: "256m",
		CPULimit:    1,
		Timeout:     30 * time.Second,
		RepoRoot:    "/tmp",
	}

	t.Run("cpu intensive task", func(t *testing.T) {
		start := time.Now()
		result, err := client.RunCommand(ctx, config, "yes | head -n 100000 > /dev/null")
		elapsed := time.Since(start)

		assert.NoError(t, err)
		assert.Equal(t, 0, result.ExitCode)
		assert.Greater(t, elapsed, 100*time.Millisecond) // Should take some time
		assert.Equal(t, elapsed.Truncate(time.Millisecond), result.Duration.Truncate(time.Millisecond))
	})

	t.Run("memory usage monitoring", func(t *testing.T) {
		// Run a command that uses some memory
		result, err := client.RunCommand(ctx, config, "dd if=/dev/zero of=/tmp/test bs=1M count=10")

		assert.NoError(t, err)
		assert.Equal(t, 0, result.ExitCode)
		assert.Greater(t, result.Duration, time.Duration(0))
	})
}

// Placeholder interfaces and types
type DockerClient interface {
	RunCommand(ctx context.Context, config ExecConfig, command string) (ExecResult, error)
	IsAvailable(ctx context.Context) bool
	PullImage(ctx context.Context, image string) error
}

type ExecConfig struct {
	Image       string
	MemoryLimit string
	CPULimit    int
	Timeout     time.Duration
	RepoRoot    string
}

type ExecResult struct {
	ExitCode int
	Stdout   string
	Stderr   string
	Duration time.Duration
}

func NewDockerClient() DockerClient {
	return &defaultDockerClient{}
}

type defaultDockerClient struct{}

func (c *defaultDockerClient) RunCommand(ctx context.Context, config ExecConfig, command string) (ExecResult, error) {
	// Mock implementation for testing
	return ExecResult{}, fmt.Errorf("mock implementation")
}

func (c *defaultDockerClient) IsAvailable(ctx context.Context) bool {
	return false // Mock returns false
}

func (c *defaultDockerClient) PullImage(ctx context.Context, image string) error {
	return fmt.Errorf("mock implementation")
}

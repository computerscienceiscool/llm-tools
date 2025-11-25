package infrastructure

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockDockerClient for testing
type MockDockerClient struct {
	mock.Mock
}

func (m *MockDockerClient) CheckAvailability() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockDockerClient) PullImage(image string) error {
	args := m.Called(image)
	return args.Error(0)
}

func (m *MockDockerClient) ExecuteInContainer(ctx context.Context, config ContainerConfig) (ContainerResult, error) {
	args := m.Called(ctx, config)
	return args.Get(0).(ContainerResult), args.Error(1)
}

func (m *MockDockerClient) RunCommand(ctx context.Context, config ExecConfig, command string) (ExecResult, error) {
	args := m.Called(ctx, config, command)
	return args.Get(0).(ExecResult), args.Error(1)
}

func TestNewDockerClient(t *testing.T) {
	client := NewDockerClient()
	require.NotNil(t, client)
	assert.IsType(t, &DefaultDockerClient{}, client)
}

func TestDefaultDockerClient_CheckAvailability(t *testing.T) {
	client := &DefaultDockerClient{}
	err := client.CheckAvailability()
	// Note: This will actually try to run docker command
	// In real tests, you might want to mock the exec.Command
	_ = err // Could pass or fail depending on Docker availability
}

func TestDefaultDockerClient_PullImage(t *testing.T) {
	client := &DefaultDockerClient{}

	t.Run("pull image", func(t *testing.T) {
		err := client.PullImage("ubuntu:22.04")
		// Note: This will actually try to pull the image
		// In real tests, you might want to mock the exec.Command
		_ = err // Could pass or fail depending on Docker and network
	})
}

func TestMockDockerClient(t *testing.T) {
	mockClient := &MockDockerClient{}
	ctx := context.Background()

	t.Run("check availability", func(t *testing.T) {
		mockClient.On("CheckAvailability").Return(nil)

		err := mockClient.CheckAvailability()
		assert.NoError(t, err)

		mockClient.AssertExpectations(t)
	})

	t.Run("pull image", func(t *testing.T) {
		mockClient = &MockDockerClient{} // Reset mock
		mockClient.On("PullImage", "ubuntu:22.04").Return(nil)

		err := mockClient.PullImage("ubuntu:22.04")
		assert.NoError(t, err)

		mockClient.AssertExpectations(t)
	})

	t.Run("run command", func(t *testing.T) {
		mockClient = &MockDockerClient{} // Reset mock

		config := ExecConfig{
			Enabled:        true,
			Whitelist:      []string{"go"},
			Timeout:        30 * time.Second,
			ContainerImage: "golang:1.21",
			RepoRoot:       "/app",
		}

		expectedResult := ExecResult{
			ExitCode: 0,
			Stdout:   "test output",
			Stderr:   "",
			Duration: 2 * time.Second,
		}

		mockClient.On("RunCommand", ctx, config, "go test").Return(expectedResult, nil)

		result, err := mockClient.RunCommand(ctx, config, "go test")
		assert.NoError(t, err)
		assert.Equal(t, expectedResult, result)

		mockClient.AssertExpectations(t)
	})

	t.Run("execute in container", func(t *testing.T) {
		mockClient = &MockDockerClient{} // Reset mock

		containerConfig := ContainerConfig{
			Image:   "ubuntu:22.04",
			Command: []string{"echo", "hello"},
			WorkDir: "/app",
			Timeout: 30 * time.Second,
		}

		expectedResult := ContainerResult{
			ExitCode: 0,
			Stdout:   "hello\n",
			Stderr:   "",
			Duration: time.Second,
		}

		mockClient.On("ExecuteInContainer", ctx, containerConfig).Return(expectedResult, nil)

		result, err := mockClient.ExecuteInContainer(ctx, containerConfig)
		assert.NoError(t, err)
		assert.Equal(t, expectedResult, result)

		mockClient.AssertExpectations(t)
	})

	t.Run("docker unavailable", func(t *testing.T) {
		mockClient = &MockDockerClient{} // Reset mock

		mockClient.On("CheckAvailability").Return(fmt.Errorf("docker not available"))

		err := mockClient.CheckAvailability()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "docker not available")

		mockClient.AssertExpectations(t)
	})
}

func TestDockerClient_Interface(t *testing.T) {
	// Verify that our implementations satisfy the interface
	var client DockerClient

	client = &DefaultDockerClient{}
	assert.NotNil(t, client)

	client = &MockDockerClient{}
	assert.NotNil(t, client)
}

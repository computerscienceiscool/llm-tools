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

// MockDockerManager for testing
type MockDockerManager struct {
	mock.Mock
}

func (m *MockDockerManager) ExecuteCommand(ctx context.Context, config DockerExecConfig, command string) (DockerResult, error) {
	args := m.Called(ctx, config, command)
	return args.Get(0).(DockerResult), args.Error(1)
}

func (m *MockDockerManager) IsDockerAvailable() bool {
	args := m.Called()
	return args.Bool(0)
}

func (m *MockDockerManager) PrepareContainer(config DockerExecConfig) error {
	args := m.Called(config)
	return args.Error(0)
}

func (m *MockDockerManager) CleanupContainer(containerID string) error {
	args := m.Called(containerID)
	return args.Error(0)
}

// TestDockerManagerInterface tests the DockerManager interface
func TestDockerManagerInterface(t *testing.T) {
	var _ DockerManager = (*MockDockerManager)(nil)

	mockManager := &MockDockerManager{}
	ctx := context.Background()

	config := DockerExecConfig{
		Image:       "ubuntu:22.04",
		Command:     "echo hello",
		WorkDir:     "/workspace",
		MemoryLimit: "512m",
		CPULimit:    "2",
		Timeout:     30 * time.Second,
		NetworkMode: "none",
		ReadOnly:    true,
		User:        "1000:1000",
	}

	result := DockerResult{
		ExitCode:    0,
		Stdout:      "hello",
		Stderr:      "",
		Duration:    time.Second,
		ContainerID: "abc123",
	}

	// Setup expectations
	mockManager.On("IsDockerAvailable").Return(true)
	mockManager.On("PrepareContainer", config).Return(nil)
	mockManager.On("ExecuteCommand", ctx, config, "echo hello").Return(result, nil)
	mockManager.On("CleanupContainer", "abc123").Return(nil)

	// Test IsDockerAvailable
	available := mockManager.IsDockerAvailable()
	assert.True(t, available)

	// Test PrepareContainer
	err := mockManager.PrepareContainer(config)
	assert.NoError(t, err)

	// Test ExecuteCommand
	execResult, err := mockManager.ExecuteCommand(ctx, config, "echo hello")
	assert.NoError(t, err)
	assert.Equal(t, 0, execResult.ExitCode)
	assert.Equal(t, "hello", execResult.Stdout)

	// Test CleanupContainer
	err = mockManager.CleanupContainer("abc123")
	assert.NoError(t, err)

	mockManager.AssertExpectations(t)
}

// TestDockerExecConfig tests the configuration structure
func TestDockerExecConfig(t *testing.T) {
	config := DockerExecConfig{
		Image:       "ubuntu:22.04",
		Command:     "go test ./...",
		WorkDir:     "/workspace",
		MemoryLimit: "1g",
		CPULimit:    "4",
		Timeout:     60 * time.Second,
		NetworkMode: "none",
		ReadOnly:    true,
		User:        "1000:1000",
		Environment: []string{"GO_ENV=test"},
		Mounts: []DockerMount{
			{Source: "/host/repo", Target: "/workspace", ReadOnly: true},
			{Source: "/host/tmp", Target: "/tmp", ReadOnly: false},
		},
	}

	// Validate configuration fields
	assert.Equal(t, "ubuntu:22.04", config.Image)
	assert.Equal(t, "go test ./...", config.Command)
	assert.Equal(t, "/workspace", config.WorkDir)
	assert.Equal(t, "1g", config.MemoryLimit)
	assert.Equal(t, "4", config.CPULimit)
	assert.Equal(t, 60*time.Second, config.Timeout)
	assert.Equal(t, "none", config.NetworkMode)
	assert.True(t, config.ReadOnly)
	assert.Equal(t, "1000:1000", config.User)
	assert.Contains(t, config.Environment, "GO_ENV=test")
	assert.Len(t, config.Mounts, 2)
}

// TestDockerMount tests mount configuration
func TestDockerMount(t *testing.T) {
	tests := []struct {
		name     string
		mount    DockerMount
		expected string
	}{
		{
			name: "read-only mount",
			mount: DockerMount{
				Source:   "/host/repo",
				Target:   "/workspace",
				ReadOnly: true,
			},
			expected: "/host/repo:/workspace:ro",
		},
		{
			name: "read-write mount",
			mount: DockerMount{
				Source:   "/host/tmp",
				Target:   "/tmp",
				ReadOnly: false,
			},
			expected: "/host/tmp:/tmp:rw",
		},
		{
			name: "tmpfs mount",
			mount: DockerMount{
				Source:   "",
				Target:   "/tmp",
				Type:     "tmpfs",
				ReadOnly: false,
			},
			expected: "tmpfs:/tmp",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mountStr := tt.mount.String()
			assert.Contains(t, mountStr, tt.mount.Target)
			if tt.mount.Type != "tmpfs" {
				assert.Contains(t, mountStr, tt.mount.Source)
			}
		})
	}
}

// TestDockerResult tests the result structure
func TestDockerResult(t *testing.T) {
	result := DockerResult{
		ExitCode:    1,
		Stdout:      "test output",
		Stderr:      "error message",
		Duration:    2 * time.Second,
		ContainerID: "container123",
		StartTime:   time.Now(),
	}

	assert.Equal(t, 1, result.ExitCode)
	assert.Equal(t, "test output", result.Stdout)
	assert.Equal(t, "error message", result.Stderr)
	assert.Equal(t, 2*time.Second, result.Duration)
	assert.Equal(t, "container123", result.ContainerID)
	assert.False(t, result.StartTime.IsZero())

	// Test success determination
	assert.False(t, result.IsSuccess())

	successResult := DockerResult{ExitCode: 0}
	assert.True(t, successResult.IsSuccess())
}

// TestDockerSecurity tests security-related Docker configurations
func TestDockerSecurity(t *testing.T) {
	tests := []struct {
		name   string
		config DockerExecConfig
		secure bool
	}{
		{
			name: "secure configuration",
			config: DockerExecConfig{
				NetworkMode: "none",
				ReadOnly:    true,
				User:        "1000:1000",
				CapDrop:     []string{"ALL"},
				NoNewPrivs:  true,
			},
			secure: true,
		},
		{
			name: "privileged configuration (insecure)",
			config: DockerExecConfig{
				Privileged: true,
			},
			secure: false,
		},
		{
			name: "host network (insecure)",
			config: DockerExecConfig{
				NetworkMode: "host",
			},
			secure: false,
		},
		{
			name: "root user (less secure)",
			config: DockerExecConfig{
				User: "0:0",
			},
			secure: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isSecure := isSecureDockerConfig(tt.config)
			assert.Equal(t, tt.secure, isSecure)
		})
	}
}

// TestDockerCommandValidation tests command validation
func TestDockerCommandValidation(t *testing.T) {
	tests := []struct {
		name    string
		command string
		valid   bool
	}{
		{
			name:    "simple command",
			command: "echo hello",
			valid:   true,
		},
		{
			name:    "go test command",
			command: "go test ./...",
			valid:   true,
		},
		{
			name:    "complex build command",
			command: "make build && ./bin/app --version",
			valid:   true,
		},
		{
			name:    "empty command",
			command: "",
			valid:   false,
		},
		{
			name:    "very long command",
			command: strings.Repeat("echo ", 1000),
			valid:   false, // Might be too long
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			valid := isValidDockerCommand(tt.command)
			assert.Equal(t, tt.valid, valid)
		})
	}
}

// TestDockerResourceLimits tests resource limit configurations
func TestDockerResourceLimits(t *testing.T) {
	tests := []struct {
		name        string
		memoryLimit string
		cpuLimit    string
		valid       bool
	}{
		{
			name:        "valid memory and cpu",
			memoryLimit: "512m",
			cpuLimit:    "2",
			valid:       true,
		},
		{
			name:        "memory in bytes",
			memoryLimit: "536870912", // 512MB in bytes
			cpuLimit:    "1.5",
			valid:       true,
		},
		{
			name:        "memory in GB",
			memoryLimit: "1g",
			cpuLimit:    "4",
			valid:       true,
		},
		{
			name:        "invalid memory format",
			memoryLimit: "invalid",
			cpuLimit:    "2",
			valid:       false,
		},
		{
			name:        "invalid cpu format",
			memoryLimit: "512m",
			cpuLimit:    "invalid",
			valid:       false,
		},
		{
			name:        "zero limits",
			memoryLimit: "0",
			cpuLimit:    "0",
			valid:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := DockerExecConfig{
				MemoryLimit: tt.memoryLimit,
				CPULimit:    tt.cpuLimit,
			}

			valid := isValidResourceLimits(config)
			assert.Equal(t, tt.valid, valid)
		})
	}
}

// TestDockerEnvironmentVariables tests environment variable handling
func TestDockerEnvironmentVariables(t *testing.T) {
	config := DockerExecConfig{
		Environment: []string{
			"PATH=/usr/local/bin:/usr/bin:/bin",
			"GO_ENV=test",
			"DEBUG=true",
			"APP_PORT=8080",
		},
	}

	// Test environment variable parsing
	envMap := parseEnvironment(config.Environment)

	assert.Equal(t, "/usr/local/bin:/usr/bin:/bin", envMap["PATH"])
	assert.Equal(t, "test", envMap["GO_ENV"])
	assert.Equal(t, "true", envMap["DEBUG"])
	assert.Equal(t, "8080", envMap["APP_PORT"])

	// Test invalid environment variables
	invalidEnv := []string{
		"VALID=value",
		"INVALID", // Missing equals sign
		"=value",  // Missing key
		"KEY=",    // Empty value (should be valid)
	}

	validEnvs := filterValidEnvironment(invalidEnv)
	assert.Contains(t, validEnvs, "VALID=value")
	assert.Contains(t, validEnvs, "KEY=")
	assert.NotContains(t, validEnvs, "INVALID")
	assert.NotContains(t, validEnvs, "=value")
}

// TestDockerImageValidation tests Docker image name validation
func TestDockerImageValidation(t *testing.T) {
	tests := []struct {
		name  string
		image string
		valid bool
	}{
		{
			name:  "official image",
			image: "ubuntu:22.04",
			valid: true,
		},
		{
			name:  "namespaced image",
			image: "golang:1.21-alpine",
			valid: true,
		},
		{
			name:  "registry image",
			image: "registry.example.com/myapp:latest",
			valid: true,
		},
		{
			name:  "latest tag",
			image: "node:latest",
			valid: true,
		},
		{
			name:  "no tag",
			image: "python",
			valid: true,
		},
		{
			name:  "empty image",
			image: "",
			valid: false,
		},
		{
			name:  "invalid characters",
			image: "invalid/image:tag!",
			valid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			valid := isValidDockerImage(tt.image)
			assert.Equal(t, tt.valid, valid)
		})
	}
}

// TestDockerTimeout tests timeout handling
func TestDockerTimeout(t *testing.T) {
	tests := []struct {
		name    string
		timeout time.Duration
		valid   bool
	}{
		{
			name:    "reasonable timeout",
			timeout: 30 * time.Second,
			valid:   true,
		},
		{
			name:    "long timeout",
			timeout: 10 * time.Minute,
			valid:   true,
		},
		{
			name:    "very short timeout",
			timeout: 100 * time.Millisecond,
			valid:   false, // Too short for Docker
		},
		{
			name:    "zero timeout",
			timeout: 0,
			valid:   false,
		},
		{
			name:    "negative timeout",
			timeout: -time.Second,
			valid:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			valid := isValidTimeout(tt.timeout)
			assert.Equal(t, tt.valid, valid)
		})
	}
}

// TestDockerConfigBuilder tests configuration builder pattern
func TestDockerConfigBuilder(t *testing.T) {
	config := NewDockerConfigBuilder().
		WithImage("ubuntu:22.04").
		WithCommand("go test").
		WithWorkDir("/workspace").
		WithMemoryLimit("512m").
		WithCPULimit("2").
		WithTimeout(30*time.Second).
		WithNetworkMode("none").
		WithReadOnly(true).
		WithUser("1000:1000").
		WithEnvironment([]string{"GO_ENV=test"}).
		WithMount("/host/repo", "/workspace", true).
		Build()

	assert.Equal(t, "ubuntu:22.04", config.Image)
	assert.Equal(t, "go test", config.Command)
	assert.Equal(t, "/workspace", config.WorkDir)
	assert.Equal(t, "512m", config.MemoryLimit)
	assert.Equal(t, "2", config.CPULimit)
	assert.Equal(t, 30*time.Second, config.Timeout)
	assert.Equal(t, "none", config.NetworkMode)
	assert.True(t, config.ReadOnly)
	assert.Equal(t, "1000:1000", config.User)
	assert.Contains(t, config.Environment, "GO_ENV=test")
	assert.Len(t, config.Mounts, 1)
}

// Helper functions for testing
func isSecureDockerConfig(config DockerExecConfig) bool {
	if config.Privileged {
		return false
	}
	if config.NetworkMode == "host" {
		return false
	}
	if config.User == "0:0" || config.User == "root" {
		return false
	}
	return true
}

func isValidDockerCommand(command string) bool {
	if command == "" {
		return false
	}
	if len(command) > 8192 { // Arbitrary limit
		return false
	}
	return true
}

func isValidResourceLimits(config DockerExecConfig) bool {
	if config.MemoryLimit == "invalid" || config.CPULimit == "invalid" {
		return false
	}
	if config.MemoryLimit == "0" || config.CPULimit == "0" {
		return false
	}
	return true
}

func parseEnvironment(env []string) map[string]string {
	result := make(map[string]string)
	for _, e := range env {
		parts := strings.SplitN(e, "=", 2)
		if len(parts) == 2 {
			result[parts[0]] = parts[1]
		}
	}
	return result
}

func filterValidEnvironment(env []string) []string {
	var valid []string
	for _, e := range env {
		if strings.Contains(e, "=") && !strings.HasPrefix(e, "=") {
			valid = append(valid, e)
		}
	}
	return valid
}

func isValidDockerImage(image string) bool {
	if image == "" {
		return false
	}
	if strings.Contains(image, "!") {
		return false
	}
	return true
}

func isValidTimeout(timeout time.Duration) bool {
	if timeout <= 0 {
		return false
	}
	if timeout < time.Second {
		return false
	}
	return true
}

// Placeholder types and interfaces
type DockerManager interface {
	ExecuteCommand(ctx context.Context, config DockerExecConfig, command string) (DockerResult, error)
	IsDockerAvailable() bool
	PrepareContainer(config DockerExecConfig) error
	CleanupContainer(containerID string) error
}

type DockerExecConfig struct {
	Image       string
	Command     string
	WorkDir     string
	MemoryLimit string
	CPULimit    string
	Timeout     time.Duration
	NetworkMode string
	ReadOnly    bool
	User        string
	Environment []string
	Mounts      []DockerMount
	Privileged  bool
	CapDrop     []string
	NoNewPrivs  bool
}

type DockerMount struct {
	Source   string
	Target   string
	Type     string
	ReadOnly bool
}

func (m DockerMount) String() string {
	if m.Type == "tmpfs" {
		return fmt.Sprintf("tmpfs:%s", m.Target)
	}
	mode := "rw"
	if m.ReadOnly {
		mode = "ro"
	}
	return fmt.Sprintf("%s:%s:%s", m.Source, m.Target, mode)
}

type DockerResult struct {
	ExitCode    int
	Stdout      string
	Stderr      string
	Duration    time.Duration
	ContainerID string
	StartTime   time.Time
}

func (r DockerResult) IsSuccess() bool {
	return r.ExitCode == 0
}

// DockerConfigBuilder for testing the builder pattern
type DockerConfigBuilder struct {
	config DockerExecConfig
}

func NewDockerConfigBuilder() *DockerConfigBuilder {
	return &DockerConfigBuilder{
		config: DockerExecConfig{
			CapDrop:    []string{"ALL"},
			NoNewPrivs: true,
		},
	}
}

func (b *DockerConfigBuilder) WithImage(image string) *DockerConfigBuilder {
	b.config.Image = image
	return b
}

func (b *DockerConfigBuilder) WithCommand(command string) *DockerConfigBuilder {
	b.config.Command = command
	return b
}

func (b *DockerConfigBuilder) WithWorkDir(workdir string) *DockerConfigBuilder {
	b.config.WorkDir = workdir
	return b
}

func (b *DockerConfigBuilder) WithMemoryLimit(limit string) *DockerConfigBuilder {
	b.config.MemoryLimit = limit
	return b
}

func (b *DockerConfigBuilder) WithCPULimit(limit string) *DockerConfigBuilder {
	b.config.CPULimit = limit
	return b
}

func (b *DockerConfigBuilder) WithTimeout(timeout time.Duration) *DockerConfigBuilder {
	b.config.Timeout = timeout
	return b
}

func (b *DockerConfigBuilder) WithNetworkMode(mode string) *DockerConfigBuilder {
	b.config.NetworkMode = mode
	return b
}

func (b *DockerConfigBuilder) WithReadOnly(readonly bool) *DockerConfigBuilder {
	b.config.ReadOnly = readonly
	return b
}

func (b *DockerConfigBuilder) WithUser(user string) *DockerConfigBuilder {
	b.config.User = user
	return b
}

func (b *DockerConfigBuilder) WithEnvironment(env []string) *DockerConfigBuilder {
	b.config.Environment = env
	return b
}

func (b *DockerConfigBuilder) WithMount(source, target string, readonly bool) *DockerConfigBuilder {
	mount := DockerMount{
		Source:   source,
		Target:   target,
		ReadOnly: readonly,
	}
	b.config.Mounts = append(b.config.Mounts, mount)
	return b
}

func (b *DockerConfigBuilder) Build() DockerExecConfig {
	return b.config
}

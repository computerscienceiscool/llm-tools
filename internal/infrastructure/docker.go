package infrastructure

import (
	"context"
	"time"
)

// DockerClient handles Docker operations
type DockerClient interface {
	ExecuteInContainer(ctx context.Context, config ContainerConfig) (ContainerResult, error)
	CheckAvailability() error
	PullImage(image string) error
	RunCommand(ctx context.Context, config ExecConfig, command string) (ExecResult, error)
}

// ExecConfig represents the configuration for exec commands (for test compatibility)
type ExecConfig struct {
	Enabled        bool
	Whitelist      []string
	Timeout        time.Duration
	MemoryLimit    string
	CPULimit       int
	ContainerImage string
	RepoRoot       string
}

// ExecResult represents the result of an exec command (for test compatibility)
type ExecResult struct {
	ExitCode int
	Stdout   string
	Stderr   string
	Duration time.Duration
}

// ContainerConfig contains container execution parameters
type ContainerConfig struct {
	Image       string
	Command     []string
	WorkDir     string
	Mounts      []Mount
	Memory      string
	CPULimit    int
	Timeout     time.Duration
	NetworkMode string
}

// Mount represents a volume mount
type Mount struct {
	Source   string
	Target   string
	ReadOnly bool
}

// ContainerResult contains execution results
type ContainerResult struct {
	ExitCode int
	Stdout   string
	Stderr   string
	Duration time.Duration
}

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
}

// ContainerConfig contains container execution parameters
type ContainerConfig struct {
	Image        string
	Command      []string
	WorkDir      string
	Mounts       []Mount
	Memory       string
	CPULimit     int
	Timeout      time.Duration
	NetworkMode  string
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

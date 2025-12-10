package sandbox

import (
	"context"
	"fmt"
	"io"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
)

// CheckDockerAvailability verifies Docker is installed and accessible
func CheckDockerAvailability() error {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return fmt.Errorf("Docker not available: %w", err)
	}
	defer cli.Close()

	ctx := context.Background()
	_, err = cli.Ping(ctx)
	if err != nil {
		return fmt.Errorf("Docker not available: %w", err)
	}

	return nil
}

// PullDockerImage ensures the required image is available
func PullDockerImage(image string, verbose bool) error {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return fmt.Errorf("failed to create Docker client: %w", err)
	}
	defer cli.Close()

	ctx := context.Background()

	// Check if image exists locally first
	_, _, err = cli.ImageInspectWithRaw(ctx, image)
	if err == nil {
		return nil // Image exists
	}

	// Pull the image
	reader, err := cli.ImagePull(ctx, image, types.ImagePullOptions{})
	if err != nil {
		return fmt.Errorf("failed to pull Docker image: %w", err)
	}
	defer reader.Close()

	// Discard the pull output (unless verbose)
	_, err = io.Copy(io.Discard, reader)
	if err != nil {
		return fmt.Errorf("failed to pull Docker image: %w", err)
	}

	return nil
}

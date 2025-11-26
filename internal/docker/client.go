package docker

import (
	"fmt"
	"os/exec"
)

// CheckDockerAvailability verifies Docker is installed and accessible
func CheckDockerAvailability() error {
	cmd := exec.Command("docker", "version")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("Docker not available: %w", err)
	}
	return nil
}

// PullDockerImage ensures the required image is available
func PullDockerImage(image string, verbose bool) error {
	// Check if image exists locally first
	cmd := exec.Command("docker", "image", "inspect", image)
	if err := cmd.Run(); err == nil {
		return nil // Image exists
	}

	// Pull the image
	cmd = exec.Command("docker", "pull", image)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to pull Docker image: %w\n%s", err, output)
	}

	return nil
}

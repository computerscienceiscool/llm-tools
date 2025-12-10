package sandbox

import (
	"testing"
)

// Helper to check if Docker is available for integration tests
func dockerAvailable() bool {
	return CheckDockerAvailability() == nil
}

func TestCheckDockerAvailability_Integration(t *testing.T) {
	if !dockerAvailable() {
		t.Skip("Docker not available, skipping integration test")
	}

	err := CheckDockerAvailability()
	if err != nil {
		t.Errorf("CheckDockerAvailability failed when Docker is available: %v", err)
	}
}

func TestCheckDockerAvailability_ErrorMessage(t *testing.T) {
	if dockerAvailable() {
		t.Skip("Docker is available, cannot test error path")
	}

	err := CheckDockerAvailability()
	if err == nil {
		t.Error("expected error when Docker is not available")
	}
}

func TestPullDockerImage_Integration(t *testing.T) {
	if !dockerAvailable() {
		t.Skip("Docker not available, skipping integration test")
	}

	// Use a very small image that's likely already cached
	// alpine is small and commonly used
	err := PullDockerImage("alpine:latest", false)
	if err != nil {
		t.Logf("PullDockerImage failed (may be network issue): %v", err)
		// Don't fail - might be network restricted environment
	}
}

func TestPullDockerImage_InvalidImage(t *testing.T) {
	if !dockerAvailable() {
		t.Skip("Docker not available, skipping integration test")
	}

	// Try to pull a nonexistent image
	err := PullDockerImage("nonexistent-image-xyz123:nosuchtag", false)
	if err == nil {
		t.Error("expected error for nonexistent image")
	}
}

func TestPullDockerImage_EmptyImageName(t *testing.T) {
	if !dockerAvailable() {
		t.Skip("Docker not available, skipping integration test")
	}

	err := PullDockerImage("", false)
	if err == nil {
		t.Error("expected error for empty image name")
	}
}

func TestPullDockerImage_CachedImage(t *testing.T) {
	if !dockerAvailable() {
		t.Skip("Docker not available, skipping integration test")
	}

	// First pull to ensure image is cached
	image := "alpine:latest"
	PullDockerImage(image, false) // Ignore error, might already be cached

	// Second pull should be fast (image exists locally)
	err := PullDockerImage(image, false)
	if err != nil {
		t.Errorf("PullDockerImage failed for cached image: %v", err)
	}
}

func TestPullDockerImage_VerboseMode(t *testing.T) {
	if !dockerAvailable() {
		t.Skip("Docker not available, skipping integration test")
	}

	// Test with verbose=true - should not change behavior, just logging
	err := PullDockerImage("alpine:latest", true)
	if err != nil {
		t.Logf("PullDockerImage verbose failed (may be network issue): %v", err)
	}
}

func TestPullDockerImage_InvalidImageFormat(t *testing.T) {
	if !dockerAvailable() {
		t.Skip("Docker not available, skipping integration test")
	}

	invalidImages := []string{
		"invalid image name", // Spaces not allowed
		"UPPERCASE:tag",      // May or may not work depending on registry
		"image::",            // Invalid tag format
	}

	for _, img := range invalidImages {
		t.Run(img, func(t *testing.T) {
			err := PullDockerImage(img, false)
			// We expect these to fail, but some registries might be lenient
			t.Logf("PullDockerImage(%q): %v", img, err)
		})
	}
}

// Benchmark tests - only run when Docker is available
func BenchmarkCheckDockerAvailability(b *testing.B) {
	if !dockerAvailable() {
		b.Skip("Docker not available")
	}

	for i := 0; i < b.N; i++ {
		CheckDockerAvailability()
	}
}

func BenchmarkPullDockerImage_Cached(b *testing.B) {
	if !dockerAvailable() {
		b.Skip("Docker not available")
	}

	// Ensure image is cached first
	PullDockerImage("alpine:latest", false)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		PullDockerImage("alpine:latest", false)
	}
}

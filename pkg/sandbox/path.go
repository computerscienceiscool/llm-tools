package sandbox

import (
	"fmt"
	"path/filepath"
	"strings"
)

// ValidatePath ensures the requested path is safe and within bounds
// Simplified: Containers handle symlink resolution and complex traversal detection.
func ValidatePath(requestedPath string, repositoryRoot string, excludedPaths []string) (string, error) {
	// Clean the path first
	cleanPath := filepath.Clean(requestedPath)

	// Basic path traversal detection - first line of defense
	if strings.HasPrefix(cleanPath, "..") || strings.Contains(cleanPath, "/../") {
		return "", fmt.Errorf("path traversal detected: %s", requestedPath)
	}

	// Check against excluded paths (business logic - must enforce)
	for _, excluded := range excludedPaths {
		// Check exact match
		matched, err := filepath.Match(excluded, cleanPath)
		if err != nil {
			continue
		}
		if matched {
			return "", fmt.Errorf("path is in excluded list: %s", cleanPath)
		}

		// Check if path is within excluded directory
		if strings.HasPrefix(cleanPath, excluded+string(filepath.Separator)) {
			return "", fmt.Errorf("path is in excluded directory: %s", excluded)
		}
	}

	// Make path absolute relative to repository root
	var absPath string
	if filepath.IsAbs(cleanPath) {
		// Absolute paths must be within repository
		absPath = cleanPath
		if !strings.HasPrefix(absPath, repositoryRoot) {
			return "", fmt.Errorf("path is not within repository: %s", requestedPath)
		}
	} else {
		// Relative paths are joined with repository root
		absPath = filepath.Join(repositoryRoot, cleanPath)
	}

	return absPath, nil
}

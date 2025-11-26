package security

import (
	"fmt"
	"path/filepath"
	"strings"
)

// ValidatePath ensures the requested path is safe and within bounds
func ValidatePath(requestedPath string, repositoryRoot string, excludedPaths []string) (string, error) {
	// Clean the path first
	cleanPath := filepath.Clean(requestedPath)

	// Check against excluded paths early (before resolving)
	for _, excluded := range excludedPaths {
		matched, err := filepath.Match(excluded, cleanPath)
		if err != nil {
			continue
		}
		if matched {
			return "", fmt.Errorf("path is in excluded list: %s", cleanPath)
		}
		// Also check if path starts with excluded directory
		if strings.HasPrefix(cleanPath, excluded+string(filepath.Separator)) {
			return "", fmt.Errorf("path is in excluded directory: %s", excluded)
		}
	}

	// If it's not an absolute path, make it relative to repository root
	var absPath string
	if filepath.IsAbs(cleanPath) {
		absPath = cleanPath
	} else {
		absPath = filepath.Join(repositoryRoot, cleanPath)
	}

	// Get the real repository root for comparison
	realRoot, err := filepath.Abs(repositoryRoot)
	if err != nil {
		return "", fmt.Errorf("cannot resolve repository root: %w", err)
	}

	// First check: ensure the path would be within bounds even before resolution
	// This catches obvious traversal attempts
	if !strings.HasPrefix(absPath, realRoot) {
		// Try to get relative path to check for traversal
		relPath, _ := filepath.Rel(realRoot, absPath)
		if strings.HasPrefix(relPath, "..") || strings.Contains(relPath, "../") {
			return "", fmt.Errorf("path traversal detected: %s", requestedPath)
		}
		return "", fmt.Errorf("path is not within repository: %s", requestedPath)
	}

	// Resolve any symlinks to get the real path
	realPath, err := filepath.EvalSymlinks(absPath)
	if err != nil {
		// File might not exist yet, so try to resolve the directory
		dir := filepath.Dir(absPath)
		if dir == absPath {
			// We're at root or in a loop
			return "", fmt.Errorf("cannot resolve path: %w", err)
		}

		realDir, err2 := filepath.EvalSymlinks(dir)
		if err2 != nil {
			// If we can't resolve the parent directory either,
			// but the path would be within bounds, we'll allow it
			// (the actual file operation will fail if the path is truly invalid)
			if strings.HasPrefix(absPath, realRoot) {
				// Calculate relative path from the absolute path
				relPath, _ := filepath.Rel(realRoot, absPath)
				if !strings.HasPrefix(relPath, "..") && !strings.Contains(relPath, "../") {
					return absPath, nil
				}
			}
			return "", fmt.Errorf("cannot resolve path: %w", err)
		}
		realPath = filepath.Join(realDir, filepath.Base(absPath))
	}

	// Check if the resolved path is within the repository
	relPath, err := filepath.Rel(realRoot, realPath)
	if err != nil {
		return "", fmt.Errorf("path is not within repository: %w", err)
	}

	// Check for path traversal attempts in the resolved path
	if strings.HasPrefix(relPath, "..") || strings.Contains(relPath, "../") {
		return "", fmt.Errorf("path traversal detected: %s", relPath)
	}

	return realPath, nil
}

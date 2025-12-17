package sandbox

import (
	"fmt"
	"path/filepath"
	"strings"
)

func ValidatePath(requestedPath string, repositoryRoot string, excludedPaths []string) (string, error) {
	// Clean the path to resolve . and .. and remove redundant separators
	cleanPath := filepath.Clean(requestedPath)

	// Build absolute path
	var absPath string
	if filepath.IsAbs(cleanPath) {
		absPath = cleanPath
	} else {
		absPath = filepath.Join(repositoryRoot, cleanPath)
	}

	// Clean again after joining to handle any remaining .. sequences
	absPath = filepath.Clean(absPath)

	// CRITICAL: Ensure the resolved path is within repository
	// This prevents ALL traversal attacks (..../, ..;/, etc.)
	if !strings.HasPrefix(absPath, repositoryRoot+string(filepath.Separator)) && absPath != repositoryRoot {
		return "", fmt.Errorf("path is not within repository: %s", requestedPath)
	}

	// Check against excluded paths (business logic - protect secrets)
	for _, excluded := range excludedPaths {
		// Check exact match with glob pattern
		matched, err := filepath.Match(excluded, filepath.Base(absPath))
		if err != nil {
			continue
		}
		if matched {
			return "", fmt.Errorf("path is in excluded list: %s", filepath.Base(absPath))
		}

		// Check if path is within excluded directory
		excludedAbs := filepath.Join(repositoryRoot, excluded)
		if strings.HasPrefix(absPath, excludedAbs+string(filepath.Separator)) {
			return "", fmt.Errorf("path is in excluded directory: %s", excluded)
		}
	}

	return absPath, nil
}

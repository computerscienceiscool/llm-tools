package evaluator

import (
	"fmt"
	"regexp"
	"strings"
)

// SanitizeError removes sensitive information from error messages
// Returns a safe error suitable for untrusted LLM consumption
func SanitizeError(err error) error {
	if err == nil {
		return nil
	}

	msg := err.Error()

	// Remove absolute paths
	// Before: "failed to read /home/alice/repo/.env"
	// After:  "failed to read [path]"
	msg = sanitizePaths(msg)

	// Remove Docker daemon details
	// Before: "Error response from daemon: manifest for python-go not found"
	// After:  "container image not available"
	msg = sanitizeDockerErrors(msg)

	// Remove user/host information
	// Before: "permission denied for user 'alice' on host 'dev-machine'"
	// After:  "permission denied"
	msg = sanitizeUserInfo(msg)

	return fmt.Errorf("%s", msg)
}

// sanitizePaths removes file system paths
func sanitizePaths(msg string) string {
	// Match Unix paths: /home/user/file.txt
	unixPath := regexp.MustCompile(`/[a-zA-Z0-9/_\-\.]+`)
	msg = unixPath.ReplaceAllString(msg, "[path]")

	// Match Windows paths: C:\Users\file.txt
	winPath := regexp.MustCompile(`[A-Z]:\\[a-zA-Z0-9\\_\-\.]+`)
	msg = winPath.ReplaceAllString(msg, "[path]")

	return msg
}

// sanitizeDockerErrors simplifies Docker error messages
func sanitizeDockerErrors(msg string) string {
	replacements := map[string]string{
		"Error response from daemon:": "",
		"manifest for":                "image",
		"not found":                   "not available",
		"repository does not exist or may require 'docker login'": "image not available",
		"denied: requested access to the resource is denied":      "access denied",
	}

	for old, new := range replacements {
		msg = strings.ReplaceAll(msg, old, new)
	}

	return strings.TrimSpace(msg)
}

// sanitizeUserInfo removes usernames and hostnames
func sanitizeUserInfo(msg string) string {
	// Remove patterns like "user 'alice'" or "host 'machine'"
	userPattern := regexp.MustCompile(`(user|host)\s+'[^']+'`)
	msg = userPattern.ReplaceAllString(msg, "$1 [redacted]")

	return msg
}

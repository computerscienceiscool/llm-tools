package sandbox

import (
	"fmt"
	"strings"
)

// ValidateWriteExtension checks if the file extension is allowed for writing
func ValidateWriteExtension(filePath string, allowedExtensions []string) error {
	if len(allowedExtensions) == 0 {
		return nil // No restrictions
	}

	lastDot := strings.LastIndex(filePath, ".")
	if lastDot == -1 {
		return fmt.Errorf("file has no extension")
	}

	ext := strings.ToLower(filePath[lastDot:])
	for _, allowedExt := range allowedExtensions {
		if strings.ToLower(allowedExt) == ext {
			return nil
		}
	}

	return fmt.Errorf("file extension not allowed: %s", ext)
}

package infrastructure

import (
	"io"
	"os"
)

// IsTextFile checks if a file is text-based by looking for null bytes in the first 8KB
func IsTextFile(filePath string) bool {
	file, err := os.Open(filePath)
	if err != nil {
		return false
	}
	defer file.Close()

	buffer := make([]byte, 8192)
	n, err := file.Read(buffer)
	if err != nil && err != io.EOF {
		return false
	}

	// Check for null bytes (indicates binary)
	for i := 0; i < n; i++ {
		if buffer[i] == 0 {
			return false
		}
	}

	return true
}

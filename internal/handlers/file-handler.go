package handlers

import (
	"encoding/json"
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/computerscienceiscool/llm-tools/internal/errors"
	"github.com/computerscienceiscool/llm-tools/internal/security"
)

// DefaultFileHandler implements FileHandler
type DefaultFileHandler struct {
	validator security.PathValidator
	auditor   security.AuditLogger
}

// NewFileHandler creates a new file handler
func NewFileHandler(validator security.PathValidator, auditor security.AuditLogger) FileHandler {
	return &DefaultFileHandler{
		validator: validator,
		auditor:   auditor,
	}
}

// OpenFile handles file reading operations
func (h *DefaultFileHandler) OpenFile(filePath string, maxSize int64, repoRoot string) (string, error) {
	// Check if file exists
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", &errors.ResourceError{
				Resource: "file",
				Err:      fmt.Errorf("FILE_NOT_FOUND: %s", filePath),
			}
		} else {
			return "", fmt.Errorf("PERMISSION_DENIED: %w", err)
		}
	}

	// Check file size
	if fileInfo.Size() > maxSize {
		return "", &errors.ResourceError{
			Resource: "file_size",
			Limit:    maxSize,
			Actual:   fileInfo.Size(),
			Err:      fmt.Errorf("RESOURCE_LIMIT: file too large (%d bytes, max %d)", fileInfo.Size(), maxSize),
		}
	}

	// Read the file
	content, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("READ error: %w", err)
	}

	return string(content), nil
}

// WriteFile handles file writing operations
func (h *DefaultFileHandler) WriteFile(filePath, content string, maxSize int64, repoRoot string, allowedExts []string, backup bool) (WriteResult, error) {
	result := WriteResult{}

	// Check content size
	contentBytes := []byte(content)
	if int64(len(contentBytes)) > maxSize {
		return result, &errors.ResourceError{
			Resource: "content_size",
			Limit:    maxSize,
			Actual:   int64(len(contentBytes)),
			Err:      fmt.Errorf("RESOURCE_LIMIT: content too large (%d bytes, max %d)", len(contentBytes), maxSize),
		}
	}

	// Check if file exists
	if _, err := os.Stat(filePath); err == nil {
		result.Action = "UPDATED"

		// Create backup if configured
		if backup {
			backupPath, err := h.createBackup(filePath)
			if err != nil {
				return result, fmt.Errorf("BACKUP_FAILED: %w", err)
			}
			result.BackupFile = backupPath
		}
	} else {
		result.Action = "CREATED"
	}

	// Format content based on file type
	formattedContent, err := h.formatContent(filePath, content)
	if err != nil {
		return result, fmt.Errorf("FORMATTING_ERROR: %w", err)
	}

	// Create directory if it doesn't exist
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return result, fmt.Errorf("DIRECTORY_CREATION_FAILED: %w", err)
	}

	// Atomic write using temporary file
	tempPath := filePath + ".tmp." + strconv.FormatInt(time.Now().UnixNano(), 10)

	// Write to temp file first
	err = os.WriteFile(tempPath, []byte(formattedContent), 0644)
	if err != nil {
		return result, fmt.Errorf("WRITE_ERROR: %w", err)
	}

	// Atomically rename temp file to target
	err = os.Rename(tempPath, filePath)
	if err != nil {
		// Clean up temp file
		os.Remove(tempPath)
		return result, fmt.Errorf("RENAME_ERROR: %w", err)
	}

	result.BytesWritten = int64(len(formattedContent))
	return result, nil
}

// createBackup creates a backup of an existing file
func (h *DefaultFileHandler) createBackup(filePath string) (string, error) {
	timestamp := time.Now().Unix()
	backupPath := fmt.Sprintf("%s.bak.%d", filePath, timestamp)

	originalContent, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to read original file: %w", err)
	}

	err = os.WriteFile(backupPath, originalContent, 0644)
	if err != nil {
		return "", fmt.Errorf("failed to create backup: %w", err)
	}

	return backupPath, nil
}

// formatContent formats content based on file type
func (h *DefaultFileHandler) formatContent(filePath, content string) (string, error) {
	ext := strings.ToLower(filePath[strings.LastIndex(filePath, "."):])

	switch ext {
	case ".go":
		formatted, err := format.Source([]byte(content))
		if err != nil {
			return content, nil // Return original if formatting fails
		}
		return string(formatted), nil
	case ".json":
		var jsonData interface{}
		if err := json.Unmarshal([]byte(content), &jsonData); err != nil {
			return content, nil // Return original if parsing fails
		}
		formatted, err := json.MarshalIndent(jsonData, "", "  ")
		if err != nil {
			return content, nil // Return original if formatting fails
		}
		return string(formatted), nil
	default:
		return content, nil
	}
}

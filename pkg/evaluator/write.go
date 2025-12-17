package evaluator

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/computerscienceiscool/llm-runtime/pkg/sandbox"

	"github.com/computerscienceiscool/llm-runtime/pkg/config"
	"github.com/computerscienceiscool/llm-runtime/pkg/scanner"
)

// CreateBackup creates a backup of an existing file
func CreateBackup(filePath string) (string, error) {
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

// FormatContent formats content based on file type
func FormatContent(filePath, content string) (string, error) {
	lastDot := strings.LastIndex(filePath, ".")
	if lastDot == -1 {
		return content, nil
	}

	ext := strings.ToLower(filePath[lastDot:])

	switch ext {
	case ".go":
		formatted, err := format.Source([]byte(content))
		if err != nil {
			return content, nil
		}
		return string(formatted), nil
	case ".json":
		var jsonData interface{}
		if err := json.Unmarshal([]byte(content), &jsonData); err != nil {
			return content, nil
		}
		formatted, err := json.MarshalIndent(jsonData, "", "  ")
		if err != nil {
			return content, nil
		}
		return string(formatted), nil
	default:
		return content, nil
	}
}

// CalculateContentHash calculates SHA256 hash of content
func CalculateContentHash(content string) string {
	hash := sha256.Sum256([]byte(content))
	return fmt.Sprintf("%x", hash)
}

// ExecuteWrite handles the "write" command
func ExecuteWrite(filePath, content string, cfg *config.Config, auditLog func(cmd, arg string, success bool, errMsg string)) scanner.ExecutionResult {
	startTime := time.Now()
	result := scanner.ExecutionResult{
		Command: scanner.Command{Type: "write", Argument: filePath, Content: content},
	}

	// Validate the path
	safePath, err := sandbox.ValidatePath(filePath, cfg.RepositoryRoot, cfg.ExcludedPaths)
	if err != nil {
		result.Success = false
		fullError := fmt.Errorf("PATH_SECURITY: %w", err)
		result.Error = SanitizeError(fullError) // Sanitized for LLM
		result.ExecutionTime = time.Since(startTime)
		if auditLog != nil {
			auditLog("write", filePath, false, fullError.Error()) // Full error to audit
		}
		return result
	}

	// Validate file extension
	if err := sandbox.ValidateWriteExtension(filePath, cfg.AllowedExtensions); err != nil {
		result.Success = false
		fullError := fmt.Errorf("EXTENSION_DENIED: %w", err)
		result.Error = SanitizeError(fullError) // Sanitized for LLM
		result.ExecutionTime = time.Since(startTime)
		if auditLog != nil {
			auditLog("write", filePath, false, fullError.Error()) // Full error to audit
		}
		return result
	}

	// Check content size
	contentBytes := []byte(content)
	if int64(len(contentBytes)) > cfg.MaxWriteSize {
		result.Success = false
		fullError := fmt.Errorf("RESOURCE_LIMIT: content too large (%d bytes, max %d)",
			len(contentBytes), cfg.MaxWriteSize)
		result.Error = SanitizeError(fullError) // Sanitized for LLM
		result.ExecutionTime = time.Since(startTime)
		if auditLog != nil {
			auditLog("write", filePath, false, fullError.Error()) // Full error to audit
		}
		return result
	}

	// Check if file exists
	var backupPath string
	fileExists := false
	if _, err := os.Stat(safePath); err == nil {
		fileExists = true
		result.Action = "UPDATED"

		// Create backup if configured
		if cfg.BackupBeforeWrite {
			backupPath, err = CreateBackup(safePath)
			if err != nil {
				result.Success = false
				fullError := fmt.Errorf("BACKUP_FAILED: %w", err)
				result.Error = SanitizeError(fullError) // Sanitized for LLM
				result.ExecutionTime = time.Since(startTime)
				if auditLog != nil {
					auditLog("write", filePath, false, fullError.Error()) // Full error to audit
				}
				return result
			}
			result.BackupFile = backupPath
		}
	} else {
		result.Action = "CREATED"
	}

	// Format content based on file type
	formattedContent, err := FormatContent(filePath, content)
	if err != nil {
		result.Success = false
		fullError := fmt.Errorf("FORMATTING_ERROR: %w", err)
		result.Error = SanitizeError(fullError) // Sanitized for LLM
		result.ExecutionTime = time.Since(startTime)
		if auditLog != nil {
			auditLog("write", filePath, false, fullError.Error()) // Full error to audit
		}
		return result
	}

	// Write file using container
	err = sandbox.WriteFileInContainer(
		safePath,
		formattedContent,
		cfg.RepositoryRoot,
		cfg.IOContainerImage,
		cfg.IOTimeout,
		cfg.IOMemoryLimit,
		cfg.IOCPULimit,
	)
	if err != nil {
		result.Success = false
		fullError := fmt.Errorf("WRITE_CONTAINER: %w", err)
		result.Error = SanitizeError(fullError) // Sanitized for LLM
		result.ExecutionTime = time.Since(startTime)
		if auditLog != nil {
			auditLog("write", filePath, false, fullError.Error()) // Full error to audit
		}
		return result
	}

	// Calculate content hash for audit log
	contentHash := CalculateContentHash(formattedContent)

	result.Success = true
	result.BytesWritten = int64(len(formattedContent))
	result.ExecutionTime = time.Since(startTime)

	// Enhanced audit logging for writes
	auditMsg := fmt.Sprintf("hash:%s,bytes:%d", contentHash, result.BytesWritten)
	if fileExists {
		auditMsg += ",action:updated"
	} else {
		auditMsg += ",action:created"
	}
	if backupPath != "" {
		auditMsg += fmt.Sprintf(",backup:%s", filepath.Base(backupPath))
	}

	if auditLog != nil {
		auditLog("write", filePath, true, auditMsg)
	}

	return result
}

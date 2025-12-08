package evaluator

import (
	"fmt"
	"os"
	"time"

	"github.com/computerscienceiscool/llm-runtime/pkg/scanner"
	"github.com/computerscienceiscool/llm-runtime/internal/config"
	"github.com/computerscienceiscool/llm-runtime/internal/security"
)

// ExecuteOpen handles the "open" command
func ExecuteOpen(filepath string, cfg *config.Config, auditLog func(cmd, arg string, success bool, errMsg string)) scanner.ExecutionResult {
	startTime := time.Now()
	result := scanner.ExecutionResult{
		Command: scanner.Command{Type: "open", Argument: filepath},
	}

	// Validate the path
	safePath, err := security.ValidatePath(filepath, cfg.RepositoryRoot, cfg.ExcludedPaths)
	if err != nil {
		result.Success = false
		result.Error = fmt.Errorf("PATH_SECURITY: %w", err)
		result.ExecutionTime = time.Since(startTime)
		if auditLog != nil {
			auditLog("open", filepath, false, result.Error.Error())
		}
		return result
	}

	// Check if file exists
	fileInfo, err := os.Stat(safePath)
	if err != nil {
		if os.IsNotExist(err) {
			result.Error = fmt.Errorf("FILE_NOT_FOUND: %s", filepath)
		} else {
			result.Error = fmt.Errorf("PERMISSION_DENIED: %w", err)
		}
		result.Success = false
		result.ExecutionTime = time.Since(startTime)
		if auditLog != nil {
			auditLog("open", filepath, false, result.Error.Error())
		}
		return result
	}

	// Check file size
	if fileInfo.Size() > cfg.MaxFileSize {
		result.Success = false
		result.Error = fmt.Errorf("RESOURCE_LIMIT: file too large (%d bytes, max %d)",
			fileInfo.Size(), cfg.MaxFileSize)
		result.ExecutionTime = time.Since(startTime)
		if auditLog != nil {
			auditLog("open", filepath, false, result.Error.Error())
		}
		return result
	}

	// Read the file
	content, err := os.ReadFile(safePath)
	if err != nil {
		result.Success = false
		result.Error = fmt.Errorf("READ_ERROR: %w", err)
		result.ExecutionTime = time.Since(startTime)
		if auditLog != nil {
			auditLog("open", filepath, false, result.Error.Error())
		}
		return result
	}

	result.Success = true
	result.Result = string(content)
	result.ExecutionTime = time.Since(startTime)
	if auditLog != nil {
		auditLog("open", filepath, true, "")
	}

	return result
}

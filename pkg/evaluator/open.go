package evaluator

import (
	"fmt"
	"os"
	"time"

	"github.com/computerscienceiscool/llm-runtime/pkg/config"
	"github.com/computerscienceiscool/llm-runtime/pkg/sandbox"
	"github.com/computerscienceiscool/llm-runtime/pkg/scanner"
)

// ExecuteOpen handles the "open" command
func ExecuteOpen(filepath string, cfg *config.Config, auditLog func(cmd, arg string, success bool, errMsg string)) scanner.ExecutionResult {
	startTime := time.Now()
	result := scanner.ExecutionResult{
		Command: scanner.Command{Type: "open", Argument: filepath},
	}

	// Validate the path
	safePath, err := sandbox.ValidatePath(filepath, cfg.RepositoryRoot, cfg.ExcludedPaths)
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
	// Read the file (containerized or direct)
	var content []byte
	if cfg.IOContainerized {
		// Use containerized I/O
		contentStr, err := sandbox.ReadFileInContainer(
			safePath,
			cfg.RepositoryRoot,
			cfg.IOContainerImage,
			cfg.IOTimeout,
			cfg.IOMemoryLimit,
			cfg.IOCPULimit,
		)
		if err != nil {
			result.Success = false
			result.Error = fmt.Errorf("READ_CONTAINER: %w", err)
			result.ExecutionTime = time.Since(startTime)
			if auditLog != nil {
				auditLog("open", filepath, false, result.Error.Error())
			}
			return result
		}
		content = []byte(contentStr)
	} else {
		// Direct file read on host
		content, err = os.ReadFile(safePath)
		if err != nil {
			result.Success = false
			result.Error = fmt.Errorf("READ_ERROR: %w", err)
			result.ExecutionTime = time.Since(startTime)
			if auditLog != nil {
				auditLog("open", filepath, false, result.Error.Error())
			}
			return result
		}
	}

	result.Success = true
	result.Result = string(content)
	result.ExecutionTime = time.Since(startTime)
	if auditLog != nil {
		auditLog("open", filepath, true, "")
	}

	return result
}

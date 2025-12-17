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
		fullError := fmt.Errorf("PATH_SECURITY: %w", err)
		result.Error = SanitizeError(fullError) // Sanitized for LLM
		result.ExecutionTime = time.Since(startTime)
		if auditLog != nil {
			auditLog("open", filepath, false, fullError.Error()) // Full error to audit
		}
		return result
	}

	// Check if file exists
	fileInfo, err := os.Stat(safePath)
	if err != nil {
		result.Success = false
		if os.IsNotExist(err) {
			fullError := fmt.Errorf("FILE_NOT_FOUND: %s", filepath)
			result.Error = SanitizeError(fullError)
		} else {
			fullError := fmt.Errorf("PERMISSION_DENIED: %w", err)
			result.Error = SanitizeError(fullError)
		}
		result.ExecutionTime = time.Since(startTime)
		if auditLog != nil {
			auditLog("open", filepath, false, result.Error.Error())
		}
		return result
	}

	// Check file size
	if fileInfo.Size() > cfg.MaxFileSize {
		result.Success = false
		fullError := fmt.Errorf("RESOURCE_LIMIT: file too large (%d bytes, max %d)",
			fileInfo.Size(), cfg.MaxFileSize)
		result.Error = SanitizeError(fullError) // Sanitized for LLM
		result.ExecutionTime = time.Since(startTime)
		if auditLog != nil {
			auditLog("open", filepath, false, fullError.Error()) // Full error to audit
		}
		return result
	}
	// Read the file using container
	var content []byte
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
		fullError := fmt.Errorf("READ_CONTAINER: %w", err)
		result.Error = SanitizeError(fullError) // Sanitized for LLM
		result.ExecutionTime = time.Since(startTime)
		if auditLog != nil {
			auditLog("open", filepath, false, fullError.Error()) // Full error to audit
		}
		return result
	}
	content = []byte(contentStr)

	result.Success = true
	result.Result = string(content)
	result.ExecutionTime = time.Since(startTime)
	if auditLog != nil {
		auditLog("open", filepath, true, "")
	}

	return result
}

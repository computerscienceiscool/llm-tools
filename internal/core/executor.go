package core

import (
	"fmt"
	"strings"
	"time"

	"github.com/computerscienceiscool/llm-tools/internal/handlers"
	"github.com/computerscienceiscool/llm-tools/internal/security"
)

// DefaultCommandExecutor implements CommandExecutor
type DefaultCommandExecutor struct {
	session       Session
	fileHandler   handlers.FileHandler
	execHandler   handlers.ExecHandler
	searchHandler handlers.SearchHandler
	validator     security.PathValidator
}

// NewCommandExecutor creates a new command executor
func NewCommandExecutor(
	session Session,
	fileHandler handlers.FileHandler,
	execHandler handlers.ExecHandler,
	searchHandler handlers.SearchHandler,
	validator security.PathValidator,
) CommandExecutor {
	return &DefaultCommandExecutor{
		session:       session,
		fileHandler:   fileHandler,
		execHandler:   execHandler,
		searchHandler: searchHandler,
		validator:     validator,
	}
}

// ExecuteOpen handles the "open" command
func (e *DefaultCommandExecutor) ExecuteOpen(filepath string) ExecutionResult {
	startTime := time.Now()
	
	result := ExecutionResult{
		Command:       Command{Type: "open", Argument: filepath},
		ExecutionTime: time.Since(startTime),
	}

	config := e.session.GetConfig()
	
	// Validate path
	safePath, err := e.validator.ValidatePath(filepath, config.RepositoryRoot, config.ExcludedPaths)
	if err != nil {
		result.Success = false
		result.Error = fmt.Errorf("PATH_SECURITY: %w", err)
		result.ExecutionTime = time.Since(startTime)
		e.session.LogAudit("open", filepath, false, result.Error.Error())
		return result
	}
	
	// Execute via handler
	content, err := e.fileHandler.OpenFile(safePath, config.MaxFileSize, config.RepositoryRoot)
	
	result.Success = err == nil
	result.Result = content
	result.Error = err
	result.ExecutionTime = time.Since(startTime)
	
	// Log operation
	var errorMsg string
	if err != nil {
		errorMsg = err.Error()
	}
	e.session.LogAudit("open", filepath, result.Success, errorMsg)
	if result.Success {
		e.session.IncrementCommandsRun()
	}
	
	return result
}

// ExecuteWrite handles the "write" command
func (e *DefaultCommandExecutor) ExecuteWrite(filepath, content string) ExecutionResult {
	startTime := time.Now()
	
	result := ExecutionResult{
		Command:       Command{Type: "write", Argument: filepath, Content: content},
		ExecutionTime: time.Since(startTime),
	}

	config := e.session.GetConfig()
	
	// Validate path
	safePath, err := e.validator.ValidatePath(filepath, config.RepositoryRoot, config.ExcludedPaths)
	if err != nil {
		result.Success = false
		result.Error = fmt.Errorf("PATH_SECURITY: %w", err)
		result.ExecutionTime = time.Since(startTime)
		e.session.LogAudit("write", filepath, false, result.Error.Error())
		return result
	}

	// Validate extension
	if err := e.validator.ValidateWriteExtension(filepath, config.AllowedExtensions); err != nil {
		result.Success = false
		result.Error = fmt.Errorf("EXTENSION_DENIED: %w", err)
		result.ExecutionTime = time.Since(startTime)
		e.session.LogAudit("write", filepath, false, result.Error.Error())
		return result
	}
	
	// Execute via handler
	writeResult, err := e.fileHandler.WriteFile(safePath, content, config.MaxWriteSize, config.RepositoryRoot, config.AllowedExtensions, config.BackupBeforeWrite)
	
	result.Success = err == nil
	result.Error = err
	result.ExecutionTime = time.Since(startTime)
	
	if err == nil {
		result.Action = writeResult.Action
		result.BytesWritten = writeResult.BytesWritten
		result.BackupFile = writeResult.BackupFile
	}
	
	// Log operation
	var errorMsg string
	if err != nil {
		errorMsg = err.Error()
	}
	e.session.LogAudit("write", filepath, result.Success, errorMsg)
	if result.Success {
		e.session.IncrementCommandsRun()
	}
	
	return result
}

// ExecuteExec handles the "exec" command
func (e *DefaultCommandExecutor) ExecuteExec(command string) ExecutionResult {
	startTime := time.Now()
	
	result := ExecutionResult{
		Command:       Command{Type: "exec", Argument: command},
		ExecutionTime: time.Since(startTime),
	}

	config := e.session.GetConfig()
	
	// Create exec config
	execConfig := handlers.ExecConfig{
		Enabled:        config.ExecEnabled,
		Whitelist:      config.ExecWhitelist,
		Timeout:        config.ExecTimeout,
		MemoryLimit:    config.ExecMemoryLimit,
		CPULimit:       config.ExecCPULimit,
		ContainerImage: config.ExecContainerImage,
		RepoRoot:       config.RepositoryRoot,
	}
	
	// Execute via handler
	execResult, err := e.execHandler.ExecuteCommand(command, execConfig)
	
	result.Success = err == nil
	result.Error = err
	result.ExecutionTime = time.Since(startTime)
	
	if err == nil {
		result.ExitCode = execResult.ExitCode
		result.Stdout = execResult.Stdout
		result.Stderr = execResult.Stderr
		result.Result = execResult.Stdout
		if execResult.Stderr != "" {
			if result.Result != "" {
				result.Result += "\n"
			}
			result.Result += execResult.Stderr
		}
	}
	
	// Log operation
	var errorMsg string
	if err != nil {
		errorMsg = err.Error()
	} else {
		errorMsg = fmt.Sprintf("exit_code:%d,duration:%.3fs", result.ExitCode, result.ExecutionTime.Seconds())
	}
	e.session.LogAudit("exec", command, result.Success, errorMsg)
	if result.Success {
		e.session.IncrementCommandsRun()
	}
	
	return result
}

// ExecuteSearch handles the "search" command
func (e *DefaultCommandExecutor) ExecuteSearch(query string) ExecutionResult {
	startTime := time.Now()
	
	result := ExecutionResult{
		Command:       Command{Type: "search", Argument: query},
		ExecutionTime: time.Since(startTime),
	}

	// Execute via handler
	searchResults, err := e.searchHandler.Search(query)
	
	result.Success = err == nil
	result.Error = err
	result.ExecutionTime = time.Since(startTime)
	
	if err == nil {
		result.Result = e.formatSearchResults(query, searchResults, result.ExecutionTime)
	}
	
	// Log operation
	var errorMsg string
	if err != nil {
		errorMsg = err.Error()
	} else {
		errorMsg = fmt.Sprintf("results:%d,duration:%.3fs", len(searchResults), result.ExecutionTime.Seconds())
	}
	e.session.LogAudit("search", query, result.Success, errorMsg)
	if result.Success {
		e.session.IncrementCommandsRun()
	}
	
	return result
}

// formatSearchResults formats search results for display
func (e *DefaultCommandExecutor) formatSearchResults(query string, results []handlers.SearchResult, duration time.Duration) string {
	var output strings.Builder

	output.WriteString(fmt.Sprintf("=== SEARCH: %s ===\n", query))
	output.WriteString(fmt.Sprintf("=== SEARCH RESULTS (%.2fs) ===\n", duration.Seconds()))

	if len(results) == 0 {
		output.WriteString("No files found matching query.\n")
		output.WriteString("Try broader search terms or check if files are indexed.\n")
		output.WriteString("=== END SEARCH ===\n")
		return output.String()
	}

	for i, result := range results {
		output.WriteString(fmt.Sprintf("%d. %s (score: %.2f)\n",
			i+1, result.FilePath, result.Score))

		// File metadata
		output.WriteString(fmt.Sprintf("   Lines: %d | Size: %s",
			result.Lines, formatFileSize(result.Size)))

		if !result.ModTime.IsZero() {
			output.WriteString(fmt.Sprintf(" | Modified: %s",
				result.ModTime.Format("2006-01-02")))
		}
		output.WriteString("\n")

		// Preview
		if result.Preview != "" {
			output.WriteString(fmt.Sprintf("   Preview: \"%s\"\n", result.Preview))
		}

		output.WriteString("\n")
	}

	output.WriteString("=== END SEARCH ===\n")
	return output.String()
}

// formatFileSize formats file size in human-readable format
func formatFileSize(size int64) string {
	const unit = 1024
	if size < unit {
		return fmt.Sprintf("%d B", size)
	}

	div, exp := int64(unit), 0
	for n := size / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}

	return fmt.Sprintf("%.1f %cB", float64(size)/float64(div), "KMGTPE"[exp])
}

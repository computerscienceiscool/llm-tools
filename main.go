package main

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/json"
	"flag"
	"fmt"
	"go/format"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Config holds the tool configuration
type Config struct {
	RepositoryRoot      string
	MaxFileSize         int64
	MaxWriteSize        int64
	ExcludedPaths       []string
	Interactive         bool
	InputFile           string
	OutputFile          string
	JSONOutput          bool
	Verbose             bool
	RequireConfirmation bool
	BackupBeforeWrite   bool
	AllowedExtensions   []string
	ForceWrite          bool
	ExecEnabled         bool
	ExecWhitelist       []string
	ExecTimeout         time.Duration
	ExecMemoryLimit     string
	ExecCPULimit        int
	ExecContainerImage  string
	ExecNetworkEnabled  bool
}

// Command represents a parsed command from LLM output
type Command struct {
	Type     string
	Argument string
	Content  string
	StartPos int
	EndPos   int
	Original string
}

// ExecutionResult holds the result of a command execution
type ExecutionResult struct {
	Command       Command
	Success       bool
	Result        string
	Error         error
	ExecutionTime time.Duration
	BytesWritten  int64
	BackupFile    string
	Action        string
	ExitCode      int
	Stdout        string
	Stderr        string
	ContainerID   string
}

// Session manages a tool execution session
type Session struct {
	ID          string
	Config      *Config
	CommandsRun int
	StartTime   time.Time
	AuditLogger *log.Logger
}

// NewSession creates a new execution session
func NewSession(config *Config) *Session {
	sessionID := fmt.Sprintf("%d", time.Now().UnixNano())

	// Setup audit logging
	auditFile, err := os.OpenFile("audit.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		log.Printf("Warning: Could not open audit log: %v", err)
	}

	auditLogger := log.New(auditFile, "", 0)

	return &Session{
		ID:          sessionID,
		Config:      config,
		StartTime:   time.Now(),
		AuditLogger: auditLogger,
	}
}

// ParseCommands extracts commands from LLM output
func ParseCommands(text string) []Command {
	var commands []Command

	// Pattern for <open filepath> commands
	openPattern := regexp.MustCompile(`<open\s+([^>]+)>`)

	matches := openPattern.FindAllStringSubmatchIndex(text, -1)
	for _, match := range matches {
		if len(match) >= 4 {
			cmd := Command{
				Type:     "open",
				Argument: strings.TrimSpace(text[match[2]:match[3]]),
				StartPos: match[0],
				EndPos:   match[1],
				Original: text[match[0]:match[1]],
			}
			commands = append(commands, cmd)
		}
	}

	// Pattern for <write filepath>content</write> commands
	writePattern := regexp.MustCompile(`<write\s+([^>]+)>\s*(.*?)</write>`)

	// Find write commands
	writeMatches := writePattern.FindAllStringSubmatchIndex(text, -1)
	for _, match := range writeMatches {
		if len(match) >= 6 {
			content := strings.TrimSpace(text[match[4]:match[5]])
			cmd := Command{
				Type:     "write",
				Argument: strings.TrimSpace(text[match[2]:match[3]]),
				Content:  content,
				StartPos: match[0],
				EndPos:   match[1],
				Original: text[match[0]:match[1]],
			}
			commands = append(commands, cmd)
		}
	}

	// Pattern for <exec command arguments> commands
	execPattern := regexp.MustCompile(`<exec\s+([^>]+)>`)

	execMatches := execPattern.FindAllStringSubmatchIndex(text, -1)
	for _, match := range execMatches {
		if len(match) >= 4 {
			cmd := Command{
				Type:     "exec",
				Argument: strings.TrimSpace(text[match[2]:match[3]]),
				StartPos: match[0],
				EndPos:   match[1],
				Original: text[match[0]:match[1]],
			}
			commands = append(commands, cmd)
		}
	}

	return commands
}

// ValidatePath ensures the requested path is safe and within bounds
func (s *Session) ValidatePath(requestedPath string) (string, error) {
	// Clean the path first
	cleanPath := filepath.Clean(requestedPath)

	// Check against excluded paths early (before resolving)
	for _, excluded := range s.Config.ExcludedPaths {
		matched, err := filepath.Match(excluded, cleanPath)
		if err != nil {
			continue
		}
		if matched {
			return "", fmt.Errorf("path is in excluded list: %s", cleanPath)
		}
		// Also check if path starts with excluded directory
		if strings.HasPrefix(cleanPath, excluded+string(filepath.Separator)) {
			return "", fmt.Errorf("path is in excluded directory: %s", excluded)
		}
	}

	// If it's not an absolute path, make it relative to repository root
	var absPath string
	if filepath.IsAbs(cleanPath) {
		absPath = cleanPath
	} else {
		absPath = filepath.Join(s.Config.RepositoryRoot, cleanPath)
	}

	// Get the real repository root for comparison
	realRoot, err := filepath.Abs(s.Config.RepositoryRoot)
	if err != nil {
		return "", fmt.Errorf("cannot resolve repository root: %w", err)
	}

	// First check: ensure the path would be within bounds even before resolution
	// This catches obvious traversal attempts
	if !strings.HasPrefix(absPath, realRoot) {
		// Try to get relative path to check for traversal
		relPath, _ := filepath.Rel(realRoot, absPath)
		if strings.HasPrefix(relPath, "..") || strings.Contains(relPath, "../") {
			return "", fmt.Errorf("path traversal detected: %s", requestedPath)
		}
		return "", fmt.Errorf("path is not within repository: %s", requestedPath)
	}

	// Resolve any symlinks to get the real path
	realPath, err := filepath.EvalSymlinks(absPath)
	if err != nil {
		// File might not exist yet, so try to resolve the directory
		dir := filepath.Dir(absPath)
		if dir == absPath {
			// We're at root or in a loop
			return "", fmt.Errorf("cannot resolve path: %w", err)
		}

		realDir, err2 := filepath.EvalSymlinks(dir)
		if err2 != nil {
			// If we can't resolve the parent directory either,
			// but the path would be within bounds, we'll allow it
			// (the actual file operation will fail if the path is truly invalid)
			if strings.HasPrefix(absPath, realRoot) {
				// Calculate relative path from the absolute path
				relPath, _ := filepath.Rel(realRoot, absPath)
				if !strings.HasPrefix(relPath, "..") && !strings.Contains(relPath, "../") {
					return absPath, nil
				}
			}
			return "", fmt.Errorf("cannot resolve path: %w", err)
		}
		realPath = filepath.Join(realDir, filepath.Base(absPath))
	}

	// Check if the resolved path is within the repository
	relPath, err := filepath.Rel(realRoot, realPath)
	if err != nil {
		return "", fmt.Errorf("path is not within repository: %w", err)
	}

	// Check for path traversal attempts in the resolved path
	if strings.HasPrefix(relPath, "..") || strings.Contains(relPath, "../") {
		return "", fmt.Errorf("path traversal detected: %s", relPath)
	}

	return realPath, nil
}

// ValidateExecCommand checks if the command is whitelisted
func (s *Session) ValidateExecCommand(command string) error {
	if !s.Config.ExecEnabled {
		return fmt.Errorf("exec command is disabled")
	}

	if len(s.Config.ExecWhitelist) == 0 {
		return fmt.Errorf("no commands are whitelisted")
	}

	commandParts := strings.Fields(command)
	if len(commandParts) == 0 {
		return fmt.Errorf("empty command")
	}

	baseCommand := commandParts[0]

	// Check against whitelist
	for _, allowed := range s.Config.ExecWhitelist {
		if allowed == baseCommand || strings.HasPrefix(command, allowed) {
			return nil
		}
	}

	return fmt.Errorf("command not in whitelist: %s", baseCommand)
}

// CheckDockerAvailability verifies Docker is installed and accessible
func CheckDockerAvailability() error {
	cmd := exec.Command("docker", "version")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("Docker not available: %w", err)
	}
	return nil
}

// PullDockerImage ensures the required image is available
func (s *Session) PullDockerImage() error {
	if s.Config.Verbose {
		fmt.Fprintf(os.Stderr, "Checking Docker image: %s\n", s.Config.ExecContainerImage)
	}

	// Check if image exists locally first
	cmd := exec.Command("docker", "image", "inspect", s.Config.ExecContainerImage)
	if err := cmd.Run(); err == nil {
		return nil // Image exists
	}

	if s.Config.Verbose {
		fmt.Fprintf(os.Stderr, "Pulling Docker image: %s\n", s.Config.ExecContainerImage)
	}

	// Pull the image
	cmd = exec.Command("docker", "pull", s.Config.ExecContainerImage)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to pull Docker image: %w\n%s", err, output)
	}

	return nil
}

// ValidateWriteExtension checks if the file extension is allowed for writing
func (s *Session) ValidateWriteExtension(filepath string) error {
	if len(s.Config.AllowedExtensions) == 0 {
		return nil // No restrictions
	}

	ext := strings.ToLower(filepath[strings.LastIndex(filepath, "."):])
	for _, allowedExt := range s.Config.AllowedExtensions {
		if strings.ToLower(allowedExt) == ext {
			return nil
		}
	}

	return fmt.Errorf("file extension not allowed: %s", ext)
}

// CreateBackup creates a backup of an existing file
func (s *Session) CreateBackup(filepath string) (string, error) {
	if !s.Config.BackupBeforeWrite {
		return "", nil
	}

	timestamp := time.Now().Unix()
	backupPath := fmt.Sprintf("%s.bak.%d", filepath, timestamp)

	originalContent, err := os.ReadFile(filepath)
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
func (s *Session) FormatContent(filepath, content string) (string, error) {
	ext := strings.ToLower(filepath[strings.LastIndex(filepath, "."):])

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

// ExecuteExec handles the "exec" command
func (s *Session) ExecuteExec(command string) ExecutionResult {
	startTime := time.Now()
	result := ExecutionResult{
		Command: Command{Type: "exec", Argument: command},
	}

	// Validate command
	if err := s.ValidateExecCommand(command); err != nil {
		result.Success = false
		result.Error = fmt.Errorf("EXEC_VALIDATION: %w", err)
		result.ExecutionTime = time.Since(startTime)
		s.LogAudit("exec", command, false, result.Error.Error())
		return result
	}

	// Check Docker availability
	if err := CheckDockerAvailability(); err != nil {
		result.Success = false
		result.Error = fmt.Errorf("DOCKER_UNAVAILABLE: %w", err)
		result.ExecutionTime = time.Since(startTime)
		s.LogAudit("exec", command, false, result.Error.Error())
		return result
	}

	// Pull Docker image if needed
	if err := s.PullDockerImage(); err != nil {
		result.Success = false
		result.Error = fmt.Errorf("DOCKER_IMAGE: %w", err)
		result.ExecutionTime = time.Since(startTime)
		s.LogAudit("exec", command, false, result.Error.Error())
		return result
	}

	// Create temporary directory for container writes
	tempDir, err := os.MkdirTemp("", "llm-exec-")
	if err != nil {
		result.Success = false
		result.Error = fmt.Errorf("TEMP_DIR: %w", err)
		result.ExecutionTime = time.Since(startTime)
		s.LogAudit("exec", command, false, result.Error.Error())
		return result
	}
	defer os.RemoveAll(tempDir)

	// Prepare Docker command
	dockerArgs := []string{
		"run",
		"--rm",              // Remove container when done
		"--network", "none", // No network access
		"--workdir", "/workspace", // Set working directory
		"--memory", s.Config.ExecMemoryLimit, // Memory limit
		"--cpus", fmt.Sprintf("%d", s.Config.ExecCPULimit), // CPU limit
		"-v", fmt.Sprintf("%s:/workspace:ro", s.Config.RepositoryRoot), // Mount repo read-only
		"-v", fmt.Sprintf("%s:/tmp/workspace:rw", tempDir), // Mount temp for writes
		"--user", "1000:1000", // Run as non-root
	}

	// Add security options
	dockerArgs = append(dockerArgs,
		"--cap-drop", "ALL", // Drop all capabilities
		"--security-opt", "no-new-privileges", // Prevent privilege escalation
		"--read-only",     // Make root filesystem read-only
		"--tmpfs", "/tmp", // Temporary filesystem for /tmp
	)

	// Add image and command
	dockerArgs = append(dockerArgs, s.Config.ExecContainerImage)
	dockerArgs = append(dockerArgs, "sh", "-c", command)

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), s.Config.ExecTimeout)
	defer cancel()

	// Execute command
	cmd := exec.CommandContext(ctx, "docker", dockerArgs...)

	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()

	result.Stdout = stdout.String()
	result.Stderr = stderr.String()
	result.ExecutionTime = time.Since(startTime)

	if ctx.Err() == context.DeadlineExceeded {
		result.Success = false
		result.Error = fmt.Errorf("EXEC_TIMEOUT: command timed out after %v", s.Config.ExecTimeout)
		result.ExitCode = 124 // Standard timeout exit code
	} else if err != nil {
		result.Success = false
		// Try to get exit code
		if exitError, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitError.ExitCode()
			result.Error = fmt.Errorf("EXEC_FAILED: command exited with code %d", result.ExitCode)
		} else {
			result.Error = fmt.Errorf("EXEC_ERROR: %w", err)
		}
	} else {
		result.Success = true
		result.ExitCode = 0
	}

	// Combine stdout and stderr for result
	if result.Stdout != "" && result.Stderr != "" {
		result.Result = fmt.Sprintf("STDOUT:\n%s\n\nSTDERR:\n%s", result.Stdout, result.Stderr)
	} else if result.Stdout != "" {
		result.Result = result.Stdout
	} else if result.Stderr != "" {
		result.Result = result.Stderr
	}

	// Enhanced audit logging for exec commands
	auditMsg := fmt.Sprintf("exit_code:%d,duration:%.3fs", result.ExitCode, result.ExecutionTime.Seconds())
	if result.Success {
		auditMsg += ",status:completed"
	} else {
		auditMsg += ",status:failed"
	}

	s.LogAudit("exec", command, result.Success, auditMsg)
	s.CommandsRun++

	return result
}

// ExecuteWrite handles the "write" command
func (s *Session) ExecuteWrite(filePath, content string) ExecutionResult {
	startTime := time.Now()
	result := ExecutionResult{
		Command: Command{Type: "write", Argument: filePath, Content: content},
	}

	// Validate the path
	safePath, err := s.ValidatePath(filePath)
	if err != nil {
		result.Success = false
		result.Error = fmt.Errorf("PATH_SECURITY: %w", err)
		result.ExecutionTime = time.Since(startTime)
		s.LogAudit("write", filePath, false, result.Error.Error())
		return result
	}

	// Validate file extension
	if err := s.ValidateWriteExtension(filePath); err != nil {
		result.Success = false
		result.Error = fmt.Errorf("EXTENSION_DENIED: %w", err)
		result.ExecutionTime = time.Since(startTime)
		s.LogAudit("write", filePath, false, result.Error.Error())
		return result
	}

	// Check content size
	contentBytes := []byte(content)
	if int64(len(contentBytes)) > s.Config.MaxWriteSize {
		result.Success = false
		result.Error = fmt.Errorf("RESOURCE_LIMIT: content too large (%d bytes, max %d)",
			len(contentBytes), s.Config.MaxWriteSize)
		result.ExecutionTime = time.Since(startTime)
		s.LogAudit("write", filePath, false, result.Error.Error())
		return result
	}

	// Check if file exists
	var backupPath string
	fileExists := false
	if _, err := os.Stat(safePath); err == nil {
		fileExists = true
		result.Action = "UPDATED"

		// Create backup if configured
		if s.Config.BackupBeforeWrite {
			backupPath, err = s.CreateBackup(safePath)
			if err != nil {
				result.Success = false
				result.Error = fmt.Errorf("BACKUP_FAILED: %w", err)
				result.ExecutionTime = time.Since(startTime)
				s.LogAudit("write", filePath, false, result.Error.Error())
				return result
			}
			result.BackupFile = backupPath
		}
	} else {
		result.Action = "CREATED"
	}

	// Format content based on file type
	formattedContent, err := s.FormatContent(filePath, content)
	if err != nil {
		result.Success = false
		result.Error = fmt.Errorf("FORMATTING_ERROR: %w", err)
		result.ExecutionTime = time.Since(startTime)
		s.LogAudit("write", filePath, false, result.Error.Error())
		return result
	}

	// Create directory if it doesn't exist
	dir := filepath.Dir(safePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		result.Success = false
		result.Error = fmt.Errorf("DIRECTORY_CREATION_FAILED: %w", err)
		result.ExecutionTime = time.Since(startTime)
		s.LogAudit("write", filePath, false, result.Error.Error())
		return result
	}

	// Atomic write using temporary file
	tempPath := safePath + ".tmp." + strconv.FormatInt(time.Now().UnixNano(), 10)

	// Write to temp file first
	err = os.WriteFile(tempPath, []byte(formattedContent), 0644)
	if err != nil {
		result.Success = false
		result.Error = fmt.Errorf("WRITE_ERROR: %w", err)
		result.ExecutionTime = time.Since(startTime)
		s.LogAudit("write", filePath, false, result.Error.Error())
		return result
	}

	// Atomically rename temp file to target
	err = os.Rename(tempPath, safePath)
	if err != nil {
		// Clean up temp file
		os.Remove(tempPath)
		result.Success = false
		result.Error = fmt.Errorf("RENAME_ERROR: %w", err)
		result.ExecutionTime = time.Since(startTime)
		s.LogAudit("write", filePath, false, result.Error.Error())
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

	s.LogAudit("write", filePath, true, auditMsg)
	s.CommandsRun++

	return result
}

// ExecuteOpen handles the "open" command
func (s *Session) ExecuteOpen(filepath string) ExecutionResult {
	startTime := time.Now()
	result := ExecutionResult{
		Command: Command{Type: "open", Argument: filepath},
	}

	// Validate the path
	safePath, err := s.ValidatePath(filepath)
	if err != nil {
		result.Success = false
		result.Error = fmt.Errorf("PATH_SECURITY: %w", err)
		result.ExecutionTime = time.Since(startTime)
		s.LogAudit("open", filepath, false, result.Error.Error())
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
		s.LogAudit("open", filepath, false, result.Error.Error())
		return result
	}

	// Check file size
	if fileInfo.Size() > s.Config.MaxFileSize {
		result.Success = false
		result.Error = fmt.Errorf("RESOURCE_LIMIT: file too large (%d bytes, max %d)",
			fileInfo.Size(), s.Config.MaxFileSize)
		result.ExecutionTime = time.Since(startTime)
		s.LogAudit("open", filepath, false, result.Error.Error())
		return result
	}

	// Read the file
	content, err := os.ReadFile(safePath)
	if err != nil {
		result.Success = false
		result.Error = fmt.Errorf("READ_ERROR: %w", err)
		result.ExecutionTime = time.Since(startTime)
		s.LogAudit("open", filepath, false, result.Error.Error())
		return result
	}

	result.Success = true
	result.Result = string(content)
	result.ExecutionTime = time.Since(startTime)
	s.LogAudit("open", filepath, true, "")
	s.CommandsRun++

	return result
}

// LogAudit writes an audit log entry
func (s *Session) LogAudit(command, argument string, success bool, errorMsg string) {
	if s.AuditLogger == nil {
		return
	}

	status := "success"
	if !success {
		status = "failed"
	}

	logEntry := fmt.Sprintf("%s|session:%s|%s|%s|%s|%s",
		time.Now().Format(time.RFC3339),
		s.ID,
		command,
		argument,
		status,
		errorMsg,
	)

	s.AuditLogger.Println(logEntry)
}

// ProcessText processes LLM output and executes commands
func (s *Session) ProcessText(text string) string {
	var output strings.Builder

	// Parse commands from the text
	commands := ParseCommands(text)

	if len(commands) == 0 {
		// No commands found, return original text
		return text
	}

	output.WriteString("=== LLM TOOL START ===\n")

	// Process text and commands in order
	lastPos := 0
	for _, cmd := range commands {
		// Write text before command
		if cmd.StartPos > lastPos {
			output.WriteString(text[lastPos:cmd.StartPos])
		}

		// Write original command
		output.WriteString(cmd.Original)
		output.WriteString("\n")

		// Execute command based on type
		var result ExecutionResult
		switch cmd.Type {
		case "open":
			result = s.ExecuteOpen(cmd.Argument)
		case "write":
			result = s.ExecuteWrite(cmd.Argument, cmd.Content)
		case "exec":
			result = s.ExecuteExec(cmd.Argument)
		default:
			result = ExecutionResult{
				Command: cmd,
				Success: false,
				Error:   fmt.Errorf("UNKNOWN_COMMAND: %s", cmd.Type),
			}
		}

		// Format and write result
		output.WriteString(fmt.Sprintf("=== COMMAND: %s ===\n", cmd.Original))
		if result.Success {
			if cmd.Type == "open" {
				output.WriteString(fmt.Sprintf("=== FILE: %s ===\n", cmd.Argument))
				output.WriteString(result.Result)
				if !strings.HasSuffix(result.Result, "\n") {
					output.WriteString("\n")
				}
				output.WriteString("=== END FILE ===\n")
			} else if cmd.Type == "write" {
				output.WriteString(fmt.Sprintf("=== WRITE SUCCESSFUL: %s ===\n", cmd.Argument))
				output.WriteString(fmt.Sprintf("Action: %s\n", result.Action))
				output.WriteString(fmt.Sprintf("Bytes written: %d\n", result.BytesWritten))
				if result.BackupFile != "" {
					output.WriteString(fmt.Sprintf("Backup: %s\n", filepath.Base(result.BackupFile)))
				}
				output.WriteString("=== END WRITE ===\n")
			} else if cmd.Type == "exec" {
				output.WriteString(fmt.Sprintf("=== EXEC SUCCESSFUL: %s ===\n", cmd.Argument))
				output.WriteString(fmt.Sprintf("Exit code: %d\n", result.ExitCode))
				output.WriteString(fmt.Sprintf("Duration: %.3fs\n", result.ExecutionTime.Seconds()))
				if result.Result != "" {
					output.WriteString("Output:\n")
					output.WriteString(result.Result)
					if !strings.HasSuffix(result.Result, "\n") {
						output.WriteString("\n")
					}
				}
				output.WriteString("=== END EXEC ===\n")
			}
		} else {
			output.WriteString(fmt.Sprintf("=== ERROR: %s ===\n", strings.Split(result.Error.Error(), ":")[0]))
			output.WriteString(fmt.Sprintf("Message: %s\n", result.Error.Error()))
			output.WriteString(fmt.Sprintf("Command: %s\n", cmd.Original))
			if cmd.Type == "exec" && result.ExitCode != 0 {
				output.WriteString(fmt.Sprintf("Exit code: %d\n", result.ExitCode))
				if result.Stderr != "" {
					output.WriteString(fmt.Sprintf("Stderr: %s\n", result.Stderr))
				}
			}
			output.WriteString("=== END ERROR ===\n")
		}
		output.WriteString("=== END COMMAND ===\n")

		lastPos = cmd.EndPos
	}

	// Write remaining text
	if lastPos < len(text) {
		output.WriteString(text[lastPos:])
	}

	// Write summary
	output.WriteString("\n=== LLM TOOL COMPLETE ===\n")
	output.WriteString(fmt.Sprintf("Commands executed: %d\n", s.CommandsRun))
	output.WriteString(fmt.Sprintf("Time elapsed: %.2fs\n", time.Since(s.StartTime).Seconds()))
	output.WriteString("=== END ===\n")

	return output.String()
}

// InteractiveMode handles continuous input/output
func (s *Session) InteractiveMode() {
	scanner := bufio.NewScanner(os.Stdin)
	var buffer strings.Builder

	fmt.Fprintln(os.Stderr, "LLM Tool - Interactive Mode")
	fmt.Fprintln(os.Stderr, "Waiting for input (send EOF with Ctrl+D to process)...")
	fmt.Fprintln(os.Stderr, "Supports commands: <open filepath>, <write filepath>content</write>, <exec command args>")

	for scanner.Scan() {
		line := scanner.Text()
		buffer.WriteString(line)
		buffer.WriteString("\n")

		// Check if line contains a command
		if strings.Contains(line, "<open") || strings.Contains(line, "<write") || strings.Contains(line, "<exec") {
			// Process accumulated text
			result := s.ProcessText(buffer.String())
			fmt.Print(result)
			buffer.Reset()

			fmt.Fprintln(os.Stderr, "\nWaiting for more input...")
		}
	}

	// Process any remaining text
	if buffer.Len() > 0 {
		result := s.ProcessText(buffer.String())
		fmt.Print(result)
	}
}

func main() {
	// Parse command-line flags
	var config Config

	flag.StringVar(&config.RepositoryRoot, "root", ".", "Repository root directory")
	flag.Int64Var(&config.MaxFileSize, "max-size", 1048576, "Maximum file size in bytes (default 1MB)")
	flag.BoolVar(&config.Interactive, "interactive", false, "Run in interactive mode")
	flag.StringVar(&config.InputFile, "input", "", "Input file (default: stdin)")
	flag.StringVar(&config.OutputFile, "output", "", "Output file (default: stdout)")
	flag.BoolVar(&config.JSONOutput, "json", false, "Output in JSON format")
	flag.BoolVar(&config.Verbose, "verbose", false, "Verbose output")
	flag.Int64Var(&config.MaxWriteSize, "max-write-size", 102400, "Maximum file size in bytes for writing (default 100KB)")
	flag.BoolVar(&config.RequireConfirmation, "require-confirmation", false, "Require confirmation for write operations")
	flag.BoolVar(&config.BackupBeforeWrite, "backup", true, "Create backup before overwriting files")
	flag.BoolVar(&config.ForceWrite, "force", false, "Force write even if conflicts exist")

	flag.BoolVar(&config.ExecEnabled, "exec-enabled", false, "Enable exec command")
	execTimeoutStr := flag.String("exec-timeout", "30s", "Timeout for exec commands")
	flag.StringVar(&config.ExecMemoryLimit, "exec-memory", "512m", "Memory limit for containers")
	flag.IntVar(&config.ExecCPULimit, "exec-cpu", 2, "CPU limit for containers")
	flag.StringVar(&config.ExecContainerImage, "exec-image", "ubuntu:22.04", "Docker image for exec commands")
	flag.BoolVar(&config.ExecNetworkEnabled, "exec-network", false, "Enable network access in containers")

	allowedExts := flag.String("allowed-extensions", ".go,.py,.js,.md,.txt,.json,.yaml,.yml,.toml",
		"Comma-separated list of allowed file extensions for writing")
	execWhitelistStr := flag.String("exec-whitelist", "go test,go build,go run,npm test,npm run build,python -m pytest,make,cargo build,cargo test",
		"Comma-separated list of allowed exec commands")

	// Parse excluded paths
	excludedPaths := flag.String("exclude", ".git,.env,*.key,*.pem", "Comma-separated list of excluded paths")

	flag.Parse()

	// Parse timeout
	var err error
	config.ExecTimeout, err = time.ParseDuration(*execTimeoutStr)
	if err != nil {
		log.Fatalf("Invalid exec timeout: %v", err)
	}

	// Set up allowed extensions
	if *allowedExts != "" {
		config.AllowedExtensions = strings.Split(*allowedExts, ",")
		for i := range config.AllowedExtensions {
			config.AllowedExtensions[i] = strings.TrimSpace(config.AllowedExtensions[i])
		}
	}

	// Set up exec whitelist
	if *execWhitelistStr != "" {
		config.ExecWhitelist = strings.Split(*execWhitelistStr, ",")
		for i := range config.ExecWhitelist {
			config.ExecWhitelist[i] = strings.TrimSpace(config.ExecWhitelist[i])
		}
	}

	// Set up excluded paths
	config.ExcludedPaths = strings.Split(*excludedPaths, ",")
	for i := range config.ExcludedPaths {
		config.ExcludedPaths[i] = strings.TrimSpace(config.ExcludedPaths[i])
	}

	if config.Verbose {
		fmt.Fprintf(os.Stderr, "Max write file size: %d bytes\n", config.MaxWriteSize)
		fmt.Fprintf(os.Stderr, "Allowed extensions: %v\n", config.AllowedExtensions)
		fmt.Fprintf(os.Stderr, "Backup enabled: %v\n", config.BackupBeforeWrite)
		fmt.Fprintf(os.Stderr, "Exec enabled: %v\n", config.ExecEnabled)
		if config.ExecEnabled {
			fmt.Fprintf(os.Stderr, "Exec whitelist: %v\n", config.ExecWhitelist)
			fmt.Fprintf(os.Stderr, "Exec timeout: %v\n", config.ExecTimeout)
			fmt.Fprintf(os.Stderr, "Exec image: %s\n", config.ExecContainerImage)
		}
	}

	// Resolve repository root to absolute path
	absRoot, err := filepath.Abs(config.RepositoryRoot)
	if err != nil {
		log.Fatalf("Cannot resolve repository root: %v", err)
	}
	config.RepositoryRoot = absRoot

	// Verify repository root exists
	if _, err := os.Stat(config.RepositoryRoot); err != nil {
		log.Fatalf("Repository root does not exist: %v", err)
	}

	// Create session
	session := NewSession(&config)

	if config.Verbose {
		fmt.Fprintf(os.Stderr, "Repository root: %s\n", config.RepositoryRoot)
		fmt.Fprintf(os.Stderr, "Max file size: %d bytes\n", config.MaxFileSize)
		fmt.Fprintf(os.Stderr, "Excluded paths: %v\n", config.ExcludedPaths)
	}

	// Handle different modes
	if config.Interactive {
		session.InteractiveMode()
	} else {
		// Read input
		var input []byte
		if config.InputFile != "" {
			data, err := os.ReadFile(config.InputFile)
			if err != nil {
				log.Fatalf("Cannot read input file: %v", err)
			}
			input = data
		} else {
			data, err := io.ReadAll(os.Stdin)
			if err != nil {
				log.Fatalf("Cannot read stdin: %v", err)
			}
			input = data
		}

		// Process text
		result := session.ProcessText(string(input))

		// Write output
		if config.OutputFile != "" {
			err := os.WriteFile(config.OutputFile, []byte(result), 0644)
			if err != nil {
				log.Fatalf("Cannot write output file: %v", err)
			}
		} else {
			fmt.Print(result)
		}
	}
}

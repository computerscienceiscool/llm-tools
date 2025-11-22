package main

import (
	"bufio"
	"crypto/sha256"
	"encoding/json"
	"flag"
	"fmt"
	"go/format"
	"io"
	"log"
	"os"
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
			}
		} else {
			output.WriteString(fmt.Sprintf("=== ERROR: %s ===\n", strings.Split(result.Error.Error(), ":")[0]))
			output.WriteString(fmt.Sprintf("Message: %s\n", result.Error.Error()))
			output.WriteString(fmt.Sprintf("Command: %s\n", cmd.Original))
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
	fmt.Fprintln(os.Stderr, "Supports commands: <open filepath>, <write filepath>content</write>")

	for scanner.Scan() {
		line := scanner.Text()
		buffer.WriteString(line)
		buffer.WriteString("\n")

		// Check if line contains a command
		if strings.Contains(line, "<open") || strings.Contains(line, "<write") {
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
	allowedExts := flag.String("allowed-extensions", ".go,.py,.js,.md,.txt,.json,.yaml,.yml,.toml",
		"Comma-separated list of allowed file extensions for writing")

	// Parse excluded paths
	excludedPaths := flag.String("exclude", ".git,.env,*.key,*.pem", "Comma-separated list of excluded paths")

	flag.Parse()
	// Set up allowed extensions
	if *allowedExts != "" {
		config.AllowedExtensions = strings.Split(*allowedExts, ",")
		for i := range config.AllowedExtensions {
			config.AllowedExtensions[i] = strings.TrimSpace(config.AllowedExtensions[i])
		}
	}

	// Set up excluded paths
	config.ExcludedPaths = strings.Split(*excludedPaths, ",")
	for i := range config.ExcludedPaths {
		config.ExcludedPaths[i] = strings.TrimSpace(config.ExcludedPaths[i])
	}
	fmt.Fprintf(os.Stderr, "Max write file size: %d bytes\n", config.MaxWriteSize)
	fmt.Fprintf(os.Stderr, "Allowed extensions: %v\n", config.AllowedExtensions)
	fmt.Fprintf(os.Stderr, "Backup enabled: %v\n", config.BackupBeforeWrite)

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

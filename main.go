package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// Config holds the tool configuration
type Config struct {
	RepositoryRoot string
	MaxFileSize    int64
	ExcludedPaths  []string
	Interactive    bool
	InputFile      string
	OutputFile     string
	JSONOutput     bool
	Verbose        bool
}

// Command represents a parsed command from LLM output
type Command struct {
	Type      string
	Argument  string
	StartPos  int
	EndPos    int
	Original  string
}

// ExecutionResult holds the result of a command execution
type ExecutionResult struct {
	Command       Command
	Success       bool
	Result        string
	Error         error
	ExecutionTime time.Duration
}

// Session manages a tool execution session
type Session struct {
	ID             string
	Config         *Config
	CommandsRun    int
	StartTime      time.Time
	AuditLogger    *log.Logger
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
		
		// Execute command
		result := s.ExecuteOpen(cmd.Argument)
		
		// Format and write result
		output.WriteString(fmt.Sprintf("=== COMMAND: %s ===\n", cmd.Original))
		if result.Success {
			output.WriteString(fmt.Sprintf("=== FILE: %s ===\n", cmd.Argument))
			output.WriteString(result.Result)
			if !strings.HasSuffix(result.Result, "\n") {
				output.WriteString("\n")
			}
			output.WriteString("=== END FILE ===\n")
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
	
	for scanner.Scan() {
		line := scanner.Text()
		buffer.WriteString(line)
		buffer.WriteString("\n")
		
		// Check if line contains a command
		if strings.Contains(line, "<open") {
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
	
	// Parse excluded paths
	excludedPaths := flag.String("exclude", ".git,.env,*.key,*.pem", "Comma-separated list of excluded paths")
	
	flag.Parse()
	
	// Set up excluded paths
	config.ExcludedPaths = strings.Split(*excludedPaths, ",")
	for i := range config.ExcludedPaths {
		config.ExcludedPaths[i] = strings.TrimSpace(config.ExcludedPaths[i])
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

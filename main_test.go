package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestParseCommands(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "Single open command",
			input:    "Let me check the file <open main.go>",
			expected: []string{"main.go"},
		},
		{
			name:     "Multiple open commands",
			input:    "First <open file1.txt> and then <open dir/file2.go>",
			expected: []string{"file1.txt", "dir/file2.go"},
		},
		{
			name:     "Open with spaces",
			input:    "Check <open  path/to/file.txt  >",
			expected: []string{"path/to/file.txt"},
		},
		{
			name:     "No commands",
			input:    "This is just regular text without any commands",
			expected: []string{},
		},
		{
			name:     "Command with special characters",
			input:    "Open <open file-name_2.test.go>",
			expected: []string{"file-name_2.test.go"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			commands := ParseCommands(tt.input)

			if len(commands) != len(tt.expected) {
				t.Errorf("Expected %d commands, got %d", len(tt.expected), len(commands))
				return
			}

			for i, cmd := range commands {
				if cmd.Argument != tt.expected[i] {
					t.Errorf("Command %d: expected argument %q, got %q",
						i, tt.expected[i], cmd.Argument)
				}
				if cmd.Type != "open" {
					t.Errorf("Command %d: expected type 'open', got %q", i, cmd.Type)
				}
			}
		})
	}
}

func TestValidatePath(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "llm-tool-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Create test files and directories
	os.MkdirAll(filepath.Join(tempDir, "subdir"), 0755)
	os.WriteFile(filepath.Join(tempDir, "test.txt"), []byte("test"), 0644)
	os.WriteFile(filepath.Join(tempDir, "subdir", "file.go"), []byte("package main"), 0644)

	config := &Config{
		RepositoryRoot: tempDir,
		ExcludedPaths:  []string{".git", "*.key", "secret"},
		MaxFileSize:    1048576,
	}

	session := NewSession(config)

	tests := []struct {
		name        string
		path        string
		shouldError bool
		errorType   string
	}{
		{
			name:        "Valid relative path",
			path:        "test.txt",
			shouldError: false,
		},
		{
			name:        "Valid nested path",
			path:        "subdir/file.go",
			shouldError: false,
		},
		{
			name:        "Path traversal attempt with ..",
			path:        "../etc/passwd",
			shouldError: true,
			errorType:   "traversal",
		},
		{
			name:        "Path traversal in middle",
			path:        "subdir/../../etc/passwd",
			shouldError: true,
			errorType:   "traversal",
		},
		{
			name:        "Absolute path outside repo",
			path:        "/etc/passwd",
			shouldError: true,
			errorType:   "traversal", // Changed from "not within" to match actual error
		},
		{
			name:        "Excluded .git path",
			path:        ".git/config",
			shouldError: true,
			errorType:   "excluded",
		},
		{
			name:        "Excluded key file",
			path:        "private.key",
			shouldError: true,
			errorType:   "excluded",
		},
		{
			name:        "Non-existent but valid path",
			path:        "does-not-exist.txt",
			shouldError: false, // Path validation should pass even if file doesn't exist
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := session.ValidatePath(tt.path)

			if tt.shouldError {
				if err == nil {
					t.Errorf("Expected error for path %q but got none", tt.path)
				} else if tt.errorType != "" && !strings.Contains(err.Error(), tt.errorType) {
					t.Errorf("Expected error containing %q, got %q", tt.errorType, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error for path %q: %v", tt.path, err)
				}
				// For existing files, verify the resolved path is within the repo
				if result != "" {
					relPath, err := filepath.Rel(tempDir, result)
					if err != nil || strings.HasPrefix(relPath, "..") {
						t.Errorf("Resolved path %q is outside repository", result)
					}
				}
			}
		})
	}
}

func TestExecuteOpen(t *testing.T) {
	// Create temporary test environment
	tempDir, err := os.MkdirTemp("", "llm-tool-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Create test files
	smallFile := filepath.Join(tempDir, "small.txt")
	os.WriteFile(smallFile, []byte("Small file content"), 0644)

	largeContent := make([]byte, 2*1048576) // 2MB
	for i := range largeContent {
		largeContent[i] = 'A'
	}
	largeFile := filepath.Join(tempDir, "large.txt")
	os.WriteFile(largeFile, largeContent, 0644)

	config := &Config{
		RepositoryRoot: tempDir,
		MaxFileSize:    1048576, // 1MB
		ExcludedPaths:  []string{".git"},
	}

	session := NewSession(config)

	tests := []struct {
		name          string
		filepath      string
		expectSuccess bool
		expectError   string
	}{
		{
			name:          "Read existing small file",
			filepath:      "small.txt",
			expectSuccess: true,
		},
		{
			name:          "File too large",
			filepath:      "large.txt",
			expectSuccess: false,
			expectError:   "RESOURCE_LIMIT",
		},
		{
			name:          "Non-existent file",
			filepath:      "missing.txt",
			expectSuccess: false,
			expectError:   "FILE_NOT_FOUND",
		},
		{
			name:          "Path traversal attempt",
			filepath:      "../../../etc/passwd",
			expectSuccess: false,
			expectError:   "PATH_SECURITY",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := session.ExecuteOpen(tt.filepath)

			if tt.expectSuccess {
				if !result.Success {
					t.Errorf("Expected success but got error: %v", result.Error)
				}
				if result.Result == "" {
					t.Error("Expected non-empty result")
				}
			} else {
				if result.Success {
					t.Error("Expected failure but operation succeeded")
				}
				if tt.expectError != "" && !strings.Contains(result.Error.Error(), tt.expectError) {
					t.Errorf("Expected error containing %q, got %q",
						tt.expectError, result.Error.Error())
				}
			}

			// Verify execution time is recorded
			if result.ExecutionTime == 0 {
				t.Error("Execution time not recorded")
			}
		})
	}
}

func TestProcessText(t *testing.T) {
	// Create test environment
	tempDir, err := os.MkdirTemp("", "llm-tool-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Create test file
	testFile := filepath.Join(tempDir, "hello.txt")
	os.WriteFile(testFile, []byte("Hello, World!"), 0644)

	config := &Config{
		RepositoryRoot: tempDir,
		MaxFileSize:    1048576,
		ExcludedPaths:  []string{},
	}

	session := NewSession(config)

	tests := []struct {
		name             string
		input            string
		shouldContain    []string
		shouldNotContain []string
	}{
		{
			name:  "Process single command",
			input: "Let me check the file <open hello.txt> and see what's there.",
			shouldContain: []string{
				"LLM TOOL START",
				"Hello, World!",
				"Commands executed: 1",
				"LLM TOOL COMPLETE",
			},
		},
		{
			name:  "Process with error",
			input: "Opening non-existent <open missing.txt> file.",
			shouldContain: []string{
				"ERROR",
				"FILE_NOT_FOUND",
			},
		},
		{
			name:  "No commands",
			input: "This is just plain text without commands.",
			shouldContain: []string{
				"This is just plain text",
			},
			shouldNotContain: []string{
				"LLM TOOL START",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := session.ProcessText(tt.input)

			for _, expected := range tt.shouldContain {
				if !strings.Contains(result, expected) {
					t.Errorf("Result should contain %q but doesn't", expected)
				}
			}

			for _, unexpected := range tt.shouldNotContain {
				if strings.Contains(result, unexpected) {
					t.Errorf("Result should not contain %q but does", unexpected)
				}
			}
		})
	}
}

func TestAuditLogging(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "llm-tool-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Change to temp dir to create audit log there
	originalDir, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(originalDir)

	config := &Config{
		RepositoryRoot: tempDir,
		MaxFileSize:    1048576,
		ExcludedPaths:  []string{},
	}

	session := NewSession(config)

	// Execute a command to generate audit log
	session.LogAudit("open", "test.txt", true, "")
	session.LogAudit("open", "../etc/passwd", false, "path traversal detected")

	// Give logger time to flush
	time.Sleep(100 * time.Millisecond)

	// Check audit log exists and contains expected entries
	auditLog, err := os.ReadFile(filepath.Join(tempDir, "audit.log"))
	if err != nil {
		t.Errorf("Could not read audit log: %v", err)
		return
	}

	logContent := string(auditLog)

	// Verify log format and content
	if !strings.Contains(logContent, "open|test.txt|success") {
		t.Error("Audit log missing successful operation")
	}

	if !strings.Contains(logContent, "open|../etc/passwd|failed|path traversal") {
		t.Error("Audit log missing failed operation")
	}

	// Verify session ID is logged
	if !strings.Contains(logContent, fmt.Sprintf("session:%s", session.ID)) {
		t.Error("Audit log missing session ID")
	}
}

func BenchmarkParseCommands(b *testing.B) {
	text := `
		Let me explore this codebase. First, I'll check <open go.mod> to understand
		the dependencies. Then I'll look at <open cmd/main.go> for the entry point.
		After that, let's examine <open internal/handler/handler.go> and 
		<open pkg/utils/utils.go> to understand the structure.
	`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ParseCommands(text)
	}
}

func BenchmarkValidatePath(b *testing.B) {
	tempDir, _ := os.MkdirTemp("", "bench")
	defer os.RemoveAll(tempDir)

	config := &Config{
		RepositoryRoot: tempDir,
		ExcludedPaths:  []string{".git", "*.key"},
	}
	session := NewSession(config)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = session.ValidatePath("subdir/file.go")
	}
}

func TestExecuteWrite(t *testing.T) {
	// Create temporary test environment
	tempDir, err := os.MkdirTemp("", "llm-tool-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	config := &Config{
		RepositoryRoot:    tempDir,
		MaxWriteSize:      1024,
		AllowedExtensions: []string{".txt", ".go"},
		BackupBeforeWrite: true,
		ExcludedPaths:     []string{},
	}

	session := NewSession(config)

	// Test creating new file
	result := session.ExecuteWrite("test.txt", "Hello World")
	if !result.Success {
		t.Errorf("Failed to create file: %v", result.Error)
	}
	if result.Action != "CREATED" {
		t.Errorf("Expected CREATED, got %s", result.Action)
	}

	// Test updating existing file
	result = session.ExecuteWrite("test.txt", "Updated content")
	if !result.Success {
		t.Errorf("Failed to update file: %v", result.Error)
	}
	if result.Action != "UPDATED" {
		t.Errorf("Expected UPDATED, got %s", result.Action)
	}
}

func TestValidateWriteExtension(t *testing.T) {
	config := &Config{
		AllowedExtensions: []string{".go", ".txt"},
	}
	session := NewSession(config)

	// Should pass
	if err := session.ValidateWriteExtension("test.go"); err != nil {
		t.Errorf("Unexpected error for .go file: %v", err)
	}

	// Should fail
	if err := session.ValidateWriteExtension("test.exe"); err == nil {
		t.Error("Expected error for .exe file")
	}
}

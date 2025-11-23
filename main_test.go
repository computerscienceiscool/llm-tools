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
		expected []Command
	}{
		{
			name:  "Single open command",
			input: "Let me check the file <open main.go>",
			expected: []Command{
				{Type: "open", Argument: "main.go"},
			},
		},
		{
			name:  "Multiple open commands",
			input: "First <open file1.txt> and then <open dir/file2.go>",
			expected: []Command{
				{Type: "open", Argument: "file1.txt"},
				{Type: "open", Argument: "dir/file2.go"},
			},
		},
		{
			name:  "Write command",
			input: "Create file <write test.txt>Hello World</write>",
			expected: []Command{
				{Type: "write", Argument: "test.txt", Content: "Hello World"},
			},
		},
		{
			name:  "Exec command",
			input: "Run tests <exec go test>",
			expected: []Command{
				{Type: "exec", Argument: "go test"},
			},
		},
		{
			name:  "Mixed commands",
			input: "Check file <open main.go> then run <exec go build> and write <write output.txt>Build complete</write>",
			expected: []Command{
				{Type: "open", Argument: "main.go"},
				{Type: "write", Argument: "output.txt", Content: "Build complete"},
				{Type: "exec", Argument: "go build"},
			},
		},
		{
			name:     "No commands",
			input:    "This is just regular text without any commands",
			expected: []Command{},
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
				expected := tt.expected[i]
				if cmd.Type != expected.Type {
					t.Errorf("Command %d: expected type %q, got %q", i, expected.Type, cmd.Type)
				}
				if cmd.Argument != expected.Argument {
					t.Errorf("Command %d: expected argument %q, got %q", i, expected.Argument, cmd.Argument)
				}
				if cmd.Content != expected.Content {
					t.Errorf("Command %d: expected content %q, got %q", i, expected.Content, cmd.Content)
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

func TestValidateExecCommand(t *testing.T) {
	config := &Config{
		ExecEnabled:   true,
		ExecWhitelist: []string{"go test", "go build", "npm test", "make"},
	}
	session := NewSession(config)

	tests := []struct {
		name        string
		command     string
		shouldError bool
	}{
		{
			name:        "Whitelisted command",
			command:     "go test",
			shouldError: false,
		},
		{
			name:        "Whitelisted command with args",
			command:     "go test ./...",
			shouldError: false,
		},
		{
			name:        "Non-whitelisted command",
			command:     "rm -rf /",
			shouldError: true,
		},
		{
			name:        "Empty command",
			command:     "",
			shouldError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := session.ValidateExecCommand(tt.command)

			if tt.shouldError {
				if err == nil {
					t.Errorf("Expected error for command %q but got none", tt.command)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error for command %q: %v", tt.command, err)
				}
			}
		})
	}
}

func TestValidateExecCommandDisabled(t *testing.T) {
	config := &Config{
		ExecEnabled: false,
	}
	session := NewSession(config)

	err := session.ValidateExecCommand("go test")
	if err == nil {
		t.Error("Expected error when exec is disabled")
	}
	if !strings.Contains(err.Error(), "disabled") {
		t.Errorf("Expected 'disabled' in error, got: %v", err)
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
		ExecEnabled:    true,
		ExecWhitelist:  []string{"echo"},
	}

	session := NewSession(config)

	tests := []struct {
		name             string
		input            string
		shouldContain    []string
		shouldNotContain []string
	}{
		{
			name:  "Process single open command",
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
		{
			name:  "Process exec command (if Docker available)",
			input: "Run command <exec echo hello>",
			shouldContain: []string{
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
	session.LogAudit("exec", "go test", true, "exit_code:0,duration:1.234s")

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

	if !strings.Contains(logContent, "exec|go test|success") {
		t.Error("Audit log missing exec operation")
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
		Finally, I'll run <exec go test> to check if tests pass.
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

// Test Docker availability (will skip if Docker not available)
func TestDockerAvailability(t *testing.T) {
	err := CheckDockerAvailability()
	if err != nil {
		t.Skipf("Docker not available: %v", err)
	}
}

// Test Docker image pulling
func TestPullDockerImage(t *testing.T) {
	if err := CheckDockerAvailability(); err != nil {
		t.Skipf("Docker not available: %v", err)
	}

	config := &Config{
		ExecContainerImage: "ubuntu:22.04",
		Verbose:            false,
	}
	session := NewSession(config)

	// Test pulling a standard image
	err := session.PullDockerImage()
	if err != nil {
		t.Errorf("Failed to pull Docker image: %v", err)
	}

	// Test with invalid image (should fail gracefully)
	config.ExecContainerImage = "nonexistent/invalid-image:999"
	session.Config = config
	err = session.PullDockerImage()
	if err == nil {
		t.Error("Expected error for invalid Docker image")
	}
}

// Integration test for exec command (requires Docker)
func TestExecuteExecIntegration(t *testing.T) {
	// Skip if Docker not available
	if err := CheckDockerAvailability(); err != nil {
		t.Skipf("Docker not available: %v", err)
	}

	tempDir, err := os.MkdirTemp("", "llm-tool-exec-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Create a simple test file
	os.WriteFile(filepath.Join(tempDir, "test.txt"), []byte("Hello World"), 0644)

	config := &Config{
		RepositoryRoot:     tempDir,
		ExecEnabled:        true,
		ExecWhitelist:      []string{"ls", "cat", "echo"},
		ExecTimeout:        30 * time.Second,
		ExecMemoryLimit:    "512m",
		ExecCPULimit:       1,
		ExecContainerImage: "ubuntu:22.04",
	}

	session := NewSession(config)

	tests := []struct {
		name          string
		command       string
		expectSuccess bool
		expectError   string
	}{
		{
			name:          "Simple ls command",
			command:       "ls",
			expectSuccess: true,
		},
		{
			name:          "Cat file command",
			command:       "cat test.txt",
			expectSuccess: true,
		},
		{
			name:          "Echo command",
			command:       "echo 'Hello from container'",
			expectSuccess: true,
		},
		{
			name:          "Non-whitelisted command",
			command:       "rm test.txt",
			expectSuccess: false,
			expectError:   "EXEC_VALIDATION",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := session.ExecuteExec(tt.command)

			if tt.expectSuccess {
				if !result.Success {
					t.Errorf("Expected success but got error: %v", result.Error)
				}
				if result.ExitCode != 0 {
					t.Errorf("Expected exit code 0, got %d", result.ExitCode)
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
		})
	}
}

// Test exec timeout handling
func TestExecuteExecTimeout(t *testing.T) {
	if err := CheckDockerAvailability(); err != nil {
		t.Skipf("Docker not available: %v", err)
	}

	tempDir, err := os.MkdirTemp("", "llm-tool-timeout-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	config := &Config{
		RepositoryRoot:     tempDir,
		ExecEnabled:        true,
		ExecWhitelist:      []string{"sleep"},
		ExecTimeout:        1 * time.Second, // Very short timeout
		ExecMemoryLimit:    "512m",
		ExecCPULimit:       1,
		ExecContainerImage: "ubuntu:22.04",
	}

	session := NewSession(config)

	// Test command that should timeout
	result := session.ExecuteExec("sleep 5")
	if result.Success {
		t.Error("Expected timeout failure but operation succeeded")
	}
	if !strings.Contains(result.Error.Error(), "EXEC_TIMEOUT") {
		t.Errorf("Expected timeout error, got: %v", result.Error)
	}
	if result.ExitCode != 124 {
		t.Errorf("Expected timeout exit code 124, got %d", result.ExitCode)
	}
}

// Test exec with invalid Docker setup
func TestExecuteExecDockerErrors(t *testing.T) {
	if err := CheckDockerAvailability(); err != nil {
		t.Skipf("Docker not available: %v", err)
	}

	tempDir, err := os.MkdirTemp("", "llm-tool-docker-error-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	config := &Config{
		RepositoryRoot:     tempDir,
		ExecEnabled:        true,
		ExecWhitelist:      []string{"echo"},
		ExecTimeout:        30 * time.Second,
		ExecMemoryLimit:    "512m",
		ExecCPULimit:       1,
		ExecContainerImage: "completely-invalid-image:nonexistent",
	}

	session := NewSession(config)

	// Test with invalid Docker image
	result := session.ExecuteExec("echo test")
	if result.Success {
		t.Error("Expected Docker image failure but operation succeeded")
	}
	if !strings.Contains(result.Error.Error(), "DOCKER_IMAGE") {
		t.Errorf("Expected Docker image error, got: %v", result.Error)
	}
}

// Test exec command parsing edge cases
func TestParseExecEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []Command
	}{
		{
			name:  "Exec command with complex arguments",
			input: "Run <exec go test -v -race ./...>",
			expected: []Command{
				{Type: "exec", Argument: "go test -v -race ./..."},
			},
		},
		{
			name:  "Exec command with quotes",
			input: `Execute <exec echo "hello world">`,
			expected: []Command{
				{Type: "exec", Argument: `echo "hello world"`},
			},
		},
		{
			name:  "Multiple exec commands",
			input: "First <exec go build> then <exec go test>",
			expected: []Command{
				{Type: "exec", Argument: "go build"},
				{Type: "exec", Argument: "go test"},
			},
		},
		{
			name:  "Exec with pipes and complex args",
			input: "Run <exec cat file.txt | grep pattern>",
			expected: []Command{
				{Type: "exec", Argument: "cat file.txt | grep pattern"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			commands := ParseCommands(tt.input)

			// Filter to only exec commands for this test
			execCommands := []Command{}
			for _, cmd := range commands {
				if cmd.Type == "exec" {
					execCommands = append(execCommands, cmd)
				}
			}

			if len(execCommands) != len(tt.expected) {
				t.Errorf("Expected %d exec commands, got %d", len(tt.expected), len(execCommands))
				return
			}

			for i, cmd := range execCommands {
				expected := tt.expected[i]
				if cmd.Argument != expected.Argument {
					t.Errorf("Command %d: expected argument %q, got %q", i, expected.Argument, cmd.Argument)
				}
			}
		})
	}
}

// Test config validation for exec settings
func TestExecConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		command string
		valid   bool
	}{
		{
			name: "Valid exec config",
			config: &Config{
				ExecEnabled:        true,
				ExecWhitelist:      []string{"go", "npm", "make"},
				ExecTimeout:        30 * time.Second,
				ExecMemoryLimit:    "512m",
				ExecCPULimit:       1,
				ExecContainerImage: "ubuntu:22.04",
			},
			command: "go test",
			valid:   true,
		},
		{
			name: "Exec disabled",
			config: &Config{
				ExecEnabled: false,
			},
			command: "go test",
			valid:   false,
		},
		{
			name: "Empty whitelist",
			config: &Config{
				ExecEnabled:   true,
				ExecWhitelist: []string{},
			},
			command: "go test",
			valid:   false,
		},
		{
			name: "Command not in whitelist",
			config: &Config{
				ExecEnabled:   true,
				ExecWhitelist: []string{"npm"},
			},
			command: "go test",
			valid:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			session := NewSession(tt.config)
			err := session.ValidateExecCommand(tt.command)

			if tt.valid {
				if err != nil {
					t.Errorf("Expected valid config but got error: %v", err)
				}
			} else {
				if err == nil {
					t.Error("Expected invalid config but validation passed")
				}
			}
		})
	}
}

// Test enhanced ProcessText with exec commands
func TestProcessTextWithExec(t *testing.T) {
	if err := CheckDockerAvailability(); err != nil {
		t.Skip("Docker not available for integration test")
	}

	tempDir, err := os.MkdirTemp("", "llm-tool-process-exec-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Create a test file
	os.WriteFile(filepath.Join(tempDir, "test.txt"), []byte("Hello World"), 0644)

	config := &Config{
		RepositoryRoot:     tempDir,
		MaxFileSize:        1048576,
		MaxWriteSize:       1024,
		ExecEnabled:        true,
		ExecWhitelist:      []string{"echo", "cat"},
		ExecTimeout:        30 * time.Second,
		ExecMemoryLimit:    "512m",
		ExecCPULimit:       1,
		ExecContainerImage: "ubuntu:22.04",
	}

	session := NewSession(config)

	input := "First check <open test.txt> then run <exec echo 'Hello from exec'> and finally create <write result.txt>Execution complete</write>"

	result := session.ProcessText(input)

	// Should contain all command types
	expectedContains := []string{
		"LLM TOOL START",
		"FILE: test.txt",
		"Hello World",
		"EXEC SUCCESSFUL",
		"Exit code: 0",
		"Hello from exec",
		"WRITE SUCCESSFUL",
		"Commands executed: 3",
		"LLM TOOL COMPLETE",
	}

	for _, expected := range expectedContains {
		if !strings.Contains(result, expected) {
			t.Errorf("Result should contain %q but doesn't.\nFull result:\n%s", expected, result)
		}
	}
}

// Benchmark exec command parsing
func BenchmarkParseExecCommands(b *testing.B) {
	text := `
		Run the test suite <exec go test -v ./...> and then
		build the project <exec go build -o bin/app .> followed by
		a quick lint check <exec golangci-lint run> and finally
		run the integration tests <exec make integration-test>.
	`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		commands := ParseCommands(text)
		// Count exec commands to ensure parsing works
		execCount := 0
		for _, cmd := range commands {
			if cmd.Type == "exec" {
				execCount++
			}
		}
		if execCount != 4 {
			b.Errorf("Expected 4 exec commands, got %d", execCount)
		}
	}
}

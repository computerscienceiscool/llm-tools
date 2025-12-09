package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/computerscienceiscool/llm-runtime/internal/config"
	"github.com/computerscienceiscool/llm-runtime/pkg/evaluator"
	"github.com/computerscienceiscool/llm-runtime/internal/search"
)

// createTestExecutor creates an executor for testing with a temp directory
func createTestExecutor(t *testing.T, repoRoot string) *evaluator.Executor {
	t.Helper()

	cfg := &config.Config{
		RepositoryRoot:    repoRoot,
		MaxFileSize:       1048576,
		MaxWriteSize:      102400,
		AllowedExtensions: []string{".go", ".txt", ".md"},
		ExcludedPaths:     []string{".git"},
		BackupBeforeWrite: false,
		ExecEnabled:       false,
	}

	searchCfg := &search.SearchConfig{
		Enabled: false,
	}

	auditLog := func(cmd, arg string, success bool, errMsg string) {
		// No-op for testing
	}

	return evaluator.NewExecutor(cfg, searchCfg, auditLog)
}

// createTestFile creates a file in the given directory
func createTestFile(t *testing.T, dir, name, content string) {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file %s: %v", path, err)
	}
}

func TestProcessText_NoCommands(t *testing.T) {
	tempDir := t.TempDir()
	exec := createTestExecutor(t, tempDir)
	startTime := time.Now()

	input := "This is just regular text with no commands."
	result := ProcessText(input, exec, startTime)

	// When no commands, should return original text
	if result != input {
		t.Errorf("ProcessText with no commands should return original text\ngot: %q\nwant: %q", result, input)
	}
}

func TestProcessText_EmptyInput(t *testing.T) {
	tempDir := t.TempDir()
	exec := createTestExecutor(t, tempDir)
	startTime := time.Now()

	input := ""
	result := ProcessText(input, exec, startTime)

	if result != input {
		t.Errorf("ProcessText with empty input should return empty string\ngot: %q\nwant: %q", result, input)
	}
}

func TestProcessText_OpenCommand_FileExists(t *testing.T) {
	tempDir := t.TempDir()

	// Create a test file
	testContent := "Hello, World!\nThis is a test file."
	testFile := "test.txt"
	createTestFile(t, tempDir, testFile, testContent)

	exec := createTestExecutor(t, tempDir)
	startTime := time.Now()
	input := "<open test.txt>"

	result := ProcessText(input, exec, startTime)

	// Check that result contains expected sections
	expectedParts := []string{
		"=== LLM TOOL START ===",
		"<open test.txt>",
		"=== COMMAND:",
		"=== FILE: test.txt ===",
		"Hello, World!",
		"This is a test file.",
		"=== END FILE ===",
		"=== END COMMAND ===",
		"=== LLM TOOL COMPLETE ===",
		"Commands executed: 1",
	}

	for _, part := range expectedParts {
		if !strings.Contains(result, part) {
			t.Errorf("Result missing expected part: %q\nFull result:\n%s", part, result)
		}
	}
}

func TestProcessText_OpenCommand_FileNotFound(t *testing.T) {
	tempDir := t.TempDir()
	exec := createTestExecutor(t, tempDir)
	startTime := time.Now()

	input := "<open nonexistent.txt>"
	result := ProcessText(input, exec, startTime)

	expectedParts := []string{
		"=== LLM TOOL START ===",
		"=== ERROR:",
		"=== END ERROR ===",
		"=== LLM TOOL COMPLETE ===",
	}

	for _, part := range expectedParts {
		if !strings.Contains(result, part) {
			t.Errorf("Result missing expected part: %q\nFull result:\n%s", part, result)
		}
	}
}

func TestProcessText_WriteCommand_CreateFile(t *testing.T) {
	tempDir := t.TempDir()
	exec := createTestExecutor(t, tempDir)
	startTime := time.Now()

	input := "<write newfile.txt>New file content</write>"
	result := ProcessText(input, exec, startTime)

	expectedParts := []string{
		"=== LLM TOOL START ===",
		"=== WRITE SUCCESSFUL: newfile.txt ===",
		"Action: CREATED",
		"Bytes written:",
		"=== END WRITE ===",
		"=== LLM TOOL COMPLETE ===",
		"Commands executed: 1",
	}

	for _, part := range expectedParts {
		if !strings.Contains(result, part) {
			t.Errorf("Result missing expected part: %q\nFull result:\n%s", part, result)
		}
	}

	// Verify file was actually created
	content, err := os.ReadFile(filepath.Join(tempDir, "newfile.txt"))
	if err != nil {
		t.Errorf("File was not created: %v", err)
	}
	if string(content) != "New file content" {
		t.Errorf("File content = %q, want %q", string(content), "New file content")
	}
}

func TestProcessText_WriteCommand_UpdateFile(t *testing.T) {
	tempDir := t.TempDir()

	// Create existing file
	testFile := "existing.txt"
	createTestFile(t, tempDir, testFile, "Original content")

	exec := createTestExecutor(t, tempDir)
	startTime := time.Now()

	input := "<write existing.txt>Updated content</write>"
	result := ProcessText(input, exec, startTime)

	expectedParts := []string{
		"=== WRITE SUCCESSFUL: existing.txt ===",
		"Action: UPDATED",
		"Commands executed: 1",
	}

	for _, part := range expectedParts {
		if !strings.Contains(result, part) {
			t.Errorf("Result missing expected part: %q\nFull result:\n%s", part, result)
		}
	}
}

func TestProcessText_WriteCommand_InvalidExtension(t *testing.T) {
	tempDir := t.TempDir()
	exec := createTestExecutor(t, tempDir)
	startTime := time.Now()

	// .exe is not in allowed extensions
	input := "<write malware.exe>bad content</write>"
	result := ProcessText(input, exec, startTime)

	expectedParts := []string{
		"=== ERROR:",
		"=== END ERROR ===",
	}

	for _, part := range expectedParts {
		if !strings.Contains(result, part) {
			t.Errorf("Result missing expected part: %q\nFull result:\n%s", part, result)
		}
	}
}

func TestProcessText_ExecCommand_Disabled(t *testing.T) {
	tempDir := t.TempDir()
	exec := createTestExecutor(t, tempDir) // ExecEnabled is false by default
	startTime := time.Now()

	input := "<exec go version>"
	result := ProcessText(input, exec, startTime)

	expectedParts := []string{
		"=== ERROR:",
		"=== END ERROR ===",
	}

	for _, part := range expectedParts {
		if !strings.Contains(result, part) {
			t.Errorf("Result missing expected part: %q\nFull result:\n%s", part, result)
		}
	}
}

func TestProcessText_SearchCommand_Disabled(t *testing.T) {
	tempDir := t.TempDir()
	exec := createTestExecutor(t, tempDir) // Search is disabled
	startTime := time.Now()

	input := "<search test query>"
	result := ProcessText(input, exec, startTime)

	expectedParts := []string{
		"=== ERROR:",
		"=== END ERROR ===",
	}

	for _, part := range expectedParts {
		if !strings.Contains(result, part) {
			t.Errorf("Result missing expected part: %q\nFull result:\n%s", part, result)
		}
	}
}

func TestProcessText_MultipleCommands(t *testing.T) {
	tempDir := t.TempDir()

	// Create a test file
	createTestFile(t, tempDir, "file1.txt", "Content of file 1")

	exec := createTestExecutor(t, tempDir)
	startTime := time.Now()

	input := "First let me read the file:\n<open file1.txt>\nNow let me create a new file:\n<write file2.txt>Content of file 2</write>"
	result := ProcessText(input, exec, startTime)

	expectedParts := []string{
		"=== LLM TOOL START ===",
		"=== FILE: file1.txt ===",
		"Content of file 1",
		"=== WRITE SUCCESSFUL: file2.txt ===",
		"Action: CREATED",
		"Commands executed: 2",
		"=== LLM TOOL COMPLETE ===",
	}

	for _, part := range expectedParts {
		if !strings.Contains(result, part) {
			t.Errorf("Result missing expected part: %q\nFull result:\n%s", part, result)
		}
	}
}

func TestProcessText_TextBetweenCommands(t *testing.T) {
	tempDir := t.TempDir()
	createTestFile(t, tempDir, "test.txt", "File content")

	exec := createTestExecutor(t, tempDir)
	startTime := time.Now()

	input := "Before command\n<open test.txt>\nAfter command"
	result := ProcessText(input, exec, startTime)

	// The surrounding text should be preserved
	if !strings.Contains(result, "Before command") {
		t.Error("Text before command should be preserved")
	}
	if !strings.Contains(result, "After command") {
		t.Error("Text after command should be preserved")
	}
}

func TestProcessText_CommandsRunCounter(t *testing.T) {
	tempDir := t.TempDir()
	createTestFile(t, tempDir, "a.txt", "A")
	createTestFile(t, tempDir, "b.txt", "B")

	exec := createTestExecutor(t, tempDir)
	startTime := time.Now()

	input := "<open a.txt>\n<open b.txt>"
	result := ProcessText(input, exec, startTime)

	if !strings.Contains(result, "Commands executed: 2") {
		t.Errorf("Expected 'Commands executed: 2' in result\nFull result:\n%s", result)
	}
}

func TestProcessText_TimeElapsed(t *testing.T) {
	tempDir := t.TempDir()
	exec := createTestExecutor(t, tempDir)
	startTime := time.Now()

	input := "<write test.txt>content</write>"
	result := ProcessText(input, exec, startTime)

	if !strings.Contains(result, "Time elapsed:") {
		t.Errorf("Expected 'Time elapsed:' in result\nFull result:\n%s", result)
	}
}

func TestProcessText_PathTraversal(t *testing.T) {
	tempDir := t.TempDir()
	exec := createTestExecutor(t, tempDir)
	startTime := time.Now()

	input := "<open ../../../etc/passwd>"
	result := ProcessText(input, exec, startTime)

	expectedParts := []string{
		"=== ERROR:",
	}

	for _, part := range expectedParts {
		if !strings.Contains(result, part) {
			t.Errorf("Result missing expected part: %q\nFull result:\n%s", part, result)
		}
	}
}

func TestProcessText_WriteWithBackup(t *testing.T) {
	tempDir := t.TempDir()

	// Create executor with backup enabled
	cfg := &config.Config{
		RepositoryRoot:    tempDir,
		MaxFileSize:       1048576,
		MaxWriteSize:      102400,
		AllowedExtensions: []string{".txt"},
		ExcludedPaths:     []string{".git"},
		BackupBeforeWrite: true,
	}

	searchCfg := &search.SearchConfig{Enabled: false}
	auditLog := func(cmd, arg string, success bool, errMsg string) {}
	exec := evaluator.NewExecutor(cfg, searchCfg, auditLog)

	// Create existing file to trigger backup
	createTestFile(t, tempDir, "backup_test.txt", "Original")

	startTime := time.Now()
	input := "<write backup_test.txt>Updated</write>"
	result := ProcessText(input, exec, startTime)

	if !strings.Contains(result, "Backup:") {
		t.Errorf("Expected 'Backup:' in result when backup is enabled\nFull result:\n%s", result)
	}
}

func TestProcessText_OpenCommandNoNewlineAtEnd(t *testing.T) {
	tempDir := t.TempDir()

	// Create file without trailing newline
	createTestFile(t, tempDir, "no_newline.txt", "No newline at end")

	exec := createTestExecutor(t, tempDir)
	startTime := time.Now()

	input := "<open no_newline.txt>"
	result := ProcessText(input, exec, startTime)

	// Should still have proper formatting
	if !strings.Contains(result, "=== END FILE ===") {
		t.Errorf("Expected '=== END FILE ===' in result\nFull result:\n%s", result)
	}
}

func TestProcessText_FailedCommandNotCounted(t *testing.T) {
	tempDir := t.TempDir()
	exec := createTestExecutor(t, tempDir)
	startTime := time.Now()

	// This should fail because file doesn't exist
	input := "<open nonexistent.txt>"
	result := ProcessText(input, exec, startTime)

	// Failed commands should show 0 executed
	if !strings.Contains(result, "Commands executed: 0") {
		t.Errorf("Expected 'Commands executed: 0' for failed command\nFull result:\n%s", result)
	}
}

func TestProcessText_MixedSuccessAndFailure(t *testing.T) {
	tempDir := t.TempDir()
	createTestFile(t, tempDir, "exists.txt", "I exist")

	exec := createTestExecutor(t, tempDir)
	startTime := time.Now()

	input := "<open exists.txt><open nonexistent.txt>"
	result := ProcessText(input, exec, startTime)

	// Only successful commands should be counted
	if !strings.Contains(result, "Commands executed: 1") {
		t.Errorf("Expected 'Commands executed: 1' (1 success, 1 failure)\nFull result:\n%s", result)
	}
}

func TestProcessText_ExecWithNonWhitelistedCommand(t *testing.T) {
	tempDir := t.TempDir()

	// Create executor with exec enabled but limited whitelist
	cfg := &config.Config{
		RepositoryRoot:    tempDir,
		MaxFileSize:       1048576,
		MaxWriteSize:      102400,
		AllowedExtensions: []string{".txt"},
		ExcludedPaths:     []string{".git"},
		ExecEnabled:       true,
		ExecWhitelist:     []string{"go test"}, // Only allow 'go test'
	}

	searchCfg := &search.SearchConfig{Enabled: false}
	auditLog := func(cmd, arg string, success bool, errMsg string) {}
	exec := evaluator.NewExecutor(cfg, searchCfg, auditLog)

	startTime := time.Now()
	input := "<exec rm -rf />" // Not in whitelist
	result := ProcessText(input, exec, startTime)

	// Should show error for non-whitelisted command
	if !strings.Contains(result, "=== ERROR:") {
		t.Errorf("Expected error for non-whitelisted command\nFull result:\n%s", result)
	}
}

func TestProcessText_WhitespaceOnlyInput(t *testing.T) {
	tempDir := t.TempDir()
	exec := createTestExecutor(t, tempDir)
	startTime := time.Now()

	input := "   \n\t\n   "
	result := ProcessText(input, exec, startTime)

	// No commands, should return original
	if result != input {
		t.Errorf("Whitespace-only input should return unchanged\ngot: %q\nwant: %q", result, input)
	}
}

func TestProcessText_NestedDirectoryWrite(t *testing.T) {
	tempDir := t.TempDir()

	// Create subdirectory
	subdir := filepath.Join(tempDir, "subdir")
	if err := os.MkdirAll(subdir, 0755); err != nil {
		t.Fatalf("Failed to create subdirectory: %v", err)
	}

	exec := createTestExecutor(t, tempDir)
	startTime := time.Now()

	input := "<write subdir/nested.txt>Nested content</write>"
	result := ProcessText(input, exec, startTime)

	if !strings.Contains(result, "=== WRITE SUCCESSFUL:") {
		t.Errorf("Expected successful write to nested directory\nFull result:\n%s", result)
	}

	// Verify file was created
	content, err := os.ReadFile(filepath.Join(subdir, "nested.txt"))
	if err != nil {
		t.Errorf("Nested file was not created: %v", err)
	}
	if string(content) != "Nested content" {
		t.Errorf("Nested file content = %q, want %q", string(content), "Nested content")
	}
}

func TestProcessText_LargeFile(t *testing.T) {
	tempDir := t.TempDir()

	// Create a file larger than default max size
	largeContent := strings.Repeat("x", 2*1024*1024) // 2MB
	createTestFile(t, tempDir, "large.txt", largeContent)

	exec := createTestExecutor(t, tempDir)
	startTime := time.Now()

	input := "<open large.txt>"
	result := ProcessText(input, exec, startTime)

	// Should error due to file size
	if !strings.Contains(result, "=== ERROR:") {
		t.Errorf("Expected error for large file\nFull result:\n%s", result)
	}
}

func TestProcessText_ExcludedPath(t *testing.T) {
	tempDir := t.TempDir()

	// Create .git directory and file
	gitDir := filepath.Join(tempDir, ".git")
	if err := os.MkdirAll(gitDir, 0755); err != nil {
		t.Fatalf("Failed to create .git directory: %v", err)
	}
	createTestFile(t, tempDir, ".git/config", "git config content")

	exec := createTestExecutor(t, tempDir)
	startTime := time.Now()

	input := "<open .git/config>"
	result := ProcessText(input, exec, startTime)

	// Should error due to excluded path
	if !strings.Contains(result, "=== ERROR:") {
		t.Errorf("Expected error for excluded path\nFull result:\n%s", result)
	}
}

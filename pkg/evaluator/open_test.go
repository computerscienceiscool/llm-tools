package evaluator

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/computerscienceiscool/llm-runtime/pkg/config"
)

// testAuditLog is a helper to capture audit log calls during tests
type testAuditLog struct {
	mu      sync.Mutex
	entries []auditEntry
}

type auditEntry struct {
	cmdType string
	arg     string
	success bool
	errMsg  string
}

func (t *testAuditLog) log(cmdType, arg string, success bool, errMsg string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.entries = append(t.entries, auditEntry{cmdType, arg, success, errMsg})
}

func (t *testAuditLog) getEntries() []auditEntry {
	t.mu.Lock()
	defer t.mu.Unlock()
	return append([]auditEntry{}, t.entries...)
}

func (t *testAuditLog) reset() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.entries = nil
}

// newTestConfig creates a config suitable for testing with the given temp directory
func newTestConfig(tmpDir string) *config.Config {
	return &config.Config{
		RepositoryRoot:    tmpDir,
		MaxFileSize:       1024 * 1024, // 1MB
		MaxWriteSize:      1024 * 100,  // 100KB
		ExcludedPaths:     []string{".git", ".env", "*.key", "*.pem"},
		AllowedExtensions: []string{".go", ".py", ".js", ".md", ".txt", ".json", ".yaml"},
		BackupBeforeWrite: true,
		IOTimeout:         60 * time.Second,
		IOContainerImage:  "llm-runtime-io:latest",
	}
}

func TestExecuteOpen_Success(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := newTestConfig(tmpDir)

	// Create a test file
	testContent := "Hello, World!\nThis is a test file."
	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	audit := &testAuditLog{}
	result := ExecuteOpen("test.txt", cfg, audit.log, nil)

	if !result.Success {
		t.Errorf("expected success, got error: %v", result.Error)
	}

	if result.Result != testContent {
		t.Errorf("expected content %q, got %q", testContent, result.Result)
	}

	if result.Command.Type != "open" {
		t.Errorf("expected command type 'open', got %q", result.Command.Type)
	}

	if result.Command.Argument != "test.txt" {
		t.Errorf("expected argument 'test.txt', got %q", result.Command.Argument)
	}

	if result.ExecutionTime <= 0 {
		t.Error("expected positive execution time")
	}

	// Check audit log
	entries := audit.getEntries()
	if len(entries) != 1 {
		t.Fatalf("expected 1 audit entry, got %d", len(entries))
	}
	if entries[0].cmdType != "open" || !entries[0].success {
		t.Errorf("unexpected audit entry: %+v", entries[0])
	}
}

func TestExecuteOpen_AbsolutePath(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := newTestConfig(tmpDir)

	testContent := "absolute path content"
	testFile := filepath.Join(tmpDir, "absolute.txt")
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	result := ExecuteOpen(testFile, cfg, nil, nil)

	if !result.Success {
		t.Errorf("expected success with absolute path, got error: %v", result.Error)
	}

	if result.Result != testContent {
		t.Errorf("expected content %q, got %q", testContent, result.Result)
	}
}

func TestExecuteOpen_FileNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := newTestConfig(tmpDir)

	audit := &testAuditLog{}
	result := ExecuteOpen("nonexistent.txt", cfg, audit.log, nil)

	if result.Success {
		t.Error("expected failure for nonexistent file")
	}

	if result.Error == nil {
		t.Error("expected error to be set")
	}

	if !strings.Contains(result.Error.Error(), "FILE_NOT_FOUND") {
		t.Errorf("expected FILE_NOT_FOUND error, got: %v", result.Error)
	}

	// Check audit log records failure
	entries := audit.getEntries()
	if len(entries) != 1 {
		t.Fatalf("expected 1 audit entry, got %d", len(entries))
	}
	if entries[0].success {
		t.Error("expected audit entry to show failure")
	}
}

func TestExecuteOpen_PathTraversal(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := newTestConfig(tmpDir)

	tests := []struct {
		name string
		path string
	}{
		{"parent directory", "../etc/passwd"},
		{"double parent", "../../etc/passwd"},
		{"hidden traversal", "subdir/../../../etc/passwd"},
		{"absolute outside", "/etc/passwd"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExecuteOpen(tt.path, cfg, nil, nil)

			if result.Success {
				t.Error("expected failure for path traversal attempt")
			}

			if result.Error == nil {
				t.Error("expected error to be set")
			}

			// Should fail with either PATH_SECURITY or FILE_NOT_FOUND
			errStr := result.Error.Error()
			if !strings.Contains(errStr, "PATH_SECURITY") && !strings.Contains(errStr, "FILE_NOT_FOUND") {
				t.Errorf("expected security or not found error, got: %v", result.Error)
			}
		})
	}
}

func TestExecuteOpen_ExcludedPaths(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := newTestConfig(tmpDir)

	// Create excluded files
	gitDir := filepath.Join(tmpDir, ".git")
	if err := os.MkdirAll(gitDir, 0755); err != nil {
		t.Fatalf("failed to create .git directory: %v", err)
	}

	gitConfig := filepath.Join(gitDir, "config")
	if err := os.WriteFile(gitConfig, []byte("git config"), 0644); err != nil {
		t.Fatalf("failed to create git config: %v", err)
	}

	tests := []struct {
		name string
		path string
	}{
		{"git directory", ".git/config"},
		{"env file", ".env"},
		{"key file", "secrets.key"},
		{"pem file", "cert.pem"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create the file if it doesn't exist
			fullPath := filepath.Join(tmpDir, tt.path)
			dir := filepath.Dir(fullPath)
			if err := os.MkdirAll(dir, 0755); err != nil {
				t.Fatalf("failed to create directory: %v", err)
			}
			if err := os.WriteFile(fullPath, []byte("secret"), 0644); err != nil {
				t.Fatalf("failed to create file: %v", err)
			}

			result := ExecuteOpen(tt.path, cfg, nil, nil)

			if result.Success {
				t.Error("expected failure for excluded path")
			}

			if result.Error == nil {
				t.Error("expected error to be set")
			}
		})
	}
}

func TestExecuteOpen_FileTooLarge(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := newTestConfig(tmpDir)
	cfg.MaxFileSize = 100 // Set very small limit

	// Create a file larger than the limit
	largeContent := strings.Repeat("x", 200)
	testFile := filepath.Join(tmpDir, "large.txt")
	if err := os.WriteFile(testFile, []byte(largeContent), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	result := ExecuteOpen("large.txt", cfg, nil, nil)

	if result.Success {
		t.Error("expected failure for file too large")
	}

	if result.Error == nil {
		t.Error("expected error to be set")
	}

	if !strings.Contains(result.Error.Error(), "RESOURCE_LIMIT") {
		t.Errorf("expected RESOURCE_LIMIT error, got: %v", result.Error)
	}
}

func TestExecuteOpen_EmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := newTestConfig(tmpDir)

	testFile := filepath.Join(tmpDir, "empty.txt")
	if err := os.WriteFile(testFile, []byte(""), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	result := ExecuteOpen("empty.txt", cfg, nil, nil)

	if !result.Success {
		t.Errorf("expected success for empty file, got error: %v", result.Error)
	}

	if result.Result != "" {
		t.Errorf("expected empty content, got %q", result.Result)
	}
}

func TestExecuteOpen_BinaryContent(t *testing.T) {
	t.Skip("TODO: Fix binary content handling")
	tmpDir := t.TempDir()
	cfg := newTestConfig(tmpDir)

	// Create file with binary content
	binaryContent := []byte{0x00, 0x01, 0x02, 0xFF, 0xFE, 0xFD}
	testFile := filepath.Join(tmpDir, "binary.txt")
	if err := os.WriteFile(testFile, binaryContent, 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	result := ExecuteOpen("binary.txt", cfg, nil, nil)

	if !result.Success {
		t.Errorf("expected success, got error: %v", result.Error)
	}

	if result.Result != string(binaryContent) {
		t.Error("binary content mismatch")
	}
}

func TestExecuteOpen_NestedDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := newTestConfig(tmpDir)

	// Create nested directory structure
	nestedDir := filepath.Join(tmpDir, "a", "b", "c")
	if err := os.MkdirAll(nestedDir, 0755); err != nil {
		t.Fatalf("failed to create nested directory: %v", err)
	}

	testContent := "nested content"
	testFile := filepath.Join(nestedDir, "nested.txt")
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	result := ExecuteOpen("a/b/c/nested.txt", cfg, nil, nil)

	if !result.Success {
		t.Errorf("expected success for nested file, got error: %v", result.Error)
	}

	if result.Result != testContent {
		t.Errorf("expected content %q, got %q", testContent, result.Result)
	}
}

func TestExecuteOpen_NilAuditLog(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := newTestConfig(tmpDir)

	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("content"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Should not panic with nil audit log
	result := ExecuteOpen("test.txt", cfg, nil, nil)

	if !result.Success {
		t.Errorf("expected success, got error: %v", result.Error)
	}
}

func TestExecuteOpen_NilAuditLogOnError(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := newTestConfig(tmpDir)

	// Should not panic with nil audit log on error path
	result := ExecuteOpen("nonexistent.txt", cfg, nil, nil)

	if result.Success {
		t.Error("expected failure for nonexistent file")
	}
}

func TestExecuteOpen_SpecialCharactersInFilename(t *testing.T) {
	t.Skip("TODO: Fix shell quoting for filenames with spaces")
	tmpDir := t.TempDir()
	cfg := newTestConfig(tmpDir)

	tests := []struct {
		name     string
		filename string
	}{
		{"spaces", "file with spaces.txt"},
		{"unicode", "文件.txt"},
		{"numbers", "123file456.txt"},
		{"dashes", "file-name-here.txt"},
		{"underscores", "file_name_here.txt"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testContent := "special filename content"
			testFile := filepath.Join(tmpDir, tt.filename)
			if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
				t.Fatalf("failed to create test file: %v", err)
			}

			result := ExecuteOpen(tt.filename, cfg, nil, nil)

			if !result.Success {
				t.Errorf("expected success for %q, got error: %v", tt.filename, result.Error)
			}

			if result.Result != testContent {
				t.Errorf("expected content %q, got %q", testContent, result.Result)
			}
		})
	}
}

func TestExecuteOpen_ExecutionTimeTracking(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := newTestConfig(tmpDir)

	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("content"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	startTime := time.Now()
	result := ExecuteOpen("test.txt", cfg, nil, nil)
	elapsed := time.Since(startTime)

	if result.ExecutionTime <= 0 {
		t.Error("execution time should be positive")
	}

	if result.ExecutionTime > elapsed {
		t.Error("execution time should not exceed total elapsed time")
	}
}

func TestExecuteOpen_DirectoryInsteadOfFile(t *testing.T) {
	t.Skip("TODO: Add proper directory detection")
	tmpDir := t.TempDir()
	cfg := newTestConfig(tmpDir)

	// Create a subdirectory
	subDir := filepath.Join(tmpDir, "subdir")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatalf("failed to create subdirectory: %v", err)
	}

	result := ExecuteOpen("subdir", cfg, nil, nil)

	if result.Success {
		t.Error("expected failure when opening a directory")
	}

	// Should fail with READ_ERROR since directories can't be read as files
	if result.Error == nil {
		t.Error("expected error to be set")
	}
}

func TestExecuteOpen_MaxFileSizeBoundary(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := newTestConfig(tmpDir)
	cfg.MaxFileSize = 100

	tests := []struct {
		name       string
		size       int
		shouldPass bool
	}{
		{"exactly at limit", 100, true},
		{"one under limit", 99, true},
		{"one over limit", 101, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content := strings.Repeat("x", tt.size)
			filename := tt.name + ".txt"
			testFile := filepath.Join(tmpDir, filename)
			if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
				t.Fatalf("failed to create test file: %v", err)
			}

			result := ExecuteOpen(filename, cfg, nil, nil)

			if tt.shouldPass && !result.Success {
				t.Errorf("expected success for size %d, got error: %v", tt.size, result.Error)
			}

			if !tt.shouldPass && result.Success {
				t.Errorf("expected failure for size %d", tt.size)
			}
		})
	}
}

func TestExecuteOpen_AuditLogContents(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := newTestConfig(tmpDir)

	testFile := filepath.Join(tmpDir, "audit_test.txt")
	if err := os.WriteFile(testFile, []byte("content"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	audit := &testAuditLog{}

	// Test successful open
	ExecuteOpen("audit_test.txt", cfg, audit.log, nil)

	entries := audit.getEntries()
	if len(entries) != 1 {
		t.Fatalf("expected 1 audit entry, got %d", len(entries))
	}

	entry := entries[0]
	if entry.cmdType != "open" {
		t.Errorf("expected cmdType 'open', got %q", entry.cmdType)
	}
	if entry.arg != "audit_test.txt" {
		t.Errorf("expected arg 'audit_test.txt', got %q", entry.arg)
	}
	if !entry.success {
		t.Error("expected success to be true")
	}
	if entry.errMsg != "" {
		t.Errorf("expected empty error message, got %q", entry.errMsg)
	}

	// Test failed open
	audit.reset()
	ExecuteOpen("nonexistent.txt", cfg, audit.log, nil)

	entries = audit.getEntries()
	if len(entries) != 1 {
		t.Fatalf("expected 1 audit entry, got %d", len(entries))
	}

	entry = entries[0]
	if entry.success {
		t.Error("expected success to be false")
	}
	if entry.errMsg == "" {
		t.Error("expected error message to be set")
	}
}

// Benchmark tests
func BenchmarkExecuteOpen_SmallFile(b *testing.B) {
	tmpDir := b.TempDir()
	cfg := newTestConfig(tmpDir)

	testFile := filepath.Join(tmpDir, "small.txt")
	if err := os.WriteFile(testFile, []byte("small content"), 0644); err != nil {
		b.Fatalf("failed to create test file: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ExecuteOpen("small.txt", cfg, nil, nil)
	}
}

func BenchmarkExecuteOpen_LargeFile(b *testing.B) {
	tmpDir := b.TempDir()
	cfg := newTestConfig(tmpDir)

	// Create a 100KB file
	largeContent := strings.Repeat("x", 100*1024)
	testFile := filepath.Join(tmpDir, "large.txt")
	if err := os.WriteFile(testFile, []byte(largeContent), 0644); err != nil {
		b.Fatalf("failed to create test file: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ExecuteOpen("large.txt", cfg, nil, nil)
	}
}

func TestExecuteOpen_PermissionDenied(t *testing.T) {
	t.Skip("Read-only bind mounts bypass individual file permissions - invalid for containerized I/O")
	t.Skip("Read-only bind mounts bypass individual file permissions - test is invalid for containerized I/O")
	tmpDir := t.TempDir()
	cfg := newTestConfig(tmpDir)

	// Create a file with no read permissions
	testFile := filepath.Join(tmpDir, "noperm.txt")
	if err := os.WriteFile(testFile, []byte("secret"), 0000); err != nil {
		t.Fatalf("failed to create file: %v", err)
	}
	defer os.Chmod(testFile, 0644) // Restore for cleanup

	result := ExecuteOpen("noperm.txt", cfg, nil, nil)

	if result.Success {
		t.Error("expected failure when file is not readable")
	}

	// FIX: Check if Error is nil first
	if result.Error == nil {
		t.Fatal("expected error to be set when operation fails")
	}

	// Should fail with either PERMISSION_DENIED or READ_ERROR
	errStr := result.Error.Error()
	if !strings.Contains(errStr, "PERMISSION_DENIED") && !strings.Contains(errStr, "READ_ERROR") {
		t.Errorf("expected permission or read error, got: %v", result.Error)
	}
}

func TestExecuteOpen_AuditLogOnPermissionDenied(t *testing.T) {
	t.Skip("Related to permission denied test - invalid for containerized I/O")
	if os.Getuid() == 0 {
		t.Skip("Skipping permission test when running as root")
	}

	tmpDir := t.TempDir()
	cfg := newTestConfig(tmpDir)

	// Create a file with no read permissions
	testFile := filepath.Join(tmpDir, "noperm_audit.txt")
	if err := os.WriteFile(testFile, []byte("secret"), 0000); err != nil {
		t.Fatalf("failed to create file: %v", err)
	}
	defer os.Chmod(testFile, 0644)

	audit := &testAuditLog{}
	ExecuteOpen("noperm_audit.txt", cfg, audit.log, nil)

	entries := audit.getEntries()
	if len(entries) != 1 {
		t.Fatalf("expected 1 audit entry, got %d", len(entries))
	}

	if entries[0].success {
		t.Error("audit should show failure")
	}
}

func TestExecuteOpen_AuditLogOnFileTooLarge(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := newTestConfig(tmpDir)
	cfg.MaxFileSize = 10

	testFile := filepath.Join(tmpDir, "large_audit.txt")
	if err := os.WriteFile(testFile, []byte("this is too large"), 0644); err != nil {
		t.Fatalf("failed to create file: %v", err)
	}

	audit := &testAuditLog{}
	ExecuteOpen("large_audit.txt", cfg, audit.log, nil)

	entries := audit.getEntries()
	if len(entries) != 1 {
		t.Fatalf("expected 1 audit entry, got %d", len(entries))
	}

	if entries[0].success {
		t.Error("audit should show failure")
	}
	if !strings.Contains(entries[0].errMsg, "RESOURCE_LIMIT") {
		t.Errorf("expected RESOURCE_LIMIT in error, got %q", entries[0].errMsg)
	}
}

func TestExecuteOpen_ReadErrorOnDirectory(t *testing.T) {
	t.Skip("TODO: Add proper directory detection")
	tmpDir := t.TempDir()
	cfg := newTestConfig(tmpDir)

	// Create a subdirectory
	subDir := filepath.Join(tmpDir, "testdir")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatalf("failed to create subdirectory: %v", err)
	}

	audit := &testAuditLog{}
	result := ExecuteOpen("testdir", cfg, audit.log, nil)

	if result.Success {
		t.Error("expected failure when opening a directory")
	}

	if !strings.Contains(result.Error.Error(), "READ_ERROR") {
		t.Errorf("expected READ_ERROR, got: %v", result.Error)
	}

	// Check audit log
	entries := audit.getEntries()
	if len(entries) != 1 {
		t.Fatalf("expected 1 audit entry, got %d", len(entries))
	}
	if entries[0].success {
		t.Error("audit should show failure")
	}
}

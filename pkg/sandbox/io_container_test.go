package sandbox

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestRunIOContainer_SimpleCommand(t *testing.T) {
	if !isDockerAvailable() {
		t.Skip("Docker not available")
	}

	tempDir := t.TempDir()
	output, err := RunIOContainer(
		tempDir,
		"alpine:latest",
		"echo hello",
		5*time.Second,
		"128m",
		1,
	)
	if err != nil {
		t.Fatalf("RunIOContainer() error = %v", err)
	}
	if !strings.Contains(output, "hello") {
		t.Errorf("Expected 'hello' in output, got: %s", output)
	}
}

func TestRunIOContainer_Timeout(t *testing.T) {
	if !isDockerAvailable() {
		t.Skip("Docker not available")
	}

	tempDir := t.TempDir()
	_, err := RunIOContainer(
		tempDir,
		"alpine:latest",
		"sleep 10",
		100*time.Millisecond,
		"128m",
		1,
	)
	if err == nil {
		t.Error("Expected timeout error")
	}
	// Accept either timeout message or context deadline exceeded
	if !strings.Contains(err.Error(), "timed out") && !strings.Contains(err.Error(), "context deadline exceeded") {
		t.Errorf("Expected timeout error, got: %v", err)
	}
}

func TestReadFileInContainer_Success(t *testing.T) {
	if !isDockerAvailable() {
		t.Skip("Docker not available")
	}

	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.txt")
	testContent := "file content here"
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	content, err := ReadFileInContainer(
		testFile,
		tempDir,
		"alpine:latest",
		5*time.Second,
		"128m",
		1,
	)
	if err != nil {
		t.Fatalf("ReadFileInContainer() error = %v", err)
	}
	if !strings.Contains(content, testContent) {
		t.Errorf("Expected %q in output, got: %s", testContent, content)
	}
}

func TestWriteFileInContainer_Success(t *testing.T) {
	if !isDockerAvailable() {
		t.Skip("Docker not available")
	}

	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "output.txt")
	testContent := "written by container"

	err := WriteFileInContainer(
		testFile,
		testContent,
		tempDir,
		"alpine:latest",
		5*time.Second,
		"128m",
		1,
	)
	if err != nil {
		t.Fatalf("WriteFileInContainer() error = %v", err)
	}

	content, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read written file: %v", err)
	}
	if string(content) != testContent {
		t.Errorf("Expected %q, got: %q", testContent, string(content))
	}
}

func TestEnsureIOContainerImage_AlpineExists(t *testing.T) {
	if !isDockerAvailable() {
		t.Skip("Docker not available")
	}

	err := EnsureIOContainerImage("alpine:latest")
	if err != nil {
		t.Errorf("EnsureIOContainerImage(alpine:latest) error = %v", err)
	}
}

func TestValidateIOContainer_Success(t *testing.T) {
	if !isDockerAvailable() {
		t.Skip("Docker not available")
	}

	tempDir := t.TempDir()
	err := ValidateIOContainer(tempDir, "alpine:latest")
	if err != nil {
		t.Errorf("ValidateIOContainer() error = %v", err)
	}
}

func TestValidateIOContainer_NonExistentRepo(t *testing.T) {
	if !isDockerAvailable() {
		t.Skip("Docker not available")
	}

	err := ValidateIOContainer("/nonexistent/path/12345", "alpine:latest")
	if err == nil {
		t.Error("Expected error for non-existent repository")
	}
}

// TestParseMemoryLimitIO tests memory limit string parsing for IO containers
func TestParseMemoryLimitIO(t *testing.T) {
	tests := []struct {
		name     string
		limit    string
		expected int64
	}{
		{"empty string", "", 0},
		{"128 megabytes lowercase", "128m", 128 * 1024 * 1024},
		{"128 megabytes uppercase", "128M", 128 * 1024 * 1024},
		{"512 megabytes", "512m", 512 * 1024 * 1024},
		{"1 gigabyte lowercase", "1g", 1 * 1024 * 1024 * 1024},
		{"1 gigabyte uppercase", "1G", 1 * 1024 * 1024 * 1024},
		{"2 gigabytes", "2g", 2 * 1024 * 1024 * 1024},
		{"invalid format", "invalid", 0},
		{"no suffix", "256", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseMemoryLimitIO(tt.limit)
			if result != tt.expected {
				t.Errorf("parseMemoryLimitIO(%q) = %d, want %d", tt.limit, result, tt.expected)
			}
		})
	}
}

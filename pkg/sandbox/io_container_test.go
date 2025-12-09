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
	if !strings.Contains(err.Error(), "timed out") {
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

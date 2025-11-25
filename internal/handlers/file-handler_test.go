package handlers

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

//
// MOCKS THAT MATCH PRODUCTION CODE
//

// MockPathValidator implements security.PathValidator
type MockPathValidator struct {
	ValidateWriteExtensionFunc func(string, []string) error
}

func (m *MockPathValidator) ValidatePath(requestedPath, repoRoot string, excluded []string) (string, error) {
	return filepath.Join(repoRoot, requestedPath), nil
}

func (m *MockPathValidator) ValidateWriteExtension(path string, allowed []string) error {
	if m.ValidateWriteExtensionFunc != nil {
		return m.ValidateWriteExtensionFunc(path, allowed)
	}
	return nil
}

// MockAuditLogger implements security.AuditLogger
type MockAuditLogger struct{}

func (m *MockAuditLogger) LogOperation(sessionID, command, argument string, success bool, errorMsg string) {
	// no-op
}

//
// TESTS
//

func TestOpenFile(t *testing.T) {
	tempDir := t.TempDir()

	// Create a test file
	testPath := filepath.Join(tempDir, "test.txt")
	err := os.WriteFile(testPath, []byte("Hello, World!"), 0644)
	require.NoError(t, err)

	handler := NewFileHandler(&MockPathValidator{}, &MockAuditLogger{})

	content, err := handler.OpenFile(testPath, 1024*1024, tempDir)
	assert.NoError(t, err)
	assert.Equal(t, "Hello, World!", content)
}

func TestOpenFileTooLarge(t *testing.T) {
	tempDir := t.TempDir()

	data := strings.Repeat("X", 200000)
	testPath := filepath.Join(tempDir, "large.txt")
	err := os.WriteFile(testPath, []byte(data), 0644)
	require.NoError(t, err)

	handler := NewFileHandler(&MockPathValidator{}, &MockAuditLogger{})

	_, err = handler.OpenFile(testPath, 50000, tempDir)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "RESOURCE_LIMIT")
}

func TestWriteFileCreate(t *testing.T) {
	tempDir := t.TempDir()

	handler := NewFileHandler(&MockPathValidator{}, &MockAuditLogger{})

	result, err := handler.WriteFile(
		filepath.Join(tempDir, "new.txt"),
		"content",
		1024*1024,
		tempDir,
		[]string{".txt"},
		false,
	)

	assert.NoError(t, err)
	assert.Equal(t, "CREATED", result.Action)

	actual, _ := os.ReadFile(filepath.Join(tempDir, "new.txt"))
	assert.Equal(t, "content", string(actual))
}

func TestWriteFileUpdateWithBackup(t *testing.T) {
	tempDir := t.TempDir()

	original := filepath.Join(tempDir, "file.txt")
	err := os.WriteFile(original, []byte("old"), 0644)
	require.NoError(t, err)

	handler := NewFileHandler(&MockPathValidator{}, &MockAuditLogger{})

	result, err := handler.WriteFile(
		original,
		"new",
		1024*1024,
		tempDir,
		[]string{".txt"},
		true,
	)

	assert.NoError(t, err)
	assert.Equal(t, "UPDATED", result.Action)
	assert.NotEmpty(t, result.BackupFile)

	// verify backup has old content
	backup, _ := os.ReadFile(result.BackupFile)
	assert.Equal(t, "old", string(backup))

	// verify main file updated
	updated, _ := os.ReadFile(original)
	assert.Equal(t, "new", string(updated))
}

func TestWriteFileInvalidExtension(t *testing.T) {
	tempDir := t.TempDir()

	validator := &MockPathValidator{}
	logger := &MockAuditLogger{}
	handler := NewFileHandler(validator, logger)

	// override extension validator correctly
	validator.ValidateWriteExtensionFunc = func(path string, allowed []string) error {
		return fmt.Errorf("EXTENSION_DENIED")
	}

	_, err := handler.WriteFile(
		filepath.Join(tempDir, "bad.exe"),
		"content",
		1024,
		tempDir,
		[]string{".txt"},
		false,
	)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "EXTENSION_DENIED")
}

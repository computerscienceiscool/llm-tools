package handlers

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockFileHandler for testing
type MockFileHandler struct {
	mock.Mock
}

func (m *MockFileHandler) OpenFile(filePath string, maxSize int64, repoRoot string) (string, error) {
	args := m.Called(filePath, maxSize, repoRoot)
	return args.String(0), args.Error(1)
}

func (m *MockFileHandler) WriteFile(filePath, content string, maxSize int64, repoRoot string, allowedExts []string, backup bool) (WriteResult, error) {
	args := m.Called(filePath, content, maxSize, repoRoot, allowedExts, backup)
	return args.Get(0).(WriteResult), args.Error(1)
}

// MockExecHandler for testing
type MockExecHandler struct {
	mock.Mock
}

func (m *MockExecHandler) ExecuteCommand(command string, config ExecConfig) (ExecResult, error) {
	args := m.Called(command, config)
	return args.Get(0).(ExecResult), args.Error(1)
}

// MockSearchHandler for testing
type MockSearchHandler struct {
	mock.Mock
}

func (m *MockSearchHandler) Search(query string) ([]SearchResult, error) {
	args := m.Called(query)
	return args.Get(0).([]SearchResult), args.Error(1)
}

// TestFileHandlerInterface tests the FileHandler interface
func TestFileHandlerInterface(t *testing.T) {
	var _ FileHandler = (*MockFileHandler)(nil)

	mockHandler := &MockFileHandler{}

	// Test OpenFile
	mockHandler.On("OpenFile", "test.txt", int64(1024), "/repo").Return("content", nil)
	content, err := mockHandler.OpenFile("test.txt", 1024, "/repo")
	assert.NoError(t, err)
	assert.Equal(t, "content", content)

	// Test WriteFile
	writeResult := WriteResult{Action: "CREATED", BytesWritten: 10}
	mockHandler.On("WriteFile", "out.txt", "data", int64(1024), "/repo", []string{".txt"}, true).
		Return(writeResult, nil)

	result, err := mockHandler.WriteFile("out.txt", "data", 1024, "/repo", []string{".txt"}, true)
	assert.NoError(t, err)
	assert.Equal(t, "CREATED", result.Action)
	assert.Equal(t, int64(10), result.BytesWritten)

	mockHandler.AssertExpectations(t)
}

// TestExecHandlerInterface tests the ExecHandler interface
func TestExecHandlerInterface(t *testing.T) {
	var _ ExecHandler = (*MockExecHandler)(nil)

	mockHandler := &MockExecHandler{}
	config := ExecConfig{
		Enabled:   true,
		Whitelist: []string{"echo"},
		Timeout:   30 * time.Second,
	}

	execResult := ExecResult{ExitCode: 0, Stdout: "hello", Duration: time.Second}
	mockHandler.On("ExecuteCommand", "echo hello", config).Return(execResult, nil)

	result, err := mockHandler.ExecuteCommand("echo hello", config)
	assert.NoError(t, err)
	assert.Equal(t, 0, result.ExitCode)
	assert.Equal(t, "hello", result.Stdout)

	mockHandler.AssertExpectations(t)
}

// TestSearchHandlerInterface tests the SearchHandler interface
func TestSearchHandlerInterface(t *testing.T) {
	var _ SearchHandler = (*MockSearchHandler)(nil)

	mockHandler := &MockSearchHandler{}
	searchResults := []SearchResult{
		{FilePath: "file.go", Score: 0.95, Lines: 100},
	}

	mockHandler.On("Search", "test query").Return(searchResults, nil)

	results, err := mockHandler.Search("test query")
	assert.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, "file.go", results[0].FilePath)

	mockHandler.AssertExpectations(t)
}

// TestWriteResult tests the WriteResult structure
func TestWriteResult(t *testing.T) {
	result := WriteResult{
		Action:       "CREATED",
		BytesWritten: 1024,
		BackupFile:   "backup.txt",
	}

	assert.Equal(t, "CREATED", result.Action)
	assert.Equal(t, int64(1024), result.BytesWritten)
	assert.Equal(t, "backup.txt", result.BackupFile)
}

// TestExecConfig tests the ExecConfig structure
func TestExecConfig(t *testing.T) {
	config := ExecConfig{
		Enabled:        true,
		Whitelist:      []string{"go", "npm"},
		Timeout:        30 * time.Second,
		MemoryLimit:    "512m",
		CPULimit:       2,
		ContainerImage: "ubuntu:22.04",
		RepoRoot:       "/repo",
	}

	assert.True(t, config.Enabled)
	assert.Contains(t, config.Whitelist, "go")
	assert.Equal(t, 30*time.Second, config.Timeout)
}

// TestExecResult tests the ExecResult structure
func TestExecResult(t *testing.T) {
	result := ExecResult{
		ExitCode: 0,
		Stdout:   "success output",
		Stderr:   "",
		Duration: time.Millisecond * 500,
	}

	assert.Equal(t, 0, result.ExitCode)
	assert.Equal(t, "success output", result.Stdout)
	assert.Empty(t, result.Stderr)
	assert.Equal(t, time.Millisecond*500, result.Duration)
}

// TestSearchResult tests the SearchResult structure
func TestSearchResult(t *testing.T) {
	result := SearchResult{
		FilePath: "src/main.go",
		Score:    0.85,
		Lines:    200,
		Size:     4096,
		Preview:  "function main() {",
		ModTime:  time.Now(),
	}

	assert.Equal(t, "src/main.go", result.FilePath)
	assert.Equal(t, 0.85, result.Score)
	assert.Equal(t, 200, result.Lines)
	assert.Equal(t, int64(4096), result.Size)
	assert.Contains(t, result.Preview, "main")
}

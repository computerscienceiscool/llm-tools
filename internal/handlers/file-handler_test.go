package handlers

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockFileSystem for testing file handler
type MockFileSystem struct {
	mock.Mock
}

func (m *MockFileSystem) ReadFile(filename string) ([]byte, error) {
	args := m.Called(filename)
	return args.Get(0).([]byte), args.Error(1)
}

func (m *MockFileSystem) WriteFile(filename string, data []byte, perm os.FileMode) error {
	args := m.Called(filename, data, perm)
	return args.Error(0)
}

func (m *MockFileSystem) Stat(filename string) (os.FileInfo, error) {
	args := m.Called(filename)
	return args.Get(0).(os.FileInfo), args.Error(1)
}

func (m *MockFileSystem) Exists(filename string) bool {
	args := m.Called(filename)
	return args.Bool(0)
}

// MockPathValidator for testing
type MockPathValidator struct {
	mock.Mock
}

func (m *MockPathValidator) ValidatePath(requestedPath, repositoryRoot string, excludedPaths []string) (string, error) {
	args := m.Called(requestedPath, repositoryRoot, excludedPaths)
	return args.String(0), args.Error(1)
}

func (m *MockPathValidator) ValidateWriteExtension(filepath string, allowedExtensions []string) error {
	args := m.Called(filepath, allowedExtensions)
	return args.Error(0)
}

// MockFileInfo for testing
type MockFileInfo struct {
	name    string
	size    int64
	mode    os.FileMode
	modTime time.Time
	isDir   bool
}

func (m MockFileInfo) Name() string       { return m.name }
func (m MockFileInfo) Size() int64        { return m.size }
func (m MockFileInfo) Mode() os.FileMode  { return m.mode }
func (m MockFileInfo) ModTime() time.Time { return m.modTime }
func (m MockFileInfo) IsDir() bool        { return m.isDir }
func (m MockFileInfo) Sys() interface{}   { return nil }

// TestDefaultFileHandler tests the default file handler implementation
func TestDefaultFileHandler(t *testing.T) {
	mockFS := &MockFileSystem{}
	mockValidator := &MockPathValidator{}

	handler := NewFileHandler(mockFS, mockValidator)
	require.NotNil(t, handler)

	t.Run("read file successfully", func(t *testing.T) {
		testContent := []byte("package main\n\nimport \"fmt\"\n\nfunc main() {\n\tfmt.Println(\"Hello, World!\")\n}")

		// Setup mocks
		mockValidator.On("ValidatePath", "main.go", "/repo", []string{".git"}).
			Return("/repo/main.go", nil)
		mockFS.On("ReadFile", "/repo/main.go").Return(testContent, nil)
		mockFS.On("Stat", "/repo/main.go").Return(MockFileInfo{
			name: "main.go",
			size: int64(len(testContent)),
		}, nil)

		// Test read
		content, err := handler.ReadFile("main.go", 1048576, "/repo", []string{".git"})
		assert.NoError(t, err)
		assert.Equal(t, string(testContent), content)

		mockFS.AssertExpectations(t)
		mockValidator.AssertExpectations(t)
	})

	t.Run("read file exceeds size limit", func(t *testing.T) {
		// Reset mocks
		mockFS = &MockFileSystem{}
		mockValidator = &MockPathValidator{}
		handler = NewFileHandler(mockFS, mockValidator)

		mockValidator.On("ValidatePath", "large.txt", "/repo", mock.Anything).
			Return("/repo/large.txt", nil)
		mockFS.On("Stat", "/repo/large.txt").Return(MockFileInfo{
			name: "large.txt",
			size: 2048576, // 2MB
		}, nil)

		// Test read with 1MB limit
		_, err := handler.ReadFile("large.txt", 1048576, "/repo", []string{})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "file size")

		mockFS.AssertExpectations(t)
		mockValidator.AssertExpectations(t)
	})

	t.Run("read invalid path", func(t *testing.T) {
		mockFS = &MockFileSystem{}
		mockValidator = &MockPathValidator{}
		handler = NewFileHandler(mockFS, mockValidator)

		mockValidator.On("ValidatePath", "../etc/passwd", "/repo", mock.Anything).
			Return("", fmt.Errorf("PATH_SECURITY: path traversal detected"))

		_, err := handler.ReadFile("../etc/passwd", 1048576, "/repo", []string{})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "PATH_SECURITY")

		mockValidator.AssertExpectations(t)
	})
}

// TestFileHandlerWrite tests file writing functionality
func TestFileHandlerWrite(t *testing.T) {
	mockFS := &MockFileSystem{}
	mockValidator := &MockPathValidator{}
	handler := NewFileHandler(mockFS, mockValidator)

	t.Run("create new file", func(t *testing.T) {
		content := "package main\n\nfunc main() {\n\tprintln(\"Hello\")\n}"

		mockValidator.On("ValidatePath", "new.go", "/repo", []string{".git"}).
			Return("/repo/new.go", nil)
		mockValidator.On("ValidateWriteExtension", "/repo/new.go", []string{".go", ".txt"}).
			Return(nil)
		mockFS.On("Exists", "/repo/new.go").Return(false)
		mockFS.On("WriteFile", "/repo/new.go", []byte(content), os.FileMode(0644)).
			Return(nil)

		result, err := handler.WriteFile("new.go", content, 102400, "/repo", []string{".go", ".txt"}, []string{".git"}, false)
		assert.NoError(t, err)
		assert.Equal(t, "CREATED", result.Action)
		assert.Equal(t, int64(len(content)), result.BytesWritten)
		assert.Empty(t, result.BackupFile)

		mockFS.AssertExpectations(t)
		mockValidator.AssertExpectations(t)
	})

	t.Run("update existing file with backup", func(t *testing.T) {
		mockFS = &MockFileSystem{}
		mockValidator = &MockPathValidator{}
		handler = NewFileHandler(mockFS, mockValidator)

		newContent := "updated content"
		originalContent := []byte("original content")

		mockValidator.On("ValidatePath", "existing.txt", "/repo", []string{}).
			Return("/repo/existing.txt", nil)
		mockValidator.On("ValidateWriteExtension", "/repo/existing.txt", []string{".txt"}).
			Return(nil)
		mockFS.On("Exists", "/repo/existing.txt").Return(true)
		mockFS.On("ReadFile", "/repo/existing.txt").Return(originalContent, nil)
		mockFS.On("WriteFile", mock.MatchedBy(func(filename string) bool {
			return strings.Contains(filename, "existing.txt.backup")
		}), originalContent, os.FileMode(0644)).Return(nil)
		mockFS.On("WriteFile", "/repo/existing.txt", []byte(newContent), os.FileMode(0644)).
			Return(nil)

		result, err := handler.WriteFile("existing.txt", newContent, 102400, "/repo", []string{".txt"}, []string{}, true)
		assert.NoError(t, err)
		assert.Equal(t, "UPDATED", result.Action)
		assert.Equal(t, int64(len(newContent)), result.BytesWritten)
		assert.NotEmpty(t, result.BackupFile)

		mockFS.AssertExpectations(t)
		mockValidator.AssertExpectations(t)
	})

	t.Run("write exceeds size limit", func(t *testing.T) {
		mockFS = &MockFileSystem{}
		mockValidator = &MockPathValidator{}
		handler = NewFileHandler(mockFS, mockValidator)

		largeContent := strings.Repeat("x", 200000) // 200KB content

		mockValidator.On("ValidatePath", "large.txt", "/repo", []string{}).
			Return("/repo/large.txt", nil)

		_, err := handler.WriteFile("large.txt", largeContent, 102400, "/repo", []string{".txt"}, []string{}, false) // 100KB limit
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "content size")

		mockValidator.AssertExpectations(t)
	})

	t.Run("forbidden extension", func(t *testing.T) {
		mockFS = &MockFileSystem{}
		mockValidator = &MockPathValidator{}
		handler = NewFileHandler(mockFS, mockValidator)

		mockValidator.On("ValidatePath", "malware.exe", "/repo", []string{}).
			Return("/repo/malware.exe", nil)
		mockValidator.On("ValidateWriteExtension", "/repo/malware.exe", []string{".txt", ".go"}).
			Return(fmt.Errorf("EXTENSION_DENIED: .exe files not allowed"))

		_, err := handler.WriteFile("malware.exe", "content", 102400, "/repo", []string{".txt", ".go"}, []string{}, false)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "EXTENSION_DENIED")

		mockValidator.AssertExpectations(t)
	})
}

// TestFileHandlerRealFilesystem tests with actual filesystem
func TestFileHandlerRealFilesystem(t *testing.T) {
	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "filehandler-test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Use real filesystem and path validator
	fs := NewRealFileSystem()
	validator := NewDefaultPathValidator()
	handler := NewFileHandler(fs, validator)

	t.Run("read actual file", func(t *testing.T) {
		testFile := filepath.Join(tempDir, "test.txt")
		testContent := "This is a test file\nwith multiple lines\n"

		// Create test file
		err := os.WriteFile(testFile, []byte(testContent), 0644)
		require.NoError(t, err)

		// Read through handler
		content, err := handler.ReadFile("test.txt", 1048576, tempDir, []string{})
		assert.NoError(t, err)
		assert.Equal(t, testContent, content)
	})

	t.Run("write actual file", func(t *testing.T) {
		content := "New file content\nCreated by handler\n"

		result, err := handler.WriteFile("new.txt", content, 102400, tempDir, []string{".txt"}, []string{}, false)
		assert.NoError(t, err)
		assert.Equal(t, "CREATED", result.Action)

		// Verify file was created
		createdFile := filepath.Join(tempDir, "new.txt")
		actualContent, err := os.ReadFile(createdFile)
		assert.NoError(t, err)
		assert.Equal(t, content, string(actualContent))
	})

	t.Run("backup functionality", func(t *testing.T) {
		originalContent := "Original content"
		newContent := "Updated content"

		// Create original file
		testFile := "backup_test.txt"
		fullPath := filepath.Join(tempDir, testFile)
		err := os.WriteFile(fullPath, []byte(originalContent), 0644)
		require.NoError(t, err)

		// Update with backup
		result, err := handler.WriteFile(testFile, newContent, 102400, tempDir, []string{".txt"}, []string{}, true)
		assert.NoError(t, err)
		assert.Equal(t, "UPDATED", result.Action)
		assert.NotEmpty(t, result.BackupFile)

		// Verify backup file exists and contains original content
		backupContent, err := os.ReadFile(result.BackupFile)
		assert.NoError(t, err)
		assert.Equal(t, originalContent, string(backupContent))

		// Verify main file has new content
		mainContent, err := os.ReadFile(fullPath)
		assert.NoError(t, err)
		assert.Equal(t, newContent, string(mainContent))
	})

	t.Run("security validation", func(t *testing.T) {
		// Try to read outside repository
		_, err := handler.ReadFile("../../../etc/passwd", 1048576, tempDir, []string{})
		assert.Error(t, err)

		// Try to write outside repository
		_, err = handler.WriteFile("../../../malicious.txt", "bad content", 102400, tempDir, []string{".txt"}, []string{}, false)
		assert.Error(t, err)
	})
}

// TestFileHandlerEdgeCases tests edge cases and error conditions
func TestFileHandlerEdgeCases(t *testing.T) {
	mockFS := &MockFileSystem{}
	mockValidator := &MockPathValidator{}
	handler := NewFileHandler(mockFS, mockValidator)

	t.Run("empty file content", func(t *testing.T) {
		mockValidator.On("ValidatePath", "empty.txt", "/repo", []string{}).
			Return("/repo/empty.txt", nil)
		mockFS.On("ReadFile", "/repo/empty.txt").Return([]byte{}, nil)
		mockFS.On("Stat", "/repo/empty.txt").Return(MockFileInfo{
			name: "empty.txt",
			size: 0,
		}, nil)

		content, err := handler.ReadFile("empty.txt", 1048576, "/repo", []string{})
		assert.NoError(t, err)
		assert.Empty(t, content)

		mockFS.AssertExpectations(t)
		mockValidator.AssertExpectations(t)
	})

	t.Run("binary file handling", func(t *testing.T) {
		mockFS = &MockFileSystem{}
		mockValidator = &MockPathValidator{}
		handler = NewFileHandler(mockFS, mockValidator)

		binaryData := []byte{0x00, 0x01, 0x02, 0xFF, 0xFE, 0xFD}

		mockValidator.On("ValidatePath", "binary.dat", "/repo", []string{}).
			Return("/repo/binary.dat", nil)
		mockFS.On("ReadFile", "/repo/binary.dat").Return(binaryData, nil)
		mockFS.On("Stat", "/repo/binary.dat").Return(MockFileInfo{
			name: "binary.dat",
			size: int64(len(binaryData)),
		}, nil)

		content, err := handler.ReadFile("binary.dat", 1048576, "/repo", []string{})
		assert.NoError(t, err)
		assert.Equal(t, string(binaryData), content)

		mockFS.AssertExpectations(t)
		mockValidator.AssertExpectations(t)
	})

	t.Run("unicode content", func(t *testing.T) {
		mockFS = &MockFileSystem{}
		mockValidator = &MockPathValidator{}
		handler = NewFileHandler(mockFS, mockValidator)

		unicodeContent := "Hello 世界 🌍 测试"

		mockValidator.On("ValidatePath", "unicode.txt", "/repo", []string{}).
			Return("/repo/unicode.txt", nil)
		mockValidator.On("ValidateWriteExtension", "/repo/unicode.txt", []string{".txt"}).
			Return(nil)
		mockFS.On("Exists", "/repo/unicode.txt").Return(false)
		mockFS.On("WriteFile", "/repo/unicode.txt", []byte(unicodeContent), os.FileMode(0644)).
			Return(nil)

		result, err := handler.WriteFile("unicode.txt", unicodeContent, 102400, "/repo", []string{".txt"}, []string{}, false)
		assert.NoError(t, err)
		assert.Equal(t, "CREATED", result.Action)
		assert.Equal(t, int64(len([]byte(unicodeContent))), result.BytesWritten)

		mockFS.AssertExpectations(t)
		mockValidator.AssertExpectations(t)
	})

	t.Run("very long filename", func(t *testing.T) {
		mockFS = &MockFileSystem{}
		mockValidator = &MockPathValidator{}
		handler = NewFileHandler(mockFS, mockValidator)

		longFilename := strings.Repeat("very-long-filename-", 10) + ".txt"

		mockValidator.On("ValidatePath", longFilename, "/repo", []string{}).
			Return("/repo/"+longFilename, nil)
		mockFS.On("ReadFile", "/repo/"+longFilename).Return([]byte("content"), nil)
		mockFS.On("Stat", "/repo/"+longFilename).Return(MockFileInfo{
			name: longFilename,
			size: 7,
		}, nil)

		content, err := handler.ReadFile(longFilename, 1048576, "/repo", []string{})
		assert.NoError(t, err)
		assert.Equal(t, "content", content)

		mockFS.AssertExpectations(t)
		mockValidator.AssertExpectations(t)
	})
}

// TestFileHandlerConcurrency tests concurrent file operations
func TestFileHandlerConcurrency(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "filehandler-concurrent-test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	fs := NewRealFileSystem()
	validator := NewDefaultPathValidator()
	handler := NewFileHandler(fs, validator)

	const numGoroutines = 10
	done := make(chan bool, numGoroutines)

	// Test concurrent reads and writes
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer func() { done <- true }()

			filename := fmt.Sprintf("concurrent_%d.txt", id)
			content := fmt.Sprintf("Content for file %d\nGenerated concurrently", id)

			// Write file
			result, err := handler.WriteFile(filename, content, 102400, tempDir, []string{".txt"}, []string{}, false)
			assert.NoError(t, err)
			assert.Equal(t, "CREATED", result.Action)

			// Read file back
			readContent, err := handler.ReadFile(filename, 1048576, tempDir, []string{})
			assert.NoError(t, err)
			assert.Equal(t, content, readContent)
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < numGoroutines; i++ {
		<-done
	}

	// Verify all files were created
	files, err := os.ReadDir(tempDir)
	assert.NoError(t, err)
	assert.Len(t, files, numGoroutines)
}

// BenchmarkFileHandler benchmarks file handler performance
func BenchmarkFileHandler(b *testing.B) {
	tempDir, err := os.MkdirTemp("", "filehandler-benchmark")
	if err != nil {
		b.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	fs := NewRealFileSystem()
	validator := NewDefaultPathValidator()
	handler := NewFileHandler(fs, validator)

	// Create test file for read benchmark
	testContent := strings.Repeat("benchmark test content\n", 100)
	testFile := filepath.Join(tempDir, "benchmark.txt")
	err = os.WriteFile(testFile, []byte(testContent), 0644)
	if err != nil {
		b.Fatal(err)
	}

	b.Run("ReadFile", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := handler.ReadFile("benchmark.txt", 1048576, tempDir, []string{})
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("WriteFile", func(b *testing.B) {
		writeContent := "benchmark write content"
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			filename := fmt.Sprintf("write_bench_%d.txt", i)
			_, err := handler.WriteFile(filename, writeContent, 102400, tempDir, []string{".txt"}, []string{}, false)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

// Placeholder implementations for testing
func NewFileHandler(fs FileSystem, validator PathValidator) FileHandler {
	return &defaultFileHandler{
		filesystem: fs,
		validator:  validator,
	}
}

func NewRealFileSystem() FileSystem {
	return &realFileSystem{}
}

func NewDefaultPathValidator() PathValidator {
	return &defaultPathValidator{}
}

// Interface definitions for testing
type FileSystem interface {
	ReadFile(filename string) ([]byte, error)
	WriteFile(filename string, data []byte, perm os.FileMode) error
	Stat(filename string) (os.FileInfo, error)
	Exists(filename string) bool
}

type PathValidator interface {
	ValidatePath(requestedPath, repositoryRoot string, excludedPaths []string) (string, error)
	ValidateWriteExtension(filepath string, allowedExtensions []string) error
}

type FileHandler interface {
	ReadFile(filePath string, maxSize int64, repoRoot string, excludedPaths []string) (string, error)
	WriteFile(filePath, content string, maxSize int64, repoRoot string, allowedExts, excludedPaths []string, backup bool) (WriteResult, error)
}

type WriteResult struct {
	Action       string // "CREATED" or "UPDATED"
	BytesWritten int64
	BackupFile   string // Path to backup file if created
}

// Mock implementations for testing
type defaultFileHandler struct {
	filesystem FileSystem
	validator  PathValidator
}

func (h *defaultFileHandler) ReadFile(filePath string, maxSize int64, repoRoot string, excludedPaths []string) (string, error) {
	// Validate path
	safePath, err := h.validator.ValidatePath(filePath, repoRoot, excludedPaths)
	if err != nil {
		return "", err
	}

	// Check file size
	info, err := h.filesystem.Stat(safePath)
	if err != nil {
		return "", err
	}

	if info.Size() > maxSize {
		return "", fmt.Errorf("file size %d exceeds maximum %d", info.Size(), maxSize)
	}

	// Read file
	data, err := h.filesystem.ReadFile(safePath)
	if err != nil {
		return "", err
	}

	return string(data), nil
}

func (h *defaultFileHandler) WriteFile(filePath, content string, maxSize int64, repoRoot string, allowedExts, excludedPaths []string, backup bool) (WriteResult, error) {
	// Validate content size
	if int64(len(content)) > maxSize {
		return WriteResult{}, fmt.Errorf("content size %d exceeds maximum %d", len(content), maxSize)
	}

	// Validate path
	safePath, err := h.validator.ValidatePath(filePath, repoRoot, excludedPaths)
	if err != nil {
		return WriteResult{}, err
	}

	// Validate extension
	err = h.validator.ValidateWriteExtension(safePath, allowedExts)
	if err != nil {
		return WriteResult{}, err
	}

	result := WriteResult{
		BytesWritten: int64(len(content)),
	}

	// Check if file exists
	exists := h.filesystem.Exists(safePath)
	if exists {
		result.Action = "UPDATED"

		// Create backup if requested
		if backup {
			originalData, err := h.filesystem.ReadFile(safePath)
			if err == nil {
				backupPath := safePath + ".backup." + time.Now().Format("20060102_150405")
				h.filesystem.WriteFile(backupPath, originalData, 0644)
				result.BackupFile = backupPath
			}
		}
	} else {
		result.Action = "CREATED"
	}

	// Write file
	err = h.filesystem.WriteFile(safePath, []byte(content), 0644)
	if err != nil {
		return WriteResult{}, err
	}

	return result, nil
}

type realFileSystem struct{}

func (fs *realFileSystem) ReadFile(filename string) ([]byte, error) {
	return os.ReadFile(filename)
}

func (fs *realFileSystem) WriteFile(filename string, data []byte, perm os.FileMode) error {
	return os.WriteFile(filename, data, perm)
}

func (fs *realFileSystem) Stat(filename string) (os.FileInfo, error) {
	return os.Stat(filename)
}

func (fs *realFileSystem) Exists(filename string) bool {
	_, err := os.Stat(filename)
	return err == nil
}

type defaultPathValidator struct{}

func (v *defaultPathValidator) ValidatePath(requestedPath, repositoryRoot string, excludedPaths []string) (string, error) {
	// Basic path validation - in real implementation this would be more comprehensive
	cleanPath := filepath.Clean(requestedPath)
	if strings.Contains(cleanPath, "..") {
		return "", fmt.Errorf("PATH_SECURITY: path traversal detected")
	}
	return filepath.Join(repositoryRoot, cleanPath), nil
}

func (v *defaultPathValidator) ValidateWriteExtension(filePath string, allowedExtensions []string) error {
	if len(allowedExtensions) == 0 {
		return nil // No restrictions
	}

	ext := filepath.Ext(filePath)
	for _, allowed := range allowedExtensions {
		if strings.EqualFold(ext, allowed) {
			return nil
		}
	}

	return fmt.Errorf("EXTENSION_DENIED: %s files not allowed", ext)
}

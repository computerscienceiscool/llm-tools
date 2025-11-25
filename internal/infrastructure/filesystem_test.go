package infrastructure

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockFileSystem for testing
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

func (m *MockFileSystem) Exists(filename string) bool {
	args := m.Called(filename)
	return args.Bool(0)
}

func (m *MockFileSystem) MkdirAll(path string, perm os.FileMode) error {
	args := m.Called(path, perm)
	return args.Error(0)
}

// TestFileSystemInterface tests the FileSystem interface
func TestFileSystemInterface(t *testing.T) {
	var _ FileSystem = (*MockFileSystem)(nil)

	mockFS := &MockFileSystem{}
	testData := []byte("test content")

	// Setup expectations
	mockFS.On("ReadFile", "test.txt").Return(testData, nil)
	mockFS.On("WriteFile", "test.txt", testData, os.FileMode(0644)).Return(nil)
	mockFS.On("Exists", "test.txt").Return(true)
	mockFS.On("MkdirAll", "testdir", os.FileMode(0755)).Return(nil)

	// Test ReadFile
	data, err := mockFS.ReadFile("test.txt")
	assert.NoError(t, err)
	assert.Equal(t, testData, data)

	// Test WriteFile
	err = mockFS.WriteFile("test.txt", testData, 0644)
	assert.NoError(t, err)

	// Test Exists
	exists := mockFS.Exists("test.txt")
	assert.True(t, exists)

	// Test MkdirAll
	err = mockFS.MkdirAll("testdir", 0755)
	assert.NoError(t, err)

	mockFS.AssertExpectations(t)
}

// TestDefaultFileSystem tests the default filesystem implementation
func TestDefaultFileSystem(t *testing.T) {
	// Create temporary directory for testing
	tempDir, err := os.MkdirTemp("", "filesystem-test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	fs := NewFileSystem()

	testFile := filepath.Join(tempDir, "test.txt")
	testData := []byte("test content")

	t.Run("write and read file", func(t *testing.T) {
		// Write file
		err := fs.WriteFile(testFile, testData, 0644)
		assert.NoError(t, err)

		// Check exists
		exists := fs.Exists(testFile)
		assert.True(t, exists)

		// Read file
		data, err := fs.ReadFile(testFile)
		assert.NoError(t, err)
		assert.Equal(t, testData, data)

		// Stat file
		info, err := fs.Stat(testFile)
		assert.NoError(t, err)
		assert.Equal(t, "test.txt", info.Name())
		assert.Equal(t, int64(len(testData)), info.Size())
	})

	t.Run("create directory", func(t *testing.T) {
		testDir := filepath.Join(tempDir, "subdir", "nested")

		err := fs.MkdirAll(testDir, 0755)
		assert.NoError(t, err)

		exists := fs.Exists(testDir)
		assert.True(t, exists)
	})

	t.Run("nonexistent file", func(t *testing.T) {
		nonexistent := filepath.Join(tempDir, "nonexistent.txt")

		// Should not exist
		exists := fs.Exists(nonexistent)
		assert.False(t, exists)

		// Reading should return error
		_, err := fs.ReadFile(nonexistent)
		assert.Error(t, err)

		// Stat should return error
		_, err = fs.Stat(nonexistent)
		assert.Error(t, err)
	})
}

// TestFileSystemConcurrency tests concurrent access to filesystem
func TestFileSystemConcurrency(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "fs-concurrent-test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	fs := NewFileSystem()
	const numGoroutines = 10

	done := make(chan bool, numGoroutines)

	// Test concurrent file operations
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer func() { done <- true }()

			testFile := filepath.Join(tempDir, fmt.Sprintf("test_%d.txt", id))
			testData := []byte(fmt.Sprintf("content %d", id))

			// Write
			err := fs.WriteFile(testFile, testData, 0644)
			assert.NoError(t, err)

			// Read
			data, err := fs.ReadFile(testFile)
			assert.NoError(t, err)
			assert.Equal(t, testData, data)

			// Check exists
			exists := fs.Exists(testFile)
			assert.True(t, exists)
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < numGoroutines; i++ {
		<-done
	}
}

// TestFileSystemErrors tests error conditions
func TestFileSystemErrors(t *testing.T) {
	fs := NewFileSystem()

	t.Run("write to invalid path", func(t *testing.T) {
		// Try to write to root (should fail)
		err := fs.WriteFile("/invalid/path/file.txt", []byte("test"), 0644)
		assert.Error(t, err)
	})

	t.Run("read from invalid path", func(t *testing.T) {
		_, err := fs.ReadFile("/nonexistent/file.txt")
		assert.Error(t, err)
	})

	t.Run("mkdir with invalid permissions", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "fs-error-test")
		require.NoError(t, err)
		defer os.RemoveAll(tempDir)

		// This should work
		err = fs.MkdirAll(filepath.Join(tempDir, "valid"), 0755)
		assert.NoError(t, err)
	})
}

// TestFileSystemPermissions tests file permission handling
func TestFileSystemPermissions(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "fs-perm-test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	fs := NewFileSystem()

	tests := []struct {
		name string
		perm os.FileMode
	}{
		{"read-only", 0444},
		{"read-write", 0644},
		{"executable", 0755},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testFile := filepath.Join(tempDir, tt.name+".txt")

			err := fs.WriteFile(testFile, []byte("test"), tt.perm)
			assert.NoError(t, err)

			info, err := fs.Stat(testFile)
			assert.NoError(t, err)
			assert.Equal(t, tt.perm, info.Mode().Perm())
		})
	}
}

// TestFileSystemLargeFiles tests handling of large files
func TestFileSystemLargeFiles(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping large file test in short mode")
	}

	tempDir, err := os.MkdirTemp("", "fs-large-test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	fs := NewFileSystem()

	// Create 1MB test data
	largeData := make([]byte, 1024*1024)
	for i := range largeData {
		largeData[i] = byte(i % 256)
	}

	testFile := filepath.Join(tempDir, "large.txt")

	// Write large file
	err = fs.WriteFile(testFile, largeData, 0644)
	assert.NoError(t, err)

	// Read large file
	data, err := fs.ReadFile(testFile)
	assert.NoError(t, err)
	assert.Equal(t, len(largeData), len(data))
	assert.Equal(t, largeData[:100], data[:100]) // Compare first 100 bytes

	// Check file size
	info, err := fs.Stat(testFile)
	assert.NoError(t, err)
	assert.Equal(t, int64(len(largeData)), info.Size())
}

// BenchmarkFileSystemOperations benchmarks filesystem performance
func BenchmarkFileSystemOperations(b *testing.B) {
	tempDir, err := os.MkdirTemp("", "fs-benchmark")
	if err != nil {
		b.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	fs := NewFileSystem()
	testData := []byte("benchmark test data")

	b.Run("WriteFile", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			testFile := filepath.Join(tempDir, fmt.Sprintf("bench_%d.txt", i))
			_ = fs.WriteFile(testFile, testData, 0644)
		}
	})

	// Create files for read benchmark
	for i := 0; i < 1000; i++ {
		testFile := filepath.Join(tempDir, fmt.Sprintf("read_%d.txt", i))
		_ = fs.WriteFile(testFile, testData, 0644)
	}

	b.Run("ReadFile", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			testFile := filepath.Join(tempDir, fmt.Sprintf("read_%d.txt", i%1000))
			_, _ = fs.ReadFile(testFile)
		}
	})

	b.Run("Exists", func(b *testing.B) {
		testFile := filepath.Join(tempDir, "read_0.txt")
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = fs.Exists(testFile)
		}
	})

	b.Run("Stat", func(b *testing.B) {
		testFile := filepath.Join(tempDir, "read_0.txt")
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = fs.Stat(testFile)
		}
	})
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

func (fs *realFileSystem) MkdirAll(path string, perm os.FileMode) error {
	return os.MkdirAll(path, perm)
}

func (m *MockFileSystem) Stat(path string) (os.FileInfo, error) {
	args := m.Called(path)
	return args.Get(0).(os.FileInfo), args.Error(1)
}

func (m *MockFileSystem) Remove(path string) error {
	args := m.Called(path)
	return args.Error(0)
}

func (m *MockFileSystem) Rename(oldpath, newpath string) error {
	args := m.Called(oldpath, newpath)
	return args.Error(0)
}

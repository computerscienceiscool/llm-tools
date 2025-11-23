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

// MockSearchIndex for testing
type MockSearchIndex struct {
	mock.Mock
}

func (m *MockSearchIndex) Search(query string, limit int) ([]SearchResult, error) {
	args := m.Called(query, limit)
	return args.Get(0).([]SearchResult), args.Error(1)
}

func (m *MockSearchIndex) IndexFile(filepath, content string) error {
	args := m.Called(filepath, content)
	return args.Error(0)
}

func (m *MockSearchIndex) RemoveFile(filepath string) error {
	args := m.Called(filepath)
	return args.Error(0)
}

func (m *MockSearchIndex) IsReady() bool {
	args := m.Called()
	return args.Bool(0)
}

// TestDefaultSearchHandler tests the default search handler implementation
func TestDefaultSearchHandler(t *testing.T) {
	mockIndex := &MockSearchIndex{}
	handler := NewSearchHandler(mockIndex)
	require.NotNil(t, handler)

	t.Run("successful search", func(t *testing.T) {
		query := "function main"
		expectedResults := []SearchResult{
			{
				FilePath: "main.go",
				Score:    0.95,
				Lines:    25,
				Size:     1024,
				Preview:  "func main() {",
				ModTime:  time.Now(),
			},
			{
				FilePath: "cmd/main.go",
				Score:    0.85,
				Lines:    50,
				Size:     2048,
				Preview:  "package main",
				ModTime:  time.Now().Add(-time.Hour),
			},
		}

		mockIndex.On("IsReady").Return(true)
		mockIndex.On("Search", query, 10).Return(expectedResults, nil)

		results, err := handler.Search(query, 10)
		assert.NoError(t, err)
		assert.Len(t, results, 2)
		assert.Equal(t, "main.go", results[0].FilePath)
		assert.Greater(t, results[0].Score, results[1].Score) // Should be sorted by score

		mockIndex.AssertExpectations(t)
	})

	t.Run("search index not ready", func(t *testing.T) {
		mockIndex = &MockSearchIndex{}
		handler = NewSearchHandler(mockIndex)

		mockIndex.On("IsReady").Return(false)

		_, err := handler.Search("test", 10)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "search index not ready")

		mockIndex.AssertExpectations(t)
	})

	t.Run("empty query", func(t *testing.T) {
		_, err := handler.Search("", 10)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "empty")
	})

	t.Run("search returns no results", func(t *testing.T) {
		mockIndex = &MockSearchIndex{}
		handler = NewSearchHandler(mockIndex)

		mockIndex.On("IsReady").Return(true)
		mockIndex.On("Search", "nonexistent", 10).Return([]SearchResult{}, nil)

		results, err := handler.Search("nonexistent", 10)
		assert.NoError(t, err)
		assert.Empty(t, results)

		mockIndex.AssertExpectations(t)
	})
}

// TestSearchHandlerIndexing tests file indexing functionality
func TestSearchHandlerIndexing(t *testing.T) {
	mockIndex := &MockSearchIndex{}
	handler := NewSearchHandler(mockIndex)

	t.Run("index file", func(t *testing.T) {
		filePath := "src/example.go"
		content := "package main\n\nfunc main() {\n\tfmt.Println(\"Hello\")\n}"

		mockIndex.On("IndexFile", filePath, content).Return(nil)

		err := handler.IndexFile(filePath, content)
		assert.NoError(t, err)

		mockIndex.AssertExpectations(t)
	})

	t.Run("remove file from index", func(t *testing.T) {
		filePath := "src/deleted.go"

		mockIndex.On("RemoveFile", filePath).Return(nil)

		err := handler.RemoveFile(filePath)
		assert.NoError(t, err)

		mockIndex.AssertExpectations(t)
	})

	t.Run("index multiple files", func(t *testing.T) {
		mockIndex = &MockSearchIndex{}
		handler = NewSearchHandler(mockIndex)

		files := map[string]string{
			"main.go":   "package main\nfunc main() {}",
			"utils.go":  "package main\nfunc helper() {}",
			"README.md": "# Project Title\nThis is a test project",
		}

		for path, content := range files {
			mockIndex.On("IndexFile", path, content).Return(nil)
		}

		for path, content := range files {
			err := handler.IndexFile(path, content)
			assert.NoError(t, err)
		}

		mockIndex.AssertExpectations(t)
	})
}

// TestSearchHandlerFiltering tests search filtering and ranking
func TestSearchHandlerFiltering(t *testing.T) {
	mockIndex := &MockSearchIndex{}
	handler := NewSearchHandler(mockIndex)

	t.Run("filter by file extension", func(t *testing.T) {
		query := "function"
		allResults := []SearchResult{
			{FilePath: "main.go", Score: 0.9},
			{FilePath: "script.py", Score: 0.85},
			{FilePath: "README.md", Score: 0.8},
			{FilePath: "test.go", Score: 0.75},
		}

		mockIndex.On("IsReady").Return(true)
		mockIndex.On("Search", query, 10).Return(allResults, nil)

		// Get results and filter .go files
		results, err := handler.Search(query, 10)
		assert.NoError(t, err)

		goFiles := filterByExtension(results, ".go")
		assert.Len(t, goFiles, 2)
		assert.Equal(t, "main.go", goFiles[0].FilePath)
		assert.Equal(t, "test.go", goFiles[1].FilePath)

		mockIndex.AssertExpectations(t)
	})

	t.Run("limit results", func(t *testing.T) {
		mockIndex = &MockSearchIndex{}
		handler = NewSearchHandler(mockIndex)

		query := "test"
		manyResults := make([]SearchResult, 20)
		for i := 0; i < 20; i++ {
			manyResults[i] = SearchResult{
				FilePath: fmt.Sprintf("test_%d.go", i),
				Score:    0.9 - float64(i)*0.01, // Decreasing scores
			}
		}

		mockIndex.On("IsReady").Return(true)
		mockIndex.On("Search", query, 5).Return(manyResults[:5], nil)

		results, err := handler.Search(query, 5)
		assert.NoError(t, err)
		assert.Len(t, results, 5)

		// Should be sorted by score (highest first)
		for i := 1; i < len(results); i++ {
			assert.GreaterOrEqual(t, results[i-1].Score, results[i].Score)
		}

		mockIndex.AssertExpectations(t)
	})
}

// TestSearchHandlerRealIndex tests with a real search index implementation
func TestSearchHandlerRealIndex(t *testing.T) {
	// Create temporary directory with test files
	tempDir, err := os.MkdirTemp("", "search-test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create test files
	testFiles := map[string]string{
		"main.go": `package main

import "fmt"

func main() {
	fmt.Println("Hello, World!")
}

func calculate() int {
	return 42
}`,
		"utils.go": `package main

import "strings"

func stringHelper(s string) string {
	return strings.ToUpper(s)
}

func calculate() float64 {
	return 3.14
}`,
		"README.md": `# Test Project

This is a test project for search functionality.

## Features
- String processing
- Mathematical calculations
`,
	}

	for filename, content := range testFiles {
		fullPath := filepath.Join(tempDir, filename)
		err := os.WriteFile(fullPath, []byte(content), 0644)
		require.NoError(t, err)
	}

	// Use real index implementation (mock for this test)
	index := NewInMemorySearchIndex()
	handler := NewSearchHandler(index)

	// Index all files
	for filename, content := range testFiles {
		err := handler.IndexFile(filename, content)
		assert.NoError(t, err)
	}

	t.Run("search for function", func(t *testing.T) {
		results, err := handler.Search("function", 10)
		assert.NoError(t, err)
		// Should find references in .go files
		assert.NotEmpty(t, results)
	})

	t.Run("search for calculate", func(t *testing.T) {
		results, err := handler.Search("calculate", 10)
		assert.NoError(t, err)

		// Should find both calculate functions
		calculateFiles := make([]string, 0)
		for _, result := range results {
			calculateFiles = append(calculateFiles, result.FilePath)
		}
		assert.Contains(t, calculateFiles, "main.go")
		assert.Contains(t, calculateFiles, "utils.go")
	})

	t.Run("search for string", func(t *testing.T) {
		results, err := handler.Search("string", 10)
		assert.NoError(t, err)

		// Should find in multiple files
		assert.NotEmpty(t, results)
	})
}

// TestSearchHandlerConcurrency tests concurrent search operations
func TestSearchHandlerConcurrency(t *testing.T) {
	mockIndex := &MockSearchIndex{}
	handler := NewSearchHandler(mockIndex)

	const numGoroutines = 10
	done := make(chan bool, numGoroutines)

	// Setup mock expectations for concurrent calls
	mockIndex.On("IsReady").Return(true)
	mockIndex.On("Search", mock.AnythingOfType("string"), 10).
		Return([]SearchResult{{FilePath: "test.go", Score: 0.9}}, nil)

	// Run concurrent searches
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer func() { done <- true }()

			query := fmt.Sprintf("test %d", id)
			results, err := handler.Search(query, 10)
			assert.NoError(t, err)
			assert.NotEmpty(t, results)
		}(i)
	}

	// Wait for all to complete
	for i := 0; i < numGoroutines; i++ {
		<-done
	}

	mockIndex.AssertExpectations(t)
}

// TestSearchHandlerEdgeCases tests edge cases
func TestSearchHandlerEdgeCases(t *testing.T) {
	mockIndex := &MockSearchIndex{}
	handler := NewSearchHandler(mockIndex)

	t.Run("very long query", func(t *testing.T) {
		longQuery := strings.Repeat("very long query ", 100)

		mockIndex.On("IsReady").Return(true)
		mockIndex.On("Search", longQuery, 10).Return([]SearchResult{}, nil)

		results, err := handler.Search(longQuery, 10)
		assert.NoError(t, err)
		assert.Empty(t, results)

		mockIndex.AssertExpectations(t)
	})

	t.Run("unicode query", func(t *testing.T) {
		unicodeQuery := "测试 search 🔍"

		mockIndex.On("IsReady").Return(true)
		mockIndex.On("Search", unicodeQuery, 10).Return([]SearchResult{}, nil)

		results, err := handler.Search(unicodeQuery, 10)
		assert.NoError(t, err)
		assert.Empty(t, results)

		mockIndex.AssertExpectations(t)
	})

	t.Run("special characters in query", func(t *testing.T) {
		specialQuery := "func main() {"

		mockIndex.On("IsReady").Return(true)
		mockIndex.On("Search", specialQuery, 10).Return([]SearchResult{
			{FilePath: "main.go", Score: 0.95, Preview: "func main() {"},
		}, nil)

		results, err := handler.Search(specialQuery, 10)
		assert.NoError(t, err)
		assert.Len(t, results, 1)

		mockIndex.AssertExpectations(t)
	})
}

// BenchmarkSearchHandler benchmarks search performance
func BenchmarkSearchHandler(b *testing.B) {
	mockIndex := &MockSearchIndex{}
	handler := NewSearchHandler(mockIndex)

	mockIndex.On("IsReady").Return(true)
	mockIndex.On("Search", "benchmark query", 10).
		Return([]SearchResult{{FilePath: "test.go", Score: 0.9}}, nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := handler.Search("benchmark query", 10)
		if err != nil {
			b.Fatal(err)
		}
	}
}

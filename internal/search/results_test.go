package search

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFormatSearchResults(t *testing.T) {
	tests := []struct {
		name       string
		results    []SearchResult
		query      string
		maxResults int
		contains   []string
	}{
		{
			name:       "empty results",
			results:    []SearchResult{},
			query:      "test query",
			maxResults: 10,
			contains:   []string{"No results found", "test query"},
		},
		{
			name: "single result",
			results: []SearchResult{
				{FilePath: "main.go", Score: 0.85, Preview: "package main", LineCount: 10, FileSize: 256},
			},
			query:      "main",
			maxResults: 10,
			contains:   []string{"Search results for: main", "Found 1 matching files", "main.go", "85.00%", "256 B", "Lines: 10"},
		},
		{
			name: "multiple results",
			results: []SearchResult{
				{FilePath: "file1.go", Score: 0.9, LineCount: 100, FileSize: 1024},
				{FilePath: "file2.go", Score: 0.8, LineCount: 50, FileSize: 512},
				{FilePath: "file3.go", Score: 0.7, LineCount: 25, FileSize: 256},
			},
			query:      "search",
			maxResults: 10,
			contains:   []string{"Found 3 matching files", "file1.go", "file2.go", "file3.go"},
		},
		{
			name: "results truncated by maxResults",
			results: []SearchResult{
				{FilePath: "file1.go", Score: 0.9, LineCount: 10, FileSize: 100},
				{FilePath: "file2.go", Score: 0.8, LineCount: 10, FileSize: 100},
				{FilePath: "file3.go", Score: 0.7, LineCount: 10, FileSize: 100},
			},
			query:      "test",
			maxResults: 2,
			contains:   []string{"Found 3 matching files", "file1.go", "file2.go", "... and 1 more results"},
		},
		{
			name: "result with preview",
			results: []SearchResult{
				{FilePath: "main.go", Score: 0.85, Preview: "  func main() {\n    // code\n  }", LineCount: 10, FileSize: 256},
			},
			query:      "main",
			maxResults: 10,
			contains:   []string{"Preview:", "func main()"},
		},
		{
			name: "maxResults zero shows all",
			results: []SearchResult{
				{FilePath: "file1.go", Score: 0.9, LineCount: 10, FileSize: 100},
				{FilePath: "file2.go", Score: 0.8, LineCount: 10, FileSize: 100},
			},
			query:      "test",
			maxResults: 0,
			contains:   []string{"file1.go", "file2.go"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatSearchResults(tt.results, tt.query, tt.maxResults)

			for _, expected := range tt.contains {
				if !strings.Contains(result, expected) {
					t.Errorf("expected result to contain %q, got:\n%s", expected, result)
				}
			}
		})
	}
}

func TestFormatSearchResults_NotContains(t *testing.T) {
	results := []SearchResult{
		{FilePath: "file1.go", Score: 0.9, LineCount: 10, FileSize: 100},
		{FilePath: "file2.go", Score: 0.8, LineCount: 10, FileSize: 100},
		{FilePath: "file3.go", Score: 0.7, LineCount: 10, FileSize: 100},
	}

	output := FormatSearchResults(results, "test", 2)

	if strings.Contains(output, "file3.go") {
		t.Error("file3.go should not appear when maxResults=2")
	}
}

func TestFormatFileSize(t *testing.T) {
	tests := []struct {
		name     string
		size     int64
		expected string
	}{
		{"zero bytes", 0, "0 B"},
		{"small bytes", 100, "100 B"},
		{"one KB", 1024, "1.00 KB"},
		{"kilobytes", 2560, "2.50 KB"},
		{"one MB", 1024 * 1024, "1.00 MB"},
		{"megabytes", 1536 * 1024, "1.50 MB"},
		{"one GB", 1024 * 1024 * 1024, "1.00 GB"},
		{"gigabytes", int64(2.5 * 1024 * 1024 * 1024), "2.50 GB"},
		{"just under KB", 1023, "1023 B"},
		{"just under MB", 1024*1024 - 1, "1024.00 KB"},
		{"just under GB", 1024*1024*1024 - 1, "1024.00 MB"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatFileSize(tt.size)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestRankSearchResults(t *testing.T) {
	tests := []struct {
		name     string
		results  []SearchResult
		expected []float32
	}{
		{
			name:     "empty results",
			results:  []SearchResult{},
			expected: []float32{},
		},
		{
			name: "already sorted",
			results: []SearchResult{
				{FilePath: "a.go", Score: 0.9},
				{FilePath: "b.go", Score: 0.8},
				{FilePath: "c.go", Score: 0.7},
			},
			expected: []float32{0.9, 0.8, 0.7},
		},
		{
			name: "reverse sorted",
			results: []SearchResult{
				{FilePath: "a.go", Score: 0.7},
				{FilePath: "b.go", Score: 0.8},
				{FilePath: "c.go", Score: 0.9},
			},
			expected: []float32{0.9, 0.8, 0.7},
		},
		{
			name: "unsorted",
			results: []SearchResult{
				{FilePath: "a.go", Score: 0.5},
				{FilePath: "b.go", Score: 0.9},
				{FilePath: "c.go", Score: 0.7},
				{FilePath: "d.go", Score: 0.3},
			},
			expected: []float32{0.9, 0.7, 0.5, 0.3},
		},
		{
			name: "equal scores",
			results: []SearchResult{
				{FilePath: "a.go", Score: 0.8},
				{FilePath: "b.go", Score: 0.8},
				{FilePath: "c.go", Score: 0.8},
			},
			expected: []float32{0.8, 0.8, 0.8},
		},
		{
			name: "single result",
			results: []SearchResult{
				{FilePath: "a.go", Score: 0.5},
			},
			expected: []float32{0.5},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rankSearchResults(tt.results)

			if len(tt.results) != len(tt.expected) {
				t.Fatalf("expected %d results, got %d", len(tt.expected), len(tt.results))
			}

			for i, expectedScore := range tt.expected {
				if tt.results[i].Score != expectedScore {
					t.Errorf("result %d: expected score %f, got %f", i, expectedScore, tt.results[i].Score)
				}
			}
		})
	}
}

func TestGetRelevanceLabel(t *testing.T) {
	tests := []struct {
		score    float32
		expected string
	}{
		{1.0, "Excellent"},
		{0.95, "Excellent"},
		{0.9, "Excellent"},
		{0.89, "Very Good"},
		{0.85, "Very Good"},
		{0.8, "Very Good"},
		{0.79, "Good"},
		{0.75, "Good"},
		{0.7, "Good"},
		{0.69, "Fair"},
		{0.65, "Fair"},
		{0.6, "Fair"},
		{0.59, "Marginal"},
		{0.55, "Marginal"},
		{0.5, "Marginal"},
		{0.49, "Low"},
		{0.3, "Low"},
		{0.0, "Low"},
		{-0.1, "Low"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := GetRelevanceLabel(tt.score)
			if result != tt.expected {
				t.Errorf("GetRelevanceLabel(%f): expected %q, got %q", tt.score, tt.expected, result)
			}
		})
	}
}

func TestGetRelevanceLabel_Boundaries(t *testing.T) {
	boundaries := []struct {
		score    float32
		expected string
	}{
		{0.9, "Excellent"},
		{0.8, "Very Good"},
		{0.7, "Good"},
		{0.6, "Fair"},
		{0.5, "Marginal"},
	}

	for _, tt := range boundaries {
		result := GetRelevanceLabel(tt.score)
		if result != tt.expected {
			t.Errorf("boundary score %f: expected %q, got %q", tt.score, tt.expected, result)
		}

		belowResult := GetRelevanceLabel(tt.score - 0.001)
		if belowResult == tt.expected {
			t.Errorf("score just below %f should not be %q", tt.score, tt.expected)
		}
	}
}

func TestCountLines(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name     string
		content  string
		expected int
	}{
		{"single line", "hello world", 1},
		{"two lines", "line1\nline2", 2},
		{"three lines", "line1\nline2\nline3", 3},
		{"empty file", "", 1},
		{"trailing newline", "line1\nline2\n", 3},
		{"multiple empty lines", "\n\n\n", 4},
		{"windows line endings", "line1\r\nline2\r\n", 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filePath := filepath.Join(tmpDir, "test.txt")
			err := os.WriteFile(filePath, []byte(tt.content), 0644)
			if err != nil {
				t.Fatalf("failed to write test file: %v", err)
			}

			result := countLines(filePath)
			if result != tt.expected {
				t.Errorf("expected %d lines, got %d", tt.expected, result)
			}
		})
	}
}

func TestCountLines_NonexistentFile(t *testing.T) {
	result := countLines("/nonexistent/path/file.txt")
	if result != 0 {
		t.Errorf("expected 0 for nonexistent file, got %d", result)
	}
}

func TestGeneratePreview(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name      string
		content   string
		maxLength int
		contains  []string
	}{
		{
			name:      "short content",
			content:   "Hello World",
			maxLength: 100,
			contains:  []string{"Hello World"},
		},
		{
			name:      "content with indentation",
			content:   "Line 1\nLine 2",
			maxLength: 100,
			contains:  []string{"  Line 1", "  Line 2"},
		},
		{
			name:      "truncated content",
			content:   "This is a very long line that should be truncated at some point",
			maxLength: 20,
			contains:  []string{"..."},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filePath := "test.txt"
			fullPath := filepath.Join(tmpDir, filePath)
			err := os.WriteFile(fullPath, []byte(tt.content), 0644)
			if err != nil {
				t.Fatalf("failed to write test file: %v", err)
			}

			result := generatePreview(tmpDir, filePath, tt.maxLength)

			for _, expected := range tt.contains {
				if !strings.Contains(result, expected) {
					t.Errorf("expected preview to contain %q, got:\n%s", expected, result)
				}
			}
		})
	}
}

func TestGeneratePreview_NonexistentFile(t *testing.T) {
	result := generatePreview("/nonexistent", "file.txt", 100)
	if result != "" {
		t.Errorf("expected empty string for nonexistent file, got %q", result)
	}
}

func TestGeneratePreview_TruncationAtNewline(t *testing.T) {
	tmpDir := t.TempDir()

	content := "Line 1\nLine 2\nLine 3\nLine 4\nLine 5"
	filePath := filepath.Join(tmpDir, "test.txt")
	err := os.WriteFile(filePath, []byte(content), 0644)
	if err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	result := generatePreview(tmpDir, "test.txt", 15)

	if !strings.Contains(result, "...") {
		t.Error("expected truncation marker ...")
	}
}

func TestGeneratePreview_EmptyFile(t *testing.T) {
	tmpDir := t.TempDir()

	filePath := filepath.Join(tmpDir, "empty.txt")
	err := os.WriteFile(filePath, []byte(""), 0644)
	if err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	result := generatePreview(tmpDir, "empty.txt", 100)
	// Empty file after trim results in empty string, split gives [""], each line gets "  " prefix
	if result != "  " {
		t.Errorf("expected %q for empty file, got %q", "  ", result)
	}
}

func BenchmarkFormatSearchResults(b *testing.B) {
	results := make([]SearchResult, 100)
	for i := range results {
		results[i] = SearchResult{
			FilePath:  "file.go",
			Score:     float32(i) / 100,
			Preview:   "preview content",
			LineCount: 100,
			FileSize:  1024,
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		FormatSearchResults(results, "query", 10)
	}
}

func BenchmarkRankSearchResults(b *testing.B) {
	results := make([]SearchResult, 1000)
	for i := range results {
		results[i] = SearchResult{
			FilePath: "file.go",
			Score:    float32(i%100) / 100,
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cp := make([]SearchResult, len(results))
		for j := range results {
			cp[j] = results[j]
		}
		rankSearchResults(cp)
	}
}

func BenchmarkGetRelevanceLabel(b *testing.B) {
	scores := []float32{0.95, 0.85, 0.75, 0.65, 0.55, 0.45}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		GetRelevanceLabel(scores[i%len(scores)])
	}
}

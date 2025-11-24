package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Test search command parsing
func TestParseSearchCommand(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Simple search command",
			input:    "Let me find <search authentication logic>",
			expected: "authentication logic",
		},
		{
			name:     "Search with multiple words",
			input:    "Looking for <search database connection handling>",
			expected: "database connection handling",
		},
		{
			name:     "No search command",
			input:    "This is just regular text",
			expected: "",
		},
		{
			name:     "Search with special characters",
			input:    "Find <search error-handling in API>",
			expected: "error-handling in API",
		},
		{
			name:     "Multiple search commands (first one)",
			input:    "First <search auth> then <search database>",
			expected: "auth",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseSearchCommand(tt.input)
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

// Test search configuration
func TestSearchConfig(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "search-config-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Test default configuration
	config := getDefaultSearchConfig()
	if config.Enabled {
		t.Error("Default config should have search disabled")
	}
	if config.EmbeddingModel != "all-MiniLM-L6-v2" {
		t.Errorf("Expected all-MiniLM-L6-v2, got %s", config.EmbeddingModel)
	}

	// Test configuration file creation
	configPath := filepath.Join(tempDir, "test-config.yaml")
	if err := UpdateSearchConfigInFile(configPath, config); err != nil {
		t.Errorf("Failed to save config: %v", err)
	}

	// Test loading configuration
	loadedConfig, err := LoadConfig(configPath)
	if err != nil {
		t.Errorf("Failed to load config: %v", err)
	}

	if loadedConfig.Commands.Search.EmbeddingModel != config.EmbeddingModel {
		t.Errorf("Config not loaded correctly")
	}
}

// Test embedding serialization
func TestEmbeddingSerialization(t *testing.T) {
	// Create test embedding
	original := make([]float32, embeddingDimensions)
	for i := range original {
		original[i] = float32(i) * 0.1
	}

	// Serialize
	serialized := serializeEmbedding(original)
	if len(serialized) != embeddingDimensions*4 {
		t.Errorf("Expected %d bytes, got %d", embeddingDimensions*4, len(serialized))
	}

	// Deserialize
	deserialized := deserializeEmbedding(serialized)
	if len(deserialized) != len(original) {
		t.Errorf("Length mismatch after deserialization")
	}

	// Check values
	for i := range original {
		if deserialized[i] != original[i] {
			t.Errorf("Value mismatch at index %d: expected %f, got %f",
				i, original[i], deserialized[i])
			break
		}
	}
}

// Test cosine similarity calculation
func TestCosineSimilarity(t *testing.T) {
	tests := []struct {
		name      string
		a, b      []float32
		expected  float32
		tolerance float32
	}{
		{
			name:      "Identical vectors",
			a:         []float32{1, 2, 3},
			b:         []float32{1, 2, 3},
			expected:  1.0,
			tolerance: 0.001,
		},
		{
			name:      "Orthogonal vectors",
			a:         []float32{1, 0, 0},
			b:         []float32{0, 1, 0},
			expected:  0.0,
			tolerance: 0.001,
		},
		{
			name:      "Opposite vectors",
			a:         []float32{1, 1, 1},
			b:         []float32{-1, -1, -1},
			expected:  -1.0,
			tolerance: 0.001,
		},
		{
			name:      "Similar vectors",
			a:         []float32{1, 2, 3},
			b:         []float32{1, 2, 4},
			expected:  0.992,
			tolerance: 0.01,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cosineSimilarity(tt.a, tt.b)
			diff := result - tt.expected
			if diff < 0 {
				diff = -diff
			}
			if diff > tt.tolerance {
				t.Errorf("Expected %f (±%f), got %f", tt.expected, tt.tolerance, result)
			}
		})
	}
}

// Test file indexing logic
func TestShouldIndexFile(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "index-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Create session and search engine for testing
	config := &Config{
		RepositoryRoot: tempDir,
		ExcludedPaths:  []string{".git", "*.key"},
	}
	session := NewSession(config)

	searchConfig := &SearchConfig{
		IndexExtensions: []string{".go", ".py", ".md"},
	}

	engine := &SearchEngine{
		config:  searchConfig,
		session: session,
	}

	tests := []struct {
		name     string
		filepath string
		expected bool
	}{
		{
			name:     "Go file (should index)",
			filepath: "main.go",
			expected: true,
		},
		{
			name:     "Python file (should index)",
			filepath: "script.py",
			expected: true,
		},
		{
			name:     "Markdown file (should index)",
			filepath: "README.md",
			expected: true,
		},
		{
			name:     "Binary file (should not index)",
			filepath: "binary.exe",
			expected: false,
		},
		{
			name:     "Key file (should not index)",
			filepath: "private.key",
			expected: false,
		},
		{
			name:     "Git file (should not index)",
			filepath: ".git/config",
			expected: false,
		},
		{
			name:     "No extension (should not index)",
			filepath: "Makefile",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := engine.shouldIndexFile(tt.filepath)
			if result != tt.expected {
				t.Errorf("Expected %v for %s, got %v", tt.expected, tt.filepath, result)
			}
		})
	}
}

// Test text file detection
func TestIsTextFile(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "text-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Create text file
	textFile := filepath.Join(tempDir, "text.txt")
	os.WriteFile(textFile, []byte("This is a text file"), 0644)

	// Create binary file (with null bytes)
	binaryFile := filepath.Join(tempDir, "binary.bin")
	binaryData := []byte{0x00, 0x01, 0x02, 0x00, 0x03}
	os.WriteFile(binaryFile, binaryData, 0644)

	tests := []struct {
		name     string
		filepath string
		expected bool
	}{
		{
			name:     "Text file",
			filepath: textFile,
			expected: true,
		},
		{
			name:     "Binary file",
			filepath: binaryFile,
			expected: false,
		},
		{
			name:     "Non-existent file",
			filepath: filepath.Join(tempDir, "missing.txt"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isTextFile(tt.filepath)
			if result != tt.expected {
				t.Errorf("Expected %v for %s, got %v", tt.expected, tt.filepath, result)
			}
		})
	}
}

// Test search result formatting
func TestFormatSearchResults(t *testing.T) {
	searchConfig := &SearchConfig{
		MaxResults: 10,
	}

	engine := &SearchEngine{
		config: searchConfig,
	}

	// Test empty results
	results := []SearchResult{}
	output := engine.FormatSearchResults("test query", results, time.Millisecond*100)

	if !strings.Contains(output, "No files found") {
		t.Error("Expected 'No files found' message for empty results")
	}

	// Test with results
	results = []SearchResult{
		{
			FilePath: "main.go",
			Score:    0.95,
			Lines:    100,
			Size:     1024,
			Preview:  "package main",
			ModTime:  time.Now(),
		},
		{
			FilePath: "README.md",
			Score:    0.75,
			Lines:    50,
			Size:     512,
			Preview:  "# Project Title",
			ModTime:  time.Now(),
		},
	}

	output = engine.FormatSearchResults("test query", results, time.Millisecond*100)

	expectedStrings := []string{
		"SEARCH: test query",
		"main.go (score: 0.95)",
		"README.md (score: 0.75)",
		"package main",
		"# Project Title",
		"END SEARCH",
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(output, expected) {
			t.Errorf("Expected output to contain %q", expected)
		}
	}
}

// Test file size formatting
func TestFormatFileSize(t *testing.T) {
	tests := []struct {
		size     int64
		expected string
	}{
		{512, "512 B"},
		{1024, "1.0 KB"},
		{1536, "1.5 KB"},
		{1048576, "1.0 MB"},
		{1073741824, "1.0 GB"},
	}

	for _, tt := range tests {
		result := formatFileSize(tt.size)
		if result != tt.expected {
			t.Errorf("formatFileSize(%d) = %q, expected %q", tt.size, result, tt.expected)
		}
	}
}

// Benchmark cosine similarity calculation
func BenchmarkCosineSimilarity(b *testing.B) {
	a := make([]float32, embeddingDimensions)
	c := make([]float32, embeddingDimensions)

	for i := range a {
		a[i] = float32(i) * 0.1
		c[i] = float32(i) * 0.2
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cosineSimilarity(a, c)
	}
}

// Benchmark embedding serialization
func BenchmarkEmbeddingSerialization(b *testing.B) {
	embedding := make([]float32, embeddingDimensions)
	for i := range embedding {
		embedding[i] = float32(i) * 0.1
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		serialized := serializeEmbedding(embedding)
		deserializeEmbedding(serialized)
	}
}

// Test search ranking algorithm
func TestSearchRanking(t *testing.T) {
	searchConfig := &SearchConfig{
		MaxResults: 10,
	}

	engine := &SearchEngine{
		config: searchConfig,
	}

	results := []SearchResult{
		{
			FilePath: "auth/main.go",
			Score:    0.8,
			Size:     1000,
			ModTime:  time.Now().Add(-time.Hour),
		},
		{
			FilePath: "src/auth.go",
			Score:    0.8,
			Size:     100000, // Large file
			ModTime:  time.Now().Add(-48 * time.Hour),
		},
		{
			FilePath: "main.go",
			Score:    0.8,
			Size:     2000,
			ModTime:  time.Now(), // Recent file
		},
	}

	engine.rankSearchResults(results, "auth")

	// Check that ranking algorithm modified scores appropriately
	// Files with "auth" in path should get boost
	// Recent files should get boost
	// Large files should get penalty
	// src/ directories should get boost

	found := false
	for _, result := range results {
		if result.FilePath == "main.go" {
			if result.Score <= 0.8 {
				t.Error("Recent file should have gotten a boost")
			}
			found = true
			break
		}
	}
	if !found {
		t.Error("main.go result not found")
	}
}

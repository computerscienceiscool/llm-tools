package search

import (
	"os/exec"
	"strings"
	"testing"
)

func TestGenerateEmbedding_EmptyText(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"empty string", ""},
		{"whitespace only", "   "},
		{"tabs and newlines", "\t\n\t"},
		{"mixed whitespace", "  \t\n  "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := generateEmbedding("http://localhost:11434", tt.input)
			if err != nil {
				t.Fatalf("unexpected error for empty/whitespace input: %v", err)
			}

			if len(result) != embeddingDimensions {
				t.Errorf("expected %d dimensions, got %d", embeddingDimensions, len(result))
			}

			// Verify it's a zero vector
			for i, val := range result {
				if val != 0.0 {
					t.Errorf("expected zero at index %d, got %f", i, val)
					break
				}
			}
		})
	}
}

func TestGenerateEmbedding_InvalidOllamaURL(t *testing.T) {
	_, err := generateEmbedding("http://localhost:99999", "test text")
	if err == nil {
		t.Error("expected error for invalid python path, got nil")
	}

	if !strings.Contains(err.Error(), "Ollama API") {
		t.Errorf("expected 'Ollama API' in error, got: %v", err)
	}
}

func TestGenerateEmbedding_ValidText(t *testing.T) {
	// Skip if Python or sentence-transformers not available
	if !pythonAvailable(t) {
		t.Skip("Python with sentence-transformers not available")
	}

	result, err := generateEmbedding("http://localhost:11434", "Hello world")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result) != embeddingDimensions {
		t.Errorf("expected %d dimensions, got %d", embeddingDimensions, len(result))
	}

	// Verify it's not a zero vector (actual embedding should have non-zero values)
	allZero := true
	for _, val := range result {
		if val != 0.0 {
			allZero = false
			break
		}
	}
	if allZero {
		t.Error("expected non-zero embedding for valid text")
	}
}

func TestGenerateEmbedding_DifferentTextsDifferentEmbeddings(t *testing.T) {
	// Skip if Python or sentence-transformers not available
	if !pythonAvailable(t) {
		t.Skip("Python with sentence-transformers not available")
	}

	embedding1, err := generateEmbedding("http://localhost:11434", "The cat sat on the mat")
	if err != nil {
		t.Fatalf("unexpected error for first text: %v", err)
	}

	embedding2, err := generateEmbedding("http://localhost:11434", "Quantum physics is complex")
	if err != nil {
		t.Fatalf("unexpected error for second text: %v", err)
	}

	// Embeddings should be different for different texts
	differences := 0
	for i := range embedding1 {
		if embedding1[i] != embedding2[i] {
			differences++
		}
	}

	if differences == 0 {
		t.Error("expected different embeddings for different texts")
	}
}

func TestGenerateEmbedding_SimilarTextsCloseEmbeddings(t *testing.T) {
	// Skip if Python or sentence-transformers not available
	if !pythonAvailable(t) {
		t.Skip("Python with sentence-transformers not available")
	}

	embedding1, err := generateEmbedding("http://localhost:11434", "The dog is happy")
	if err != nil {
		t.Fatalf("unexpected error for first text: %v", err)
	}

	embedding2, err := generateEmbedding("http://localhost:11434", "The dog is joyful")
	if err != nil {
		t.Fatalf("unexpected error for second text: %v", err)
	}

	// Calculate cosine similarity
	similarity := cosineSimilarity(embedding1, embedding2)

	// Similar sentences should have high similarity (> 0.7)
	if similarity < 0.7 {
		t.Errorf("expected similarity > 0.7 for similar texts, got %f", similarity)
	}
}

func TestGenerateEmbedding_NormalizedOutput(t *testing.T) {
	// Skip if Python or sentence-transformers not available
	if !pythonAvailable(t) {
		t.Skip("Python with sentence-transformers not available")
	}

	result, err := generateEmbedding("http://localhost:11434", "Test normalization")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Calculate magnitude (should be ~1.0 for normalized vectors)
	var sumSquares float64
	for _, val := range result {
		sumSquares += float64(val) * float64(val)
	}
	magnitude := sumSquares // sqrt would give ~1.0, sum of squares should be ~1.0

	// Allow small floating point tolerance
	if magnitude < 0.99 || magnitude > 1.01 {
		t.Errorf("expected normalized vector (magnitude ~1.0), got magnitude squared %f", magnitude)
	}
}

func TestGenerateEmbedding_SpecialCharacters(t *testing.T) {
	// Skip if Python or sentence-transformers not available
	if !pythonAvailable(t) {
		t.Skip("Python with sentence-transformers not available")
	}

	tests := []struct {
		name string
		text string
	}{
		{"unicode", "Hello 世界 🌍"},
		{"quotes", `He said "hello"`},
		{"newlines", "Line1\nLine2\nLine3"},
		{"tabs", "Col1\tCol2\tCol3"},
		{"special chars", "!@#$%^&*()"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := generateEmbedding("http://localhost:11434", tt.text)
			if err != nil {
				t.Fatalf("unexpected error for %s: %v", tt.name, err)
			}

			if len(result) != embeddingDimensions {
				t.Errorf("expected %d dimensions, got %d", embeddingDimensions, len(result))
			}
		})
	}
}

func TestGenerateEmbedding_LongText(t *testing.T) {
	// Skip if Python or sentence-transformers not available
	if !pythonAvailable(t) {
		t.Skip("Python with sentence-transformers not available")
	}

	// Generate a long text
	longText := strings.Repeat("This is a test sentence. ", 100)

	result, err := generateEmbedding("http://localhost:11434", longText)
	if err != nil {
		t.Fatalf("unexpected error for long text: %v", err)
	}

	if len(result) != embeddingDimensions {
		t.Errorf("expected %d dimensions, got %d", embeddingDimensions, len(result))
	}
}

// Helper function to check if Python with sentence-transformers is available
func pythonAvailable(t *testing.T) bool {
	t.Helper()

	cmd := exec.Command("http://localhost:11434", "-c", "from sentence_transformers import SentenceTransformer")
	err := cmd.Run()
	return err == nil
}

// TestTruncateText tests the truncateText function
func TestTruncateText(t *testing.T) {
	tests := []struct {
		name      string
		text      string
		maxTokens int
		expected  string
	}{
		{
			name:      "short text no truncation",
			text:      "hello world",
			maxTokens: 100,
			expected:  "hello world",
		},
		{
			name:      "exact length",
			text:      "1234",
			maxTokens: 1, // maxChars = 1 * 4 = 4
			expected:  "1234",
		},
		{
			name:      "needs truncation",
			text:      "12345678",
			maxTokens: 1, // maxChars = 1 * 4 = 4
			expected:  "1234",
		},
		{
			name:      "empty text",
			text:      "",
			maxTokens: 100,
			expected:  "",
		},
		{
			name:      "large text",
			text:      strings.Repeat("a", 1000),
			maxTokens: 10, // maxChars = 10 * 4 = 40
			expected:  strings.Repeat("a", 40),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := truncateText(tt.text, tt.maxTokens)
			if result != tt.expected {
				t.Errorf("truncateText() = %q (len=%d), want %q (len=%d)",
					result, len(result), tt.expected, len(tt.expected))
			}
		})
	}
}

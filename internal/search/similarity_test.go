package search

import (
	"math"
	"testing"
)

func TestCosineSimilarity(t *testing.T) {
	tests := []struct {
		name     string
		a        []float32
		b        []float32
		expected float32
		epsilon  float32 // tolerance for floating point comparison
	}{
		{
			name:     "identical vectors",
			a:        []float32{1.0, 2.0, 3.0},
			b:        []float32{1.0, 2.0, 3.0},
			expected: 1.0,
			epsilon:  0.0001,
		},
		{
			name:     "orthogonal vectors",
			a:        []float32{1.0, 0.0, 0.0},
			b:        []float32{0.0, 1.0, 0.0},
			expected: 0.0,
			epsilon:  0.0001,
		},
		{
			name:     "opposite vectors",
			a:        []float32{1.0, 2.0, 3.0},
			b:        []float32{-1.0, -2.0, -3.0},
			expected: -1.0,
			epsilon:  0.0001,
		},
		{
			name:     "different length vectors",
			a:        []float32{1.0, 2.0},
			b:        []float32{1.0, 2.0, 3.0},
			expected: 0.0,
			epsilon:  0.0001,
		},
		{
			name:     "empty vectors",
			a:        []float32{},
			b:        []float32{},
			expected: 0.0,
			epsilon:  0.0001,
		},
		{
			name:     "zero vector a",
			a:        []float32{0.0, 0.0, 0.0},
			b:        []float32{1.0, 2.0, 3.0},
			expected: 0.0,
			epsilon:  0.0001,
		},
		{
			name:     "zero vector b",
			a:        []float32{1.0, 2.0, 3.0},
			b:        []float32{0.0, 0.0, 0.0},
			expected: 0.0,
			epsilon:  0.0001,
		},
		{
			name:     "both zero vectors",
			a:        []float32{0.0, 0.0, 0.0},
			b:        []float32{0.0, 0.0, 0.0},
			expected: 0.0,
			epsilon:  0.0001,
		},
		{
			name:     "partial similarity",
			a:        []float32{1.0, 0.0, 0.0},
			b:        []float32{1.0, 1.0, 0.0},
			expected: float32(1.0 / math.Sqrt(2)),
			epsilon:  0.0001,
		},
		{
			name:     "unit vectors at 45 degrees",
			a:        []float32{1.0, 0.0},
			b:        []float32{float32(math.Sqrt(2) / 2), float32(math.Sqrt(2) / 2)},
			expected: float32(math.Sqrt(2) / 2),
			epsilon:  0.0001,
		},
		{
			name:     "single element vectors",
			a:        []float32{5.0},
			b:        []float32{3.0},
			expected: 1.0,
			epsilon:  0.0001,
		},
		{
			name:     "negative values",
			a:        []float32{-1.0, -2.0, -3.0},
			b:        []float32{-1.0, -2.0, -3.0},
			expected: 1.0,
			epsilon:  0.0001,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cosineSimilarity(tt.a, tt.b)
			diff := result - tt.expected
			if diff < 0 {
				diff = -diff
			}
			if diff > tt.epsilon {
				t.Errorf("expected %f, got %f (diff: %f)", tt.expected, result, diff)
			}
		})
	}
}

func TestSerializeEmbedding(t *testing.T) {
	tests := []struct {
		name      string
		embedding []float32
	}{
		{
			name:      "simple embedding",
			embedding: []float32{1.0, 2.0, 3.0},
		},
		{
			name:      "empty embedding",
			embedding: []float32{},
		},
		{
			name:      "negative values",
			embedding: []float32{-1.0, -0.5, 0.0, 0.5, 1.0},
		},
		{
			name:      "very small values",
			embedding: []float32{0.0001, 0.0002, 0.0003},
		},
		{
			name:      "very large values",
			embedding: []float32{1000000.0, 2000000.0, 3000000.0},
		},
		{
			name:      "single element",
			embedding: []float32{42.0},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			serialized := serializeEmbedding(tt.embedding)

			// Check that we got the right number of bytes (4 bytes per float32)
			expectedBytes := len(tt.embedding) * 4
			if len(serialized) != expectedBytes {
				t.Errorf("expected %d bytes, got %d", expectedBytes, len(serialized))
			}
		})
	}
}

func TestDeserializeEmbedding(t *testing.T) {
	tests := []struct {
		name        string
		data        []byte
		expectNil   bool
		expectedLen int
	}{
		{
			name:        "correct size for embeddingDimensions",
			data:        make([]byte, embeddingDimensions*4),
			expectNil:   false,
			expectedLen: embeddingDimensions,
		},
		{
			name:        "wrong size - too small",
			data:        make([]byte, 100),
			expectNil:   true,
			expectedLen: 0,
		},
		{
			name:        "wrong size - too large",
			data:        make([]byte, embeddingDimensions*4+100),
			expectNil:   true,
			expectedLen: 0,
		},
		{
			name:        "empty data",
			data:        []byte{},
			expectNil:   true,
			expectedLen: 0,
		},
		{
			name:        "nil data",
			data:        nil,
			expectNil:   true,
			expectedLen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := deserializeEmbedding(tt.data)

			if tt.expectNil {
				if result != nil {
					t.Errorf("expected nil, got slice of length %d", len(result))
				}
			} else {
				if result == nil {
					t.Error("expected non-nil result, got nil")
				} else if len(result) != tt.expectedLen {
					t.Errorf("expected length %d, got %d", tt.expectedLen, len(result))
				}
			}
		})
	}
}

func TestSerializeDeserializeRoundTrip(t *testing.T) {
	// Create a full-size embedding
	original := make([]float32, embeddingDimensions)
	for i := range original {
		original[i] = float32(i) * 0.01
	}

	// Serialize
	serialized := serializeEmbedding(original)

	// Deserialize
	deserialized := deserializeEmbedding(serialized)

	if deserialized == nil {
		t.Fatal("deserialized result is nil")
	}

	if len(deserialized) != len(original) {
		t.Fatalf("length mismatch: original %d, deserialized %d", len(original), len(deserialized))
	}

	// Compare values
	for i := range original {
		if original[i] != deserialized[i] {
			t.Errorf("value mismatch at index %d: original %f, deserialized %f", i, original[i], deserialized[i])
		}
	}
}

func TestSerializeDeserializeSpecialValues(t *testing.T) {
	// Test special float values
	original := make([]float32, embeddingDimensions)

	// Set some special values
	original[0] = 0.0
	original[1] = -0.0
	original[2] = float32(math.Inf(1))
	original[3] = float32(math.Inf(-1))
	// NaN behavior is tricky - skipping

	// Fill rest with normal values
	for i := 4; i < len(original); i++ {
		original[i] = float32(i) * 0.001
	}

	serialized := serializeEmbedding(original)
	deserialized := deserializeEmbedding(serialized)

	if deserialized == nil {
		t.Fatal("deserialized result is nil")
	}

	// Check special values preserved (except NaN)
	if deserialized[0] != 0.0 {
		t.Errorf("zero not preserved: got %f", deserialized[0])
	}
	if !math.IsInf(float64(deserialized[2]), 1) {
		t.Errorf("positive infinity not preserved: got %f", deserialized[2])
	}
	if !math.IsInf(float64(deserialized[3]), -1) {
		t.Errorf("negative infinity not preserved: got %f", deserialized[3])
	}
}

func TestEmbeddingDimensionsConstant(t *testing.T) {
	// Verify the constant is set correctly for all-MiniLM-L6-v2
	if embeddingDimensions != 384 {
		t.Errorf("expected embeddingDimensions to be 384, got %d", embeddingDimensions)
	}
}

// Benchmark tests
func BenchmarkCosineSimilarity(b *testing.B) {
	a := make([]float32, embeddingDimensions)
	vec2 := make([]float32, embeddingDimensions)
	for i := range a {
		a[i] = float32(i) * 0.01
		vec2[i] = float32(embeddingDimensions-i) * 0.01
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cosineSimilarity(a, vec2)
	}
}

func BenchmarkSerializeEmbedding(b *testing.B) {
	embedding := make([]float32, embeddingDimensions)
	for i := range embedding {
		embedding[i] = float32(i) * 0.01
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		serializeEmbedding(embedding)
	}
}

func BenchmarkDeserializeEmbedding(b *testing.B) {
	embedding := make([]float32, embeddingDimensions)
	for i := range embedding {
		embedding[i] = float32(i) * 0.01
	}
	data := serializeEmbedding(embedding)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		deserializeEmbedding(data)
	}
}

package search

import (
	"bytes"
	"encoding/binary"
	"math"
)

const embeddingDimensions = 768 // nomic-embed-text dimensions

// cosineSimilarity calculates cosine similarity between two vectors
func cosineSimilarity(a, b []float32) float32 {
	if len(a) != len(b) {
		return 0.0
	}

	var dotProduct, normA, normB float32
	for i := range a {
		dotProduct += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}

	if normA == 0 || normB == 0 {
		return 0.0
	}

	return dotProduct / (float32(math.Sqrt(float64(normA))) * float32(math.Sqrt(float64(normB))))
}

// serializeEmbedding converts float32 slice to bytes for storage
func serializeEmbedding(embedding []float32) []byte {
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.LittleEndian, embedding)
	return buf.Bytes()
}

// deserializeEmbedding converts bytes back to float32 slice
func deserializeEmbedding(data []byte) []float32 {
	if len(data) != embeddingDimensions*4 {
		return nil
	}

	embedding := make([]float32, embeddingDimensions)
	buf := bytes.NewReader(data)
	binary.Read(buf, binary.LittleEndian, &embedding)
	return embedding
}

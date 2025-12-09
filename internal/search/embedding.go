package search

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// OllamaEmbeddingRequest represents the request to Ollama API
type OllamaEmbeddingRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
}

// OllamaEmbeddingResponse represents the response from Ollama API
type OllamaEmbeddingResponse struct {
	Embedding []float64 `json:"embedding"`
}

// generateEmbedding calls Ollama API to generate embedding for text
func generateEmbedding(ollamaURL string, text string) ([]float32, error) {
	if strings.TrimSpace(text) == "" {
		return make([]float32, embeddingDimensions), nil
	}

	// Prepare request
	reqBody := OllamaEmbeddingRequest{
		Model:  "nomic-embed-text",
		Prompt: text,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Make HTTP request to Ollama
	resp, err := http.Post(ollamaURL+"/api/embeddings", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("Ollama API request failed: %w", err)
	}
	defer resp.Body.Close()
	

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Ollama API error (status %d): %s", resp.StatusCode, string(body))
	}

	// Parse response
	var ollamaResp OllamaEmbeddingResponse
	if err := json.NewDecoder(resp.Body).Decode(&ollamaResp); err != nil {
		return nil, fmt.Errorf("failed to parse Ollama response: %w", err)
	}

	// Convert []float64 to []float32
	if len(ollamaResp.Embedding) != embeddingDimensions {
		return nil, fmt.Errorf("unexpected embedding dimension: got %d, expected %d", len(ollamaResp.Embedding), embeddingDimensions)
	}

	embedding := make([]float32, embeddingDimensions)
	for i, v := range ollamaResp.Embedding {
		embedding[i] = float32(v)
	}

	return embedding, nil
}

// truncateText limits text to approximately maxTokens
// Rough estimate: 1 token ≈ 4 characters
func truncateText(text string, maxTokens int) string {
	maxChars := maxTokens * 4
	if len(text) <= maxChars {
		return text
	}
	return text[:maxChars]
}

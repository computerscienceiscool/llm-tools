package search

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

// Python script embedded for generating embeddings
const embeddingScript = `
import sys
import json
import numpy as np
try:
    from sentence_transformers import SentenceTransformer
    
    # Load model (cached after first use)
    model = SentenceTransformer('all-MiniLM-L6-v2')
    
    # Read text from stdin
    text = sys.stdin.read().strip()
    if not text:
        print(json.dumps([0.0] * 384))  # Return zero vector for empty text
        sys.exit(0)
    
    # Generate embedding
    embedding = model.encode(text, normalize_embeddings=True)
    
    # Convert to list and output as JSON
    result = embedding.tolist()
    print(json.dumps(result))
    
except ImportError:
    print("ERROR: sentence-transformers not installed", file=sys.stderr)
    sys.exit(1)
except Exception as e:
    print(f"ERROR: {str(e)}", file=sys.stderr)
    sys.exit(1)
`

// generateEmbedding calls Python script to generate embedding for text
func generateEmbedding(pythonPath string, text string) ([]float32, error) {
	if strings.TrimSpace(text) == "" {
		return make([]float32, embeddingDimensions), nil
	}

	// Create command
	cmd := exec.Command(pythonPath, "-c", embeddingScript)
	cmd.Stdin = strings.NewReader(text)

	// Run command and get combined output
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("Python script failed: %w\nOutput: %s", err, output)
	}

	// Parse JSON result
	var embedding []float32
	if err := json.Unmarshal(output, &embedding); err != nil {
		return nil, fmt.Errorf("failed to parse embedding JSON: %w\nOutput: %s", err, output)
	}

	if len(embedding) != embeddingDimensions {
		return nil, fmt.Errorf("unexpected embedding dimension: got %d, expected %d", len(embedding), embeddingDimensions)
	}

	return embedding, nil
}

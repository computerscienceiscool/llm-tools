package main

import (
	"bytes"
	"database/sql"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// SearchConfig holds search-related configuration
type SearchConfig struct {
	Enabled            bool     `yaml:"enabled"`
	VectorDBPath       string   `yaml:"vector_db_path"`
	EmbeddingModel     string   `yaml:"embedding_model"`
	MaxResults         int      `yaml:"max_results"`
	MinSimilarityScore float64  `yaml:"min_similarity_score"`
	MaxPreviewLength   int      `yaml:"max_preview_length"`
	ChunkSize          int      `yaml:"chunk_size"`
	PythonPath         string   `yaml:"python_path"`
	IndexExtensions    []string `yaml:"index_extensions"`
	MaxFileSize        int64    `yaml:"max_file_size"`
}

// SearchResult represents a single search result
type SearchResult struct {
	FilePath string
	Score    float64
	Lines    int
	Size     int64
	Preview  string
	ModTime  time.Time
}

// SearchEngine handles semantic search functionality
type SearchEngine struct {
	config  *SearchConfig
	db      *sql.DB
	session *Session
}

// FileInfo holds metadata about indexed files
type FileInfo struct {
	FilePath     string
	ContentHash  string
	Embedding    []float32
	LastModified int64
	FileSize     int64
	IndexedAt    int64
}

const embeddingDimensions = 384 // all-MiniLM-L6-v2 dimensions

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

// NewSearchEngine creates a new search engine instance
func NewSearchEngine(config *SearchConfig, session *Session) (*SearchEngine, error) {
	if !config.Enabled {
		return nil, fmt.Errorf("search is disabled")
	}

	// Check Python availability
	if err := checkPythonDependencies(config.PythonPath); err != nil {
		return nil, fmt.Errorf("Python dependencies not available: %w", err)
	}

	// Initialize database
	db, err := initSearchDB(config.VectorDBPath)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize search database: %w", err)
	}

	return &SearchEngine{
		config:  config,
		db:      db,
		session: session,
	}, nil
}

// Close closes the search engine and database
func (se *SearchEngine) Close() error {
	if se.db != nil {
		return se.db.Close()
	}
	return nil
}

// checkPythonDependencies verifies that Python and required packages are available
func checkPythonDependencies(pythonPath string) error {
	// Test Python availability
	cmd := exec.Command(pythonPath, "-c", "import sentence_transformers; print('OK')")
	output, err := cmd.CombinedOutput()
	fmt.Fprintf(os.Stderr, "DEBUG: Python command finished\n")
	if err != nil {
		return fmt.Errorf("Python or sentence-transformers not available: %w\nOutput: %s", err, output)
	}

	if !strings.Contains(string(output), "OK") {
		return fmt.Errorf("sentence-transformers import failed")
	}

	return nil
}

// initSearchDB initializes the SQLite database for storing embeddings
func initSearchDB(dbPath string) (*sql.DB, error) {
	// Create directory if it doesn't exist
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}

	// Create embeddings table
	schema := `
	CREATE TABLE IF NOT EXISTS embeddings (
		filepath TEXT PRIMARY KEY,
		content_hash TEXT NOT NULL,
		embedding BLOB NOT NULL,
		last_modified INTEGER NOT NULL,
		file_size INTEGER NOT NULL,
		indexed_at INTEGER NOT NULL
	);
	CREATE INDEX IF NOT EXISTS idx_hash ON embeddings(content_hash);
	CREATE INDEX IF NOT EXISTS idx_modified ON embeddings(last_modified);
	`

	if _, err := db.Exec(schema); err != nil {
		db.Close()
		return nil, err
	}

	return db, nil
}

// generateEmbedding calls Python script to generate embedding for text
func (se *SearchEngine) generateEmbedding(text string) ([]float32, error) {
	fmt.Fprintf(os.Stderr, "DEBUG: generateEmbedding called with text: %q\n", text[:min(50, len(text))])
	if strings.TrimSpace(text) == "" {
		return make([]float32, embeddingDimensions), nil
	}

	// Create command
	fmt.Fprintf(os.Stderr, "DEBUG: Using Python path: %s\n", se.config.PythonPath)
	cmd := exec.Command(se.config.PythonPath, "-c", embeddingScript)
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

// shouldIndexFile determines if a file should be indexed based on extension and other criteria
func (se *SearchEngine) shouldIndexFile(filePath string) bool {
	// Check file extension
	ext := strings.ToLower(filepath.Ext(filePath))
	if ext == "" {
		return false
	}

	found := false
	for _, allowedExt := range se.config.IndexExtensions {
		if strings.ToLower(allowedExt) == ext {
			found = true
			break
		}
	}
	if !found {
		return false
	}

	// Check if path is excluded
	for _, excluded := range se.session.Config.ExcludedPaths {
		if matched, _ := filepath.Match(excluded, filepath.Base(filePath)); matched {
			return false
		}
		if strings.HasPrefix(filePath, excluded+string(filepath.Separator)) {
			return false
		}
	}

	return true
}

// isTextFile checks if a file is text-based by looking for null bytes in the first 8KB
func isTextFile(filePath string) bool {
	file, err := os.Open(filePath)
	if err != nil {
		return false
	}
	defer file.Close()

	buffer := make([]byte, 8192)
	n, err := file.Read(buffer)
	if err != nil && err != io.EOF {
		return false
	}

	// Check for null bytes (indicates binary)
	for i := 0; i < n; i++ {
		if buffer[i] == 0 {
			return false
		}
	}

	return true
}

// calculateContentHash calculates SHA-256 hash of file content
func (se *SearchEngine) calculateContentHash(filePath string) (string, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", content), nil
}

// getFileInfo retrieves file metadata and embedding from database
func (se *SearchEngine) getFileInfo(filePath string) (*FileInfo, error) {
	var info FileInfo
	var embeddingData []byte

	err := se.db.QueryRow(`
		SELECT filepath, content_hash, embedding, last_modified, file_size, indexed_at 
		FROM embeddings WHERE filepath = ?
	`, filePath).Scan(
		&info.FilePath, &info.ContentHash, &embeddingData,
		&info.LastModified, &info.FileSize, &info.IndexedAt,
	)

	if err != nil {
		return nil, err
	}

	info.Embedding = deserializeEmbedding(embeddingData)
	return &info, nil
}

// storeFileInfo stores file metadata and embedding in database
func (se *SearchEngine) storeFileInfo(info *FileInfo) error {
	embeddingData := serializeEmbedding(info.Embedding)

	_, err := se.db.Exec(`
		INSERT OR REPLACE INTO embeddings 
		(filepath, content_hash, embedding, last_modified, file_size, indexed_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`, info.FilePath, info.ContentHash, embeddingData,
		info.LastModified, info.FileSize, info.IndexedAt)

	return err
}

// removeFileInfo removes file info from database (for deleted files)
func (se *SearchEngine) removeFileInfo(filePath string) error {
	_, err := se.db.Exec("DELETE FROM embeddings WHERE filepath = ?", filePath)
	return err
}

// generatePreview creates a preview snippet for search results
func (se *SearchEngine) generatePreview(filePath string, maxLength int) string {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return "Error reading file"
	}

	text := string(content)

	// Remove excessive whitespace
	text = regexp.MustCompile(`\s+`).ReplaceAllString(text, " ")
	text = strings.TrimSpace(text)

	if len(text) <= maxLength {
		return text
	}

	// Truncate and add ellipsis
	return text[:maxLength-3] + "..."
}

// ParseSearchCommand extracts search query from LLM output
func ParseSearchCommand(text string) string {
	// Pattern for <search query terms>
	searchPattern := regexp.MustCompile(`<search\s+([^>]+)>`)
	matches := searchPattern.FindStringSubmatch(text)

	if len(matches) >= 2 {
		return strings.TrimSpace(matches[1])
	}

	return ""
}

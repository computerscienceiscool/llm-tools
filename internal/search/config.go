package search

// SearchConfig holds search-related configuration
type SearchConfig struct {
	Enabled            bool     `yaml:"enabled"`
	VectorDBPath       string   `yaml:"vector_db_path"`
	EmbeddingModel     string   `yaml:"embedding_model"`
	MaxResults         int      `yaml:"max_results"`
	MinSimilarityScore float64  `yaml:"min_similarity_score"`
	MaxPreviewLength   int      `yaml:"max_preview_length"`
	ChunkSize          int      `yaml:"chunk_size"`
	OllamaURL         string   `yaml:"ollama_url"`
	IndexExtensions    []string `yaml:"index_extensions"`
	MaxFileSize        int64    `yaml:"max_file_size"`
}

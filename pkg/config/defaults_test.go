package config

import (
	"testing"

	"github.com/spf13/viper"
)

func TestGetDefaultSearchConfig(t *testing.T) {
	cfg := getDefaultSearchConfig()

	if cfg == nil {
		t.Fatal("expected non-nil config")
	}

	// Test default values
	tests := []struct {
		name     string
		got      interface{}
		expected interface{}
	}{
		{"Enabled", cfg.Enabled, false},
		{"VectorDBPath", cfg.VectorDBPath, "./embeddings.db"},
		{"EmbeddingModel", cfg.EmbeddingModel, "all-MiniLM-L6-v2"},
		{"EmbeddingDimensions", cfg.EmbeddingDimensions, DefaultEmbeddingDims},
		{"MaxResults", cfg.MaxResults, DefaultMaxSearchResults},
		{"MinSimilarityScore", cfg.MinSimilarityScore, DefaultMinSimilarity},
		{"MaxPreviewLength", cfg.MaxPreviewLength, 100},
		{"ChunkSize", cfg.ChunkSize, 1000},
		{"OllamaURL", cfg.OllamaURL, "http://localhost:11434"},
		{"MaxFileSize", cfg.MaxFileSize, int64(DefaultMaxFileSize)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, tt.got)
			}
		})
	}
}

func TestGetDefaultSearchConfig_IndexExtensions(t *testing.T) {
	cfg := getDefaultSearchConfig()

	expectedExtensions := []string{".go", ".py", ".js", ".md", ".txt", ".yaml", ".json"}

	if len(cfg.IndexExtensions) != len(expectedExtensions) {
		t.Fatalf("expected %d extensions, got %d", len(expectedExtensions), len(cfg.IndexExtensions))
	}

	for i, ext := range expectedExtensions {
		if cfg.IndexExtensions[i] != ext {
			t.Errorf("extension %d: expected %q, got %q", i, ext, cfg.IndexExtensions[i])
		}
	}
}

func TestGetDefaultSearchConfig_ReturnsNewInstance(t *testing.T) {
	cfg1 := getDefaultSearchConfig()
	cfg2 := getDefaultSearchConfig()

	// Modify cfg1
	cfg1.MaxResults = 999
	cfg1.IndexExtensions[0] = ".modified"

	// cfg2 should be unaffected
	if cfg2.MaxResults == 999 {
		t.Error("modifying one config should not affect another")
	}
	if cfg2.IndexExtensions[0] == ".modified" {
		t.Error("modifying one config's slice should not affect another")
	}
}

func TestsetFullConfigDefaults(t *testing.T) {
	cfg := &fullConfig{}
	setFullConfigDefaults(cfg)

	// Test repository defaults
	t.Run("Repository", func(t *testing.T) {
		if cfg.Repository.Root != "." {
			t.Errorf("expected Root '.', got %q", cfg.Repository.Root)
		}
		if len(cfg.Repository.ExcludedPaths) != 4 {
			t.Errorf("expected 4 excluded paths, got %d", len(cfg.Repository.ExcludedPaths))
		}
		expectedExcluded := []string{".git", ".env", "*.key", "*.pem"}
		for i, path := range expectedExcluded {
			if cfg.Repository.ExcludedPaths[i] != path {
				t.Errorf("excluded path %d: expected %q, got %q", i, path, cfg.Repository.ExcludedPaths[i])
			}
		}
	})

	// Test open command defaults
	t.Run("Commands.Open", func(t *testing.T) {
		if !cfg.Commands.Open.Enabled {
			t.Error("expected Open.Enabled to be true")
		}
		if cfg.Commands.Open.MaxFileSize != DefaultMaxFileSize {
			t.Errorf("expected MaxFileSize 1048576, got %d", cfg.Commands.Open.MaxFileSize)
		}
		expectedExtensions := []string{".go", ".py", ".js", ".md", ".txt", ".json", ".yaml"}
		if len(cfg.Commands.Open.AllowedExtensions) != len(expectedExtensions) {
			t.Errorf("expected %d allowed extensions, got %d",
				len(expectedExtensions), len(cfg.Commands.Open.AllowedExtensions))
		}
	})

	// Test write command defaults
	t.Run("Commands.Write", func(t *testing.T) {
		if !cfg.Commands.Write.Enabled {
			t.Error("expected Write.Enabled to be true")
		}
		if cfg.Commands.Write.MaxFileSize != DefaultMaxWriteSize {
			t.Errorf("expected MaxFileSize 102400, got %d", cfg.Commands.Write.MaxFileSize)
		}
		if !cfg.Commands.Write.BackupBeforeWrite {
			t.Error("expected BackupBeforeWrite to be true")
		}
	})

	// Test exec command defaults
	t.Run("Commands.Exec", func(t *testing.T) {
		if cfg.Commands.Exec.Enabled {
			t.Error("expected Exec.Enabled to be false")
		}
		if cfg.Commands.Exec.ContainerImage != "ubuntu:22.04" {
			t.Errorf("expected ContainerImage 'ubuntu:22.04', got %q", cfg.Commands.Exec.ContainerImage)
		}
		if cfg.Commands.Exec.TimeoutSeconds != int(DefaultExecTimeout.Seconds()) {
			t.Errorf("expected TimeoutSeconds 30, got %d", cfg.Commands.Exec.TimeoutSeconds)
		}
		if cfg.Commands.Exec.MemoryLimit != DefaultContainerMemory {
			t.Errorf("expected MemoryLimit '512m', got %q", cfg.Commands.Exec.MemoryLimit)
		}
		if cfg.Commands.Exec.CPULimit != 2 {
			t.Errorf("expected CPULimit 2, got %d", cfg.Commands.Exec.CPULimit)
		}
		expectedWhitelist := []string{"go test", "go build", "npm test", "make"}
		if len(cfg.Commands.Exec.Whitelist) != len(expectedWhitelist) {
			t.Errorf("expected %d whitelist entries, got %d",
				len(expectedWhitelist), len(cfg.Commands.Exec.Whitelist))
		}
	})

	// Test search command defaults
	t.Run("Commands.Search", func(t *testing.T) {
		if cfg.Commands.Search.Enabled {
			t.Error("expected Search.Enabled to be false")
		}
		if cfg.Commands.Search.VectorDBPath != "./embeddings.db" {
			t.Errorf("expected VectorDBPath './embeddings.db', got %q", cfg.Commands.Search.VectorDBPath)
		}
		if cfg.Commands.Search.EmbeddingModel != "all-MiniLM-L6-v2" {
			t.Errorf("expected EmbeddingModel 'all-MiniLM-L6-v2', got %q", cfg.Commands.Search.EmbeddingModel)
		}
		if cfg.Commands.Search.MaxResults != DefaultMaxSearchResults {
			t.Errorf("expected MaxResults 10, got %d", cfg.Commands.Search.MaxResults)
		}
		if cfg.Commands.Search.MinSimilarityScore != DefaultMinSimilarity {
			t.Errorf("expected MinSimilarityScore 0.5, got %f", cfg.Commands.Search.MinSimilarityScore)
		}
		if cfg.Commands.Search.MaxPreviewLength != 100 {
			t.Errorf("expected MaxPreviewLength 100, got %d", cfg.Commands.Search.MaxPreviewLength)
		}
		if cfg.Commands.Search.ChunkSize != 1000 {
			t.Errorf("expected ChunkSize 1000, got %d", cfg.Commands.Search.ChunkSize)
		}
		if cfg.Commands.Search.OllamaURL != "http://localhost:11434" {
			t.Errorf("expected OllamaURL 'python3', got %q", cfg.Commands.Search.OllamaURL)
		}
		if cfg.Commands.Search.MaxFileSize != int64(DefaultMaxFileSize) {
			t.Errorf("expected MaxFileSize 1048576, got %d", cfg.Commands.Search.MaxFileSize)
		}
	})

	// Test security defaults
	t.Run("Security", func(t *testing.T) {
		if cfg.Security.RateLimitPerMinute != 100 {
			t.Errorf("expected RateLimitPerMinute 100, got %d", cfg.Security.RateLimitPerMinute)
		}
		if !cfg.Security.LogAllOperations {
			t.Error("expected LogAllOperations to be true")
		}
		if cfg.Security.AuditLogPath != "./audit.log" {
			t.Errorf("expected AuditLogPath './audit.log', got %q", cfg.Security.AuditLogPath)
		}
	})

	// Test output defaults
	t.Run("Output", func(t *testing.T) {
		if !cfg.Output.ShowSummaries {
			t.Error("expected ShowSummaries to be true")
		}
		if !cfg.Output.ShowExecutionTime {
			t.Error("expected ShowExecutionTime to be true")
		}
		if !cfg.Output.TruncateLargeOutputs {
			t.Error("expected TruncateLargeOutputs to be true")
		}
		if cfg.Output.MaxOutputLines != 1000 {
			t.Errorf("expected MaxOutputLines 1000, got %d", cfg.Output.MaxOutputLines)
		}
	})

	// Test logging defaults
	t.Run("Logging", func(t *testing.T) {
		if cfg.Logging.Level != "info" {
			t.Errorf("expected Level 'info', got %q", cfg.Logging.Level)
		}
		if cfg.Logging.File != "./llm-runtime.log" {
			t.Errorf("expected File './llm-runtime.log', got %q", cfg.Logging.File)
		}
		if cfg.Logging.Format != "json" {
			t.Errorf("expected Format 'json', got %q", cfg.Logging.Format)
		}
	})
}

func TestSetFullConfigDefaults_OverwritesExisting(t *testing.T) {
	cfg := &fullConfig{}

	// Set some non-default values
	cfg.Repository.Root = "/custom/path"
	cfg.Commands.Open.Enabled = false
	cfg.Security.RateLimitPerMinute = 999

	// Apply defaults - should overwrite
	setFullConfigDefaults(cfg)

	if cfg.Repository.Root != "." {
		t.Errorf("expected Root to be overwritten to '.', got %q", cfg.Repository.Root)
	}
	if !cfg.Commands.Open.Enabled {
		t.Error("expected Open.Enabled to be overwritten to true")
	}
	if cfg.Security.RateLimitPerMinute != 100 {
		t.Errorf("expected RateLimitPerMinute to be overwritten to 100, got %d", cfg.Security.RateLimitPerMinute)
	}
}

func TestSetFullConfigDefaults_NilConfig(t *testing.T) {
	// This should panic - testing that we handle nil properly
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic with nil config")
		}
	}()

	setFullConfigDefaults(nil)
}

// Test that default file sizes are sensible
func TestDefaultFileSizes(t *testing.T) {
	cfg := &fullConfig{}
	setFullConfigDefaults(cfg)

	// Open max file size should be larger than write max file size
	if cfg.Commands.Open.MaxFileSize <= cfg.Commands.Write.MaxFileSize {
		t.Error("Open.MaxFileSize should be larger than Write.MaxFileSize")
	}

	// Max file size for search should match open
	if cfg.Commands.Search.MaxFileSize != cfg.Commands.Open.MaxFileSize {
		t.Error("Search.MaxFileSize should match Open.MaxFileSize")
	}

	// All file sizes should be positive
	if cfg.Commands.Open.MaxFileSize <= 0 {
		t.Error("Open.MaxFileSize should be positive")
	}
	if cfg.Commands.Write.MaxFileSize <= 0 {
		t.Error("Write.MaxFileSize should be positive")
	}
	if cfg.Commands.Search.MaxFileSize <= 0 {
		t.Error("Search.MaxFileSize should be positive")
	}
}

// Benchmark
func BenchmarkGetDefaultSearchConfig(b *testing.B) {
	for i := 0; i < b.N; i++ {
		getDefaultSearchConfig()
	}
}

func BenchmarksetFullConfigDefaults(b *testing.B) {
	for i := 0; i < b.N; i++ {
		cfg := &fullConfig{}
		setFullConfigDefaults(cfg)
	}
}

// TestSetViperDefaults tests that viper defaults are set correctly
func TestSetViperDefaults(t *testing.T) {
	viper.Reset()

	SetViperDefaults()

	// Check repository defaults
	if viper.GetString("repository.root") != "." {
		t.Errorf("repository.root = %q, want '.'", viper.GetString("repository.root"))
	}

	excludedPaths := viper.GetStringSlice("repository.excluded_paths")
	if len(excludedPaths) == 0 {
		t.Error("repository.excluded_paths should not be empty")
	}

	// Check open command defaults
	if !viper.GetBool("commands.open.enabled") {
		t.Error("commands.open.enabled should be true by default")
	}

	if viper.GetInt("commands.open.max_file_size") != DefaultMaxFileSize {
		t.Errorf("commands.open.max_file_size = %d, want 1048576", viper.GetInt("commands.open.max_file_size"))
	}

	// Check write command defaults
	if !viper.GetBool("commands.write.enabled") {
		t.Error("commands.write.enabled should be true by default")
	}

	if viper.GetInt("commands.write.max_file_size") != DefaultMaxWriteSize {
		t.Errorf("commands.write.max_file_size = %d, want 102400", viper.GetInt("commands.write.max_file_size"))
	}

	if !viper.GetBool("commands.write.backup_before_write") {
		t.Error("commands.write.backup_before_write should be true by default")
	}

	// Check exec command defaults
	if viper.GetBool("commands.exec.enabled") {
		t.Error("commands.exec.enabled should be false by default")
	}

	if viper.GetInt("commands.exec.timeout_seconds") != int(DefaultExecTimeout.Seconds()) {
		t.Errorf("commands.exec.timeout_seconds = %d, want 30", viper.GetInt("commands.exec.timeout_seconds"))
	}
}

// TestLoadSearchConfig_Defaults tests LoadSearchConfig with default values
func TestLoadSearchConfig_Defaults(t *testing.T) {
	viper.Reset()

	cfg := LoadSearchConfig()

	if cfg == nil {
		t.Fatal("LoadSearchConfig() returned nil")
	}

	// Should return default config when no viper values are set
	defaultCfg := getDefaultSearchConfig()

	if cfg.Enabled != defaultCfg.Enabled {
		t.Errorf("Enabled = %v, want %v", cfg.Enabled, defaultCfg.Enabled)
	}

	if cfg.MaxResults != defaultCfg.MaxResults {
		t.Errorf("MaxResults = %d, want %d", cfg.MaxResults, defaultCfg.MaxResults)
	}
}

// TestLoadSearchConfig_CustomValues tests LoadSearchConfig with custom viper values
func TestLoadSearchConfig_CustomValues(t *testing.T) {
	viper.Reset()

	// Set custom values
	viper.Set("commands.search.enabled", true)
	viper.Set("commands.search.max_results", 25)
	viper.Set("commands.search.min_similarity_score", 0.8)
	viper.Set("commands.search.ollama_url", "http://custom:11434")

	cfg := LoadSearchConfig()

	if !cfg.Enabled {
		t.Error("Enabled should be true")
	}

	if cfg.MaxResults != 25 {
		t.Errorf("MaxResults = %d, want 25", cfg.MaxResults)
	}

	if cfg.MinSimilarityScore != 0.8 {
		t.Errorf("MinSimilarityScore = %f, want 0.8", cfg.MinSimilarityScore)
	}

	if cfg.OllamaURL != "http://custom:11434" {
		t.Errorf("OllamaURL = %q, want 'http://custom:11434'", cfg.OllamaURL)
	}
}

// TestLoadSearchConfig_PartialOverride tests partial config override
func TestLoadSearchConfig_PartialOverride(t *testing.T) {
	viper.Reset()

	// Override only some values
	viper.Set("commands.search.max_results", 50)
	viper.Set("commands.search.chunk_size", 2000)

	cfg := LoadSearchConfig()
	defaultCfg := getDefaultSearchConfig()

	// Overridden values
	if cfg.MaxResults != 50 {
		t.Errorf("MaxResults = %d, want 50", cfg.MaxResults)
	}

	if cfg.ChunkSize != 2000 {
		t.Errorf("ChunkSize = %d, want 2000", cfg.ChunkSize)
	}

	// Non-overridden values should still be defaults
	if cfg.Enabled != defaultCfg.Enabled {
		t.Error("Non-overridden Enabled should match default")
	}

	if cfg.EmbeddingModel != defaultCfg.EmbeddingModel {
		t.Error("Non-overridden EmbeddingModel should match default")
	}
}

// TestLoadSearchConfig_IndexExtensions tests loading custom index extensions
func TestLoadSearchConfig_IndexExtensions(t *testing.T) {
	viper.Reset()

	customExtensions := []string{".rs", ".cpp", ".java"}
	viper.Set("commands.search.index_extensions", customExtensions)

	cfg := LoadSearchConfig()

	if len(cfg.IndexExtensions) != 3 {
		t.Errorf("IndexExtensions length = %d, want 3", len(cfg.IndexExtensions))
	}

	for i, ext := range customExtensions {
		if cfg.IndexExtensions[i] != ext {
			t.Errorf("IndexExtensions[%d] = %q, want %q", i, cfg.IndexExtensions[i], ext)
		}
	}
}

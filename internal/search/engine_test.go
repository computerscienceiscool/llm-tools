package search

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewSearchEngine_Success(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	cfg := &SearchConfig{
		Enabled:            true,
		VectorDBPath:       dbPath,
		MaxResults:         10,
		MinSimilarityScore: 0.5,
		MaxPreviewLength:   100,
		PythonPath:         "python3",
	}

	engine, err := NewSearchEngine(cfg, tmpDir)
	if err != nil {
		t.Fatalf("NewSearchEngine failed: %v", err)
	}
	defer engine.Close()

	if engine == nil {
		t.Fatal("expected non-nil engine")
	}

	// Verify database file was created
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Error("database file was not created")
	}
}

func TestNewSearchEngine_Disabled(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := &SearchConfig{
		Enabled:      false,
		VectorDBPath: filepath.Join(tmpDir, "test.db"),
	}

	_, err := NewSearchEngine(cfg, tmpDir)
	if err == nil {
		t.Error("expected error when search is disabled")
	}
}

func TestNewSearchEngine_CreatesDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	nestedPath := filepath.Join(tmpDir, "a", "b", "c", "test.db")

	cfg := &SearchConfig{
		Enabled:      true,
		VectorDBPath: nestedPath,
	}

	engine, err := NewSearchEngine(cfg, tmpDir)
	if err != nil {
		t.Fatalf("NewSearchEngine failed to create nested directories: %v", err)
	}
	defer engine.Close()

	// Verify directory was created
	if _, err := os.Stat(filepath.Dir(nestedPath)); os.IsNotExist(err) {
		t.Error("nested directories were not created")
	}
}

func TestSearchEngine_Close(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	cfg := &SearchConfig{
		Enabled:      true,
		VectorDBPath: dbPath,
	}

	engine, err := NewSearchEngine(cfg, tmpDir)
	if err != nil {
		t.Fatalf("NewSearchEngine failed: %v", err)
	}

	// Close should not error
	err = engine.Close()
	if err != nil {
		t.Errorf("Close failed: %v", err)
	}

	// Closing again should not panic (though may error)
	engine.Close()
}

func TestSearchEngine_Close_NilDB(t *testing.T) {
	engine := &SearchEngine{
		db:       nil,
		config:   &SearchConfig{},
		repoRoot: "/tmp",
	}

	// Should not panic or error with nil db
	err := engine.Close()
	if err != nil {
		t.Errorf("Close with nil db should not error: %v", err)
	}
}

func TestSearchEngine_GetDB(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	cfg := &SearchConfig{
		Enabled:      true,
		VectorDBPath: dbPath,
	}

	engine, err := NewSearchEngine(cfg, tmpDir)
	if err != nil {
		t.Fatalf("NewSearchEngine failed: %v", err)
	}
	defer engine.Close()

	db := engine.GetDB()
	if db == nil {
		t.Error("GetDB returned nil")
	}

	// Verify we can use the returned db
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM embeddings").Scan(&count)
	if err != nil {
		t.Errorf("failed to query using GetDB result: %v", err)
	}
}

func TestSearchEngine_GetConfig(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	cfg := &SearchConfig{
		Enabled:            true,
		VectorDBPath:       dbPath,
		MaxResults:         25,
		MinSimilarityScore: 0.75,
		MaxPreviewLength:   200,
		EmbeddingModel:     "test-model",
	}

	engine, err := NewSearchEngine(cfg, tmpDir)
	if err != nil {
		t.Fatalf("NewSearchEngine failed: %v", err)
	}
	defer engine.Close()

	returnedCfg := engine.GetConfig()

	if returnedCfg != cfg {
		t.Error("GetConfig should return the same config instance")
	}

	if returnedCfg.MaxResults != 25 {
		t.Errorf("MaxResults mismatch: expected 25, got %d", returnedCfg.MaxResults)
	}

	if returnedCfg.MinSimilarityScore != 0.75 {
		t.Errorf("MinSimilarityScore mismatch: expected 0.75, got %f", returnedCfg.MinSimilarityScore)
	}

	if returnedCfg.EmbeddingModel != "test-model" {
		t.Errorf("EmbeddingModel mismatch: expected 'test-model', got %q", returnedCfg.EmbeddingModel)
	}
}

func TestSearchEngine_GetRepoRoot(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	cfg := &SearchConfig{
		Enabled:      true,
		VectorDBPath: dbPath,
	}

	engine, err := NewSearchEngine(cfg, tmpDir)
	if err != nil {
		t.Fatalf("NewSearchEngine failed: %v", err)
	}
	defer engine.Close()

	repoRoot := engine.GetRepoRoot()
	if repoRoot != tmpDir {
		t.Errorf("GetRepoRoot mismatch: expected %q, got %q", tmpDir, repoRoot)
	}
}

func TestSearchEngine_MultipleDatabases(t *testing.T) {
	tmpDir := t.TempDir()

	// Create two separate engines with different databases
	cfg1 := &SearchConfig{
		Enabled:      true,
		VectorDBPath: filepath.Join(tmpDir, "db1.db"),
	}
	cfg2 := &SearchConfig{
		Enabled:      true,
		VectorDBPath: filepath.Join(tmpDir, "db2.db"),
	}

	engine1, err := NewSearchEngine(cfg1, tmpDir)
	if err != nil {
		t.Fatalf("NewSearchEngine 1 failed: %v", err)
	}
	defer engine1.Close()

	engine2, err := NewSearchEngine(cfg2, tmpDir)
	if err != nil {
		t.Fatalf("NewSearchEngine 2 failed: %v", err)
	}
	defer engine2.Close()

	// Add data to engine1
	info := &FileInfo{
		FilePath:     "test.go",
		ContentHash:  "hash",
		Embedding:    createTestEmbedding(),
		LastModified: 1000,
		FileSize:     100,
		IndexedAt:    1000,
	}
	if err := storeFileInfo(engine1.GetDB(), info); err != nil {
		t.Fatalf("storeFileInfo failed: %v", err)
	}

	// Verify engine2 doesn't have the data
	files, err := getAllIndexedFiles(engine2.GetDB())
	if err != nil {
		t.Fatalf("getAllIndexedFiles failed: %v", err)
	}

	if len(files) != 0 {
		t.Error("engine2 should have separate database")
	}
}

func TestSearchEngine_DBSchemaValid(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	cfg := &SearchConfig{
		Enabled:      true,
		VectorDBPath: dbPath,
	}

	engine, err := NewSearchEngine(cfg, tmpDir)
	if err != nil {
		t.Fatalf("NewSearchEngine failed: %v", err)
	}
	defer engine.Close()

	// Verify embeddings table exists and has correct schema
	db := engine.GetDB()

	// Check table exists
	var tableName string
	err = db.QueryRow(`
		SELECT name FROM sqlite_master 
		WHERE type='table' AND name='embeddings'
	`).Scan(&tableName)

	if err != nil {
		t.Fatalf("embeddings table not found: %v", err)
	}

	// Verify we can insert and query
	_, err = db.Exec(`
		INSERT INTO embeddings (filepath, content_hash, embedding, last_modified, file_size, indexed_at)
		VALUES ('test.go', 'hash', X'01020304', 1000, 100, 1000)
	`)
	if err != nil {
		t.Fatalf("insert failed: %v", err)
	}

	var filepath string
	err = db.QueryRow("SELECT filepath FROM embeddings WHERE filepath = ?", "test.go").Scan(&filepath)
	if err != nil {
		t.Fatalf("query failed: %v", err)
	}
}

func TestSearchEngine_InMemoryDB(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := &SearchConfig{
		Enabled:      true,
		VectorDBPath: ":memory:",
	}

	engine, err := NewSearchEngine(cfg, tmpDir)
	if err != nil {
		t.Fatalf("NewSearchEngine with :memory: failed: %v", err)
	}
	defer engine.Close()

	// Verify we can use the in-memory database
	db := engine.GetDB()
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM embeddings").Scan(&count)
	if err != nil {
		t.Fatalf("query on in-memory db failed: %v", err)
	}
}

func TestSearchEngine_ReopenExistingDB(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "persistent.db")

	cfg := &SearchConfig{
		Enabled:      true,
		VectorDBPath: dbPath,
	}

	// Create engine and add data
	engine1, err := NewSearchEngine(cfg, tmpDir)
	if err != nil {
		t.Fatalf("first NewSearchEngine failed: %v", err)
	}

	info := &FileInfo{
		FilePath:     "persistent.go",
		ContentHash:  "hash",
		Embedding:    createTestEmbedding(),
		LastModified: 1000,
		FileSize:     100,
		IndexedAt:    1000,
	}
	if err := storeFileInfo(engine1.GetDB(), info); err != nil {
		t.Fatalf("storeFileInfo failed: %v", err)
	}
	engine1.Close()

	// Reopen and verify data persisted
	engine2, err := NewSearchEngine(cfg, tmpDir)
	if err != nil {
		t.Fatalf("second NewSearchEngine failed: %v", err)
	}
	defer engine2.Close()

	files, err := getAllIndexedFiles(engine2.GetDB())
	if err != nil {
		t.Fatalf("getAllIndexedFiles failed: %v", err)
	}

	if len(files) != 1 || files[0] != "persistent.go" {
		t.Error("data should persist across engine reopens")
	}
}

func TestSearchEngine_ConfigNil(t *testing.T) {
	// NewSearchEngine with nil config should be handled
	// (the function expects a valid config, so this tests defensive behavior)
	defer func() {
		if r := recover(); r != nil {
			// If it panics, that's expected for nil config
			t.Logf("NewSearchEngine panicked with nil config (expected): %v", r)
		}
	}()

	_, err := NewSearchEngine(nil, "/tmp")
	if err == nil {
		t.Error("expected error or panic with nil config")
	}
}

func TestSearchEngine_EmptyRepoRoot(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := &SearchConfig{
		Enabled:      true,
		VectorDBPath: filepath.Join(tmpDir, "test.db"),
	}

	// Empty repo root should still work
	engine, err := NewSearchEngine(cfg, "")
	if err != nil {
		t.Fatalf("NewSearchEngine with empty repo root failed: %v", err)
	}
	defer engine.Close()

	if engine.GetRepoRoot() != "" {
		t.Errorf("expected empty repo root, got %q", engine.GetRepoRoot())
	}
}

func TestSearchEngine_RelativeDBPath(t *testing.T) {
	// Save current dir
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current dir: %v", err)
	}

	tmpDir := t.TempDir()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to change to temp dir: %v", err)
	}
	defer os.Chdir(origDir)

	cfg := &SearchConfig{
		Enabled:      true,
		VectorDBPath: "relative.db",
	}

	engine, err := NewSearchEngine(cfg, tmpDir)
	if err != nil {
		t.Fatalf("NewSearchEngine with relative path failed: %v", err)
	}
	defer engine.Close()

	// Verify database was created
	if _, err := os.Stat("relative.db"); os.IsNotExist(err) {
		t.Error("relative database file was not created")
	}
}

// Skip tests that require Python/sentence-transformers
// These would need integration test setup

func TestSearchEngine_Search_NoPython(t *testing.T) {
	t.Skip("Search tests require Python with sentence-transformers installed")
}

// Benchmark tests
func BenchmarkNewSearchEngine(b *testing.B) {
	tmpDir := b.TempDir()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		dbPath := filepath.Join(tmpDir, "bench.db")
		cfg := &SearchConfig{
			Enabled:      true,
			VectorDBPath: dbPath,
		}
		engine, err := NewSearchEngine(cfg, tmpDir)
		if err != nil {
			b.Fatalf("NewSearchEngine failed: %v", err)
		}
		engine.Close()
		os.Remove(dbPath)
	}
}

func BenchmarkSearchEngine_GetDB(b *testing.B) {
	tmpDir := b.TempDir()
	dbPath := filepath.Join(tmpDir, "bench.db")

	cfg := &SearchConfig{
		Enabled:      true,
		VectorDBPath: dbPath,
	}

	engine, err := NewSearchEngine(cfg, tmpDir)
	if err != nil {
		b.Fatalf("NewSearchEngine failed: %v", err)
	}
	defer engine.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		engine.GetDB()
	}
}

func BenchmarkSearchEngine_GetConfig(b *testing.B) {
	tmpDir := b.TempDir()
	dbPath := filepath.Join(tmpDir, "bench.db")

	cfg := &SearchConfig{
		Enabled:      true,
		VectorDBPath: dbPath,
	}

	engine, err := NewSearchEngine(cfg, tmpDir)
	if err != nil {
		b.Fatalf("NewSearchEngine failed: %v", err)
	}
	defer engine.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		engine.GetConfig()
	}
}

func BenchmarkSearchEngine_GetRepoRoot(b *testing.B) {
	tmpDir := b.TempDir()
	dbPath := filepath.Join(tmpDir, "bench.db")

	cfg := &SearchConfig{
		Enabled:      true,
		VectorDBPath: dbPath,
	}

	engine, err := NewSearchEngine(cfg, tmpDir)
	if err != nil {
		b.Fatalf("NewSearchEngine failed: %v", err)
	}
	defer engine.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		engine.GetRepoRoot()
	}
}

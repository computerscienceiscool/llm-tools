package search

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/computerscienceiscool/llm-tools/internal/infrastructure"
)

// Helper to create a test database
func createTestDB(t *testing.T) (*SearchEngine, func()) {
	t.Helper()

	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	cfg := &SearchConfig{
		Enabled:            true,
		VectorDBPath:       dbPath,
		MaxResults:         10,
		MinSimilarityScore: 0.5,
		MaxPreviewLength:   100,
	}

	db, err := infrastructure.InitSearchDB(dbPath)
	if err != nil {
		t.Fatalf("failed to init test db: %v", err)
	}

	engine := &SearchEngine{
		db:       db,
		config:   cfg,
		repoRoot: tmpDir,
	}

	cleanup := func() {
		db.Close()
	}

	return engine, cleanup
}

// Helper to create a test embedding
func createTestEmbedding() []float32 {
	emb := make([]float32, embeddingDimensions)
	for i := range emb {
		emb[i] = float32(i) / float32(embeddingDimensions)
	}
	return emb
}

func TestStoreFileInfo_Success(t *testing.T) {
	engine, cleanup := createTestDB(t)
	defer cleanup()

	info := &FileInfo{
		FilePath:     "internal/main.go",
		ContentHash:  "abc123def456",
		Embedding:    createTestEmbedding(),
		LastModified: time.Now().Unix(),
		FileSize:     1024,
		IndexedAt:    time.Now().Unix(),
	}

	err := storeFileInfo(engine.db, info)
	if err != nil {
		t.Fatalf("storeFileInfo failed: %v", err)
	}

	// Verify it was stored
	retrieved, err := getFileInfo(engine.db, "internal/main.go")
	if err != nil {
		t.Fatalf("getFileInfo failed: %v", err)
	}

	if retrieved.FilePath != info.FilePath {
		t.Errorf("FilePath mismatch: expected %q, got %q", info.FilePath, retrieved.FilePath)
	}
	if retrieved.ContentHash != info.ContentHash {
		t.Errorf("ContentHash mismatch: expected %q, got %q", info.ContentHash, retrieved.ContentHash)
	}
	if retrieved.FileSize != info.FileSize {
		t.Errorf("FileSize mismatch: expected %d, got %d", info.FileSize, retrieved.FileSize)
	}
}

func TestStoreFileInfo_Update(t *testing.T) {
	engine, cleanup := createTestDB(t)
	defer cleanup()

	// Store initial version
	info1 := &FileInfo{
		FilePath:     "test.go",
		ContentHash:  "hash1",
		Embedding:    createTestEmbedding(),
		LastModified: 1000,
		FileSize:     100,
		IndexedAt:    1000,
	}
	if err := storeFileInfo(engine.db, info1); err != nil {
		t.Fatalf("first store failed: %v", err)
	}

	// Store updated version (same filepath)
	info2 := &FileInfo{
		FilePath:     "test.go",
		ContentHash:  "hash2",
		Embedding:    createTestEmbedding(),
		LastModified: 2000,
		FileSize:     200,
		IndexedAt:    2000,
	}
	if err := storeFileInfo(engine.db, info2); err != nil {
		t.Fatalf("second store failed: %v", err)
	}

	// Verify updated values
	retrieved, err := getFileInfo(engine.db, "test.go")
	if err != nil {
		t.Fatalf("getFileInfo failed: %v", err)
	}

	if retrieved.ContentHash != "hash2" {
		t.Errorf("expected updated hash 'hash2', got %q", retrieved.ContentHash)
	}
	if retrieved.FileSize != 200 {
		t.Errorf("expected updated size 200, got %d", retrieved.FileSize)
	}
}

func TestStoreFileInfo_MultipleFiles(t *testing.T) {
	engine, cleanup := createTestDB(t)
	defer cleanup()

	files := []string{
		"file1.go",
		"file2.go",
		"dir/file3.go",
		"dir/subdir/file4.go",
	}

	for i, f := range files {
		info := &FileInfo{
			FilePath:     f,
			ContentHash:  "hash" + string(rune('0'+i)),
			Embedding:    createTestEmbedding(),
			LastModified: int64(1000 + i),
			FileSize:     int64(100 * (i + 1)),
			IndexedAt:    int64(1000 + i),
		}
		if err := storeFileInfo(engine.db, info); err != nil {
			t.Fatalf("storeFileInfo(%s) failed: %v", f, err)
		}
	}

	// Verify all files stored
	allFiles, err := getAllIndexedFiles(engine.db)
	if err != nil {
		t.Fatalf("getAllIndexedFiles failed: %v", err)
	}

	if len(allFiles) != len(files) {
		t.Errorf("expected %d files, got %d", len(files), len(allFiles))
	}
}

func TestGetFileInfo_NotFound(t *testing.T) {
	engine, cleanup := createTestDB(t)
	defer cleanup()

	_, err := getFileInfo(engine.db, "nonexistent.go")
	if err == nil {
		t.Error("expected error for nonexistent file, got nil")
	}
}

func TestGetFileInfo_EmbeddingDeserialization(t *testing.T) {
	engine, cleanup := createTestDB(t)
	defer cleanup()

	originalEmb := createTestEmbedding()
	info := &FileInfo{
		FilePath:     "test.go",
		ContentHash:  "hash",
		Embedding:    originalEmb,
		LastModified: 1000,
		FileSize:     100,
		IndexedAt:    1000,
	}

	if err := storeFileInfo(engine.db, info); err != nil {
		t.Fatalf("storeFileInfo failed: %v", err)
	}

	retrieved, err := getFileInfo(engine.db, "test.go")
	if err != nil {
		t.Fatalf("getFileInfo failed: %v", err)
	}

	if len(retrieved.Embedding) != embeddingDimensions {
		t.Fatalf("expected embedding length %d, got %d", embeddingDimensions, len(retrieved.Embedding))
	}

	// Verify embedding values match
	for i := 0; i < embeddingDimensions; i++ {
		if retrieved.Embedding[i] != originalEmb[i] {
			t.Errorf("embedding[%d] mismatch: expected %f, got %f", i, originalEmb[i], retrieved.Embedding[i])
			break
		}
	}
}

func TestRemoveFileInfo_Success(t *testing.T) {
	engine, cleanup := createTestDB(t)
	defer cleanup()

	// Store a file
	info := &FileInfo{
		FilePath:     "to_delete.go",
		ContentHash:  "hash",
		Embedding:    createTestEmbedding(),
		LastModified: 1000,
		FileSize:     100,
		IndexedAt:    1000,
	}
	if err := storeFileInfo(engine.db, info); err != nil {
		t.Fatalf("storeFileInfo failed: %v", err)
	}

	// Verify it exists
	_, err := getFileInfo(engine.db, "to_delete.go")
	if err != nil {
		t.Fatalf("file should exist before deletion: %v", err)
	}

	// Remove it
	if err := removeFileInfo(engine.db, "to_delete.go"); err != nil {
		t.Fatalf("removeFileInfo failed: %v", err)
	}

	// Verify it's gone
	_, err = getFileInfo(engine.db, "to_delete.go")
	if err == nil {
		t.Error("file should not exist after deletion")
	}
}

func TestRemoveFileInfo_NonexistentFile(t *testing.T) {
	engine, cleanup := createTestDB(t)
	defer cleanup()

	// Remove nonexistent file - should not error
	err := removeFileInfo(engine.db, "nonexistent.go")
	if err != nil {
		t.Errorf("removeFileInfo on nonexistent file should not error: %v", err)
	}
}

func TestRemoveFileInfo_DoesNotAffectOthers(t *testing.T) {
	engine, cleanup := createTestDB(t)
	defer cleanup()

	// Store two files
	for _, f := range []string{"keep.go", "delete.go"} {
		info := &FileInfo{
			FilePath:     f,
			ContentHash:  "hash",
			Embedding:    createTestEmbedding(),
			LastModified: 1000,
			FileSize:     100,
			IndexedAt:    1000,
		}
		if err := storeFileInfo(engine.db, info); err != nil {
			t.Fatalf("storeFileInfo(%s) failed: %v", f, err)
		}
	}

	// Remove one
	if err := removeFileInfo(engine.db, "delete.go"); err != nil {
		t.Fatalf("removeFileInfo failed: %v", err)
	}

	// Verify other still exists
	_, err := getFileInfo(engine.db, "keep.go")
	if err != nil {
		t.Error("keep.go should still exist after deleting delete.go")
	}
}

func TestGetAllIndexedFiles_Empty(t *testing.T) {
	engine, cleanup := createTestDB(t)
	defer cleanup()

	files, err := getAllIndexedFiles(engine.db)
	if err != nil {
		t.Fatalf("getAllIndexedFiles failed: %v", err)
	}

	if len(files) != 0 {
		t.Errorf("expected 0 files in empty db, got %d", len(files))
	}
}

func TestGetAllIndexedFiles_MultipleFiles(t *testing.T) {
	engine, cleanup := createTestDB(t)
	defer cleanup()

	expectedFiles := []string{
		"main.go",
		"utils.go",
		"internal/handler.go",
		"pkg/lib.go",
	}

	for _, f := range expectedFiles {
		info := &FileInfo{
			FilePath:     f,
			ContentHash:  "hash",
			Embedding:    createTestEmbedding(),
			LastModified: 1000,
			FileSize:     100,
			IndexedAt:    1000,
		}
		if err := storeFileInfo(engine.db, info); err != nil {
			t.Fatalf("storeFileInfo(%s) failed: %v", f, err)
		}
	}

	files, err := getAllIndexedFiles(engine.db)
	if err != nil {
		t.Fatalf("getAllIndexedFiles failed: %v", err)
	}

	if len(files) != len(expectedFiles) {
		t.Errorf("expected %d files, got %d", len(expectedFiles), len(files))
	}

	// Verify all expected files are present
	fileSet := make(map[string]bool)
	for _, f := range files {
		fileSet[f] = true
	}

	for _, expected := range expectedFiles {
		if !fileSet[expected] {
			t.Errorf("missing file: %s", expected)
		}
	}
}

func TestGetIndexStats_Empty(t *testing.T) {
	engine, cleanup := createTestDB(t)
	defer cleanup()

	stats, err := getIndexStats(engine.db)
	if err != nil {
		t.Fatalf("getIndexStats failed: %v", err)
	}

	if stats["total_files"].(int64) != 0 {
		t.Errorf("expected 0 total_files, got %v", stats["total_files"])
	}

	if stats["total_size"].(int64) != 0 {
		t.Errorf("expected 0 total_size, got %v", stats["total_size"])
	}
}

func TestGetIndexStats_WithFiles(t *testing.T) {
	engine, cleanup := createTestDB(t)
	defer cleanup()

	now := time.Now().Unix()

	// Store some files
	files := []struct {
		path      string
		size      int64
		indexedAt int64
	}{
		{"file1.go", 100, now - 100},
		{"file2.go", 200, now - 50},
		{"file3.go", 300, now},
	}

	for _, f := range files {
		info := &FileInfo{
			FilePath:     f.path,
			ContentHash:  "hash",
			Embedding:    createTestEmbedding(),
			LastModified: 1000,
			FileSize:     f.size,
			IndexedAt:    f.indexedAt,
		}
		if err := storeFileInfo(engine.db, info); err != nil {
			t.Fatalf("storeFileInfo(%s) failed: %v", f.path, err)
		}
	}

	stats, err := getIndexStats(engine.db)
	if err != nil {
		t.Fatalf("getIndexStats failed: %v", err)
	}

	// Check total files
	if stats["total_files"].(int64) != 3 {
		t.Errorf("expected 3 total_files, got %v", stats["total_files"])
	}

	// Check total size (100 + 200 + 300 = 600)
	if stats["total_size"].(int64) != 600 {
		t.Errorf("expected 600 total_size, got %v", stats["total_size"])
	}

	// Check oldest/newest index times
	oldestIndex := stats["oldest_index"].(time.Time)
	newestIndex := stats["newest_index"].(time.Time)

	if oldestIndex.Unix() != now-100 {
		t.Errorf("oldest_index mismatch: expected %d, got %d", now-100, oldestIndex.Unix())
	}
	if newestIndex.Unix() != now {
		t.Errorf("newest_index mismatch: expected %d, got %d", now, newestIndex.Unix())
	}
}

func TestGetIndexStats_SingleFile(t *testing.T) {
	engine, cleanup := createTestDB(t)
	defer cleanup()

	now := time.Now().Unix()

	info := &FileInfo{
		FilePath:     "single.go",
		ContentHash:  "hash",
		Embedding:    createTestEmbedding(),
		LastModified: 1000,
		FileSize:     500,
		IndexedAt:    now,
	}
	if err := storeFileInfo(engine.db, info); err != nil {
		t.Fatalf("storeFileInfo failed: %v", err)
	}

	stats, err := getIndexStats(engine.db)
	if err != nil {
		t.Fatalf("getIndexStats failed: %v", err)
	}

	if stats["total_files"].(int64) != 1 {
		t.Errorf("expected 1 total_files, got %v", stats["total_files"])
	}

	// For single file, oldest == newest
	oldestIndex := stats["oldest_index"].(time.Time)
	newestIndex := stats["newest_index"].(time.Time)

	if oldestIndex.Unix() != newestIndex.Unix() {
		t.Errorf("for single file, oldest and newest should be equal")
	}
}

func TestStoreFileInfo_SpecialCharacters(t *testing.T) {
	engine, cleanup := createTestDB(t)
	defer cleanup()

	specialPaths := []string{
		"path with spaces/file.go",
		"path/with'quotes/file.go",
		"path/日本語/ファイル.go",
		"path/emoji🎉/file.go",
	}

	for _, path := range specialPaths {
		t.Run(path, func(t *testing.T) {
			info := &FileInfo{
				FilePath:     path,
				ContentHash:  "hash",
				Embedding:    createTestEmbedding(),
				LastModified: 1000,
				FileSize:     100,
				IndexedAt:    1000,
			}

			if err := storeFileInfo(engine.db, info); err != nil {
				t.Fatalf("storeFileInfo failed: %v", err)
			}

			retrieved, err := getFileInfo(engine.db, path)
			if err != nil {
				t.Fatalf("getFileInfo failed: %v", err)
			}

			if retrieved.FilePath != path {
				t.Errorf("path mismatch: expected %q, got %q", path, retrieved.FilePath)
			}
		})
	}
}

func TestStoreFileInfo_EmptyEmbedding(t *testing.T) {
	engine, cleanup := createTestDB(t)
	defer cleanup()

	// Store with empty embedding - should fail due to NOT NULL constraint
	info := &FileInfo{
		FilePath:     "empty_emb.go",
		ContentHash:  "hash",
		Embedding:    []float32{},
		LastModified: 1000,
		FileSize:     100,
		IndexedAt:    1000,
	}

	err := storeFileInfo(engine.db, info)
	if err == nil {
		t.Error("expected error for empty embedding due to NOT NULL constraint")
	}
}

func TestStoreFileInfo_LargeContentHash(t *testing.T) {
	engine, cleanup := createTestDB(t)
	defer cleanup()

	// Create a very long content hash (simulating SHA256 hex)
	longHash := ""
	for i := 0; i < 64; i++ {
		longHash += "a"
	}

	info := &FileInfo{
		FilePath:     "test.go",
		ContentHash:  longHash,
		Embedding:    createTestEmbedding(),
		LastModified: 1000,
		FileSize:     100,
		IndexedAt:    1000,
	}

	if err := storeFileInfo(engine.db, info); err != nil {
		t.Fatalf("storeFileInfo failed: %v", err)
	}

	retrieved, err := getFileInfo(engine.db, "test.go")
	if err != nil {
		t.Fatalf("getFileInfo failed: %v", err)
	}

	if retrieved.ContentHash != longHash {
		t.Errorf("hash mismatch: lengths %d vs %d", len(longHash), len(retrieved.ContentHash))
	}
}

func TestFileInfoRoundTrip(t *testing.T) {
	engine, cleanup := createTestDB(t)
	defer cleanup()

	original := &FileInfo{
		FilePath:     "roundtrip/test.go",
		ContentHash:  "abc123xyz789",
		Embedding:    createTestEmbedding(),
		LastModified: 1700000000,
		FileSize:     12345,
		IndexedAt:    1700000001,
	}

	if err := storeFileInfo(engine.db, original); err != nil {
		t.Fatalf("storeFileInfo failed: %v", err)
	}

	retrieved, err := getFileInfo(engine.db, original.FilePath)
	if err != nil {
		t.Fatalf("getFileInfo failed: %v", err)
	}

	// Compare all fields
	if retrieved.FilePath != original.FilePath {
		t.Errorf("FilePath: expected %q, got %q", original.FilePath, retrieved.FilePath)
	}
	if retrieved.ContentHash != original.ContentHash {
		t.Errorf("ContentHash: expected %q, got %q", original.ContentHash, retrieved.ContentHash)
	}
	if retrieved.LastModified != original.LastModified {
		t.Errorf("LastModified: expected %d, got %d", original.LastModified, retrieved.LastModified)
	}
	if retrieved.FileSize != original.FileSize {
		t.Errorf("FileSize: expected %d, got %d", original.FileSize, retrieved.FileSize)
	}
	if retrieved.IndexedAt != original.IndexedAt {
		t.Errorf("IndexedAt: expected %d, got %d", original.IndexedAt, retrieved.IndexedAt)
	}
	if len(retrieved.Embedding) != len(original.Embedding) {
		t.Errorf("Embedding length: expected %d, got %d", len(original.Embedding), len(retrieved.Embedding))
	}
}

// Benchmark tests
func BenchmarkStoreFileInfo(b *testing.B) {
	tmpDir := b.TempDir()
	dbPath := filepath.Join(tmpDir, "bench.db")

	db, err := infrastructure.InitSearchDB(dbPath)
	if err != nil {
		b.Fatalf("InitSearchDB failed: %v", err)
	}
	defer db.Close()

	emb := createTestEmbedding()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		info := &FileInfo{
			FilePath:     "file" + string(rune('0'+i%10)) + ".go",
			ContentHash:  "hash",
			Embedding:    emb,
			LastModified: int64(i),
			FileSize:     100,
			IndexedAt:    int64(i),
		}
		storeFileInfo(db, info)
	}
}

func BenchmarkGetFileInfo(b *testing.B) {
	tmpDir := b.TempDir()
	dbPath := filepath.Join(tmpDir, "bench.db")

	db, err := infrastructure.InitSearchDB(dbPath)
	if err != nil {
		b.Fatalf("InitSearchDB failed: %v", err)
	}
	defer db.Close()

	// Store test file
	info := &FileInfo{
		FilePath:     "test.go",
		ContentHash:  "hash",
		Embedding:    createTestEmbedding(),
		LastModified: 1000,
		FileSize:     100,
		IndexedAt:    1000,
	}
	storeFileInfo(db, info)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		getFileInfo(db, "test.go")
	}
}

func BenchmarkGetAllIndexedFiles(b *testing.B) {
	tmpDir := b.TempDir()
	dbPath := filepath.Join(tmpDir, "bench.db")

	db, err := infrastructure.InitSearchDB(dbPath)
	if err != nil {
		b.Fatalf("InitSearchDB failed: %v", err)
	}
	defer db.Close()

	// Store 100 files
	emb := createTestEmbedding()
	for i := 0; i < 100; i++ {
		info := &FileInfo{
			FilePath:     "file" + string(rune('0'+i/10)) + string(rune('0'+i%10)) + ".go",
			ContentHash:  "hash",
			Embedding:    emb,
			LastModified: int64(i),
			FileSize:     100,
			IndexedAt:    int64(i),
		}
		storeFileInfo(db, info)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		getAllIndexedFiles(db)
	}
}

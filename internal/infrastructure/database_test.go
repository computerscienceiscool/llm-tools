package infrastructure

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"
)

func TestInitSearchDB_Success(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := InitSearchDB(dbPath)
	if err != nil {
		t.Fatalf("InitSearchDB failed: %v", err)
	}
	defer db.Close()

	if db == nil {
		t.Fatal("expected non-nil database connection")
	}

	// Verify database file was created
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Error("database file was not created")
	}
}

func TestInitSearchDB_CreatesDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	nestedPath := filepath.Join(tmpDir, "a", "b", "c", "test.db")

	db, err := InitSearchDB(nestedPath)
	if err != nil {
		t.Fatalf("InitSearchDB failed to create nested directories: %v", err)
	}
	defer db.Close()

	// Verify nested directories were created
	if _, err := os.Stat(filepath.Dir(nestedPath)); os.IsNotExist(err) {
		t.Error("nested directories were not created")
	}
}

func TestInitSearchDB_CreatesEmbeddingsTable(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := InitSearchDB(dbPath)
	if err != nil {
		t.Fatalf("InitSearchDB failed: %v", err)
	}
	defer db.Close()

	// Check that embeddings table exists
	var tableName string
	err = db.QueryRow(`
		SELECT name FROM sqlite_master 
		WHERE type='table' AND name='embeddings'
	`).Scan(&tableName)

	if err != nil {
		t.Fatalf("embeddings table not found: %v", err)
	}

	if tableName != "embeddings" {
		t.Errorf("expected table name 'embeddings', got %q", tableName)
	}
}

func TestInitSearchDB_TableSchema(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := InitSearchDB(dbPath)
	if err != nil {
		t.Fatalf("InitSearchDB failed: %v", err)
	}
	defer db.Close()

	// Get table info
	rows, err := db.Query("PRAGMA table_info(embeddings)")
	if err != nil {
		t.Fatalf("failed to get table info: %v", err)
	}
	defer rows.Close()

	expectedColumns := map[string]string{
		"filepath":      "TEXT",
		"content_hash":  "TEXT",
		"embedding":     "BLOB",
		"last_modified": "INTEGER",
		"file_size":     "INTEGER",
		"indexed_at":    "INTEGER",
	}

	foundColumns := make(map[string]bool)

	for rows.Next() {
		var cid int
		var name, colType string
		var notNull, pk int
		var dfltValue interface{}

		if err := rows.Scan(&cid, &name, &colType, &notNull, &dfltValue, &pk); err != nil {
			t.Fatalf("failed to scan column info: %v", err)
		}

		foundColumns[name] = true

		if expectedType, ok := expectedColumns[name]; ok {
			if colType != expectedType {
				t.Errorf("column %s: expected type %s, got %s", name, expectedType, colType)
			}
		}
	}

	// Verify all expected columns exist
	for col := range expectedColumns {
		if !foundColumns[col] {
			t.Errorf("missing column: %s", col)
		}
	}
}

func TestInitSearchDB_CreatesIndexes(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := InitSearchDB(dbPath)
	if err != nil {
		t.Fatalf("InitSearchDB failed: %v", err)
	}
	defer db.Close()

	// Check for indexes
	rows, err := db.Query(`
		SELECT name FROM sqlite_master 
		WHERE type='index' AND tbl_name='embeddings'
	`)
	if err != nil {
		t.Fatalf("failed to query indexes: %v", err)
	}
	defer rows.Close()

	indexes := make(map[string]bool)
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			t.Fatalf("failed to scan index name: %v", err)
		}
		indexes[name] = true
	}

	expectedIndexes := []string{"idx_hash", "idx_modified"}
	for _, idx := range expectedIndexes {
		if !indexes[idx] {
			t.Errorf("missing index: %s", idx)
		}
	}
}

func TestInitSearchDB_Idempotent(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	// Initialize once
	db1, err := InitSearchDB(dbPath)
	if err != nil {
		t.Fatalf("first InitSearchDB failed: %v", err)
	}

	// Insert some data
	_, err = db1.Exec(`
		INSERT INTO embeddings (filepath, content_hash, embedding, last_modified, file_size, indexed_at)
		VALUES ('test.go', 'hash123', X'0102030405', 1234567890, 100, 1234567890)
	`)
	if err != nil {
		t.Fatalf("failed to insert test data: %v", err)
	}
	db1.Close()

	// Initialize again - should not lose data
	db2, err := InitSearchDB(dbPath)
	if err != nil {
		t.Fatalf("second InitSearchDB failed: %v", err)
	}
	defer db2.Close()

	// Verify data persisted
	var count int
	err = db2.QueryRow("SELECT COUNT(*) FROM embeddings").Scan(&count)
	if err != nil {
		t.Fatalf("failed to count rows: %v", err)
	}

	if count != 1 {
		t.Errorf("expected 1 row after re-init, got %d", count)
	}
}

func TestInitSearchDB_CanInsertAndQuery(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := InitSearchDB(dbPath)
	if err != nil {
		t.Fatalf("InitSearchDB failed: %v", err)
	}
	defer db.Close()

	// Insert test data
	embedding := []byte{0x01, 0x02, 0x03, 0x04, 0x05}
	_, err = db.Exec(`
		INSERT INTO embeddings (filepath, content_hash, embedding, last_modified, file_size, indexed_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`, "internal/main.go", "abc123", embedding, 1700000000, 1024, 1700000001)

	if err != nil {
		t.Fatalf("failed to insert: %v", err)
	}

	// Query back
	var filepath, hash string
	var embBytes []byte
	var lastMod, size, indexedAt int64

	err = db.QueryRow(`
		SELECT filepath, content_hash, embedding, last_modified, file_size, indexed_at
		FROM embeddings WHERE filepath = ?
	`, "internal/main.go").Scan(&filepath, &hash, &embBytes, &lastMod, &size, &indexedAt)

	if err != nil {
		t.Fatalf("failed to query: %v", err)
	}

	if filepath != "internal/main.go" {
		t.Errorf("filepath mismatch: got %q", filepath)
	}
	if hash != "abc123" {
		t.Errorf("hash mismatch: got %q", hash)
	}
	if len(embBytes) != 5 {
		t.Errorf("embedding length mismatch: got %d", len(embBytes))
	}
	if lastMod != 1700000000 {
		t.Errorf("last_modified mismatch: got %d", lastMod)
	}
	if size != 1024 {
		t.Errorf("file_size mismatch: got %d", size)
	}
}

func TestInitSearchDB_PrimaryKeyConstraint(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := InitSearchDB(dbPath)
	if err != nil {
		t.Fatalf("InitSearchDB failed: %v", err)
	}
	defer db.Close()

	// Insert first record
	_, err = db.Exec(`
		INSERT INTO embeddings (filepath, content_hash, embedding, last_modified, file_size, indexed_at)
		VALUES ('test.go', 'hash1', X'01', 1, 100, 1)
	`)
	if err != nil {
		t.Fatalf("first insert failed: %v", err)
	}

	// Try to insert duplicate - should fail
	_, err = db.Exec(`
		INSERT INTO embeddings (filepath, content_hash, embedding, last_modified, file_size, indexed_at)
		VALUES ('test.go', 'hash2', X'02', 2, 200, 2)
	`)
	if err == nil {
		t.Error("expected error for duplicate primary key, got nil")
	}
}

func TestInitSearchDB_ReplaceOnDuplicate(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := InitSearchDB(dbPath)
	if err != nil {
		t.Fatalf("InitSearchDB failed: %v", err)
	}
	defer db.Close()

	// Insert first record
	_, err = db.Exec(`
		INSERT INTO embeddings (filepath, content_hash, embedding, last_modified, file_size, indexed_at)
		VALUES ('test.go', 'hash1', X'01', 1, 100, 1)
	`)
	if err != nil {
		t.Fatalf("first insert failed: %v", err)
	}

	// Use INSERT OR REPLACE to update
	_, err = db.Exec(`
		INSERT OR REPLACE INTO embeddings (filepath, content_hash, embedding, last_modified, file_size, indexed_at)
		VALUES ('test.go', 'hash2', X'02', 2, 200, 2)
	`)
	if err != nil {
		t.Fatalf("replace failed: %v", err)
	}

	// Verify updated values
	var hash string
	var size int64
	err = db.QueryRow("SELECT content_hash, file_size FROM embeddings WHERE filepath = ?", "test.go").Scan(&hash, &size)
	if err != nil {
		t.Fatalf("query failed: %v", err)
	}

	if hash != "hash2" {
		t.Errorf("expected hash 'hash2', got %q", hash)
	}
	if size != 200 {
		t.Errorf("expected size 200, got %d", size)
	}
}

func TestInitSearchDB_InMemory(t *testing.T) {
	// Test with in-memory database
	db, err := InitSearchDB(":memory:")
	if err != nil {
		t.Fatalf("InitSearchDB with :memory: failed: %v", err)
	}
	defer db.Close()

	// Verify table exists
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM embeddings").Scan(&count)
	if err != nil {
		t.Fatalf("query on in-memory db failed: %v", err)
	}

	if count != 0 {
		t.Errorf("expected 0 rows in new db, got %d", count)
	}
}

func TestInitSearchDB_MultipleConnections(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	// Open first connection
	db1, err := InitSearchDB(dbPath)
	if err != nil {
		t.Fatalf("first InitSearchDB failed: %v", err)
	}
	defer db1.Close()

	// Open second connection
	db2, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatalf("second connection failed: %v", err)
	}
	defer db2.Close()

	// Insert via db1
	_, err = db1.Exec(`
		INSERT INTO embeddings (filepath, content_hash, embedding, last_modified, file_size, indexed_at)
		VALUES ('test.go', 'hash1', X'01', 1, 100, 1)
	`)
	if err != nil {
		t.Fatalf("insert via db1 failed: %v", err)
	}

	// Query via db2
	var count int
	err = db2.QueryRow("SELECT COUNT(*) FROM embeddings").Scan(&count)
	if err != nil {
		t.Fatalf("query via db2 failed: %v", err)
	}

	if count != 1 {
		t.Errorf("expected 1 row visible from second connection, got %d", count)
	}
}

func TestInitSearchDB_LargeEmbedding(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := InitSearchDB(dbPath)
	if err != nil {
		t.Fatalf("InitSearchDB failed: %v", err)
	}
	defer db.Close()

	// Create a large embedding (384 float32s = 1536 bytes)
	largeEmbedding := make([]byte, 384*4)
	for i := range largeEmbedding {
		largeEmbedding[i] = byte(i % 256)
	}

	_, err = db.Exec(`
		INSERT INTO embeddings (filepath, content_hash, embedding, last_modified, file_size, indexed_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`, "large.go", "hash", largeEmbedding, 1, 100, 1)

	if err != nil {
		t.Fatalf("failed to insert large embedding: %v", err)
	}

	// Query back and verify size
	var embBytes []byte
	err = db.QueryRow("SELECT embedding FROM embeddings WHERE filepath = ?", "large.go").Scan(&embBytes)
	if err != nil {
		t.Fatalf("failed to query large embedding: %v", err)
	}

	if len(embBytes) != 384*4 {
		t.Errorf("expected embedding size %d, got %d", 384*4, len(embBytes))
	}
}

func TestInitSearchDB_SpecialCharactersInPath(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := InitSearchDB(dbPath)
	if err != nil {
		t.Fatalf("InitSearchDB failed: %v", err)
	}
	defer db.Close()

	// Test various special characters in filepath
	specialPaths := []string{
		"path/with spaces/file.go",
		"path/with-dashes/file.go",
		"path/with_underscores/file.go",
		"path/with.dots/file.go",
		"path/with'quotes/file.go",
		"path/with\"doublequotes/file.go",
		"path/日本語/file.go",
		"path/émojis🎉/file.go",
	}

	for i, path := range specialPaths {
		_, err = db.Exec(`
			INSERT INTO embeddings (filepath, content_hash, embedding, last_modified, file_size, indexed_at)
			VALUES (?, ?, ?, ?, ?, ?)
		`, path, "hash"+string(rune('0'+i)), []byte{0x01}, 1, 100, 1)

		if err != nil {
			t.Errorf("failed to insert path %q: %v", path, err)
			continue
		}

		// Query back
		var retrieved string
		err = db.QueryRow("SELECT filepath FROM embeddings WHERE filepath = ?", path).Scan(&retrieved)
		if err != nil {
			t.Errorf("failed to query path %q: %v", path, err)
			continue
		}

		if retrieved != path {
			t.Errorf("path mismatch: expected %q, got %q", path, retrieved)
		}
	}
}

func TestInitSearchDB_DeleteOperation(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := InitSearchDB(dbPath)
	if err != nil {
		t.Fatalf("InitSearchDB failed: %v", err)
	}
	defer db.Close()

	// Insert
	_, err = db.Exec(`
		INSERT INTO embeddings (filepath, content_hash, embedding, last_modified, file_size, indexed_at)
		VALUES ('test.go', 'hash1', X'01', 1, 100, 1)
	`)
	if err != nil {
		t.Fatalf("insert failed: %v", err)
	}

	// Delete
	result, err := db.Exec("DELETE FROM embeddings WHERE filepath = ?", "test.go")
	if err != nil {
		t.Fatalf("delete failed: %v", err)
	}

	affected, _ := result.RowsAffected()
	if affected != 1 {
		t.Errorf("expected 1 row affected, got %d", affected)
	}

	// Verify deleted
	var count int
	db.QueryRow("SELECT COUNT(*) FROM embeddings").Scan(&count)
	if count != 0 {
		t.Errorf("expected 0 rows after delete, got %d", count)
	}
}

// Benchmark tests
func BenchmarkInitSearchDB(b *testing.B) {
	tmpDir := b.TempDir()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		dbPath := filepath.Join(tmpDir, "bench.db")
		db, err := InitSearchDB(dbPath)
		if err != nil {
			b.Fatalf("InitSearchDB failed: %v", err)
		}
		db.Close()
		os.Remove(dbPath)
	}
}

func BenchmarkInsertEmbedding(b *testing.B) {
	tmpDir := b.TempDir()
	dbPath := filepath.Join(tmpDir, "bench.db")

	db, err := InitSearchDB(dbPath)
	if err != nil {
		b.Fatalf("InitSearchDB failed: %v", err)
	}
	defer db.Close()

	embedding := make([]byte, 384*4)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		filepath := "file" + string(rune('0'+i%10)) + ".go"
		db.Exec(`
			INSERT OR REPLACE INTO embeddings (filepath, content_hash, embedding, last_modified, file_size, indexed_at)
			VALUES (?, ?, ?, ?, ?, ?)
		`, filepath, "hash", embedding, i, 100, i)
	}
}

func BenchmarkQueryEmbedding(b *testing.B) {
	tmpDir := b.TempDir()
	dbPath := filepath.Join(tmpDir, "bench.db")

	db, err := InitSearchDB(dbPath)
	if err != nil {
		b.Fatalf("InitSearchDB failed: %v", err)
	}
	defer db.Close()

	// Insert test data
	embedding := make([]byte, 384*4)
	for i := 0; i < 100; i++ {
		filepath := "file" + string(rune('0'+i%10)) + "_" + string(rune('0'+i/10)) + ".go"
		db.Exec(`
			INSERT INTO embeddings (filepath, content_hash, embedding, last_modified, file_size, indexed_at)
			VALUES (?, ?, ?, ?, ?, ?)
		`, filepath, "hash", embedding, i, 100, i)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var embBytes []byte
		db.QueryRow("SELECT embedding FROM embeddings WHERE filepath = ?", "file5_5.go").Scan(&embBytes)
	}
}

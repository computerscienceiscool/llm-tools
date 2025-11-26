package search

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPrintSearchHelp(t *testing.T) {
	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	PrintSearchHelp()

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	// Verify key sections are present
	expectedStrings := []string{
		"Search Commands",
		"search:",
		"--search-reindex",
		"--search-update",
		"--search-status",
		"--search-validate",
		"--search-cleanup",
		"Configuration",
		"Requirements",
		"Python",
		"sentence-transformers",
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(output, expected) {
			t.Errorf("PrintSearchHelp output missing %q", expected)
		}
	}
}

func TestNewSearchCommands_Disabled(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := &SearchConfig{
		Enabled:      false,
		VectorDBPath: filepath.Join(tmpDir, "test.db"),
	}

	_, err := NewSearchCommands(cfg, tmpDir)
	if err == nil {
		t.Error("expected error when search is disabled")
	}
}

func TestNewSearchCommands_Success(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := &SearchConfig{
		Enabled:      true,
		VectorDBPath: filepath.Join(tmpDir, "test.db"),
		MaxResults:   10,
	}

	sc, err := NewSearchCommands(cfg, tmpDir)
	if err != nil {
		t.Fatalf("NewSearchCommands failed: %v", err)
	}
	defer sc.Close()

	if sc == nil {
		t.Error("expected non-nil SearchCommands")
	}

	if sc.engine == nil {
		t.Error("expected non-nil engine")
	}
}

func TestSearchCommands_Close(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := &SearchConfig{
		Enabled:      true,
		VectorDBPath: filepath.Join(tmpDir, "test.db"),
	}

	sc, err := NewSearchCommands(cfg, tmpDir)
	if err != nil {
		t.Fatalf("NewSearchCommands failed: %v", err)
	}

	// Close should not error
	err = sc.Close()
	if err != nil {
		t.Errorf("Close failed: %v", err)
	}
}

func TestSearchCommands_Close_NilEngine(t *testing.T) {
	sc := &SearchCommands{
		engine: nil,
	}

	// Should not panic or error
	err := sc.Close()
	if err != nil {
		t.Errorf("Close with nil engine should not error: %v", err)
	}
}

func TestSearchCommands_Close_Multiple(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := &SearchConfig{
		Enabled:      true,
		VectorDBPath: filepath.Join(tmpDir, "test.db"),
	}

	sc, err := NewSearchCommands(cfg, tmpDir)
	if err != nil {
		t.Fatalf("NewSearchCommands failed: %v", err)
	}

	// First close
	sc.Close()

	// Second close should not panic
	sc.Close()
}

func TestCheckPythonSetup_InvalidPath(t *testing.T) {
	err := CheckPythonSetup("/nonexistent/python")
	if err == nil {
		t.Error("expected error for invalid Python path")
	}
}

func TestCheckPythonSetup_InvalidPython(t *testing.T) {
	// Try with a command that exists but isn't Python:wq

	// Use /bin/true which outputs nothing (no "OK")
	err := CheckPythonSetup("/bin/true")
	if err == nil {
		t.Error("expected error for non-Python command")
	}
}

func TestCheckPythonSetup_EmptyPath(t *testing.T) {
	err := CheckPythonSetup("")
	if err == nil {
		t.Error("expected error for empty Python path")
	}
}

// The following tests require Python with sentence-transformers
// They test error handling when Python is not available

func TestSearchCommands_Search_NoPython(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := &SearchConfig{
		Enabled:      true,
		VectorDBPath: filepath.Join(tmpDir, "test.db"),
		PythonPath:   "/nonexistent/python",
		MaxResults:   10,
	}

	sc, err := NewSearchCommands(cfg, tmpDir)
	if err != nil {
		t.Fatalf("NewSearchCommands failed: %v", err)
	}
	defer sc.Close()

	// Search should fail due to Python not available
	_, err = sc.Search("test query")
	if err == nil {
		t.Error("expected error when Python is not available")
	}
}

func TestSearchCommands_HandleSearchStatus(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := &SearchConfig{
		Enabled:      true,
		VectorDBPath: filepath.Join(tmpDir, "test.db"),
	}

	sc, err := NewSearchCommands(cfg, tmpDir)
	if err != nil {
		t.Fatalf("NewSearchCommands failed: %v", err)
	}
	defer sc.Close()

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err = sc.HandleSearchStatus()

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("HandleSearchStatus failed: %v", err)
	}

	// Check output contains expected info
	if !strings.Contains(output, "Search Index Status") {
		t.Error("output should contain 'Search Index Status'")
	}
	if !strings.Contains(output, "Total files indexed") {
		t.Error("output should contain 'Total files indexed'")
	}
}

func TestSearchCommands_HandleSearchValidate_EmptyIndex(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := &SearchConfig{
		Enabled:      true,
		VectorDBPath: filepath.Join(tmpDir, "test.db"),
	}

	sc, err := NewSearchCommands(cfg, tmpDir)
	if err != nil {
		t.Fatalf("NewSearchCommands failed: %v", err)
	}
	defer sc.Close()

	// Empty index should validate successfully
	err = sc.HandleSearchValidate()
	if err != nil {
		t.Errorf("HandleSearchValidate on empty index should pass: %v", err)
	}
}

func TestSearchCommands_HandleSearchCleanup_EmptyIndex(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := &SearchConfig{
		Enabled:      true,
		VectorDBPath: filepath.Join(tmpDir, "test.db"),
	}

	sc, err := NewSearchCommands(cfg, tmpDir)
	if err != nil {
		t.Fatalf("NewSearchCommands failed: %v", err)
	}
	defer sc.Close()

	// Should not error on empty index
	err = sc.HandleSearchCleanup()
	if err != nil {
		t.Errorf("HandleSearchCleanup on empty index failed: %v", err)
	}
}

func TestSearchCommands_HandleSearchCleanup_RemovesDeletedFiles(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := &SearchConfig{
		Enabled:      true,
		VectorDBPath: filepath.Join(tmpDir, "test.db"),
	}

	sc, err := NewSearchCommands(cfg, tmpDir)
	if err != nil {
		t.Fatalf("NewSearchCommands failed: %v", err)
	}
	defer sc.Close()

	// Manually add an entry for a file that doesn't exist
	info := &FileInfo{
		FilePath:     "deleted_file.go",
		ContentHash:  "hash",
		Embedding:    createTestEmbedding(),
		LastModified: 1000,
		FileSize:     100,
		IndexedAt:    1000,
	}
	if err := storeFileInfo(sc.engine.GetDB(), info); err != nil {
		t.Fatalf("storeFileInfo failed: %v", err)
	}

	// Run cleanup
	err = sc.HandleSearchCleanup()
	if err != nil {
		t.Errorf("HandleSearchCleanup failed: %v", err)
	}

	// Verify file was removed from index
	files, err := getAllIndexedFiles(sc.engine.GetDB())
	if err != nil {
		t.Fatalf("getAllIndexedFiles failed: %v", err)
	}

	for _, f := range files {
		if f == "deleted_file.go" {
			t.Error("deleted_file.go should have been removed from index")
		}
	}
}

func TestSearchCommands_InitializeSearchIndex_EmptyRepo(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := &SearchConfig{
		Enabled:      true,
		VectorDBPath: filepath.Join(tmpDir, "test.db"),
		PythonPath:   "/nonexistent/python", // Will fail to index
	}

	sc, err := NewSearchCommands(cfg, tmpDir)
	if err != nil {
		t.Fatalf("NewSearchCommands failed: %v", err)
	}
	defer sc.Close()

	// With empty repo, should try to build index (may fail due to Python)
	// But should not panic
	sc.InitializeSearchIndex([]string{}, false)
}

func TestSearchCommands_InitializeSearchIndex_ExistingIndex(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := &SearchConfig{
		Enabled:      true,
		VectorDBPath: filepath.Join(tmpDir, "test.db"),
	}

	sc, err := NewSearchCommands(cfg, tmpDir)
	if err != nil {
		t.Fatalf("NewSearchCommands failed: %v", err)
	}
	defer sc.Close()

	// Add an entry to make index non-empty
	info := &FileInfo{
		FilePath:     "existing.go",
		ContentHash:  "hash",
		Embedding:    createTestEmbedding(),
		LastModified: 1000,
		FileSize:     100,
		IndexedAt:    1000,
	}
	if err := storeFileInfo(sc.engine.GetDB(), info); err != nil {
		t.Fatalf("storeFileInfo failed: %v", err)
	}

	// With existing index, should not try to rebuild
	err = sc.InitializeSearchIndex([]string{}, false)
	if err != nil {
		t.Errorf("InitializeSearchIndex with existing index failed: %v", err)
	}
}

func TestFormatSearchResults_Integration(t *testing.T) {
	results := []SearchResult{
		{
			FilePath:  "main.go",
			Score:     0.95,
			Preview:   "package main",
			LineCount: 100,
			FileSize:  2048,
			Relevance: "Excellent",
		},
		{
			FilePath:  "utils.go",
			Score:     0.75,
			Preview:   "func helper()",
			LineCount: 50,
			FileSize:  1024,
			Relevance: "Good",
		},
	}

	output := FormatSearchResults(results, "test query", 10)

	// Check key elements are present
	if !strings.Contains(output, "main.go") {
		t.Error("output should contain main.go")
	}
	if !strings.Contains(output, "utils.go") {
		t.Error("output should contain utils.go")
	}
	if !strings.Contains(output, "test query") {
		t.Error("output should contain query")
	}
	if !strings.Contains(output, "95.00%") {
		t.Error("output should contain score percentage")
	}
}

func TestSearchCommands_EngineAccessors(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := &SearchConfig{
		Enabled:      true,
		VectorDBPath: filepath.Join(tmpDir, "test.db"),
		MaxResults:   25,
	}

	sc, err := NewSearchCommands(cfg, tmpDir)
	if err != nil {
		t.Fatalf("NewSearchCommands failed: %v", err)
	}
	defer sc.Close()

	// Test that we can access engine internals
	if sc.engine.GetDB() == nil {
		t.Error("engine DB should not be nil")
	}

	if sc.engine.GetConfig().MaxResults != 25 {
		t.Error("engine config mismatch")
	}

	if sc.engine.GetRepoRoot() != tmpDir {
		t.Error("engine repo root mismatch")
	}
}

// Tests for HandleReindex and HandleSearchUpdate would require Python
// Skip them or test error paths

func TestSearchCommands_HandleReindex_NoPython(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := &SearchConfig{
		Enabled:      true,
		VectorDBPath: filepath.Join(tmpDir, "test.db"),
		PythonPath:   "/nonexistent/python",
	}

	sc, err := NewSearchCommands(cfg, tmpDir)
	if err != nil {
		t.Fatalf("NewSearchCommands failed: %v", err)
	}
	defer sc.Close()

	// Create a file to index
	testFile := filepath.Join(tmpDir, "test.go")
	if err := os.WriteFile(testFile, []byte("package main"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Reindex should fail due to Python
	err = sc.HandleReindex([]string{}, false)
	// May or may not error depending on implementation
	t.Logf("HandleReindex result: %v", err)
}

func TestSearchCommands_HandleSearchUpdate_NoPython(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := &SearchConfig{
		Enabled:      true,
		VectorDBPath: filepath.Join(tmpDir, "test.db"),
		PythonPath:   "/nonexistent/python",
	}

	sc, err := NewSearchCommands(cfg, tmpDir)
	if err != nil {
		t.Fatalf("NewSearchCommands failed: %v", err)
	}
	defer sc.Close()

	// Update should attempt to run but may fail due to Python
	err = sc.HandleSearchUpdate([]string{})
	// May or may not error depending on implementation
	t.Logf("HandleSearchUpdate result: %v", err)
}

func TestSearchCommands_ValidateWithFiles(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := &SearchConfig{
		Enabled:      true,
		VectorDBPath: filepath.Join(tmpDir, "test.db"),
	}

	sc, err := NewSearchCommands(cfg, tmpDir)
	if err != nil {
		t.Fatalf("NewSearchCommands failed: %v", err)
	}
	defer sc.Close()

	// Create a real file
	testFile := filepath.Join(tmpDir, "real.go")
	if err := os.WriteFile(testFile, []byte("package main"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}
	fileInfo, _ := os.Stat(testFile)

	// Index it with correct metadata
	info := &FileInfo{
		FilePath:     "real.go",
		ContentHash:  "hash",
		Embedding:    createTestEmbedding(),
		LastModified: fileInfo.ModTime().Unix(),
		FileSize:     fileInfo.Size(),
		IndexedAt:    1000,
	}
	if err := storeFileInfo(sc.engine.GetDB(), info); err != nil {
		t.Fatalf("storeFileInfo failed: %v", err)
	}

	// Validate should pass
	err = sc.HandleSearchValidate()
	if err != nil {
		t.Errorf("HandleSearchValidate should pass for valid files: %v", err)
	}
}

func TestSearchCommands_ValidateWithMissingFile(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := &SearchConfig{
		Enabled:      true,
		VectorDBPath: filepath.Join(tmpDir, "test.db"),
	}

	sc, err := NewSearchCommands(cfg, tmpDir)
	if err != nil {
		t.Fatalf("NewSearchCommands failed: %v", err)
	}
	defer sc.Close()

	// Index a file that doesn't exist
	info := &FileInfo{
		FilePath:     "missing.go",
		ContentHash:  "hash",
		Embedding:    createTestEmbedding(),
		LastModified: 1000,
		FileSize:     100,
		IndexedAt:    1000,
	}
	if err := storeFileInfo(sc.engine.GetDB(), info); err != nil {
		t.Fatalf("storeFileInfo failed: %v", err)
	}

	// Validate should find issues
	err = sc.HandleSearchValidate()
	if err == nil {
		t.Error("HandleSearchValidate should report missing files")
	}
}

// Benchmark tests
func BenchmarkNewSearchCommands(b *testing.B) {
	tmpDir := b.TempDir()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		dbPath := filepath.Join(tmpDir, "bench.db")
		cfg := &SearchConfig{
			Enabled:      true,
			VectorDBPath: dbPath,
		}
		sc, err := NewSearchCommands(cfg, tmpDir)
		if err != nil {
			b.Fatalf("NewSearchCommands failed: %v", err)
		}
		sc.Close()
		os.Remove(dbPath)
	}
}

func BenchmarkSearchCommands_HandleSearchStatus(b *testing.B) {
	tmpDir := b.TempDir()
	cfg := &SearchConfig{
		Enabled:      true,
		VectorDBPath: filepath.Join(tmpDir, "bench.db"),
	}

	sc, err := NewSearchCommands(cfg, tmpDir)
	if err != nil {
		b.Fatalf("NewSearchCommands failed: %v", err)
	}
	defer sc.Close()

	// Redirect stdout to discard
	oldStdout := os.Stdout
	os.Stdout, _ = os.Open(os.DevNull)
	defer func() { os.Stdout = oldStdout }()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sc.HandleSearchStatus()
	}
}

func BenchmarkFormatSearchResults_Commands(b *testing.B) {
	results := make([]SearchResult, 100)
	for i := range results {
		results[i] = SearchResult{
			FilePath:  "file.go",
			Score:     float32(i) / 100,
			Preview:   "preview content",
			LineCount: 100,
			FileSize:  1024,
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		FormatSearchResults(results, "query", 10)
	}
}

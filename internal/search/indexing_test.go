package search

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/computerscienceiscool/llm-runtime/internal/infrastructure"
)

func TestShouldIndexFile_ValidExtensions(t *testing.T) {
	indexExtensions := []string{".go", ".py", ".js", ".md", ".txt"}
	excludedPaths := []string{".git", "node_modules", "vendor"}

	tests := []struct {
		name     string
		filePath string
		expected bool
	}{
		{"Go file", "main.go", true},
		{"Python file", "script.py", true},
		{"JavaScript file", "app.js", true},
		{"Markdown file", "README.md", true},
		{"Text file", "notes.txt", true},
		{"Nested Go file", "internal/handler/main.go", true},
		{"Deep nested", "a/b/c/d/file.py", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := shouldIndexFile(tt.filePath, indexExtensions, excludedPaths)
			if result != tt.expected {
				t.Errorf("shouldIndexFile(%q) = %v, want %v", tt.filePath, result, tt.expected)
			}
		})
	}
}

func TestShouldIndexFile_InvalidExtensions(t *testing.T) {
	indexExtensions := []string{".go", ".py", ".js"}
	excludedPaths := []string{}

	tests := []struct {
		name     string
		filePath string
	}{
		{"Binary file", "program.exe"},
		{"Image file", "logo.png"},
		{"Archive file", "backup.zip"},
		{"Config file", "config.yaml"},
		{"No extension", "Makefile"},
		{"Hidden file", ".gitignore"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := shouldIndexFile(tt.filePath, indexExtensions, excludedPaths)
			if result != false {
				t.Errorf("shouldIndexFile(%q) = %v, want false", tt.filePath, result)
			}
		})
	}
}

func TestShouldIndexFile_ExcludedPaths(t *testing.T) {
	indexExtensions := []string{".go", ".py", ".js"}
	excludedPaths := []string{".git", "node_modules", "vendor", "*.test.go"}

	tests := []struct {
		name     string
		filePath string
		expected bool
	}{
		{"Git directory", ".git/config.go", false},
		{"Node modules", "node_modules/pkg/index.js", false},
		{"Vendor directory", "vendor/github.com/pkg/main.go", false},
		{"Test file pattern", "handler.test.go", false},
		{"Regular Go file", "main.go", true},
		{"Nested non-excluded", "internal/main.go", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := shouldIndexFile(tt.filePath, indexExtensions, excludedPaths)
			if result != tt.expected {
				t.Errorf("shouldIndexFile(%q) = %v, want %v", tt.filePath, result, tt.expected)
			}
		})
	}
}

func TestShouldIndexFile_CaseInsensitiveExtension(t *testing.T) {
	indexExtensions := []string{".go", ".GO", ".Go"}
	excludedPaths := []string{}

	tests := []struct {
		filePath string
		expected bool
	}{
		{"lowercase.go", true},
		{"uppercase.GO", true},
		{"mixed.Go", true},
		{"other.py", false},
	}

	for _, tt := range tests {
		t.Run(tt.filePath, func(t *testing.T) {
			result := shouldIndexFile(tt.filePath, indexExtensions, excludedPaths)
			if result != tt.expected {
				t.Errorf("shouldIndexFile(%q) = %v, want %v", tt.filePath, result, tt.expected)
			}
		})
	}
}

func TestShouldIndexFile_EmptyExtensionsList(t *testing.T) {
	result := shouldIndexFile("main.go", []string{}, []string{})
	if result != false {
		t.Error("empty extensions list should match nothing")
	}
}

func TestShouldIndexFile_EmptyFilePath(t *testing.T) {
	result := shouldIndexFile("", []string{".go"}, []string{})
	if result != false {
		t.Error("empty file path should not be indexed")
	}
}

func TestFileNeedsIndexing_NewFile(t *testing.T) {
	engine, cleanup := createTestDB(t)
	defer cleanup()

	// Create a real file
	testFile := filepath.Join(engine.repoRoot, "new.go")
	if err := os.WriteFile(testFile, []byte("package main"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	info, err := os.Stat(testFile)
	if err != nil {
		t.Fatalf("failed to stat file: %v", err)
	}

	// New file should need indexing
	needsIndexing, err := fileNeedsIndexing(engine.db, "new.go", info, false)
	if err != nil {
		t.Fatalf("fileNeedsIndexing failed: %v", err)
	}

	if !needsIndexing {
		t.Error("new file should need indexing")
	}
}

func TestFileNeedsIndexing_ExistingUnchanged(t *testing.T) {
	engine, cleanup := createTestDB(t)
	defer cleanup()

	// Create a real file
	testFile := filepath.Join(engine.repoRoot, "existing.go")
	if err := os.WriteFile(testFile, []byte("package main"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	info, err := os.Stat(testFile)
	if err != nil {
		t.Fatalf("failed to stat file: %v", err)
	}

	// Store in database with matching metadata
	fileInfo := &FileInfo{
		FilePath:     "existing.go",
		ContentHash:  "hash",
		Embedding:    createTestEmbedding(),
		LastModified: info.ModTime().Unix(),
		FileSize:     info.Size(),
		IndexedAt:    time.Now().Unix(),
	}
	if err := storeFileInfo(engine.db, fileInfo); err != nil {
		t.Fatalf("storeFileInfo failed: %v", err)
	}

	// File should not need indexing
	needsIndexing, err := fileNeedsIndexing(engine.db, "existing.go", info, false)
	if err != nil {
		t.Fatalf("fileNeedsIndexing failed: %v", err)
	}

	if needsIndexing {
		t.Error("unchanged file should not need indexing")
	}
}

func TestFileNeedsIndexing_ModifiedFile(t *testing.T) {
	engine, cleanup := createTestDB(t)
	defer cleanup()

	// Create a real file
	testFile := filepath.Join(engine.repoRoot, "modified.go")
	if err := os.WriteFile(testFile, []byte("package main"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	info, err := os.Stat(testFile)
	if err != nil {
		t.Fatalf("failed to stat file: %v", err)
	}

	// Store in database with OLD modification time
	fileInfo := &FileInfo{
		FilePath:     "modified.go",
		ContentHash:  "hash",
		Embedding:    createTestEmbedding(),
		LastModified: info.ModTime().Unix() - 1000, // Old timestamp
		FileSize:     info.Size(),
		IndexedAt:    time.Now().Unix(),
	}
	if err := storeFileInfo(engine.db, fileInfo); err != nil {
		t.Fatalf("storeFileInfo failed: %v", err)
	}

	// File should need indexing due to modified time
	needsIndexing, err := fileNeedsIndexing(engine.db, "modified.go", info, false)
	if err != nil {
		t.Fatalf("fileNeedsIndexing failed: %v", err)
	}

	if !needsIndexing {
		t.Error("modified file should need indexing")
	}
}

func TestFileNeedsIndexing_SizeChanged(t *testing.T) {
	engine, cleanup := createTestDB(t)
	defer cleanup()

	// Create a real file
	testFile := filepath.Join(engine.repoRoot, "resized.go")
	if err := os.WriteFile(testFile, []byte("package main\n\nfunc main() {}"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	info, err := os.Stat(testFile)
	if err != nil {
		t.Fatalf("failed to stat file: %v", err)
	}

	// Store in database with different size
	fileInfo := &FileInfo{
		FilePath:     "resized.go",
		ContentHash:  "hash",
		Embedding:    createTestEmbedding(),
		LastModified: info.ModTime().Unix(),
		FileSize:     info.Size() + 100, // Different size
		IndexedAt:    time.Now().Unix(),
	}
	if err := storeFileInfo(engine.db, fileInfo); err != nil {
		t.Fatalf("storeFileInfo failed: %v", err)
	}

	// File should need indexing due to size change
	needsIndexing, err := fileNeedsIndexing(engine.db, "resized.go", info, false)
	if err != nil {
		t.Fatalf("fileNeedsIndexing failed: %v", err)
	}

	if !needsIndexing {
		t.Error("resized file should need indexing")
	}
}

func TestFileNeedsIndexing_ForceReindex(t *testing.T) {
	engine, cleanup := createTestDB(t)
	defer cleanup()

	// Create a real file
	testFile := filepath.Join(engine.repoRoot, "force.go")
	if err := os.WriteFile(testFile, []byte("package main"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	info, err := os.Stat(testFile)
	if err != nil {
		t.Fatalf("failed to stat file: %v", err)
	}

	// Store in database with matching metadata
	fileInfo := &FileInfo{
		FilePath:     "force.go",
		ContentHash:  "hash",
		Embedding:    createTestEmbedding(),
		LastModified: info.ModTime().Unix(),
		FileSize:     info.Size(),
		IndexedAt:    time.Now().Unix(),
	}
	if err := storeFileInfo(engine.db, fileInfo); err != nil {
		t.Fatalf("storeFileInfo failed: %v", err)
	}

	// With forceReindex=true, should always need indexing
	needsIndexing, err := fileNeedsIndexing(engine.db, "force.go", info, true)
	if err != nil {
		t.Fatalf("fileNeedsIndexing failed: %v", err)
	}

	if !needsIndexing {
		t.Error("forceReindex should always return true")
	}
}

func TestCleanupIndex_RemovesDeletedFiles(t *testing.T) {
	engine, cleanup := createTestDB(t)
	defer cleanup()

	// Store some files in the index
	files := []string{"exists.go", "deleted.go", "also_exists.go"}
	for _, f := range files {
		fileInfo := &FileInfo{
			FilePath:     f,
			ContentHash:  "hash",
			Embedding:    createTestEmbedding(),
			LastModified: time.Now().Unix(),
			FileSize:     100,
			IndexedAt:    time.Now().Unix(),
		}
		if err := storeFileInfo(engine.db, fileInfo); err != nil {
			t.Fatalf("storeFileInfo(%s) failed: %v", f, err)
		}
	}

	// Create only some files on disk
	for _, f := range []string{"exists.go", "also_exists.go"} {
		fullPath := filepath.Join(engine.repoRoot, f)
		if err := os.WriteFile(fullPath, []byte("content"), 0644); err != nil {
			t.Fatalf("failed to create file: %v", err)
		}
	}
	// Note: deleted.go is NOT created on disk

	// Run cleanup
	if err := CleanupIndex(engine.db, engine.repoRoot); err != nil {
		t.Fatalf("CleanupIndex failed: %v", err)
	}

	// Verify deleted.go was removed from index
	allFiles, err := getAllIndexedFiles(engine.db)
	if err != nil {
		t.Fatalf("getAllIndexedFiles failed: %v", err)
	}

	for _, f := range allFiles {
		if f == "deleted.go" {
			t.Error("deleted.go should have been removed from index")
		}
	}

	// Verify existing files still in index
	if len(allFiles) != 2 {
		t.Errorf("expected 2 files after cleanup, got %d", len(allFiles))
	}
}

func TestCleanupIndex_EmptyIndex(t *testing.T) {
	engine, cleanup := createTestDB(t)
	defer cleanup()

	// Should not error on empty index
	if err := CleanupIndex(engine.db, engine.repoRoot); err != nil {
		t.Errorf("CleanupIndex on empty index should not error: %v", err)
	}
}

func TestCleanupIndex_AllFilesExist(t *testing.T) {
	engine, cleanup := createTestDB(t)
	defer cleanup()

	// Create files and index them
	files := []string{"a.go", "b.go", "c.go"}
	for _, f := range files {
		fullPath := filepath.Join(engine.repoRoot, f)
		if err := os.WriteFile(fullPath, []byte("content"), 0644); err != nil {
			t.Fatalf("failed to create file: %v", err)
		}

		fileInfo := &FileInfo{
			FilePath:     f,
			ContentHash:  "hash",
			Embedding:    createTestEmbedding(),
			LastModified: time.Now().Unix(),
			FileSize:     100,
			IndexedAt:    time.Now().Unix(),
		}
		if err := storeFileInfo(engine.db, fileInfo); err != nil {
			t.Fatalf("storeFileInfo(%s) failed: %v", f, err)
		}
	}

	// Run cleanup
	if err := CleanupIndex(engine.db, engine.repoRoot); err != nil {
		t.Fatalf("CleanupIndex failed: %v", err)
	}

	// Verify all files still in index
	allFiles, err := getAllIndexedFiles(engine.db)
	if err != nil {
		t.Fatalf("getAllIndexedFiles failed: %v", err)
	}

	if len(allFiles) != len(files) {
		t.Errorf("expected %d files after cleanup, got %d", len(files), len(allFiles))
	}
}

func TestValidateIndex_AllFilesValid(t *testing.T) {
	engine, cleanup := createTestDB(t)
	defer cleanup()

	// Create files and index them with correct metadata
	files := []string{"valid1.go", "valid2.go"}
	for _, f := range files {
		fullPath := filepath.Join(engine.repoRoot, f)
		if err := os.WriteFile(fullPath, []byte("content"), 0644); err != nil {
			t.Fatalf("failed to create file: %v", err)
		}

		info, _ := os.Stat(fullPath)

		fileInfo := &FileInfo{
			FilePath:     f,
			ContentHash:  "hash",
			Embedding:    createTestEmbedding(),
			LastModified: info.ModTime().Unix(),
			FileSize:     info.Size(),
			IndexedAt:    time.Now().Unix(),
		}
		if err := storeFileInfo(engine.db, fileInfo); err != nil {
			t.Fatalf("storeFileInfo(%s) failed: %v", f, err)
		}
	}

	// Validate should pass
	err := ValidateIndex(engine.db, engine.repoRoot)
	if err != nil {
		t.Errorf("ValidateIndex should pass for valid index: %v", err)
	}
}

func TestValidateIndex_MissingFile(t *testing.T) {
	engine, cleanup := createTestDB(t)
	defer cleanup()

	// Index a file that doesn't exist on disk
	fileInfo := &FileInfo{
		FilePath:     "missing.go",
		ContentHash:  "hash",
		Embedding:    createTestEmbedding(),
		LastModified: time.Now().Unix(),
		FileSize:     100,
		IndexedAt:    time.Now().Unix(),
	}
	if err := storeFileInfo(engine.db, fileInfo); err != nil {
		t.Fatalf("storeFileInfo failed: %v", err)
	}

	// Validate should find issues
	err := ValidateIndex(engine.db, engine.repoRoot)
	if err == nil {
		t.Error("ValidateIndex should report issues for missing files")
	}
}

func TestValidateIndex_ModifiedFile(t *testing.T) {
	engine, cleanup := createTestDB(t)
	defer cleanup()

	// Create a file
	fullPath := filepath.Join(engine.repoRoot, "modified.go")
	if err := os.WriteFile(fullPath, []byte("content"), 0644); err != nil {
		t.Fatalf("failed to create file: %v", err)
	}

	// Index with old modification time
	fileInfo := &FileInfo{
		FilePath:     "modified.go",
		ContentHash:  "hash",
		Embedding:    createTestEmbedding(),
		LastModified: time.Now().Unix() - 1000, // Old time
		FileSize:     100,
		IndexedAt:    time.Now().Unix(),
	}
	if err := storeFileInfo(engine.db, fileInfo); err != nil {
		t.Fatalf("storeFileInfo failed: %v", err)
	}

	// Validate should find the modification
	err := ValidateIndex(engine.db, engine.repoRoot)
	if err == nil {
		t.Error("ValidateIndex should report issues for modified files")
	}
}

func TestValidateIndex_EmptyIndex(t *testing.T) {
	engine, cleanup := createTestDB(t)
	defer cleanup()

	// Empty index should validate successfully
	err := ValidateIndex(engine.db, engine.repoRoot)
	if err != nil {
		t.Errorf("ValidateIndex on empty index should pass: %v", err)
	}
}

func TestIndexStats_PrintFormat(t *testing.T) {
	// This is a visual test - just ensure printIndexStats doesn't panic
	stats := &IndexStats{
		TotalFiles:   100,
		IndexedFiles: 80,
		SkippedFiles: 15,
		ErrorFiles:   5,
		StartTime:    time.Now().Add(-10 * time.Second),
		EndTime:      time.Now(),
		BytesIndexed: 1024 * 1024,
	}

	// Should not panic
	printIndexStats(stats)
}

func TestShouldIndexFile_DotFiles(t *testing.T) {
	indexExtensions := []string{".go", ".md"}
	excludedPaths := []string{}

	tests := []struct {
		filePath string
		expected bool
	}{
		{".hidden.go", true}, // filepath.Ext returns ".go"
		{"normal.go", true},
		{".gitignore", false}, // No extension
		{".env", false},       // No extension
	}
	for _, tt := range tests {
		t.Run(tt.filePath, func(t *testing.T) {
			result := shouldIndexFile(tt.filePath, indexExtensions, excludedPaths)
			if result != tt.expected {
				t.Errorf("shouldIndexFile(%q) = %v, want %v", tt.filePath, result, tt.expected)
			}
		})
	}
}

// Benchmark tests
func BenchmarkShouldIndexFile(b *testing.B) {
	indexExtensions := []string{".go", ".py", ".js", ".md", ".txt", ".yaml", ".json"}
	excludedPaths := []string{".git", "node_modules", "vendor", "__pycache__", ".venv"}

	paths := []string{
		"main.go",
		"internal/handler/service.go",
		"vendor/github.com/pkg/lib.go",
		"node_modules/package/index.js",
		"regular/path/file.py",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, p := range paths {
			shouldIndexFile(p, indexExtensions, excludedPaths)
		}
	}
}

func BenchmarkFileNeedsIndexing_CacheHit(b *testing.B) {
	tmpDir := b.TempDir()
	dbPath := filepath.Join(tmpDir, "bench.db")

	db, err := infrastructure.InitSearchDB(dbPath)
	if err != nil {
		b.Fatalf("InitSearchDB failed: %v", err)
	}
	defer db.Close()

	// Create a file and index it
	testFile := filepath.Join(tmpDir, "test.go")
	if err := os.WriteFile(testFile, []byte("package main"), 0644); err != nil {
		b.Fatalf("failed to create file: %v", err)
	}

	info, _ := os.Stat(testFile)

	fileInfo := &FileInfo{
		FilePath:     "test.go",
		ContentHash:  "hash",
		Embedding:    createTestEmbedding(),
		LastModified: info.ModTime().Unix(),
		FileSize:     info.Size(),
		IndexedAt:    time.Now().Unix(),
	}
	storeFileInfo(db, fileInfo)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		fileNeedsIndexing(db, "test.go", info, false)
	}
}

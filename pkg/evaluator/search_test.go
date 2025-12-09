package evaluator

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/computerscienceiscool/llm-runtime/internal/infrastructure"
	"github.com/computerscienceiscool/llm-runtime/internal/search"
)

func TestExecuteSearch_Disabled(t *testing.T) {
	cfg := newTestConfig(t.TempDir())

	searchCfg := &search.SearchConfig{
		Enabled: false,
	}

	audit := &testAuditLog{}
	result := ExecuteSearch("test query", cfg, searchCfg, audit.log)

	if result.Success {
		t.Error("expected failure when search is disabled")
	}

	if !strings.Contains(result.Error.Error(), "SEARCH_DISABLED") {
		t.Errorf("expected SEARCH_DISABLED error, got: %v", result.Error)
	}

	// Check audit log
	entries := audit.getEntries()
	if len(entries) != 1 {
		t.Fatalf("expected 1 audit entry, got %d", len(entries))
	}
	if entries[0].success {
		t.Error("audit should show failure")
	}
}

func TestExecuteSearch_NilSearchConfig(t *testing.T) {
	cfg := newTestConfig(t.TempDir())

	result := ExecuteSearch("test query", cfg, nil, nil)

	if result.Success {
		t.Error("expected failure when search config is nil")
	}

	if !strings.Contains(result.Error.Error(), "SEARCH_DISABLED") {
		t.Errorf("expected SEARCH_DISABLED error, got: %v", result.Error)
	}
}

func TestExecuteSearch_CommandType(t *testing.T) {
	cfg := newTestConfig(t.TempDir())

	result := ExecuteSearch("my query", cfg, nil, nil)

	if result.Command.Type != "search" {
		t.Errorf("expected command type 'search', got %q", result.Command.Type)
	}

	if result.Command.Argument != "my query" {
		t.Errorf("expected argument 'my query', got %q", result.Command.Argument)
	}
}

func TestExecuteSearch_ExecutionTime(t *testing.T) {
	cfg := newTestConfig(t.TempDir())

	result := ExecuteSearch("test", cfg, nil, nil)

	if result.ExecutionTime <= 0 {
		t.Error("execution time should be positive")
	}
}

func TestExecuteSearch_NilAuditLog(t *testing.T) {
	cfg := newTestConfig(t.TempDir())

	searchCfg := &search.SearchConfig{
		Enabled: false,
	}

	// Should not panic with nil audit log
	result := ExecuteSearch("test", cfg, searchCfg, nil)

	if result.Success {
		t.Error("expected failure")
	}
}

func TestExecuteSearch_AuditLogOnFailure(t *testing.T) {
	cfg := newTestConfig(t.TempDir())

	searchCfg := &search.SearchConfig{
		Enabled: false,
	}

	audit := &testAuditLog{}
	ExecuteSearch("test query", cfg, searchCfg, audit.log)

	entries := audit.getEntries()
	if len(entries) != 1 {
		t.Fatalf("expected 1 audit entry, got %d", len(entries))
	}

	entry := entries[0]
	if entry.cmdType != "search" {
		t.Errorf("expected cmdType 'search', got %q", entry.cmdType)
	}
	if entry.arg != "test query" {
		t.Errorf("expected arg 'test query', got %q", entry.arg)
	}
	if entry.success {
		t.Error("expected success=false")
	}
}

func TestExecuteSearch_InitFailure(t *testing.T) {
	cfg := newTestConfig(t.TempDir())

	// Invalid database path should cause init failure
	searchCfg := &search.SearchConfig{
		Enabled:      true,
		VectorDBPath: "/nonexistent/path/that/cannot/be/created/\x00/db.sqlite",
	}

	result := ExecuteSearch("test", cfg, searchCfg, nil)

	if result.Success {
		t.Error("expected failure for invalid db path")
	}

	if !strings.Contains(result.Error.Error(), "SEARCH_INIT_FAILED") {
		t.Errorf("expected SEARCH_INIT_FAILED error, got: %v", result.Error)
	}
}

func TestExecuteSearch_EmptyQuery(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := newTestConfig(tmpDir)

	searchCfg := &search.SearchConfig{
		Enabled:            true,
		VectorDBPath:       filepath.Join(tmpDir, "test.db"),
		MaxResults:         10,
		MinSimilarityScore: 0.5,
		OllamaURL:         "/nonexistent/python", // Will fail at search, not init
	}

	result := ExecuteSearch("", cfg, searchCfg, nil)

	// Empty query should still attempt to search (and fail due to Python)
	t.Logf("Empty query result: success=%v, error=%v", result.Success, result.Error)
}

func TestExecuteSearch_WithValidDB(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := newTestConfig(tmpDir)

	dbPath := filepath.Join(tmpDir, "search.db")

	// Initialize database
	db, err := infrastructure.InitSearchDB(dbPath)
	if err != nil {
		t.Fatalf("failed to init db: %v", err)
	}
	db.Close()

	searchCfg := &search.SearchConfig{
		Enabled:            true,
		VectorDBPath:       dbPath,
		MaxResults:         10,
		MinSimilarityScore: 0.5,
		MaxPreviewLength:   100,
		OllamaURL:         "/nonexistent/python", // Will fail at Python check
	}

	result := ExecuteSearch("test query", cfg, searchCfg, nil)

	// Should fail at Python check, not database init
	if result.Success {
		t.Error("expected failure (Python not available)")
	}

	// Should be SEARCH_FAILED, not SEARCH_INIT_FAILED
	if strings.Contains(result.Error.Error(), "SEARCH_INIT_FAILED") {
		t.Error("should not fail at init stage with valid DB")
	}
}

func TestExecuteSearch_ResultFormatting(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := newTestConfig(tmpDir)

	searchCfg := &search.SearchConfig{
		Enabled: false, // Disabled to avoid needing Python
	}

	result := ExecuteSearch("test query", cfg, searchCfg, nil)

	// Even on failure, command metadata should be correct
	if result.Command.Type != "search" {
		t.Errorf("command type should be 'search'")
	}
	if result.Command.Argument != "test query" {
		t.Errorf("argument should be preserved")
	}
}

func TestFormatSearchOutput(t *testing.T) {
	tests := []struct {
		name       string
		query      string
		results    []search.SearchResult
		maxResults int
		contains   []string
	}{
		{
			name:       "no results",
			query:      "test",
			results:    []search.SearchResult{},
			maxResults: 10,
			contains:   []string{"SEARCH", "test", "No files found"},
		},
		{
			name:  "single result",
			query: "main",
			results: []search.SearchResult{
				{FilePath: "main.go", Score: 0.85, LineCount: 100, FileSize: 2048},
			},
			maxResults: 10,
			contains:   []string{"main.go", "85.00"},
		},
		{
			name:  "multiple results",
			query: "handler",
			results: []search.SearchResult{
				{FilePath: "handler.go", Score: 0.90, LineCount: 50, FileSize: 1024},
				{FilePath: "handler_test.go", Score: 0.75, LineCount: 100, FileSize: 2048},
			},
			maxResults: 10,
			contains:   []string{"handler.go", "handler_test.go", "90.00", "75.00"},
		},
		{
			name:  "with preview",
			query: "func",
			results: []search.SearchResult{
				{FilePath: "main.go", Score: 0.85, Preview: "func main() {", LineCount: 10, FileSize: 256},
			},
			maxResults: 10,
			contains:   []string{"Preview", "func main()"},
		},
		{
			name:  "results limited by maxResults",
			query: "test",
			results: []search.SearchResult{
				{FilePath: "a.go", Score: 0.9, LineCount: 10, FileSize: 100},
				{FilePath: "b.go", Score: 0.8, LineCount: 10, FileSize: 100},
				{FilePath: "c.go", Score: 0.7, LineCount: 10, FileSize: 100},
			},
			maxResults: 2,
			contains:   []string{"Showing top 2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := formatSearchOutput(tt.query, tt.results, tt.maxResults, 0)

			for _, expected := range tt.contains {
				if !strings.Contains(output, expected) {
					t.Errorf("output should contain %q, got:\n%s", expected, output)
				}
			}
		})
	}
}

func TestFormatSearchOutput_Headers(t *testing.T) {
	output := formatSearchOutput("query", []search.SearchResult{}, 10, 0)

	if !strings.Contains(output, "=== SEARCH:") {
		t.Error("output should contain SEARCH header")
	}
	if !strings.Contains(output, "=== END SEARCH ===") {
		t.Error("output should contain END SEARCH footer")
	}
}

func TestFormatFileSizeForSearch(t *testing.T) {
	tests := []struct {
		size     int64
		expected string
	}{
		{0, "0 B"},
		{100, "100 B"},
		{1023, "1023 B"},
		{1024, "1.0 KB"},
		{1536, "1.5 KB"},
		{1024 * 1024, "1.0 MB"},
		{1024 * 1024 * 1024, "1.0 GB"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := formatFileSizeForSearch(tt.size)
			if result != tt.expected {
				t.Errorf("formatFileSizeForSearch(%d) = %q, want %q", tt.size, result, tt.expected)
			}
		})
	}
}

func TestExecuteSearch_VariousQueries(t *testing.T) {
	cfg := newTestConfig(t.TempDir())

	searchCfg := &search.SearchConfig{
		Enabled: false,
	}

	queries := []string{
		"simple",
		"multi word query",
		"query with 'quotes'",
		"query with \"double quotes\"",
		"query with special chars: @#$%",
		"日本語クエリ",
		"",
		"   whitespace   ",
	}

	for _, query := range queries {
		t.Run(query, func(t *testing.T) {
			result := ExecuteSearch(query, cfg, searchCfg, nil)

			if result.Success {
				t.Error("expected failure with search disabled")
			}

			if result.Command.Argument != query {
				t.Errorf("query not preserved: expected %q, got %q", query, result.Command.Argument)
			}
		})
	}
}

func TestExecuteSearch_ConfigValues(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := newTestConfig(tmpDir)

	configs := []struct {
		name      string
		searchCfg *search.SearchConfig
	}{
		{
			name: "minimal config",
			searchCfg: &search.SearchConfig{
				Enabled:      true,
				VectorDBPath: filepath.Join(tmpDir, "min.db"),
			},
		},
		{
			name: "full config",
			searchCfg: &search.SearchConfig{
				Enabled:            true,
				VectorDBPath:       filepath.Join(tmpDir, "full.db"),
				MaxResults:         25,
				MinSimilarityScore: 0.75,
				MaxPreviewLength:   200,
				OllamaURL:         "/nonexistent",
			},
		},
		{
			name: "zero max results",
			searchCfg: &search.SearchConfig{
				Enabled:      true,
				VectorDBPath: filepath.Join(tmpDir, "zero.db"),
				MaxResults:   0,
			},
		},
	}

	for _, tc := range configs {
		t.Run(tc.name, func(t *testing.T) {
			result := ExecuteSearch("test", cfg, tc.searchCfg, nil)
			t.Logf("Result: success=%v, error=%v", result.Success, result.Error)
		})
	}
}

// Benchmark tests
func BenchmarkExecuteSearch_Disabled(b *testing.B) {
	cfg := newTestConfig(b.TempDir())

	searchCfg := &search.SearchConfig{
		Enabled: false,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ExecuteSearch("test query", cfg, searchCfg, nil)
	}
}

func BenchmarkExecuteSearch_NilConfig(b *testing.B) {
	cfg := newTestConfig(b.TempDir())

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ExecuteSearch("test query", cfg, nil, nil)
	}
}

func BenchmarkFormatSearchOutput_Empty(b *testing.B) {
	results := []search.SearchResult{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		formatSearchOutput("query", results, 10, 0)
	}
}

func BenchmarkFormatSearchOutput_ManyResults(b *testing.B) {
	results := make([]search.SearchResult, 100)
	for i := range results {
		results[i] = search.SearchResult{
			FilePath:  "file.go",
			Score:     float32(i) / 100,
			Preview:   "preview content here",
			LineCount: 100,
			FileSize:  1024,
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		formatSearchOutput("query", results, 10, 0)
	}
}

func BenchmarkFormatFileSizeForSearch(b *testing.B) {
	sizes := []int64{100, 1024, 1024 * 1024, 1024 * 1024 * 1024}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		formatFileSizeForSearch(sizes[i%len(sizes)])
	}
}

func TestFormatSearchOutput_AllFields(t *testing.T) {
	results := []search.SearchResult{
		{
			FilePath:  "internal/main.go",
			Score:     0.9234,
			LineCount: 150,
			FileSize:  4096,
			Preview:   "func main() { ... }",
		},
	}

	output := formatSearchOutput("main function", results, 10, 250*time.Millisecond)

	checks := []string{
		"=== SEARCH: main function ===",
		"SEARCH RESULTS",
		"0.25s",
		"internal/main.go",
		"92.34",
		"Lines: 150",
		"4.0 KB",
		"Preview:",
		"func main()",
		"=== END SEARCH ===",
	}

	for _, check := range checks {
		if !strings.Contains(output, check) {
			t.Errorf("output should contain %q\nGot:\n%s", check, output)
		}
	}
}

func TestFormatSearchOutput_MultipleResults(t *testing.T) {
	results := []search.SearchResult{
		{FilePath: "a.go", Score: 0.95, LineCount: 100, FileSize: 1024, Preview: "preview a"},
		{FilePath: "b.go", Score: 0.85, LineCount: 200, FileSize: 2048, Preview: "preview b"},
		{FilePath: "c.go", Score: 0.75, LineCount: 300, FileSize: 3072, Preview: "preview c"},
	}

	output := formatSearchOutput("test", results, 10, 0)

	// Check ordering (should be 1, 2, 3)
	if !strings.Contains(output, "1. a.go") {
		t.Error("first result should be a.go")
	}
	if !strings.Contains(output, "2. b.go") {
		t.Error("second result should be b.go")
	}
	if !strings.Contains(output, "3. c.go") {
		t.Error("third result should be c.go")
	}
}

func TestFormatFileSizeForSearch_LargeFiles(t *testing.T) {
	tests := []struct {
		size     int64
		contains string
	}{
		{1024 * 1024 * 1024 * 10, "GB"},          // 10 GB
		{1024 * 1024 * 1024 * 1024, "TB"},        // 1 TB
		{1024 * 1024 * 1024 * 1024 * 1024, "PB"}, // 1 PB
	}

	for _, tt := range tests {
		t.Run(tt.contains, func(t *testing.T) {
			result := formatFileSizeForSearch(tt.size)
			if !strings.Contains(result, tt.contains) {
				t.Errorf("formatFileSizeForSearch(%d) = %q, should contain %q", tt.size, result, tt.contains)
			}
		})
	}
}

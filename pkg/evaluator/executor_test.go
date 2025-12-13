package evaluator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/computerscienceiscool/llm-runtime/pkg/config"
	"github.com/computerscienceiscool/llm-runtime/pkg/search"
	"github.com/computerscienceiscool/llm-runtime/pkg/scanner"
)

func TestNewExecutor(t *testing.T) {
	cfg := &config.Config{
		RepositoryRoot: "/tmp/test",
		MaxFileSize:    1024,
	}

	searchCfg := &search.SearchConfig{
		Enabled:    false,
		MaxResults: 10,
	}

	audit := func(cmd, arg string, success bool, errMsg string) {}

	executor := NewExecutor(cfg, searchCfg, audit)

	if executor == nil {
		t.Fatal("NewExecutor returned nil")
	}

	if executor.config != cfg {
		t.Error("config not set correctly")
	}

	if executor.searchCfg != searchCfg {
		t.Error("searchCfg not set correctly")
	}

	if executor.auditLog == nil {
		t.Error("auditLog not set correctly")
	}

	if executor.commandsRun != 0 {
		t.Errorf("expected commandsRun to be 0, got %d", executor.commandsRun)
	}
}

func TestNewExecutor_NilSearchConfig(t *testing.T) {
	cfg := &config.Config{
		RepositoryRoot: "/tmp/test",
	}

	executor := NewExecutor(cfg, nil, nil)

	if executor == nil {
		t.Fatal("NewExecutor returned nil")
	}

	if executor.searchCfg != nil {
		t.Error("searchCfg should be nil")
	}
}

func TestNewExecutor_NilAuditLog(t *testing.T) {
	cfg := &config.Config{
		RepositoryRoot: "/tmp/test",
	}

	executor := NewExecutor(cfg, nil, nil)

	if executor == nil {
		t.Fatal("NewExecutor returned nil")
	}

	if executor.auditLog != nil {
		t.Error("auditLog should be nil")
	}
}

func TestExecutor_Execute_OpenCommand(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := newTestConfig(tmpDir)

	// Create test file
	testContent := "test file content"
	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	executor := NewExecutor(cfg, nil, nil)

	cmd := scanner.Command{
		Type:     "open",
		Argument: "test.txt",
	}

	result := executor.Execute(cmd)

	if !result.Success {
		t.Errorf("expected success, got error: %v", result.Error)
	}

	if result.Result != testContent {
		t.Errorf("expected content %q, got %q", testContent, result.Result)
	}

	if result.Command.Type != "open" {
		t.Errorf("expected command type 'open', got %q", result.Command.Type)
	}
}

func TestExecutor_Execute_WriteCommand(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := newTestConfig(tmpDir)
	cfg.BackupBeforeWrite = false

	executor := NewExecutor(cfg, nil, nil)

	content := "new file content"
	cmd := scanner.Command{
		Type:     "write",
		Argument: "new_file.txt",
		Content:  content,
	}

	result := executor.Execute(cmd)

	if !result.Success {
		t.Errorf("expected success, got error: %v", result.Error)
	}

	if result.Action != "CREATED" {
		t.Errorf("expected action 'CREATED', got %q", result.Action)
	}

	// Verify file was created
	createdContent, err := os.ReadFile(filepath.Join(tmpDir, "new_file.txt"))
	if err != nil {
		t.Fatalf("failed to read created file: %v", err)
	}

	if string(createdContent) != content {
		t.Errorf("content mismatch: expected %q, got %q", content, string(createdContent))
	}
}

func TestExecutor_Execute_ExecCommand_Disabled(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := newTestConfig(tmpDir)

	executor := NewExecutor(cfg, nil, nil)

	cmd := scanner.Command{
		Type:     "exec",
		Argument: "ls -la",
	}

	result := executor.Execute(cmd)

	if result.Success {
		t.Error("expected failure when exec is disabled")
	}

	if result.Error == nil {
		t.Error("expected error to be set")
	}

	if !strings.Contains(result.Error.Error(), "EXEC_VALIDATION") {
		t.Errorf("expected EXEC_VALIDATION error, got: %v", result.Error)
	}
}

func TestExecutor_Execute_SearchCommand_Disabled(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := newTestConfig(tmpDir)

	// Search disabled
	searchCfg := &search.SearchConfig{
		Enabled: false,
	}

	executor := NewExecutor(cfg, searchCfg, nil)

	cmd := scanner.Command{
		Type:     "search",
		Argument: "test query",
	}

	result := executor.Execute(cmd)

	if result.Success {
		t.Error("expected failure when search is disabled")
	}

	if result.Error == nil {
		t.Error("expected error to be set")
	}

	if !strings.Contains(result.Error.Error(), "SEARCH_DISABLED") {
		t.Errorf("expected SEARCH_DISABLED error, got: %v", result.Error)
	}
}

func TestExecutor_Execute_SearchCommand_NilConfig(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := newTestConfig(tmpDir)

	executor := NewExecutor(cfg, nil, nil)

	cmd := scanner.Command{
		Type:     "search",
		Argument: "test query",
	}

	result := executor.Execute(cmd)

	if result.Success {
		t.Error("expected failure when search config is nil")
	}

	if !strings.Contains(result.Error.Error(), "SEARCH_DISABLED") {
		t.Errorf("expected SEARCH_DISABLED error, got: %v", result.Error)
	}
}

func TestExecutor_Execute_UnknownCommand(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := newTestConfig(tmpDir)

	executor := NewExecutor(cfg, nil, nil)

	cmd := scanner.Command{
		Type:     "unknown",
		Argument: "some argument",
	}

	result := executor.Execute(cmd)

	if result.Success {
		t.Error("expected failure for unknown command")
	}

	if result.Error == nil {
		t.Error("expected error to be set")
	}

	if !strings.Contains(result.Error.Error(), "UNKNOWN_COMMAND") {
		t.Errorf("expected UNKNOWN_COMMAND error, got: %v", result.Error)
	}

	if !strings.Contains(result.Error.Error(), "unknown") {
		t.Errorf("expected command type in error message, got: %v", result.Error)
	}
}

func TestExecutor_Execute_EmptyCommandType(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := newTestConfig(tmpDir)

	executor := NewExecutor(cfg, nil, nil)

	cmd := scanner.Command{
		Type:     "",
		Argument: "some argument",
	}

	result := executor.Execute(cmd)

	if result.Success {
		t.Error("expected failure for empty command type")
	}

	if !strings.Contains(result.Error.Error(), "UNKNOWN_COMMAND") {
		t.Errorf("expected UNKNOWN_COMMAND error, got: %v", result.Error)
	}
}

func TestExecutor_GetCommandsRun(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := newTestConfig(tmpDir)

	executor := NewExecutor(cfg, nil, nil)

	// Initial count should be 0
	if executor.GetCommandsRun() != 0 {
		t.Errorf("expected 0 commands run initially, got %d", executor.GetCommandsRun())
	}

	// Create a test file
	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("content"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Execute successful command
	cmd := scanner.Command{
		Type:     "open",
		Argument: "test.txt",
	}
	executor.Execute(cmd)

	if executor.GetCommandsRun() != 1 {
		t.Errorf("expected 1 command run, got %d", executor.GetCommandsRun())
	}

	// Execute another successful command
	executor.Execute(cmd)

	if executor.GetCommandsRun() != 2 {
		t.Errorf("expected 2 commands run, got %d", executor.GetCommandsRun())
	}
}

func TestExecutor_GetCommandsRun_FailedCommandsNotCounted(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := newTestConfig(tmpDir)

	executor := NewExecutor(cfg, nil, nil)

	// Execute failing command
	cmd := scanner.Command{
		Type:     "open",
		Argument: "nonexistent.txt",
	}
	executor.Execute(cmd)

	if executor.GetCommandsRun() != 0 {
		t.Errorf("failed commands should not be counted, got %d", executor.GetCommandsRun())
	}

	// Execute unknown command
	unknownCmd := scanner.Command{
		Type:     "unknown",
		Argument: "arg",
	}
	executor.Execute(unknownCmd)

	if executor.GetCommandsRun() != 0 {
		t.Errorf("unknown commands should not be counted, got %d", executor.GetCommandsRun())
	}
}

func TestExecutor_GetConfig(t *testing.T) {
	cfg := &config.Config{
		RepositoryRoot: "/custom/path",
		MaxFileSize:    12345,
		Verbose:        true,
	}

	executor := NewExecutor(cfg, nil, nil)

	returnedCfg := executor.GetConfig()

	if returnedCfg != cfg {
		t.Error("GetConfig should return the same config instance")
	}

	if returnedCfg.RepositoryRoot != "/custom/path" {
		t.Errorf("expected RepositoryRoot '/custom/path', got %q", returnedCfg.RepositoryRoot)
	}

	if returnedCfg.MaxFileSize != 12345 {
		t.Errorf("expected MaxFileSize 12345, got %d", returnedCfg.MaxFileSize)
	}

	if !returnedCfg.Verbose {
		t.Error("expected Verbose to be true")
	}
}

func TestExecutor_GetSearchConfig(t *testing.T) {
	searchCfg := &search.SearchConfig{
		Enabled:            true,
		VectorDBPath:       "/custom/db",
		MaxResults:         25,
		MinSimilarityScore: 0.75,
	}

	executor := NewExecutor(&config.Config{}, searchCfg, nil)

	returnedCfg := executor.GetSearchConfig()

	if returnedCfg != searchCfg {
		t.Error("GetSearchConfig should return the same config instance")
	}

	if !returnedCfg.Enabled {
		t.Error("expected Enabled to be true")
	}

	if returnedCfg.VectorDBPath != "/custom/db" {
		t.Errorf("expected VectorDBPath '/custom/db', got %q", returnedCfg.VectorDBPath)
	}

	if returnedCfg.MaxResults != 25 {
		t.Errorf("expected MaxResults 25, got %d", returnedCfg.MaxResults)
	}
}

func TestExecutor_GetSearchConfig_Nil(t *testing.T) {
	executor := NewExecutor(&config.Config{}, nil, nil)

	returnedCfg := executor.GetSearchConfig()

	if returnedCfg != nil {
		t.Error("expected nil search config")
	}
}

func TestExecutor_WithAuditLog(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := newTestConfig(tmpDir)

	audit := &testAuditLog{}
	executor := NewExecutor(cfg, nil, audit.log)

	// Create test file
	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("content"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Execute open command
	openCmd := scanner.Command{
		Type:     "open",
		Argument: "test.txt",
	}
	executor.Execute(openCmd)

	// Check audit log was called
	entries := audit.getEntries()
	if len(entries) != 1 {
		t.Fatalf("expected 1 audit entry, got %d", len(entries))
	}

	if entries[0].cmdType != "open" {
		t.Errorf("expected cmdType 'open', got %q", entries[0].cmdType)
	}
}

func TestExecutor_MultipleCommands(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := newTestConfig(tmpDir)
	cfg.BackupBeforeWrite = false

	executor := NewExecutor(cfg, nil, nil)

	// Write a file
	writeCmd := scanner.Command{
		Type:     "write",
		Argument: "test.txt",
		Content:  "hello world",
	}
	writeResult := executor.Execute(writeCmd)
	if !writeResult.Success {
		t.Fatalf("write failed: %v", writeResult.Error)
	}

	// Read the file back
	openCmd := scanner.Command{
		Type:     "open",
		Argument: "test.txt",
	}
	openResult := executor.Execute(openCmd)
	if !openResult.Success {
		t.Fatalf("open failed: %v", openResult.Error)
	}

	if openResult.Result != "hello world" {
		t.Errorf("expected 'hello world', got %q", openResult.Result)
	}

	// Both commands should be counted
	if executor.GetCommandsRun() != 2 {
		t.Errorf("expected 2 commands run, got %d", executor.GetCommandsRun())
	}
}

func TestExecutor_Execute_PreservesCommandInResult(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := newTestConfig(tmpDir)

	executor := NewExecutor(cfg, nil, nil)

	cmd := scanner.Command{
		Type:     "unknown",
		Argument: "test_arg",
		Content:  "test_content",
		StartPos: 10,
		EndPos:   20,
		Original: "<unknown test_arg>",
	}

	result := executor.Execute(cmd)

	// The result should preserve the original command
	if result.Command.Type != cmd.Type {
		t.Errorf("command type not preserved: expected %q, got %q", cmd.Type, result.Command.Type)
	}

	if result.Command.Argument != cmd.Argument {
		t.Errorf("command argument not preserved: expected %q, got %q", cmd.Argument, result.Command.Argument)
	}
}

func TestExecutor_Execute_CaseSensitiveCommandType(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := newTestConfig(tmpDir)

	executor := NewExecutor(cfg, nil, nil)

	// Command types should be case-sensitive
	tests := []string{"Open", "OPEN", "Open", "wRiTe", "EXEC", "SEARCH"}

	for _, cmdType := range tests {
		t.Run(cmdType, func(t *testing.T) {
			cmd := scanner.Command{
				Type:     cmdType,
				Argument: "test",
			}

			result := executor.Execute(cmd)

			if result.Success {
				t.Errorf("expected failure for case-mismatched command type %q", cmdType)
			}

			if !strings.Contains(result.Error.Error(), "UNKNOWN_COMMAND") {
				t.Errorf("expected UNKNOWN_COMMAND for %q, got: %v", cmdType, result.Error)
			}
		})
	}
}

func TestExecutor_Execute_ExecWithEmptyWhitelist(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := newTestConfig(tmpDir)
	cfg.ExecWhitelist = []string{} // Empty whitelist

	executor := NewExecutor(cfg, nil, nil)

	cmd := scanner.Command{
		Type:     "exec",
		Argument: "ls",
	}

	result := executor.Execute(cmd)

	if result.Success {
		t.Error("expected failure with empty whitelist")
	}

	// Should fail at validation, not Docker check
	if !strings.Contains(result.Error.Error(), "EXEC_VALIDATION") {
		t.Errorf("expected EXEC_VALIDATION error, got: %v", result.Error)
	}
}

func TestExecutor_Execute_ExecWithEmptyCommand(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := newTestConfig(tmpDir)
	cfg.ExecWhitelist = []string{"ls"}

	executor := NewExecutor(cfg, nil, nil)

	cmd := scanner.Command{
		Type:     "exec",
		Argument: "", // Empty command
	}

	result := executor.Execute(cmd)

	if result.Success {
		t.Error("expected failure with empty exec command")
	}

	if !strings.Contains(result.Error.Error(), "EXEC_VALIDATION") {
		t.Errorf("expected EXEC_VALIDATION error, got: %v", result.Error)
	}
}

// Integration-style tests

func TestExecutor_FullWorkflow(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := newTestConfig(tmpDir)
	cfg.BackupBeforeWrite = true

	audit := &testAuditLog{}
	executor := NewExecutor(cfg, nil, audit.log)

	// Step 1: Write initial file
	writeCmd1 := scanner.Command{
		Type:     "write",
		Argument: "workflow.txt",
		Content:  "version 1",
	}
	result1 := executor.Execute(writeCmd1)
	if !result1.Success {
		t.Fatalf("initial write failed: %v", result1.Error)
	}

	// Step 2: Read the file
	readCmd := scanner.Command{
		Type:     "open",
		Argument: "workflow.txt",
	}
	result2 := executor.Execute(readCmd)
	if !result2.Success {
		t.Fatalf("read failed: %v", result2.Error)
	}
	if result2.Result != "version 1" {
		t.Errorf("unexpected content: %q", result2.Result)
	}

	// Step 3: Update the file
	writeCmd2 := scanner.Command{
		Type:     "write",
		Argument: "workflow.txt",
		Content:  "version 2",
	}
	result3 := executor.Execute(writeCmd2)
	if !result3.Success {
		t.Fatalf("update write failed: %v", result3.Error)
	}
	if result3.BackupFile == "" {
		t.Error("expected backup to be created")
	}

	// Step 4: Verify update
	result4 := executor.Execute(readCmd)
	if !result4.Success {
		t.Fatalf("verification read failed: %v", result4.Error)
	}
	if result4.Result != "version 2" {
		t.Errorf("expected 'version 2', got %q", result4.Result)
	}

	// Check command count
	if executor.GetCommandsRun() != 4 {
		t.Errorf("expected 4 commands run, got %d", executor.GetCommandsRun())
	}

	// Check audit log
	entries := audit.getEntries()
	if len(entries) != 4 {
		t.Errorf("expected 4 audit entries, got %d", len(entries))
	}
}

// Benchmark tests

func BenchmarkNewExecutor(b *testing.B) {
	cfg := &config.Config{
		RepositoryRoot: "/tmp",
	}
	searchCfg := &search.SearchConfig{}
	auditFn := func(cmd, arg string, success bool, errMsg string) {}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		NewExecutor(cfg, searchCfg, auditFn)
	}
}

func BenchmarkExecutor_Execute_Open(b *testing.B) {
	tmpDir := b.TempDir()
	cfg := newTestConfig(tmpDir)

	testFile := filepath.Join(tmpDir, "test.txt")
	os.WriteFile(testFile, []byte("benchmark content"), 0644)

	executor := NewExecutor(cfg, nil, nil)

	cmd := scanner.Command{
		Type:     "open",
		Argument: "test.txt",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		executor.Execute(cmd)
	}
}

func BenchmarkExecutor_Execute_Write(b *testing.B) {
	tmpDir := b.TempDir()
	cfg := newTestConfig(tmpDir)
	cfg.BackupBeforeWrite = false

	executor := NewExecutor(cfg, nil, nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cmd := scanner.Command{
			Type:     "write",
			Argument: "bench" + string(rune('0'+i%10)) + ".txt",
			Content:  "benchmark content",
		}
		executor.Execute(cmd)
	}
}

func BenchmarkExecutor_GetCommandsRun(b *testing.B) {
	executor := NewExecutor(&config.Config{}, nil, nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		executor.GetCommandsRun()
	}
}

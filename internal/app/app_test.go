package app

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/computerscienceiscool/llm-runtime/internal/cli"
	"github.com/computerscienceiscool/llm-runtime/internal/config"
)

// captureStderr captures stderr during function execution
func captureStderr(t *testing.T, f func()) string {
	t.Helper()

	oldStderr := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("Failed to create stderr pipe: %v", err)
	}
	os.Stderr = w

	f()

	w.Close()
	os.Stderr = oldStderr

	var buf bytes.Buffer
	io.Copy(&buf, r)
	r.Close()

	return buf.String()
}

// captureStdout captures stdout during function execution
func captureStdout(t *testing.T, f func()) string {
	t.Helper()

	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("Failed to create stdout pipe: %v", err)
	}
	os.Stdout = w

	f()

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	io.Copy(&buf, r)
	r.Close()

	return buf.String()
}

func TestApp_GetConfig(t *testing.T) {
	tempDir := t.TempDir()

	cfg := &config.Config{
		RepositoryRoot:    tempDir,
		MaxFileSize:       1048576,
		MaxWriteSize:      102400,
		AllowedExtensions: []string{".txt"},
		ExcludedPaths:     []string{".git"},
	}

	app, err := Bootstrap(cfg)
	if err != nil {
		t.Fatalf("Bootstrap() error = %v", err)
	}

	gotCfg := app.GetConfig()
	if gotCfg == nil {
		t.Fatal("GetConfig() returned nil")
	}

	if gotCfg.MaxFileSize != cfg.MaxFileSize {
		t.Errorf("GetConfig().MaxFileSize = %d, want %d", gotCfg.MaxFileSize, cfg.MaxFileSize)
	}
}

func TestApp_GetSession(t *testing.T) {
	tempDir := t.TempDir()

	cfg := &config.Config{
		RepositoryRoot:    tempDir,
		MaxFileSize:       1048576,
		MaxWriteSize:      102400,
		AllowedExtensions: []string{".txt"},
		ExcludedPaths:     []string{".git"},
	}

	app, err := Bootstrap(cfg)
	if err != nil {
		t.Fatalf("Bootstrap() error = %v", err)
	}

	session := app.GetSession()
	if session == nil {
		t.Fatal("GetSession() returned nil")
	}

	if session.ID == "" {
		t.Error("Session ID should not be empty")
	}

	if session.StartTime.IsZero() {
		t.Error("Session StartTime should not be zero")
	}
}

func TestApp_GetExecutor(t *testing.T) {
	tempDir := t.TempDir()

	cfg := &config.Config{
		RepositoryRoot:    tempDir,
		MaxFileSize:       1048576,
		MaxWriteSize:      102400,
		AllowedExtensions: []string{".txt"},
		ExcludedPaths:     []string{".git"},
	}

	app, err := Bootstrap(cfg)
	if err != nil {
		t.Fatalf("Bootstrap() error = %v", err)
	}

	executor := app.GetExecutor()
	if executor == nil {
		t.Fatal("GetExecutor() returned nil")
	}

	if executor.GetCommandsRun() != 0 {
		t.Errorf("New executor should have 0 commands run, got %d", executor.GetCommandsRun())
	}
}

func TestApp_GetSearchConfig(t *testing.T) {
	tempDir := t.TempDir()

	cfg := &config.Config{
		RepositoryRoot:    tempDir,
		MaxFileSize:       1048576,
		MaxWriteSize:      102400,
		AllowedExtensions: []string{".txt"},
		ExcludedPaths:     []string{".git"},
	}

	app, err := Bootstrap(cfg)
	if err != nil {
		t.Fatalf("Bootstrap() error = %v", err)
	}

	searchCfg := app.GetSearchConfig()
	// Search config may be nil if not configured, or may have defaults
	// Just verify it doesn't panic
	_ = searchCfg
}

func TestApp_Run_PipeMode_Stdin(t *testing.T) {
	tempDir := t.TempDir()

	// Create a test file to open
	testFile := filepath.Join(tempDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("Hello, World!"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	cfg := &config.Config{
		RepositoryRoot:    tempDir,
		MaxFileSize:       1048576,
		MaxWriteSize:      102400,
		AllowedExtensions: []string{".txt"},
		ExcludedPaths:     []string{".git"},
		Interactive:       false,
		// No InputFile - should read from stdin
	}

	app, err := Bootstrap(cfg)
	if err != nil {
		t.Fatalf("Bootstrap() error = %v", err)
	}

	// Mock stdin
	oldStdin := os.Stdin
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("Failed to create stdin pipe: %v", err)
	}
	os.Stdin = r

	go func() {
		defer w.Close()
		w.WriteString("<open test.txt>")
	}()

	// Capture stdout
	stdout := captureStdout(t, func() {
		err = app.Run()
	})

	os.Stdin = oldStdin
	r.Close()

	if err != nil {
		t.Errorf("Run() error = %v", err)
	}

	if !strings.Contains(stdout, "Hello, World!") {
		t.Errorf("Expected file content in output\nGot: %s", stdout)
	}
}

func TestApp_Run_PipeMode_InputFile(t *testing.T) {
	tempDir := t.TempDir()

	// Create a test file to open
	testFile := filepath.Join(tempDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("File content here"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create input file with command
	inputFile := filepath.Join(tempDir, "input.txt")
	if err := os.WriteFile(inputFile, []byte("<open test.txt>"), 0644); err != nil {
		t.Fatalf("Failed to create input file: %v", err)
	}

	cfg := &config.Config{
		RepositoryRoot:    tempDir,
		MaxFileSize:       1048576,
		MaxWriteSize:      102400,
		AllowedExtensions: []string{".txt"},
		ExcludedPaths:     []string{".git"},
		Interactive:       false,
		InputFile:         inputFile,
	}

	app, err := Bootstrap(cfg)
	if err != nil {
		t.Fatalf("Bootstrap() error = %v", err)
	}

	stdout := captureStdout(t, func() {
		err = app.Run()
	})

	if err != nil {
		t.Errorf("Run() error = %v", err)
	}

	if !strings.Contains(stdout, "File content here") {
		t.Errorf("Expected file content in output\nGot: %s", stdout)
	}
}

func TestApp_Run_PipeMode_OutputFile(t *testing.T) {
	tempDir := t.TempDir()

	// Create a test file to open
	testFile := filepath.Join(tempDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("Output test content"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create input file with command
	inputFile := filepath.Join(tempDir, "input.txt")
	if err := os.WriteFile(inputFile, []byte("<open test.txt>"), 0644); err != nil {
		t.Fatalf("Failed to create input file: %v", err)
	}

	outputFile := filepath.Join(tempDir, "output.txt")

	cfg := &config.Config{
		RepositoryRoot:    tempDir,
		MaxFileSize:       1048576,
		MaxWriteSize:      102400,
		AllowedExtensions: []string{".txt"},
		ExcludedPaths:     []string{".git"},
		Interactive:       false,
		InputFile:         inputFile,
		OutputFile:        outputFile,
	}

	app, err := Bootstrap(cfg)
	if err != nil {
		t.Fatalf("Bootstrap() error = %v", err)
	}

	err = app.Run()
	if err != nil {
		t.Errorf("Run() error = %v", err)
	}

	// Check output file was created
	content, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}

	if !strings.Contains(string(content), "Output test content") {
		t.Errorf("Expected file content in output file\nGot: %s", string(content))
	}
}

func TestApp_Run_PipeMode_NonExistentInputFile(t *testing.T) {
	tempDir := t.TempDir()

	cfg := &config.Config{
		RepositoryRoot:    tempDir,
		MaxFileSize:       1048576,
		MaxWriteSize:      102400,
		AllowedExtensions: []string{".txt"},
		ExcludedPaths:     []string{".git"},
		Interactive:       false,
		InputFile:         "/nonexistent/input.txt",
	}

	app, err := Bootstrap(cfg)
	if err != nil {
		t.Fatalf("Bootstrap() error = %v", err)
	}

	err = app.Run()
	if err == nil {
		t.Error("Run() should fail for non-existent input file")
	}

	if !strings.Contains(err.Error(), "cannot read input file") {
		t.Errorf("Error should mention input file, got: %v", err)
	}
}

func TestApp_Run_PipeMode_CannotWriteOutputFile(t *testing.T) {
	tempDir := t.TempDir()

	// Create input file
	inputFile := filepath.Join(tempDir, "input.txt")
	if err := os.WriteFile(inputFile, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create input file: %v", err)
	}

	cfg := &config.Config{
		RepositoryRoot:    tempDir,
		MaxFileSize:       1048576,
		MaxWriteSize:      102400,
		AllowedExtensions: []string{".txt"},
		ExcludedPaths:     []string{".git"},
		Interactive:       false,
		InputFile:         inputFile,
		OutputFile:        "/nonexistent/directory/output.txt",
	}

	app, err := Bootstrap(cfg)
	if err != nil {
		t.Fatalf("Bootstrap() error = %v", err)
	}

	err = app.Run()
	if err == nil {
		t.Error("Run() should fail for non-writable output path")
	}

	if !strings.Contains(err.Error(), "cannot write output file") {
		t.Errorf("Error should mention output file, got: %v", err)
	}
}

func TestApp_Run_VerboseMode(t *testing.T) {
	tempDir := t.TempDir()

	cfg := &config.Config{
		RepositoryRoot:    tempDir,
		MaxFileSize:       1048576,
		MaxWriteSize:      102400,
		AllowedExtensions: []string{".txt", ".go"},
		ExcludedPaths:     []string{".git", "vendor"},
		Interactive:       false,
		Verbose:           true,
		ExecEnabled:       true,
		ExecWhitelist:     []string{"go test"},
	}

	app, err := Bootstrap(cfg)
	if err != nil {
		t.Fatalf("Bootstrap() error = %v", err)
	}

	// Mock stdin with empty input
	oldStdin := os.Stdin
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("Failed to create stdin pipe: %v", err)
	}
	os.Stdin = r
	w.Close()

	stderr := captureStderr(t, func() {
		_ = app.Run()
	})

	os.Stdin = oldStdin
	r.Close()

	// Verbose mode should print config info
	expectedParts := []string{
		"Repository root:",
		"Max file size:",
		"Max write file size:",
		"Allowed extensions:",
		"Excluded paths:",
		"Backup enabled:",
		"Exec enabled:",
	}

	for _, part := range expectedParts {
		if !strings.Contains(stderr, part) {
			t.Errorf("Verbose output missing: %q\nGot: %s", part, stderr)
		}
	}
}

func TestApp_Run_VerboseMode_ExecDetails(t *testing.T) {
	tempDir := t.TempDir()

	cfg := &config.Config{
		RepositoryRoot:     tempDir,
		MaxFileSize:        1048576,
		MaxWriteSize:       102400,
		AllowedExtensions:  []string{".txt"},
		ExcludedPaths:      []string{".git"},
		Interactive:        false,
		Verbose:            true,
		ExecEnabled:        true,
		ExecWhitelist:      []string{"go test", "make"},
		ExecContainerImage: "alpine:latest",
	}

	app, err := Bootstrap(cfg)
	if err != nil {
		t.Fatalf("Bootstrap() error = %v", err)
	}

	// Mock stdin with empty input
	oldStdin := os.Stdin
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("Failed to create stdin pipe: %v", err)
	}
	os.Stdin = r
	w.Close()

	stderr := captureStderr(t, func() {
		_ = app.Run()
	})

	os.Stdin = oldStdin
	r.Close()

	// Should show exec details when exec is enabled
	if !strings.Contains(stderr, "Exec whitelist:") {
		t.Errorf("Verbose output should show exec whitelist\nGot: %s", stderr)
	}
	if !strings.Contains(stderr, "Exec image:") {
		t.Errorf("Verbose output should show exec image\nGot: %s", stderr)
	}
}

func TestApp_RunSearchCommand_SearchDisabled(t *testing.T) {
	tempDir := t.TempDir()

	cfg := &config.Config{
		RepositoryRoot:    tempDir,
		MaxFileSize:       1048576,
		MaxWriteSize:      102400,
		AllowedExtensions: []string{".txt"},
		ExcludedPaths:     []string{".git"},
	}

	app, err := Bootstrap(cfg)
	if err != nil {
		t.Fatalf("Bootstrap() error = %v", err)
	}

	// If search is not enabled in config, RunSearchCommand should fail
	flags := &cli.CLIFlags{
		Reindex: true,
	}

	err = app.RunSearchCommand(flags)
	// Should fail because search is not enabled
	if err == nil {
		// Search might be enabled by default config, which is okay
		return
	}

	if !strings.Contains(err.Error(), "search") {
		t.Errorf("Error should mention search, got: %v", err)
	}
}

func TestApp_RunSearchCommand_CheckOllamaSetup(t *testing.T) {
	tempDir := t.TempDir()

	cfg := &config.Config{
		RepositoryRoot:    tempDir,
		MaxFileSize:       1048576,
		MaxWriteSize:      102400,
		AllowedExtensions: []string{".txt"},
		ExcludedPaths:     []string{".git"},
	}

	app, err := Bootstrap(cfg)
	if err != nil {
		t.Fatalf("Bootstrap() error = %v", err)
	}

	flags := &cli.CLIFlags{
		CheckOllamaSetup: true,
	}

	stderr := captureStderr(t, func() {
		err = app.RunSearchCommand(flags)
	})

	// CheckOllamaSetup might succeed or fail depending on environment
	// Just verify it runs and produces output
	if !strings.Contains(stderr, "Ollama") {
		t.Errorf("CheckOllamaSetup should mention Ollama\nGot: %s", stderr)
	}
}

func TestApp_Run_NoCommands(t *testing.T) {
	tempDir := t.TempDir()
	// Create input file with no commands
	inputFile := filepath.Join(tempDir, "input.txt")
	if err := os.WriteFile(inputFile, []byte("Just plain text, no commands here."), 0644); err != nil {
		t.Fatalf("Failed to create input file: %v", err)
	}
	outputFile := filepath.Join(tempDir, "output.txt")
	cfg := &config.Config{
		RepositoryRoot:    tempDir,
		MaxFileSize:       1048576,
		MaxWriteSize:      102400,
		AllowedExtensions: []string{".txt"},
		ExcludedPaths:     []string{".git"},
		Interactive:       false,
		InputFile:         inputFile,
		OutputFile:        outputFile,
	}
	app, err := Bootstrap(cfg)
	if err != nil {
		t.Fatalf("Bootstrap() error = %v", err)
	}
	err = app.Run()
	if err != nil {
		t.Errorf("Run() error = %v", err)
	}
	// Read output file
	output, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}
	// Should produce no output when there are no commands
	if len(output) != 0 {
		t.Errorf("Expected no output without commands, got: %s", string(output))
	}
}
func TestApp_Run_WriteCommand(t *testing.T) {
	tempDir := t.TempDir()

	// Create input file with write command
	inputFile := filepath.Join(tempDir, "input.txt")
	if err := os.WriteFile(inputFile, []byte("<write output.txt>Created by test</write>"), 0644); err != nil {
		t.Fatalf("Failed to create input file: %v", err)
	}

	cfg := &config.Config{
		RepositoryRoot:    tempDir,
		MaxFileSize:       1048576,
		MaxWriteSize:      102400,
		AllowedExtensions: []string{".txt"},
		ExcludedPaths:     []string{".git"},
		Interactive:       false,
		InputFile:         inputFile,
	}

	app, err := Bootstrap(cfg)
	if err != nil {
		t.Fatalf("Bootstrap() error = %v", err)
	}

	stdout := captureStdout(t, func() {
		err = app.Run()
	})

	if err != nil {
		t.Errorf("Run() error = %v", err)
	}

	if !strings.Contains(stdout, "WRITE SUCCESSFUL") {
		t.Errorf("Expected write success in output\nGot: %s", stdout)
	}

	// Verify file was created
	content, err := os.ReadFile(filepath.Join(tempDir, "output.txt"))
	if err != nil {
		t.Fatalf("Output file not created: %v", err)
	}

	if string(content) != "Created by test" {
		t.Errorf("File content = %q, want %q", string(content), "Created by test")
	}
}

func TestApp_Run_EmptyInput(t *testing.T) {
	tempDir := t.TempDir()

	// Create empty input file
	inputFile := filepath.Join(tempDir, "input.txt")
	if err := os.WriteFile(inputFile, []byte(""), 0644); err != nil {
		t.Fatalf("Failed to create input file: %v", err)
	}

	cfg := &config.Config{
		RepositoryRoot:    tempDir,
		MaxFileSize:       1048576,
		MaxWriteSize:      102400,
		AllowedExtensions: []string{".txt"},
		ExcludedPaths:     []string{".git"},
		Interactive:       false,
		InputFile:         inputFile,
	}

	app, err := Bootstrap(cfg)
	if err != nil {
		t.Fatalf("Bootstrap() error = %v", err)
	}

	err = app.Run()
	if err != nil {
		t.Errorf("Run() with empty input should not error, got: %v", err)
	}
}

func TestApp_MultipleRuns(t *testing.T) {
	tempDir := t.TempDir()

	// Create test file
	testFile := filepath.Join(tempDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("Test content"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create input file
	inputFile := filepath.Join(tempDir, "input.txt")
	if err := os.WriteFile(inputFile, []byte("<open test.txt>"), 0644); err != nil {
		t.Fatalf("Failed to create input file: %v", err)
	}

	cfg := &config.Config{
		RepositoryRoot:    tempDir,
		MaxFileSize:       1048576,
		MaxWriteSize:      102400,
		AllowedExtensions: []string{".txt"},
		ExcludedPaths:     []string{".git"},
		Interactive:       false,
		InputFile:         inputFile,
	}

	app, err := Bootstrap(cfg)
	if err != nil {
		t.Fatalf("Bootstrap() error = %v", err)
	}

	// Run multiple times
	for i := 0; i < 3; i++ {
		stdout := captureStdout(t, func() {
			err = app.Run()
		})

		if err != nil {
			t.Errorf("Run() #%d error = %v", i+1, err)
		}

		if !strings.Contains(stdout, "Test content") {
			t.Errorf("Run() #%d missing content\nGot: %s", i+1, stdout)
		}
	}
}

func TestApp_Run_VerboseMode_ExecDisabled(t *testing.T) {
	tempDir := t.TempDir()

	cfg := &config.Config{
		RepositoryRoot:    tempDir,
		MaxFileSize:       1048576,
		MaxWriteSize:      102400,
		AllowedExtensions: []string{".txt"},
		ExcludedPaths:     []string{".git"},
		Interactive:       false,
		Verbose:           true,
		ExecEnabled:       false, // Exec disabled
	}

	app, err := Bootstrap(cfg)
	if err != nil {
		t.Fatalf("Bootstrap() error = %v", err)
	}

	// Mock stdin with empty input
	oldStdin := os.Stdin
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("Failed to create stdin pipe: %v", err)
	}
	os.Stdin = r
	w.Close()

	stderr := captureStderr(t, func() {
		_ = app.Run()
	})

	os.Stdin = oldStdin
	r.Close()

	// Should show basic info but NOT exec details when exec is disabled
	if !strings.Contains(stderr, "Repository root:") {
		t.Errorf("Verbose output should show repository root\nGot: %s", stderr)
	}
	if !strings.Contains(stderr, "Exec enabled: false") {
		t.Errorf("Verbose output should show exec disabled\nGot: %s", stderr)
	}
	// Should NOT show exec whitelist when exec is disabled
	if strings.Contains(stderr, "Exec whitelist:") {
		t.Errorf("Verbose output should NOT show exec whitelist when disabled\nGot: %s", stderr)
	}
}

func TestApp_RunSearchCommand_Reindex_SearchDisabled(t *testing.T) {
	tempDir := t.TempDir()

	cfg := &config.Config{
		RepositoryRoot:    tempDir,
		MaxFileSize:       1048576,
		MaxWriteSize:      102400,
		AllowedExtensions: []string{".txt"},
		ExcludedPaths:     []string{".git"},
	}

	app, err := Bootstrap(cfg)
	if err != nil {
		t.Fatalf("Bootstrap() error = %v", err)
	}

	flags := &cli.CLIFlags{
		Reindex: true,
	}

	err = app.RunSearchCommand(flags)
	// Should fail because search config is not enabled
	if err == nil {
		// If search happens to be enabled via config file, skip this test
		t.Skip("Search is enabled, skipping disabled test")
	}

	if !strings.Contains(err.Error(), "search") {
		t.Errorf("Error should mention search, got: %v", err)
	}
}

func TestApp_RunSearchCommand_SearchStatus_SearchDisabled(t *testing.T) {
	tempDir := t.TempDir()

	cfg := &config.Config{
		RepositoryRoot:    tempDir,
		MaxFileSize:       1048576,
		MaxWriteSize:      102400,
		AllowedExtensions: []string{".txt"},
		ExcludedPaths:     []string{".git"},
	}

	app, err := Bootstrap(cfg)
	if err != nil {
		t.Fatalf("Bootstrap() error = %v", err)
	}

	flags := &cli.CLIFlags{
		SearchStatus: true,
	}

	err = app.RunSearchCommand(flags)
	if err == nil {
		t.Skip("Search is enabled, skipping disabled test")
	}

	if !strings.Contains(err.Error(), "search") {
		t.Errorf("Error should mention search, got: %v", err)
	}
}

func TestApp_RunSearchCommand_SearchValidate_SearchDisabled(t *testing.T) {
	tempDir := t.TempDir()

	cfg := &config.Config{
		RepositoryRoot:    tempDir,
		MaxFileSize:       1048576,
		MaxWriteSize:      102400,
		AllowedExtensions: []string{".txt"},
		ExcludedPaths:     []string{".git"},
	}

	app, err := Bootstrap(cfg)
	if err != nil {
		t.Fatalf("Bootstrap() error = %v", err)
	}

	flags := &cli.CLIFlags{
		SearchValidate: true,
	}

	err = app.RunSearchCommand(flags)
	if err == nil {
		t.Skip("Search is enabled, skipping disabled test")
	}

	if !strings.Contains(err.Error(), "search") {
		t.Errorf("Error should mention search, got: %v", err)
	}
}

func TestApp_RunSearchCommand_SearchCleanup_SearchDisabled(t *testing.T) {
	tempDir := t.TempDir()

	cfg := &config.Config{
		RepositoryRoot:    tempDir,
		MaxFileSize:       1048576,
		MaxWriteSize:      102400,
		AllowedExtensions: []string{".txt"},
		ExcludedPaths:     []string{".git"},
	}

	app, err := Bootstrap(cfg)
	if err != nil {
		t.Fatalf("Bootstrap() error = %v", err)
	}

	flags := &cli.CLIFlags{
		SearchCleanup: true,
	}

	err = app.RunSearchCommand(flags)
	if err == nil {
		t.Skip("Search is enabled, skipping disabled test")
	}

	if !strings.Contains(err.Error(), "search") {
		t.Errorf("Error should mention search, got: %v", err)
	}
}

func TestApp_RunSearchCommand_SearchUpdate_SearchDisabled(t *testing.T) {
	tempDir := t.TempDir()

	cfg := &config.Config{
		RepositoryRoot:    tempDir,
		MaxFileSize:       1048576,
		MaxWriteSize:      102400,
		AllowedExtensions: []string{".txt"},
		ExcludedPaths:     []string{".git"},
	}

	app, err := Bootstrap(cfg)
	if err != nil {
		t.Fatalf("Bootstrap() error = %v", err)
	}

	flags := &cli.CLIFlags{
		SearchUpdate: true,
	}

	err = app.RunSearchCommand(flags)
	if err == nil {
		t.Skip("Search is enabled, skipping disabled test")
	}

	if !strings.Contains(err.Error(), "search") {
		t.Errorf("Error should mention search, got: %v", err)
	}
}

func TestApp_RunSearchCommand_NoFlags(t *testing.T) {
	tempDir := t.TempDir()

	cfg := &config.Config{
		RepositoryRoot:    tempDir,
		MaxFileSize:       1048576,
		MaxWriteSize:      102400,
		AllowedExtensions: []string{".txt"},
		ExcludedPaths:     []string{".git"},
	}

	app, err := Bootstrap(cfg)
	if err != nil {
		t.Fatalf("Bootstrap() error = %v", err)
	}

	// No search flags set
	flags := &cli.CLIFlags{}

	err = app.RunSearchCommand(flags)
	// With no flags, behavior depends on search being enabled or not
	// Either way, it shouldn't panic
	_ = err
}

func TestApp_Run_InteractiveMode(t *testing.T) {
	tempDir := t.TempDir()

	// Create a test file
	testFile := filepath.Join(tempDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("Interactive test content"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	cfg := &config.Config{
		RepositoryRoot:    tempDir,
		MaxFileSize:       1048576,
		MaxWriteSize:      102400,
		AllowedExtensions: []string{".txt"},
		ExcludedPaths:     []string{".git"},
		Interactive:       true, // Enable interactive mode
	}

	app, err := Bootstrap(cfg)
	if err != nil {
		t.Fatalf("Bootstrap() error = %v", err)
	}

	// Mock stdin with a command
	oldStdin := os.Stdin
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("Failed to create stdin pipe: %v", err)
	}
	os.Stdin = r

	go func() {
		defer w.Close()
		w.WriteString("<open test.txt>\n")
	}()

	var stdout string
	var stderr string

	// Capture both stdout and stderr
	oldStdout := os.Stdout
	rOut, wOut, _ := os.Pipe()
	os.Stdout = wOut

	oldStderr := os.Stderr
	rErr, wErr, _ := os.Pipe()
	os.Stderr = wErr

	err = app.Run()

	wOut.Close()
	wErr.Close()
	os.Stdout = oldStdout
	os.Stderr = oldStderr

	var bufOut, bufErr bytes.Buffer
	io.Copy(&bufOut, rOut)
	io.Copy(&bufErr, rErr)
	rOut.Close()
	rErr.Close()

	stdout = bufOut.String()
	stderr = bufErr.String()

	os.Stdin = oldStdin
	r.Close()

	if err != nil {
		t.Errorf("Run() error = %v", err)
	}

	// Interactive mode should show welcome message
	if !strings.Contains(stderr, "Interactive Mode") {
		t.Errorf("Expected Interactive Mode message\nStderr: %s", stderr)
	}

	// Should process the open command
	if !strings.Contains(stdout, "Interactive test content") {
		t.Errorf("Expected file content in output\nStdout: %s", stdout)
	}
}

func TestApp_Run_VerboseMode_AllFields(t *testing.T) {
	tempDir := t.TempDir()

	cfg := &config.Config{
		RepositoryRoot:     tempDir,
		MaxFileSize:        2097152,
		MaxWriteSize:       204800,
		AllowedExtensions:  []string{".txt", ".go", ".md"},
		ExcludedPaths:      []string{".git", "vendor", "node_modules"},
		Interactive:        false,
		Verbose:            true,
		BackupBeforeWrite:  true,
		ExecEnabled:        true,
		ExecWhitelist:      []string{"go test", "go build", "make"},
		ExecTimeout:        60000000000, // 60s in nanoseconds
		ExecContainerImage: "golang:1.21",
	}

	app, err := Bootstrap(cfg)
	if err != nil {
		t.Fatalf("Bootstrap() error = %v", err)
	}

	// Mock stdin with empty input
	oldStdin := os.Stdin
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("Failed to create stdin pipe: %v", err)
	}
	os.Stdin = r
	w.Close()

	stderr := captureStderr(t, func() {
		_ = app.Run()
	})

	os.Stdin = oldStdin
	r.Close()

	// Check all verbose output fields
	checks := []string{
		"Repository root:",
		"Max file size:",
		"Max write file size:",
		"Allowed extensions:",
		"Excluded paths:",
		"Backup enabled:",
		"Exec enabled:",
		"Exec whitelist:",
		"Exec timeout:",
		"Exec image:",
	}

	for _, check := range checks {
		if !strings.Contains(stderr, check) {
			t.Errorf("Verbose output missing: %q\nGot: %s", check, stderr)
		}
	}
}

func TestApp_GettersAfterMultipleOperations(t *testing.T) {
	tempDir := t.TempDir()

	cfg := &config.Config{
		RepositoryRoot:    tempDir,
		MaxFileSize:       1048576,
		MaxWriteSize:      102400,
		AllowedExtensions: []string{".txt"},
		ExcludedPaths:     []string{".git"},
	}

	app, err := Bootstrap(cfg)
	if err != nil {
		t.Fatalf("Bootstrap() error = %v", err)
	}

	// Get initial state
	initialSession := app.GetSession()
	initialConfig := app.GetConfig()
	initialExecutor := app.GetExecutor()
	initialSearchConfig := app.GetSearchConfig()

	// Run some operations
	inputFile := filepath.Join(tempDir, "input.txt")
	if err := os.WriteFile(inputFile, []byte("<write test.txt>content</write>"), 0644); err != nil {
		t.Fatalf("Failed to create input file: %v", err)
	}
	app.GetConfig().InputFile = inputFile

	captureStdout(t, func() {
		_ = app.Run()
	})

	// Verify getters still return same objects
	if app.GetSession() != initialSession {
		t.Error("GetSession() returned different object after Run()")
	}
	if app.GetConfig() != initialConfig {
		t.Error("GetConfig() returned different object after Run()")
	}
	if app.GetExecutor() != initialExecutor {
		t.Error("GetExecutor() returned different object after Run()")
	}
	if app.GetSearchConfig() != initialSearchConfig {
		t.Error("GetSearchConfig() returned different object after Run()")
	}
}

func TestApp_Run_LargeInput(t *testing.T) {
	tempDir := t.TempDir()

	// Create a large input with many commands
	var inputBuilder strings.Builder
	for i := 0; i < 10; i++ {
		inputBuilder.WriteString(fmt.Sprintf("<write file%d.txt>Content for file %d</write>\n", i, i))
	}

	inputFile := filepath.Join(tempDir, "input.txt")
	if err := os.WriteFile(inputFile, []byte(inputBuilder.String()), 0644); err != nil {
		t.Fatalf("Failed to create input file: %v", err)
	}

	cfg := &config.Config{
		RepositoryRoot:    tempDir,
		MaxFileSize:       1048576,
		MaxWriteSize:      102400,
		AllowedExtensions: []string{".txt"},
		ExcludedPaths:     []string{".git"},
		Interactive:       false,
		InputFile:         inputFile,
	}

	app, err := Bootstrap(cfg)
	if err != nil {
		t.Fatalf("Bootstrap() error = %v", err)
	}

	stdout := captureStdout(t, func() {
		err = app.Run()
	})

	if err != nil {
		t.Errorf("Run() error = %v", err)
	}

	// Should have 10 successful writes
	successCount := strings.Count(stdout, "WRITE SUCCESSFUL")
	if successCount != 10 {
		t.Errorf("Expected 10 successful writes, got %d\nOutput: %s", successCount, stdout)
	}

	// Verify all files were created
	for i := 0; i < 10; i++ {
		filePath := filepath.Join(tempDir, fmt.Sprintf("file%d.txt", i))
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			t.Errorf("file%d.txt was not created", i)
		}
	}
}

func TestApp_RunSearchCommand_WithSearchConfig(t *testing.T) {
	tempDir := t.TempDir()

	// Create a config file that enables search
	configContent := `
repository:
  root: "."
  excluded_paths: [".git"]
commands:
  search:
    enabled: true
    vector_db_path: "test_search.db"
    embedding_model: "all-MiniLM-L6-v2"
    max_results: 10
    min_similarity_score: 0.5
    python_path: "python3"
`
	configPath := filepath.Join(tempDir, "llm-runtime.config.yaml")
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to create config file: %v", err)
	}

	// Change to tempDir so config is found
	oldDir, _ := os.Getwd()
	defer os.Chdir(oldDir)
	os.Chdir(tempDir)

	cfg := &config.Config{
		RepositoryRoot:    tempDir,
		MaxFileSize:       1048576,
		MaxWriteSize:      102400,
		AllowedExtensions: []string{".txt"},
		ExcludedPaths:     []string{".git"},
	}

	app, err := Bootstrap(cfg)
	if err != nil {
		t.Fatalf("Bootstrap() error = %v", err)
	}

	// Test various search commands - they may fail due to missing Python/DB
	// but should at least attempt to run (covering more code paths)

	searchTests := []struct {
		name  string
		flags cli.CLIFlags
	}{
		{"Reindex", cli.CLIFlags{Reindex: true}},
		{"SearchStatus", cli.CLIFlags{SearchStatus: true}},
		{"SearchValidate", cli.CLIFlags{SearchValidate: true}},
		{"SearchCleanup", cli.CLIFlags{SearchCleanup: true}},
		{"SearchUpdate", cli.CLIFlags{SearchUpdate: true}},
	}

	for _, tt := range searchTests {
		t.Run(tt.name, func(t *testing.T) {
			// These will likely fail due to Python not being available,
			// but they exercise the code paths
			_ = app.RunSearchCommand(&tt.flags)
		})
	}
}

func TestApp_Run_PipeMode_StdinError(t *testing.T) {
	tempDir := t.TempDir()

	cfg := &config.Config{
		RepositoryRoot:    tempDir,
		MaxFileSize:       1048576,
		MaxWriteSize:      102400,
		AllowedExtensions: []string{".txt"},
		ExcludedPaths:     []string{".git"},
		Interactive:       false,
		// No InputFile - reads from stdin
	}

	app, err := Bootstrap(cfg)
	if err != nil {
		t.Fatalf("Bootstrap() error = %v", err)
	}

	// Close stdin to simulate error
	oldStdin := os.Stdin
	r, w, _ := os.Pipe()
	os.Stdin = r
	w.Close()
	r.Close() // Close read end too to cause error

	// Create new pipe for stdin that's already closed
	r2, w2, _ := os.Pipe()
	w2.Close()
	os.Stdin = r2

	captureStdout(t, func() {
		// This should handle the closed stdin gracefully
		_ = app.Run()
	})

	os.Stdin = oldStdin
	r2.Close()
}

func TestApp_Run_WithAllCommandTypes(t *testing.T) {
	tempDir := t.TempDir()

	// Create a file to open
	existingFile := filepath.Join(tempDir, "existing.txt")
	if err := os.WriteFile(existingFile, []byte("Existing content"), 0644); err != nil {
		t.Fatalf("Failed to create existing file: %v", err)
	}

	// Input with multiple command types
	input := `Let me open a file:
<open existing.txt>
Now write a new file:
<write new.txt>New content here</write>
Try exec (will fail):
<exec echo hello>
Try search (will fail):
<search test query>
`

	inputFile := filepath.Join(tempDir, "input.txt")
	if err := os.WriteFile(inputFile, []byte(input), 0644); err != nil {
		t.Fatalf("Failed to create input file: %v", err)
	}

	cfg := &config.Config{
		RepositoryRoot:    tempDir,
		MaxFileSize:       1048576,
		MaxWriteSize:      102400,
		AllowedExtensions: []string{".txt"},
		ExcludedPaths:     []string{".git"},
		Interactive:       false,
		InputFile:         inputFile,
		ExecEnabled:       false, // Exec disabled
	}

	app, err := Bootstrap(cfg)
	if err != nil {
		t.Fatalf("Bootstrap() error = %v", err)
	}

	stdout := captureStdout(t, func() {
		err = app.Run()
	})

	if err != nil {
		t.Errorf("Run() error = %v", err)
	}

	// Should have open success
	if !strings.Contains(stdout, "Existing content") {
		t.Error("Expected open command to show file content")
	}

	// Should have write success
	if !strings.Contains(stdout, "WRITE SUCCESSFUL") {
		t.Error("Expected write command to succeed")
	}

	// Should have exec error (disabled)
	if !strings.Contains(stdout, "ERROR") {
		t.Error("Expected exec command to show error")
	}
}

func TestApp_Run_Verbose_BackupDisabled(t *testing.T) {
	tempDir := t.TempDir()

	cfg := &config.Config{
		RepositoryRoot:    tempDir,
		MaxFileSize:       1048576,
		MaxWriteSize:      102400,
		AllowedExtensions: []string{".txt"},
		ExcludedPaths:     []string{".git"},
		Interactive:       false,
		Verbose:           true,
		BackupBeforeWrite: false, // Backup disabled
		ExecEnabled:       false,
	}

	app, err := Bootstrap(cfg)
	if err != nil {
		t.Fatalf("Bootstrap() error = %v", err)
	}

	// Mock stdin with empty input
	oldStdin := os.Stdin
	r, w, _ := os.Pipe()
	os.Stdin = r
	w.Close()

	stderr := captureStderr(t, func() {
		_ = app.Run()
	})

	os.Stdin = oldStdin
	r.Close()

	// Should show backup as false
	if !strings.Contains(stderr, "Backup enabled: false") {
		t.Errorf("Expected 'Backup enabled: false' in output\nGot: %s", stderr)
	}
}

func TestApp_SessionConfigReference(t *testing.T) {
	tempDir := t.TempDir()

	cfg := &config.Config{
		RepositoryRoot:    tempDir,
		MaxFileSize:       1048576,
		MaxWriteSize:      102400,
		AllowedExtensions: []string{".txt"},
		ExcludedPaths:     []string{".git"},
	}

	app, err := Bootstrap(cfg)
	if err != nil {
		t.Fatalf("Bootstrap() error = %v", err)
	}

	// Session should have reference to same config
	session := app.GetSession()
	if session.Config == nil {
		t.Error("Session.Config should not be nil")
	}

	// The config in session should match app config
	if session.Config.MaxFileSize != app.GetConfig().MaxFileSize {
		t.Error("Session config should match app config")
	}
}

func TestApp_ExecutorSearchConfig(t *testing.T) {
	tempDir := t.TempDir()

	cfg := &config.Config{
		RepositoryRoot:    tempDir,
		MaxFileSize:       1048576,
		MaxWriteSize:      102400,
		AllowedExtensions: []string{".txt"},
		ExcludedPaths:     []string{".git"},
	}

	app, err := Bootstrap(cfg)
	if err != nil {
		t.Fatalf("Bootstrap() error = %v", err)
	}

	// Executor should have search config (may or may not be enabled)
	execSearchCfg := app.GetExecutor().GetSearchConfig()
	appSearchCfg := app.GetSearchConfig()

	// Both should be the same reference or both nil
	if execSearchCfg != appSearchCfg {
		t.Error("Executor search config should match app search config")
	}
}

func TestApp_Run_OutputToFile_Success(t *testing.T) {
	tempDir := t.TempDir()

	// Create input with a simple command
	inputFile := filepath.Join(tempDir, "input.txt")
	if err := os.WriteFile(inputFile, []byte("<write created.txt>File created!</write>"), 0644); err != nil {
		t.Fatalf("Failed to create input file: %v", err)
	}

	outputFile := filepath.Join(tempDir, "output.txt")

	cfg := &config.Config{
		RepositoryRoot:    tempDir,
		MaxFileSize:       1048576,
		MaxWriteSize:      102400,
		AllowedExtensions: []string{".txt"},
		ExcludedPaths:     []string{".git"},
		Interactive:       false,
		InputFile:         inputFile,
		OutputFile:        outputFile,
	}

	app, err := Bootstrap(cfg)
	if err != nil {
		t.Fatalf("Bootstrap() error = %v", err)
	}

	err = app.Run()
	if err != nil {
		t.Errorf("Run() error = %v", err)
	}

	// Verify output file exists and has content
	content, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}

	if len(content) == 0 {
		t.Error("Output file should not be empty")
	}

	if !strings.Contains(string(content), "WRITE SUCCESSFUL") {
		t.Errorf("Output file should contain success message\nGot: %s", string(content))
	}
}

func TestApp_Run_InteractiveMode_EmptyInput(t *testing.T) {
	tempDir := t.TempDir()

	cfg := &config.Config{
		RepositoryRoot:    tempDir,
		MaxFileSize:       1048576,
		MaxWriteSize:      102400,
		AllowedExtensions: []string{".txt"},
		ExcludedPaths:     []string{".git"},
		Interactive:       true,
	}

	app, err := Bootstrap(cfg)
	if err != nil {
		t.Fatalf("Bootstrap() error = %v", err)
	}

	// Mock stdin with immediate EOF
	oldStdin := os.Stdin
	r, w, _ := os.Pipe()
	os.Stdin = r
	w.Close() // Immediate EOF

	oldStderr := os.Stderr
	rErr, wErr, _ := os.Pipe()
	os.Stderr = wErr

	err = app.Run()

	wErr.Close()
	os.Stderr = oldStderr
	var buf bytes.Buffer
	io.Copy(&buf, rErr)
	rErr.Close()
	stderr := buf.String()

	os.Stdin = oldStdin
	r.Close()

	if err != nil {
		t.Errorf("Run() error = %v", err)
	}

	// Should show welcome message
	if !strings.Contains(stderr, "LLM Tool") {
		t.Errorf("Expected welcome message in stderr\nGot: %s", stderr)
	}
}

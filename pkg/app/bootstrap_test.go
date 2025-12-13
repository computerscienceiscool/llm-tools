package app

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/computerscienceiscool/llm-runtime/pkg/config"
)

func TestBootstrap_Success(t *testing.T) {
	tempDir := t.TempDir()

	cfg := &config.Config{
		RepositoryRoot:    tempDir,
		MaxFileSize:       1048576,
		MaxWriteSize:      102400,
		AllowedExtensions: []string{".go", ".txt"},
		ExcludedPaths:     []string{".git"},
		BackupBeforeWrite: true,
	}

	app, err := Bootstrap(cfg)
	if err != nil {
		t.Fatalf("Bootstrap() error = %v", err)
	}

	if app == nil {
		t.Fatal("Bootstrap() returned nil app")
	}

	// Verify config is set
	if app.GetConfig() == nil {
		t.Error("App config is nil")
	}

	// Verify session is created
	if app.GetSession() == nil {
		t.Error("App session is nil")
	}

	// Verify executor is created
	if app.GetExecutor() == nil {
		t.Error("App executor is nil")
	}

	// Verify repository root is resolved to absolute path
	if !filepath.IsAbs(app.GetConfig().RepositoryRoot) {
		t.Errorf("RepositoryRoot should be absolute, got %q", app.GetConfig().RepositoryRoot)
	}
}

func TestBootstrap_ResolvesRelativePath(t *testing.T) {
	// Create a temp directory and change to it
	tempDir := t.TempDir()
	subDir := filepath.Join(tempDir, "subdir")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatalf("Failed to create subdir: %v", err)
	}

	// Save current directory
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer os.Chdir(originalDir)

	// Change to temp directory
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change directory: %v", err)
	}

	cfg := &config.Config{
		RepositoryRoot:    "subdir", // Relative path
		MaxFileSize:       1048576,
		MaxWriteSize:      102400,
		AllowedExtensions: []string{".txt"},
		ExcludedPaths:     []string{".git"},
	}

	app, err := Bootstrap(cfg)
	if err != nil {
		t.Fatalf("Bootstrap() error = %v", err)
	}

	// Should be resolved to absolute path
	if !filepath.IsAbs(app.GetConfig().RepositoryRoot) {
		t.Errorf("RepositoryRoot should be absolute, got %q", app.GetConfig().RepositoryRoot)
	}

	// Should point to the subdir
	if filepath.Base(app.GetConfig().RepositoryRoot) != "subdir" {
		t.Errorf("RepositoryRoot should end with 'subdir', got %q", app.GetConfig().RepositoryRoot)
	}
}

func TestBootstrap_CurrentDirectory(t *testing.T) {
	// Use current directory as root
	currentDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}

	cfg := &config.Config{
		RepositoryRoot:    ".",
		MaxFileSize:       1048576,
		MaxWriteSize:      102400,
		AllowedExtensions: []string{".txt"},
		ExcludedPaths:     []string{".git"},
	}

	app, err := Bootstrap(cfg)
	if err != nil {
		t.Fatalf("Bootstrap() error = %v", err)
	}

	// Should resolve to current directory
	if app.GetConfig().RepositoryRoot != currentDir {
		t.Errorf("RepositoryRoot = %q, want %q", app.GetConfig().RepositoryRoot, currentDir)
	}
}

func TestBootstrap_NonExistentRoot(t *testing.T) {
	cfg := &config.Config{
		RepositoryRoot:    "/nonexistent/path/that/does/not/exist",
		MaxFileSize:       1048576,
		MaxWriteSize:      102400,
		AllowedExtensions: []string{".txt"},
		ExcludedPaths:     []string{".git"},
	}

	app, err := Bootstrap(cfg)
	if err == nil {
		t.Error("Bootstrap() should fail for non-existent root")
	}
	if app != nil {
		t.Error("Bootstrap() should return nil app on error")
	}
}

func TestBootstrap_FileAsRoot(t *testing.T) {
	tempDir := t.TempDir()

	// Create a file instead of directory
	filePath := filepath.Join(tempDir, "file.txt")
	if err := os.WriteFile(filePath, []byte("content"), 0644); err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}

	cfg := &config.Config{
		RepositoryRoot:    filePath, // File, not directory
		MaxFileSize:       1048576,
		MaxWriteSize:      102400,
		AllowedExtensions: []string{".txt"},
		ExcludedPaths:     []string{".git"},
	}

	// Bootstrap should succeed (os.Stat passes for files)
	// but subsequent operations may fail
	app, err := Bootstrap(cfg)
	if err != nil {
		// Some implementations may reject files as root
		return
	}

	// If it succeeds, verify it's set
	if app.GetConfig().RepositoryRoot != filePath {
		t.Errorf("RepositoryRoot = %q, want %q", app.GetConfig().RepositoryRoot, filePath)
	}
}

func TestBootstrap_SessionHasUniqueID(t *testing.T) {
	tempDir := t.TempDir()

	cfg := &config.Config{
		RepositoryRoot:    tempDir,
		MaxFileSize:       1048576,
		MaxWriteSize:      102400,
		AllowedExtensions: []string{".txt"},
		ExcludedPaths:     []string{".git"},
	}

	app1, err := Bootstrap(cfg)
	if err != nil {
		t.Fatalf("Bootstrap() error = %v", err)
	}

	app2, err := Bootstrap(cfg)
	if err != nil {
		t.Fatalf("Bootstrap() error = %v", err)
	}

	// Each session should have a unique ID
	if app1.GetSession().ID == app2.GetSession().ID {
		t.Error("Different bootstrap calls should create sessions with different IDs")
	}
}

func TestBootstrap_SessionHasStartTime(t *testing.T) {
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

	if app.GetSession().StartTime.IsZero() {
		t.Error("Session StartTime should not be zero")
	}
}

func TestBootstrap_ExecutorHasConfig(t *testing.T) {
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

	execCfg := app.GetExecutor().GetConfig()
	if execCfg == nil {
		t.Fatal("Executor config is nil")
	}

	// Verify executor has the same config values
	if execCfg.MaxFileSize != cfg.MaxFileSize {
		t.Errorf("Executor MaxFileSize = %d, want %d", execCfg.MaxFileSize, cfg.MaxFileSize)
	}
	if execCfg.MaxWriteSize != cfg.MaxWriteSize {
		t.Errorf("Executor MaxWriteSize = %d, want %d", execCfg.MaxWriteSize, cfg.MaxWriteSize)
	}
}

func TestBootstrap_SearchConfigLoaded(t *testing.T) {
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

	// Search config may or may not be enabled, but should be loaded
	searchCfg := app.GetSearchConfig()
	if searchCfg == nil {
		t.Error("Search config should not be nil")
	}
}

// func TestBootstrap_ConfigWithExecEnabled(t *testing.T) {
// 	tempDir := t.TempDir()
// 
// 	cfg := &config.Config{
// 		RepositoryRoot:     tempDir,
// 		MaxFileSize:        1048576,
// 		MaxWriteSize:       102400,
// 		AllowedExtensions:  []string{".txt"},
// 		ExcludedPaths:      []string{".git"},
// 		ExecWhitelist:      []string{"go test", "make"},
// 		ExecContainerImage: "alpine:latest",
// 		ExecMemoryLimit:    "256m",
// 		ExecCPULimit:       1,
// 	}
// 
// 	app, err := Bootstrap(cfg)
// 	if err != nil {
// 		t.Fatalf("Bootstrap() error = %v", err)
// 	}
// 
// 	execCfg := app.GetExecutor().GetConfig()
// 	if !execCfg.ExecEnabled {
// 		t.Error("ExecEnabled should be true")
// 	}
// 	if len(execCfg.ExecWhitelist) != 2 {
// 		t.Errorf("ExecWhitelist length = %d, want 2", len(execCfg.ExecWhitelist))
// 	}
// 	if execCfg.ExecContainerImage != "alpine:latest" {
// 		t.Errorf("ExecContainerImage = %q, want %q", execCfg.ExecContainerImage, "alpine:latest")
// 	}
// }

func TestBootstrap_EmptyConfig(t *testing.T) {
	tempDir := t.TempDir()

	// Minimal config with just repository root
	cfg := &config.Config{
		RepositoryRoot: tempDir,
	}

	app, err := Bootstrap(cfg)
	if err != nil {
		t.Fatalf("Bootstrap() error = %v", err)
	}

	if app == nil {
		t.Fatal("Bootstrap() returned nil app")
	}

	// Should still have valid components
	if app.GetConfig() == nil {
		t.Error("App config is nil")
	}
	if app.GetSession() == nil {
		t.Error("App session is nil")
	}
	if app.GetExecutor() == nil {
		t.Error("App executor is nil")
	}
}

func TestBootstrap_SymlinkRoot(t *testing.T) {
	tempDir := t.TempDir()
	realDir := filepath.Join(tempDir, "real")
	linkDir := filepath.Join(tempDir, "link")

	// Create real directory
	if err := os.MkdirAll(realDir, 0755); err != nil {
		t.Fatalf("Failed to create real directory: %v", err)
	}

	// Create symlink
	if err := os.Symlink(realDir, linkDir); err != nil {
		t.Skipf("Cannot create symlink (maybe no permission): %v", err)
	}

	cfg := &config.Config{
		RepositoryRoot:    linkDir,
		MaxFileSize:       1048576,
		MaxWriteSize:      102400,
		AllowedExtensions: []string{".txt"},
		ExcludedPaths:     []string{".git"},
	}

	app, err := Bootstrap(cfg)
	if err != nil {
		t.Fatalf("Bootstrap() error = %v", err)
	}

	// Should resolve to absolute path (may or may not resolve symlink)
	if !filepath.IsAbs(app.GetConfig().RepositoryRoot) {
		t.Errorf("RepositoryRoot should be absolute, got %q", app.GetConfig().RepositoryRoot)
	}
}

func TestBootstrap_PreservesAllConfigFields(t *testing.T) {
	tempDir := t.TempDir()

	cfg := &config.Config{
		RepositoryRoot:      tempDir,
		MaxFileSize:         2097152,
		MaxWriteSize:        204800,
		ExcludedPaths:       []string{".git", "node_modules", "vendor"},
		Interactive:         true,
		InputFile:           "input.txt",
		OutputFile:          "output.txt",
		JSONOutput:          true,
		Verbose:             true,
		RequireConfirmation: true,
		BackupBeforeWrite:   true,
		AllowedExtensions:   []string{".go", ".py", ".js"},
		ForceWrite:          true,
		ExecWhitelist:       []string{"go test"},
		ExecMemoryLimit:     "1g",
		ExecCPULimit:        4,
		ExecContainerImage:  "golang:latest",
		ExecNetworkEnabled:  true,
	}

	app, err := Bootstrap(cfg)
	if err != nil {
		t.Fatalf("Bootstrap() error = %v", err)
	}

	appCfg := app.GetConfig()

	// Verify all fields are preserved (except RepositoryRoot which is made absolute)
	if appCfg.MaxFileSize != cfg.MaxFileSize {
		t.Errorf("MaxFileSize = %d, want %d", appCfg.MaxFileSize, cfg.MaxFileSize)
	}
	if appCfg.MaxWriteSize != cfg.MaxWriteSize {
		t.Errorf("MaxWriteSize = %d, want %d", appCfg.MaxWriteSize, cfg.MaxWriteSize)
	}
	if appCfg.Interactive != cfg.Interactive {
		t.Errorf("Interactive = %v, want %v", appCfg.Interactive, cfg.Interactive)
	}
	if appCfg.InputFile != cfg.InputFile {
		t.Errorf("InputFile = %q, want %q", appCfg.InputFile, cfg.InputFile)
	}
	if appCfg.OutputFile != cfg.OutputFile {
		t.Errorf("OutputFile = %q, want %q", appCfg.OutputFile, cfg.OutputFile)
	}
	if appCfg.JSONOutput != cfg.JSONOutput {
		t.Errorf("JSONOutput = %v, want %v", appCfg.JSONOutput, cfg.JSONOutput)
	}
	if appCfg.Verbose != cfg.Verbose {
		t.Errorf("Verbose = %v, want %v", appCfg.Verbose, cfg.Verbose)
	}
	if appCfg.RequireConfirmation != cfg.RequireConfirmation {
		t.Errorf("RequireConfirmation = %v, want %v", appCfg.RequireConfirmation, cfg.RequireConfirmation)
	}
	if appCfg.BackupBeforeWrite != cfg.BackupBeforeWrite {
		t.Errorf("BackupBeforeWrite = %v, want %v", appCfg.BackupBeforeWrite, cfg.BackupBeforeWrite)
	}
	if appCfg.ForceWrite != cfg.ForceWrite {
		t.Errorf("ForceWrite = %v, want %v", appCfg.ForceWrite, cfg.ForceWrite)
	}
	}
	if appCfg.ExecNetworkEnabled != cfg.ExecNetworkEnabled {
		t.Errorf("ExecNetworkEnabled = %v, want %v", appCfg.ExecNetworkEnabled, cfg.ExecNetworkEnabled)
	}
}

func TestBootstrap_ExecutorCommandsRunStartsAtZero(t *testing.T) {
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

	if app.GetExecutor().GetCommandsRun() != 0 {
		t.Errorf("CommandsRun should start at 0, got %d", app.GetExecutor().GetCommandsRun())
	}
}

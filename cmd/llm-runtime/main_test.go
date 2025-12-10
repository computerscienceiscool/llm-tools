package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestMainPackageBuilds verifies the main package compiles correctly
func TestMainPackageBuilds(t *testing.T) {
	cmd := exec.Command("go", "build", "-o", os.DevNull, ".")
	cmd.Dir = getPackageDir(t)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("main package failed to build: %v\nOutput: %s", err, output)
	}
}

// TestCLIHelp verifies the help flag works and shows expected output
func TestCLIHelp(t *testing.T) {
	binary := buildTestBinary(t)
	defer os.Remove(binary)

	cmd := exec.Command(binary, "--help")
	output, _ := cmd.CombinedOutput()

	// Help should mention key flags
	expectedFlags := []string{
		"root",
		"interactive",
		"verbose",
	}

	outputStr := string(output)
	for _, flag := range expectedFlags {
		if !strings.Contains(outputStr, flag) {
			t.Errorf("help output missing expected flag: %s", flag)
		}
	}
}

// TestCLIInvalidFlag verifies invalid flags are rejected
func TestCLIInvalidFlag(t *testing.T) {
	binary := buildTestBinary(t)
	defer os.Remove(binary)

	cmd := exec.Command(binary, "--nonexistent-flag")
	err := cmd.Run()
	if err == nil {
		t.Error("expected error for invalid flag, got nil")
	}
}

// TestSearchSubcommands verifies search subcommands are recognized
func TestSearchSubcommands(t *testing.T) {
	binary := buildTestBinary(t)
	defer os.Remove(binary)

	// These subcommands should be recognized (may fail due to missing config, but should not fail on command parsing)
	subcommands := []string{
		"reindex",
		"search-status",
		"search-validate",
		"search-cleanup",
		"search-update",
		"check-ollama",
	}

	for _, subcmd := range subcommands {
		t.Run(subcmd, func(t *testing.T) {
			cmd := exec.Command(binary, subcmd, "--help")
			output, err := cmd.CombinedOutput()
			outputStr := string(output)

			// Help should work for each subcommand
			if err != nil && !strings.Contains(outputStr, "Usage:") {
				t.Errorf("subcommand %s help failed: %v\nOutput: %s", subcmd, err, outputStr)
			}

			// Should show usage for the subcommand
			if !strings.Contains(outputStr, subcmd) {
				t.Errorf("help output for %s doesn't mention the command\nOutput: %s", subcmd, outputStr)
			}
		})
	}
}

// TestBootstrapWithInvalidRoot verifies bootstrap fails gracefully with invalid root
func TestBootstrapWithInvalidRoot(t *testing.T) {
	binary := buildTestBinary(t)
	defer os.Remove(binary)

	cmd := exec.Command(binary, "--root", "/nonexistent/path/that/does/not/exist")
	err := cmd.Run()
	if err == nil {
		t.Error("expected error for invalid root path, got nil")
	}
}

// TestPipeMode verifies basic pipe mode functionality
func TestPipeMode(t *testing.T) {
	binary := buildTestBinary(t)
	defer os.Remove(binary)

	// Create a temp directory as the repository root
	tempDir := t.TempDir()

	// Create a test file
	testFile := filepath.Join(tempDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("hello world"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	cmd := exec.Command(binary, "--root", tempDir)
	cmd.Stdin = strings.NewReader("Read this: <open test.txt>")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("pipe mode failed: %v\nOutput: %s", err, output)
	}

	outputStr := string(output)

	// Verify output format markers
	expectedMarkers := []string{
		"=== LLM TOOL START ===",
		"=== COMMAND:",
		"=== FILE:",
		"hello world",
		"=== END FILE ===",
		"=== END COMMAND ===",
		"=== LLM TOOL COMPLETE ===",
	}

	for _, marker := range expectedMarkers {
		if !strings.Contains(outputStr, marker) {
			t.Errorf("output missing expected marker: %s\nGot: %s", marker, outputStr)
		}
	}
}

// Helper function to get the package directory
func getPackageDir(t *testing.T) string {
	t.Helper()
	// Get the directory of the test file
	_, filename, _, ok := runtimeCaller(0)
	if !ok {
		t.Fatal("failed to get package directory")
	}
	return filepath.Dir(filename)
}

// runtimeCaller is a variable to allow testing (can be mocked)
var runtimeCaller = func(skip int) (pc uintptr, file string, line int, ok bool) {
	// Use runtime.Caller but import it locally to avoid import in package
	// For simplicity, return current directory
	dir, err := os.Getwd()
	if err != nil {
		return 0, "", 0, false
	}
	return 0, filepath.Join(dir, "main.go"), 0, true
}

// buildTestBinary builds the binary for testing and returns the path
func buildTestBinary(t *testing.T) string {
	t.Helper()

	binary := filepath.Join(t.TempDir(), "llm-runtime-test")
	cmd := exec.Command("go", "build", "-o", binary, ".")
	cmd.Dir = getPackageDir(t)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("failed to build test binary: %v\nOutput: %s", err, output)
	}

	return binary
}

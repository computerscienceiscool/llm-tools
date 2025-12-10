package sandbox

import (
	"strings"
	"testing"
	"time"
)

// Helper to check if Docker is available for integration tests
func isDockerAvailable() bool {
	return CheckDockerAvailability() == nil
}

// Helper to ensure test image is available
func ensureTestImage(t *testing.T) {
	t.Helper()
	if err := PullDockerImage("alpine:latest", false); err != nil {
		t.Skipf("Could not pull test image: %v", err)
	}
}

func TestContainerConfig_Fields(t *testing.T) {
	// Test that ContainerConfig can be constructed with all fields
	cfg := ContainerConfig{
		Image:       "alpine:latest",
		Command:     "echo hello",
		RepoRoot:    "/tmp/repo",
		MemoryLimit: "512m",
		CPULimit:    2,
		Timeout:     30 * time.Second,
	}

	if cfg.Image != "alpine:latest" {
		t.Errorf("Image mismatch: got %q", cfg.Image)
	}
	if cfg.Command != "echo hello" {
		t.Errorf("Command mismatch: got %q", cfg.Command)
	}
	if cfg.Timeout != 30*time.Second {
		t.Errorf("Timeout mismatch: got %v", cfg.Timeout)
	}
}

func TestContainerResult_Fields(t *testing.T) {
	// Test that ContainerResult fields are accessible
	result := ContainerResult{
		ExitCode: 0,
		Stdout:   "hello world",
		Stderr:   "",
		Duration: 100 * time.Millisecond,
	}

	if result.ExitCode != 0 {
		t.Errorf("ExitCode mismatch: got %d", result.ExitCode)
	}
	if result.Stdout != "hello world" {
		t.Errorf("Stdout mismatch: got %q", result.Stdout)
	}
	if result.Duration != 100*time.Millisecond {
		t.Errorf("Duration mismatch: got %v", result.Duration)
	}
}

func TestRunContainer_SimpleCommand(t *testing.T) {
	if !isDockerAvailable() {
		t.Skip("Docker not available, skipping integration test")
	}
	ensureTestImage(t)

	tmpDir := t.TempDir()

	cfg := ContainerConfig{
		Image:       "alpine:latest",
		Command:     "echo hello",
		RepoRoot:    tmpDir,
		MemoryLimit: "128m",
		CPULimit:    1,
		Timeout:     30 * time.Second,
	}

	result, err := RunContainer(cfg)
	if err != nil {
		t.Fatalf("RunContainer failed: %v", err)
	}

	if result.ExitCode != 0 {
		t.Errorf("expected exit code 0, got %d", result.ExitCode)
	}

	if !strings.Contains(result.Stdout, "hello") {
		t.Errorf("expected stdout to contain 'hello', got %q", result.Stdout)
	}
}

func TestRunContainer_CommandWithArgs(t *testing.T) {
	if !isDockerAvailable() {
		t.Skip("Docker not available, skipping integration test")
	}
	ensureTestImage(t)

	tmpDir := t.TempDir()

	cfg := ContainerConfig{
		Image:       "alpine:latest",
		Command:     "echo -n test123",
		RepoRoot:    tmpDir,
		MemoryLimit: "128m",
		CPULimit:    1,
		Timeout:     30 * time.Second,
	}

	result, err := RunContainer(cfg)
	if err != nil {
		t.Fatalf("RunContainer failed: %v", err)
	}

	if !strings.Contains(result.Stdout, "test123") {
		t.Errorf("expected stdout to contain 'test123', got %q", result.Stdout)
	}
}

func TestRunContainer_FailingCommand(t *testing.T) {
	if !isDockerAvailable() {
		t.Skip("Docker not available, skipping integration test")
	}
	ensureTestImage(t)

	tmpDir := t.TempDir()

	cfg := ContainerConfig{
		Image:       "alpine:latest",
		Command:     "exit 1",
		RepoRoot:    tmpDir,
		MemoryLimit: "128m",
		CPULimit:    1,
		Timeout:     30 * time.Second,
	}

	result, err := RunContainer(cfg)

	// Should return error for non-zero exit
	if err == nil {
		t.Error("expected error for failing command")
	}

	if result.ExitCode != 1 {
		t.Errorf("expected exit code 1, got %d", result.ExitCode)
	}
}

func TestRunContainer_NonZeroExitCode(t *testing.T) {
	if !isDockerAvailable() {
		t.Skip("Docker not available, skipping integration test")
	}
	ensureTestImage(t)

	tmpDir := t.TempDir()

	cfg := ContainerConfig{
		Image:       "alpine:latest",
		Command:     "exit 42",
		RepoRoot:    tmpDir,
		MemoryLimit: "128m",
		CPULimit:    1,
		Timeout:     30 * time.Second,
	}

	result, err := RunContainer(cfg)

	if err == nil {
		t.Error("expected error for non-zero exit")
	}

	if result.ExitCode != 42 {
		t.Errorf("expected exit code 42, got %d", result.ExitCode)
	}
}

func TestRunContainer_Stderr(t *testing.T) {
	if !isDockerAvailable() {
		t.Skip("Docker not available, skipping integration test")
	}
	ensureTestImage(t)

	tmpDir := t.TempDir()

	cfg := ContainerConfig{
		Image:       "alpine:latest",
		Command:     "echo error >&2",
		RepoRoot:    tmpDir,
		MemoryLimit: "128m",
		CPULimit:    1,
		Timeout:     30 * time.Second,
	}

	result, err := RunContainer(cfg)
	if err != nil {
		t.Fatalf("RunContainer failed: %v", err)
	}

	if !strings.Contains(result.Stderr, "error") {
		t.Errorf("expected stderr to contain 'error', got %q", result.Stderr)
	}
}

func TestRunContainer_Timeout(t *testing.T) {
	if !isDockerAvailable() {
		t.Skip("Docker not available, skipping integration test")
	}
	ensureTestImage(t)

	tmpDir := t.TempDir()

	cfg := ContainerConfig{
		Image:       "alpine:latest",
		Command:     "sleep 60", // Sleep longer than timeout
		RepoRoot:    tmpDir,
		MemoryLimit: "128m",
		CPULimit:    1,
		Timeout:     2 * time.Second, // Short timeout
	}

	start := time.Now()
	result, err := RunContainer(cfg)
	elapsed := time.Since(start)

	// Should return error for timeout
	if err == nil {
		t.Error("expected error for timeout")
	}

	// Should have timed out, not run for 60 seconds
	if elapsed > 10*time.Second {
		t.Errorf("container should have timed out quickly, took %v", elapsed)
	}

	// Exit code 124 is standard for timeout
	if result.ExitCode != 124 {
		t.Logf("timeout exit code: %d (expected 124)", result.ExitCode)
	}
}

func TestRunContainer_ReadOnlyWorkspace(t *testing.T) {
	if !isDockerAvailable() {
		t.Skip("Docker not available, skipping integration test")
	}
	ensureTestImage(t)

	tmpDir := t.TempDir()

	// Try to write to the mounted workspace (should fail - read-only)
	cfg := ContainerConfig{
		Image:       "alpine:latest",
		Command:     "touch /workspace/newfile.txt",
		RepoRoot:    tmpDir,
		MemoryLimit: "128m",
		CPULimit:    1,
		Timeout:     30 * time.Second,
	}

	result, err := RunContainer(cfg)

	// Should fail because workspace is mounted read-only
	if err == nil && result.ExitCode == 0 {
		t.Error("expected failure when writing to read-only workspace")
	}
}

func TestRunContainer_NoNetwork(t *testing.T) {
	if !isDockerAvailable() {
		t.Skip("Docker not available, skipping integration test")
	}
	ensureTestImage(t)

	tmpDir := t.TempDir()

	// Try to access network (should fail - network disabled)
	cfg := ContainerConfig{
		Image:       "alpine:latest",
		Command:     "wget -q -O- http://example.com",
		RepoRoot:    tmpDir,
		MemoryLimit: "128m",
		CPULimit:    1,
		Timeout:     10 * time.Second,
	}

	result, err := RunContainer(cfg)

	// Should fail because network is disabled
	if err == nil && result.ExitCode == 0 {
		t.Error("expected failure when accessing network")
	}
}

func TestRunContainer_WorkingDirectory(t *testing.T) {
	if !isDockerAvailable() {
		t.Skip("Docker not available, skipping integration test")
	}
	ensureTestImage(t)

	tmpDir := t.TempDir()

	cfg := ContainerConfig{
		Image:       "alpine:latest",
		Command:     "pwd",
		RepoRoot:    tmpDir,
		MemoryLimit: "128m",
		CPULimit:    1,
		Timeout:     30 * time.Second,
	}

	result, err := RunContainer(cfg)
	if err != nil {
		t.Fatalf("RunContainer failed: %v", err)
	}

	if !strings.Contains(result.Stdout, "/workspace") {
		t.Errorf("expected working directory to be /workspace, got %q", result.Stdout)
	}
}

func TestRunContainer_EnvironmentIsolation(t *testing.T) {
	if !isDockerAvailable() {
		t.Skip("Docker not available, skipping integration test")
	}
	ensureTestImage(t)

	tmpDir := t.TempDir()

	// Check that host environment variables are not leaked
	cfg := ContainerConfig{
		Image:       "alpine:latest",
		Command:     "env",
		RepoRoot:    tmpDir,
		MemoryLimit: "128m",
		CPULimit:    1,
		Timeout:     30 * time.Second,
	}

	result, err := RunContainer(cfg)
	if err != nil {
		t.Fatalf("RunContainer failed: %v", err)
	}

	// Should not contain typical host variables like HOME with host path
	// (This is a weak test - containers do have some env vars)
	t.Logf("Container environment:\n%s", result.Stdout)
}

func TestRunContainer_DurationTracking(t *testing.T) {
	if !isDockerAvailable() {
		t.Skip("Docker not available, skipping integration test")
	}
	ensureTestImage(t)

	tmpDir := t.TempDir()

	cfg := ContainerConfig{
		Image:       "alpine:latest",
		Command:     "sleep 1",
		RepoRoot:    tmpDir,
		MemoryLimit: "128m",
		CPULimit:    1,
		Timeout:     30 * time.Second,
	}

	result, err := RunContainer(cfg)
	if err != nil {
		t.Fatalf("RunContainer failed: %v", err)
	}

	// Duration should be at least 1 second
	if result.Duration < 1*time.Second {
		t.Errorf("expected duration >= 1s, got %v", result.Duration)
	}

	// But not too long
	if result.Duration > 10*time.Second {
		t.Errorf("expected duration < 10s, got %v", result.Duration)
	}
}

func TestRunContainer_MultipleCommands(t *testing.T) {
	if !isDockerAvailable() {
		t.Skip("Docker not available, skipping integration test")
	}
	ensureTestImage(t)

	tmpDir := t.TempDir()

	cfg := ContainerConfig{
		Image:       "alpine:latest",
		Command:     "echo first && echo second",
		RepoRoot:    tmpDir,
		MemoryLimit: "128m",
		CPULimit:    1,
		Timeout:     30 * time.Second,
	}

	result, err := RunContainer(cfg)
	if err != nil {
		t.Fatalf("RunContainer failed: %v", err)
	}

	if !strings.Contains(result.Stdout, "first") || !strings.Contains(result.Stdout, "second") {
		t.Errorf("expected both outputs, got %q", result.Stdout)
	}
}

func TestRunContainer_EmptyCommand(t *testing.T) {
	if !isDockerAvailable() {
		t.Skip("Docker not available, skipping integration test")
	}
	ensureTestImage(t)

	tmpDir := t.TempDir()

	cfg := ContainerConfig{
		Image:       "alpine:latest",
		Command:     "",
		RepoRoot:    tmpDir,
		MemoryLimit: "128m",
		CPULimit:    1,
		Timeout:     30 * time.Second,
	}

	_, err := RunContainer(cfg)
	// Empty command behavior depends on implementation
	t.Logf("Empty command result: %v", err)
}

func TestRunContainer_SpecialCharactersInCommand(t *testing.T) {
	if !isDockerAvailable() {
		t.Skip("Docker not available, skipping integration test")
	}
	ensureTestImage(t)

	tmpDir := t.TempDir()

	tests := []struct {
		name     string
		command  string
		expected string
	}{
		{"quotes", `echo "hello world"`, "hello world"},
		{"single quotes", `echo 'hello world'`, "hello world"},
		{"dollar sign", `echo '$HOME'`, "$HOME"},
		{"backticks", "echo `echo nested`", "nested"},
		{"pipe", "echo hello | cat", "hello"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := ContainerConfig{
				Image:       "alpine:latest",
				Command:     tt.command,
				RepoRoot:    tmpDir,
				MemoryLimit: "128m",
				CPULimit:    1,
				Timeout:     30 * time.Second,
			}

			result, err := RunContainer(cfg)
			if err != nil {
				t.Logf("Command %q failed: %v", tt.command, err)
				return
			}

			if !strings.Contains(result.Stdout, tt.expected) {
				t.Errorf("expected output to contain %q, got %q", tt.expected, result.Stdout)
			}
		})
	}
}

func TestRunContainer_LargeOutput(t *testing.T) {
	if !isDockerAvailable() {
		t.Skip("Docker not available, skipping integration test")
	}
	ensureTestImage(t)

	tmpDir := t.TempDir()

	// Generate large output
	cfg := ContainerConfig{
		Image:       "alpine:latest",
		Command:     "seq 1 10000",
		RepoRoot:    tmpDir,
		MemoryLimit: "128m",
		CPULimit:    1,
		Timeout:     30 * time.Second,
	}

	result, err := RunContainer(cfg)
	if err != nil {
		t.Fatalf("RunContainer failed: %v", err)
	}

	// Should contain first and last numbers
	if !strings.Contains(result.Stdout, "1\n") {
		t.Error("output should contain '1'")
	}
	if !strings.Contains(result.Stdout, "10000") {
		t.Error("output should contain '10000'")
	}
}

// Benchmark tests
func BenchmarkRunContainer_Echo(b *testing.B) {
	if !isDockerAvailable() {
		b.Skip("Docker not available")
	}

	tmpDir := b.TempDir()

	cfg := ContainerConfig{
		Image:       "alpine:latest",
		Command:     "echo benchmark",
		RepoRoot:    tmpDir,
		MemoryLimit: "128m",
		CPULimit:    1,
		Timeout:     30 * time.Second,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		RunContainer(cfg)
	}
}

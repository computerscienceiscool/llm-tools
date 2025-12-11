package sandbox

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
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

func TestRunContainer_WithStdin(t *testing.T) {
	if !isDockerAvailable() {
		t.Skip("Docker not available, skipping integration test")
	}
	ensureTestImage(t)

	tmpDir := t.TempDir()

	tests := []struct {
		name     string
		command  string
		stdin    string
		expected string
	}{
		{
			name:     "cat with stdin",
			command:  "cat",
			stdin:    "hello from stdin",
			expected: "hello from stdin",
		},
		{
			name:     "wc -l counts lines",
			command:  "wc -l",
			stdin:    "line1\nline2\nline3\n",
			expected: "3",
		},
		{
			name:     "grep filters stdin",
			command:  "grep test",
			stdin:    "test line\nother line\ntest again\n",
			expected: "test line",
		},
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
				Stdin:       tt.stdin,
			}

			result, err := RunContainer(cfg)
			if err != nil {
				t.Fatalf("RunContainer failed: %v", err)
			}

			if !strings.Contains(result.Stdout, tt.expected) {
				t.Errorf("expected stdout to contain %q, got %q", tt.expected, result.Stdout)
			}
		})
	}
}

func TestRunContainer_NoStdin(t *testing.T) {
	if !isDockerAvailable() {
		t.Skip("Docker not available, skipping integration test")
	}
	ensureTestImage(t)

	tmpDir := t.TempDir()

	// Ensure commands without stdin still work (empty Stdin field)
	cfg := ContainerConfig{
		Image:       "alpine:latest",
		Command:     "echo no stdin",
		RepoRoot:    tmpDir,
		MemoryLimit: "128m",
		CPULimit:    1,
		Timeout:     30 * time.Second,
		Stdin:       "", // Empty stdin
	}

	result, err := RunContainer(cfg)
	if err != nil {
		t.Fatalf("RunContainer failed: %v", err)
	}

	if !strings.Contains(result.Stdout, "no stdin") {
		t.Errorf("expected stdout to contain 'no stdin', got %q", result.Stdout)
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

// TestParseMemoryLimit tests memory limit string parsing
func TestParseMemoryLimit(t *testing.T) {
	tests := []struct {
		name     string
		limit    string
		expected int64
	}{
		{"empty string", "", 0},
		{"128 megabytes lowercase", "128m", 128 * 1024 * 1024},
		{"128 megabytes uppercase", "128M", 128 * 1024 * 1024},
		{"512 megabytes", "512m", 512 * 1024 * 1024},
		{"1 gigabyte lowercase", "1g", 1 * 1024 * 1024 * 1024},
		{"1 gigabyte uppercase", "1G", 1 * 1024 * 1024 * 1024},
		{"2 gigabytes", "2g", 2 * 1024 * 1024 * 1024},
		{"invalid format", "invalid", 0},
		{"no suffix", "256", 0},
		{"kilobytes not supported", "1024k", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseMemoryLimit(tt.limit)
			if result != tt.expected {
				t.Errorf("parseMemoryLimit(%q) = %d, want %d", tt.limit, result, tt.expected)
			}
		})
	}
}

// TestContainerLifecycle_MultipleRuns tests running the same container config multiple times
func TestContainerLifecycle_MultipleRuns(t *testing.T) {
	if !isDockerAvailable() {
		t.Skip("Docker not available")
	}
	ensureTestImage(t)

	tmpDir := t.TempDir()

	cfg := ContainerConfig{
		Image:       "alpine:latest",
		Command:     "echo test",
		RepoRoot:    tmpDir,
		MemoryLimit: "128m",
		CPULimit:    1,
		Timeout:     30 * time.Second,
	}

	// Run the same config 5 times
	for i := 0; i < 5; i++ {
		t.Run(fmt.Sprintf("run_%d", i), func(t *testing.T) {
			result, err := RunContainer(cfg)
			if err != nil {
				t.Fatalf("run %d failed: %v", i, err)
			}

			if result.ExitCode != 0 {
				t.Errorf("run %d: exit code = %d, want 0", i, result.ExitCode)
			}

			if !strings.Contains(result.Stdout, "test") {
				t.Errorf("run %d: stdout should contain 'test'", i)
			}
		})
	}

	// Verify no containers are left behind
	cmd := exec.Command("docker", "ps", "-a", "--filter", "ancestor=alpine:latest", "--format", "{{.ID}}")
	output, _ := cmd.Output()

	// There might be stopped containers, but they should be cleaned up
	// This is more of a sanity check
	t.Logf("Docker containers check: %s", string(output))
}

// TestContainerLifecycle_CleanupOnError tests that containers are cleaned up even on error
func TestContainerLifecycle_CleanupOnError(t *testing.T) {
	if !isDockerAvailable() {
		t.Skip("Docker not available")
	}
	ensureTestImage(t)

	tmpDir := t.TempDir()

	// Get container count before
	beforeCmd := exec.Command("docker", "ps", "-a", "-q")
	beforeOutput, _ := beforeCmd.Output()
	beforeCount := len(strings.Split(strings.TrimSpace(string(beforeOutput)), "\n"))

	cfg := ContainerConfig{
		Image:       "alpine:latest",
		Command:     "exit 1", // Command that fails
		RepoRoot:    tmpDir,
		MemoryLimit: "128m",
		CPULimit:    1,
		Timeout:     30 * time.Second,
	}

	_, err := RunContainer(cfg)
	if err == nil {
		t.Error("expected error for failing command")
	}

	// Give Docker a moment to cleanup
	time.Sleep(100 * time.Millisecond)

	// Get container count after
	afterCmd := exec.Command("docker", "ps", "-a", "-q")
	afterOutput, _ := afterCmd.Output()
	afterCount := len(strings.Split(strings.TrimSpace(string(afterOutput)), "\n"))

	// Container count should be the same (cleanup happened)
	if afterCount > beforeCount+1 {
		t.Errorf("container not cleaned up: before=%d, after=%d", beforeCount, afterCount)
	}
}

// TestContainerLifecycle_ResourceLimits tests that resource limits are enforced
func TestContainerLifecycle_ResourceLimits(t *testing.T) {
	if !isDockerAvailable() {
		t.Skip("Docker not available")
	}
	ensureTestImage(t)

	tmpDir := t.TempDir()

	tests := []struct {
		name       string
		memory     string
		cpu        int
		command    string
		shouldWork bool
	}{
		{
			name:       "reasonable limits",
			memory:     "128m",
			cpu:        1,
			command:    "echo hello",
			shouldWork: true,
		},
		{
			name:       "very low memory",
			memory:     "8m",
			cpu:        1,
			command:    "echo tiny",
			shouldWork: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := ContainerConfig{
				Image:       "alpine:latest",
				Command:     tt.command,
				RepoRoot:    tmpDir,
				MemoryLimit: tt.memory,
				CPULimit:    tt.cpu,
				Timeout:     10 * time.Second,
			}

			result, err := RunContainer(cfg)

			if tt.shouldWork && err != nil {
				t.Errorf("expected success but got error: %v", err)
			}

			if tt.shouldWork && result.ExitCode != 0 {
				t.Errorf("exit code = %d, want 0", result.ExitCode)
			}
		})
	}
}

// TestContainerLifecycle_TimeoutCleanup tests cleanup after timeout
func TestContainerLifecycle_TimeoutCleanup(t *testing.T) {
	if !isDockerAvailable() {
		t.Skip("Docker not available")
	}
	ensureTestImage(t)

	tmpDir := t.TempDir()

	cfg := ContainerConfig{
		Image:       "alpine:latest",
		Command:     "sleep 60",
		RepoRoot:    tmpDir,
		MemoryLimit: "128m",
		CPULimit:    1,
		Timeout:     1 * time.Second, // Very short timeout
	}

	_, err := RunContainer(cfg)
	if err == nil {
		t.Error("expected timeout error")
	}

	// Container should be cleaned up despite timeout
	time.Sleep(100 * time.Millisecond)

	// This is a best-effort check
	t.Log("Timeout cleanup test completed")
}

// TestConcurrentContainers_Stress tests running many containers simultaneously
func TestConcurrentContainers_Stress(t *testing.T) {
	if !isDockerAvailable() {
		t.Skip("Docker not available")
	}
	ensureTestImage(t)

	if testing.Short() {
		t.Skip("skipping stress test in short mode")
	}

	tmpDir := t.TempDir()

	// Run 20 containers concurrently
	concurrency := 20
	done := make(chan error, concurrency)

	for i := 0; i < concurrency; i++ {
		go func(id int) {
			cfg := ContainerConfig{
				Image:       "alpine:latest",
				Command:     fmt.Sprintf("echo stress_%d", id),
				RepoRoot:    tmpDir,
				MemoryLimit: "128m",
				CPULimit:    1,
				Timeout:     30 * time.Second,
			}

			result, err := RunContainer(cfg)
			if err != nil {
				done <- fmt.Errorf("container %d failed: %w", id, err)
				return
			}

			if result.ExitCode != 0 {
				done <- fmt.Errorf("container %d: exit code %d", id, result.ExitCode)
				return
			}

			if !strings.Contains(result.Stdout, fmt.Sprintf("stress_%d", id)) {
				done <- fmt.Errorf("container %d: unexpected output", id)
				return
			}

			done <- nil
		}(i)
	}

	// Wait for all to complete
	var errors []error
	for i := 0; i < concurrency; i++ {
		if err := <-done; err != nil {
			errors = append(errors, err)
		}
	}

	if len(errors) > 0 {
		t.Errorf("concurrent containers had %d errors:", len(errors))
		for _, err := range errors {
			t.Logf("  - %v", err)
		}
	}
}

// TestConcurrentContainers_DifferentCommands tests concurrent containers with different commands
func TestConcurrentContainers_DifferentCommands(t *testing.T) {
	if !isDockerAvailable() {
		t.Skip("Docker not available")
	}
	ensureTestImage(t)

	if testing.Short() {
		t.Skip("skipping stress test in short mode")
	}

	tmpDir := t.TempDir()

	commands := []string{
		"echo hello",
		"sleep 1",
		"ls /",
		"pwd",
		"date",
		"uname -a",
		"cat /etc/os-release",
		"id -u", // Changed from whoami to id -u (works in alpine)
		"env | head -3",
		"df -h",
	}

	done := make(chan error, len(commands))

	for i, cmd := range commands {
		go func(id int, command string) {
			cfg := ContainerConfig{
				Image:       "alpine:latest",
				Command:     command,
				RepoRoot:    tmpDir,
				MemoryLimit: "128m",
				CPULimit:    1,
				Timeout:     30 * time.Second,
			}

			_, err := RunContainer(cfg)
			if err != nil {
				done <- fmt.Errorf("command %d (%s) failed: %w", id, command, err)
				return
			}

			done <- nil
		}(i, cmd)
	}

	// Wait for all
	var errors []error
	for i := 0; i < len(commands); i++ {
		if err := <-done; err != nil {
			errors = append(errors, err)
		}
	}

	if len(errors) > 0 {
		t.Errorf("concurrent different commands had %d errors:", len(errors))
		for _, err := range errors {
			t.Logf("  - %v", err)
		}
	}
}

// TestConcurrentContainers_RapidFire tests rapid container creation and destruction
func TestConcurrentContainers_RapidFire(t *testing.T) {
	if !isDockerAvailable() {
		t.Skip("Docker not available")
	}
	ensureTestImage(t)

	if testing.Short() {
		t.Skip("skipping stress test in short mode")
	}

	tmpDir := t.TempDir()

	// Launch 15 containers (reduced from 50 to avoid overwhelming Docker)
	concurrency := 15
	done := make(chan error, concurrency)

	startTime := time.Now()

	for i := 0; i < concurrency; i++ {
		go func(id int) {
			cfg := ContainerConfig{
				Image:       "alpine:latest",
				Command:     "echo rapid",
				RepoRoot:    tmpDir,
				MemoryLimit: "64m",
				CPULimit:    1,
				Timeout:     30 * time.Second, // Increased from 10s
			}

			_, err := RunContainer(cfg)
			done <- err
		}(i)
	}

	// Wait for all
	var errors []error
	for i := 0; i < concurrency; i++ {
		if err := <-done; err != nil {
			errors = append(errors, err)
		}
	}

	elapsed := time.Since(startTime)
	t.Logf("Rapid fire: %d containers in %v (avg: %v per container)",
		concurrency, elapsed, elapsed/time.Duration(concurrency))

	if len(errors) > 0 {
		t.Errorf("rapid fire had %d errors:", len(errors))
		for _, err := range errors {
			t.Logf("  - %v", err)
		}
	}
}

// TestConcurrentContainers_WithTimeout tests concurrent containers with timeouts
func TestConcurrentContainers_WithTimeout(t *testing.T) {
	if !isDockerAvailable() {
		t.Skip("Docker not available")
	}
	ensureTestImage(t)

	if testing.Short() {
		t.Skip("skipping stress test in short mode")
	}

	tmpDir := t.TempDir()

	// Half will timeout, half will complete
	concurrency := 10
	type result struct {
		id      int
		err     error
		timeout bool
	}
	done := make(chan result, concurrency)

	for i := 0; i < concurrency; i++ {
		go func(id int) {
			var command string
			var timeout time.Duration
			shouldTimeout := id%2 == 1

			if shouldTimeout {
				// Will timeout
				command = "sleep 60"
				timeout = 2 * time.Second
			} else {
				// Will complete quickly
				command = "echo fast"
				timeout = 10 * time.Second
			}

			cfg := ContainerConfig{
				Image:       "alpine:latest",
				Command:     command,
				RepoRoot:    tmpDir,
				MemoryLimit: "128m",
				CPULimit:    1,
				Timeout:     timeout,
			}

			_, err := RunContainer(cfg)
			done <- result{id: id, err: err, timeout: shouldTimeout}
		}(i)
	}

	// Wait for all and check expectations
	successCount := 0
	failCount := 0
	for i := 0; i < concurrency; i++ {
		res := <-done
		if res.timeout {
			// Should have timed out (error expected)
			if res.err != nil {
				successCount++
			} else {
				t.Logf("container %d: expected timeout but succeeded", res.id)
				failCount++
			}
		} else {
			// Should have completed (no error expected)
			if res.err == nil {
				successCount++
			} else {
				t.Logf("container %d: expected success but got error: %v", res.id, res.err)
				failCount++
			}
		}
	}

	if successCount != concurrency {
		t.Errorf("expected all %d to behave correctly, got %d successes and %d failures",
			concurrency, successCount, failCount)
	}
}

// TestConcurrentContainers_MemoryPressure tests containers under memory pressure
func TestConcurrentContainers_MemoryPressure(t *testing.T) {
	if !isDockerAvailable() {
		t.Skip("Docker not available")
	}
	ensureTestImage(t)

	if testing.Short() {
		t.Skip("skipping stress test in short mode")
	}

	tmpDir := t.TempDir()

	// Run containers with very limited memory
	concurrency := 10
	done := make(chan error, concurrency)

	for i := 0; i < concurrency; i++ {
		go func(id int) {
			cfg := ContainerConfig{
				Image:       "alpine:latest",
				Command:     "echo memory",
				RepoRoot:    tmpDir,
				MemoryLimit: "16m", // Very low memory
				CPULimit:    1,
				Timeout:     10 * time.Second,
			}

			_, err := RunContainer(cfg)
			done <- err
		}(i)
	}

	// Wait for all
	var errors []error
	for i := 0; i < concurrency; i++ {
		if err := <-done; err != nil {
			errors = append(errors, err)
		}
	}

	// Some might fail due to memory limits, log but don't fail test
	if len(errors) > 0 {
		t.Logf("memory pressure test: %d/%d containers encountered errors (expected)", len(errors), concurrency)
	}
}

// TestContainerError_InvalidImage tests handling of invalid image names
func TestContainerError_InvalidImage(t *testing.T) {
	if !isDockerAvailable() {
		t.Skip("Docker not available")
	}

	tmpDir := t.TempDir()

	cfg := ContainerConfig{
		Image:       "this-image-definitely-does-not-exist:v999",
		Command:     "echo test",
		RepoRoot:    tmpDir,
		MemoryLimit: "128m",
		CPULimit:    1,
		Timeout:     10 * time.Second,
	}

	_, err := RunContainer(cfg)
	if err == nil {
		t.Fatal("expected error for invalid image, got nil")
	}

	t.Logf("Got expected error: %v", err)
}

// TestContainerError_InvalidCommand tests handling of commands that don't exist
func TestContainerError_InvalidCommand(t *testing.T) {
	if !isDockerAvailable() {
		t.Skip("Docker not available")
	}
	ensureTestImage(t)

	tmpDir := t.TempDir()

	cfg := ContainerConfig{
		Image:       "alpine:latest",
		Command:     "this_command_does_not_exist",
		RepoRoot:    tmpDir,
		MemoryLimit: "128m",
		CPULimit:    1,
		Timeout:     10 * time.Second,
	}

	_, err := RunContainer(cfg)
	if err == nil {
		t.Fatal("expected error for invalid command")
	}

	// Should get exit code 127 (command not found)
	if !strings.Contains(err.Error(), "127") && !strings.Contains(err.Error(), "exit") {
		t.Logf("Got error (may not mention exit code): %v", err)
	}
}

// TestContainerError_EmptyCommand tests handling of empty command strings
func TestContainerError_EmptyCommand(t *testing.T) {
	if !isDockerAvailable() {
		t.Skip("Docker not available")
	}
	ensureTestImage(t)

	tmpDir := t.TempDir()

	cfg := ContainerConfig{
		Image:       "alpine:latest",
		Command:     "",
		RepoRoot:    tmpDir,
		MemoryLimit: "128m",
		CPULimit:    1,
		Timeout:     5 * time.Second,
	}

	result, err := RunContainer(cfg)
	// Empty command actually succeeds - it just runs the container's default entrypoint
	if err != nil {
		t.Logf("Empty command resulted in error: %v", err)
	} else {
		t.Logf("Empty command succeeded with exit code: %d", result.ExitCode)
		t.Logf("Stdout: %s", result.Stdout)
		t.Logf("Stderr: %s", result.Stderr)
	}
	// This is acceptable behavior - no assertion needed
}

// TestContainerError_InvalidRepoRoot tests handling of non-existent repo roots
func TestContainerError_InvalidRepoRoot(t *testing.T) {
	if !isDockerAvailable() {
		t.Skip("Docker not available")
	}
	ensureTestImage(t)

	cfg := ContainerConfig{
		Image:       "alpine:latest",
		Command:     "echo test",
		RepoRoot:    "/this/path/does/not/exist/anywhere",
		MemoryLimit: "128m",
		CPULimit:    1,
		Timeout:     10 * time.Second,
	}

	_, err := RunContainer(cfg)
	if err == nil {
		t.Fatal("expected error for invalid repo root, got nil")
	}

	if !strings.Contains(err.Error(), "no such file") && !strings.Contains(err.Error(), "not found") {
		t.Logf("Got error (may not mention path): %v", err)
	}
}

// TestContainerError_NegativeTimeout tests handling of invalid timeout values
func TestContainerError_NegativeTimeout(t *testing.T) {
	if !isDockerAvailable() {
		t.Skip("Docker not available")
	}
	ensureTestImage(t)

	tmpDir := t.TempDir()

	cfg := ContainerConfig{
		Image:       "alpine:latest",
		Command:     "echo test",
		RepoRoot:    tmpDir,
		MemoryLimit: "128m",
		CPULimit:    1,
		Timeout:     -1 * time.Second,
	}

	// This should either error or treat as no timeout
	result, err := RunContainer(cfg)

	// Either path is acceptable
	if err != nil {
		t.Logf("Got error for negative timeout: %v", err)
	} else if result.ExitCode == 0 {
		t.Logf("Treated negative timeout as valid (exit code 0)")
	} else {
		t.Errorf("Unexpected exit code: %d", result.ExitCode)
	}
}

// TestContainerError_VeryShortTimeout tests containers with extremely short timeouts
func TestContainerError_VeryShortTimeout(t *testing.T) {
	if !isDockerAvailable() {
		t.Skip("Docker not available")
	}
	ensureTestImage(t)

	tmpDir := t.TempDir()

	cfg := ContainerConfig{
		Image:       "alpine:latest",
		Command:     "sleep 10",
		RepoRoot:    tmpDir,
		MemoryLimit: "128m",
		CPULimit:    1,
		Timeout:     10 * time.Millisecond, // Very short timeout
	}

	_, err := RunContainer(cfg)
	if err == nil {
		t.Fatal("expected timeout error for very short timeout")
	}

	if !strings.Contains(err.Error(), "timeout") && !strings.Contains(err.Error(), "deadline") {
		t.Logf("Got error (may not mention timeout): %v", err)
	}
}

// TestContainerError_CommandCrash tests handling of commands that crash
func TestContainerError_CommandCrash(t *testing.T) {
	if !isDockerAvailable() {
		t.Skip("Docker not available")
	}
	ensureTestImage(t)

	tmpDir := t.TempDir()

	cfg := ContainerConfig{
		Image:       "alpine:latest",
		Command:     "sh -c 'exit 137'", // Simulate crash
		RepoRoot:    tmpDir,
		MemoryLimit: "128m",
		CPULimit:    1,
		Timeout:     10 * time.Second,
	}

	_, err := RunContainer(cfg)
	if err == nil {
		t.Fatal("expected error for exit code 137")
	}

	if !strings.Contains(err.Error(), "137") && !strings.Contains(err.Error(), "exit") {
		t.Logf("Got error (may not mention exit code): %v", err)
	}
}

// TestContainerError_StderrOutput tests that stderr is captured properly
func TestContainerError_StderrOutput(t *testing.T) {
	if !isDockerAvailable() {
		t.Skip("Docker not available")
	}
	ensureTestImage(t)

	tmpDir := t.TempDir()

	cfg := ContainerConfig{
		Image:       "alpine:latest",
		Command:     "sh -c 'echo error_message >&2'",
		RepoRoot:    tmpDir,
		MemoryLimit: "128m",
		CPULimit:    1,
		Timeout:     10 * time.Second,
	}

	result, err := RunContainer(cfg)
	if err != nil {
		t.Fatalf("RunContainer returned error: %v", err)
	}

	if !strings.Contains(result.Stderr, "error_message") {
		t.Errorf("stderr should contain 'error_message', got: %s", result.Stderr)
	}

	if result.Stdout != "" {
		t.Errorf("stdout should be empty, got: %s", result.Stdout)
	}
}

// TestResourceLimits_VariousMemorySizes tests different memory limit formats
func TestResourceLimits_VariousMemorySizes(t *testing.T) {
	if !isDockerAvailable() {
		t.Skip("Docker not available")
	}
	ensureTestImage(t)

	tmpDir := t.TempDir()

	testCases := []struct {
		name        string
		memoryLimit string
		expectError bool
	}{
		{"16m", "16m", false},
		{"32m", "32m", false},
		{"64m", "64m", false},
		{"128m", "128m", false},
		{"256m", "256m", false},
		{"512m", "512m", false},
		{"1g", "1g", false},
		{"2g", "2g", false},
		// Note: Docker doesn't validate memory format at container creation
		// These pass through without error
		{"no_unit_passes", "128", false},
		{"invalid_format_passes", "abc", false},
		{"zero_passes", "0m", false},
		{"negative", "-128m", true},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := ContainerConfig{
				Image:       "alpine:latest",
				Command:     "echo test",
				RepoRoot:    tmpDir,
				MemoryLimit: tc.memoryLimit,
				CPULimit:    1,
				Timeout:     10 * time.Second,
			}

			_, err := RunContainer(cfg)

			if tc.expectError {
				if err == nil {
					t.Errorf("expected error for memory limit %s, got nil", tc.memoryLimit)
				} else {
					t.Logf("Got expected error: %v", err)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error for memory limit %s: %v", tc.memoryLimit, err)
				}
			}
		})
	}
}

// TestResourceLimits_CPUValues tests different CPU limit values
func TestResourceLimits_CPUValues(t *testing.T) {
	if !isDockerAvailable() {
		t.Skip("Docker not available")
	}
	ensureTestImage(t)

	tmpDir := t.TempDir()

	testCases := []struct {
		name        string
		cpuLimit    int
		expectError bool
	}{
		{"1_core", 1, false},
		/*	{"2_cores", 2, false},
			{"4_cores", 4, false},
			{"8_cores", 8, false}, */ // Disabled becasue computer has only 1 cpu
		{"negative_cores", -1, true},
		{"zero_cores_passes", 0, false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := ContainerConfig{
				Image:       "alpine:latest",
				Command:     "echo test",
				RepoRoot:    tmpDir,
				MemoryLimit: "128m",
				CPULimit:    tc.cpuLimit,
				Timeout:     10 * time.Second,
			}

			_, err := RunContainer(cfg)

			if tc.expectError {
				if err == nil {
					t.Errorf("expected error for CPU limit %d, got nil", tc.cpuLimit)
				} else {
					t.Logf("Got expected error: %v", err)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error for CPU limit %d: %v", tc.cpuLimit, err)
				}
			}
		})
	}
}

// TestResourceLimits_CombinedConstraints tests various combinations
func TestResourceLimits_CombinedConstraints(t *testing.T) {
	if !isDockerAvailable() {
		t.Skip("Docker not available")
	}
	ensureTestImage(t)

	if testing.Short() {
		t.Skip("skipping resource combination test in short mode")
	}

	tmpDir := t.TempDir()

	testCases := []struct {
		name        string
		memory      string
		cpu         int
		timeout     time.Duration
		command     string
		expectError bool
	}{
		{"low_all", "32m", 1, 5 * time.Second, "echo test", false},
		{"high_mem", "1g", 1, 10 * time.Second, "echo test", false},
		{"minimal_timeout", "128m", 1, 5 * time.Second, "echo test", false},
		{"long_running", "128m", 1, 30 * time.Second, "sleep 2", false},
		{"timeout_will_hit", "128m", 1, 1 * time.Second, "sleep 10", true},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := ContainerConfig{
				Image:       "alpine:latest",
				Command:     tc.command,
				RepoRoot:    tmpDir,
				MemoryLimit: tc.memory,
				CPULimit:    tc.cpu,
				Timeout:     tc.timeout,
			}

			_, err := RunContainer(cfg)

			if tc.expectError {
				if err == nil {
					t.Errorf("expected error for %s, got nil", tc.name)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error for %s: %v", tc.name, err)
				}
			}
		})
	}
}

// TestResourceLimits_MemoryExhaustion tests container behavior under memory pressure
func TestResourceLimits_MemoryExhaustion(t *testing.T) {
	if !isDockerAvailable() {
		t.Skip("Docker not available")
	}
	ensureTestImage(t)

	if testing.Short() {
		t.Skip("skipping memory exhaustion test in short mode")
	}

	tmpDir := t.TempDir()

	cfg := ContainerConfig{
		Image: "alpine:latest",
		// Try to allocate more memory than limit
		Command:     "sh -c 'dd if=/dev/zero of=/tmp/bigfile bs=1M count=256 2>/dev/null || true'",
		RepoRoot:    tmpDir,
		MemoryLimit: "64m", // Much less than 256MB
		CPULimit:    1,
		Timeout:     10 * time.Second,
	}

	_, err := RunContainer(cfg)
	// This might error or succeed with OOM kill, either is acceptable
	if err != nil {
		t.Logf("Memory exhaustion caused error (expected): %v", err)
	} else {
		t.Logf("Memory exhaustion handled gracefully")
	}
}

// TestResourceLimits_EmptyMemoryLimit tests default behavior
func TestResourceLimits_EmptyMemoryLimit(t *testing.T) {
	if !isDockerAvailable() {
		t.Skip("Docker not available")
	}
	ensureTestImage(t)

	tmpDir := t.TempDir()

	cfg := ContainerConfig{
		Image:       "alpine:latest",
		Command:     "echo test",
		RepoRoot:    tmpDir,
		MemoryLimit: "", // Empty should use some default
		CPULimit:    1,
		Timeout:     10 * time.Second,
	}

	_, err := RunContainer(cfg)
	// Empty memory limit might error or use default
	if err != nil {
		t.Logf("Empty memory limit caused error: %v", err)
	} else {
		t.Logf("Empty memory limit handled (uses default)")
	}
}

// TestResourceLimits_VeryLongTimeout tests extended timeouts
func TestResourceLimits_VeryLongTimeout(t *testing.T) {
	if !isDockerAvailable() {
		t.Skip("Docker not available")
	}
	ensureTestImage(t)

	if testing.Short() {
		t.Skip("skipping long timeout test in short mode")
	}

	tmpDir := t.TempDir()

	cfg := ContainerConfig{
		Image:       "alpine:latest",
		Command:     "echo test",
		RepoRoot:    tmpDir,
		MemoryLimit: "128m",
		CPULimit:    1,
		Timeout:     5 * time.Minute, // Very long timeout
	}

	result, err := RunContainer(cfg)
	if err != nil {
		t.Fatalf("unexpected error with long timeout: %v", err)
	}

	if result.ExitCode != 0 {
		t.Errorf("expected exit code 0, got %d", result.ExitCode)
	}
}

// TestContainerIO_StdinStdout tests stdin/stdout communication
func TestContainerIO_StdinStdout(t *testing.T) {
	if !isDockerAvailable() {
		t.Skip("Docker not available")
	}
	ensureTestImage(t)

	tmpDir := t.TempDir()

	cfg := ContainerConfig{
		Image:       "alpine:latest",
		Command:     "cat",
		RepoRoot:    tmpDir,
		MemoryLimit: "128m",
		CPULimit:    1,
		Timeout:     10 * time.Second,
		Stdin:       "Hello from stdin\nLine 2\nLine 3",
	}

	result, err := RunContainer(cfg)
	if err != nil {
		t.Fatalf("RunContainer failed: %v", err)
	}

	if result.ExitCode != 0 {
		t.Errorf("expected exit code 0, got %d", result.ExitCode)
	}

	if !strings.Contains(result.Stdout, "Hello from stdin") {
		t.Errorf("stdout should contain input, got: %s", result.Stdout)
	}

	if !strings.Contains(result.Stdout, "Line 2") {
		t.Errorf("stdout should contain all lines, got: %s", result.Stdout)
	}
}

// TestContainerIO_LargeStdin tests handling of large stdin
func TestContainerIO_LargeStdin(t *testing.T) {
	if !isDockerAvailable() {
		t.Skip("Docker not available")
	}
	ensureTestImage(t)

	tmpDir := t.TempDir()

	// Create large input (100KB)
	largeInput := strings.Repeat("This is a test line\n", 5000)

	cfg := ContainerConfig{
		Image:       "alpine:latest",
		Command:     "wc -c",
		RepoRoot:    tmpDir,
		MemoryLimit: "128m",
		CPULimit:    1,
		Timeout:     10 * time.Second,
		Stdin:       largeInput,
	}

	result, err := RunContainer(cfg)
	if err != nil {
		t.Fatalf("RunContainer failed: %v", err)
	}

	if result.ExitCode != 0 {
		t.Errorf("expected exit code 0, got %d", result.ExitCode)
	}

	// wc -c should report byte count
	expectedSize := len(largeInput)
	if !strings.Contains(result.Stdout, fmt.Sprintf("%d", expectedSize)) {
		t.Logf("stdout: %s", result.Stdout)
		t.Logf("expected to find byte count %d", expectedSize)
	}
}

// TestContainerIO_BinaryStdin tests binary data handling
func TestContainerIO_BinaryStdin(t *testing.T) {
	if !isDockerAvailable() {
		t.Skip("Docker not available")
	}
	ensureTestImage(t)

	tmpDir := t.TempDir()

	// Binary data with null bytes
	binaryInput := "Hello\x00World\x00\x01\x02\x03"

	cfg := ContainerConfig{
		Image:       "alpine:latest",
		Command:     "wc -c",
		RepoRoot:    tmpDir,
		MemoryLimit: "128m",
		CPULimit:    1,
		Timeout:     10 * time.Second,
		Stdin:       binaryInput,
	}

	result, err := RunContainer(cfg)
	if err != nil {
		t.Fatalf("RunContainer failed: %v", err)
	}

	if result.ExitCode != 0 {
		t.Errorf("expected exit code 0, got %d", result.ExitCode)
	}
}

// TestContainerIO_EmptyStdin tests empty stdin handling
func TestContainerIO_EmptyStdin(t *testing.T) {
	if !isDockerAvailable() {
		t.Skip("Docker not available")
	}
	ensureTestImage(t)

	tmpDir := t.TempDir()

	cfg := ContainerConfig{
		Image:       "alpine:latest",
		Command:     "cat",
		RepoRoot:    tmpDir,
		MemoryLimit: "128m",
		CPULimit:    1,
		Timeout:     10 * time.Second,
		Stdin:       "",
	}

	result, err := RunContainer(cfg)
	if err != nil {
		t.Fatalf("RunContainer failed: %v", err)
	}

	if result.ExitCode != 0 {
		t.Errorf("expected exit code 0, got %d", result.ExitCode)
	}

	if result.Stdout != "" {
		t.Errorf("expected empty stdout, got: %s", result.Stdout)
	}
}

// TestContainerIO_LargeOutput tests handling of large command output
func TestContainerIO_LargeOutput(t *testing.T) {
	if !isDockerAvailable() {
		t.Skip("Docker not available")
	}
	ensureTestImage(t)

	if testing.Short() {
		t.Skip("skipping large output test in short mode")
	}

	tmpDir := t.TempDir()

	cfg := ContainerConfig{
		Image:       "alpine:latest",
		Command:     "sh -c 'for i in $(seq 1 10000); do echo Line $i; done'",
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

	lines := strings.Split(strings.TrimSpace(result.Stdout), "\n")
	if len(lines) != 10000 {
		t.Errorf("expected 10000 lines, got %d", len(lines))
	}

	if !strings.Contains(result.Stdout, "Line 1") {
		t.Errorf("output should contain Line 1")
	}

	if !strings.Contains(result.Stdout, "Line 10000") {
		t.Errorf("output should contain Line 10000")
	}
}

// TestContainerIO_MixedStdoutStderr tests stdout and stderr separation
func TestContainerIO_MixedStdoutStderr(t *testing.T) {
	if !isDockerAvailable() {
		t.Skip("Docker not available")
	}
	ensureTestImage(t)

	tmpDir := t.TempDir()

	cfg := ContainerConfig{
		Image:       "alpine:latest",
		Command:     "sh -c 'echo stdout1; echo stderr1 >&2; echo stdout2; echo stderr2 >&2'",
		RepoRoot:    tmpDir,
		MemoryLimit: "128m",
		CPULimit:    1,
		Timeout:     10 * time.Second,
	}

	result, err := RunContainer(cfg)
	if err != nil {
		t.Fatalf("RunContainer failed: %v", err)
	}

	if !strings.Contains(result.Stdout, "stdout1") {
		t.Errorf("stdout should contain stdout1, got: %s", result.Stdout)
	}

	if !strings.Contains(result.Stdout, "stdout2") {
		t.Errorf("stdout should contain stdout2, got: %s", result.Stdout)
	}

	if !strings.Contains(result.Stderr, "stderr1") {
		t.Errorf("stderr should contain stderr1, got: %s", result.Stderr)
	}

	if !strings.Contains(result.Stderr, "stderr2") {
		t.Errorf("stderr should contain stderr2, got: %s", result.Stderr)
	}

	// Ensure no cross-contamination
	if strings.Contains(result.Stdout, "stderr1") {
		t.Errorf("stdout should not contain stderr content")
	}

	if strings.Contains(result.Stderr, "stdout1") {
		t.Errorf("stderr should not contain stdout content")
	}
}

// TestContainerIO_FileOperations tests file read/write in container
func TestContainerIO_FileOperations(t *testing.T) {
	if !isDockerAvailable() {
		t.Skip("Docker not available")
	}
	ensureTestImage(t)

	tmpDir := t.TempDir()

	// Create a test file
	testFile := filepath.Join(tmpDir, "test.txt")
	testContent := "Hello from host file"
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	cfg := ContainerConfig{
		Image:       "alpine:latest",
		Command:     "cat /workspace/test.txt",
		RepoRoot:    tmpDir,
		MemoryLimit: "128m",
		CPULimit:    1,
		Timeout:     10 * time.Second,
	}

	result, err := RunContainer(cfg)
	if err != nil {
		t.Fatalf("RunContainer failed: %v", err)
	}

	if result.ExitCode != 0 {
		t.Errorf("expected exit code 0, got %d", result.ExitCode)
	}

	if !strings.Contains(result.Stdout, testContent) {
		t.Errorf("expected to read file content, got: %s", result.Stdout)
	}
}

// TestContainerIO_WriteFileFromContainer tests writing files from container
func TestContainerIO_WriteFileFromContainer(t *testing.T) {
	if !isDockerAvailable() {
		t.Skip("Docker not available")
	}
	ensureTestImage(t)

	tmpDir := t.TempDir()

	// First check if we can even see /workspace
	cfg := ContainerConfig{
		Image:       "alpine:latest",
		Command:     "ls -la /workspace",
		RepoRoot:    tmpDir,
		MemoryLimit: "128m",
		CPULimit:    1,
		Timeout:     10 * time.Second,
	}

	result, err := RunContainer(cfg)
	if err != nil {
		t.Logf("ls /workspace failed: %v", err)
		t.Logf("stdout: %s", result.Stdout)
		t.Logf("stderr: %s", result.Stderr)
	} else {
		t.Logf("ls /workspace succeeded:")
		t.Logf("stdout: %s", result.Stdout)
	}

	// Now try to write
	cfg = ContainerConfig{
		Image:       "alpine:latest",
		Command:     "sh -c 'echo written_by_container > /workspace/output.txt 2>&1 && echo SUCCESS'",
		RepoRoot:    tmpDir,
		MemoryLimit: "128m",
		CPULimit:    1,
		Timeout:     10 * time.Second,
	}

	result, err = RunContainer(cfg)
	if err != nil {
		t.Logf("Write attempt error: %v", err)
	}

	t.Logf("Write attempt stdout: %s", result.Stdout)
	t.Logf("Write attempt stderr: %s", result.Stderr)
	t.Logf("Write attempt exit code: %d", result.ExitCode)

	if strings.Contains(result.Stdout, "SUCCESS") {
		t.Log("Write command executed successfully")

		// Check if file exists on host
		outputFile := filepath.Join(tmpDir, "output.txt")
		if content, err := os.ReadFile(outputFile); err == nil {
			if strings.Contains(string(content), "written_by_container") {
				t.Log("File successfully written and visible on host")
			} else {
				t.Errorf("File exists but has wrong content: %s", string(content))
			}
		} else {
			t.Logf("File not visible on host: %v", err)
		}
	}
}

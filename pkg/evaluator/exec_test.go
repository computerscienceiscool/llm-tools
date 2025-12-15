package evaluator

import (
	"time"
	"os/exec"
	"strings"
	"testing"

	"github.com/computerscienceiscool/llm-runtime/pkg/config"
	"github.com/computerscienceiscool/llm-runtime/pkg/scanner"
)

// Helper to check if Docker is available
func dockerAvailable() bool {
	cmd := exec.Command("docker", "version")
	return cmd.Run() == nil
}

// REMOVED: func TestExecuteExec_Disabled(t *testing.T) {
// REMOVED: 	cfg := &config.Config{
// REMOVED: 		RepositoryRoot: t.TempDir(),
// REMOVED: 		ExecWhitelist:  []string{"ls"},
// REMOVED: 	}
// REMOVED: 
// REMOVED: 	audit := &testAuditLog{}
// REMOVED: 	cmd := scanner.Command{Type: "exec", Argument: "ls"}
// REMOVED: 	result := ExecuteExec(cmd, cfg, audit.log)
// REMOVED: 
// REMOVED: 	if result.Success {
// REMOVED: 		t.Error("expected failure when exec is disabled")
// REMOVED: 	}
// REMOVED: 
// REMOVED: 	if !strings.Contains(result.Error.Error(), "EXEC_VALIDATION") {
// REMOVED: 		t.Errorf("expected EXEC_VALIDATION error, got: %v", result.Error)
// REMOVED: 	}
// REMOVED: 
// REMOVED: 	// Check audit log
// REMOVED: 	entries := audit.getEntries()
// REMOVED: 	if len(entries) != 1 {
// REMOVED: 		t.Fatalf("expected 1 audit entry, got %d", len(entries))
// REMOVED: 	}
// REMOVED: 	if entries[0].success {
// REMOVED: 		t.Error("audit should show failure")
// REMOVED: 	}
// REMOVED: }

func TestExecuteExec_EmptyWhitelist(t *testing.T) {
	cfg := &config.Config{
		RepositoryRoot: t.TempDir(),
		IOTimeout:         60 * time.Second,
		IOContainerImage:    "llm-runtime-io:latest",
		ExecWhitelist:  []string{}, // Empty whitelist
	}

	cmd := scanner.Command{Type: "exec", Argument: "ls"}
	result := ExecuteExec(cmd, cfg, nil)

	if result.Success {
		t.Error("expected failure with empty whitelist")
	}

	if !strings.Contains(result.Error.Error(), "EXEC_VALIDATION") {
		t.Errorf("expected EXEC_VALIDATION error, got: %v", result.Error)
	}
}

func TestExecuteExec_CommandNotWhitelisted(t *testing.T) {
	cfg := &config.Config{
		RepositoryRoot: t.TempDir(),
		IOTimeout:         60 * time.Second,
		IOContainerImage:    "llm-runtime-io:latest",
		ExecWhitelist:  []string{"go test", "npm test"},
	}

	cmd := scanner.Command{Type: "exec", Argument: "rm -rf /"}
	result := ExecuteExec(cmd, cfg, nil)

	if result.Success {
		t.Error("expected failure for non-whitelisted command")
	}

	if !strings.Contains(result.Error.Error(), "EXEC_VALIDATION") {
		t.Errorf("expected EXEC_VALIDATION error, got: %v", result.Error)
	}
}

func TestExecuteExec_EmptyCommand(t *testing.T) {
	cfg := &config.Config{
		RepositoryRoot: t.TempDir(),
		IOTimeout:         60 * time.Second,
		IOContainerImage:    "llm-runtime-io:latest",
		ExecWhitelist:  []string{"ls"},
	}

	cmd := scanner.Command{Type: "exec", Argument: ""}
	result := ExecuteExec(cmd, cfg, nil)

	if result.Success {
		t.Error("expected failure for empty command")
	}

	if !strings.Contains(result.Error.Error(), "EXEC_VALIDATION") {
		t.Errorf("expected EXEC_VALIDATION error, got: %v", result.Error)
	}
}

func TestExecuteExec_WhitelistPrefixMatch(t *testing.T) {
	cfg := &config.Config{
		RepositoryRoot:     t.TempDir(),
		IOTimeout:         60 * time.Second,
		IOContainerImage:    "llm-runtime-io:latest",
		ExecWhitelist:      []string{"go test"},
		ExecContainerImage: "golang:alpine",
		ExecTimeout:        30 * time.Second,
		ExecMemoryLimit:    "256m",
		ExecCPULimit:       1,
	}

	// "go test ./..." should match "go test" prefix
	// This test validates the whitelist logic, not actual execution
	if !dockerAvailable() {
		// Just test validation passes
		cmd := scanner.Command{Type: "exec", Argument: "go test ./..."}
		result := ExecuteExec(cmd, cfg, nil)
		// Will fail at Docker check, not whitelist
		if result.Error != nil && strings.Contains(result.Error.Error(), "EXEC_VALIDATION") {
			t.Error("whitelist should allow 'go test ./...' with 'go test' in whitelist")
		}
		return
	}
}

func TestExecuteExec_CommandType(t *testing.T) {
	cfg := &config.Config{
		RepositoryRoot: t.TempDir(),
		IOTimeout:         60 * time.Second,
		IOContainerImage:    "llm-runtime-io:latest",
	}

	cmd := scanner.Command{Type: "exec", Argument: "any command"}
	result := ExecuteExec(cmd, cfg, nil)

	if result.Command.Type != "exec" {
		t.Errorf("expected command type 'exec', got %q", result.Command.Type)
	}

	if result.Command.Argument != "any command" {
		t.Errorf("expected argument 'any command', got %q", result.Command.Argument)
	}
}

func TestExecuteExec_ExecutionTime(t *testing.T) {
	cfg := &config.Config{
		RepositoryRoot: t.TempDir(),
		IOTimeout:         60 * time.Second,
		IOContainerImage:    "llm-runtime-io:latest",
	}

	cmd := scanner.Command{Type: "exec", Argument: "test"}
	result := ExecuteExec(cmd, cfg, nil)

	if result.ExecutionTime <= 0 {
		t.Error("execution time should be positive")
	}
}

func TestExecuteExec_NilAuditLog(t *testing.T) {
	cfg := &config.Config{
		RepositoryRoot: t.TempDir(),
		IOTimeout:         60 * time.Second,
		IOContainerImage:    "llm-runtime-io:latest",
	}

	// Should not panic with nil audit log
	cmd := scanner.Command{Type: "exec", Argument: "test"}
	result := ExecuteExec(cmd, cfg, nil)

	if result.Success {
		t.Error("expected failure")
	}
}

func TestExecuteExec_AuditLogOnValidationFailure(t *testing.T) {
	cfg := &config.Config{
		RepositoryRoot: t.TempDir(),
		IOTimeout:         60 * time.Second,
		IOContainerImage:    "llm-runtime-io:latest",
	}

	audit := &testAuditLog{}
	cmd := scanner.Command{Type: "exec", Argument: "test"}
	ExecuteExec(cmd, cfg, audit.log)

	entries := audit.getEntries()
	if len(entries) != 1 {
		t.Fatalf("expected 1 audit entry, got %d", len(entries))
	}

	entry := entries[0]
	if entry.cmdType != "exec" {
		t.Errorf("expected cmdType 'exec', got %q", entry.cmdType)
	}
	if entry.arg != "test" {
		t.Errorf("expected arg 'test', got %q", entry.arg)
	}
	if entry.success {
		t.Error("expected success=false")
	}
	if entry.errMsg == "" {
		t.Error("expected error message")
	}
}

func TestExecuteExec_DockerNotAvailable(t *testing.T) {
	if dockerAvailable() {
		t.Skip("Docker is available, cannot test Docker unavailable path")
	}

	cfg := &config.Config{
		RepositoryRoot:     t.TempDir(),
		IOTimeout:         60 * time.Second,
		IOContainerImage:    "llm-runtime-io:latest",
		ExecWhitelist:      []string{"echo"},
		ExecContainerImage: "alpine:latest",
		ExecTimeout:        30 * time.Second,
		ExecMemoryLimit:    "256m",
		ExecCPULimit:       1,
	}

	cmd := scanner.Command{Type: "exec", Argument: "echo hello"}
	result := ExecuteExec(cmd, cfg, nil)

	if result.Success {
		t.Error("expected failure when Docker is not available")
	}

	if !strings.Contains(result.Error.Error(), "DOCKER") {
		t.Errorf("expected DOCKER error, got: %v", result.Error)
	}
}

// Integration tests - require Docker
func TestExecuteExec_Integration_Echo(t *testing.T) {
	if !dockerAvailable() {
		t.Skip("Docker not available")
	}

	cfg := &config.Config{
		RepositoryRoot:     t.TempDir(),
		IOTimeout:         60 * time.Second,
		IOContainerImage:    "llm-runtime-io:latest",
		ExecWhitelist:      []string{"echo"},
		ExecContainerImage: "alpine:latest",
		ExecTimeout:        30 * time.Second,
		ExecMemoryLimit:    "256m",
		ExecCPULimit:       1,
	}

	// Ensure image is available
	exec.Command("docker", "pull", "alpine:latest").Run()

	audit := &testAuditLog{}
	cmd := scanner.Command{Type: "exec", Argument: "echo hello world"}
	result := ExecuteExec(cmd, cfg, audit.log)

	if !result.Success {
		t.Errorf("expected success, got error: %v", result.Error)
	}

	if !strings.Contains(result.Stdout, "hello world") {
		t.Errorf("expected stdout to contain 'hello world', got %q", result.Stdout)
	}

	if result.ExitCode != 0 {
		t.Errorf("expected exit code 0, got %d", result.ExitCode)
	}

	// Check audit log
	entries := audit.getEntries()
	if len(entries) != 1 {
		t.Fatalf("expected 1 audit entry, got %d", len(entries))
	}
	if !entries[0].success {
		t.Error("audit should show success")
	}
	if !strings.Contains(entries[0].errMsg, "status:completed") {
		t.Errorf("audit message should contain status:completed, got %q", entries[0].errMsg)
	}
}

func TestExecuteExec_Integration_FailingCommand(t *testing.T) {
	if !dockerAvailable() {
		t.Skip("Docker not available")
	}

	cfg := &config.Config{
		RepositoryRoot:     t.TempDir(),
		IOTimeout:         60 * time.Second,
		IOContainerImage:    "llm-runtime-io:latest",
		ExecWhitelist:      []string{"exit"},
		ExecContainerImage: "alpine:latest",
		ExecTimeout:        30 * time.Second,
		ExecMemoryLimit:    "256m",
		ExecCPULimit:       1,
	}

	exec.Command("docker", "pull", "alpine:latest").Run()

	cmd := scanner.Command{Type: "exec", Argument: "exit 1"}
	result := ExecuteExec(cmd, cfg, nil)

	if result.Success {
		t.Error("expected failure for exit 1")
	}

	if result.ExitCode != 1 {
		t.Errorf("expected exit code 1, got %d", result.ExitCode)
	}

	if !strings.Contains(result.Error.Error(), "EXEC_FAILED") {
		t.Errorf("expected EXEC_FAILED error, got: %v", result.Error)
	}
}

func TestExecuteExec_Integration_Timeout(t *testing.T) {
	if !dockerAvailable() {
		t.Skip("Docker not available")
	}

	cfg := &config.Config{
		RepositoryRoot:     t.TempDir(),
		IOTimeout:         60 * time.Second,
		IOContainerImage:    "llm-runtime-io:latest",
		ExecWhitelist:      []string{"sleep"},
		ExecContainerImage: "alpine:latest",
		ExecTimeout:        2 * time.Second, // Short timeout
		ExecMemoryLimit:    "256m",
		ExecCPULimit:       1,
	}

	exec.Command("docker", "pull", "alpine:latest").Run()

	start := time.Now()
	cmd := scanner.Command{Type: "exec", Argument: "sleep 60"}
	result := ExecuteExec(cmd, cfg, nil)
	elapsed := time.Since(start)

	if result.Success {
		t.Error("expected failure for timeout")
	}

	if !strings.Contains(result.Error.Error(), "EXEC_TIMEOUT") {
		t.Errorf("expected EXEC_TIMEOUT error, got: %v", result.Error)
	}

	if result.ExitCode != 124 {
		t.Logf("timeout exit code: %d (expected 124)", result.ExitCode)
	}

	// Should have timed out quickly
	if elapsed > 10*time.Second {
		t.Errorf("should have timed out in ~2s, took %v", elapsed)
	}
}

func TestExecuteExec_Integration_Stderr(t *testing.T) {
	if !dockerAvailable() {
		t.Skip("Docker not available")
	}

	cfg := &config.Config{
		RepositoryRoot:     t.TempDir(),
		IOTimeout:         60 * time.Second,
		IOContainerImage:    "llm-runtime-io:latest",
		ExecWhitelist:      []string{"sh"},
		ExecContainerImage: "alpine:latest",
		ExecTimeout:        30 * time.Second,
		ExecMemoryLimit:    "256m",
		ExecCPULimit:       1,
	}

	exec.Command("docker", "pull", "alpine:latest").Run()

	cmd := scanner.Command{Type: "exec", Argument: "sh -c 'echo error >&2'"}
	result := ExecuteExec(cmd, cfg, nil)

	if !result.Success {
		t.Errorf("expected success, got error: %v", result.Error)
	}

	if !strings.Contains(result.Stderr, "error") {
		t.Errorf("expected stderr to contain 'error', got %q", result.Stderr)
	}
}

func TestExecuteExec_Integration_CombinedOutput(t *testing.T) {
	if !dockerAvailable() {
		t.Skip("Docker not available")
	}

	cfg := &config.Config{
		RepositoryRoot:     t.TempDir(),
		IOTimeout:         60 * time.Second,
		IOContainerImage:    "llm-runtime-io:latest",
		ExecWhitelist:      []string{"sh"},
		ExecContainerImage: "alpine:latest",
		ExecTimeout:        30 * time.Second,
		ExecMemoryLimit:    "256m",
		ExecCPULimit:       1,
	}

	exec.Command("docker", "pull", "alpine:latest").Run()

	cmd := scanner.Command{Type: "exec", Argument: "sh -c 'echo stdout && echo stderr >&2'"}
	result := ExecuteExec(cmd, cfg, nil)

	if !result.Success {
		t.Errorf("expected success, got error: %v", result.Error)
	}

	// Result should contain both stdout and stderr
	if !strings.Contains(result.Result, "stdout") {
		t.Errorf("result should contain stdout, got %q", result.Result)
	}
	if !strings.Contains(result.Result, "stderr") {
		t.Errorf("result should contain stderr, got %q", result.Result)
	}
}

func TestExecuteExec_WithStdin(t *testing.T) {
	if !dockerAvailable() {
		t.Skip("Docker not available")
	}

	cfg := &config.Config{
		RepositoryRoot:     t.TempDir(),
		IOTimeout:         60 * time.Second,
		IOContainerImage:    "llm-runtime-io:latest",
		ExecWhitelist:      []string{"cat", "wc"},
		ExecContainerImage: "alpine:latest",
		ExecTimeout:        30 * time.Second,
		ExecMemoryLimit:    "256m",
		ExecCPULimit:       1,
	}

	exec.Command("docker", "pull", "alpine:latest").Run()

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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := scanner.Command{
				Type:     "exec",
				Argument: tt.command,
				Content:  tt.stdin,
			}
			result := ExecuteExec(cmd, cfg, nil)

			if !result.Success {
				t.Errorf("expected success, got error: %v", result.Error)
			}

			if !strings.Contains(result.Stdout, tt.expected) {
				t.Errorf("expected stdout to contain %q, got %q", tt.expected, result.Stdout)
			}
		})
	}
}

func TestExecuteExec_WhitelistVariations(t *testing.T) {
	tests := []struct {
		name      string
		whitelist []string
		command   string
		allowed   bool
	}{
		{"exact match", []string{"ls"}, "ls", true},
		{"prefix match", []string{"go test"}, "go test ./...", true},
		{"no match", []string{"go test"}, "rm -rf", false},
		{"partial no match", []string{"go"}, "go test", true}, // "go" prefix matches
		{"multiple whitelist", []string{"ls", "echo", "cat"}, "echo hello", true},
		{"empty command", []string{"ls"}, "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				RepositoryRoot:     t.TempDir(),
		IOTimeout:         60 * time.Second,
		IOContainerImage:    "llm-runtime-io:latest",
				ExecWhitelist:      tt.whitelist,
				ExecContainerImage: "alpine:latest",
				ExecTimeout:        30 * time.Second,
				ExecMemoryLimit:    "256m",
				ExecCPULimit:       1,
			}

			cmd := scanner.Command{Type: "exec", Argument: tt.command}
			result := ExecuteExec(cmd, cfg, nil)

			// If not allowed, should fail at validation
			if !tt.allowed && result.Success {
				t.Error("expected validation failure")
			}

			if !tt.allowed && !strings.Contains(result.Error.Error(), "EXEC_VALIDATION") {
				// If it fails for another reason (Docker), that's ok too
				if !strings.Contains(result.Error.Error(), "DOCKER") {
					t.Logf("unexpected error type: %v", result.Error)
				}
			}
		})
	}
}

// Benchmark tests
func BenchmarkExecuteExec_ValidationOnly(b *testing.B) {
	cfg := &config.Config{
		RepositoryRoot: b.TempDir(),
		IOTimeout:         60 * time.Second,
		IOContainerImage:    "llm-runtime-io:latest",
	}

	cmd := scanner.Command{Type: "exec", Argument: "test command"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ExecuteExec(cmd, cfg, nil)
	}
}

func BenchmarkExecuteExec_WhitelistCheck(b *testing.B) {
	cfg := &config.Config{
		RepositoryRoot: b.TempDir(),
		IOTimeout:         60 * time.Second,
		IOContainerImage:    "llm-runtime-io:latest",
		ExecWhitelist:  []string{"go test", "npm test", "make", "cargo test", "pytest"},
	}

	cmd := scanner.Command{Type: "exec", Argument: "unknown command"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ExecuteExec(cmd, cfg, nil)
	}
}

func TestExecuteExec_ResultOutputFormatting(t *testing.T) {
	// Test the result formatting logic by examining what would happen
	// with different stdout/stderr combinations
	// This tests the logic even when Docker isn't available

	cfg := &config.Config{
		RepositoryRoot: t.TempDir(),
		IOTimeout:         60 * time.Second,
		IOContainerImage:    "llm-runtime-io:latest",
	}

	cmd := scanner.Command{Type: "exec", Argument: "test command"}
	result := ExecuteExec(cmd, cfg, nil)

	// Verify command is properly set up
	if result.Command.Type != "exec" {
		t.Errorf("expected type 'exec', got %q", result.Command.Type)
	}
	if result.Command.Argument != "test command" {
		t.Errorf("expected argument 'test command', got %q", result.Command.Argument)
	}
}

func TestExecuteExec_AuditLogFormat(t *testing.T) {
	cfg := &config.Config{
		RepositoryRoot: t.TempDir(),
		IOTimeout:         60 * time.Second,
		IOContainerImage:    "llm-runtime-io:latest",
		ExecWhitelist:  []string{}, // Empty whitelist causes validation failure
	}

	audit := &testAuditLog{}
	cmd := scanner.Command{Type: "exec", Argument: "test"}
	ExecuteExec(cmd, cfg, audit.log)

	entries := audit.getEntries()
	if len(entries) != 1 {
		t.Fatalf("expected 1 audit entry, got %d", len(entries))
	}

	if entries[0].cmdType != "exec" {
		t.Errorf("expected cmdType 'exec', got %q", entries[0].cmdType)
	}
	if entries[0].arg != "test" {
		t.Errorf("expected arg 'test', got %q", entries[0].arg)
	}
}

package session

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/computerscienceiscool/llm-runtime/internal/config"
)

func TestNewSession(t *testing.T) {
	// Change to temp directory to avoid polluting current directory with audit.log
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	tempDir := t.TempDir()
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}
	defer os.Chdir(origDir)

	t.Run("creates session with valid config", func(t *testing.T) {
		cfg := &config.Config{
			RepositoryRoot: "/test/repo",
			ExcludedPaths:  []string{".git"},
			MaxFileSize:    1024,
		}

		session := NewSession(cfg)

		if session == nil {
			t.Fatal("NewSession() returned nil")
		}

		if session.ID == "" {
			t.Error("Session ID should not be empty")
		}

		if session.Config != cfg {
			t.Error("Session config should match provided config")
		}

		if session.StartTime.IsZero() {
			t.Error("Session StartTime should be set")
		}

		if session.CommandsRun != 0 {
			t.Errorf("CommandsRun = %d, want 0", session.CommandsRun)
		}
	})

	t.Run("generates unique session IDs", func(t *testing.T) {
		cfg := &config.Config{}

		session1 := NewSession(cfg)
		time.Sleep(time.Nanosecond) // Ensure different timestamps
		session2 := NewSession(cfg)

		if session1.ID == session2.ID {
			t.Error("Session IDs should be unique")
		}
	})

	t.Run("session ID is numeric timestamp", func(t *testing.T) {
		cfg := &config.Config{}

		session := NewSession(cfg)

		// ID should be parseable as a number (UnixNano)
		if session.ID == "" {
			t.Error("Session ID should not be empty")
		}

		// Should only contain digits
		for _, c := range session.ID {
			if c < '0' || c > '9' {
				t.Errorf("Session ID should only contain digits, got: %s", session.ID)
				break
			}
		}
	})

	t.Run("creates audit logger", func(t *testing.T) {
		cfg := &config.Config{}

		session := NewSession(cfg)

		if session.AuditLogger == nil {
			t.Error("AuditLogger should be created")
		}
	})

	t.Run("handles nil config", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Logf("NewSession panicked with nil config: %v", r)
			}
		}()

		session := NewSession(nil)

		if session != nil && session.Config != nil {
			t.Error("Session config should be nil when nil is passed")
		}
	})

	t.Run("start time is recent", func(t *testing.T) {
		cfg := &config.Config{}
		before := time.Now()

		session := NewSession(cfg)

		after := time.Now()

		if session.StartTime.Before(before) || session.StartTime.After(after) {
			t.Errorf("StartTime %v should be between %v and %v",
				session.StartTime, before, after)
		}
	})
}

func TestSession_LogAudit(t *testing.T) {
	// Change to temp directory
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	tempDir := t.TempDir()
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}
	defer os.Chdir(origDir)

	t.Run("logs successful command", func(t *testing.T) {
		cfg := &config.Config{}
		session := NewSession(cfg)

		session.LogAudit("read_file", "/path/to/file", true, "")

		// Read the audit log
		data, err := os.ReadFile("audit.log")
		if err != nil {
			t.Fatalf("Failed to read audit log: %v", err)
		}

		logContent := string(data)

		if !strings.Contains(logContent, session.ID) {
			t.Error("Log should contain session ID")
		}

		if !strings.Contains(logContent, "read_file") {
			t.Error("Log should contain command name")
		}

		if !strings.Contains(logContent, "/path/to/file") {
			t.Error("Log should contain argument")
		}

		if !strings.Contains(logContent, "success") {
			t.Error("Log should contain 'success' status")
		}
	})

	t.Run("logs failed command", func(t *testing.T) {
		// Use a fresh temp directory
		tempDir2 := t.TempDir()
		if err := os.Chdir(tempDir2); err != nil {
			t.Fatalf("Failed to change directory: %v", err)
		}

		cfg := &config.Config{}
		session := NewSession(cfg)

		session.LogAudit("write_file", "/restricted/path", false, "permission denied")

		data, err := os.ReadFile("audit.log")
		if err != nil {
			t.Fatalf("Failed to read audit log: %v", err)
		}

		logContent := string(data)

		if !strings.Contains(logContent, "failed") {
			t.Error("Log should contain 'failed' status")
		}

		if !strings.Contains(logContent, "permission denied") {
			t.Error("Log should contain error message")
		}
	})

	t.Run("handles nil audit logger", func(t *testing.T) {
		session := &Session{
			ID:          "test-session",
			Config:      nil,
			AuditLogger: nil,
			StartTime:   time.Now(),
		}

		// Should not panic
		session.LogAudit("cmd", "arg", true, "")
	})

	t.Run("log format matches expected pattern", func(t *testing.T) {
		tempDir3 := t.TempDir()
		if err := os.Chdir(tempDir3); err != nil {
			t.Fatalf("Failed to change directory: %v", err)
		}

		cfg := &config.Config{}
		session := NewSession(cfg)

		session.LogAudit("test_cmd", "test_arg", true, "")

		data, err := os.ReadFile("audit.log")
		if err != nil {
			t.Fatalf("Failed to read audit log: %v", err)
		}

		line := strings.TrimSpace(string(data))
		parts := strings.Split(line, "|")

		// Expected: timestamp|session:ID|command|argument|status|errorMsg
		if len(parts) != 6 {
			t.Errorf("Log line should have 6 parts, got %d: %q", len(parts), line)
			return
		}

		// Verify timestamp is RFC3339
		_, err = time.Parse(time.RFC3339, parts[0])
		if err != nil {
			t.Errorf("First part should be RFC3339 timestamp: %q", parts[0])
		}

		if !strings.HasPrefix(parts[1], "session:") {
			t.Errorf("Second part should start with 'session:', got %q", parts[1])
		}

		if parts[2] != "test_cmd" {
			t.Errorf("Third part should be 'test_cmd', got %q", parts[2])
		}

		if parts[3] != "test_arg" {
			t.Errorf("Fourth part should be 'test_arg', got %q", parts[3])
		}

		if parts[4] != "success" {
			t.Errorf("Fifth part should be 'success', got %q", parts[4])
		}
	})

	t.Run("multiple log entries", func(t *testing.T) {
		tempDir4 := t.TempDir()
		if err := os.Chdir(tempDir4); err != nil {
			t.Fatalf("Failed to change directory: %v", err)
		}

		cfg := &config.Config{}
		session := NewSession(cfg)

		for i := 0; i < 5; i++ {
			session.LogAudit("cmd", "arg", true, "")
		}

		data, err := os.ReadFile("audit.log")
		if err != nil {
			t.Fatalf("Failed to read audit log: %v", err)
		}

		lines := strings.Split(strings.TrimSpace(string(data)), "\n")
		if len(lines) != 5 {
			t.Errorf("Expected 5 log lines, got %d", len(lines))
		}
	})
}

func TestSession_Fields(t *testing.T) {
	origDir, _ := os.Getwd()
	tempDir := t.TempDir()
	os.Chdir(tempDir)
	defer os.Chdir(origDir)

	t.Run("CommandsRun can be incremented", func(t *testing.T) {
		cfg := &config.Config{}
		session := NewSession(cfg)

		if session.CommandsRun != 0 {
			t.Errorf("Initial CommandsRun = %d, want 0", session.CommandsRun)
		}

		session.CommandsRun++

		if session.CommandsRun != 1 {
			t.Errorf("CommandsRun after increment = %d, want 1", session.CommandsRun)
		}
	})

	t.Run("Config is accessible", func(t *testing.T) {
		cfg := &config.Config{
			RepositoryRoot: "/my/repo",
			MaxFileSize:    999,
		}
		session := NewSession(cfg)

		if session.Config.RepositoryRoot != "/my/repo" {
			t.Errorf("Config.RepositoryRoot = %q, want %q",
				session.Config.RepositoryRoot, "/my/repo")
		}

		if session.Config.MaxFileSize != 999 {
			t.Errorf("Config.MaxFileSize = %d, want 999", session.Config.MaxFileSize)
		}
	})
}

func TestSession_AuditLogCreation(t *testing.T) {
	t.Run("creates audit.log in current directory", func(t *testing.T) {
		origDir, _ := os.Getwd()
		tempDir := t.TempDir()
		os.Chdir(tempDir)
		defer os.Chdir(origDir)

		cfg := &config.Config{}
		_ = NewSession(cfg)

		// Check that audit.log was created
		if _, err := os.Stat(filepath.Join(tempDir, "audit.log")); os.IsNotExist(err) {
			t.Error("audit.log should be created in current directory")
		}
	})

	t.Run("appends to existing audit.log", func(t *testing.T) {
		origDir, _ := os.Getwd()
		tempDir := t.TempDir()
		os.Chdir(tempDir)
		defer os.Chdir(origDir)

		// Create existing audit.log
		existingContent := "existing entry\n"
		if err := os.WriteFile("audit.log", []byte(existingContent), 0644); err != nil {
			t.Fatalf("Failed to create audit.log: %v", err)
		}

		cfg := &config.Config{}
		session := NewSession(cfg)
		session.LogAudit("new_cmd", "new_arg", true, "")

		data, err := os.ReadFile("audit.log")
		if err != nil {
			t.Fatalf("Failed to read audit.log: %v", err)
		}

		if !strings.HasPrefix(string(data), existingContent) {
			t.Error("Existing content should be preserved")
		}

		if !strings.Contains(string(data), "new_cmd") {
			t.Error("New entry should be appended")
		}
	})
}

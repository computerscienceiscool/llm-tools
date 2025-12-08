package sandbox

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestNewAuditLogger(t *testing.T) {
	t.Run("creates new audit logger", func(t *testing.T) {
		tempDir := t.TempDir()
		logPath := filepath.Join(tempDir, "audit.log")

		logger, err := NewAuditLogger(logPath)
		if err != nil {
			t.Fatalf("NewAuditLogger() error = %v", err)
		}
		defer logger.Close()

		if logger == nil {
			t.Fatal("NewAuditLogger() returned nil logger")
		}

		if logger.logger == nil {
			t.Error("Internal logger should not be nil")
		}

		if logger.file == nil {
			t.Error("File should not be nil")
		}
	})

	t.Run("creates log file if not exists", func(t *testing.T) {
		tempDir := t.TempDir()
		logPath := filepath.Join(tempDir, "new_audit.log")

		logger, err := NewAuditLogger(logPath)
		if err != nil {
			t.Fatalf("NewAuditLogger() error = %v", err)
		}
		defer logger.Close()

		if _, err := os.Stat(logPath); os.IsNotExist(err) {
			t.Error("Log file was not created")
		}
	})

	t.Run("appends to existing log file", func(t *testing.T) {
		tempDir := t.TempDir()
		logPath := filepath.Join(tempDir, "existing_audit.log")

		// Create file with existing content
		existingContent := "previous log entry\n"
		if err := os.WriteFile(logPath, []byte(existingContent), 0644); err != nil {
			t.Fatalf("Failed to create existing log: %v", err)
		}

		logger, err := NewAuditLogger(logPath)
		if err != nil {
			t.Fatalf("NewAuditLogger() error = %v", err)
		}

		logger.Log("session1", "test", "arg", true, "")
		logger.Close()

		data, err := os.ReadFile(logPath)
		if err != nil {
			t.Fatalf("Failed to read log file: %v", err)
		}

		if !strings.HasPrefix(string(data), existingContent) {
			t.Error("Existing content should be preserved")
		}

		if !strings.Contains(string(data), "session1") {
			t.Error("New log entry should be appended")
		}
	})

	t.Run("fails on invalid path", func(t *testing.T) {
		if os.Getuid() == 0 {
			t.Skip("Skipping permission test when running as root")
		}

		_, err := NewAuditLogger("/root/impossible/audit.log")
		if err == nil {
			t.Error("NewAuditLogger() expected error for invalid path")
		}
	})

	t.Run("error message includes path info", func(t *testing.T) {
		if os.Getuid() == 0 {
			t.Skip("Skipping permission test when running as root")
		}

		_, err := NewAuditLogger("/root/impossible/audit.log")
		if err == nil {
			t.Error("Expected error")
			return
		}

		if !strings.Contains(err.Error(), "could not open audit log") {
			t.Errorf("Error message should contain 'could not open audit log', got: %v", err)
		}
	})

	t.Run("creates in nested directory", func(t *testing.T) {
		tempDir := t.TempDir()
		logPath := filepath.Join(tempDir, "nested", "dir", "audit.log")

		// First create the directory
		if err := os.MkdirAll(filepath.Dir(logPath), 0755); err != nil {
			t.Fatalf("Failed to create directory: %v", err)
		}

		logger, err := NewAuditLogger(logPath)
		if err != nil {
			t.Fatalf("NewAuditLogger() error = %v", err)
		}
		defer logger.Close()

		if _, err := os.Stat(logPath); os.IsNotExist(err) {
			t.Error("Log file was not created in nested directory")
		}
	})
}

func TestAuditLogger_Log(t *testing.T) {
	t.Run("logs successful command", func(t *testing.T) {
		tempDir := t.TempDir()
		logPath := filepath.Join(tempDir, "audit.log")

		logger, err := NewAuditLogger(logPath)
		if err != nil {
			t.Fatalf("NewAuditLogger() error = %v", err)
		}
		defer logger.Close()

		logger.Log("session123", "read_file", "/path/to/file", true, "")

		// Flush and read
		logger.Close()

		data, err := os.ReadFile(logPath)
		if err != nil {
			t.Fatalf("Failed to read log: %v", err)
		}

		logContent := string(data)

		if !strings.Contains(logContent, "session:session123") {
			t.Error("Log should contain session ID")
		}

		if !strings.Contains(logContent, "read_file") {
			t.Error("Log should contain command")
		}

		if !strings.Contains(logContent, "/path/to/file") {
			t.Error("Log should contain argument")
		}

		if !strings.Contains(logContent, "success") {
			t.Error("Log should contain 'success' status")
		}
	})

	t.Run("logs failed command", func(t *testing.T) {
		tempDir := t.TempDir()
		logPath := filepath.Join(tempDir, "audit.log")

		logger, err := NewAuditLogger(logPath)
		if err != nil {
			t.Fatalf("NewAuditLogger() error = %v", err)
		}
		defer logger.Close()

		logger.Log("session456", "write_file", "/restricted/path", false, "permission denied")

		logger.Close()

		data, err := os.ReadFile(logPath)
		if err != nil {
			t.Fatalf("Failed to read log: %v", err)
		}

		logContent := string(data)

		if !strings.Contains(logContent, "failed") {
			t.Error("Log should contain 'failed' status")
		}

		if !strings.Contains(logContent, "permission denied") {
			t.Error("Log should contain error message")
		}
	})

	t.Run("logs with timestamp", func(t *testing.T) {
		tempDir := t.TempDir()
		logPath := filepath.Join(tempDir, "audit.log")

		logger, err := NewAuditLogger(logPath)
		if err != nil {
			t.Fatalf("NewAuditLogger() error = %v", err)
		}
		defer logger.Close()

		beforeLog := time.Now()
		logger.Log("session", "cmd", "arg", true, "")
		afterLog := time.Now()

		logger.Close()

		data, err := os.ReadFile(logPath)
		if err != nil {
			t.Fatalf("Failed to read log: %v", err)
		}

		logContent := string(data)

		// Check that timestamp is present (RFC3339 format)
		if !strings.Contains(logContent, beforeLog.Format("2006-01-02")) {
			if !strings.Contains(logContent, afterLog.Format("2006-01-02")) {
				t.Error("Log should contain date in RFC3339 format")
			}
		}
	})

	t.Run("handles nil logger gracefully", func(t *testing.T) {
		logger := &AuditLogger{
			logger: nil,
			file:   nil,
		}

		// Should not panic
		logger.Log("session", "cmd", "arg", true, "")
	})

	t.Run("handles empty strings", func(t *testing.T) {
		tempDir := t.TempDir()
		logPath := filepath.Join(tempDir, "audit.log")

		logger, err := NewAuditLogger(logPath)
		if err != nil {
			t.Fatalf("NewAuditLogger() error = %v", err)
		}
		defer logger.Close()

		logger.Log("", "", "", true, "")

		logger.Close()

		data, err := os.ReadFile(logPath)
		if err != nil {
			t.Fatalf("Failed to read log: %v", err)
		}

		// Should still write something
		if len(data) == 0 {
			t.Error("Log should not be empty")
		}
	})

	t.Run("handles special characters", func(t *testing.T) {
		tempDir := t.TempDir()
		logPath := filepath.Join(tempDir, "audit.log")

		logger, err := NewAuditLogger(logPath)
		if err != nil {
			t.Fatalf("NewAuditLogger() error = %v", err)
		}
		defer logger.Close()

		specialArg := "path/with|pipe|chars"
		logger.Log("session", "cmd", specialArg, true, "error with |pipe|")

		logger.Close()

		data, err := os.ReadFile(logPath)
		if err != nil {
			t.Fatalf("Failed to read log: %v", err)
		}

		// Log uses pipe as separator, so internal pipes might affect parsing
		// But the log should still be written
		if len(data) == 0 {
			t.Error("Log should not be empty")
		}
	})

	t.Run("multiple log entries", func(t *testing.T) {
		tempDir := t.TempDir()
		logPath := filepath.Join(tempDir, "audit.log")

		logger, err := NewAuditLogger(logPath)
		if err != nil {
			t.Fatalf("NewAuditLogger() error = %v", err)
		}
		defer logger.Close()

		for i := 0; i < 10; i++ {
			logger.Log("session", "command", "argument", true, "")
		}

		logger.Close()

		data, err := os.ReadFile(logPath)
		if err != nil {
			t.Fatalf("Failed to read log: %v", err)
		}

		lines := strings.Split(strings.TrimSpace(string(data)), "\n")
		if len(lines) != 10 {
			t.Errorf("Expected 10 log lines, got %d", len(lines))
		}
	})
}

func TestAuditLogger_Close(t *testing.T) {
	t.Run("closes file successfully", func(t *testing.T) {
		tempDir := t.TempDir()
		logPath := filepath.Join(tempDir, "audit.log")

		logger, err := NewAuditLogger(logPath)
		if err != nil {
			t.Fatalf("NewAuditLogger() error = %v", err)
		}

		err = logger.Close()
		if err != nil {
			t.Errorf("Close() error = %v", err)
		}
	})

	t.Run("close with nil file returns nil", func(t *testing.T) {
		logger := &AuditLogger{
			logger: nil,
			file:   nil,
		}

		err := logger.Close()
		if err != nil {
			t.Errorf("Close() with nil file should return nil, got: %v", err)
		}
	})

	t.Run("multiple close calls", func(t *testing.T) {
		tempDir := t.TempDir()
		logPath := filepath.Join(tempDir, "audit.log")

		logger, err := NewAuditLogger(logPath)
		if err != nil {
			t.Fatalf("NewAuditLogger() error = %v", err)
		}

		// First close should succeed
		err = logger.Close()
		if err != nil {
			t.Errorf("First Close() error = %v", err)
		}

		// Second close might return error (file already closed)
		// but should not panic
		_ = logger.Close()
	})
}

func TestAuditLogger_LogFormat(t *testing.T) {
	tempDir := t.TempDir()
	logPath := filepath.Join(tempDir, "audit.log")

	logger, err := NewAuditLogger(logPath)
	if err != nil {
		t.Fatalf("NewAuditLogger() error = %v", err)
	}

	logger.Log("sess123", "test_cmd", "test_arg", true, "")
	logger.Close()

	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("Failed to read log: %v", err)
	}

	line := strings.TrimSpace(string(data))
	parts := strings.Split(line, "|")

	// Expected format: timestamp|session:ID|command|argument|status|errorMsg
	if len(parts) != 6 {
		t.Errorf("Log line should have 6 parts separated by |, got %d: %q", len(parts), line)
		return
	}

	// Check timestamp format (RFC3339)
	_, err = time.Parse(time.RFC3339, parts[0])
	if err != nil {
		t.Errorf("First part should be RFC3339 timestamp, got %q", parts[0])
	}

	if parts[1] != "session:sess123" {
		t.Errorf("Second part should be 'session:sess123', got %q", parts[1])
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
}

func TestAuditLogger_Concurrent(t *testing.T) {
	tempDir := t.TempDir()
	logPath := filepath.Join(tempDir, "audit.log")

	logger, err := NewAuditLogger(logPath)
	if err != nil {
		t.Fatalf("NewAuditLogger() error = %v", err)
	}
	defer logger.Close()

	// Launch multiple goroutines writing to the same logger
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(id int) {
			for j := 0; j < 10; j++ {
				logger.Log("session", "cmd", "arg", true, "")
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	logger.Close()

	// Verify file was written (may have interleaved writes, but shouldn't crash)
	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("Failed to read log: %v", err)
	}

	if len(data) == 0 {
		t.Error("Log file should not be empty after concurrent writes")
	}
}

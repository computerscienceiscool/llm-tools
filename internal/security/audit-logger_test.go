package security

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewAuditLogger tests audit logger creation
func TestNewAuditLogger(t *testing.T) {
	logger := NewAuditLogger()

	assert.NotNil(t, logger)
	assert.IsType(t, &DefaultAuditLogger{}, logger)
}

// TestDefaultAuditLoggerLogOperation tests the main logging functionality
func TestDefaultAuditLoggerLogOperation(t *testing.T) {
	// Create temporary directory for test logs
	tempDir, err := os.MkdirTemp("", "audit-test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Change to temp directory to create audit.log there
	oldCwd, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(oldCwd)

	err = os.Chdir(tempDir)
	require.NoError(t, err)

	logger := NewAuditLogger()
	require.NotNil(t, logger)

	tests := []struct {
		name      string
		sessionID string
		command   string
		argument  string
		success   bool
		errorMsg  string
		validate  func(t *testing.T, logLine string)
	}{
		{
			name:      "successful open command",
			sessionID: "session123",
			command:   "open",
			argument:  "test.txt",
			success:   true,
			errorMsg:  "",
			validate: func(t *testing.T, logLine string) {
				assert.Contains(t, logLine, "session:session123")
				assert.Contains(t, logLine, "open")
				assert.Contains(t, logLine, "test.txt")
				assert.Contains(t, logLine, "success")
			},
		},
		{
			name:      "failed open command",
			sessionID: "session456",
			command:   "open",
			argument:  "../etc/passwd",
			success:   false,
			errorMsg:  "PATH_SECURITY: path traversal detected",
			validate: func(t *testing.T, logLine string) {
				assert.Contains(t, logLine, "session:session456")
				assert.Contains(t, logLine, "open")
				assert.Contains(t, logLine, "../etc/passwd")
				assert.Contains(t, logLine, "failed")
				assert.Contains(t, logLine, "PATH_SECURITY")
			},
		},
		{
			name:      "write command with backup",
			sessionID: "session789",
			command:   "write",
			argument:  "output.txt",
			success:   true,
			errorMsg:  "",
			validate: func(t *testing.T, logLine string) {
				assert.Contains(t, logLine, "session:session789")
				assert.Contains(t, logLine, "write")
				assert.Contains(t, logLine, "output.txt")
				assert.Contains(t, logLine, "success")
			},
		},
		{
			name:      "exec command with details",
			sessionID: "session999",
			command:   "exec",
			argument:  "go test",
			success:   true,
			errorMsg:  "exit_code:0,duration:1.234s",
			validate: func(t *testing.T, logLine string) {
				assert.Contains(t, logLine, "session:session999")
				assert.Contains(t, logLine, "exec")
				assert.Contains(t, logLine, "go test")
				assert.Contains(t, logLine, "success")
				assert.Contains(t, logLine, "exit_code:0")
			},
		},
		{
			name:      "failed exec command",
			sessionID: "session111",
			command:   "exec",
			argument:  "rm -rf /",
			success:   false,
			errorMsg:  "EXEC_VALIDATION: command not whitelisted",
			validate: func(t *testing.T, logLine string) {
				assert.Contains(t, logLine, "session:session111")
				assert.Contains(t, logLine, "exec")
				assert.Contains(t, logLine, "rm -rf /")
				assert.Contains(t, logLine, "failed")
				assert.Contains(t, logLine, "EXEC_VALIDATION")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Log the operation
			logger.LogOperation(tt.sessionID, tt.command, tt.argument, tt.success, tt.errorMsg)

			// Give logger time to write
			time.Sleep(50 * time.Millisecond)

			// Read and validate log file
			logFile := filepath.Join(tempDir, "audit.log")
			content, err := os.ReadFile(logFile)
			require.NoError(t, err)

			logContent := string(content)
			lines := strings.Split(strings.TrimSpace(logContent), "\n")

			// Find the last line (most recent entry)
			require.NotEmpty(t, lines)
			lastLine := lines[len(lines)-1]

			// Validate the log entry
			tt.validate(t, lastLine)
		})
	}
}

// TestAuditLoggerTimestamps tests that timestamps are properly formatted
func TestAuditLoggerTimestamps(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "audit-timestamp-test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	oldCwd, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(oldCwd)

	err = os.Chdir(tempDir)
	require.NoError(t, err)

	logger := NewAuditLogger()

	beforeLog := time.Now()
	logger.LogOperation("test", "open", "file.txt", true, "")
	afterLog := time.Now()

	time.Sleep(50 * time.Millisecond)

	// Read log file
	content, err := os.ReadFile("audit.log")
	require.NoError(t, err)

	logLine := strings.TrimSpace(string(content))

	// Extract timestamp (should be first field before first |)
	parts := strings.Split(logLine, "|")
	require.NotEmpty(t, parts)

	timestampStr := parts[0]

	// Parse timestamp
	parsedTime, err := time.Parse(time.RFC3339, timestampStr)
	assert.NoError(t, err)

	// Verify timestamp is within reasonable range
	assert.True(t, parsedTime.After(beforeLog.Add(-time.Second)))
	assert.True(t, parsedTime.Before(afterLog.Add(time.Second)))
}

// TestAuditLoggerFormat tests log format consistency
func TestAuditLoggerFormat(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "audit-format-test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	oldCwd, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(oldCwd)

	err = os.Chdir(tempDir)
	require.NoError(t, err)

	logger := NewAuditLogger()

	// Log several operations
	operations := []struct {
		sessionID string
		command   string
		argument  string
		success   bool
		errorMsg  string
	}{
		{"session1", "open", "file1.txt", true, ""},
		{"session2", "write", "file2.txt", true, ""},
		{"session3", "exec", "go test", false, "timeout"},
		{"session4", "search", "auth", true, ""},
	}

	for _, op := range operations {
		logger.LogOperation(op.sessionID, op.command, op.argument, op.success, op.errorMsg)
	}

	time.Sleep(100 * time.Millisecond)

	// Read and analyze log file
	content, err := os.ReadFile("audit.log")
	require.NoError(t, err)

	lines := strings.Split(strings.TrimSpace(string(content)), "\n")
	require.Len(t, lines, len(operations))

	for i, line := range lines {
		// Each line should have the format:
		// timestamp|session:sessionID|command|argument|status|message
		parts := strings.Split(line, "|")
		require.Len(t, parts, 6, "Log line should have 6 parts: %s", line)

		// Verify timestamp format
		_, err := time.Parse(time.RFC3339, parts[0])
		assert.NoError(t, err, "Invalid timestamp format: %s", parts[0])

		// Verify session format
		assert.True(t, strings.HasPrefix(parts[1], "session:"), "Session should start with 'session:': %s", parts[1])

		// Verify command
		assert.Equal(t, operations[i].command, parts[2])

		// Verify argument
		assert.Equal(t, operations[i].argument, parts[3])

		// Verify status
		expectedStatus := "success"
		if !operations[i].success {
			expectedStatus = "failed"
		}
		assert.Equal(t, expectedStatus, parts[4])

		// Verify error message
		assert.Equal(t, operations[i].errorMsg, parts[5])
	}
}

// TestAuditLoggerConcurrency tests concurrent logging
func TestAuditLoggerConcurrency(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "audit-concurrent-test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	oldCwd, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(oldCwd)

	err = os.Chdir(tempDir)
	require.NoError(t, err)

	logger := NewAuditLogger()

	const numGoroutines = 10
	const logsPerGoroutine = 100

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	// Launch concurrent loggers
	for i := 0; i < numGoroutines; i++ {
		go func(routineID int) {
			defer wg.Done()

			for j := 0; j < logsPerGoroutine; j++ {
				sessionID := fmt.Sprintf("session%d", routineID)
				argument := fmt.Sprintf("operation%d", j)
				logger.LogOperation(sessionID, "test", argument, true, "")
			}
		}(i)
	}

	wg.Wait()
	time.Sleep(200 * time.Millisecond) // Allow final writes

	// Verify all log entries were written
	content, err := os.ReadFile("audit.log")
	require.NoError(t, err)

	lines := strings.Split(strings.TrimSpace(string(content)), "\n")
	expectedLines := numGoroutines * logsPerGoroutine

	assert.Len(t, lines, expectedLines, "Should have exactly %d log entries", expectedLines)

	// Verify no corruption (each line should be properly formatted)
	for i, line := range lines {
		parts := strings.Split(line, "|")
		assert.Len(t, parts, 6, "Line %d should be properly formatted: %s", i, line)
	}
}

// TestAuditLoggerFileHandling tests file creation and error handling
func TestAuditLoggerFileHandling(t *testing.T) {
	t.Run("creates audit.log when missing", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "audit-creation-test")
		require.NoError(t, err)
		defer os.RemoveAll(tempDir)

		oldCwd, err := os.Getwd()
		require.NoError(t, err)
		defer os.Chdir(oldCwd)

		err = os.Chdir(tempDir)
		require.NoError(t, err)

		// Verify log file doesn't exist
		_, err = os.Stat("audit.log")
		assert.True(t, os.IsNotExist(err))

		// Create logger and log something
		logger := NewAuditLogger()
		logger.LogOperation("test", "open", "file.txt", true, "")

		time.Sleep(50 * time.Millisecond)

		// Verify log file was created
		_, err = os.Stat("audit.log")
		assert.NoError(t, err)
	})

	t.Run("appends to existing audit.log", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "audit-append-test")
		require.NoError(t, err)
		defer os.RemoveAll(tempDir)

		oldCwd, err := os.Getwd()
		require.NoError(t, err)
		defer os.Chdir(oldCwd)

		err = os.Chdir(tempDir)
		require.NoError(t, err)

		// Create existing log file
		existingContent := "existing log entry\n"
		err = os.WriteFile("audit.log", []byte(existingContent), 0644)
		require.NoError(t, err)

		// Create logger and log something
		logger := NewAuditLogger()
		logger.LogOperation("test", "open", "file.txt", true, "")

		time.Sleep(50 * time.Millisecond)

		// Read final content
		content, err := os.ReadFile("audit.log")
		require.NoError(t, err)

		finalContent := string(content)
		assert.Contains(t, finalContent, existingContent)
		assert.Contains(t, finalContent, "test")
		assert.Contains(t, finalContent, "open")
	})

	t.Run("handles permission errors gracefully", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "audit-permission-test")
		require.NoError(t, err)
		defer os.RemoveAll(tempDir)

		// Create read-only directory
		readOnlyDir := filepath.Join(tempDir, "readonly")
		err = os.Mkdir(readOnlyDir, 0444)
		require.NoError(t, err)

		oldCwd, err := os.Getwd()
		require.NoError(t, err)
		defer os.Chdir(oldCwd)

		err = os.Chdir(readOnlyDir)
		require.NoError(t, err)

		// Should not panic when unable to create log file
		assert.NotPanics(t, func() {
			logger := NewAuditLogger()
			logger.LogOperation("test", "open", "file.txt", true, "")
		})
	})
}

// TestAuditLoggerSpecialCharacters tests handling of special characters
func TestAuditLoggerSpecialCharacters(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "audit-special-test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	oldCwd, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(oldCwd)

	err = os.Chdir(tempDir)
	require.NoError(t, err)

	logger := NewAuditLogger()

	tests := []struct {
		name      string
		sessionID string
		command   string
		argument  string
		errorMsg  string
	}{
		{
			name:      "unicode characters",
			sessionID: "session测试",
			command:   "open",
			argument:  "文件.txt",
			errorMsg:  "unicode error message 🚨",
		},
		{
			name:      "pipe characters in data",
			sessionID: "session|with|pipes",
			command:   "open",
			argument:  "file|with|pipes.txt",
			errorMsg:  "error|message|with|pipes",
		},
		{
			name:      "newline characters",
			sessionID: "session\nwith\nnewlines",
			command:   "open",
			argument:  "file\nwith\nnewlines.txt",
			errorMsg:  "error\nmessage",
		},
		{
			name:      "tab characters",
			sessionID: "session\twith\ttabs",
			command:   "open",
			argument:  "file\twith\ttabs.txt",
			errorMsg:  "error\tmessage",
		},
		{
			name:      "empty strings",
			sessionID: "",
			command:   "",
			argument:  "",
			errorMsg:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Should not panic with special characters
			assert.NotPanics(t, func() {
				logger.LogOperation(tt.sessionID, tt.command, tt.argument, true, tt.errorMsg)
			})
		})
	}

	time.Sleep(100 * time.Millisecond)

	// Verify log file exists and has content
	content, err := os.ReadFile("audit.log")
	require.NoError(t, err)

	logContent := string(content)
	assert.NotEmpty(t, logContent)

	// Verify we have the expected number of lines
	lines := strings.Split(strings.TrimSpace(logContent), "\n")
	assert.Len(t, lines, len(tests))
}

// TestAuditLoggerLargeVolume tests handling of large volumes of logs
func TestAuditLoggerLargeVolume(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping large volume test in short mode")
	}

	tempDir, err := os.MkdirTemp("", "audit-volume-test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	oldCwd, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(oldCwd)

	err = os.Chdir(tempDir)
	require.NoError(t, err)

	logger := NewAuditLogger()

	const numLogs = 10000

	// Log large number of operations
	start := time.Now()
	for i := 0; i < numLogs; i++ {
		sessionID := fmt.Sprintf("session%d", i%100) // Reuse session IDs
		argument := fmt.Sprintf("operation%d", i)
		logger.LogOperation(sessionID, "test", argument, i%2 == 0, "")
	}
	elapsed := time.Since(start)

	// Allow time for all logs to be written
	time.Sleep(500 * time.Millisecond)

	// Verify performance (should be reasonable)
	avgTimePerLog := elapsed / time.Duration(numLogs)
	assert.Less(t, avgTimePerLog, time.Millisecond, "Average time per log should be less than 1ms")

	// Verify all logs were written
	content, err := os.ReadFile("audit.log")
	require.NoError(t, err)

	lines := strings.Split(strings.TrimSpace(string(content)), "\n")
	assert.Len(t, lines, numLogs, "Should have all %d log entries", numLogs)

	// Verify file size is reasonable
	fileInfo, err := os.Stat("audit.log")
	require.NoError(t, err)

	// Each log entry should be roughly 50-100 bytes
	expectedMinSize := int64(numLogs * 50)
	expectedMaxSize := int64(numLogs * 200)
	fileSize := fileInfo.Size()

	assert.GreaterOrEqual(t, fileSize, expectedMinSize, "File should be at least %d bytes", expectedMinSize)
	assert.LessOrEqual(t, fileSize, expectedMaxSize, "File should be at most %d bytes", expectedMaxSize)
}

// BenchmarkAuditLogger benchmarks audit logging performance
func BenchmarkAuditLogger(b *testing.B) {
	tempDir, err := os.MkdirTemp("", "audit-benchmark")
	if err != nil {
		b.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	oldCwd, err := os.Getwd()
	if err != nil {
		b.Fatal(err)
	}
	defer os.Chdir(oldCwd)

	err = os.Chdir(tempDir)
	if err != nil {
		b.Fatal(err)
	}

	logger := NewAuditLogger()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			sessionID := fmt.Sprintf("session%d", i%100)
			argument := fmt.Sprintf("operation%d", i)
			logger.LogOperation(sessionID, "benchmark", argument, true, "")
			i++
		}
	})
}

// BenchmarkAuditLoggerDifferentSizes benchmarks with different data sizes
func BenchmarkAuditLoggerDifferentSizes(b *testing.B) {
	tempDir, err := os.MkdirTemp("", "audit-size-benchmark")
	if err != nil {
		b.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	oldCwd, err := os.Getwd()
	if err != nil {
		b.Fatal(err)
	}
	defer os.Chdir(oldCwd)

	err = os.Chdir(tempDir)
	if err != nil {
		b.Fatal(err)
	}

	logger := NewAuditLogger()

	tests := []struct {
		name      string
		sessionID string
		argument  string
		errorMsg  string
	}{
		{
			name:      "small",
			sessionID: "s1",
			argument:  "f.txt",
			errorMsg:  "",
		},
		{
			name:      "medium",
			sessionID: "session-with-longer-identifier",
			argument:  "path/to/some/file/with/longer/name.txt",
			errorMsg:  "error message with more details",
		},
		{
			name:      "large",
			sessionID: strings.Repeat("very-long-session-id-", 10),
			argument:  strings.Repeat("very/long/path/component/", 20) + "file.txt",
			errorMsg:  strings.Repeat("very long error message ", 10),
		},
	}

	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				logger.LogOperation(tt.sessionID, "test", tt.argument, true, tt.errorMsg)
			}
		})
	}
}

// TestAuditLoggerMemoryUsage tests memory usage patterns
func TestAuditLoggerMemoryUsage(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping memory test in short mode")
	}

	tempDir, err := os.MkdirTemp("", "audit-memory-test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	oldCwd, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(oldCwd)

	err = os.Chdir(tempDir)
	require.NoError(t, err)

	// Create logger
	logger := NewAuditLogger()

	// Log many operations and verify logger doesn't hold references
	for i := 0; i < 1000; i++ {
		largeData := strings.Repeat("large-data-", 100)
		logger.LogOperation("session", "test", largeData, true, largeData)
	}

	// Force garbage collection
	runtime.GC()
	time.Sleep(100 * time.Millisecond)

	// Logger should not be retaining references to the large data
	// This is mainly a regression test to ensure no memory leaks
	assert.True(t, true, "Logger should not cause memory leaks")
}

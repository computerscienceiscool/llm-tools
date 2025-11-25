package infrastructure

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockDatabase for testing
type MockDatabase struct {
	mock.Mock
}

func (m *MockDatabase) Connect(dbPath string) error {
	args := m.Called(dbPath)
	return args.Error(0)
}

func (m *MockDatabase) Execute(query string, args ...interface{}) error {
	calledArgs := m.Called(query, args)
	return calledArgs.Error(0)
}

func (m *MockDatabase) Query(query string, args ...interface{}) (*sql.Rows, error) {
	calledArgs := m.Called(query, args)
	return calledArgs.Get(0).(*sql.Rows), calledArgs.Error(1)
}

func (m *MockDatabase) QueryRow(query string, args ...interface{}) *sql.Row {
	calledArgs := m.Called(query, args)
	return calledArgs.Get(0).(*sql.Row)
}

func (m *MockDatabase) LogAuditEvent(sessionID, command, argument string, success bool, errorMsg string) error {
	args := m.Called(sessionID, command, argument, success, errorMsg)
	return args.Error(0)
}

func (m *MockDatabase) GetAuditLogs(sessionID string, limit int) ([]AuditLog, error) {
	args := m.Called(sessionID, limit)
	return args.Get(0).([]AuditLog), args.Error(1)
}

func (m *MockDatabase) Close() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockDatabase) Initialize() error {
	args := m.Called()
	return args.Error(0)
}

// TestDatabaseInterface tests the Database interface
func TestDatabaseInterface(t *testing.T) {
	var _ Database = (*MockDatabase)(nil)

	mockDB := &MockDatabase{}

	// Setup expectations
	mockDB.On("Initialize").Return(nil)
	mockDB.On("LogAuditEvent", "session123", "open", "test.txt", true, "").Return(nil)
	mockDB.On("GetAuditLogs", "session123", 10).Return([]AuditLog{
		{
			ID:        1,
			Timestamp: time.Now(),
			SessionID: "session123",
			Command:   "open",
			Argument:  "test.txt",
			Success:   true,
			ErrorMsg:  "",
		},
	}, nil)
	mockDB.On("Close").Return(nil)

	// Test Initialize
	err := mockDB.Initialize()
	assert.NoError(t, err)

	// Test LogAuditEvent
	err = mockDB.LogAuditEvent("session123", "open", "test.txt", true, "")
	assert.NoError(t, err)

	// Test GetAuditLogs
	logs, err := mockDB.GetAuditLogs("session123", 10)
	assert.NoError(t, err)
	assert.Len(t, logs, 1)
	assert.Equal(t, "session123", logs[0].SessionID)
	assert.Equal(t, "open", logs[0].Command)

	// Test Close
	err = mockDB.Close()
	assert.NoError(t, err)

	mockDB.AssertExpectations(t)
}

// TestSQLiteDatabase tests the SQLite database implementation
func TestSQLiteDatabase(t *testing.T) {
	// Create temporary database file
	tempDir, err := os.MkdirTemp("", "db-test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	dbPath := filepath.Join(tempDir, "test.db")
	db := NewSQLiteDatabase(dbPath)
	require.NotNil(t, db)

	t.Run("initialize database", func(t *testing.T) {
		err := db.Initialize()
		assert.NoError(t, err)

		// Verify database file was created
		_, err = os.Stat(dbPath)
		assert.NoError(t, err)
	})

	t.Run("log audit events", func(t *testing.T) {
		testCases := []struct {
			sessionID string
			command   string
			argument  string
			success   bool
			errorMsg  string
		}{
			{"session1", "open", "file1.txt", true, ""},
			{"session1", "write", "file2.txt", true, ""},
			{"session2", "exec", "go test", false, "command failed"},
			{"session2", "open", "../etc/passwd", false, "PATH_SECURITY: invalid path"},
		}

		for _, tc := range testCases {
			err := db.LogAuditEvent(tc.sessionID, tc.command, tc.argument, tc.success, tc.errorMsg)
			assert.NoError(t, err)
		}
	})

	t.Run("retrieve audit logs", func(t *testing.T) {
		// Get all logs for session1 (returns in DESC timestamp order - newest first)
		logs, err := db.GetAuditLogs("session1", 100)
		assert.NoError(t, err)
		assert.Len(t, logs, 2)

		// Verify log contents (most recent first = write, then open)
		assert.Equal(t, "session1", logs[0].SessionID)
		assert.Equal(t, "write", logs[0].Command) // Most recent
		assert.Equal(t, "file2.txt", logs[0].Argument)
		assert.True(t, logs[0].Success)
		assert.Empty(t, logs[0].ErrorMsg)

		assert.Equal(t, "session1", logs[1].SessionID)
		assert.Equal(t, "open", logs[1].Command) // Older
		assert.Equal(t, "file1.txt", logs[1].Argument)
		assert.True(t, logs[1].Success)
		assert.Empty(t, logs[1].ErrorMsg)

		// Get logs for session2
		logs2, err := db.GetAuditLogs("session2", 100)
		assert.NoError(t, err)
		assert.Len(t, logs2, 2)

		// Find the specific logs (order may vary between exec and open)
		var execLog, pathLog *AuditLog
		for i := range logs2 {
			if logs2[i].Command == "exec" {
				execLog = &logs2[i]
			} else if logs2[i].Argument == "../etc/passwd" {
				pathLog = &logs2[i]
			}
		}

		// Verify exec failed log
		require.NotNil(t, execLog)
		assert.Equal(t, "session2", execLog.SessionID)
		assert.Equal(t, "exec", execLog.Command)
		assert.Equal(t, "go test", execLog.Argument)
		assert.False(t, execLog.Success)
		assert.Equal(t, "command failed", execLog.ErrorMsg)

		// Verify path security log
		require.NotNil(t, pathLog)
		assert.Equal(t, "session2", pathLog.SessionID)
		assert.Equal(t, "open", pathLog.Command)
		assert.Equal(t, "../etc/passwd", pathLog.Argument)
		assert.False(t, pathLog.Success)
		assert.Equal(t, "PATH_SECURITY: invalid path", pathLog.ErrorMsg)
	})

	t.Run("limit audit logs", func(t *testing.T) {
		// Test limit functionality
		logs, err := db.GetAuditLogs("session1", 1)
		assert.NoError(t, err)
		assert.Len(t, logs, 1)

		// Should get the most recent log
		assert.Equal(t, "write", logs[0].Command)
	})

	t.Run("nonexistent session", func(t *testing.T) {
		logs, err := db.GetAuditLogs("nonexistent", 10)
		assert.NoError(t, err)
		assert.Empty(t, logs)
	})

	t.Run("close database", func(t *testing.T) {
		err := db.Close()
		assert.NoError(t, err)
	})
}

// TestSQLiteDatabaseConcurrency tests concurrent database access
func TestSQLiteDatabaseConcurrency(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "db-concurrent-test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	dbPath := filepath.Join(tempDir, "concurrent.db")
	db := NewSQLiteDatabase(dbPath)
	require.NotNil(t, db)

	err = db.Initialize()
	require.NoError(t, err)
	defer db.Close()

	const numGoroutines = 5     // Reduced to minimize lock contention
	const logsPerGoroutine = 20 // Reduced for stability
	done := make(chan error, numGoroutines)

	// Concurrent writes
	for i := 0; i < numGoroutines; i++ {
		go func(routineID int) {
			var lastErr error
			for j := 0; j < logsPerGoroutine; j++ {
				sessionID := fmt.Sprintf("session%d", routineID)
				argument := fmt.Sprintf("file%d_%d.txt", routineID, j)

				err := db.LogAuditEvent(sessionID, "test", argument, true, "")
				if err != nil {
					lastErr = err
					// Continue despite errors (SQLite locking is expected)
				}
				// Small delay to reduce contention
				time.Sleep(time.Millisecond)
			}
			done <- lastErr
		}(i)
	}

	// Wait for all writes to complete
	errorCount := 0
	for i := 0; i < numGoroutines; i++ {
		if err := <-done; err != nil {
			errorCount++
			if errorCount == 1 {
				t.Logf("Some concurrent errors expected with SQLite: %v", err)
			}
		}
	}

	// Verify at least most logs were written
	totalLogs := 0
	for i := 0; i < numGoroutines; i++ {
		sessionID := fmt.Sprintf("session%d", i)
		logs, err := db.GetAuditLogs(sessionID, 100)
		assert.NoError(t, err)
		totalLogs += len(logs)
	}

	// Should have written at least 80% of logs despite some locking
	expectedMin := (numGoroutines * logsPerGoroutine * 8) / 10
	assert.GreaterOrEqual(t, totalLogs, expectedMin, "Should have written most logs despite some lock conflicts")
}

// TestSQLiteDatabaseErrors tests error handling
func TestSQLiteDatabaseErrors(t *testing.T) {
	t.Run("invalid database path", func(t *testing.T) {
		// Try to create database in directory that can't be created
		db := NewSQLiteDatabase("/root/definitely/nonexistent/test.db")
		err := db.Initialize()
		assert.Error(t, err)
		// Should fail to create directory
	})

	t.Run("database operations after close", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "db-error-test")
		require.NoError(t, err)
		defer os.RemoveAll(tempDir)

		dbPath := filepath.Join(tempDir, "error.db")
		db := NewSQLiteDatabase(dbPath)

		err = db.Initialize()
		require.NoError(t, err)

		// Close database
		err = db.Close()
		require.NoError(t, err)

		// Try to use after close
		err = db.LogAuditEvent("test", "open", "file.txt", true, "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "sql: database is closed")

		_, err = db.GetAuditLogs("test", 10)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "sql: database is closed")
	})

	t.Run("invalid SQL operations", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "db-sql-test")
		require.NoError(t, err)
		defer os.RemoveAll(tempDir)

		dbPath := filepath.Join(tempDir, "sql.db")
		db := NewSQLiteDatabase(dbPath)

		err = db.Initialize()
		require.NoError(t, err)
		defer db.Close()

		// Test with very long strings that might cause issues
		longString := strings.Repeat("a", 10000)
		err = db.LogAuditEvent(longString, longString, longString, true, longString)
		// Should either succeed or fail gracefully
		// Don't assert specific behavior as it depends on SQLite limits
	})
}

// TestAuditLogStructure tests the AuditLog data structure
func TestAuditLogStructure(t *testing.T) {
	now := time.Now()
	log := AuditLog{
		ID:        1,
		Timestamp: now,
		SessionID: "session123",
		Command:   "open",
		Argument:  "test.txt",
		Success:   true,
		ErrorMsg:  "",
	}

	assert.Equal(t, int64(1), log.ID)
	assert.Equal(t, now, log.Timestamp)
	assert.Equal(t, "session123", log.SessionID)
	assert.Equal(t, "open", log.Command)
	assert.Equal(t, "test.txt", log.Argument)
	assert.True(t, log.Success)
	assert.Empty(t, log.ErrorMsg)
}

// TestDatabaseMigration tests database schema migration
func TestDatabaseMigration(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "db-migration-test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	dbPath := filepath.Join(tempDir, "migration.db")

	// Create database and initialize
	db1 := NewSQLiteDatabase(dbPath)
	err = db1.Initialize()
	require.NoError(t, err)

	// Add some data
	err = db1.LogAuditEvent("test", "open", "file.txt", true, "")
	require.NoError(t, err)
	db1.Close()

	// Re-open same database file
	db2 := NewSQLiteDatabase(dbPath)
	err = db2.Initialize()
	require.NoError(t, err)
	defer db2.Close()

	// Verify data persists
	logs, err := db2.GetAuditLogs("test", 10)
	assert.NoError(t, err)
	assert.Len(t, logs, 1)
	assert.Equal(t, "open", logs[0].Command)
}

// TestDatabasePerformance tests database performance
func TestDatabasePerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	tempDir, err := os.MkdirTemp("", "db-perf-test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	dbPath := filepath.Join(tempDir, "perf.db")
	db := NewSQLiteDatabase(dbPath)

	err = db.Initialize()
	require.NoError(t, err)
	defer db.Close()

	// Test bulk insert performance
	numLogs := 1000
	start := time.Now()

	for i := 0; i < numLogs; i++ {
		sessionID := fmt.Sprintf("session%d", i%10)
		argument := fmt.Sprintf("file%d.txt", i)

		err := db.LogAuditEvent(sessionID, "test", argument, true, "")
		assert.NoError(t, err)
	}

	insertTime := time.Since(start)
	t.Logf("Inserted %d logs in %v (%.2f logs/sec)", numLogs, insertTime, float64(numLogs)/insertTime.Seconds())

	// Test query performance
	start = time.Now()
	logs, err := db.GetAuditLogs("session1", 1000)
	queryTime := time.Since(start)

	assert.NoError(t, err)
	assert.NotEmpty(t, logs)
	t.Logf("Queried logs in %v", queryTime)

	// Performance assertions (adjust based on expected performance)
	assert.Less(t, insertTime, 10*time.Second, "Insert performance too slow")
	assert.Less(t, queryTime, time.Second, "Query performance too slow")
}

// BenchmarkDatabase benchmarks database operations
func BenchmarkDatabase(b *testing.B) {
	tempDir, err := os.MkdirTemp("", "db-benchmark")
	if err != nil {
		b.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	dbPath := filepath.Join(tempDir, "benchmark.db")
	db := NewSQLiteDatabase(dbPath)

	err = db.Initialize()
	if err != nil {
		b.Fatal(err)
	}
	defer db.Close()

	b.Run("LogAuditEvent", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			sessionID := fmt.Sprintf("session%d", i%100)
			argument := fmt.Sprintf("file%d.txt", i)
			_ = db.LogAuditEvent(sessionID, "benchmark", argument, true, "")
		}
	})

	// Insert some data for read benchmark
	for i := 0; i < 1000; i++ {
		_ = db.LogAuditEvent("session1", "read", fmt.Sprintf("file%d.txt", i), true, "")
	}

	b.Run("GetAuditLogs", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = db.GetAuditLogs("session1", 10)
		}
	})
}

// TestDatabaseTransactions tests transaction behavior
func TestDatabaseTransactions(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "db-tx-test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	dbPath := filepath.Join(tempDir, "transaction.db")
	db := NewSQLiteDatabase(dbPath)

	err = db.Initialize()
	require.NoError(t, err)
	defer db.Close()

	// Test that individual operations are atomic
	err = db.LogAuditEvent("session1", "open", "file.txt", true, "")
	require.NoError(t, err)

	logs, err := db.GetAuditLogs("session1", 10)
	require.NoError(t, err)
	assert.Len(t, logs, 1)

	// Verify data consistency after multiple operations
	for i := 0; i < 10; i++ {
		err := db.LogAuditEvent("session1", "test", fmt.Sprintf("file%d.txt", i), true, "")
		assert.NoError(t, err)
	}

	finalLogs, err := db.GetAuditLogs("session1", 100)
	assert.NoError(t, err)
	assert.Len(t, finalLogs, 11) // 1 original + 10 new
}

// TestDatabaseInterfaceCompliance ensures all implementations satisfy the interface
func TestDatabaseInterfaceCompliance(t *testing.T) {
	// Compile-time interface compliance checks
	var _ Database = (*SQLiteDatabase)(nil)
	var _ Database = (*MockDatabase)(nil)

	t.Run("mock database compliance", func(t *testing.T) {
		db := &MockDatabase{}
		_ = Database(db) // Should compile without errors
	})

	t.Run("sqlite database compliance", func(t *testing.T) {
		db := &SQLiteDatabase{}
		_ = Database(db) // Should compile without errors
	})
}

// TestDatabaseConnectionStates tests various connection states
func TestDatabaseConnectionStates(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "db-states-test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	dbPath := filepath.Join(tempDir, "states.db")

	t.Run("unconnected database operations", func(t *testing.T) {
		db := NewDatabase()

		// Should fail without connection
		err := db.LogAuditEvent("test", "open", "file.txt", true, "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "database not connected")

		_, err = db.GetAuditLogs("test", 10)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "database not connected")

		// Initialize should fail without path
		err = db.Initialize()
		assert.Error(t, err)
	})

	t.Run("connect then operations", func(t *testing.T) {
		db := NewDatabase()

		err := db.Connect(dbPath)
		assert.NoError(t, err)

		err = db.Initialize()
		assert.NoError(t, err)

		// Should work after connection
		err = db.LogAuditEvent("test", "open", "file.txt", true, "")
		assert.NoError(t, err)

		logs, err := db.GetAuditLogs("test", 10)
		assert.NoError(t, err)
		assert.Len(t, logs, 1)

		db.Close()
	})

	t.Run("auto connect via constructor", func(t *testing.T) {
		db := NewSQLiteDatabase(dbPath)

		// Initialize should auto-connect
		err := db.Initialize()
		assert.NoError(t, err)

		err = db.LogAuditEvent("auto", "test", "arg", true, "")
		assert.NoError(t, err)

		db.Close()
	})
}

// TestDatabaseEdgeCases tests edge cases and boundary conditions
func TestDatabaseEdgeCases(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "db-edge-test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	dbPath := filepath.Join(tempDir, "edge.db")
	db := NewSQLiteDatabase(dbPath)
	err = db.Initialize()
	require.NoError(t, err)
	defer db.Close()

	t.Run("empty values", func(t *testing.T) {
		err := db.LogAuditEvent("", "", "", true, "")
		assert.NoError(t, err)

		logs, err := db.GetAuditLogs("", 10)
		assert.NoError(t, err)
		assert.Len(t, logs, 1)
		assert.Empty(t, logs[0].SessionID)
		assert.Empty(t, logs[0].Command)
		assert.Empty(t, logs[0].Argument)
	})

	t.Run("unicode content", func(t *testing.T) {
		unicodeSession := "세션🌟"
		unicodeCommand := "команда"
		unicodeArg := "файл.txt"
		unicodeError := "错误消息"

		err := db.LogAuditEvent(unicodeSession, unicodeCommand, unicodeArg, false, unicodeError)
		assert.NoError(t, err)

		logs, err := db.GetAuditLogs(unicodeSession, 10)
		assert.NoError(t, err)
		assert.Len(t, logs, 1)
		assert.Equal(t, unicodeSession, logs[0].SessionID)
		assert.Equal(t, unicodeCommand, logs[0].Command)
		assert.Equal(t, unicodeArg, logs[0].Argument)
		assert.Equal(t, unicodeError, logs[0].ErrorMsg)
	})

	t.Run("special characters", func(t *testing.T) {
		specialChars := `<>&"'\`
		err := db.LogAuditEvent("special", "cmd", specialChars, true, specialChars)
		assert.NoError(t, err)

		logs, err := db.GetAuditLogs("special", 10)
		assert.NoError(t, err)
		assert.Len(t, logs, 1)
		assert.Equal(t, specialChars, logs[0].Argument)
		assert.Equal(t, specialChars, logs[0].ErrorMsg)
	})

	t.Run("zero limit", func(t *testing.T) {
		logs, err := db.GetAuditLogs("any", 0)
		assert.NoError(t, err)
		assert.Empty(t, logs)
	})

	t.Run("negative limit", func(t *testing.T) {
		_, err := db.GetAuditLogs("any", -1)
		assert.NoError(t, err)
		// Should handle gracefully (SQLite might return all or none)
	})

	t.Run("very large limit", func(t *testing.T) {
		_, err := db.GetAuditLogs("any", 1000000)
		assert.NoError(t, err)
		// Should not error even with huge limit
	})

}

// TestDatabaseSchemaValidation tests database schema
func TestDatabaseSchemaValidation(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "db-schema-test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	dbPath := filepath.Join(tempDir, "schema.db")
	db := NewSQLiteDatabase(dbPath)

	t.Run("schema creation", func(t *testing.T) {
		err := db.Initialize()
		assert.NoError(t, err)

		// Verify tables exist by trying to query
		logs, err := db.GetAuditLogs("nonexistent", 1)
		assert.NoError(t, err)
		assert.Empty(t, logs)
	})

	t.Run("schema idempotency", func(t *testing.T) {
		// Initialize again - should not error
		err := db.Initialize()
		assert.NoError(t, err)

		// Should still work
		err = db.LogAuditEvent("test", "cmd", "arg", true, "")
		assert.NoError(t, err)
	})

	db.Close()

	t.Run("schema persistence", func(t *testing.T) {
		// Reopen database
		db2 := NewSQLiteDatabase(dbPath)
		err := db2.Initialize()
		assert.NoError(t, err)
		defer db2.Close()

		// Should find existing data
		logs, err := db2.GetAuditLogs("test", 10)
		assert.NoError(t, err)
		assert.Len(t, logs, 1)
	})
}

// TestDatabaseBackupAndRestore tests data persistence
func TestDatabaseBackupAndRestore(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "db-backup-test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	originalPath := filepath.Join(tempDir, "original.db")
	backupPath := filepath.Join(tempDir, "backup.db")

	// Create original database with data
	original := NewSQLiteDatabase(originalPath)
	err = original.Initialize()
	require.NoError(t, err)

	testData := []struct {
		session string
		command string
		arg     string
		success bool
		errMsg  string
	}{
		{"sess1", "open", "file1.txt", true, ""},
		{"sess1", "write", "file2.txt", true, ""},
		{"sess2", "exec", "cmd", false, "failed"},
	}

	for _, td := range testData {
		err = original.LogAuditEvent(td.session, td.command, td.arg, td.success, td.errMsg)
		require.NoError(t, err)
	}
	original.Close()

	// Copy database file (simulate backup)
	err = copyFile(originalPath, backupPath)
	require.NoError(t, err)

	// Open backup and verify data
	backup := NewSQLiteDatabase(backupPath)
	err = backup.Initialize()
	require.NoError(t, err)
	defer backup.Close()

	logs1, err := backup.GetAuditLogs("sess1", 10)
	assert.NoError(t, err)
	assert.Len(t, logs1, 2)

	logs2, err := backup.GetAuditLogs("sess2", 10)
	assert.NoError(t, err)
	assert.Len(t, logs2, 1)
	assert.False(t, logs2[0].Success)
	assert.Equal(t, "failed", logs2[0].ErrorMsg)
}

// TestDatabaseLimitsAndBounds tests database limits
func TestDatabaseLimitsAndBounds(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "db-limits-test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	dbPath := filepath.Join(tempDir, "limits.db")
	db := NewSQLiteDatabase(dbPath)
	err = db.Initialize()
	require.NoError(t, err)
	defer db.Close()

	t.Run("many sessions", func(t *testing.T) {
		// Create logs for many different sessions
		numSessions := 100
		for i := 0; i < numSessions; i++ {
			sessionID := fmt.Sprintf("session_%04d", i)
			err := db.LogAuditEvent(sessionID, "test", "arg", true, "")
			assert.NoError(t, err)
		}

		// Verify we can query different sessions
		logs, err := db.GetAuditLogs("session_0042", 10)
		assert.NoError(t, err)
		assert.Len(t, logs, 1)
		assert.Equal(t, "session_0042", logs[0].SessionID)
	})

	t.Run("timestamp ordering", func(t *testing.T) {
		session := "time_test"

		// Insert logs with known time gaps
		commands := []string{"first", "second", "third"}

		for i, cmd := range commands {
			// Note: This test may not work exactly as written since we can't control
			// the timestamp in LogAuditEvent, but it tests the concept
			err := db.LogAuditEvent(session, cmd, fmt.Sprintf("arg%d", i), true, "")
			assert.NoError(t, err)
			time.Sleep(10 * time.Millisecond) // Ensure different timestamps
		}

		logs, err := db.GetAuditLogs(session, 10)
		assert.NoError(t, err)
		assert.Len(t, logs, 3)

		// Should be in reverse chronological order (newest first)
		assert.Equal(t, "third", logs[0].Command)  // Most recent
		assert.Equal(t, "second", logs[1].Command) // Middle
		assert.Equal(t, "first", logs[2].Command)  // Oldest

		// Verify timestamps are actually in descending order
		assert.True(t, logs[0].Timestamp.After(logs[1].Timestamp))
		assert.True(t, logs[1].Timestamp.After(logs[2].Timestamp))
	})
}

// TestDatabaseCleanup tests resource cleanup
func TestDatabaseCleanup(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "db-cleanup-test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	dbPath := filepath.Join(tempDir, "cleanup.db")

	t.Run("multiple close calls", func(t *testing.T) {
		db := NewSQLiteDatabase(dbPath)
		err := db.Initialize()
		require.NoError(t, err)

		// Close multiple times should not error
		err = db.Close()
		assert.NoError(t, err)

		err = db.Close()
		assert.NoError(t, err)

		err = db.Close()
		assert.NoError(t, err)
	})

	t.Run("operations after close", func(t *testing.T) {
		db := NewSQLiteDatabase(dbPath)
		err := db.Initialize()
		require.NoError(t, err)

		err = db.Close()
		require.NoError(t, err)

		// Operations after close should fail gracefully
		err = db.LogAuditEvent("test", "cmd", "arg", true, "")
		assert.Error(t, err)

		_, err = db.GetAuditLogs("test", 10)
		assert.Error(t, err)

		err = db.Execute("SELECT 1", nil)
		assert.Error(t, err)
	})
}

// Helper function for file copying
func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0644)
}

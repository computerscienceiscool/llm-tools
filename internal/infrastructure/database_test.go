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
		// Get all logs for session1
		logs, err := db.GetAuditLogs("session1", 100)
		assert.NoError(t, err)
		assert.Len(t, logs, 2)

		// Verify log contents
		assert.Equal(t, "session1", logs[0].SessionID)
		assert.Equal(t, "open", logs[0].Command)
		assert.Equal(t, "file1.txt", logs[0].Argument)
		assert.True(t, logs[0].Success)
		assert.Empty(t, logs[0].ErrorMsg)

		assert.Equal(t, "session1", logs[1].SessionID)
		assert.Equal(t, "write", logs[1].Command)
		assert.Equal(t, "file2.txt", logs[1].Argument)
		assert.True(t, logs[1].Success)
		assert.Empty(t, logs[1].ErrorMsg)

		// Get logs for session2
		logs2, err := db.GetAuditLogs("session2", 100)
		assert.NoError(t, err)
		assert.Len(t, logs2, 2)

		// Verify failed operation log
		failedLog := logs2[0] // First entry for session2
		assert.Equal(t, "session2", failedLog.SessionID)
		assert.Equal(t, "exec", failedLog.Command)
		assert.Equal(t, "go test", failedLog.Argument)
		assert.False(t, failedLog.Success)
		assert.Equal(t, "command failed", failedLog.ErrorMsg)
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

	const numGoroutines = 10
	const logsPerGoroutine = 50
	done := make(chan bool, numGoroutines)

	// Concurrent writes
	for i := 0; i < numGoroutines; i++ {
		go func(routineID int) {
			defer func() { done <- true }()

			for j := 0; j < logsPerGoroutine; j++ {
				sessionID := fmt.Sprintf("session%d", routineID)
				argument := fmt.Sprintf("file%d_%d.txt", routineID, j)

				err := db.LogAuditEvent(sessionID, "test", argument, true, "")
				assert.NoError(t, err)
			}
		}(i)
	}

	// Wait for all writes to complete
	for i := 0; i < numGoroutines; i++ {
		<-done
	}

	// Verify all logs were written
	for i := 0; i < numGoroutines; i++ {
		sessionID := fmt.Sprintf("session%d", i)
		logs, err := db.GetAuditLogs(sessionID, 100)
		assert.NoError(t, err)
		assert.Len(t, logs, logsPerGoroutine)
	}
}

// TestSQLiteDatabaseErrors tests error handling
func TestSQLiteDatabaseErrors(t *testing.T) {
	t.Run("invalid database path", func(t *testing.T) {
		// Try to create database in non-existent directory
		db := NewSQLiteDatabase("/nonexistent/path/test.db")
		err := db.Initialize()
		assert.Error(t, err)
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

		_, err = db.GetAuditLogs("test", 10)
		assert.Error(t, err)
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

// Placeholder interfaces and types
type Database interface {
	LogAuditEvent(sessionID, command, argument string, success bool, errorMsg string) error
	GetAuditLogs(sessionID string, limit int) ([]AuditLog, error)
	Initialize() error
	Close() error
}

type AuditLog struct {
	ID        int64
	Timestamp time.Time
	SessionID string
	Command   string
	Argument  string
	Success   bool
	ErrorMsg  string
}

func NewSQLiteDatabase(path string) Database {
	return &sqliteDatabase{path: path}
}

type sqliteDatabase struct {
	path string
	db   *sql.DB
}

func (s *sqliteDatabase) Initialize() error {
	// Mock implementation
	return fmt.Errorf("mock implementation")
}

func (s *sqliteDatabase) LogAuditEvent(sessionID, command, argument string, success bool, errorMsg string) error {
	return fmt.Errorf("mock implementation")
}

func (s *sqliteDatabase) GetAuditLogs(sessionID string, limit int) ([]AuditLog, error) {
	return nil, fmt.Errorf("mock implementation")
}

func (s *sqliteDatabase) Close() error {
	return fmt.Errorf("mock implementation")
}

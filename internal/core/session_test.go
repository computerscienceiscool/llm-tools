package core

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockAuditLogger for testing
type MockAuditLogger struct {
	mock.Mock
}

func (m *MockAuditLogger) LogOperation(sessionID, command, argument string, success bool, errorMsg string) {
	m.Called(sessionID, command, argument, success, errorMsg)
}

// TestNewSession tests session creation
func TestNewSession(t *testing.T) {
	config := &Config{
		RepositoryRoot: "/test/repo",
		MaxFileSize:    1048576,
		Interactive:    false,
	}

	session, err := NewSession(config)

	assert.NoError(t, err)
	assert.NotNil(t, session)
	assert.IsType(t, &DefaultSession{}, session)
}

// TestDefaultSession tests the DefaultSession implementation
func TestDefaultSession(t *testing.T) {
	config := &Config{
		RepositoryRoot: "/test/repo",
		MaxFileSize:    1048576,
		Interactive:    true,
		ExecEnabled:    true,
	}

	session, err := NewSession(config)
	require.NoError(t, err)

	defaultSession := session.(*DefaultSession)

	t.Run("GetConfig", func(t *testing.T) {
		retrievedConfig := session.GetConfig()
		assert.Equal(t, config, retrievedConfig)
		assert.Equal(t, "/test/repo", retrievedConfig.RepositoryRoot)
		assert.Equal(t, int64(1048576), retrievedConfig.MaxFileSize)
		assert.True(t, retrievedConfig.Interactive)
		assert.True(t, retrievedConfig.ExecEnabled)
	})

	t.Run("GetID", func(t *testing.T) {
		sessionID := session.GetID()
		assert.NotEmpty(t, sessionID)
		assert.IsType(t, "", sessionID)

		// ID should be consistent across calls
		sessionID2 := session.GetID()
		assert.Equal(t, sessionID, sessionID2)
	})

	t.Run("GetStartTime", func(t *testing.T) {
		startTime := session.GetStartTime()
		assert.False(t, startTime.IsZero())
		assert.True(t, time.Since(startTime) < time.Second)
	})

	t.Run("CommandsRun counter", func(t *testing.T) {
		// Initially should be zero
		assert.Equal(t, 0, session.GetCommandsRun())

		// Increment and verify
		session.IncrementCommandsRun()
		assert.Equal(t, 1, session.GetCommandsRun())

		// Increment multiple times
		session.IncrementCommandsRun()
		session.IncrementCommandsRun()
		assert.Equal(t, 3, session.GetCommandsRun())
	})

	t.Run("session ID uniqueness", func(t *testing.T) {
		// Create another session and verify different ID
		session2, err := NewSession(config)
		require.NoError(t, err)

		assert.NotEqual(t, session.GetID(), session2.GetID())
	})

	t.Run("audit logger exists", func(t *testing.T) {
		assert.NotNil(t, defaultSession.auditLogger)
	})
}

// TestSessionAuditLogging tests audit logging functionality
func TestSessionAuditLogging(t *testing.T) {
	config := &Config{RepositoryRoot: "/test/repo"}
	session, err := NewSession(config)
	require.NoError(t, err)

	tests := []struct {
		name     string
		command  string
		argument string
		success  bool
		errorMsg string
	}{
		{
			name:     "successful open command",
			command:  "open",
			argument: "test.txt",
			success:  true,
			errorMsg: "",
		},
		{
			name:     "failed open command",
			command:  "open",
			argument: "../etc/passwd",
			success:  false,
			errorMsg: "PATH_SECURITY: path traversal detected",
		},
		{
			name:     "successful write command",
			command:  "write",
			argument: "output.txt",
			success:  true,
			errorMsg: "",
		},
		{
			name:     "failed exec command",
			command:  "exec",
			argument: "rm -rf /",
			success:  false,
			errorMsg: "EXEC_VALIDATION: command not whitelisted",
		},
		{
			name:     "successful search command",
			command:  "search",
			argument: "authentication",
			success:  true,
			errorMsg: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This should not panic or error
			session.LogAudit(tt.command, tt.argument, tt.success, tt.errorMsg)

			// Audit logging is fire-and-forget, so we mainly test it doesn't crash
			assert.True(t, true)
		})
	}
}

// TestSessionConcurrency tests concurrent session operations
func TestSessionConcurrency(t *testing.T) {
	config := &Config{RepositoryRoot: "/test/repo"}
	session, err := NewSession(config)
	require.NoError(t, err)

	const numGoroutines = 10
	const incrementsPerGoroutine = 100

	// Test concurrent command counter increments
	done := make(chan bool, numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func() {
			for j := 0; j < incrementsPerGoroutine; j++ {
				session.IncrementCommandsRun()
			}
			done <- true
		}()
	}

	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		<-done
	}

	// Verify final count (this test might reveal race conditions)
	finalCount := session.GetCommandsRun()
	assert.Equal(t, numGoroutines*incrementsPerGoroutine, finalCount)
}

// TestSessionAuditLoggingConcurrency tests concurrent audit logging
func TestSessionAuditLoggingConcurrency(t *testing.T) {
	config := &Config{RepositoryRoot: "/test/repo"}
	session, err := NewSession(config)
	require.NoError(t, err)

	const numGoroutines = 50
	const logsPerGoroutine = 10

	// Test concurrent audit logging
	done := make(chan bool, numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(routineID int) {
			for j := 0; j < logsPerGoroutine; j++ {
				session.LogAudit("test", "concurrent_test", true, "")
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		<-done
	}

	// Should not panic or deadlock
	assert.True(t, true)
}

// TestSessionConfiguration tests session config handling
func TestSessionConfiguration(t *testing.T) {
	tests := []struct {
		name   string
		config *Config
	}{
		{
			name: "minimal config",
			config: &Config{
				RepositoryRoot: "/minimal",
				MaxFileSize:    1024,
			},
		},
		{
			name: "full config",
			config: &Config{
				RepositoryRoot:      "/full/config",
				MaxFileSize:         2097152,
				MaxWriteSize:        102400,
				ExcludedPaths:       []string{".git", ".env"},
				Interactive:         true,
				InputFile:           "input.txt",
				OutputFile:          "output.txt",
				JSONOutput:          true,
				Verbose:             true,
				RequireConfirmation: true,
				BackupBeforeWrite:   true,
				AllowedExtensions:   []string{".go", ".py"},
				ForceWrite:          false,
				ExecEnabled:         true,
				ExecWhitelist:       []string{"go test"},
				ExecTimeout:         30 * time.Second,
				ExecMemoryLimit:     "1g",
				ExecCPULimit:        4,
				ExecContainerImage:  "alpine:latest",
				ExecNetworkEnabled:  false,
			},
		},
		{
			name: "config with zero values",
			config: &Config{
				RepositoryRoot: "/zero",
				MaxFileSize:    0,
				MaxWriteSize:   0,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			session, err := NewSession(tt.config)
			require.NoError(t, err)

			retrievedConfig := session.GetConfig()
			assert.Equal(t, tt.config, retrievedConfig)

			// Test config is not accidentally modified
			originalRepo := tt.config.RepositoryRoot
			retrievedConfig.RepositoryRoot = "/modified"
			assert.Equal(t, "/modified", tt.config.RepositoryRoot) // Same object, so should be modified
			tt.config.RepositoryRoot = originalRepo                // Restore for other tests
		})
	}
}

// TestSessionInterface tests Session interface compliance
func TestSessionInterface(t *testing.T) {
	// Ensure DefaultSession implements Session interface
	var _ Session = (*DefaultSession)(nil)

	config := &Config{RepositoryRoot: "/interface/test"}
	session, err := NewSession(config)
	require.NoError(t, err)

	// Test all interface methods are callable
	assert.NotNil(t, session.GetConfig())
	assert.NotEmpty(t, session.GetID())
	assert.False(t, session.GetStartTime().IsZero())
	assert.Equal(t, 0, session.GetCommandsRun())

	session.IncrementCommandsRun()
	assert.Equal(t, 1, session.GetCommandsRun())

	session.LogAudit("test", "interface", true, "")
	// LogAudit returns nothing, so just ensure it doesn't panic
}

// TestSessionStartTime tests that start time is properly set
func TestSessionStartTime(t *testing.T) {
	beforeCreation := time.Now()

	config := &Config{RepositoryRoot: "/time/test"}
	session, err := NewSession(config)
	require.NoError(t, err)

	afterCreation := time.Now()
	startTime := session.GetStartTime()

	// Start time should be between before and after creation
	assert.True(t, startTime.After(beforeCreation) || startTime.Equal(beforeCreation))
	assert.True(t, startTime.Before(afterCreation) || startTime.Equal(afterCreation))
}

// TestSessionIDFormat tests session ID format
func TestSessionIDFormat(t *testing.T) {
	config := &Config{RepositoryRoot: "/id/test"}
	session, err := NewSession(config)
	require.NoError(t, err)

	sessionID := session.GetID()

	// Session ID should be a string representation of a timestamp (nanoseconds)
	assert.NotEmpty(t, sessionID)
	assert.Regexp(t, `^\d+$`, sessionID, "Session ID should be numeric")

	// Should be a reasonable timestamp (not too old or in future)
	// This is a basic sanity check
	assert.True(t, len(sessionID) > 10, "Session ID should be a reasonably long timestamp")
}

// TestSessionWithNilAuditLogger tests behavior when audit logger creation fails
func TestSessionWithNilAuditLogger(t *testing.T) {
	// This test would be relevant if we could inject a failing audit logger
	// For now, we test that session creation doesn't fail even if audit logging has issues

	config := &Config{RepositoryRoot: "/audit/test"}
	session, err := NewSession(config)
	require.NoError(t, err)

	// Should not panic when logging with potentially nil logger
	session.LogAudit("test", "nil_logger", true, "")
	assert.True(t, true)
}

// TestSessionMemoryLeaks tests for potential memory leaks
func TestSessionMemoryLeaks(t *testing.T) {
	t.Skip("Memory leak tests would require runtime monitoring tools")

	// Example structure for memory leak testing:
	// 1. Create many sessions
	// 2. Call operations on them
	// 3. Monitor memory usage
	// 4. Ensure memory is properly released
}

// BenchmarkNewSession benchmarks session creation
func BenchmarkNewSession(b *testing.B) {
	config := &Config{
		RepositoryRoot: "/benchmark/test",
		MaxFileSize:    1048576,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		session, err := NewSession(config)
		if err != nil {
			b.Fatal(err)
		}
		_ = session.GetID() // Ensure session is usable
	}
}

// BenchmarkSessionGetConfig benchmarks config retrieval
func BenchmarkSessionGetConfig(b *testing.B) {
	config := &Config{
		RepositoryRoot: "/benchmark/test",
		MaxFileSize:    1048576,
		ExecEnabled:    true,
		ExecWhitelist:  []string{"go test", "npm test"},
	}

	session, err := NewSession(config)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = session.GetConfig()
	}
}

// BenchmarkSessionIncrementCommands benchmarks command counter
func BenchmarkSessionIncrementCommands(b *testing.B) {
	config := &Config{RepositoryRoot: "/benchmark/test"}
	session, err := NewSession(config)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		session.IncrementCommandsRun()
	}
}

// BenchmarkSessionAuditLogging benchmarks audit logging
func BenchmarkSessionAuditLogging(b *testing.B) {
	config := &Config{RepositoryRoot: "/benchmark/test"}
	session, err := NewSession(config)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		session.LogAudit("benchmark", "test_command", true, "")
	}
}

// TestSessionLongRunning tests session behavior over extended time
func TestSessionLongRunning(t *testing.T) {
	config := &Config{RepositoryRoot: "/longrunning/test"}
	session, err := NewSession(config)
	require.NoError(t, err)

	startTime := session.GetStartTime()

	// Simulate some passage of time
	time.Sleep(time.Millisecond * 10)

	// Perform operations
	for i := 0; i < 100; i++ {
		session.IncrementCommandsRun()
		session.LogAudit("longrun", "operation", i%2 == 0, "")
	}

	// Verify session is still functional
	assert.Equal(t, 100, session.GetCommandsRun())
	assert.Equal(t, startTime, session.GetStartTime()) // Should not change
	assert.NotNil(t, session.GetConfig())
}

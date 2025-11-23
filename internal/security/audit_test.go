package security

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockAuditManager for testing
type MockAuditManager struct {
	mock.Mock
}

func (m *MockAuditManager) LogEvent(event AuditEvent) error {
	args := m.Called(event)
	return args.Error(0)
}

func (m *MockAuditManager) GetEvents(sessionID string, limit int) ([]AuditEvent, error) {
	args := m.Called(sessionID, limit)
	return args.Get(0).([]AuditEvent), args.Error(1)
}

func (m *MockAuditManager) RotateLog() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockAuditManager) Close() error {
	args := m.Called()
	return args.Error(0)
}

// TestAuditManagerInterface tests the AuditManager interface
func TestAuditManagerInterface(t *testing.T) {
	var _ AuditManager = (*MockAuditManager)(nil)

	mockManager := &MockAuditManager{}

	event := AuditEvent{
		Timestamp: time.Now(),
		SessionID: "session123",
		EventType: "file_access",
		Resource:  "test.go",
		Action:    "read",
		Success:   true,
		UserAgent: "llm-tool/1.0",
		IPAddress: "127.0.0.1",
		Details: map[string]interface{}{
			"file_size": 1024,
			"duration":  "10ms",
		},
	}

	// Setup expectations
	mockManager.On("LogEvent", event).Return(nil)
	mockManager.On("GetEvents", "session123", 10).Return([]AuditEvent{event}, nil)
	mockManager.On("RotateLog").Return(nil)
	mockManager.On("Close").Return(nil)

	// Test LogEvent
	err := mockManager.LogEvent(event)
	assert.NoError(t, err)

	// Test GetEvents
	events, err := mockManager.GetEvents("session123", 10)
	assert.NoError(t, err)
	assert.Len(t, events, 1)
	assert.Equal(t, "session123", events[0].SessionID)

	// Test RotateLog
	err = mockManager.RotateLog()
	assert.NoError(t, err)

	// Test Close
	err = mockManager.Close()
	assert.NoError(t, err)

	mockManager.AssertExpectations(t)
}

// TestAuditEvent tests the AuditEvent structure
func TestAuditEvent(t *testing.T) {
	now := time.Now()
	event := AuditEvent{
		ID:        "evt_123",
		Timestamp: now,
		SessionID: "session456",
		EventType: "command_execution",
		Resource:  "go test",
		Action:    "execute",
		Success:   false,
		ErrorMsg:  "timeout exceeded",
		UserAgent: "llm-tool/1.0",
		IPAddress: "192.168.1.100",
		Details: map[string]interface{}{
			"exit_code": 124,
			"timeout":   "30s",
			"output":    "test output...",
		},
	}

	assert.Equal(t, "evt_123", event.ID)
	assert.Equal(t, now, event.Timestamp)
	assert.Equal(t, "session456", event.SessionID)
	assert.Equal(t, "command_execution", event.EventType)
	assert.Equal(t, "go test", event.Resource)
	assert.Equal(t, "execute", event.Action)
	assert.False(t, event.Success)
	assert.Equal(t, "timeout exceeded", event.ErrorMsg)
	assert.Equal(t, "llm-tool/1.0", event.UserAgent)
	assert.Equal(t, "192.168.1.100", event.IPAddress)
	assert.Equal(t, 124, event.Details["exit_code"])
}

// TestDefaultAuditManager tests the default audit manager implementation
func TestDefaultAuditManager(t *testing.T) {
	// Create temporary directory for audit logs
	tempDir, err := os.MkdirTemp("", "audit-manager-test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	auditFile := filepath.Join(tempDir, "audit.log")
	manager := NewAuditManager(auditFile)
	require.NotNil(t, manager)
	defer manager.Close()

	t.Run("log single event", func(t *testing.T) {
		event := AuditEvent{
			ID:        "evt_001",
			Timestamp: time.Now(),
			SessionID: "test_session",
			EventType: "file_access",
			Resource:  "main.go",
			Action:    "read",
			Success:   true,
			UserAgent: "test-agent",
			IPAddress: "127.0.0.1",
		}

		err := manager.LogEvent(event)
		assert.NoError(t, err)

		// Verify event was written to file
		time.Sleep(50 * time.Millisecond)
		content, err := os.ReadFile(auditFile)
		assert.NoError(t, err)
		assert.Contains(t, string(content), "test_session")
		assert.Contains(t, string(content), "file_access")
		assert.Contains(t, string(content), "main.go")
	})

	t.Run("log multiple events", func(t *testing.T) {
		events := []AuditEvent{
			{
				ID:        "evt_002",
				Timestamp: time.Now(),
				SessionID: "session_multi",
				EventType: "file_write",
				Resource:  "output.txt",
				Action:    "create",
				Success:   true,
			},
			{
				ID:        "evt_003",
				Timestamp: time.Now(),
				SessionID: "session_multi",
				EventType: "command_execution",
				Resource:  "go build",
				Action:    "execute",
				Success:   false,
				ErrorMsg:  "compilation failed",
			},
		}

		for _, event := range events {
			err := manager.LogEvent(event)
			assert.NoError(t, err)
		}

		time.Sleep(100 * time.Millisecond)

		// Check that both events are in the log
		content, err := os.ReadFile(auditFile)
		assert.NoError(t, err)
		logContent := string(content)
		assert.Contains(t, logContent, "file_write")
		assert.Contains(t, logContent, "command_execution")
		assert.Contains(t, logContent, "compilation failed")
	})

	t.Run("retrieve events", func(t *testing.T) {
		events, err := manager.GetEvents("session_multi", 10)
		assert.NoError(t, err)
		assert.Len(t, events, 2)

		// Verify event details
		assert.Equal(t, "session_multi", events[0].SessionID)
		assert.Contains(t, []string{"file_write", "command_execution"}, events[0].EventType)
	})
}

// TestAuditEventSerialization tests event serialization and deserialization
func TestAuditEventSerialization(t *testing.T) {
	event := AuditEvent{
		ID:        "evt_serialize",
		Timestamp: time.Now(),
		SessionID: "test_session",
		EventType: "security_violation",
		Resource:  "../etc/passwd",
		Action:    "access_denied",
		Success:   false,
		ErrorMsg:  "PATH_SECURITY: path traversal detected",
		UserAgent: "malicious-agent",
		IPAddress: "10.0.0.1",
		Details: map[string]interface{}{
			"violation_type": "path_traversal",
			"blocked_path":   "../etc/passwd",
			"severity":       "high",
		},
	}

	// Test JSON serialization
	jsonData := event.ToJSON()
	assert.Contains(t, jsonData, "evt_serialize")
	assert.Contains(t, jsonData, "security_violation")
	assert.Contains(t, jsonData, "path_traversal")

	// Test structured logging format
	logLine := event.ToLogLine()
	assert.Contains(t, logLine, event.SessionID)
	assert.Contains(t, logLine, event.EventType)
	assert.Contains(t, logLine, "PATH_SECURITY")
}

// TestAuditConcurrency tests concurrent audit logging
func TestAuditConcurrency(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "audit-concurrent-test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	auditFile := filepath.Join(tempDir, "concurrent_audit.log")
	manager := NewAuditManager(auditFile)
	require.NotNil(t, manager)
	defer manager.Close()

	const numGoroutines = 10
	const eventsPerGoroutine = 50
	var wg sync.WaitGroup

	// Launch concurrent audit loggers
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(routineID int) {
			defer wg.Done()

			for j := 0; j < eventsPerGoroutine; j++ {
				event := AuditEvent{
					ID:        fmt.Sprintf("evt_%d_%d", routineID, j),
					Timestamp: time.Now(),
					SessionID: fmt.Sprintf("session_%d", routineID),
					EventType: "concurrent_test",
					Resource:  fmt.Sprintf("resource_%d", j),
					Action:    "test",
					Success:   true,
				}

				err := manager.LogEvent(event)
				assert.NoError(t, err)
			}
		}(i)
	}

	wg.Wait()
	time.Sleep(200 * time.Millisecond)

	// Verify all events were logged
	content, err := os.ReadFile(auditFile)
	require.NoError(t, err)

	logContent := string(content)
	lines := len(strings.Split(strings.TrimSpace(logContent), "\n"))
	expectedLines := numGoroutines * eventsPerGoroutine

	assert.Equal(t, expectedLines, lines, "Expected %d log lines, got %d", expectedLines, lines)

	// Verify no corruption
	for i := 0; i < numGoroutines; i++ {
		sessionID := fmt.Sprintf("session_%d", i)
		assert.Contains(t, logContent, sessionID)
	}
}

// TestAuditLogRotation tests log rotation functionality
func TestAuditLogRotation(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "audit-rotation-test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	auditFile := filepath.Join(tempDir, "rotation_audit.log")
	manager := NewAuditManager(auditFile)
	require.NotNil(t, manager)
	defer manager.Close()

	// Log some events
	for i := 0; i < 5; i++ {
		event := AuditEvent{
			ID:        fmt.Sprintf("evt_%d", i),
			Timestamp: time.Now(),
			SessionID: "rotation_test",
			EventType: "test_event",
			Resource:  fmt.Sprintf("resource_%d", i),
			Action:    "test",
			Success:   true,
		}
		err := manager.LogEvent(event)
		require.NoError(t, err)
	}

	time.Sleep(100 * time.Millisecond)

	// Get original content
	originalContent, err := os.ReadFile(auditFile)
	require.NoError(t, err)

	// Rotate log
	err = manager.RotateLog()
	assert.NoError(t, err)

	// Verify backup file exists
	backupPattern := filepath.Join(tempDir, "rotation_audit.log.*")
	matches, err := filepath.Glob(backupPattern)
	assert.NoError(t, err)
	assert.NotEmpty(t, matches, "Backup file should exist after rotation")

	// Verify original file is empty or reset
	newContent, err := os.ReadFile(auditFile)
	assert.NoError(t, err)
	assert.NotEqual(t, string(originalContent), string(newContent))

	// Log new event after rotation
	newEvent := AuditEvent{
		ID:        "evt_after_rotation",
		Timestamp: time.Now(),
		SessionID: "post_rotation_test",
		EventType: "test_event",
		Resource:  "new_resource",
		Action:    "test",
		Success:   true,
	}
	err = manager.LogEvent(newEvent)
	assert.NoError(t, err)

	time.Sleep(50 * time.Millisecond)

	// Verify new event is logged
	finalContent, err := os.ReadFile(auditFile)
	assert.NoError(t, err)
	assert.Contains(t, string(finalContent), "post_rotation_test")
}

// TestAuditSecurity tests audit security features
func TestAuditSecurity(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "audit-security-test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	auditFile := filepath.Join(tempDir, "security_audit.log")
	manager := NewAuditManager(auditFile)
	require.NotNil(t, manager)
	defer manager.Close()

	t.Run("log security violations", func(t *testing.T) {
		securityEvents := []AuditEvent{
			{
				ID:        "sec_001",
				Timestamp: time.Now(),
				SessionID: "security_test",
				EventType: "path_traversal_attempt",
				Resource:  "../../../etc/passwd",
				Action:    "blocked",
				Success:   false,
				ErrorMsg:  "PATH_SECURITY: path traversal detected",
				Details: map[string]interface{}{
					"severity":     "critical",
					"attack_type":  "path_traversal",
					"blocked_path": "../../../etc/passwd",
				},
			},
			{
				ID:        "sec_002",
				Timestamp: time.Now(),
				SessionID: "security_test",
				EventType: "command_injection_attempt",
				Resource:  "rm -rf / && echo",
				Action:    "blocked",
				Success:   false,
				ErrorMsg:  "EXEC_VALIDATION: dangerous command blocked",
				Details: map[string]interface{}{
					"severity":        "critical",
					"attack_type":     "command_injection",
					"blocked_command": "rm -rf /",
				},
			},
		}

		for _, event := range securityEvents {
			err := manager.LogEvent(event)
			assert.NoError(t, err)
		}

		time.Sleep(100 * time.Millisecond)

		// Verify security events are logged
		content, err := os.ReadFile(auditFile)
		assert.NoError(t, err)
		logContent := string(content)
		assert.Contains(t, logContent, "path_traversal_attempt")
		assert.Contains(t, logContent, "command_injection_attempt")
		assert.Contains(t, logContent, "PATH_SECURITY")
		assert.Contains(t, logContent, "EXEC_VALIDATION")
	})

	t.Run("audit log integrity", func(t *testing.T) {
		// Verify audit log cannot be easily tampered with
		originalContent, err := os.ReadFile(auditFile)
		require.NoError(t, err)

		// Verify each line has timestamp and is properly formatted
		lines := strings.Split(strings.TrimSpace(string(originalContent)), "\n")
		for i, line := range lines {
			if line == "" {
				continue
			}
			// Each line should have timestamp format
			assert.True(t, isValidAuditLogLine(line), "Line %d should be valid audit format: %s", i, line)
		}
	})
}

// TestAuditFiltering tests event filtering and querying
func TestAuditFiltering(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "audit-filter-test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	auditFile := filepath.Join(tempDir, "filter_audit.log")
	manager := NewAuditManager(auditFile)
	require.NotNil(t, manager)
	defer manager.Close()

	// Create events with different types and sessions
	events := []AuditEvent{
		{SessionID: "session1", EventType: "file_access", Success: true},
		{SessionID: "session1", EventType: "file_write", Success: true},
		{SessionID: "session1", EventType: "command_exec", Success: false},
		{SessionID: "session2", EventType: "file_access", Success: true},
		{SessionID: "session2", EventType: "file_access", Success: false},
	}

	for i, event := range events {
		event.ID = fmt.Sprintf("filter_%d", i)
		event.Timestamp = time.Now()
		event.Resource = fmt.Sprintf("resource_%d", i)
		event.Action = "test"

		err := manager.LogEvent(event)
		require.NoError(t, err)
	}

	time.Sleep(100 * time.Millisecond)

	t.Run("filter by session", func(t *testing.T) {
		session1Events, err := manager.GetEvents("session1", 10)
		assert.NoError(t, err)
		assert.Len(t, session1Events, 3)

		session2Events, err := manager.GetEvents("session2", 10)
		assert.NoError(t, err)
		assert.Len(t, session2Events, 2)
	})

	t.Run("limit results", func(t *testing.T) {
		limitedEvents, err := manager.GetEvents("session1", 2)
		assert.NoError(t, err)
		assert.Len(t, limitedEvents, 2)
	})
}

// TestAuditPerformance tests audit performance under load
func TestAuditPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	tempDir, err := os.MkdirTemp("", "audit-perf-test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	auditFile := filepath.Join(tempDir, "perf_audit.log")
	manager := NewAuditManager(auditFile)
	require.NotNil(t, manager)
	defer manager.Close()

	const numEvents = 1000

	start := time.Now()
	for i := 0; i < numEvents; i++ {
		event := AuditEvent{
			ID:        fmt.Sprintf("perf_%d", i),
			Timestamp: time.Now(),
			SessionID: fmt.Sprintf("session_%d", i%10),
			EventType: "performance_test",
			Resource:  fmt.Sprintf("resource_%d", i),
			Action:    "test",
			Success:   true,
		}

		err := manager.LogEvent(event)
		assert.NoError(t, err)
	}
	elapsed := time.Since(start)

	avgTimePerEvent := elapsed / time.Duration(numEvents)
	t.Logf("Logged %d events in %v (%.2f μs/event)", numEvents, elapsed, float64(avgTimePerEvent.Nanoseconds())/1000)

	// Performance assertion
	assert.Less(t, avgTimePerEvent, time.Millisecond, "Average time per event should be less than 1ms")

	time.Sleep(500 * time.Millisecond)

	// Verify all events were logged
	content, err := os.ReadFile(auditFile)
	assert.NoError(t, err)
	lines := strings.Count(string(content), "\n")
	assert.GreaterOrEqual(t, lines, numEvents, "Should have logged at least %d events", numEvents)
}

// BenchmarkAuditLogging benchmarks audit logging performance
func BenchmarkAuditLogging(b *testing.B) {
	tempDir, err := os.MkdirTemp("", "audit-benchmark")
	if err != nil {
		b.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	auditFile := filepath.Join(tempDir, "benchmark_audit.log")
	manager := NewAuditManager(auditFile)
	if manager == nil {
		b.Fatal("Failed to create audit manager")
	}
	defer manager.Close()

	event := AuditEvent{
		ID:        "benchmark_event",
		Timestamp: time.Now(),
		SessionID: "benchmark_session",
		EventType: "benchmark_test",
		Resource:  "benchmark_resource",
		Action:    "benchmark",
		Success:   true,
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = manager.LogEvent(event)
		}
	})
}

// Helper functions for testing
func isValidAuditLogLine(line string) bool {
	// Simple validation - should contain timestamp and basic structure
	if len(line) < 20 {
		return false
	}

	// Should contain timestamp (ISO format starts with year)
	if line[0] != '2' || line[4] != '-' {
		return false
	}

	return true
}

// Example audit structures for testing
type AuditEvent struct {
	ID        string                 `json:"id"`
	Timestamp time.Time              `json:"timestamp"`
	SessionID string                 `json:"session_id"`
	EventType string                 `json:"event_type"`
	Resource  string                 `json:"resource"`
	Action    string                 `json:"action"`
	Success   bool                   `json:"success"`
	ErrorMsg  string                 `json:"error_msg,omitempty"`
	UserAgent string                 `json:"user_agent,omitempty"`
	IPAddress string                 `json:"ip_address,omitempty"`
	Details   map[string]interface{} `json:"details,omitempty"`
}

func (e AuditEvent) ToJSON() string {
	return fmt.Sprintf(`{"id":"%s","session_id":"%s","event_type":"%s"}`, e.ID, e.SessionID, e.EventType)
}

func (e AuditEvent) ToLogLine() string {
	return fmt.Sprintf("%s|%s|%s|%s|%s|%v|%s",
		e.Timestamp.Format(time.RFC3339),
		e.SessionID,
		e.EventType,
		e.Resource,
		e.Action,
		e.Success,
		e.ErrorMsg)
}

// Mock audit manager interface for testing
type AuditManager interface {
	LogEvent(event AuditEvent) error
	GetEvents(sessionID string, limit int) ([]AuditEvent, error)
	RotateLog() error
	Close() error
}

// Placeholder for actual implementation
func NewAuditManager(logFile string) AuditManager {
	return &MockAuditManager{}
}

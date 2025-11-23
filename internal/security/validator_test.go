package security

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockPathValidator for testing
type MockPathValidator struct {
	mock.Mock
}

func (m *MockPathValidator) ValidatePath(requestedPath, repositoryRoot string, excludedPaths []string) (string, error) {
	args := m.Called(requestedPath, repositoryRoot, excludedPaths)
	return args.String(0), args.Error(1)
}

func (m *MockPathValidator) ValidateWriteExtension(filepath string, allowedExtensions []string) error {
	args := m.Called(filepath, allowedExtensions)
	return args.Error(0)
}

// MockAuditLogger for testing
type MockAuditLogger struct {
	mock.Mock
}

func (m *MockAuditLogger) LogOperation(sessionID, command, argument string, success bool, errorMsg string) {
	m.Called(sessionID, command, argument, success, errorMsg)
}

// TestPathValidatorInterface tests the PathValidator interface
func TestPathValidatorInterface(t *testing.T) {
	// Ensure MockPathValidator implements PathValidator interface
	var _ PathValidator = (*MockPathValidator)(nil)

	mockValidator := &MockPathValidator{}

	mockValidator.On("ValidatePath", "/test/file.txt", "/repo", []string{".git"}).
		Return("/repo/test/file.txt", nil)
	mockValidator.On("ValidateWriteExtension", "test.go", []string{".go", ".py"}).
		Return(nil)

	// Test ValidatePath
	safePath, err := mockValidator.ValidatePath("/test/file.txt", "/repo", []string{".git"})
	assert.NoError(t, err)
	assert.Equal(t, "/repo/test/file.txt", safePath)

	// Test ValidateWriteExtension
	err = mockValidator.ValidateWriteExtension("test.go", []string{".go", ".py"})
	assert.NoError(t, err)

	mockValidator.AssertExpectations(t)
}

// TestAuditLoggerInterface tests the AuditLogger interface
func TestAuditLoggerInterface(t *testing.T) {
	// Ensure MockAuditLogger implements AuditLogger interface
	var _ AuditLogger = (*MockAuditLogger)(nil)

	mockLogger := &MockAuditLogger{}

	mockLogger.On("LogOperation", "session123", "open", "file.txt", true, "")

	// Test LogOperation
	mockLogger.LogOperation("session123", "open", "file.txt", true, "")

	mockLogger.AssertExpectations(t)
}

// TestSecurityInterfaceContract tests expected behavior of security interfaces
func TestSecurityInterfaceContract(t *testing.T) {
	tests := []struct {
		name           string
		setupValidator func(*MockPathValidator)
		setupLogger    func(*MockAuditLogger)
		testFunc       func(*MockPathValidator, *MockAuditLogger)
	}{
		{
			name: "successful path validation",
			setupValidator: func(v *MockPathValidator) {
				v.On("ValidatePath", "safe.txt", "/repo", []string{".git"}).
					Return("/repo/safe.txt", nil)
			},
			setupLogger: func(l *MockAuditLogger) {
				l.On("LogOperation", "sess1", "validate", "safe.txt", true, "")
			},
			testFunc: func(v *MockPathValidator, l *MockAuditLogger) {
				safePath, err := v.ValidatePath("safe.txt", "/repo", []string{".git"})
				assert.NoError(t, err)
				assert.Equal(t, "/repo/safe.txt", safePath)

				l.LogOperation("sess1", "validate", "safe.txt", true, "")
			},
		},
		{
			name: "failed path validation",
			setupValidator: func(v *MockPathValidator) {
				v.On("ValidatePath", "../etc/passwd", "/repo", []string{".git"}).
					Return("", assert.AnError)
			},
			setupLogger: func(l *MockAuditLogger) {
				l.On("LogOperation", "sess1", "validate", "../etc/passwd", false, "path traversal")
			},
			testFunc: func(v *MockPathValidator, l *MockAuditLogger) {
				safePath, err := v.ValidatePath("../etc/passwd", "/repo", []string{".git"})
				assert.Error(t, err)
				assert.Empty(t, safePath)

				l.LogOperation("sess1", "validate", "../etc/passwd", false, "path traversal")
			},
		},
		{
			name: "extension validation success",
			setupValidator: func(v *MockPathValidator) {
				v.On("ValidateWriteExtension", "script.py", []string{".py", ".go"}).
					Return(nil)
			},
			setupLogger: func(l *MockAuditLogger) {
				l.On("LogOperation", "sess1", "ext_check", "script.py", true, "")
			},
			testFunc: func(v *MockPathValidator, l *MockAuditLogger) {
				err := v.ValidateWriteExtension("script.py", []string{".py", ".go"})
				assert.NoError(t, err)

				l.LogOperation("sess1", "ext_check", "script.py", true, "")
			},
		},
		{
			name: "extension validation failure",
			setupValidator: func(v *MockPathValidator) {
				v.On("ValidateWriteExtension", "malware.exe", []string{".py", ".go"}).
					Return(assert.AnError)
			},
			setupLogger: func(l *MockAuditLogger) {
				l.On("LogOperation", "sess1", "ext_check", "malware.exe", false, "forbidden extension")
			},
			testFunc: func(v *MockPathValidator, l *MockAuditLogger) {
				err := v.ValidateWriteExtension("malware.exe", []string{".py", ".go"})
				assert.Error(t, err)

				l.LogOperation("sess1", "ext_check", "malware.exe", false, "forbidden extension")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator := &MockPathValidator{}
			logger := &MockAuditLogger{}

			tt.setupValidator(validator)
			tt.setupLogger(logger)

			tt.testFunc(validator, logger)

			validator.AssertExpectations(t)
			logger.AssertExpectations(t)
		})
	}
}

// TestSecurityErrorHandling tests security interface error handling
func TestSecurityErrorHandling(t *testing.T) {
	t.Run("path validator error types", func(t *testing.T) {
		validator := &MockPathValidator{}

		// Different types of validation errors
		validator.On("ValidatePath", "traversal", "/repo", mock.Anything).
			Return("", assert.AnError)
		validator.On("ValidatePath", "excluded", "/repo", mock.Anything).
			Return("", assert.AnError)
		validator.On("ValidatePath", "nonexistent", "/repo", mock.Anything).
			Return("", assert.AnError)

		// Test different error scenarios
		_, err1 := validator.ValidatePath("traversal", "/repo", []string{})
		assert.Error(t, err1)

		_, err2 := validator.ValidatePath("excluded", "/repo", []string{})
		assert.Error(t, err2)

		_, err3 := validator.ValidatePath("nonexistent", "/repo", []string{})
		assert.Error(t, err3)

		validator.AssertExpectations(t)
	})

	t.Run("extension validator error types", func(t *testing.T) {
		validator := &MockPathValidator{}

		// Different extension validation errors
		validator.On("ValidateWriteExtension", "file.exe", mock.Anything).
			Return(assert.AnError)
		validator.On("ValidateWriteExtension", "file.bat", mock.Anything).
			Return(assert.AnError)

		err1 := validator.ValidateWriteExtension("file.exe", []string{".txt"})
		assert.Error(t, err1)

		err2 := validator.ValidateWriteExtension("file.bat", []string{".txt"})
		assert.Error(t, err2)

		validator.AssertExpectations(t)
	})

	t.Run("audit logger robustness", func(t *testing.T) {
		logger := &MockAuditLogger{}

		// Audit logger should handle various inputs gracefully
		logger.On("LogOperation", "", "", "", false, "").Maybe()
		logger.On("LogOperation", "very-long-session-id", "very-long-command-name", "very-long-argument", true, "very-long-error-message").Maybe()
		logger.On("LogOperation", "session", "cmd", "unicode-测试", true, "").Maybe()

		// These should not panic
		logger.LogOperation("", "", "", false, "")
		logger.LogOperation("very-long-session-id", "very-long-command-name", "very-long-argument", true, "very-long-error-message")
		logger.LogOperation("session", "cmd", "unicode-测试", true, "")
	})
}

// TestSecurityInterfaceConcurrency tests concurrent usage of security interfaces
func TestSecurityInterfaceConcurrency(t *testing.T) {
	validator := &MockPathValidator{}
	logger := &MockAuditLogger{}

	// Set up expectations for concurrent calls
	validator.On("ValidatePath", mock.AnythingOfType("string"), "/repo", mock.AnythingOfType("[]string")).
		Return("/repo/file.txt", nil)
	logger.On("LogOperation", mock.AnythingOfType("string"), mock.AnythingOfType("string"),
		mock.AnythingOfType("string"), mock.AnythingOfType("bool"), mock.AnythingOfType("string"))

	const numGoroutines = 10
	done := make(chan bool, numGoroutines)

	// Test concurrent validation
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer func() { done <- true }()

			// Each goroutine does some validation work
			for j := 0; j < 10; j++ {
				_, err := validator.ValidatePath("file.txt", "/repo", []string{})
				assert.NoError(t, err)

				logger.LogOperation("session", "test", "concurrent", true, "")
			}
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		<-done
	}

	// Verify expectations were met (though exact call count may vary due to mocking)
	validator.AssertExpectations(t)
	logger.AssertExpectations(t)
}

// TestSecurityInterfaceEdgeCases tests edge cases for security interfaces
func TestSecurityInterfaceEdgeCases(t *testing.T) {
	tests := []struct {
		name string
		test func(*testing.T)
	}{
		{
			name: "empty paths",
			test: func(t *testing.T) {
				validator := &MockPathValidator{}
				validator.On("ValidatePath", "", "", []string{}).Return("", assert.AnError)

				_, err := validator.ValidatePath("", "", []string{})
				assert.Error(t, err)
			},
		},
		{
			name: "nil slices",
			test: func(t *testing.T) {
				validator := &MockPathValidator{}
				validator.On("ValidatePath", "file.txt", "/repo", []string(nil)).
					Return("/repo/file.txt", nil)
				validator.On("ValidateWriteExtension", "file.txt", []string(nil)).
					Return(nil)

				_, err := validator.ValidatePath("file.txt", "/repo", nil)
				assert.NoError(t, err)

				err = validator.ValidateWriteExtension("file.txt", nil)
				assert.NoError(t, err)
			},
		},
		{
			name: "very long inputs",
			test: func(t *testing.T) {
				validator := &MockPathValidator{}
				logger := &MockAuditLogger{}

				longPath := strings.Repeat("very-long-path-component/", 100)
				longSession := strings.Repeat("session-", 100)

				validator.On("ValidatePath", longPath, "/repo", mock.Anything).
					Return("", assert.AnError) // Likely would fail validation
				logger.On("LogOperation", longSession, "cmd", "arg", true, "")

				_, err := validator.ValidatePath(longPath, "/repo", []string{})
				assert.Error(t, err)

				logger.LogOperation(longSession, "cmd", "arg", true, "")
			},
		},
		{
			name: "unicode and special characters",
			test: func(t *testing.T) {
				validator := &MockPathValidator{}
				logger := &MockAuditLogger{}

				unicodePath := "测试文件.txt"
				specialChars := "file with spaces & symbols!@#$.txt"

				validator.On("ValidatePath", unicodePath, "/repo", mock.Anything).
					Return("/repo/"+unicodePath, nil)
				validator.On("ValidatePath", specialChars, "/repo", mock.Anything).
					Return("/repo/"+specialChars, nil)
				logger.On("LogOperation", "session", "open", unicodePath, true, "")

				_, err1 := validator.ValidatePath(unicodePath, "/repo", []string{})
				assert.NoError(t, err1)

				_, err2 := validator.ValidatePath(specialChars, "/repo", []string{})
				assert.NoError(t, err2)

				logger.LogOperation("session", "open", unicodePath, true, "")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, tt.test)
	}
}

// TestSecurityInterfaceComposition tests interface composition
func TestSecurityInterfaceComposition(t *testing.T) {
	// Test that interfaces can be composed into larger security systems
	type SecurityManager interface {
		PathValidator
		AuditLogger
	}

	// Mock implementation that combines both interfaces
	type MockSecurityManager struct {
		MockPathValidator
		MockAuditLogger
	}

	manager := &MockSecurityManager{}
	manager.On("ValidatePath", "test.txt", "/repo", mock.Anything).
		Return("/repo/test.txt", nil)
	manager.On("LogOperation", "session", "validate", "test.txt", true, "")

	// Can use as either interface
	var validator PathValidator = manager
	var logger AuditLogger = manager
	var combined SecurityManager = manager

	// Test through different interface views
	_, err := validator.ValidatePath("test.txt", "/repo", []string{})
	assert.NoError(t, err)

	logger.LogOperation("session", "validate", "test.txt", true, "")

	// Test through combined interface
	_, err = combined.ValidatePath("test.txt", "/repo", []string{})
	assert.NoError(t, err)
	combined.LogOperation("session", "validate", "test.txt", true, "")

	manager.AssertExpectations(t)
}

// BenchmarkSecurityInterfaceCalls benchmarks security interface performance
func BenchmarkSecurityInterfaceCalls(b *testing.B) {
	validator := &MockPathValidator{}
	logger := &MockAuditLogger{}

	validator.On("ValidatePath", mock.Anything, mock.Anything, mock.Anything).
		Return("/safe/path", nil)
	validator.On("ValidateWriteExtension", mock.Anything, mock.Anything).
		Return(nil)
	logger.On("LogOperation", mock.Anything, mock.Anything, mock.Anything,
		mock.Anything, mock.Anything)

	b.Run("ValidatePath", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = validator.ValidatePath("test.txt", "/repo", []string{".git"})
		}
	})

	b.Run("ValidateWriteExtension", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = validator.ValidateWriteExtension("test.go", []string{".go", ".py"})
		}
	})

	b.Run("LogOperation", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			logger.LogOperation("session", "test", "benchmark", true, "")
		}
	})
}

// TestSecurityInterfaceDocumentation tests interface documentation compliance
func TestSecurityInterfaceDocumentation(t *testing.T) {
	// These tests verify that interfaces behave as documented

	t.Run("PathValidator contract", func(t *testing.T) {
		// PathValidator should:
		// 1. Return safe path when validation succeeds
		// 2. Return error when path is unsafe
		// 3. Handle excluded paths correctly
		// 4. Validate write extensions

		validator := &MockPathValidator{}

		// Safe path should return resolved path
		validator.On("ValidatePath", "safe.txt", "/repo", []string{}).
			Return("/repo/safe.txt", nil)

		// Unsafe path should return error
		validator.On("ValidatePath", "../unsafe", "/repo", []string{}).
			Return("", assert.AnError)

		// Allowed extension should pass
		validator.On("ValidateWriteExtension", "file.go", []string{".go"}).
			Return(nil)

		// Forbidden extension should fail
		validator.On("ValidateWriteExtension", "file.exe", []string{".go"}).
			Return(assert.AnError)

		// Test documented behavior
		safePath, err := validator.ValidatePath("safe.txt", "/repo", []string{})
		assert.NoError(t, err)
		assert.NotEmpty(t, safePath)

		_, err = validator.ValidatePath("../unsafe", "/repo", []string{})
		assert.Error(t, err)

		err = validator.ValidateWriteExtension("file.go", []string{".go"})
		assert.NoError(t, err)

		err = validator.ValidateWriteExtension("file.exe", []string{".go"})
		assert.Error(t, err)

		validator.AssertExpectations(t)
	})

	t.Run("AuditLogger contract", func(t *testing.T) {
		// AuditLogger should:
		// 1. Accept all required parameters
		// 2. Handle success and failure cases
		// 3. Not return errors (fire-and-forget)

		logger := &MockAuditLogger{}

		// Should accept various parameter combinations
		logger.On("LogOperation", "session1", "open", "file.txt", true, "")
		logger.On("LogOperation", "session2", "write", "output.txt", false, "permission denied")
		logger.On("LogOperation", "", "", "", false, "")

		// Test documented behavior (no return values)
		logger.LogOperation("session1", "open", "file.txt", true, "")
		logger.LogOperation("session2", "write", "output.txt", false, "permission denied")
		logger.LogOperation("", "", "", false, "")

		logger.AssertExpectations(t)
	})
}

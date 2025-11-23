package errors

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestErrorTypes tests all custom error types
func TestErrorTypes(t *testing.T) {
	tests := []struct {
		name      string
		err       error
		errString string
	}{
		{
			name:      "ErrFileNotFound",
			err:       ErrFileNotFound,
			errString: "FILE_NOT_FOUND",
		},
		{
			name:      "ErrPathSecurity",
			err:       ErrPathSecurity,
			errString: "PATH_SECURITY",
		},
		{
			name:      "ErrResourceLimit",
			err:       ErrResourceLimit,
			errString: "RESOURCE_LIMIT",
		},
		{
			name:      "ErrPermissionDenied",
			err:       ErrPermissionDenied,
			errString: "PERMISSION_DENIED",
		},
		{
			name:      "ErrExecValidation",
			err:       ErrExecValidation,
			errString: "EXEC_VALIDATION",
		},
		{
			name:      "ErrDockerUnavailable",
			err:       ErrDockerUnavailable,
			errString: "DOCKER_UNAVAILABLE",
		},
		{
			name:      "ErrExecTimeout",
			err:       ErrExecTimeout,
			errString: "EXEC_TIMEOUT",
		},
		{
			name:      "ErrExecFailed",
			err:       ErrExecFailed,
			errString: "EXEC_FAILED",
		},
		{
			name:      "ErrSearchDisabled",
			err:       ErrSearchDisabled,
			errString: "SEARCH_DISABLED",
		},
		{
			name:      "ErrSearchInitFailed",
			err:       ErrSearchInitFailed,
			errString: "SEARCH_INIT_FAILED",
		},
		{
			name:      "ErrExtensionDenied",
			err:       ErrExtensionDenied,
			errString: "EXTENSION_DENIED",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Error(t, tt.err)
			assert.Contains(t, tt.err.Error(), tt.errString)
		})
	}
}

// TestSecurityError tests SecurityError functionality
func TestSecurityError(t *testing.T) {
	tests := []struct {
		name        string
		op          string
		path        string
		innerErr    error
		expectedMsg string
	}{
		{
			name:        "path traversal",
			op:          "path_validation",
			path:        "../etc/passwd",
			innerErr:    fmt.Errorf("path traversal detected"),
			expectedMsg: "security violation in path_validation for path ../etc/passwd: path traversal detected",
		},
		{
			name:        "excluded path",
			op:          "path_validation",
			path:        ".git/config",
			innerErr:    fmt.Errorf("path is excluded"),
			expectedMsg: "security violation in path_validation for path .git/config: path is excluded",
		},
		{
			name:        "write operation security",
			op:          "write_validation",
			path:        "malicious.exe",
			innerErr:    fmt.Errorf("forbidden extension"),
			expectedMsg: "security violation in write_validation for path malicious.exe: forbidden extension",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			secErr := &SecurityError{
				Op:   tt.op,
				Path: tt.path,
				Err:  tt.innerErr,
			}

			assert.Equal(t, tt.expectedMsg, secErr.Error())
			assert.Equal(t, tt.innerErr, secErr.Unwrap())

			// Test error type checking
			var securityErr *SecurityError
			assert.True(t, errors.As(secErr, &securityErr))
		})
	}
}

// TestValidationError tests ValidationError functionality
func TestValidationError(t *testing.T) {
	tests := []struct {
		name        string
		field       string
		value       interface{}
		innerErr    error
		expectedMsg string
	}{
		{
			name:        "string field validation",
			field:       "file_extension",
			value:       ".exe",
			innerErr:    fmt.Errorf("forbidden extension"),
			expectedMsg: "validation failed for field file_extension (value: .exe): forbidden extension",
		},
		{
			name:        "numeric field validation",
			field:       "file_size",
			value:       2097152,
			innerErr:    fmt.Errorf("exceeds maximum size"),
			expectedMsg: "validation failed for field file_size (value: 2097152): exceeds maximum size",
		},
		{
			name:        "boolean field validation",
			field:       "exec_enabled",
			value:       true,
			innerErr:    fmt.Errorf("exec not allowed"),
			expectedMsg: "validation failed for field exec_enabled (value: true): exec not allowed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			valErr := &ValidationError{
				Field: tt.field,
				Value: tt.value,
				Err:   tt.innerErr,
			}

			assert.Equal(t, tt.expectedMsg, valErr.Error())
			assert.Equal(t, tt.innerErr, valErr.Unwrap())

			// Test error type checking
			var validationErr *ValidationError
			assert.True(t, errors.As(valErr, &validationErr))
		})
	}
}

// TestResourceError tests ResourceError functionality
func TestResourceError(t *testing.T) {
	tests := []struct {
		name        string
		resource    string
		limit       interface{}
		actual      interface{}
		innerErr    error
		expectedMsg string
	}{
		{
			name:        "file size limit",
			resource:    "file_size",
			limit:       1048576,
			actual:      2097152,
			innerErr:    fmt.Errorf("file too large"),
			expectedMsg: "resource file_size exceeded limit 1048576 (actual: 2097152): file too large",
		},
		{
			name:        "memory limit",
			resource:    "memory",
			limit:       "512m",
			actual:      "1g",
			innerErr:    fmt.Errorf("insufficient memory"),
			expectedMsg: "resource memory exceeded limit 512m (actual: 1g): insufficient memory",
		},
		{
			name:        "timeout limit",
			resource:    "execution_time",
			limit:       30.0,
			actual:      45.5,
			innerErr:    fmt.Errorf("execution timed out"),
			expectedMsg: "resource execution_time exceeded limit 30 (actual: 45.5): execution timed out",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resErr := &ResourceError{
				Resource: tt.resource,
				Limit:    tt.limit,
				Actual:   tt.actual,
				Err:      tt.innerErr,
			}

			assert.Equal(t, tt.expectedMsg, resErr.Error())
			assert.Equal(t, tt.innerErr, resErr.Unwrap())

			// Test error type checking
			var resourceErr *ResourceError
			assert.True(t, errors.As(resErr, &resourceErr))
		})
	}
}

// TestErrorWrapping tests error wrapping behavior
func TestErrorWrapping(t *testing.T) {
	t.Run("SecurityError wrapping", func(t *testing.T) {
		baseErr := fmt.Errorf("original error")
		secErr := &SecurityError{
			Op:   "test_op",
			Path: "test_path",
			Err:  baseErr,
		}

		// Test unwrapping
		assert.True(t, errors.Is(secErr, baseErr))
		assert.Equal(t, baseErr, errors.Unwrap(secErr))
	})

	t.Run("ValidationError wrapping", func(t *testing.T) {
		baseErr := fmt.Errorf("validation failed")
		valErr := &ValidationError{
			Field: "test_field",
			Value: "test_value",
			Err:   baseErr,
		}

		// Test unwrapping
		assert.True(t, errors.Is(valErr, baseErr))
		assert.Equal(t, baseErr, errors.Unwrap(valErr))
	})

	t.Run("ResourceError wrapping", func(t *testing.T) {
		baseErr := fmt.Errorf("resource exceeded")
		resErr := &ResourceError{
			Resource: "test_resource",
			Limit:    100,
			Actual:   150,
			Err:      baseErr,
		}

		// Test unwrapping
		assert.True(t, errors.Is(resErr, baseErr))
		assert.Equal(t, baseErr, errors.Unwrap(resErr))
	})
}

// TestErrorTypeDetection tests error type detection
func TestErrorTypeDetection(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		testFunc func(t *testing.T, err error)
	}{
		{
			name: "detect SecurityError",
			err: &SecurityError{
				Op:   "test",
				Path: "test.txt",
				Err:  fmt.Errorf("test error"),
			},
			testFunc: func(t *testing.T, err error) {
				var secErr *SecurityError
				assert.True(t, errors.As(err, &secErr))
				assert.Equal(t, "test", secErr.Op)
			},
		},
		{
			name: "detect ValidationError",
			err: &ValidationError{
				Field: "test_field",
				Value: "test_value",
				Err:   fmt.Errorf("validation failed"),
			},
			testFunc: func(t *testing.T, err error) {
				var valErr *ValidationError
				assert.True(t, errors.As(err, &valErr))
				assert.Equal(t, "test_field", valErr.Field)
			},
		},
		{
			name: "detect ResourceError",
			err: &ResourceError{
				Resource: "memory",
				Limit:    100,
				Actual:   150,
				Err:      fmt.Errorf("exceeded"),
			},
			testFunc: func(t *testing.T, err error) {
				var resErr *ResourceError
				assert.True(t, errors.As(err, &resErr))
				assert.Equal(t, "memory", resErr.Resource)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.testFunc(t, tt.err)
		})
	}
}

// TestErrorChaining tests error chaining with multiple levels
func TestErrorChaining(t *testing.T) {
	// Create a chain: base -> validation -> security
	baseErr := fmt.Errorf("original filesystem error")

	valErr := &ValidationError{
		Field: "path",
		Value: "../etc/passwd",
		Err:   baseErr,
	}

	secErr := &SecurityError{
		Op:   "path_validation",
		Path: "../etc/passwd",
		Err:  valErr,
	}

	// Test that we can detect all error types in the chain
	assert.True(t, errors.Is(secErr, baseErr))

	var foundSecErr *SecurityError
	assert.True(t, errors.As(secErr, &foundSecErr))

	var foundValErr *ValidationError
	assert.True(t, errors.As(secErr, &foundValErr))

	// Test unwrapping through the chain
	unwrapped := errors.Unwrap(secErr)
	assert.Equal(t, valErr, unwrapped)

	unwrapped2 := errors.Unwrap(unwrapped)
	assert.Equal(t, baseErr, unwrapped2)
}

// TestErrorStringRepresentation tests string representations
func TestErrorStringRepresentation(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		contains []string
	}{
		{
			name: "SecurityError contains all info",
			err: &SecurityError{
				Op:   "validation",
				Path: "sensitive.txt",
				Err:  fmt.Errorf("access denied"),
			},
			contains: []string{"security violation", "validation", "sensitive.txt", "access denied"},
		},
		{
			name: "ValidationError contains all info",
			err: &ValidationError{
				Field: "extension",
				Value: ".exe",
				Err:   fmt.Errorf("not allowed"),
			},
			contains: []string{"validation failed", "extension", ".exe", "not allowed"},
		},
		{
			name: "ResourceError contains all info",
			err: &ResourceError{
				Resource: "memory",
				Limit:    "512M",
				Actual:   "1G",
				Err:      fmt.Errorf("exceeded"),
			},
			contains: []string{"resource", "memory", "exceeded limit", "512M", "1G", "exceeded"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errStr := tt.err.Error()
			for _, expected := range tt.contains {
				assert.Contains(t, errStr, expected)
			}
		})
	}
}

// TestNilErrorHandling tests handling of nil inner errors
func TestNilErrorHandling(t *testing.T) {
	t.Run("SecurityError with nil inner error", func(t *testing.T) {
		secErr := &SecurityError{
			Op:   "test",
			Path: "test.txt",
			Err:  nil,
		}

		errStr := secErr.Error()
		assert.Contains(t, errStr, "test")
		assert.Contains(t, errStr, "test.txt")
		assert.Nil(t, secErr.Unwrap())
	})

	t.Run("ValidationError with nil inner error", func(t *testing.T) {
		valErr := &ValidationError{
			Field: "test",
			Value: "test",
			Err:   nil,
		}

		errStr := valErr.Error()
		assert.Contains(t, errStr, "test")
		assert.Nil(t, valErr.Unwrap())
	})

	t.Run("ResourceError with nil inner error", func(t *testing.T) {
		resErr := &ResourceError{
			Resource: "test",
			Limit:    100,
			Actual:   150,
			Err:      nil,
		}

		errStr := resErr.Error()
		assert.Contains(t, errStr, "test")
		assert.Nil(t, resErr.Unwrap())
	})
}

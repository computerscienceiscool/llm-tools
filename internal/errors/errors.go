package errors

import "fmt"

// Error types for the application
var (
	ErrFileNotFound      = fmt.Errorf("FILE_NOT_FOUND")
	ErrPathSecurity      = fmt.Errorf("PATH_SECURITY")
	ErrResourceLimit     = fmt.Errorf("RESOURCE_LIMIT")
	ErrPermissionDenied  = fmt.Errorf("PERMISSION_DENIED")
	ErrExecValidation    = fmt.Errorf("EXEC_VALIDATION")
	ErrDockerUnavailable = fmt.Errorf("DOCKER_UNAVAILABLE")
	ErrExecTimeout       = fmt.Errorf("EXEC_TIMEOUT")
	ErrExecFailed        = fmt.Errorf("EXEC_FAILED")
	ErrSearchDisabled    = fmt.Errorf("SEARCH_DISABLED")
	ErrSearchInitFailed  = fmt.Errorf("SEARCH_INIT_FAILED")
	ErrExtensionDenied   = fmt.Errorf("EXTENSION_DENIED")
)

// SecurityError wraps security-related errors
type SecurityError struct {
	Op   string
	Path string
	Err  error
}

func (e *SecurityError) Error() string {
	return fmt.Sprintf("security violation in %s for path %s: %v", e.Op, e.Path, e.Err)
}

func (e *SecurityError) Unwrap() error {
	return e.Err
}

// ValidationError wraps validation errors
type ValidationError struct {
	Field string
	Value interface{}
	Err   error
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation failed for field %s (value: %v): %v", e.Field, e.Value, e.Err)
}

func (e *ValidationError) Unwrap() error {
	return e.Err
}

// ResourceError wraps resource-related errors
type ResourceError struct {
	Resource string
	Limit    interface{}
	Actual   interface{}
	Err      error
}

func (e *ResourceError) Error() string {
	return fmt.Sprintf("resource %s exceeded limit %v (actual: %v): %v", e.Resource, e.Limit, e.Actual, e.Err)
}

func (e *ResourceError) Unwrap() error {
	return e.Err
}

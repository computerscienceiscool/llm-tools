package security

// PathValidator validates file paths for security
type PathValidator interface {
	ValidatePath(requestedPath, repositoryRoot string, excludedPaths []string) (string, error)
	ValidateWriteExtension(filepath string, allowedExtensions []string) error
}

// AuditLogger logs operations for security audit
type AuditLogger interface {
	LogOperation(sessionID, command, argument string, success bool, errorMsg string)
}

package security

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/computerscienceiscool/llm-tools/internal/errors"
)

// TestNewPathValidator tests validator creation
func TestNewPathValidator(t *testing.T) {
	validator := NewPathValidator()

	assert.NotNil(t, validator)
	assert.IsType(t, &DefaultPathValidator{}, validator)
}

// TestDefaultPathValidatorValidatePath tests the main path validation functionality
func TestDefaultPathValidatorValidatePath(t *testing.T) {
	// Create temporary directory structure for testing
	tempDir, err := os.MkdirTemp("", "path-validator-test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create test subdirectories and files
	subDir := filepath.Join(tempDir, "subdir")
	err = os.MkdirAll(subDir, 0755)
	require.NoError(t, err)

	testFile := filepath.Join(tempDir, "test.txt")
	err = os.WriteFile(testFile, []byte("test"), 0644)
	require.NoError(t, err)

	validator := NewPathValidator()

	tests := []struct {
		name           string
		requestedPath  string
		repositoryRoot string
		excludedPaths  []string
		expectError    bool
		errorType      string
		validateResult func(t *testing.T, result string)
	}{
		{
			name:           "valid relative path",
			requestedPath:  "test.txt",
			repositoryRoot: tempDir,
			excludedPaths:  []string{".git"},
			expectError:    false,
			validateResult: func(t *testing.T, result string) {
				assert.Equal(t, testFile, result)
			},
		},
		{
			name:           "valid subdirectory path",
			requestedPath:  "subdir",
			repositoryRoot: tempDir,
			excludedPaths:  []string{".git"},
			expectError:    false,
			validateResult: func(t *testing.T, result string) {
				assert.Equal(t, subDir, result)
			},
		},
		{
			name:           "absolute path within repo",
			requestedPath:  testFile,
			repositoryRoot: tempDir,
			excludedPaths:  []string{".git"},
			expectError:    false,
			validateResult: func(t *testing.T, result string) {
				assert.Equal(t, testFile, result)
			},
		},
		{
			name:           "path traversal with ../",
			requestedPath:  "../../../etc/passwd",
			repositoryRoot: tempDir,
			excludedPaths:  []string{".git"},
			expectError:    true,
			errorType:      "PATH_SECURITY",
		},
		{
			name:           "path traversal in middle",
			requestedPath:  "subdir/../../etc/passwd",
			repositoryRoot: tempDir,
			excludedPaths:  []string{".git"},
			expectError:    true,
			errorType:      "PATH_SECURITY",
		},
		{
			name:           "excluded .git path",
			requestedPath:  ".git/config",
			repositoryRoot: tempDir,
			excludedPaths:  []string{".git"},
			expectError:    true,
			errorType:      "PATH_SECURITY",
		},
		{
			name:           "excluded .env file",
			requestedPath:  ".env",
			repositoryRoot: tempDir,
			excludedPaths:  []string{".env"},
			expectError:    true,
			errorType:      "PATH_SECURITY",
		},
		{
			name:           "excluded pattern *.key",
			requestedPath:  "private.key",
			repositoryRoot: tempDir,
			excludedPaths:  []string{"*.key"},
			expectError:    true,
			errorType:      "PATH_SECURITY",
		},
		{
			name:           "excluded directory prefix",
			requestedPath:  ".git/objects/abc123",
			repositoryRoot: tempDir,
			excludedPaths:  []string{".git"},
			expectError:    true,
			errorType:      "PATH_SECURITY",
		},
		{
			name:           "absolute path outside repo",
			requestedPath:  "/etc/passwd",
			repositoryRoot: tempDir,
			excludedPaths:  []string{".git"},
			expectError:    true,
			errorType:      "PATH_SECURITY",
		},
		{
			name:           "nonexistent but valid path",
			requestedPath:  "nonexistent.txt",
			repositoryRoot: tempDir,
			excludedPaths:  []string{".git"},
			expectError:    false,
			validateResult: func(t *testing.T, result string) {
				expected := filepath.Join(tempDir, "nonexistent.txt")
				assert.Equal(t, expected, result)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := validator.ValidatePath(tt.requestedPath, tt.repositoryRoot, tt.excludedPaths)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorType != "" {
					assert.Contains(t, err.Error(), tt.errorType)
				}
				assert.Empty(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotEmpty(t, result)
				if tt.validateResult != nil {
					tt.validateResult(t, result)
				}
			}
		})
	}
}

// TestPathValidatorExcludedPaths tests excluded path handling
func TestPathValidatorExcludedPaths(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "excluded-paths-test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	validator := NewPathValidator()

	tests := []struct {
		name          string
		requestedPath string
		excludedPaths []string
		shouldExclude bool
	}{
		{
			name:          "exact match .git",
			requestedPath: ".git",
			excludedPaths: []string{".git"},
			shouldExclude: true,
		},
		{
			name:          "exact match .env",
			requestedPath: ".env",
			excludedPaths: []string{".env"},
			shouldExclude: true,
		},
		{
			name:          "pattern match *.key",
			requestedPath: "private.key",
			excludedPaths: []string{"*.key"},
			shouldExclude: true,
		},
		{
			name:          "pattern match *.pem",
			requestedPath: "certificate.pem",
			excludedPaths: []string{"*.pem"},
			shouldExclude: true,
		},
		{
			name:          "directory prefix .git/",
			requestedPath: ".git/config",
			excludedPaths: []string{".git"},
			shouldExclude: true,
		},
		{
			name:          "nested directory .git/objects/abc",
			requestedPath: ".git/objects/abc123",
			excludedPaths: []string{".git"},
			shouldExclude: true,
		},
		{
			name:          "similar but not excluded git.txt",
			requestedPath: "git.txt",
			excludedPaths: []string{".git"},
			shouldExclude: false,
		},
		{
			name:          "allowed file with excluded extension in name",
			requestedPath: "keyfile.txt",
			excludedPaths: []string{"*.key"},
			shouldExclude: false,
		},
		{
			name:          "multiple exclusions",
			requestedPath: "secret.key",
			excludedPaths: []string{".git", "*.key", ".env"},
			shouldExclude: true,
		},
		{
			name:          "empty exclusions",
			requestedPath: "anyfile.txt",
			excludedPaths: []string{},
			shouldExclude: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := validator.ValidatePath(tt.requestedPath, tempDir, tt.excludedPaths)

			if tt.shouldExclude {
				assert.Error(t, err)
				var secErr *errors.SecurityError
				assert.ErrorAs(t, err, &secErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestPathValidatorSymlinks tests symlink handling
func TestPathValidatorSymlinks(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "symlink-test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create outside directory
	outsideDir, err := os.MkdirTemp("", "outside-repo")
	require.NoError(t, err)
	defer os.RemoveAll(outsideDir)

	outsideFile := filepath.Join(outsideDir, "outside.txt")
	err = os.WriteFile(outsideFile, []byte("outside content"), 0644)
	require.NoError(t, err)

	// Create symlink from inside repo to outside file
	symlinkPath := filepath.Join(tempDir, "symlink-to-outside")
	err = os.Symlink(outsideFile, symlinkPath)
	if err != nil {
		t.Skip("Cannot create symlinks on this system")
	}

	// Create legitimate file inside repo
	insideFile := filepath.Join(tempDir, "inside.txt")
	err = os.WriteFile(insideFile, []byte("inside content"), 0644)
	require.NoError(t, err)

	// Create symlink to legitimate inside file
	goodSymlink := filepath.Join(tempDir, "good-symlink")
	err = os.Symlink(insideFile, goodSymlink)
	require.NoError(t, err)

	validator := NewPathValidator()

	tests := []struct {
		name          string
		requestedPath string
		expectError   bool
		description   string
	}{
		{
			name:          "symlink to outside repo",
			requestedPath: "symlink-to-outside",
			expectError:   true,
			description:   "Should reject symlinks that point outside repository",
		},
		{
			name:          "symlink to inside repo",
			requestedPath: "good-symlink",
			expectError:   false,
			description:   "Should allow symlinks that point within repository",
		},
		{
			name:          "regular file",
			requestedPath: "inside.txt",
			expectError:   false,
			description:   "Should allow regular files",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := validator.ValidatePath(tt.requestedPath, tempDir, []string{})

			if tt.expectError {
				assert.Error(t, err, tt.description)
			} else {
				assert.NoError(t, err, tt.description)
			}
		})
	}
}

// TestValidateWriteExtension tests write extension validation
func TestValidateWriteExtension(t *testing.T) {
	validator := NewPathValidator()

	tests := []struct {
		name              string
		filepath          string
		allowedExtensions []string
		expectError       bool
		errorType         string
	}{
		{
			name:              "allowed .go extension",
			filepath:          "main.go",
			allowedExtensions: []string{".go", ".py"},
			expectError:       false,
		},
		{
			name:              "allowed .py extension",
			filepath:          "script.py",
			allowedExtensions: []string{".go", ".py"},
			expectError:       false,
		},
		{
			name:              "forbidden .exe extension",
			filepath:          "malware.exe",
			allowedExtensions: []string{".go", ".py"},
			expectError:       true,
			errorType:         "EXTENSION_DENIED",
		},
		{
			name:              "forbidden .bat extension",
			filepath:          "script.bat",
			allowedExtensions: []string{".go", ".py"},
			expectError:       true,
			errorType:         "EXTENSION_DENIED",
		},
		{
			name:              "case sensitivity .GO vs .go",
			filepath:          "file.GO",
			allowedExtensions: []string{".go"},
			expectError:       false, // Should be case insensitive
		},
		{
			name:              "no extension",
			filepath:          "Makefile",
			allowedExtensions: []string{".go", ".py"},
			expectError:       true,
			errorType:         "EXTENSION_DENIED",
		},
		{
			name:              "empty allowed extensions",
			filepath:          "any.txt",
			allowedExtensions: []string{},
			expectError:       false, // No restrictions when empty
		},
		{
			name:              "multiple dots in filename",
			filepath:          "file.min.js",
			allowedExtensions: []string{".js"},
			expectError:       false, // Should check last extension
		},
		{
			name:              "path with extension",
			filepath:          "dir/subdir/file.py",
			allowedExtensions: []string{".py"},
			expectError:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateWriteExtension(tt.filepath, tt.allowedExtensions)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorType != "" {
					var validationErr *errors.ValidationError
					assert.ErrorAs(t, err, &validationErr)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestPathValidatorErrorTypes tests specific error type generation
func TestPathValidatorErrorTypes(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "error-types-test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	validator := NewPathValidator()

	tests := []struct {
		name          string
		operation     func() error
		expectedError interface{}
		description   string
	}{
		{
			name: "security error for path traversal",
			operation: func() error {
				_, err := validator.ValidatePath("../etc/passwd", tempDir, []string{})
				return err
			},
			expectedError: &errors.SecurityError{},
			description:   "Should return SecurityError for path traversal",
		},
		{
			name: "security error for excluded path",
			operation: func() error {
				_, err := validator.ValidatePath(".git/config", tempDir, []string{".git"})
				return err
			},
			expectedError: &errors.SecurityError{},
			description:   "Should return SecurityError for excluded paths",
		},
		{
			name: "validation error for forbidden extension",
			operation: func() error {
				return validator.ValidateWriteExtension("malware.exe", []string{".txt"})
			},
			expectedError: &errors.ValidationError{},
			description:   "Should return ValidationError for forbidden extensions",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.operation()

			require.Error(t, err, tt.description)
			assert.ErrorAs(t, err, tt.expectedError, tt.description)
		})
	}
}

// TestPathValidatorEdgeCases tests edge cases and boundary conditions
func TestPathValidatorEdgeCases(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "edge-cases-test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	validator := NewPathValidator()

	tests := []struct {
		name string
		test func(*testing.T)
	}{
		{
			name: "empty path",
			test: func(t *testing.T) {
				_, err := validator.ValidatePath("", tempDir, []string{})
				assert.Error(t, err)
			},
		},
		{
			name: "dot path (.)",
			test: func(t *testing.T) {
				result, err := validator.ValidatePath(".", tempDir, []string{})
				assert.NoError(t, err)
				assert.Contains(t, result, tempDir)
			},
		},
		{
			name: "double dot path (..)",
			test: func(t *testing.T) {
				_, err := validator.ValidatePath("..", tempDir, []string{})
				assert.Error(t, err)
			},
		},
		{
			name: "very long path",
			test: func(t *testing.T) {
				longPath := strings.Repeat("very-long-component/", 50) + "file.txt"
				_, err := validator.ValidatePath(longPath, tempDir, []string{})
				// Should not crash, but may fail validation
				// The exact behavior depends on filesystem limits
				assert.NotPanics(t, func() {
					validator.ValidatePath(longPath, tempDir, []string{})
				})
			},
		},
		{
			name: "unicode characters",
			test: func(t *testing.T) {
				unicodePath := "测试文件.txt"
				_, err := validator.ValidatePath(unicodePath, tempDir, []string{})
				assert.NoError(t, err)
			},
		},
		{
			name: "special characters",
			test: func(t *testing.T) {
				specialPath := "file with spaces & symbols!@#$.txt"
				_, err := validator.ValidatePath(specialPath, tempDir, []string{})
				assert.NoError(t, err)
			},
		},
		{
			name: "null bytes in path",
			test: func(t *testing.T) {
				nullPath := "file\x00.txt"
				_, err := validator.ValidatePath(nullPath, tempDir, []string{})
				// Should handle gracefully (likely error)
				assert.NotPanics(t, func() {
					validator.ValidatePath(nullPath, tempDir, []string{})
				})
			},
		},
		{
			name: "path with only extension",
			test: func(t *testing.T) {
				err := validator.ValidateWriteExtension(".gitignore", []string{""})
				assert.NoError(t, err) // Empty string should match files without extension
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, tt.test)
	}
}

// TestPathValidatorConcurrency tests concurrent usage
func TestPathValidatorConcurrency(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "concurrency-test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	validator := NewPathValidator()

	const numGoroutines = 50
	done := make(chan bool, numGoroutines)

	// Test concurrent validation
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer func() { done <- true }()

			for j := 0; j < 10; j++ {
				// Test various operations concurrently
				_, err1 := validator.ValidatePath("test.txt", tempDir, []string{".git"})
				assert.NoError(t, err1)

				_, err2 := validator.ValidatePath("../outside", tempDir, []string{})
				assert.Error(t, err2)

				err3 := validator.ValidateWriteExtension("script.py", []string{".py"})
				assert.NoError(t, err3)

				err4 := validator.ValidateWriteExtension("malware.exe", []string{".py"})
				assert.Error(t, err4)
			}
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		<-done
	}
}

// BenchmarkPathValidation benchmarks path validation performance
func BenchmarkPathValidation(b *testing.B) {
	tempDir, err := os.MkdirTemp("", "benchmark-test")
	if err != nil {
		b.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	validator := NewPathValidator()
	excludedPaths := []string{".git", ".env", "*.key"}

	b.Run("ValidPath", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = validator.ValidatePath("safe/file.txt", tempDir, excludedPaths)
		}
	})

	b.Run("InvalidPath", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = validator.ValidatePath("../../../etc/passwd", tempDir, excludedPaths)
		}
	})

	b.Run("ExtensionValidation", func(b *testing.B) {
		allowedExts := []string{".go", ".py", ".js", ".md", ".txt"}
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = validator.ValidateWriteExtension("script.py", allowedExts)
		}
	})
}

// BenchmarkExcludedPathsMatching benchmarks excluded path matching
func BenchmarkExcludedPathsMatching(b *testing.B) {
	validator := NewPathValidator()

	// Create various excluded path patterns
	excludedPaths := []string{
		".git", ".env", ".env.local", "*.key", "*.pem", "*.p12", "*.pfx",
		"node_modules", "__pycache__", ".DS_Store", "*.sqlite", "*.db",
		"secrets", "credentials", ".idea", ".vscode", "*.tmp", "*.log",
	}

	tests := []struct {
		name string
		path string
	}{
		{"AllowedFile", "src/main.go"},
		{"ExcludedExact", ".git"},
		{"ExcludedPattern", "private.key"},
		{"ExcludedDirectory", ".git/config"},
		{"LongAllowedPath", "src/components/auth/handlers/login.go"},
		{"LongExcludedPath", "node_modules/package/dist/index.js"},
	}

	tempDir, err := os.MkdirTemp("", "benchmark-exclude")
	if err != nil {
		b.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, _ = validator.ValidatePath(tt.path, tempDir, excludedPaths)
			}
		})
	}
}

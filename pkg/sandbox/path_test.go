package sandbox

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidatePath(t *testing.T) {
	// Create temporary directory as repository root
	repoRoot := t.TempDir()

	// Create subdirectories and files for testing
	subDir := filepath.Join(repoRoot, "src")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatalf("Failed to create subdirectory: %v", err)
	}

	testFile := filepath.Join(subDir, "test.go")
	if err := os.WriteFile(testFile, []byte("package main"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	tests := []struct {
		name          string
		requestedPath string
		repoRoot      string
		excludedPaths []string
		wantErr       bool
		errContains   string
	}{
		{
			name:          "valid relative path to existing file",
			requestedPath: "src/test.go",
			repoRoot:      repoRoot,
			excludedPaths: []string{},
			wantErr:       false,
		},
		{
			name:          "valid relative path to existing directory",
			requestedPath: "src",
			repoRoot:      repoRoot,
			excludedPaths: []string{},
			wantErr:       false,
		},
		{
			name:          "valid absolute path within repo",
			requestedPath: filepath.Join(repoRoot, "src", "test.go"),
			repoRoot:      repoRoot,
			excludedPaths: []string{},
			wantErr:       false,
		},
		{
			name:          "path traversal with ..",
			requestedPath: "../../../etc/passwd",
			repoRoot:      repoRoot,
			excludedPaths: []string{},
			wantErr:       true,
			errContains:   "traversal",
		},
		{
			name:          "path traversal hidden in middle",
			requestedPath: "src/../../etc/passwd",
			repoRoot:      repoRoot,
			excludedPaths: []string{},
			wantErr:       true,
			errContains:   "traversal",
		},
		{
			name:          "excluded path - exact match",
			requestedPath: ".git",
			repoRoot:      repoRoot,
			excludedPaths: []string{".git", "node_modules"},
			wantErr:       true,
			errContains:   "excluded list",
		},
		{
			name:          "excluded path - subdirectory",
			requestedPath: ".git/config",
			repoRoot:      repoRoot,
			excludedPaths: []string{".git", "node_modules"},
			wantErr:       true,
			errContains:   "excluded directory",
		},
		{
			name:          "excluded path - node_modules",
			requestedPath: "node_modules/package/index.js",
			repoRoot:      repoRoot,
			excludedPaths: []string{".git", "node_modules"},
			wantErr:       true,
			errContains:   "excluded directory",
		},
		{
			name:          "path with redundant slashes cleaned",
			requestedPath: "src//test.go",
			repoRoot:      repoRoot,
			excludedPaths: []string{},
			wantErr:       false,
		},
		{
			name:          "path with dot components",
			requestedPath: "./src/./test.go",
			repoRoot:      repoRoot,
			excludedPaths: []string{},
			wantErr:       false,
		},
		{
			name:          "empty excluded paths",
			requestedPath: "src/test.go",
			repoRoot:      repoRoot,
			excludedPaths: nil,
			wantErr:       false,
		},
		{
			name:          "non-existent file in valid directory",
			requestedPath: "src/newfile.go",
			repoRoot:      repoRoot,
			excludedPaths: []string{},
			wantErr:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ValidatePath(tt.requestedPath, tt.repoRoot, tt.excludedPaths)

			if tt.wantErr {
				if err == nil {
					t.Errorf("ValidatePath() expected error containing %q, got nil", tt.errContains)
					return
				}
				if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("ValidatePath() error = %v, want error containing %q", err, tt.errContains)
				}
			} else {
				if err != nil {
					t.Errorf("ValidatePath() unexpected error = %v", err)
					return
				}
				// Verify the result is an absolute path within the repo
				if !filepath.IsAbs(result) {
					t.Errorf("ValidatePath() returned non-absolute path: %s", result)
				}
			}
		})
	}
}

func TestValidatePath_Symlinks(t *testing.T) {
	// Create temporary directory as repository root
	repoRoot := t.TempDir()

	// Create a subdirectory and file
	subDir := filepath.Join(repoRoot, "src")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatalf("Failed to create subdirectory: %v", err)
	}

	realFile := filepath.Join(subDir, "real.go")
	if err := os.WriteFile(realFile, []byte("package main"), 0644); err != nil {
		t.Fatalf("Failed to create real file: %v", err)
	}

	// Create a symlink within the repository
	symlinkWithin := filepath.Join(repoRoot, "link_to_src")
	if err := os.Symlink(subDir, symlinkWithin); err != nil {
		t.Skipf("Cannot create symlinks on this system: %v", err)
	}

	t.Run("symlink within repository", func(t *testing.T) {
		result, err := ValidatePath("link_to_src/real.go", repoRoot, nil)
		if err != nil {
			t.Errorf("ValidatePath() unexpected error for valid symlink: %v", err)
			return
		}
		if !strings.HasSuffix(result, "real.go") {
			t.Errorf("ValidatePath() expected path to real.go, got: %s", result)
		}
	})

	// Create a symlink pointing outside the repository
	outsideDir := t.TempDir()
	outsideFile := filepath.Join(outsideDir, "outside.txt")
	if err := os.WriteFile(outsideFile, []byte("outside content"), 0644); err != nil {
		t.Fatalf("Failed to create outside file: %v", err)
	}

	symlinkOutside := filepath.Join(repoRoot, "link_outside")
	if err := os.Symlink(outsideDir, symlinkOutside); err != nil {
		t.Fatalf("Failed to create symlink to outside: %v", err)
	}

	t.Run("symlink escaping repository", func(t *testing.T) {
		_, err := ValidatePath("link_outside/outside.txt", repoRoot, nil)
		if err == nil {
			t.Error("ValidatePath() expected error for symlink escaping repository")
		}
	})
}

func TestValidatePath_AbsolutePaths(t *testing.T) {
	repoRoot := t.TempDir()

	// Create a file in the repo
	testFile := filepath.Join(repoRoot, "test.txt")
	if err := os.WriteFile(testFile, []byte("content"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create a separate directory outside repo
	outsideDir := t.TempDir()
	outsideFile := filepath.Join(outsideDir, "outside.txt")
	if err := os.WriteFile(outsideFile, []byte("outside"), 0644); err != nil {
		t.Fatalf("Failed to create outside file: %v", err)
	}

	tests := []struct {
		name          string
		requestedPath string
		wantErr       bool
	}{
		{
			name:          "absolute path inside repo",
			requestedPath: testFile,
			wantErr:       false,
		},
		{
			name:          "absolute path outside repo",
			requestedPath: outsideFile,
			wantErr:       true,
		},
		{
			name:          "absolute path to /etc/passwd",
			requestedPath: "/etc/passwd",
			wantErr:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ValidatePath(tt.requestedPath, repoRoot, nil)
			if tt.wantErr && err == nil {
				t.Error("ValidatePath() expected error for path outside repo")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("ValidatePath() unexpected error: %v", err)
			}
		})
	}
}

func TestValidatePath_ExcludedPatterns(t *testing.T) {
	repoRoot := t.TempDir()

	tests := []struct {
		name          string
		requestedPath string
		excludedPaths []string
		wantErr       bool
	}{
		{
			name:          "glob pattern match",
			requestedPath: "secret.txt",
			excludedPaths: []string{"*.txt"},
			wantErr:       true,
		},
		{
			name:          "no glob match",
			requestedPath: "secret.go",
			excludedPaths: []string{"*.txt"},
			wantErr:       false,
		},
		{
			name:          "multiple excluded paths",
			requestedPath: ".env",
			excludedPaths: []string{".git", ".env", "node_modules"},
			wantErr:       true,
		},
		{
			name:          "partial name not excluded",
			requestedPath: ".gitignore",
			excludedPaths: []string{".git"},
			wantErr:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ValidatePath(tt.requestedPath, repoRoot, tt.excludedPaths)
			if tt.wantErr && err == nil {
				t.Error("ValidatePath() expected error for excluded path")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("ValidatePath() unexpected error: %v", err)
			}
		})
	}
}

func TestValidatePath_EdgeCases(t *testing.T) {
	repoRoot := t.TempDir()

	tests := []struct {
		name          string
		requestedPath string
		repoRoot      string
		excludedPaths []string
		wantErr       bool
	}{
		{
			name:          "empty path",
			requestedPath: "",
			repoRoot:      repoRoot,
			excludedPaths: nil,
			wantErr:       false, // Empty path becomes "."
		},
		{
			name:          "just dot",
			requestedPath: ".",
			repoRoot:      repoRoot,
			excludedPaths: nil,
			wantErr:       false,
		},
		{
			name:          "double dots only",
			requestedPath: "..",
			repoRoot:      repoRoot,
			excludedPaths: nil,
			wantErr:       true,
		},
		{
			name:          "deeply nested traversal",
			requestedPath: "a/b/c/../../../../etc/passwd",
			repoRoot:      repoRoot,
			excludedPaths: nil,
			wantErr:       true,
		},
		{
			name:          "path with spaces",
			requestedPath: "path with spaces/file.txt",
			repoRoot:      repoRoot,
			excludedPaths: nil,
			wantErr:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ValidatePath(tt.requestedPath, tt.repoRoot, tt.excludedPaths)
			if tt.wantErr && err == nil {
				t.Error("ValidatePath() expected error")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("ValidatePath() unexpected error: %v", err)
			}
		})
	}
}

func TestValidatePath_InvalidRepoRoot(t *testing.T) {
	t.Run("non-existent repo root", func(t *testing.T) {
		// This should still work since we use filepath.Abs
		_, err := ValidatePath("test.txt", "/non/existent/path", nil)
		// The function should handle this gracefully
		if err != nil {
			// This is acceptable behavior
			t.Logf("Got expected error for non-existent root: %v", err)
		}
	})
}

func TestValidatePath_NestedExclusions(t *testing.T) {
	repoRoot := t.TempDir()

	// Create nested structure
	nestedDir := filepath.Join(repoRoot, "a", "b", "c")
	if err := os.MkdirAll(nestedDir, 0755); err != nil {
		t.Fatalf("Failed to create nested directory: %v", err)
	}

	tests := []struct {
		name          string
		requestedPath string
		excludedPaths []string
		wantErr       bool
	}{
		{
			name:          "nested path not matching exclusion",
			requestedPath: "a/b/c/file.txt",
			excludedPaths: []string{"a/d"},
			wantErr:       false,
		},
		{
			name:          "nested path matching exclusion prefix",
			requestedPath: "a/b/c/file.txt",
			excludedPaths: []string{"a/b"},
			wantErr:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ValidatePath(tt.requestedPath, repoRoot, tt.excludedPaths)
			if tt.wantErr && err == nil {
				t.Error("ValidatePath() expected error for nested exclusion")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("ValidatePath() unexpected error: %v", err)
			}
		})
	}
}

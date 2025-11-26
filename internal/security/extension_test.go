package security

import (
	"strings"
	"testing"
)

func TestValidateWriteExtension(t *testing.T) {
	tests := []struct {
		name              string
		filePath          string
		allowedExtensions []string
		wantErr           bool
		errContains       string
	}{
		{
			name:              "allowed extension - go",
			filePath:          "main.go",
			allowedExtensions: []string{".go", ".py", ".js"},
			wantErr:           false,
		},
		{
			name:              "allowed extension - py",
			filePath:          "script.py",
			allowedExtensions: []string{".go", ".py", ".js"},
			wantErr:           false,
		},
		{
			name:              "allowed extension with path",
			filePath:          "internal/config/types.go",
			allowedExtensions: []string{".go", ".py", ".js"},
			wantErr:           false,
		},
		{
			name:              "disallowed extension",
			filePath:          "data.exe",
			allowedExtensions: []string{".go", ".py", ".js"},
			wantErr:           true,
			errContains:       "not allowed",
		},
		{
			name:              "no extension",
			filePath:          "Makefile",
			allowedExtensions: []string{".go", ".py", ".js"},
			wantErr:           true,
			errContains:       "no extension",
		},
		{
			name:              "empty allowed extensions - no restrictions",
			filePath:          "anything.xyz",
			allowedExtensions: []string{},
			wantErr:           false,
		},
		{
			name:              "nil allowed extensions - no restrictions",
			filePath:          "anything.xyz",
			allowedExtensions: nil,
			wantErr:           false,
		},
		{
			name:              "case insensitive - uppercase file",
			filePath:          "README.MD",
			allowedExtensions: []string{".md", ".txt"},
			wantErr:           false,
		},
		{
			name:              "case insensitive - uppercase allowed",
			filePath:          "readme.md",
			allowedExtensions: []string{".MD", ".TXT"},
			wantErr:           false,
		},
		{
			name:              "case insensitive - mixed case",
			filePath:          "File.Go",
			allowedExtensions: []string{".GO", ".py"},
			wantErr:           false,
		},
		{
			name:              "double extension - uses last",
			filePath:          "archive.tar.gz",
			allowedExtensions: []string{".gz", ".zip"},
			wantErr:           false,
		},
		{
			name:              "double extension - disallowed",
			filePath:          "archive.tar.gz",
			allowedExtensions: []string{".tar", ".zip"},
			wantErr:           true,
			errContains:       ".gz",
		},
		{
			name:              "hidden file with extension",
			filePath:          ".gitignore",
			allowedExtensions: []string{".gitignore"},
			wantErr:           false,
		},
		{
			name:              "hidden file - extension is everything after dot",
			filePath:          ".env",
			allowedExtensions: []string{".env"},
			wantErr:           false,
		},
		{
			name:              "extension with numbers",
			filePath:          "data.mp3",
			allowedExtensions: []string{".mp3", ".mp4"},
			wantErr:           false,
		},
		{
			name:              "very long extension",
			filePath:          "file.verylongextension",
			allowedExtensions: []string{".verylongextension"},
			wantErr:           false,
		},
		{
			name:              "extension only",
			filePath:          ".go",
			allowedExtensions: []string{".go"},
			wantErr:           false,
		},
		{
			name:              "path with dots in directory",
			filePath:          "some.dir/file.go",
			allowedExtensions: []string{".go"},
			wantErr:           false,
		},
		{
			name:              "multiple dots in filename",
			filePath:          "file.test.spec.js",
			allowedExtensions: []string{".js"},
			wantErr:           false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateWriteExtension(tt.filePath, tt.allowedExtensions)

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error, got nil")
				} else if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("expected error containing %q, got %q", tt.errContains, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("expected no error, got %v", err)
				}
			}
		})
	}
}

func TestValidateWriteExtension_EdgeCases(t *testing.T) {
	tests := []struct {
		name              string
		filePath          string
		allowedExtensions []string
		wantErr           bool
	}{
		{
			name:              "empty filepath",
			filePath:          "",
			allowedExtensions: []string{".go"},
			wantErr:           true, // no extension
		},
		{
			name:              "filepath is just a dot",
			filePath:          ".",
			allowedExtensions: []string{"."},
			wantErr:           false, // extension is ""
		},
		{
			name:              "filepath ends with dot",
			filePath:          "file.",
			allowedExtensions: []string{"."},
			wantErr:           false, // extension is ""
		},
		{
			name:              "whitespace in extension",
			filePath:          "file.go ",
			allowedExtensions: []string{".go "},
			wantErr:           false,
		},
		{
			name:              "special characters in extension",
			filePath:          "file.go-backup",
			allowedExtensions: []string{".go-backup"},
			wantErr:           false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateWriteExtension(tt.filePath, tt.allowedExtensions)

			if tt.wantErr && err == nil {
				t.Errorf("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("expected no error, got %v", err)
			}
		})
	}
}

func TestValidateWriteExtension_SingleExtension(t *testing.T) {
	// Test with a single allowed extension
	allowed := []string{".txt"}

	// Should pass
	if err := ValidateWriteExtension("file.txt", allowed); err != nil {
		t.Errorf("expected .txt to be allowed, got error: %v", err)
	}

	// Should fail
	if err := ValidateWriteExtension("file.go", allowed); err == nil {
		t.Error("expected .go to be disallowed, got no error")
	}
}

// Benchmark
func BenchmarkValidateWriteExtension(b *testing.B) {
	allowed := []string{".go", ".py", ".js", ".md", ".txt", ".json", ".yaml", ".toml"}
	filePath := "internal/config/types.go"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ValidateWriteExtension(filePath, allowed)
	}
}

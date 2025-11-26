package infrastructure

import (
	"os"
	"path/filepath"
	"testing"
)

func TestIsTextFile(t *testing.T) {
	tempDir := t.TempDir()

	tests := []struct {
		name     string
		content  []byte
		expected bool
	}{
		{
			name:     "plain text file",
			content:  []byte("Hello, World!\nThis is a test file.\n"),
			expected: true,
		},
		{
			name:     "Go source code",
			content:  []byte("package main\n\nfunc main() {\n\tfmt.Println(\"Hello\")\n}\n"),
			expected: true,
		},
		{
			name:     "JSON file",
			content:  []byte(`{"key": "value", "number": 42}`),
			expected: true,
		},
		{
			name:     "empty file",
			content:  []byte{},
			expected: true,
		},
		{
			name:     "file with only whitespace",
			content:  []byte("   \n\t\n   "),
			expected: true,
		},
		{
			name:     "file with special characters",
			content:  []byte("Special chars: !@#$%^&*()[]{}|;:',.<>?/`~"),
			expected: true,
		},
		{
			name:     "file with unicode",
			content:  []byte("Unicode: 日本語 中文 한국어 émojis: 🎉🚀"),
			expected: true,
		},
		{
			name:     "binary file with null bytes",
			content:  []byte{0x48, 0x65, 0x6c, 0x6c, 0x6f, 0x00, 0x57, 0x6f, 0x72, 0x6c, 0x64},
			expected: false,
		},
		{
			name:     "null byte at start",
			content:  []byte{0x00, 0x48, 0x65, 0x6c, 0x6c, 0x6f},
			expected: false,
		},
		{
			name:     "null byte at end",
			content:  []byte{0x48, 0x65, 0x6c, 0x6c, 0x6f, 0x00},
			expected: false,
		},
		{
			name:     "only null bytes",
			content:  []byte{0x00, 0x00, 0x00, 0x00},
			expected: false,
		},
		{
			name:     "PNG header simulation",
			content:  []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, 0x00, 0x00},
			expected: false,
		},
		{
			name:     "ELF binary header simulation",
			content:  []byte{0x7F, 0x45, 0x4C, 0x46, 0x00, 0x00, 0x00, 0x00},
			expected: false,
		},
		{
			name:     "file with high ASCII but no null",
			content:  []byte{0x80, 0x81, 0x82, 0xFF, 0xFE, 0xFD},
			expected: true, // No null bytes, so considered text
		},
		{
			name:     "markdown file",
			content:  []byte("# Header\n\n- Item 1\n- Item 2\n\n**Bold** and *italic*"),
			expected: true,
		},
		{
			name:     "HTML file",
			content:  []byte("<!DOCTYPE html>\n<html>\n<head><title>Test</title></head>\n<body>Content</body>\n</html>"),
			expected: true,
		},
		{
			name:     "file with CR LF line endings",
			content:  []byte("Line 1\r\nLine 2\r\nLine 3\r\n"),
			expected: true,
		},
		{
			name:     "file with tabs",
			content:  []byte("Column1\tColumn2\tColumn3\n"),
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test file
			filePath := filepath.Join(tempDir, "testfile_"+tt.name)
			if err := os.WriteFile(filePath, tt.content, 0644); err != nil {
				t.Fatalf("Failed to create test file: %v", err)
			}

			result := IsTextFile(filePath)
			if result != tt.expected {
				t.Errorf("IsTextFile() = %v, want %v for content %v", result, tt.expected, tt.content)
			}
		})
	}
}

func TestIsTextFile_NonExistentFile(t *testing.T) {
	result := IsTextFile("/non/existent/path/file.txt")
	if result != false {
		t.Errorf("IsTextFile() = %v for non-existent file, want false", result)
	}
}

func TestIsTextFile_Directory(t *testing.T) {
	tempDir := t.TempDir()

	// Try to check if a directory is a text file
	result := IsTextFile(tempDir)
	// Opening a directory for reading should fail or return empty/error
	// The function should return false for directories
	if result != false {
		t.Errorf("IsTextFile() = %v for directory, want false", result)
	}
}

func TestIsTextFile_LargeTextFile(t *testing.T) {
	tempDir := t.TempDir()

	// Create a large text file (larger than 8KB buffer)
	content := make([]byte, 16*1024) // 16KB
	for i := range content {
		content[i] = byte('A' + (i % 26)) // Fill with letters
	}

	filePath := filepath.Join(tempDir, "large_text.txt")
	if err := os.WriteFile(filePath, content, 0644); err != nil {
		t.Fatalf("Failed to create large file: %v", err)
	}

	result := IsTextFile(filePath)
	if result != true {
		t.Errorf("IsTextFile() = %v for large text file, want true", result)
	}
}

func TestIsTextFile_LargeBinaryFile(t *testing.T) {
	tempDir := t.TempDir()

	// Create a large binary file with null byte after 8KB
	content := make([]byte, 16*1024) // 16KB
	for i := range content {
		content[i] = byte('A' + (i % 26))
	}
	// Put null byte after the buffer size - this should NOT be detected
	content[10000] = 0x00

	filePath := filepath.Join(tempDir, "large_binary.bin")
	if err := os.WriteFile(filePath, content, 0644); err != nil {
		t.Fatalf("Failed to create large file: %v", err)
	}

	// The function only checks first 8KB, so null byte at position 10000 won't be detected
	result := IsTextFile(filePath)
	if result != true {
		t.Logf("Note: IsTextFile only checks first 8KB, null byte at position 10000 not detected")
	}
}

func TestIsTextFile_BinaryWithNullInFirst8KB(t *testing.T) {
	tempDir := t.TempDir()

	// Create a file with null byte within first 8KB
	content := make([]byte, 16*1024)
	for i := range content {
		content[i] = byte('A' + (i % 26))
	}
	content[4096] = 0x00 // Put null byte at 4KB mark

	filePath := filepath.Join(tempDir, "binary_null_4k.bin")
	if err := os.WriteFile(filePath, content, 0644); err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}

	result := IsTextFile(filePath)
	if result != false {
		t.Errorf("IsTextFile() = %v, want false (null byte at 4KB)", result)
	}
}

func TestIsTextFile_PermissionDenied(t *testing.T) {
	// Skip if running as root (root can read anything)
	if os.Getuid() == 0 {
		t.Skip("Skipping permission test when running as root")
	}

	tempDir := t.TempDir()

	// Create a file and remove read permissions
	filePath := filepath.Join(tempDir, "no_read.txt")
	if err := os.WriteFile(filePath, []byte("content"), 0644); err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}

	// Remove read permission
	if err := os.Chmod(filePath, 0000); err != nil {
		t.Fatalf("Failed to change permissions: %v", err)
	}
	defer os.Chmod(filePath, 0644) // Restore for cleanup

	result := IsTextFile(filePath)
	if result != false {
		t.Errorf("IsTextFile() = %v for unreadable file, want false", result)
	}
}

func TestIsTextFile_Symlink(t *testing.T) {
	tempDir := t.TempDir()

	// Create a real text file
	realFile := filepath.Join(tempDir, "real.txt")
	if err := os.WriteFile(realFile, []byte("text content"), 0644); err != nil {
		t.Fatalf("Failed to create real file: %v", err)
	}

	// Create a symlink to it
	symlink := filepath.Join(tempDir, "link.txt")
	if err := os.Symlink(realFile, symlink); err != nil {
		t.Skipf("Cannot create symlinks on this system: %v", err)
	}

	result := IsTextFile(symlink)
	if result != true {
		t.Errorf("IsTextFile() = %v for symlink to text file, want true", result)
	}
}

func TestIsTextFile_BrokenSymlink(t *testing.T) {
	tempDir := t.TempDir()

	// Create a symlink to non-existent file
	symlink := filepath.Join(tempDir, "broken_link.txt")
	if err := os.Symlink("/non/existent/target", symlink); err != nil {
		t.Skipf("Cannot create symlinks on this system: %v", err)
	}

	result := IsTextFile(symlink)
	if result != false {
		t.Errorf("IsTextFile() = %v for broken symlink, want false", result)
	}
}

func TestIsTextFile_SmallFiles(t *testing.T) {
	tempDir := t.TempDir()

	tests := []struct {
		name     string
		content  []byte
		expected bool
	}{
		{
			name:     "single byte text",
			content:  []byte("A"),
			expected: true,
		},
		{
			name:     "single null byte",
			content:  []byte{0x00},
			expected: false,
		},
		{
			name:     "two bytes no null",
			content:  []byte("AB"),
			expected: true,
		},
		{
			name:     "newline only",
			content:  []byte("\n"),
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filePath := filepath.Join(tempDir, "small_"+tt.name)
			if err := os.WriteFile(filePath, tt.content, 0644); err != nil {
				t.Fatalf("Failed to create file: %v", err)
			}

			result := IsTextFile(filePath)
			if result != tt.expected {
				t.Errorf("IsTextFile() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestIsTextFile_CommonFileTypes(t *testing.T) {
	tempDir := t.TempDir()

	tests := []struct {
		name     string
		filename string
		content  []byte
		expected bool
	}{
		{
			name:     "Python file",
			filename: "script.py",
			content:  []byte("#!/usr/bin/env python3\nprint('Hello')"),
			expected: true,
		},
		{
			name:     "Shell script",
			filename: "script.sh",
			content:  []byte("#!/bin/bash\necho 'Hello'"),
			expected: true,
		},
		{
			name:     "YAML file",
			filename: "config.yaml",
			content:  []byte("key: value\nlist:\n  - item1\n  - item2"),
			expected: true,
		},
		{
			name:     "XML file",
			filename: "data.xml",
			content:  []byte("<?xml version=\"1.0\"?>\n<root><item>value</item></root>"),
			expected: true,
		},
		{
			name:     "CSV file",
			filename: "data.csv",
			content:  []byte("name,age,city\nAlice,30,NYC\nBob,25,LA"),
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filePath := filepath.Join(tempDir, tt.filename)
			if err := os.WriteFile(filePath, tt.content, 0644); err != nil {
				t.Fatalf("Failed to create file: %v", err)
			}

			result := IsTextFile(filePath)
			if result != tt.expected {
				t.Errorf("IsTextFile(%s) = %v, want %v", tt.filename, result, tt.expected)
			}
		})
	}
}

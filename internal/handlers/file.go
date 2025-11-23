package handlers

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// FileOperations defines file operation methods
type FileOperations interface {
	Read(path string) ([]byte, error)
	Write(path string, data []byte) error
	Exists(path string) bool
	Size(path string) (int64, error)
	ModTime(path string) (time.Time, error)
	CreateDir(path string) error
	Delete(path string) error
	Copy(src, dst string) error
	Move(src, dst string) error
}

// DefaultFileOperations implements FileOperations
type DefaultFileOperations struct{}

// NewFileOperations creates a new file operations handler
func NewFileOperations() FileOperations {
	return &DefaultFileOperations{}
}

func (fo *DefaultFileOperations) Read(path string) ([]byte, error) {
	return os.ReadFile(path)
}

func (fo *DefaultFileOperations) Write(path string, data []byte) error {
	return os.WriteFile(path, data, 0644)
}

func (fo *DefaultFileOperations) Exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func (fo *DefaultFileOperations) Size(path string) (int64, error) {
	info, err := os.Stat(path)
	if err != nil {
		return 0, err
	}
	return info.Size(), nil
}

func (fo *DefaultFileOperations) ModTime(path string) (time.Time, error) {
	info, err := os.Stat(path)
	if err != nil {
		return time.Time{}, err
	}
	return info.ModTime(), nil
}

func (fo *DefaultFileOperations) CreateDir(path string) error {
	return os.MkdirAll(path, 0755)
}

func (fo *DefaultFileOperations) Delete(path string) error {
	return os.Remove(path)
}

func (fo *DefaultFileOperations) Copy(src, dst string) error {
	data, err := fo.Read(src)
	if err != nil {
		return fmt.Errorf("failed to read source file: %w", err)
	}

	// Create destination directory if needed
	if err := fo.CreateDir(filepath.Dir(dst)); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	return fo.Write(dst, data)
}

func (fo *DefaultFileOperations) Move(src, dst string) error {
	return os.Rename(src, dst)
}

// IsTextFile checks if a file is likely a text file
func IsTextFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	textExts := []string{".go", ".py", ".js", ".md", ".txt", ".json", ".yaml", ".yml", ".toml", ".html", ".css", ".xml"}

	for _, textExt := range textExts {
		if ext == textExt {
			return true
		}
	}
	return false
}

// GetFileExtension returns the file extension in lowercase
func GetFileExtension(path string) string {
	return strings.ToLower(filepath.Ext(path))
}

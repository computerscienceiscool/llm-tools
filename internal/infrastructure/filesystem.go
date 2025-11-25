package infrastructure

import (
	"os"
	"path/filepath"
)

// FileSystem handles file system operations
type FileSystem interface {
	ReadFile(path string) ([]byte, error)
	WriteFile(path string, data []byte, perm os.FileMode) error
	Exists(path string) bool  // ADD THIS LINE
	Stat(path string) (os.FileInfo, error)
	MkdirAll(path string, perm os.FileMode) error
	Remove(path string) error
	Rename(oldpath, newpath string) error
}

// DefaultFileSystem implements FileSystem using os package
type DefaultFileSystem struct{}

// NewFileSystem creates a new file system handler
func NewFileSystem() FileSystem {
	return &DefaultFileSystem{}
}

func (fs *DefaultFileSystem) ReadFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}

func (fs *DefaultFileSystem) WriteFile(path string, data []byte, perm os.FileMode) error {
	return os.WriteFile(path, data, perm)
}

// ADD THIS METHOD
func (fs *DefaultFileSystem) Exists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

func (fs *DefaultFileSystem) Stat(path string) (os.FileInfo, error) {
	return os.Stat(path)
}

func (fs *DefaultFileSystem) MkdirAll(path string, perm os.FileMode) error {
	return os.MkdirAll(path, perm)
}

func (fs *DefaultFileSystem) Remove(path string) error {
	return os.Remove(path)
}

func (fs *DefaultFileSystem) Rename(oldpath, newpath string) error {
	return os.Rename(oldpath, newpath)
}

// EnsureDirExists creates directory if it doesn't exist
func (fs *DefaultFileSystem) EnsureDirExists(path string) error {
	dir := filepath.Dir(path)
	return fs.MkdirAll(dir, 0755)
}

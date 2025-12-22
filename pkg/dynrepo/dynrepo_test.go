package dynrepo

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCreateRepo(t *testing.T) {
	dir, repo, err := CreateRepo()
	if err != nil {
		t.Fatalf("CreateRepo failed: %v", err)
	}
	defer Cleanup(dir)

	// Check directory exists
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		t.Error("repo directory should exist")
	}

	// Check README exists
	readmePath := filepath.Join(dir, "README.md")
	if _, err := os.Stat(readmePath); os.IsNotExist(err) {
		t.Error("README.md should exist")
	}

	// Check repo is valid
	if repo == nil {
		t.Error("repo should not be nil")
	}

	// Check HEAD exists
	_, err = repo.Head()
	if err != nil {
		t.Errorf("repo should have HEAD: %v", err)
	}
}

package dynrepo

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
)

const testDirBase = "/tmp/dynamic-repo"

// CreateRepo creates a new dynamic repository for llm-runtime to work in.
// If KEEP_TEST_REPOS=true, the repo persists after the program exits.
// Otherwise, the caller is responsible for cleanup.
func CreateRepo() (string, *git.Repository, error) {
	var dir string
	var err error

	if os.Getenv("KEEP_TEST_REPOS") == "true" {
		// Create parent directory if it doesn't exist
		err = os.MkdirAll(testDirBase, 0755)
		if err != nil {
			return "", nil, fmt.Errorf("failed to create base directory: %w", err)
		}
		dir, err = os.MkdirTemp(testDirBase, "repo-")
		if err != nil {
			return "", nil, fmt.Errorf("failed to create repo directory: %w", err)
		}
		fmt.Printf("Dynamic repo created at: %s\n", dir)
	} else {
		dir, err = os.MkdirTemp("", "dynamic-repo-")
		if err != nil {
			return "", nil, fmt.Errorf("failed to create temp directory: %w", err)
		}
	}

	repo, err := git.PlainInit(dir, false)
	if err != nil {
		return "", nil, fmt.Errorf("failed to initialize git repository: %w", err)
	}

	// Configure the repository with default user information
	cfg, err := repo.Config()
	if err != nil {
		return "", nil, fmt.Errorf("failed to get repository config: %w", err)
	}
	cfg.User.Name = "LLM Runtime"
	cfg.User.Email = "llm-runtime@example.com"
	if err := repo.SetConfig(cfg); err != nil {
		return "", nil, fmt.Errorf("failed to set repository config: %w", err)
	}

	// Create an initial commit so HEAD points to something
	wt, err := repo.Worktree()
	if err != nil {
		return "", nil, fmt.Errorf("failed to get worktree: %w", err)
	}

	// Create a README file for the initial commit
	readmeContent := []byte("# Dynamic Repository\n\nThis repository was created by llm-runtime.\n")
	readmePath := filepath.Join(dir, "README.md")
	if err := os.WriteFile(readmePath, readmeContent, 0644); err != nil {
		return "", nil, fmt.Errorf("failed to create README: %w", err)
	}

	if _, err := wt.Add("README.md"); err != nil {
		return "", nil, fmt.Errorf("failed to add README: %w", err)
	}

	_, err = wt.Commit("Initial commit", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "LLM Runtime",
			Email: "llm-runtime@example.com",
			When:  time.Now(),
		},
	})
	if err != nil {
		return "", nil, fmt.Errorf("failed to create initial commit: %w", err)
	}

	return dir, repo, nil
}

// Cleanup removes a dynamic repository directory.
// Call this when KEEP_TEST_REPOS is not set and you're done with the repo.
func Cleanup(dir string) error {
	return os.RemoveAll(dir)
}

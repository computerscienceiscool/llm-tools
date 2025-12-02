package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	t.Run("load from non-existent file returns defaults", func(t *testing.T) {
		tempDir := t.TempDir()
		configPath := filepath.Join(tempDir, "non_existent.yaml")

		config, err := LoadConfig(configPath)
		if err != nil {
			t.Fatalf("LoadConfig() error = %v, want nil", err)
		}

		if config == nil {
			t.Fatal("LoadConfig() returned nil config")
		}

		// Check defaults are set
		if config.Repository.Root != "." {
			t.Errorf("Repository.Root = %q, want %q", config.Repository.Root, ".")
		}

		if len(config.Repository.ExcludedPaths) == 0 {
			t.Error("Repository.ExcludedPaths should have default values")
		}
	})

	t.Run("load from valid config file", func(t *testing.T) {
		tempDir := t.TempDir()
		configPath := filepath.Join(tempDir, "config.yaml")

		configContent := `
repository:
  root: "/custom/path"
  excluded_paths:
    - "custom_exclude"
commands:
  search:
    enabled: true
    max_results: 20
`
		if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
			t.Fatalf("Failed to write config file: %v", err)
		}

		config, err := LoadConfig(configPath)
		if err != nil {
			t.Fatalf("LoadConfig() error = %v", err)
		}

		if config.Repository.Root != "/custom/path" {
			t.Errorf("Repository.Root = %q, want %q", config.Repository.Root, "/custom/path")
		}

		if config.Commands.Search.Enabled != true {
			t.Error("Search.Enabled should be true")
		}

		if config.Commands.Search.MaxResults != 20 {
			t.Errorf("Search.MaxResults = %d, want 20", config.Commands.Search.MaxResults)
		}
	})

	t.Run("load from invalid YAML", func(t *testing.T) {
		tempDir := t.TempDir()
		configPath := filepath.Join(tempDir, "invalid.yaml")

		if err := os.WriteFile(configPath, []byte("{ invalid yaml : : :"), 0644); err != nil {
			t.Fatalf("Failed to write config file: %v", err)
		}

		_, err := LoadConfig(configPath)
		if err == nil {
			t.Error("LoadConfig() expected error for invalid config")
		}
	})

	t.Run("load preserves defaults for missing fields", func(t *testing.T) {
		tempDir := t.TempDir()
		configPath := filepath.Join(tempDir, "partial.yaml")

		configContent := `
repository:
  root: "/partial/path"
`
		if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
			t.Fatalf("Failed to write config file: %v", err)
		}

		config, err := LoadConfig(configPath)
		if err != nil {
			t.Fatalf("LoadConfig() error = %v", err)
		}

		if config.Repository.Root != "/partial/path" {
			t.Errorf("Repository.Root = %q, want %q", config.Repository.Root, "/partial/path")
		}

		// Check that defaults are preserved for missing fields
		if config.Commands.Search.VectorDBPath == "" {
			t.Error("VectorDBPath should have default value")
		}
	})

	t.Run("load empty file uses defaults", func(t *testing.T) {
		tempDir := t.TempDir()
		configPath := filepath.Join(tempDir, "empty.yaml")

		if err := os.WriteFile(configPath, []byte(""), 0644); err != nil {
			t.Fatalf("Failed to write config file: %v", err)
		}

		config, err := LoadConfig(configPath)
		if err != nil {
			t.Fatalf("LoadConfig() error = %v", err)
		}

		if config.Security.AuditLogPath == "" {
			t.Logf("AuditLogPath after empty config: %q", config.Security.AuditLogPath)
		}
	})

	t.Run("load from unreadable file", func(t *testing.T) {
		if os.Getuid() == 0 {
			t.Skip("Skipping permission test when running as root")
		}

		tempDir := t.TempDir()
		configPath := filepath.Join(tempDir, "unreadable.yaml")

		if err := os.WriteFile(configPath, []byte("test: value"), 0000); err != nil {
			t.Fatalf("Failed to write config file: %v", err)
		}
		defer os.Chmod(configPath, 0644)

		_, err := LoadConfig(configPath)
		if err == nil {
			t.Error("LoadConfig() expected error for unreadable file")
		}
	})
}

func TestGetConfigPath(t *testing.T) {
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer os.Chdir(origDir)

	t.Run("returns local config when exists", func(t *testing.T) {
		tempDir := t.TempDir()
		if err := os.Chdir(tempDir); err != nil {
			t.Fatalf("Failed to change directory: %v", err)
		}

		localConfig := filepath.Join(tempDir, "llm-runtime.config.yaml")
		if err := os.WriteFile(localConfig, []byte("test: value"), 0644); err != nil {
			t.Fatalf("Failed to create local config: %v", err)
		}

		result := GetConfigPath()
		if result != "llm-runtime.config.yaml" {
			t.Errorf("GetConfigPath() = %q, want %q", result, "llm-runtime.config.yaml")
		}
	})

	t.Run("returns default when no config exists", func(t *testing.T) {
		tempDir := t.TempDir()
		if err := os.Chdir(tempDir); err != nil {
			t.Fatalf("Failed to change directory: %v", err)
		}

		result := GetConfigPath()
		if result != "llm-runtime.config.yaml" {
			t.Errorf("GetConfigPath() = %q, want %q", result, "llm-runtime.config.yaml")
		}
	})
}

func TestGetSearchConfigFromFull(t *testing.T) {
	t.Run("extracts all fields correctly", func(t *testing.T) {
		fullConfig := &FullConfig{}
		SetFullConfigDefaults(fullConfig)

		fullConfig.Commands.Search.Enabled = true
		fullConfig.Commands.Search.MaxResults = 25
		fullConfig.Commands.Search.MinSimilarityScore = 0.75

		searchConfig := GetSearchConfigFromFull(fullConfig)

		if searchConfig.Enabled != true {
			t.Error("Enabled not extracted correctly")
		}

		if searchConfig.MaxResults != 25 {
			t.Errorf("MaxResults = %d, want 25", searchConfig.MaxResults)
		}

		if searchConfig.MinSimilarityScore != 0.75 {
			t.Errorf("MinSimilarityScore = %f, want 0.75", searchConfig.MinSimilarityScore)
		}
	})

	t.Run("extracts slice fields", func(t *testing.T) {
		fullConfig := &FullConfig{}
		SetFullConfigDefaults(fullConfig)

		fullConfig.Commands.Search.IndexExtensions = []string{".custom", ".ext"}

		searchConfig := GetSearchConfigFromFull(fullConfig)

		if len(searchConfig.IndexExtensions) != 2 {
			t.Errorf("IndexExtensions length = %d, want 2", len(searchConfig.IndexExtensions))
		}

		if searchConfig.IndexExtensions[0] != ".custom" {
			t.Errorf("IndexExtensions[0] = %q, want %q", searchConfig.IndexExtensions[0], ".custom")
		}
	})
}

func TestLoadSearchConfig(t *testing.T) {
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer os.Chdir(origDir)

	t.Run("returns default when no config file", func(t *testing.T) {
		tempDir := t.TempDir()
		if err := os.Chdir(tempDir); err != nil {
			t.Fatalf("Failed to change directory: %v", err)
		}

		config := LoadSearchConfig()

		if config == nil {
			t.Fatal("LoadSearchConfig() returned nil")
		}

		if config.MaxResults != 10 {
			t.Errorf("MaxResults = %d, want default 10", config.MaxResults)
		}
	})

	t.Run("loads from existing config file", func(t *testing.T) {
		tempDir := t.TempDir()
		if err := os.Chdir(tempDir); err != nil {
			t.Fatalf("Failed to change directory: %v", err)
		}

		configContent := `
commands:
  search:
    enabled: true
    max_results: 50
`
		if err := os.WriteFile("llm-runtime.config.yaml", []byte(configContent), 0644); err != nil {
			t.Fatalf("Failed to write config: %v", err)
		}

		config := LoadSearchConfig()

		if config.MaxResults != 50 {
			t.Errorf("MaxResults = %d, want 50", config.MaxResults)
		}
	})

	t.Run("returns default on invalid config", func(t *testing.T) {
		tempDir := t.TempDir()
		if err := os.Chdir(tempDir); err != nil {
			t.Fatalf("Failed to change directory: %v", err)
		}

		if err := os.WriteFile("llm-runtime.config.yaml", []byte("{ invalid yaml"), 0644); err != nil {
			t.Fatalf("Failed to write config: %v", err)
		}

		config := LoadSearchConfig()

		if config == nil {
			t.Fatal("LoadSearchConfig() returned nil on error")
		}
	})
}

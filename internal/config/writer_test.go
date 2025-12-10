package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/computerscienceiscool/llm-runtime/internal/search"
	"github.com/spf13/viper"
)

func TestSaveConfig(t *testing.T) {
	t.Run("saves config to new file", func(t *testing.T) {
		tempDir := t.TempDir()
		configPath := filepath.Join(tempDir, "new_config.yaml")

		config := &FullConfig{}
		SetFullConfigDefaults(config)
		config.Repository.Root = "/test/path"

		err := SaveConfig(config, configPath)
		if err != nil {
			t.Fatalf("SaveConfig() error = %v", err)
		}

		// Verify file exists
		if _, err := os.Stat(configPath); os.IsNotExist(err) {
			t.Error("Config file was not created")
		}

		// Read back and verify
		data, err := os.ReadFile(configPath)
		if err != nil {
			t.Fatalf("Failed to read saved config: %v", err)
		}

		if !strings.Contains(string(data), "/test/path") {
			t.Error("Saved config doesn't contain expected repository root")
		}
	})

	t.Run("creates parent directories if needed", func(t *testing.T) {
		tempDir := t.TempDir()
		configPath := filepath.Join(tempDir, "nested", "dir", "config.yaml")

		config := &FullConfig{}
		SetFullConfigDefaults(config)

		err := SaveConfig(config, configPath)
		if err != nil {
			t.Fatalf("SaveConfig() error = %v", err)
		}

		if _, err := os.Stat(configPath); os.IsNotExist(err) {
			t.Error("Config file was not created in nested directory")
		}
	})

	t.Run("overwrites existing file", func(t *testing.T) {
		tempDir := t.TempDir()
		configPath := filepath.Join(tempDir, "existing.yaml")

		// Create initial file
		if err := os.WriteFile(configPath, []byte("old content"), 0644); err != nil {
			t.Fatalf("Failed to create initial file: %v", err)
		}

		config := &FullConfig{}
		SetFullConfigDefaults(config)
		config.Repository.Root = "/new/path"

		err := SaveConfig(config, configPath)
		if err != nil {
			t.Fatalf("SaveConfig() error = %v", err)
		}

		data, err := os.ReadFile(configPath)
		if err != nil {
			t.Fatalf("Failed to read config: %v", err)
		}

		if strings.Contains(string(data), "old content") {
			t.Error("Old content should have been overwritten")
		}

		if !strings.Contains(string(data), "/new/path") {
			t.Error("New content not found in file")
		}
	})

	t.Run("preserves all fields", func(t *testing.T) {
		tempDir := t.TempDir()
		configPath := filepath.Join(tempDir, "full_config.yaml")

		config := &FullConfig{}
		SetFullConfigDefaults(config)
		config.Commands.Search.Enabled = true
		config.Commands.Search.MaxResults = 50
		config.Commands.Search.MinSimilarityScore = 0.8

		err := SaveConfig(config, configPath)
		if err != nil {
			t.Fatalf("SaveConfig() error = %v", err)
		}

		// Load back using viper
		v := viper.New()
		v.SetConfigFile(configPath)
		if err := v.ReadInConfig(); err != nil {
			t.Fatalf("Failed to load saved config: %v", err)
		}

		if !v.GetBool("commands.search.enabled") {
			t.Error("Search.Enabled not preserved")
		}

		if v.GetInt("commands.search.max_results") != 50 {
			t.Errorf("MaxResults = %d, want 50", v.GetInt("commands.search.max_results"))
		}
	})
}

func TestUpdateSearchConfigInFile(t *testing.T) {
	t.Run("updates existing config file", func(t *testing.T) {
		tempDir := t.TempDir()
		configPath := filepath.Join(tempDir, "existing.yaml")

		// Create initial config
		initialConfig := &FullConfig{}
		SetFullConfigDefaults(initialConfig)
		initialConfig.Repository.Root = "/original/root"
		if err := SaveConfig(initialConfig, configPath); err != nil {
			t.Fatalf("Failed to save initial config: %v", err)
		}

		// Update search config
		searchConfig := &search.SearchConfig{
			Enabled:            true,
			MaxResults:         100,
			MinSimilarityScore: 0.9,
			VectorDBPath:       "/custom/vector.db",
			EmbeddingModel:     "custom-model",
			MaxPreviewLength:   1000,
			ChunkSize:          2000,
			OllamaURL:          "http://localhost:11434",
			IndexExtensions:    []string{".custom"},
			MaxFileSize:        2048,
		}

		err := UpdateSearchConfigInFile(configPath, searchConfig)
		if err != nil {
			t.Fatalf("UpdateSearchConfigInFile() error = %v", err)
		}

		// Reload and verify using viper
		v := viper.New()
		v.SetConfigFile(configPath)
		if err := v.ReadInConfig(); err != nil {
			t.Fatalf("Failed to reload config: %v", err)
		}

		// Check search config updated
		if !v.GetBool("commands.search.enabled") {
			t.Error("Search.Enabled not updated")
		}

		if v.GetInt("commands.search.max_results") != 100 {
			t.Errorf("MaxResults = %d, want 100", v.GetInt("commands.search.max_results"))
		}

		// Check original config preserved
		if v.GetString("repository.root") != "/original/root" {
			t.Errorf("Repository.Root = %q, want %q (should be preserved)", v.GetString("repository.root"), "/original/root")
		}
	})

	t.Run("creates new config if file doesn't exist", func(t *testing.T) {
		tempDir := t.TempDir()
		configPath := filepath.Join(tempDir, "new.yaml")

		searchConfig := &search.SearchConfig{
			Enabled:    true,
			MaxResults: 25,
		}

		err := UpdateSearchConfigInFile(configPath, searchConfig)
		if err != nil {
			t.Fatalf("UpdateSearchConfigInFile() error = %v", err)
		}

		// File should exist now
		if _, err := os.Stat(configPath); os.IsNotExist(err) {
			t.Error("Config file was not created")
		}

		v := viper.New()
		v.SetConfigFile(configPath)
		if err := v.ReadInConfig(); err != nil {
			t.Fatalf("Failed to load config: %v", err)
		}

		if !v.GetBool("commands.search.enabled") {
			t.Error("Search.Enabled not set in new file")
		}
	})
}

func TestEnableSearchInConfig(t *testing.T) {
	t.Run("enables search in new config", func(t *testing.T) {
		tempDir := t.TempDir()
		configPath := filepath.Join(tempDir, "enable_search.yaml")

		err := EnableSearchInConfig(configPath)
		if err != nil {
			t.Fatalf("EnableSearchInConfig() error = %v", err)
		}

		v := viper.New()
		v.SetConfigFile(configPath)
		if err := v.ReadInConfig(); err != nil {
			t.Fatalf("Failed to load config: %v", err)
		}

		if !v.GetBool("commands.search.enabled") {
			t.Error("Search should be enabled")
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
}

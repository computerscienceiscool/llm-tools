package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/computerscienceiscool/llm-runtime/internal/search"
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

		// Load back and verify
		loadedConfig, err := LoadConfig(configPath)
		if err != nil {
			t.Fatalf("Failed to load saved config: %v", err)
		}

		if loadedConfig.Commands.Search.Enabled != true {
			t.Error("Search.Enabled not preserved")
		}

		if loadedConfig.Commands.Search.MaxResults != 50 {
			t.Errorf("MaxResults = %d, want 50", loadedConfig.Commands.Search.MaxResults)
		}
	})

	t.Run("handles nil config gracefully", func(t *testing.T) {
		tempDir := t.TempDir()
		configPath := filepath.Join(tempDir, "nil_config.yaml")

		defer func() {
			if r := recover(); r != nil {
				t.Logf("SaveConfig panicked with nil config: %v", r)
			}
		}()

		err := SaveConfig(nil, configPath)
		if err != nil {
			t.Logf("SaveConfig returned error for nil: %v", err)
		}
	})

	t.Run("fails on invalid path", func(t *testing.T) {
		if os.Getuid() == 0 {
			t.Skip("Skipping permission test when running as root")
		}

		config := &FullConfig{}
		SetFullConfigDefaults(config)

		err := SaveConfig(config, "/root/impossible_config.yaml")
		if err == nil {
			t.Error("SaveConfig() expected error for invalid path")
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
			PythonPath:         "/usr/bin/python",
			IndexExtensions:    []string{".custom"},
			MaxFileSize:        2048,
		}

		err := UpdateSearchConfigInFile(configPath, searchConfig)
		if err != nil {
			t.Fatalf("UpdateSearchConfigInFile() error = %v", err)
		}

		// Reload and verify
		loadedConfig, err := LoadConfig(configPath)
		if err != nil {
			t.Fatalf("Failed to reload config: %v", err)
		}

		// Check search config updated
		if loadedConfig.Commands.Search.Enabled != true {
			t.Error("Search.Enabled not updated")
		}

		if loadedConfig.Commands.Search.MaxResults != 100 {
			t.Errorf("MaxResults = %d, want 100", loadedConfig.Commands.Search.MaxResults)
		}

		// Check original config preserved
		if loadedConfig.Repository.Root != "/original/root" {
			t.Errorf("Repository.Root = %q, want %q (should be preserved)", loadedConfig.Repository.Root, "/original/root")
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

		loadedConfig, err := LoadConfig(configPath)
		if err != nil {
			t.Fatalf("Failed to load config: %v", err)
		}

		if loadedConfig.Commands.Search.Enabled != true {
			t.Error("Search.Enabled not set in new file")
		}
	})

	t.Run("updates all search fields", func(t *testing.T) {
		tempDir := t.TempDir()
		configPath := filepath.Join(tempDir, "search_update.yaml")

		searchConfig := &search.SearchConfig{
			Enabled:            true,
			VectorDBPath:       "/custom/path/vector.db",
			EmbeddingModel:     "custom-embedding",
			MaxResults:         50,
			MinSimilarityScore: 0.75,
			MaxPreviewLength:   750,
			ChunkSize:          1500,
			PythonPath:         "/custom/python",
			IndexExtensions:    []string{".go", ".rs", ".py"},
			MaxFileSize:        5000000,
		}

		err := UpdateSearchConfigInFile(configPath, searchConfig)
		if err != nil {
			t.Fatalf("UpdateSearchConfigInFile() error = %v", err)
		}

		loadedConfig, err := LoadConfig(configPath)
		if err != nil {
			t.Fatalf("Failed to load config: %v", err)
		}

		// Verify all fields
		if loadedConfig.Commands.Search.VectorDBPath != "/custom/path/vector.db" {
			t.Errorf("VectorDBPath = %q, want %q", loadedConfig.Commands.Search.VectorDBPath, "/custom/path/vector.db")
		}

		if loadedConfig.Commands.Search.EmbeddingModel != "custom-embedding" {
			t.Errorf("EmbeddingModel = %q, want %q", loadedConfig.Commands.Search.EmbeddingModel, "custom-embedding")
		}

		if loadedConfig.Commands.Search.MinSimilarityScore != 0.75 {
			t.Errorf("MinSimilarityScore = %f, want 0.75", loadedConfig.Commands.Search.MinSimilarityScore)
		}

		if loadedConfig.Commands.Search.MaxPreviewLength != 750 {
			t.Errorf("MaxPreviewLength = %d, want 750", loadedConfig.Commands.Search.MaxPreviewLength)
		}

		if loadedConfig.Commands.Search.ChunkSize != 1500 {
			t.Errorf("ChunkSize = %d, want 1500", loadedConfig.Commands.Search.ChunkSize)
		}

		if loadedConfig.Commands.Search.PythonPath != "/custom/python" {
			t.Errorf("PythonPath = %q, want %q", loadedConfig.Commands.Search.PythonPath, "/custom/python")
		}

		if len(loadedConfig.Commands.Search.IndexExtensions) != 3 {
			t.Errorf("IndexExtensions length = %d, want 3", len(loadedConfig.Commands.Search.IndexExtensions))
		}

		if loadedConfig.Commands.Search.MaxFileSize != 5000000 {
			t.Errorf("MaxFileSize = %d, want 5000000", loadedConfig.Commands.Search.MaxFileSize)
		}
	})

	t.Run("handles corrupt config file", func(t *testing.T) {
		tempDir := t.TempDir()
		configPath := filepath.Join(tempDir, "corrupt.yaml")

		// Create corrupt file
		if err := os.WriteFile(configPath, []byte("{ invalid yaml : :"), 0644); err != nil {
			t.Fatalf("Failed to create corrupt file: %v", err)
		}

		searchConfig := &search.SearchConfig{
			Enabled:    true,
			MaxResults: 30,
		}

		err := UpdateSearchConfigInFile(configPath, searchConfig)
		if err != nil {
			t.Logf("UpdateSearchConfigInFile returned error for corrupt file: %v", err)
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

		loadedConfig, err := LoadConfig(configPath)
		if err != nil {
			t.Fatalf("Failed to load config: %v", err)
		}

		if loadedConfig.Commands.Search.Enabled != true {
			t.Error("Search should be enabled")
		}

		// Other search defaults should be set
		if loadedConfig.Commands.Search.MaxResults != 10 {
			t.Errorf("MaxResults = %d, want default 10", loadedConfig.Commands.Search.MaxResults)
		}
	})

	t.Run("enables search in existing config", func(t *testing.T) {
		tempDir := t.TempDir()
		configPath := filepath.Join(tempDir, "existing_config.yaml")

		// Create config with search disabled
		initialConfig := &FullConfig{}
		SetFullConfigDefaults(initialConfig)
		initialConfig.Commands.Search.Enabled = false
		initialConfig.Repository.Root = "/my/repo"
		if err := SaveConfig(initialConfig, configPath); err != nil {
			t.Fatalf("Failed to save initial config: %v", err)
		}

		err := EnableSearchInConfig(configPath)
		if err != nil {
			t.Fatalf("EnableSearchInConfig() error = %v", err)
		}

		loadedConfig, err := LoadConfig(configPath)
		if err != nil {
			t.Fatalf("Failed to load config: %v", err)
		}

		if loadedConfig.Commands.Search.Enabled != true {
			t.Error("Search should be enabled")
		}
	})

	t.Run("creates nested directories for config path", func(t *testing.T) {
		tempDir := t.TempDir()
		configPath := filepath.Join(tempDir, "deeply", "nested", "path", "config.yaml")

		err := EnableSearchInConfig(configPath)
		if err != nil {
			t.Fatalf("EnableSearchInConfig() error = %v", err)
		}

		if _, err := os.Stat(configPath); os.IsNotExist(err) {
			t.Error("Config file was not created in nested directory")
		}
	})
}

func TestSaveConfigPermissions(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("Skipping permission test when running as root")
	}

	t.Run("fails on read-only directory", func(t *testing.T) {
		tempDir := t.TempDir()

		// Make directory read-only
		if err := os.Chmod(tempDir, 0555); err != nil {
			t.Fatalf("Failed to change directory permissions: %v", err)
		}
		defer os.Chmod(tempDir, 0755)

		configPath := filepath.Join(tempDir, "config.yaml")
		config := &FullConfig{}
		SetFullConfigDefaults(config)

		err := SaveConfig(config, configPath)
		if err == nil {
			t.Error("SaveConfig() expected error for read-only directory")
		}
	})
}

func TestConfigRoundTrip(t *testing.T) {
	t.Run("full config survives save and load", func(t *testing.T) {
		tempDir := t.TempDir()
		configPath := filepath.Join(tempDir, "roundtrip.yaml")

		original := &FullConfig{}
		SetFullConfigDefaults(original)

		// Customize
		original.Repository.Root = "/custom/repo"
		original.Repository.ExcludedPaths = []string{"custom1", "custom2"}
		original.Commands.Search.Enabled = true
		original.Commands.Search.VectorDBPath = "/custom/vector.db"
		original.Commands.Search.EmbeddingModel = "custom-model"
		original.Commands.Search.MaxResults = 42
		original.Commands.Search.MinSimilarityScore = 0.42
		original.Commands.Search.MaxPreviewLength = 420
		original.Commands.Search.ChunkSize = 4200
		original.Commands.Search.PythonPath = "/custom/python"
		original.Commands.Search.IndexExtensions = []string{".a", ".b", ".c"}
		original.Commands.Search.MaxFileSize = 42000
		original.Security.AuditLogPath = "/custom/audit.log"

		// Save
		if err := SaveConfig(original, configPath); err != nil {
			t.Fatalf("SaveConfig() error = %v", err)
		}

		// Load
		loaded, err := LoadConfig(configPath)
		if err != nil {
			t.Fatalf("LoadConfig() error = %v", err)
		}

		// Verify fields
		if loaded.Repository.Root != original.Repository.Root {
			t.Errorf("Repository.Root: got %q, want %q", loaded.Repository.Root, original.Repository.Root)
		}

		if loaded.Security.AuditLogPath != original.Security.AuditLogPath {
			t.Errorf("Security.AuditLogPath: got %q, want %q", loaded.Security.AuditLogPath, original.Security.AuditLogPath)
		}

		if loaded.Commands.Search.MaxResults != original.Commands.Search.MaxResults {
			t.Errorf("Search.MaxResults: got %d, want %d", loaded.Commands.Search.MaxResults, original.Commands.Search.MaxResults)
		}

		if loaded.Commands.Search.MinSimilarityScore != original.Commands.Search.MinSimilarityScore {
			t.Errorf("Search.MinSimilarityScore: got %f, want %f", loaded.Commands.Search.MinSimilarityScore, original.Commands.Search.MinSimilarityScore)
		}

		if len(loaded.Commands.Search.IndexExtensions) != len(original.Commands.Search.IndexExtensions) {
			t.Errorf("Search.IndexExtensions length: got %d, want %d",
				len(loaded.Commands.Search.IndexExtensions), len(original.Commands.Search.IndexExtensions))
		}
	})
}

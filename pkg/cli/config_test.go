package cli

import (
	"testing"
	"time"

	"github.com/spf13/viper"
)

// TestBuildConfig_Defaults tests buildConfig with default values
func TestBuildConfig_Defaults(t *testing.T) {
	// Reset viper
	viper.Reset()

	// Set required values
	viper.Set("root", "/tmp/test")
	viper.Set("exec-timeout", "30s")
	viper.Set("io-timeout", "10s")

	cfg, err := buildConfig()
	if err != nil {
		t.Fatalf("buildConfig() unexpected error: %v", err)
	}

	if cfg.RepositoryRoot != "/tmp/test" {
		t.Errorf("RepositoryRoot = %q, want %q", cfg.RepositoryRoot, "/tmp/test")
	}

	if cfg.ExecTimeout != 30*time.Second {
		t.Errorf("ExecTimeout = %v, want %v", cfg.ExecTimeout, 30*time.Second)
	}

	if cfg.IOTimeout != 10*time.Second {
		t.Errorf("IOTimeout = %v, want %v", cfg.IOTimeout, 10*time.Second)
	}
}

// TestBuildConfig_InvalidExecTimeout tests invalid exec timeout
func TestBuildConfig_InvalidExecTimeout(t *testing.T) {
	viper.Reset()
	viper.Set("root", "/tmp/test")
	viper.Set("exec-timeout", "invalid")
	viper.Set("io-timeout", "10s")

	_, err := buildConfig()
	if err == nil {
		t.Error("buildConfig() expected error for invalid exec-timeout")
	}
}

// TestBuildConfig_InvalidIOTimeout tests invalid IO timeout
func TestBuildConfig_InvalidIOTimeout(t *testing.T) {
	viper.Reset()
	viper.Set("root", "/tmp/test")
	viper.Set("exec-timeout", "30s")
	viper.Set("io-timeout", "invalid")

	_, err := buildConfig()
	if err == nil {
		t.Error("buildConfig() expected error for invalid io-timeout")
	}
}

// TestBuildConfig_CustomValues tests buildConfig with custom values
func TestBuildConfig_CustomValues(t *testing.T) {
	viper.Reset()

	viper.Set("root", "/custom/path")
	viper.Set("max-size", int64(2048))
	viper.Set("verbose", true)
	viper.Set("exec-timeout", "60s")
	viper.Set("io-timeout", "20s")

	cfg, err := buildConfig()
	if err != nil {
		t.Fatalf("buildConfig() unexpected error: %v", err)
	}

	if cfg.RepositoryRoot != "/custom/path" {
		t.Errorf("RepositoryRoot = %q, want %q", cfg.RepositoryRoot, "/custom/path")
	}

	if cfg.MaxFileSize != 2048 {
		t.Errorf("MaxFileSize = %d, want %d", cfg.MaxFileSize, 2048)
	}

	if !cfg.Verbose {
		t.Error("Verbose should be true")
	}

	if cfg.ExecTimeout != 60*time.Second {
		t.Errorf("ExecTimeout = %v, want %v", cfg.ExecTimeout, 60*time.Second)
	}
}

// TestBuildConfig_ExecWhitelistFromConfig tests loading exec whitelist from config
func TestBuildConfig_ExecWhitelistFromConfig(t *testing.T) {
	viper.Reset()

	viper.Set("root", "/tmp/test")
	viper.Set("exec-timeout", "30s")
	viper.Set("io-timeout", "10s")
	viper.Set("commands.exec.whitelist", []string{"go test", "npm test"})

	cfg, err := buildConfig()
	if err != nil {
		t.Fatalf("buildConfig() unexpected error: %v", err)
	}

	if len(cfg.ExecWhitelist) != 2 {
		t.Errorf("ExecWhitelist length = %d, want 2", len(cfg.ExecWhitelist))
	}

	if cfg.ExecWhitelist[0] != "go test" {
		t.Errorf("ExecWhitelist[0] = %q, want %q", cfg.ExecWhitelist[0], "go test")
	}
}

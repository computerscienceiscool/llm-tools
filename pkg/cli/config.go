package cli

import (
	"fmt"
	"time"

	"github.com/computerscienceiscool/llm-runtime/pkg/app"
	"github.com/computerscienceiscool/llm-runtime/pkg/config"
	"github.com/computerscienceiscool/llm-runtime/pkg/dynrepo"
	"github.com/spf13/viper"
)

// initConfig reads in config file and ENV variables if set
func initConfig() {
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			// Config file was found but another error was produced
			fmt.Printf("Error reading config file: %v\n", err)
		}
		// Config file not found; using defaults and flags
	}
}

// buildConfig constructs a config.Config from Viper values
func buildConfig() (*config.Config, error) {
	// Determine repository root
	rootPath := viper.GetString("root")
	if rootPath == "." {
		// Default: create a dynamic repo
		dir, _, err := dynrepo.CreateRepo()
		if err != nil {
			return nil, fmt.Errorf("failed to create dynamic repo: %w", err)
		}
		viper.Set("root", dir)
	}

	cfg := &config.Config{
		RepositoryRoot:      viper.GetString("root"),
		MaxFileSize:         viper.GetInt64("max-size"),
		MaxWriteSize:        viper.GetInt64("max-write-size"),
		ExcludedPaths:       viper.GetStringSlice("exclude"),
		Interactive:         viper.GetBool("interactive"),
		InputFile:           viper.GetString("input"),
		OutputFile:          viper.GetString("output"),
		JSONOutput:          viper.GetBool("json"),
		Verbose:             viper.GetBool("verbose"),
		RequireConfirmation: viper.GetBool("require-confirmation"),
		BackupBeforeWrite:   viper.GetBool("backup"),
		AllowedExtensions:   viper.GetStringSlice("allowed-extensions"),
		ForceWrite:          viper.GetBool("force"),
		ExecWhitelist:       viper.GetStringSlice("exec-whitelist"),
		ExecMemoryLimit:     viper.GetString("exec-memory"),
		ExecCPULimit:        viper.GetInt("exec-cpu"),
		ExecContainerImage:  viper.GetString("exec-image"),
		ExecNetworkEnabled:  viper.GetBool("exec-network"),
		IOContainerImage:    viper.GetString("io-image"),
		IOMemoryLimit:       viper.GetString("io-memory"),
		IOCPULimit:          viper.GetInt("io-cpu"),
	}

	// Parse timeout durations
	execTimeoutStr := viper.GetString("exec-timeout")
	execTimeout, err := time.ParseDuration(execTimeoutStr)
	if err != nil {
		return nil, fmt.Errorf("invalid exec-timeout: %w", err)
	}
	cfg.ExecTimeout = execTimeout

	ioTimeoutStr := viper.GetString("io-timeout")
	ioTimeout, err := time.ParseDuration(ioTimeoutStr)
	if err != nil {
		return nil, fmt.Errorf("invalid io-timeout: %w", err)
	}
	cfg.IOTimeout = ioTimeout

	// Load container pool configuration
	cfg.ContainerPool = config.PoolConfig{
		Enabled:             viper.GetBool("container_pool.enabled"),
		Size:                viper.GetInt("container_pool.size"),
		MaxUsesPerContainer: viper.GetInt("container_pool.max_uses_per_container"),
		IdleTimeout:         viper.GetDuration("container_pool.idle_timeout"),
		HealthCheckInterval: viper.GetDuration("container_pool.health_check_interval"),
		StartupContainers:   viper.GetInt("container_pool.startup_containers"),
	}

	// If exec-whitelist is empty from flags, try loading from config file
	if len(cfg.ExecWhitelist) == 0 {
		// Viper can read from nested config like commands.exec.whitelist
		if viper.IsSet("commands.exec.whitelist") {
			cfg.ExecWhitelist = viper.GetStringSlice("commands.exec.whitelist")
		}
	}
	//fmt.Printf("DEBUG buildConfig: RepositoryRoot = %s\n", cfg.RepositoryRoot)

	return cfg, nil
}

// bootstrapApp wraps the app.Bootstrap function
func bootstrapApp(cfg *config.Config) (*app.App, error) {
	return app.Bootstrap(cfg)
}

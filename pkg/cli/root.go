package cli

import (
	"fmt"

	"github.com/computerscienceiscool/llm-runtime/pkg/config"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var rootCmd = &cobra.Command{
	Use:   "llm-runtime",
	Short: "LLM File Access Tool - Command interpreter for LLMs",
	Long: `llm-runtime enables Large Language Models to interact with local filesystems
and execute sandboxed commands. It processes commands like <open>, <write>, <exec>, and <search>.`,
	RunE: runRoot,
}

func init() {
	cobra.OnInitialize(initConfig)

	// Repository flags
	rootCmd.PersistentFlags().String("root", ".", "Repository root directory")
	rootCmd.PersistentFlags().StringSlice("exclude", []string{".git", ".env", "*.key", "*.pem"}, "Comma-separated list of excluded paths")

	// I/O flags
	rootCmd.PersistentFlags().String("input", "", "Input file (default: stdin)")
	rootCmd.PersistentFlags().String("output", "", "Output file (default: stdout)")
	rootCmd.PersistentFlags().Bool("interactive", false, "Run in interactive mode")

	// Output flags
	rootCmd.PersistentFlags().Bool("json", false, "Output in JSON format")
	rootCmd.PersistentFlags().Bool("verbose", false, "Verbose output")

	// File operation flags
	rootCmd.PersistentFlags().Int64("max-size", 1048576, "Maximum file size in bytes (default 1MB)")
	rootCmd.PersistentFlags().Int64("max-write-size", 102400, "Maximum file size in bytes for writing (default 100KB)")
	rootCmd.PersistentFlags().StringSlice("allowed-extensions", []string{".go", ".py", ".js", ".md", ".txt", ".json", ".yaml", ".yml", ".toml"}, "Comma-separated list of allowed file extensions for writing")
	rootCmd.PersistentFlags().Bool("backup", true, "Create backup before overwriting files")
	rootCmd.PersistentFlags().Bool("require-confirmation", false, "Require confirmation for write operations")
	rootCmd.PersistentFlags().Bool("force", false, "Force write even if conflicts exist")

	// Exec flags
	rootCmd.PersistentFlags().String("exec-timeout", "30s", "Timeout for exec commands")
	rootCmd.PersistentFlags().String("exec-memory", "512m", "Memory limit for containers")
	rootCmd.PersistentFlags().Int("exec-cpu", 1, "CPU limit for containers")
	rootCmd.PersistentFlags().String("exec-image", "python-go", "Docker image for exec commands")
	rootCmd.PersistentFlags().Bool("exec-network", false, "Enable network access in containers")
	rootCmd.PersistentFlags().StringSlice("exec-whitelist", []string{}, "Comma-separated list of allowed exec commands")

	// I/O Containerization flags
	rootCmd.PersistentFlags().String("io-image", "llm-runtime-io:latest", "Docker image for I/O operations")
	rootCmd.PersistentFlags().String("io-timeout", "60s", "Timeout for I/O operations")
	rootCmd.PersistentFlags().String("io-memory", "256m", "Memory limit for I/O containers")
	rootCmd.PersistentFlags().Int("io-cpu", 1, "CPU limit for I/O containers")

	// Bind flags to viper
	viper.BindPFlags(rootCmd.PersistentFlags())
}

func runRoot(cmd *cobra.Command, args []string) error {
	// Build config from viper
	cfg, err := buildConfig()
	if err != nil {
		return fmt.Errorf("failed to build config: %w", err)
	}

	// Bootstrap and run application
	app, err := bootstrapApp(cfg)
	if err != nil {
		return fmt.Errorf("bootstrap failed: %w", err)
	}
	defer app.Close()

	return app.Run()
}

// Execute runs the root command
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	// Set all default values in Viper
	config.SetViperDefaults()

	// Set default config file name
	viper.SetConfigName("llm-runtime.config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("$HOME")

	// Enable environment variables with LLM prefix
	viper.SetEnvPrefix("LLM")
	viper.AutomaticEnv()
}

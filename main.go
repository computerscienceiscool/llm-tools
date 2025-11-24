package main

import (
	"fmt"
	"os"

	"github.com/computerscienceiscool/llm-tools/cmd"
	"github.com/computerscienceiscool/llm-tools/internal/config"
	"github.com/computerscienceiscool/llm-tools/internal/core"
	"github.com/computerscienceiscool/llm-tools/internal/handlers"
	"github.com/computerscienceiscool/llm-tools/internal/infrastructure"
	"github.com/computerscienceiscool/llm-tools/internal/parser"
	"github.com/computerscienceiscool/llm-tools/internal/security"
)

func main() {
	// Load configuration
	configLoader := config.NewConfigLoader()
	config, err := configLoader.LoadConfig("")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	// Create session
	session, err := core.NewSession(config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating session: %v\n", err)
		os.Exit(1)
	}

	// Create dependencies
	validator := security.NewPathValidator()
	auditor := security.NewAuditLogger()
	parser := parser.NewCommandParser()
	dockerClient := infrastructure.NewDockerClient()
	fileHandler := handlers.NewFileHandler(validator, auditor)
	execHandler := handlers.NewExecHandler(dockerClient.(handlers.DockerClient))
	searchHandler := handlers.NewSearchHandler()

	// Create executor
	executor := core.NewCommandExecutor(
		session,
		fileHandler,
		execHandler,
		searchHandler,
		validator,
	)

	// Create and run CLI app
	app := cmd.NewCLIApp(configLoader, executor, parser)
	if err := app.Execute(os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

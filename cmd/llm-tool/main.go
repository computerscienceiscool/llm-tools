package main

import (
	"log"
	"os"

	"github.com/computerscienceiscool/llm-tools/internal/app"
	"github.com/computerscienceiscool/llm-tools/internal/cli"
)

func main() {
	// Parse command-line flags
	flags := cli.ParseFlags()

	// Handle search-related commands before full bootstrap
	if flags.HasSearchCommand() {
		application, err := app.Bootstrap(flags.Config)
		if err != nil {
			log.Fatalf("Bootstrap failed: %v", err)
		}

		if err := application.RunSearchCommand(flags); err != nil {
			log.Fatalf("Search command failed: %v", err)
		}
		return
	}

	// Bootstrap the application
	application, err := app.Bootstrap(flags.Config)
	if err != nil {
		log.Fatalf("Bootstrap failed: %v", err)
	}

	// Run the application
	if err := application.Run(); err != nil {
		log.Fatalf("Application error: %v", err)
		os.Exit(1)
	}
}

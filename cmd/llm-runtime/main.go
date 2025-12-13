package main

import (
	"log"
	"os"

	"github.com/computerscienceiscool/llm-runtime/pkg/cli"
)

func main() {
	os.Exit(run())
}

func run() int {
	if err := cli.Execute(); err != nil {
		log.Printf("Error: %v", err)
		return 1
	}
	return 0
}

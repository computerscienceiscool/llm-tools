# Makefile for LLM File Access Tool

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GOFMT=$(GOCMD) fmt
GOVET=$(GOCMD) vet

# Binary name
BINARY_NAME=llm-tooCommit using grok and push to current branch


# Commit using grok and push to current branch
# Runs build and tests first to ensure code quality
commit: build test
	@echo "Build and tests passed!"
	@echo ""
	@echo "Staging modified tracked files..."
	@git add -u

	@echo "Generating commit message with grok..."
	@grok commit | git commit -F - || { echo "Nothing to commit."; exit 1; }

	@echo "Pushing to current branch..."
	@git push origin $$(git rev-parse --abbrev-ref HEAD)

	@echo "Commit and push complete."

debug-path:
	@echo "PATH is: $$PATH"
	@which grok || echo "grok still not found"

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
BINARY_NAME=llm-runtime
BINARY_PATH=./$(BINARY_NAME)

# Source path
CMD_PATH=./cmd/llm-runtime

# Installation path
INSTALL_PATH=/usr/local/bin

# Build flags
LDFLAGS=-ldflags "-s -w"

.PHONY: all build test clean install uninstall fmt vet deps run demo example exec-demo

# Default target
all: test build

# Build the binary
build:
	@echo "Building $(BINARY_NAME)..."
	$(GOBUILD) $(LDFLAGS) -o $(BINARY_PATH) $(CMD_PATH)
	@echo "Build complete: $(BINARY_PATH)"

# Run tests
test:
	@echo "Running tests..."
	$(GOTEST) -v -race -cover ./...

# Run tests with coverage report
test-coverage:
	@echo "Running tests with coverage..."
	$(GOTEST) -v -race -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

# Run benchmarks
bench:
	@echo "Running benchmarks..."
	$(GOTEST) -bench=. -benchmem ./...

# Clean build artifacts
clean:
	@echo "Cleaning..."
	@rm -f $(BINARY_PATH)
	@rm -f coverage.out coverage.html
	@rm -f audit.log
	@rm -f llm-runtime.log
	@echo "Clean complete"
	@rm -f test_output.txt
	@rm -f *.bak.*

# Install binary to system
install: build
	@echo "Installing $(BINARY_NAME) to $(INSTALL_PATH)..."
	@sudo cp $(BINARY_PATH) $(INSTALL_PATH)/
	@sudo chmod +x $(INSTALL_PATH)/$(BINARY_NAME)
	@echo "Installation complete: $(INSTALL_PATH)/$(BINARY_NAME)"

# Uninstall binary from system
uninstall:
	@echo "Uninstalling $(BINARY_NAME) from $(INSTALL_PATH)..."
	@sudo rm -f $(INSTALL_PATH)/$(BINARY_NAME)
	@echo "Uninstall complete"

# Format code
fmt:
	@echo "Formatting code..."
	$(GOFMT) ./...

# Run go vet
vet:
	@echo "Running go vet..."
	$(GOVET) ./...

# Download dependencies
deps:
	@echo "Downloading dependencies..."
	$(GOMOD) download
	$(GOMOD) tidy

# Run the tool interactively
run: build
	$(BINARY_PATH) --interactive

# Run the demo
demo: build
	@chmod +x scripts/demo.sh
	./scripts/demo.sh

# Run the example usage
example: build
	@chmod +x scripts/example_usage.sh
	./scripts/example_usage.sh

# Run the exec command demo
exec-demo: build
	@echo "Running exec command demonstration..."
	@chmod +x scripts/exec_demo.sh
	./scripts/exec_demo.sh

# Quick test with a simple command
quick-test: build
	@echo "Testing with simple command..."
	@echo "Let me read the README: <open README.md>" | $(BINARY_PATH)

# Test write functionality
test-write: build
	@echo "Testing write functionality..."
	@echo "Creating test file: <write test_output.txt>\nThis is a test file created by the LLM tool.\nCurrent time: $$(date)\nWrite command is working!\n</write>" | $(BINARY_PATH)
	@if [ -f test_output.txt ]; then echo "Write test successful"; echo "File contents:"; cat test_output.txt; rm test_output.txt; else echo "Write test failed"; fi

# Test exec functionality (requires Docker)
test-exec: build
	@echo "Testing exec functionality..."
	@if command -v docker >/dev/null 2>&1; then \
		echo "Testing exec command: <exec echo 'Hello from Docker'>" | $(BINARY_PATH) --exec-enabled; \
	else \
		echo "Docker not available - exec test skipped"; \
	fi

# Test both read and write commands
test-both: build
	@echo "Testing both read and write commands..."
	@echo "First read README: <open README.md>" > temp_input.txt
	@echo "" >> temp_input.txt
	@echo "Now create summary: <write README_SUMMARY.md>" >> temp_input.txt
	@echo "# README Summary" >> temp_input.txt
	@echo "" >> temp_input.txt
	@echo "This is a summary of the LLM File Access Tool." >> temp_input.txt
	@echo "Generated at: $$(date)" >> temp_input.txt
	@echo "</write>" >> temp_input.txt
	@$(BINARY_PATH) --input temp_input.txt
	@rm -f temp_input.txt README_SUMMARY.md

# Test all command types
test-all-commands: build
	@echo "Testing all command types..."
	@if command -v docker >/dev/null 2>&1; then \
		echo "Testing read, write, and exec: <open README.md> <write test.txt>Test content</write> <exec echo 'All commands work'>" | $(BINARY_PATH) --exec-enabled; \
		rm -f test.txt; \
	else \
		echo "Testing read and write only: <open README.md> <write test.txt>Test content</write>" | $(BINARY_PATH); \
		rm -f test.txt; \
	fi

# Development mode - rebuild and run on file changes (requires entr)
watch:
	@echo "Watching for changes (requires 'entr')..."
	@find . -name '*.go' | entr -c make build

# Check code quality
quality: fmt vet test
	@echo "Code quality checks complete"

# Build for multiple platforms
build-all:
	@echo "Building for multiple platforms..."
	@mkdir -p dist
	
	@echo "Building for Linux (amd64)..."
	@GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o dist/$(BINARY_NAME)-linux-amd64 $(CMD_PATH)
	
	@echo "Building for Linux (arm64)..."
	@GOOS=linux GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o dist/$(BINARY_NAME)-linux-arm64 $(CMD_PATH)
	
	@echo "Building for macOS (amd64)..."
	@GOOS=darwin GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o dist/$(BINARY_NAME)-darwin-amd64 $(CMD_PATH)
	
	@echo "Building for macOS (arm64)..."
	@GOOS=darwin GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o dist/$(BINARY_NAME)-darwin-arm64 $(CMD_PATH)
	
	@echo "Building for Windows (amd64)..."
	@GOOS=windows GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o dist/$(BINARY_NAME)-windows-amd64.exe $(CMD_PATH)
	
	@echo "Cross-platform builds complete in dist/"

# Create a release tarball
release: clean build test
	@echo "Creating release tarball..."
	@mkdir -p releases
	@tar -czf releases/$(BINARY_NAME)-$(shell date +%Y%m%d).tar.gz \
		$(BINARY_PATH) \
		README.md \
		docs/SYSTEM_PROMPT.md \
		llm-runtime.config.yaml \
		scripts/demo.sh \
		scripts/example_usage.sh \
		scripts/exec_demo.sh \
		scripts/write_demo.sh
	@echo "Release created: releases/$(BINARY_NAME)-$(shell date +%Y%m%d).tar.gz"

# Check Docker availability
check-docker:
	@if command -v docker >/dev/null 2>&1; then \
		echo "Docker is available"; \
		docker version --format '{{.Server.Version}}' 2>/dev/null | sed 's/^/  Version: /' || echo "  (Could not get version)"; \
	else \
		echo "Docker not found"; \
		echo "  Install Docker to use exec commands:"; \
		echo "  - Linux: curl -fsSL https://get.docker.com | sh"; \
		echo "  - macOS: brew install docker"; \
		echo "  - Windows: Download from docker.com"; \
	fi

# Comprehensive test suite
test-suite: build check-docker
	@echo "Running comprehensive test suite..."
	@echo "1. Unit tests..."
	@$(GOTEST) -v ./...
	@echo "2. Basic functionality..."
	@$(MAKE) test-write
	@echo "3. Security tests..."
	@chmod +x scripts/security_test.sh && ./scripts/security_test.sh
	@if command -v docker >/dev/null 2>&1; then \
		echo "4. Exec functionality..."; \
		$(MAKE) test-exec; \
	else \
		echo "4. Exec functionality... SKIPPED (Docker not available)"; \
	fi
	@echo "Test suite complete!"

# Show help
help:
	@echo "Available targets:"
	@echo "  make build         - Build the binary"
	@echo "  make test          - Run tests"
	@echo "  make test-coverage - Run tests with coverage report"
	@echo "  make bench         - Run benchmarks"
	@echo "  make clean         - Remove build artifacts"
	@echo "  make install       - Install binary to system"
	@echo "  make uninstall     - Remove binary from system"
	@echo "  make fmt           - Format code"
	@echo "  make vet           - Run go vet"
	@echo "  make deps          - Download dependencies"
	@echo "  make run           - Run tool in interactive mode"
	@echo "  make demo          - Run the demo script"
	@echo "  make example       - Run the example usage"
	@echo "  make exec-demo     - Run the exec command demo"
	@echo "  make quick-test    - Quick test with README"
	@echo "  make test-write    - Test write functionality"
	@echo "  make test-exec     - Test exec functionality (requires Docker)"
	@echo "  make test-both     - Test both read and write commands"
	@echo "  make test-all-commands - Test all command types"
	@echo "  make test-suite    - Run comprehensive test suite"
	@echo "  make quality       - Run all quality checks"
	@echo "  make build-all     - Build for multiple platforms"
	@echo "  make release       - Create release tarball"
	@echo "  make check-docker  - Check Docker availability"
	@echo "  make build-io-image    - Build Docker image for containerized I/O"
	@echo "  make check-io-image    - Verify IO container image exists"
	@echo "  make test-io-container - Test containerized I/O operations"
	@echo "  make clean-io-image    - Remove IO container image"
	@echo "  make clean-all         - Full clean including Docker images"
	@echo "  make commit        - Build, test, and commit with auto-generated message"
	@echo "  make help          - Show this help message"

# Commit using grok and push to current branch
# Runs build and tests first to ensure code quality
commit: build test
	@echo "Build and tests passed!"
	@echo ""
	@echo "Staging modified tracked files..."
	@git add -u
	@if git diff --cached --quiet; then \
		echo "Nothing to commit."; \
	else \
		echo "Generating commit message with grok (this may take a moment)..."; \
		grok commit > .commit_msg.tmp; \
		echo "Committing..."; \
		git commit -F .commit_msg.tmp; \
		rm -f .commit_msg.tmp; \
		echo "Pushing to current branch..."; \
		git push origin $$(git rev-parse --abbrev-ref HEAD); \
		echo "Commit and push complete."; \
	fi

debug-path:
	@echo "PATH is: $$PATH"
	@which grok || echo "grok still not found"

# Show only failed tests
test-failures:
	@$(GOTEST) ./... 2>&1 | grep -E "^(FAIL|---.*FAIL)" || echo "All tests passed!"





# Add after the check-docker target (around line 176):

# Build Docker image for containerized I/O (Phase 5)
build-io-image:
	@echo "Building Docker image for containerized I/O..."
	docker build -f Dockerfile.io -t llm-runtime-io:latest .
	@echo "IO container image built: llm-runtime-io:latest"

# Verify IO container image exists
check-io-image:
	@if docker image inspect llm-runtime-io:latest >/dev/null 2>&1; then \
		echo "IO container image found: llm-runtime-io:latest"; \
		docker image inspect llm-runtime-io:latest --format '  Size: {{.Size}} bytes ({{printf "%.1f" (div (div (mul (toFloat64 .Size) 10.0) 1048576.0) 10.0)}} MB)'; \
		docker image inspect llm-runtime-io:latest --format '  Created: {{.Created}}'; 
	else \
		echo "IO container image not found: llm-runtime-io:latest"; \
		echo "Run 'make build-io-image' to build it"; \
		exit 1; \
	fi

# Test containerized I/O operations
test-io-container: build check-io-image
	@echo "Testing containerized I/O operations..."
	@echo "Test: <open README.md>" | $(BINARY_PATH) --io-containerized
	@echo ""
	@echo "Testing containerized write..."
	@echo "Test: <write test_io_output.txt>This file was created using containerized I/O (Phase 5)\nTimestamp: $$(date)</write>" | $(BINARY_PATH) --io-containerized
	@if [ -f test_io_output.txt ]; then \
		echo "Containerized write successful"; \
		echo "File contents:"; \
		cat test_io_output.txt; \
		rm test_io_output.txt; \
	else \
		echo "Containerized write failed"; \
	fi

# Clean Docker images for IO
clean-io-image:
	@echo "Removing IO container image..."
	@docker rmi llm-runtime-io:latest 2>/dev/null || echo "Image not found or already removed"

# Full clean including Docker artifacts
clean-all: clean clean-io-image
	@echo "Full cleanup complete (including Docker images)"

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
BINARY_NAME=llm-tool
BINARY_PATH=./$(BINARY_NAME)

# Installation path
INSTALL_PATH=/usr/local/bin

# Build flags
LDFLAGS=-ldflags "-s -w"

.PHONY: all build test clean install uninstall fmt vet deps run demo example

# Default target
all: test build

# Build the binary
build:
	@echo "Building $(BINARY_NAME)..."
	$(GOBUILD) $(LDFLAGS) -o $(BINARY_PATH) main.go
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
	@rm -f llm-tool.log
	@echo "Clean complete"

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
	@chmod +x demo.sh
	./demo.sh

# Run the example usage
example: build
	@chmod +x example_usage.sh
	./example_usage.sh

# Quick test with a simple command
quick-test: build
	@echo "Testing with simple command..."
	@echo "Let me read the README: <open README.md>" | $(BINARY_PATH)

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
	@GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o dist/$(BINARY_NAME)-linux-amd64 main.go
	
	@echo "Building for Linux (arm64)..."
	@GOOS=linux GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o dist/$(BINARY_NAME)-linux-arm64 main.go
	
	@echo "Building for macOS (amd64)..."
	@GOOS=darwin GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o dist/$(BINARY_NAME)-darwin-amd64 main.go
	
	@echo "Building for macOS (arm64)..."
	@GOOS=darwin GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o dist/$(BINARY_NAME)-darwin-arm64 main.go
	
	@echo "Building for Windows (amd64)..."
	@GOOS=windows GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o dist/$(BINARY_NAME)-windows-amd64.exe main.go
	
	@echo "Cross-platform builds complete in dist/"

# Create a release tarball
release: clean build test
	@echo "Creating release tarball..."
	@mkdir -p releases
	@tar -czf releases/$(BINARY_NAME)-$(shell date +%Y%m%d).tar.gz \
		$(BINARY_PATH) \
		README.md \
		SYSTEM_PROMPT.md \
		llm-tool.config.yaml \
		demo.sh \
		example_usage.sh
	@echo "Release created: releases/$(BINARY_NAME)-$(shell date +%Y%m%d).tar.gz"

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
	@echo "  make quick-test    - Quick test with README"
	@echo "  make quality       - Run all quality checks"
	@echo "  make build-all     - Build for multiple platforms"
	@echo "  make release       - Create release tarball"
	@echo "  make help          - Show this help message"

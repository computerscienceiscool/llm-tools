#!/bin/bash

# Change to project root directory
cd "$(dirname "$0")/.."

# Setup script for LLM File Access Tool
# This script helps you build and install the tool on your system

set -e

echo "======================================"
echo "  LLM File Access Tool - Setup"
echo "======================================"
echo

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Check for Go installation
check_go() {
    echo -n "Checking for Go installation... "
    if command -v go &> /dev/null; then
        GO_VERSION=$(go version | awk '{print $3}')
        echo -e "${GREEN}✓${NC} Found $GO_VERSION"
        
        # Check Go version (need 1.21+)
        GO_VERSION_NUM=$(echo $GO_VERSION | sed 's/go//' | cut -d'.' -f1,2)
        REQUIRED_VERSION="1.21"
        
        if [ "$(printf '%s\n' "$REQUIRED_VERSION" "$GO_VERSION_NUM" | sort -V | head -n1)" = "$REQUIRED_VERSION" ]; then
            echo -e "${GREEN}✓${NC} Go version is sufficient (>= 1.21)"
            return 0
        else
            echo -e "${YELLOW}⚠${NC} Go version is older than 1.21. Please update Go."
            return 1
        fi
    else
        echo -e "${RED}✗${NC} Go not found"
        echo
        echo "Please install Go first:"
        echo "  - macOS: brew install go"
        echo "  - Linux: sudo apt install golang-go (or use official installer)"
        echo "  - Windows: Download from https://golang.org/dl/"
        echo
        echo "Or install via official installer:"
        echo "  wget https://go.dev/dl/go1.21.0.linux-amd64.tar.gz"
        echo "  sudo tar -C /usr/local -xzf go1.21.0.linux-amd64.tar.gz"
        echo "  export PATH=$PATH:/usr/local/go/bin"
        return 1
    fi
}

# Build the tool
build_tool() {
    echo
    echo -e "${BLUE}Building the LLM File Access Tool...${NC}"
    
    if [ -f "Makefile" ]; then
        # Use Makefile if available
        make clean 2>/dev/null || true
        make build
    else
        # Direct build command
        go mod download
        go build -ldflags "-s -w" -o llm-runtime main.go
    fi
    
    if [ -f "llm-runtime" ]; then
        echo -e "${GREEN}✓${NC} Build successful!"
        
        # Make scripts executable
        chmod +x llm-runtime
        [ -f "demo.sh" ] && chmod +x demo.sh
        [ -f "example_usage.sh" ] && chmod +x example_usage.sh
        [ -f "security_test.sh" ] && chmod +x security_test.sh
        
        return 0
    else
        echo -e "${RED}✗${NC} Build failed"
        return 1
    fi
}

# Run tests
run_tests() {
    echo
    echo -e "${BLUE}Running tests...${NC}"
    
    if go test -v -race ./... ; then
        echo -e "${GREEN}✓${NC} All tests passed!"
    else
        echo -e "${YELLOW}⚠${NC} Some tests failed, but the tool may still work"
    fi
}

# Install to system (optional)
install_system() {
    echo
    read -p "Do you want to install llm-runtime to /usr/local/bin? (requires sudo) [y/N]: " -n 1 -r
    echo
    
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        echo -e "${BLUE}Installing to /usr/local/bin...${NC}"
        sudo cp llm-runtime /usr/local/bin/
        sudo chmod +x /usr/local/bin/llm-runtime
        echo -e "${GREEN}✓${NC} Installed successfully!"
        echo "You can now run 'llm-runtime' from anywhere"
    else
        echo "Skipping system installation"
        echo "You can run the tool locally with: ./llm-runtime"
    fi
}

# Quick demo
run_demo() {
    echo
    read -p "Do you want to run a quick demo? [Y/n]: " -n 1 -r
    echo
    
    if [[ ! $REPLY =~ ^[Nn]$ ]]; then
        echo -e "${BLUE}Running demo...${NC}"
        echo
        
        # Create a simple test
        TEST_DIR=$(mktemp -d)
        echo "# Test Repository" > "$TEST_DIR/README.md"
        echo "package main" > "$TEST_DIR/main.go"
        
        echo "Testing the tool with a simple command:"
        echo "Input: 'Check the README <open README.md>'"
        echo "---"
        echo "Check the README <open README.md>" | ./llm-runtime --root "$TEST_DIR"
        
        rm -rf "$TEST_DIR"
        
        echo
        echo -e "${GREEN}✓${NC} Demo complete!"
    fi
}

# Print usage instructions
print_usage() {
    echo
    echo "======================================"
    echo -e "${GREEN}Setup Complete!${NC}"
    echo "======================================"
    echo
    echo "The LLM File Access Tool is ready to use!"
    echo
    echo "Usage examples:"
    echo "  # Basic usage (pipe mode)"
    echo "  echo 'Read file <open main.go>' | ./llm-runtime"
    echo
    echo "  # Interactive mode"
    echo "  ./llm-runtime --interactive"
    echo
    echo "  # With custom repository"
    echo "  ./llm-runtime --root /path/to/repo"
    echo
    echo "Available commands:"
    echo "  make test          - Run tests"
    echo "  make demo          - Run full demo"
    echo "  make example       - Run example with sample Go app"
    echo "  ./security_test.sh - Run security tests"
    echo
    echo "Configuration:"
    echo "  Edit llm-runtime.config.yaml to customize behavior"
    echo
    echo "System Prompt:"
    echo "  See SYSTEM_PROMPT.md for LLM integration instructions"
    echo
    echo "Documentation:"
    echo "  See README.md for complete documentation"
}

# Main setup flow
main() {
    cd "$(dirname "$0")"
    
    echo "Starting setup in: $(pwd)"
    echo
    
    # Check prerequisites
    if ! check_go; then
        exit 1
    fi
    
    # Build the tool
    if ! build_tool; then
        echo -e "${RED}Build failed. Please check the error messages above.${NC}"
        exit 1
    fi
    
    # Run tests
    run_tests
    
    # Optional: Install to system
    install_system
    
    # Optional: Run demo
    run_demo
    
    # Print usage instructions
    print_usage
}

# Run main function
main "$@"

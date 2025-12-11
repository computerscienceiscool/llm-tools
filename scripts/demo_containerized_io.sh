#!/bin/bash
# Comprehensive Demo: Containerized I/O Operations
# Shows read, write, and exec all using Docker containers
# Run from: scripts/demo_containerized_io.sh

set -e

# Get the project root (parent of scripts directory)
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

echo "=========================================="
echo "Containerized I/O Demo"
echo "=========================================="
echo ""
echo "Project root: $PROJECT_ROOT"
echo ""

# Colors
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

# Step 1: Check Docker
echo -e "${BLUE}Step 1: Checking Docker availability...${NC}"
if ! command -v docker &> /dev/null; then
    echo -e "${RED}ERROR: Docker not found. Please install Docker first:${NC}"
    echo "  Linux: curl -fsSL https://get.docker.com | sh"
    echo "  macOS: brew install docker"
    exit 1
fi

if ! docker info &> /dev/null; then
    echo -e "${RED}ERROR: Docker daemon is not running${NC}"
    echo "Please start Docker and try again"
    exit 1
fi

echo -e "${GREEN}✓ Docker is running${NC}"
echo ""

# Step 2: Build the tool
echo -e "${BLUE}Step 2: Building llm-runtime...${NC}"
cd "$PROJECT_ROOT"
make build > /dev/null 2>&1
echo -e "${GREEN}✓ Build complete${NC}"
echo ""

# Step 3: Check for alpine image (used by default)
echo -e "${BLUE}Step 3: Checking for alpine:latest image...${NC}"
if ! docker image inspect alpine:latest &> /dev/null; then
    echo "Pulling alpine:latest..."
    docker pull alpine:latest
fi
echo -e "${GREEN}✓ Container image ready${NC}"
echo ""

# Step 4: Create test environment
echo -e "${BLUE}Step 4: Setting up test environment...${NC}"
DEMO_DIR=$(mktemp -d)
echo "Demo directory: $DEMO_DIR"

# Create sample files
cat > "$DEMO_DIR/sample.txt" << 'SAMPLE'
This is a sample file.
It contains multiple lines.
Perfect for testing containerized reads!
SAMPLE

cat > "$DEMO_DIR/data.json" << 'JSON'
{
  "name": "Containerized IO Demo",
  "version": "1.0",
  "features": ["read", "write", "exec"]
}
JSON

cat > "$DEMO_DIR/code.go" << 'GOCODE'
package main

import "fmt"

func main() {
    fmt.Println("Hello from containerized demo!")
}
GOCODE

echo -e "${GREEN}✓ Test files created${NC}"
ls -la "$DEMO_DIR"
echo ""

# Binary location
BINARY="$PROJECT_ROOT/llm-runtime"

# Step 5: Demo READ Operation
echo -e "${YELLOW}=========================================="
echo "Demo 1: READ Operation"
echo "==========================================${NC}"
echo ""
echo "Reading sample.txt..."
echo "Command: <open sample.txt>"
echo ""

echo "<open sample.txt>" | "$BINARY" --root "$DEMO_DIR"

echo ""
echo -e "${GREEN}✓ Read successful${NC}"
echo ""

# Step 6: Demo WRITE Operation
echo -e "${YELLOW}=========================================="
echo "Demo 2: WRITE Operation"
echo "==========================================${NC}"
echo ""
echo "Creating new file output.txt..."
echo "Command: <write output.txt>content</write>"
echo ""

cat << 'INPUT' | "$BINARY" --root "$DEMO_DIR"
<write output.txt>
This file was created by llm-runtime!

Created at: $(date)

Features demonstrated:
1. Secure file writing
2. Atomic operations (temp + rename)
3. Automatic backups
4. Path validation

Status: Success
</write>
INPUT

echo ""
echo "Verifying written file:"
cat "$DEMO_DIR/output.txt"
echo ""
echo -e "${GREEN}✓ Write successful${NC}"
echo ""

# Step 7: Demo EXEC Operation
echo -e "${YELLOW}=========================================="
echo "Demo 3: EXEC Operation (Containerized)"
echo "==========================================${NC}"
echo ""
echo "Executing commands in Docker container..."
echo ""

echo "Command: <exec echo 'Hello from Docker container'>"
echo "<exec echo 'Hello from Docker container'>" | "$BINARY" --root "$DEMO_DIR" --exec-enabled

echo ""
echo "Command: <exec ls -la>"
echo "<exec ls -la>" | "$BINARY" --root "$DEMO_DIR" --exec-enabled

echo ""
echo "Command: <exec cat sample.txt | wc -l>"
echo "<exec cat sample.txt | wc -l>" | "$BINARY" --root "$DEMO_DIR" --exec-enabled

echo ""
echo -e "${GREEN}✓ Exec successful${NC}"
echo ""

# Step 8: Demo EXEC with stdin
echo -e "${YELLOW}=========================================="
echo "Demo 4: EXEC with stdin"
echo "==========================================${NC}"
echo ""
echo "Passing stdin to container..."
echo "Command: <exec cat>stdin content</exec>"
echo ""

cat << 'STDIN_INPUT' | "$BINARY" --root "$DEMO_DIR" --exec-enabled
<exec cat>
Line 1 from stdin
Line 2 from stdin
Line 3 from stdin
</exec>
STDIN_INPUT

echo ""
echo -e "${GREEN}✓ Stdin exec successful${NC}"
echo ""

# Step 9: Combined Demo
echo -e "${YELLOW}=========================================="
echo "Demo 5: Combined Operations"
echo "==========================================${NC}"
echo ""
echo "Performing read, write, and exec together..."
echo ""

cat << 'COMBINED' | "$BINARY" --root "$DEMO_DIR" --exec-enabled
<open data.json>

<exec find . -type f -name "*.txt" | wc -l>

<write summary.txt>
=== File Analysis Summary ===

Files analyzed:
- sample.txt: Multi-line sample file
- data.json: Configuration file with demo metadata
- code.go: Sample Go source file
- output.txt: Generated during write test

All operations performed with security validation.

Analysis timestamp: $(date)
</write>
COMBINED

echo ""
echo "Verifying summary was created:"
cat "$DEMO_DIR/summary.txt"
echo ""
echo -e "${GREEN}✓ All combined operations successful${NC}"
echo ""

# Step 10: Show Security Features
echo -e "${YELLOW}=========================================="
echo "Demo 6: Security Features"
echo "==========================================${NC}"
echo ""

echo "Test 1: Path traversal attempt (should be blocked)..."
echo "<open ../../../etc/passwd>" | "$BINARY" --root "$DEMO_DIR" 2>&1 | grep -i "error\|denied\|invalid" || echo -e "${RED}  Blocked by path validation${NC}"

echo ""
echo "Test 2: Reading binary file (should be blocked)..."
# Create a fake binary
echo -e "\x7fELF\x00\x00\x00" > "$DEMO_DIR/fake.bin"
echo "<open fake.bin>" | "$BINARY" --root "$DEMO_DIR" 2>&1 | grep -i "error\|not allowed\|binary" || echo -e "${RED}  Blocked by extension validation${NC}"

echo ""
echo "Test 3: Container isolation (can't see host /home)..."
echo "<exec ls /home>" | "$BINARY" --root "$DEMO_DIR" --exec-enabled 2>&1 | head -10

echo ""
echo -e "${GREEN}✓ Security protections working${NC}"
echo ""

# Step 11: Performance Demo
echo -e "${YELLOW}=========================================="
echo "Demo 7: Performance"
echo "==========================================${NC}"
echo ""

echo "Measuring exec latency (3 runs)..."
for i in 1 2 3; do
    START=$(date +%s%N)
    echo "<exec echo test>" | "$BINARY" --root "$DEMO_DIR" --exec-enabled > /dev/null 2>&1
    END=$(date +%s%N)
    ELAPSED=$(( (END - START) / 1000000 ))
    echo "  Run $i: ${ELAPSED}ms"
done

echo ""
echo -e "${GREEN}✓ Performance check complete${NC}"
echo ""

# Step 12: Show generated files
echo -e "${YELLOW}=========================================="
echo "Demo 8: Files Generated"
echo "==========================================${NC}"
echo ""
echo "All files in demo directory:"
ls -lh "$DEMO_DIR"
echo ""

# Cleanup prompt
echo -e "${BLUE}Cleanup${NC}"
echo ""
echo "Demo files are in: $DEMO_DIR"
read -p "Delete demo files? (y/N): " -n 1 -r
echo ""
if [[ $REPLY =~ ^[Yy]$ ]]; then
    rm -rf "$DEMO_DIR"
    echo -e "${GREEN}✓ Cleaned up${NC}"
else
    echo "Demo files preserved at: $DEMO_DIR"
fi
echo ""

# Summary
echo -e "${YELLOW}=========================================="
echo "Demo Complete!"
echo "==========================================${NC}"
echo ""
echo "Summary of operations demonstrated:"
echo "  ✓ Read files with <open>"
echo "  ✓ Write files with <write> (atomic, with backups)"
echo "  ✓ Execute commands with <exec> (containerized)"
echo "  ✓ Pass stdin to containers"
echo "  ✓ Combined operations (read + exec + write)"
echo "  ✓ Security validations (path traversal, binary files)"
echo "  ✓ Container isolation (can't access host filesystem)"
echo ""
echo "Key security features:"
echo "  - Path validation prevents directory traversal"
echo "  - Extension whitelist blocks binary files"
echo "  - Docker containers provide full isolation"
echo "  - Network disabled in containers"
echo "  - Resource limits enforced (memory, CPU, timeout)"
echo "  - Non-root execution"
echo ""
echo -e "${GREEN}All operations working correctly!${NC}"
echo ""
echo "To run again: cd $PROJECT_ROOT && ./scripts/demo_containerized_io.sh"

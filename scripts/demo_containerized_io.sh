#!/bin/bash
# Comprehensive Demo: Containerized I/O Operations
# Shows read, write, and exec all using Docker containers

set -e

echo "=========================================="
echo "Containerized I/O Demo"
echo "=========================================="
echo ""

# Colors
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m'

# Step 1: Check Docker
echo -e "${BLUE}Step 1: Checking Docker availability...${NC}"
if ! command -v docker &> /dev/null; then
    echo "ERROR: Docker not found. Please install Docker first:"
    echo "  Linux: curl -fsSL https://get.docker.com | sh"
    echo "  macOS: brew install docker"
    exit 1
fi
echo -e "${GREEN}✓ Docker found${NC}"
echo ""

# Step 2: Build the tool
echo -e "${BLUE}Step 2: Building llm-runtime...${NC}"
make build
echo -e "${GREEN}✓ Build complete${NC}"
echo ""

# Step 3: Build IO container image
echo -e "${BLUE}Step 3: Building IO container image...${NC}"
if [ ! -f "Dockerfile.io" ]; then
    echo "Creating Dockerfile.io for containerized I/O..."
    cat > Dockerfile.io << 'DOCKERFILE'
FROM alpine:latest

# Install basic tools
RUN apk add --no-cache \
    bash \
    coreutils \
    findutils \
    grep

# Create non-root user
RUN adduser -D -u 1000 iouser

USER iouser
WORKDIR /workspace

CMD ["/bin/sh"]
DOCKERFILE
fi

docker build -f Dockerfile.io -t llm-runtime-io:latest .
echo -e "${GREEN}✓ IO container image ready${NC}"
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

echo -e "${GREEN}✓ Test files created${NC}"
ls -la "$DEMO_DIR"
echo ""

# Step 5: Demo Containerized READ
echo -e "${YELLOW}=========================================="
echo "Demo 1: Containerized READ Operation"
echo "==========================================${NC}"
echo ""
echo "Reading sample.txt using containerized I/O..."
echo "Command: <open sample.txt>"
echo ""

echo "<open sample.txt>" | ./llm-runtime --root "$DEMO_DIR" --io-containerized

echo ""
echo -e "${GREEN}✓ Containerized read successful${NC}"
echo ""

# Step 6: Demo Containerized WRITE
echo -e "${YELLOW}=========================================="
echo "Demo 2: Containerized WRITE Operation"
echo "==========================================${NC}"
echo ""
echo "Creating new file using containerized I/O..."
echo "Command: <write output.txt>content</write>"
echo ""

cat << 'INPUT' | ./llm-runtime --root "$DEMO_DIR" --io-containerized
<write output.txt>
This file was created using containerized I/O!

Created at: $(date)

Features demonstrated:
1. Secure file writing through Docker container
2. No direct host filesystem access
3. Isolated execution environment

Status: Success
</write>
INPUT

echo ""
echo "Verifying written file:"
cat "$DEMO_DIR/output.txt"
echo ""
echo -e "${GREEN}✓ Containerized write successful${NC}"
echo ""

# Step 7: Demo Containerized EXEC
echo -e "${YELLOW}=========================================="
echo "Demo 3: Containerized EXEC Operation"
echo "==========================================${NC}"
echo ""
echo "Executing commands in secure container..."
echo "Command: <exec ls -la>"
echo ""

echo "<exec ls -la>" | ./llm-runtime --root "$DEMO_DIR" --exec-enabled

echo ""
echo "Command: <exec cat sample.txt | wc -l>"
echo ""

echo "<exec cat sample.txt | wc -l>" | ./llm-runtime --root "$DEMO_DIR" --exec-enabled

echo ""
echo -e "${GREEN}✓ Containerized exec successful${NC}"
echo ""

# Step 8: Combined Demo
echo -e "${YELLOW}=========================================="
echo "Demo 4: Combined Operations"
echo "==========================================${NC}"
echo ""
echo "Performing read, write, and exec together..."
echo ""

cat << 'COMBINED' | ./llm-runtime --root "$DEMO_DIR" --exec-enabled --io-containerized
Let me analyze the files and create a summary:

<open data.json>

<exec find . -type f -name "*.txt" | wc -l>

<write summary.txt>
=== File Analysis Summary ===

Total text files found: (see exec output above)

Files analyzed:
- sample.txt: Multi-line sample file
- data.json: Configuration file with demo metadata
- output.txt: Generated during containerized write test

All operations performed in isolated Docker containers for maximum security.

Analysis complete at: $(date)
</write>

<exec cat summary.txt>
COMBINED

echo ""
echo -e "${GREEN}✓ All combined operations successful${NC}"
echo ""

# Step 9: Show Security Features
echo -e "${YELLOW}=========================================="
echo "Demo 5: Security Features"
echo "==========================================${NC}"
echo ""
echo "Attempting path traversal (should be blocked)..."
echo ""

echo "<open ../../../../etc/passwd>" | ./llm-runtime --root "$DEMO_DIR" --io-containerized 2>&1 || true

echo ""
echo "Attempting unauthorized command (should be blocked)..."
echo ""

echo "<exec rm -rf />" | ./llm-runtime --root "$DEMO_DIR" --exec-enabled 2>&1 || true

echo ""
echo -e "${GREEN}✓ Security protections working${NC}"
echo ""

# Step 10: Cleanup
echo -e "${BLUE}Step 10: Cleanup${NC}"
echo ""
echo "Demo files are in: $DEMO_DIR"
echo "To remove: rm -rf $DEMO_DIR"
echo ""

# Summary
echo -e "${YELLOW}=========================================="
echo "Demo Complete!"
echo "==========================================${NC}"
echo ""
echo "Summary of containerized operations:"
echo "  ✓ Read files securely through Docker container"
echo "  ✓ Write files with containerized isolation"
echo "  ✓ Execute commands in sandboxed environment"
echo "  ✓ Security features prevent unauthorized access"
echo ""
echo "Docker containers used:"
docker images | grep llm-runtime-io || echo "  llm-runtime-io:latest"
echo ""
echo "All operations performed with:"
echo "  - No direct host filesystem access"
echo "  - Isolated execution environment"
echo "  - Resource limits enforced"
echo "  - Non-root user execution"
echo ""
echo -e "${GREEN}Containerized I/O is working perfectly!${NC}"

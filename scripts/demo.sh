#!/bin/bash

# Change to project root directory
cd "$(dirname "$0")/.."

# Example script demonstrating the LLM File Access Tool

echo "=== LLM File Access Tool - Demo ==="
echo

# Create a temporary demo repository
DEMO_DIR=$(mktemp -d)
echo "Creating demo repository at: $DEMO_DIR"
echo "Note: By default, llm-runtime uses /tmp/dynamic-repo/. Using --root to specify this demo repo."

# Create some sample files
mkdir -p "$DEMO_DIR/src"
mkdir -p "$DEMO_DIR/docs"
mkdir -p "$DEMO_DIR/.git"  # This will be excluded

cat > "$DEMO_DIR/README.md" << 'EOF'
# Demo Project

This is a demonstration repository for the LLM File Access Tool.

## Structure
- src/ - Source code
- docs/ - Documentation
EOF

cat > "$DEMO_DIR/src/main.go" << 'EOF'
package main

import "fmt"

func main() {
    fmt.Println("Hello from the demo project!")
}
EOF

cat > "$DEMO_DIR/docs/architecture.md" << 'EOF'
# Architecture

This demo project uses a simple structure:
1. Main entry point in src/main.go
2. Documentation in docs/
3. Configuration in the root directory
EOF

cat > "$DEMO_DIR/.env" << 'EOF'
SECRET_KEY=this-should-not-be-accessible
EOF

cat > "$DEMO_DIR/private.key" << 'EOF'
-----BEGIN PRIVATE KEY-----
This should also be blocked
-----END PRIVATE KEY-----
EOF

echo
echo "Demo repository created with sample files."
echo "Building the tool..."

# Build the tool if not already built
if [ ! -f "./llm-runtime" ]; then
    go build -o llm-runtime main.go
fi

echo
echo "=== Demo 1: Simple file reading ==="
echo "Input: 'Let me read <open README.md>'"
echo "---"

echo "Let me read <open README.md>" | ./llm-runtime --root "$DEMO_DIR"

echo
echo "=== Demo 2: Multiple commands ==="
echo "Input: Multiple open commands"
echo "---"

cat << 'EOF' | ./llm-runtime --root "$DEMO_DIR"
I'll explore this repository step by step.

First, let me check the README:
<open README.md>

Now let's look at the source code:
<open src/main.go>

And finally, the architecture documentation:
<open docs/architecture.md>

Based on these files, this appears to be a simple Go demonstration project.
EOF

echo
echo "=== Demo 3: Security - Attempting to read excluded files ==="
echo "Input: Trying to open .env and .git/config"
echo "---"

cat << 'EOF' | ./llm-runtime --root "$DEMO_DIR"
Let me try to access sensitive files:
<open .env>
<open .git/config>
<open private.key>
EOF

echo
echo "=== Demo 4: Security - Path traversal attempt ==="
echo "Input: Trying to escape the repository"
echo "---"

cat << 'EOF' | ./llm-runtime --root "$DEMO_DIR"
Attempting path traversal:
<open ../../etc/passwd>
<open ../../../etc/hosts>
EOF

echo
echo "=== Demo 5: Non-existent file ==="
echo "Input: Trying to open a file that doesn't exist"
echo "---"

echo "Looking for config: <open config.yaml>" | ./llm-runtime --root "$DEMO_DIR"

echo
echo "=== Demo 6: Interactive mode (skipping in script) ==="
echo "To try interactive mode, run:"
echo "  ./llm-runtime --root \"$DEMO_DIR\" --interactive"
echo "Then type commands and see real-time results."

echo
echo "=== Checking audit log ==="
if [ -f "audit.log" ]; then
    echo "Recent audit log entries:"
    tail -5 audit.log
else
    echo "No audit log found (run the tool first)"
fi

echo
echo "=== Cleanup ==="
echo "Demo directory: $DEMO_DIR"
echo "To clean up, run: rm -rf $DEMO_DIR"
echo
echo "Demo complete!"

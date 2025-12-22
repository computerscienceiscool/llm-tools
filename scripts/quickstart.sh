#!/bin/bash

# Change to project root directory
cd "$(dirname "$0")/.."

# Quick Start Script for LLM File Access Tool
# Run this to get started immediately!

echo "LLM File Access Tool - Quick Start"
echo "====================================="
echo

# Check if tool is built
if [ ! -f "llm-runtime" ]; then
    echo "Tool not built yet. Running setup..."
    echo
    bash setup.sh
else
    echo "Tool already built!"
fi

echo
echo "Quick Command Reference:"
echo "---------------------------"
echo
echo "Test with a simple command:"
echo "   echo 'Read the README <open README.md>' | ./llm-runtime"
echo
echo "Explore a specific directory (default is /tmp/dynamic-repo/):"
echo "   ./llm-runtime --root /path/to/your/project"
echo
echo "Run the full demo:"
echo "   ./demo.sh"
echo
echo "See a real example:"
echo "   ./example_usage.sh"
echo
echo "Test security features:"
echo "   ./security_test.sh"
echo
echo "For more information, see README.md"
echo "To customize, edit llm-runtime.config.yaml"
echo "For LLM integration, see SYSTEM_PROMPT.md"
echo
echo "Ready to go! Try command #1 above to start."

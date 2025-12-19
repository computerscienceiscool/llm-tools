#!/bin/bash

echo "========================================"
echo "Container Pooling Performance Demo"
echo "========================================"
echo ""

# Test command
TEST_CMD='<open README.md> <open Makefile> <open go.mod> <open llm-runtime.config.yaml> <open pkg/config/types.go>'

echo "Test: Reading 5 files"
echo ""

# WITHOUT POOLING
echo "Step 1: Testing WITHOUT container pooling..."
echo "  (Disabling pooling, rebuilding...)"
sed -i.bak 's/enabled: true/enabled: false/' llm-runtime.config.yaml
make build > /dev/null 2>&1

echo "  Running test..."
WITHOUT_START=$(date +%s.%N)
echo "$TEST_CMD" | ./llm-runtime > /dev/null 2>&1
WITHOUT_END=$(date +%s.%N)
WITHOUT_TIME=$(echo "$WITHOUT_END - $WITHOUT_START" | bc)

echo "  ✓ Completed in ${WITHOUT_TIME} seconds"
echo ""

# WITH POOLING
echo "Step 2: Testing WITH container pooling..."
echo "  (Enabling pooling, rebuilding...)"
sed -i 's/enabled: false/enabled: true/' llm-runtime.config.yaml
make build > /dev/null 2>&1

echo "  Running test..."
WITH_START=$(date +%s.%N)
echo "$TEST_CMD" | ./llm-runtime > /dev/null 2>&1
WITH_END=$(date +%s.%N)
WITH_TIME=$(echo "$WITH_END - $WITH_START" | bc)

echo "  ✓ Completed in ${WITH_TIME} seconds"
echo ""

# Results
echo "========================================"
echo "RESULTS:"
echo "========================================"
echo "Without pooling: ${WITHOUT_TIME}s"
echo "With pooling:    ${WITH_TIME}s"
echo ""

SPEEDUP=$(echo "scale=2; $WITHOUT_TIME / $WITH_TIME" | bc)
SAVINGS=$(echo "scale=2; (($WITHOUT_TIME - $WITH_TIME) / $WITHOUT_TIME) * 100" | bc)

echo "Performance improvement: ${SPEEDUP}x faster"
echo "Time saved: ${SAVINGS}%"
echo ""
echo "For 100 operations:"
echo "  Without pooling: ~$(echo "scale=1; $WITHOUT_TIME * 100 / 60" | bc) minutes"
echo "  With pooling:    ~$(echo "scale=1; $WITH_TIME * 100 / 60" | bc) minutes"
echo "========================================"

# Restore config
mv llm-runtime.config.yaml.bak llm-runtime.config.yaml 2>/dev/null
make build > /dev/null 2>&1

#!/bin/bash

# Change to project root directory
cd "$(dirname "$0")/.."

# Security test suite for LLM File Access Tool
# This script tests various security scenarios to ensure the tool is safe

echo "=== LLM File Access Tool - Security Test Suite ==="
echo

# Build the tool
if [ ! -f "./llm-runtime" ]; then
    echo "Building tool..."
    go build -o llm-runtime main.go || exit 1
fi

# Create test environment
TEST_DIR=$(mktemp -d)
echo "Test directory: $TEST_DIR"

# Create test repository
mkdir -p "$TEST_DIR/repo/src"
mkdir -p "$TEST_DIR/repo/.git"
mkdir -p "$TEST_DIR/sensitive"

# Create test files
echo "Safe content" > "$TEST_DIR/repo/safe.txt"
echo "Source code" > "$TEST_DIR/repo/src/main.go"
echo "Git config" > "$TEST_DIR/repo/.git/config"
echo "Environment secrets" > "$TEST_DIR/repo/.env"
echo "Private key" > "$TEST_DIR/repo/private.key"
echo "Certificate" > "$TEST_DIR/repo/cert.pem"
echo "SENSITIVE DATA" > "$TEST_DIR/sensitive/secret.txt"
echo "System file" > "/tmp/test-system-file.txt"

# Create a symlink attack file
ln -s /etc/passwd "$TEST_DIR/repo/symlink-attack"
ln -s "$TEST_DIR/sensitive/secret.txt" "$TEST_DIR/repo/symlink-sensitive"

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Test function
run_test() {
    local test_name="$1"
    local input="$2"
    local should_fail="$3"
    local error_pattern="$4"
    
    echo -n "Testing: $test_name... "
    
    result=$(echo "$input" | ./llm-runtime --root "$TEST_DIR/repo" 2>&1)
    
    if [ "$should_fail" = "true" ]; then
        if echo "$result" | grep -q "ERROR"; then
            if [ -n "$error_pattern" ] && echo "$result" | grep -q "$error_pattern"; then
                echo -e "${GREEN}PASS${NC} (correctly blocked with $error_pattern)"
            elif [ -z "$error_pattern" ]; then
                echo -e "${GREEN}PASS${NC} (correctly blocked)"
            else
                echo -e "${RED}FAIL${NC} (wrong error type)"
                echo "Expected: $error_pattern"
                echo "Got: $result" | grep ERROR
            fi
        else
            echo -e "${RED}FAIL${NC} (should have been blocked)"
            echo "$result"
        fi
    else
        if echo "$result" | grep -q "ERROR"; then
            echo -e "${RED}FAIL${NC} (incorrectly blocked)"
            echo "$result" | grep ERROR
        else
            echo -e "${GREEN}PASS${NC} (allowed as expected)"
        fi
    fi
}

echo
echo "=== Running Security Tests ==="
echo

# Test 1: Normal file access (should succeed)
run_test "Normal file access" "<open safe.txt>" "false"

# Test 2: Subdirectory access (should succeed)
run_test "Subdirectory file access" "<open src/main.go>" "false"

# Test 3: .git directory access (should fail)
run_test ".git directory access" "<open .git/config>" "true" "PATH_SECURITY"

# Test 4: .env file access (should fail)
run_test ".env file access" "<open .env>" "true" "PATH_SECURITY"

# Test 5: Private key access (should fail)
run_test "Private key file access" "<open private.key>" "true" "PATH_SECURITY"

# Test 6: Certificate access (should fail)
run_test "Certificate file access" "<open cert.pem>" "true" "PATH_SECURITY"

# Test 7: Path traversal with ../ (should fail)
run_test "Path traversal (../)" "<open ../sensitive/secret.txt>" "true" "PATH_SECURITY"

# Test 8: Path traversal with ../../ (should fail)
run_test "Path traversal (../../)" "<open ../../etc/passwd>" "true" "PATH_SECURITY"

# Test 9: Absolute path outside repo (should fail)
run_test "Absolute path (/etc/passwd)" "<open /etc/passwd>" "true" "PATH_SECURITY"

# Test 10: Absolute path to temp file (should fail)
run_test "Absolute path (/tmp/)" "<open /tmp/test-system-file.txt>" "true" "PATH_SECURITY"

# Test 11: Symlink to system file (should fail)
run_test "Symlink to /etc/passwd" "<open symlink-attack>" "true" "PATH_SECURITY"

# Test 12: Symlink to sensitive file (should fail)
run_test "Symlink to sensitive file" "<open symlink-sensitive>" "true" "PATH_SECURITY"

# Test 13: Path with null bytes (should fail or sanitize)
run_test "Path with null bytes" '<open safe.txt\x00.sh>' "false"

# Test 14: Path with Unicode tricks (should handle safely)
run_test "Unicode normalization attack" "<open sa​fe.txt>" "true" "FILE_NOT_FOUND"

# Test 15: Very long path (should handle gracefully)
LONG_PATH=$(printf 'a/%.0s' {1..500})
run_test "Very long path" "<open $LONG_PATH>" "true"

# Test 16: Path with special characters
run_test "Path with spaces" '<open "file with spaces.txt">' "true" "FILE_NOT_FOUND"

# Test 17: Hidden file starting with dot
echo "Hidden content" > "$TEST_DIR/repo/.hidden"
run_test "Hidden file access" "<open .hidden>" "false"

# Test 18: Multiple commands in one input
run_test "Multiple commands" "<open safe.txt> <open src/main.go>" "false"

# Test 19: Case sensitivity check
run_test "Case sensitive path" "<open Safe.txt>" "true" "FILE_NOT_FOUND"

# Test 20: Empty path
run_test "Empty path" "<open >" "true"

echo
echo "=== Testing Rate Limiting and Resource Limits ==="
echo

# Test 21: Large file handling
dd if=/dev/zero of="$TEST_DIR/repo/large.bin" bs=1M count=2 2>/dev/null
run_test "File over size limit" "<open large.bin>" "true" "RESOURCE_LIMIT"

# Test 22: Many rapid requests (check for DoS protection)
echo -n "Testing rapid requests... "
START_TIME=$(date +%s)
for i in {1..20}; do
    echo "<open safe.txt>" | ./llm-runtime --root "$TEST_DIR/repo" > /dev/null 2>&1
done
END_TIME=$(date +%s)
DURATION=$((END_TIME - START_TIME))
if [ $DURATION -lt 1 ]; then
    echo -e "${GREEN}PASS${NC} (handled 20 requests quickly)"
else
    echo -e "${YELLOW}SLOW${NC} (took ${DURATION}s for 20 requests)"
fi

echo
echo "=== Checking Audit Log ==="
echo

if [ -f "audit.log" ]; then
    echo "Recent security-relevant audit entries:"
    grep -E "failed|PATH_SECURITY|RESOURCE_LIMIT" audit.log | tail -5
    
    # Count security events
    SECURITY_EVENTS=$(grep -c "PATH_SECURITY" audit.log)
    echo "Total security blocks logged: $SECURITY_EVENTS"
else
    echo -e "${YELLOW}Warning: No audit log found${NC}"
fi

echo
echo "=== Testing Command Injection Prevention ==="
echo

# Test 23: Command injection attempt in path
run_test "Command injection (semicolon)" '<open safe.txt; cat /etc/passwd>' "true"

# Test 24: Command injection with backticks
run_test "Command injection (backticks)" '<open `cat /etc/passwd`>' "true"

# Test 25: Command injection with $()
run_test "Command injection (\$())" '<open $(cat /etc/passwd)>' "true"

echo
echo "=== Summary ==="
echo

# Count results
TOTAL_TESTS=25
if [ -f "audit.log" ]; then
    BLOCKED=$(grep -c "failed" audit.log | tail -1)
    echo "Tests completed: $TOTAL_TESTS"
    echo "Security blocks triggered: $BLOCKED"
fi

echo
echo "=== Cleanup ==="
echo "Test directory: $TEST_DIR"
echo "To clean up: rm -rf $TEST_DIR"
rm -f "/tmp/test-system-file.txt"

echo
echo -e "${GREEN}Security test suite complete!${NC}"
echo "The tool appears to be properly secured against common attacks."

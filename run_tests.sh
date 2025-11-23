
#!/bin/bash

# LLM Tools Test Suite Runner
# Comprehensive testing script for the refactored LLM Tools application

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
COVERAGE_THRESHOLD=80
TIMEOUT=300s

echo -e "${BLUE}🧪 LLM Tools - Comprehensive Test Suite${NC}"
echo "========================================"
echo

# Function to print status
print_status() {
    local status=$1
    local message=$2
    if [ "$status" = "PASS" ]; then
        echo -e "${GREEN}✓${NC} $message"
    elif [ "$status" = "FAIL" ]; then
        echo -e "${RED}✗${NC} $message"
    elif [ "$status" = "WARN" ]; then
        echo -e "${YELLOW}⚠${NC} $message"
    else
        echo -e "${BLUE}ℹ${NC} $message"
    fi
}

# Function to run tests with timeout
run_with_timeout() {
    local cmd="$1"
    local description="$2"
    
    print_status "INFO" "Running: $description"
    
    if timeout $TIMEOUT $cmd; then
        print_status "PASS" "$description completed"
        return 0
    else
        local exit_code=$?
        if [ $exit_code -eq 124 ]; then
            print_status "FAIL" "$description timed out after $TIMEOUT"
        else
            print_status "FAIL" "$description failed with exit code $exit_code"
        fi
        return $exit_code
    fi
}

# Check Go installation
check_go() {
    if ! command -v go &> /dev/null; then
        print_status "FAIL" "Go is not installed"
        exit 1
    fi
    
    GO_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
    print_status "PASS" "Go version $GO_VERSION detected"
}

# Check dependencies
check_dependencies() {
    print_status "INFO" "Checking dependencies..."
    
    if ! go mod download; then
        print_status "FAIL" "Failed to download dependencies"
        return 1
    fi
    
    if ! go mod tidy; then
        print_status "FAIL" "Failed to tidy modules"
        return 1
    fi
    
    print_status "PASS" "Dependencies verified"
}

# Run basic test suite
run_basic_tests() {
    echo
    print_status "INFO" "Running basic test suite..."
    
    if run_with_timeout "go test ./..." "Basic test suite"; then
        print_status "PASS" "All basic tests passed"
        return 0
    else
        print_status "FAIL" "Basic tests failed"
        return 1
    fi
}

# Run tests with race detection
run_race_tests() {
    echo
    print_status "INFO" "Running race detection tests..."
    
    if run_with_timeout "go test -race ./..." "Race detection tests"; then
        print_status "PASS" "No race conditions detected"
        return 0
    else
        print_status "FAIL" "Race conditions detected"
        return 1
    fi
}

# Run coverage analysis
run_coverage_tests() {
    echo
    print_status "INFO" "Running coverage analysis..."
    
    if run_with_timeout "go test -coverprofile=coverage.out ./..." "Coverage analysis"; then
        # Parse coverage percentage
        local coverage=$(go tool cover -func=coverage.out | tail -1 | awk '{print $3}' | sed 's/%//')
        
        if [ -n "$coverage" ]; then
            local coverage_int=$(echo "$coverage" | cut -d'.' -f1)
            
            if [ "$coverage_int" -ge "$COVERAGE_THRESHOLD" ]; then
                print_status "PASS" "Coverage: ${coverage}% (threshold: ${COVERAGE_THRESHOLD}%)"
            else
                print_status "WARN" "Coverage: ${coverage}% (below threshold: ${COVERAGE_THRESHOLD}%)"
            fi
            
            # Generate HTML coverage report
            if go tool cover -html=coverage.out -o coverage.html; then
                print_status "INFO" "Coverage report generated: coverage.html"
            fi
        else
            print_status "WARN" "Could not parse coverage percentage"
        fi
        
        return 0
    else
        print_status "FAIL" "Coverage analysis failed"
        return 1
    fi
}

# Run security tests
run_security_tests() {
    echo
    print_status "INFO" "Running security tests..."
    
    local security_packages=(
        "./internal/security/..."
        "./internal/handlers/..."
        "./internal/core/..."
    )
    
    for package in "${security_packages[@]}"; do
        if ! run_with_timeout "go test -v $package" "Security tests: $package"; then
            print_status "FAIL" "Security tests failed for $package"
            return 1
        fi
    done
    
    print_status "PASS" "All security tests passed"
    return 0
}

# Run benchmarks
run_benchmarks() {
    echo
    print_status "INFO" "Running performance benchmarks..."
    
    if run_with_timeout "go test -bench=. -benchmem ./..." "Performance benchmarks"; then
        print_status "PASS" "Benchmarks completed"
        return 0
    else
        print_status "FAIL" "Benchmarks failed"
        return 1
    fi
}

# Run specific package tests
run_package_tests() {
    echo
    print_status "INFO" "Running package-specific tests..."
    
    local packages=(
        "./cmd/..."
        "./internal/config/..."
        "./internal/core/..."
        "./internal/parser/..."
        "./internal/security/..."
        "./internal/errors/..."
    )
    
    for package in "${packages[@]}"; do
        if ! run_with_timeout "go test -v $package" "Package tests: $package"; then
            print_status "FAIL" "Package tests failed for $package"
            return 1
        fi
    done
    
    print_status "PASS" "All package tests passed"
    return 0
}

# Validate test files
validate_test_files() {
    echo
    print_status "INFO" "Validating test file structure..."
    
    local required_files=(
        "main_test.go"
        "cmd/cli_test.go"
        "cmd/flags_test.go"
        "internal/config/config_test.go"
        "internal/config/loader_test.go"
        "internal/core/commands_test.go"
        "internal/core/executor_test.go"
        "internal/core/session_test.go"
        "internal/parser/parser_test.go"
        "internal/parser/llm-parser_test.go"
        "internal/security/validator_test.go"
        "internal/security/path-validator_test.go"
        "internal/security/audit-logger_test.go"
        "internal/handlers/interfaces_test.go"
        "internal/errors/errors_test.go"
    )
    
    local missing_files=0
    
    for file in "${required_files[@]}"; do
        if [ ! -f "$file" ]; then
            print_status "FAIL" "Missing test file: $file"
            missing_files=$((missing_files + 1))
        fi
    done
    
    if [ $missing_files -eq 0 ]; then
        print_status "PASS" "All required test files present"
        return 0
    else
        print_status "FAIL" "$missing_files test files missing"
        return 1
    fi
}

# Generate test report
generate_report() {
    echo
    print_status "INFO" "Generating test report..."
    
    cat > test_report.md << EOF
# Test Suite Report

**Generated:** $(date)
**Go Version:** $(go version | awk '{print $3}')

## Test Results

EOF

    if [ -f "coverage.out" ]; then
        local coverage=$(go tool cover -func=coverage.out | tail -1 | awk '{print $3}')
        echo "**Coverage:** $coverage" >> test_report.md
        echo "" >> test_report.md
    fi

    echo "## Package Coverage" >> test_report.md
    echo "" >> test_report.md
    echo "\`\`\`" >> test_report.md
    if [ -f "coverage.out" ]; then
        go tool cover -func=coverage.out >> test_report.md
    fi
    echo "\`\`\`" >> test_report.md

    print_status "PASS" "Test report generated: test_report.md"
}

# Clean up temporary files
cleanup() {
    print_status "INFO" "Cleaning up temporary files..."
    
    local temp_files=(
        "coverage.out"
        "cpu.prof"
        "mem.prof"
        "*.test"
    )
    
    for pattern in "${temp_files[@]}"; do
        rm -f $pattern 2>/dev/null || true
    done
    
    print_status "INFO" "Cleanup completed"
}

# Main execution
main() {
    local start_time=$(date +%s)
    local exit_code=0
    
    # Pre-flight checks
    check_go || exit 1
    check_dependencies || exit 1
    validate_test_files || exit 1
    
    # Run test suites
    run_basic_tests || exit_code=1
    run_race_tests || exit_code=1
    run_coverage_tests || exit_code=1
    run_security_tests || exit_code=1
    run_package_tests || exit_code=1
    
    # Optional: run benchmarks (can be slow)
    if [ "$1" = "--with-benchmarks" ]; then
        run_benchmarks || exit_code=1
    fi
    
    # Generate report
    generate_report
    
    # Calculate execution time
    local end_time=$(date +%s)
    local duration=$((end_time - start_time))
    
    echo
    echo "========================================"
    if [ $exit_code -eq 0 ]; then
        print_status "PASS" "All tests completed successfully in ${duration}s"
    else
        print_status "FAIL" "Some tests failed (duration: ${duration}s)"
    fi
    
    # Clean up
    if [ "$1" != "--keep-files" ]; then
        cleanup
    fi
    
    exit $exit_code
}

# Handle script arguments
case "$1" in
    --help|-h)
        echo "Usage: $0 [OPTIONS]"
        echo ""
        echo "Options:"
        echo "  --with-benchmarks    Include performance benchmarks"
        echo "  --keep-files        Keep temporary files after completion"
        echo "  --help, -h          Show this help message"
        echo ""
        echo "Environment Variables:"
        echo "  COVERAGE_THRESHOLD  Minimum coverage percentage (default: 80)"
        echo "  TIMEOUT            Test timeout duration (default: 300s)"
        exit 0
        ;;
    *)
        main "$1"
        ;;
esac

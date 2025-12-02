#!/bin/bash

# Change to project root directory
cd "$(dirname "$0")/.."

# Demo script for LLM File Access Tool exec command functionality

echo "=== LLM File Access Tool - Exec Command Demo ==="
echo

# Create a temporary demo repository
DEMO_DIR=$(mktemp -d)
echo "Creating demo repository at: $DEMO_DIR"

# Create a comprehensive test project structure
mkdir -p "$DEMO_DIR/src"
mkdir -p "$DEMO_DIR/tests"
mkdir -p "$DEMO_DIR/docs"

# Create a Go project
cat > "$DEMO_DIR/go.mod" << 'EOF'
module demo-project

go 1.21

require (
    github.com/stretchr/testify v1.8.4
)
EOF

cat > "$DEMO_DIR/src/calculator.go" << 'EOF'
package main

import "fmt"

// Calculator provides basic arithmetic operations
type Calculator struct{}

// Add returns the sum of two integers
func (c Calculator) Add(a, b int) int {
    return a + b
}

// Subtract returns the difference of two integers
func (c Calculator) Subtract(a, b int) int {
    return a - b
}

// Multiply returns the product of two integers
func (c Calculator) Multiply(a, b int) int {
    return a * b
}

// Divide returns the quotient of two integers
func (c Calculator) Divide(a, b int) (int, error) {
    if b == 0 {
        return 0, fmt.Errorf("division by zero")
    }
    return a / b, nil
}

func main() {
    calc := Calculator{}
    
    fmt.Printf("5 + 3 = %d\n", calc.Add(5, 3))
    fmt.Printf("5 - 3 = %d\n", calc.Subtract(5, 3))
    fmt.Printf("5 * 3 = %d\n", calc.Multiply(5, 3))
    
    result, err := calc.Divide(10, 2)
    if err != nil {
        fmt.Printf("Error: %v\n", err)
    } else {
        fmt.Printf("10 / 2 = %d\n", result)
    }
}
EOF

cat > "$DEMO_DIR/src/calculator_test.go" << 'EOF'
package main

import (
    "testing"
)

func TestCalculatorAdd(t *testing.T) {
    calc := Calculator{}
    
    tests := []struct {
        name     string
        a, b     int
        expected int
    }{
        {"positive numbers", 2, 3, 5},
        {"negative numbers", -2, -3, -5},
        {"mixed numbers", -2, 3, 1},
        {"zero", 0, 5, 5},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := calc.Add(tt.a, tt.b)
            if result != tt.expected {
                t.Errorf("Add(%d, %d) = %d; want %d", tt.a, tt.b, result, tt.expected)
            }
        })
    }
}

func TestCalculatorDivide(t *testing.T) {
    calc := Calculator{}
    
    t.Run("normal division", func(t *testing.T) {
        result, err := calc.Divide(10, 2)
        if err != nil {
            t.Errorf("Unexpected error: %v", err)
        }
        if result != 5 {
            t.Errorf("Divide(10, 2) = %d; want 5", result)
        }
    })
    
    t.Run("division by zero", func(t *testing.T) {
        _, err := calc.Divide(10, 0)
        if err == nil {
            t.Error("Expected error for division by zero")
        }
    })
}

func BenchmarkCalculatorAdd(b *testing.B) {
    calc := Calculator{}
    for i := 0; i < b.N; i++ {
        calc.Add(123, 456)
    }
}
EOF

# Create a simple Makefile
cat > "$DEMO_DIR/Makefile" << 'EOF'
.PHONY: build test clean run

build:
	go build -o bin/calculator src/calculator.go

test:
	go test ./src -v

benchmark:
	go test ./src -bench=. -benchmem

clean:
	rm -rf bin/

run: build
	./bin/calculator

install-deps:
	go mod download
	go mod tidy

lint:
	go vet ./src

fmt:
	go fmt ./src

all: fmt lint test build
EOF

# Create README
cat > "$DEMO_DIR/README.md" << 'EOF'
# Demo Project for Exec Commands

This is a comprehensive demonstration project for testing the LLM File Access Tool's exec command functionality.

## Project Structure

- `src/` - Source code (Go calculator example)
- `tests/` - Test files
- `docs/` - Documentation
- `Makefile` - Build automation
- `go.mod` - Go module definition

## Available Commands

The project supports various commands that can be executed via the `<exec>` command:

### Go Commands
- `go test ./src` - Run tests
- `go build -o bin/calculator src/calculator.go` - Build the application
- `go run src/calculator.go` - Run directly
- `go mod tidy` - Clean up dependencies

### Make Commands
- `make build` - Build the project
- `make test` - Run tests
- `make clean` - Clean build artifacts
- `make all` - Run full build pipeline

### System Commands
- `ls -la` - List files
- `find . -name "*.go"` - Find Go files
- `wc -l src/*.go` - Count lines of code
- `cat src/calculator.go` - Display file contents
EOF

echo
echo "Demo repository created with comprehensive test project."
echo "Building the tool..."

# Build the tool if not already built
if [ ! -f "./llm-runtime" ]; then
    go build -o llm-runtime main.go
fi

echo
echo "=== Demo 1: Project Analysis with Exec Commands ==="
echo "Demonstrating how an LLM would analyze and test a project"
echo "---"

cat << 'EOF' | ./llm-runtime --root "$DEMO_DIR" --exec-enabled --verbose
I'll analyze this demo project comprehensively using all available commands.

First, let me understand the project structure:
<open README.md>

Now let's see what files we have:
<exec ls -la>

Let me check the Go module configuration:
<open go.mod>

Now I'll examine the main source code:
<open src/calculator.go>

Let me look at the test file:
<open src/calculator_test.go>

Let's see how many lines of Go code we have:
<exec find . -name "*.go" -exec wc -l {} \;>

Now let me run the tests to see if everything works:
<exec go test ./src -v>

Let me build the project:
<exec make clean>
<exec make build>

Let's verify the binary was created and run it:
<exec ls -la bin/>
<exec ./bin/calculator>

Finally, let's get some project statistics:
<exec find . -type f -name "*.go" | wc -l>

Based on my analysis, this is a well-structured Go project with comprehensive tests and working functionality.
EOF

echo
echo "=== Demo 2: Security Testing - Blocked Commands ==="
echo "Testing command validation and security restrictions"
echo "---"

cat << 'EOF' | ./llm-runtime --root "$DEMO_DIR" --exec-enabled
Let me try some commands that should be blocked for security:

Attempting to run a non-whitelisted dangerous command:
<exec rm -rf />

Trying to access system files:
<exec cat /etc/passwd>

These should fail due to security restrictions and Docker isolation.
EOF

echo
echo "=== Demo 3: Development Workflow ==="
echo "Demonstrating complete development workflow"
echo "---"

cat << 'EOF' | ./llm-runtime --root "$DEMO_DIR" --exec-enabled
I'll demonstrate a complete development workflow combining all command types.

First, let me create a new feature file:
<write src/advanced_calculator.go>
package main

import (
    "fmt"
    "math"
)

// AdvancedCalculator extends basic calculator
type AdvancedCalculator struct {
    Calculator
}

// Power calculates a raised to the power of b
func (ac AdvancedCalculator) Power(a, b float64) float64 {
    return math.Pow(a, b)
}

// SquareRoot calculates the square root
func (ac AdvancedCalculator) SquareRoot(a float64) float64 {
    return math.Sqrt(a)
}
</write>

Now let me create a test for the new functionality:
<write src/advanced_calculator_test.go>
package main

import (
    "testing"
)

func TestAdvancedCalculatorPower(t *testing.T) {
    calc := AdvancedCalculator{}
    
    result := calc.Power(2, 3)
    expected := 8.0
    if result != expected {
        t.Errorf("Power(2, 3) = %f; want %f", result, expected)
    }
}

func TestAdvancedCalculatorSquareRoot(t *testing.T) {
    calc := AdvancedCalculator{}
    
    result := calc.SquareRoot(9)
    expected := 3.0
    if result != expected {
        t.Errorf("SquareRoot(9) = %f; want %f", result, expected)
    }
}
</write>

Now let's test our new code:
<exec go test ./src -v>

Let's check if everything builds:
<exec go build -o bin/advanced src/advanced_calculator.go src/calculator.go>

Perfect! I've successfully created new functionality and verified it works through testing.
EOF

echo
echo "=== Demo 4: Error Handling ==="
echo "Testing error conditions and debugging workflows"
echo "---"

cat << 'EOF' | ./llm-runtime --root "$DEMO_DIR" --exec-enabled
Let me demonstrate error handling capabilities.

Let's try to build a non-existent file (should fail gracefully):
<exec go build nonexistent.go>

Try to run tests on a non-existent package:
<exec go test ./nonexistent>

Let's create a file with a syntax error:
<write src/broken.go>
package main

import "fmt"

func main() {
    fmt.Println("Missing closing quote and parenthesis
}
</write>

Now try to build the broken file:
<exec go build src/broken.go>

The tool properly reports compilation errors and handles command failures gracefully.
EOF

echo
echo "=== Demo Results Summary ==="
echo

echo "Files created in demo directory:"
find "$DEMO_DIR" -type f | sort

echo
echo "Checking audit log:"
if [ -f "audit.log" ]; then
    echo "Recent exec command entries:"
    grep "exec|" audit.log | tail -5
    
    echo
    echo "Command execution statistics:"
    echo "Total exec commands: $(grep -c "exec|" audit.log 2>/dev/null || echo "0")"
    echo "Successful exec commands: $(grep -c "exec|.*|success" audit.log 2>/dev/null || echo "0")"
    echo "Failed exec commands: $(grep -c "exec|.*|failed" audit.log 2>/dev/null || echo "0")"
else
    echo "No audit log found"
fi

echo
echo "=== Cleanup ==="
echo "Demo directory: $DEMO_DIR"
echo "To clean up: rm -rf $DEMO_DIR"

echo
echo "=== Exec Command Demo Complete! ==="
echo
echo "Key features demonstrated:"
echo "✓ Project analysis with file reading + command execution"
echo "✓ Build automation (go build, make)"
echo "✓ Test execution and validation"
echo "✓ Security validation (blocked dangerous commands)" 
echo "✓ Complete development workflow (read + write + exec)"
echo "✓ Error handling and debugging"
echo "✓ Mixed command workflows"
echo "✓ Comprehensive audit logging"
echo
echo "The exec command enables LLMs to:"
echo "- Validate code by running tests"
echo "- Build and verify projects"
echo "- Execute development workflows"
echo "- Debug issues with real command output"
echo "- All while maintaining security through Docker isolation"


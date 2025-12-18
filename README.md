# LLM File Access Tool

A secure tool that enables Large Language Models to autonomously explore and work with local repositories through embedded commands in their responses. Now includes secure command execution capabilities!

## Features

- **Secure Path Validation**: Prevents directory traversal and access outside repository boundaries
- **Command Parsing**: Extracts and executes `<open filename>`, `<write filename>content</write>`, `<exec command>`, and `<search query>` commands from LLM output
- **Docker-based Execution**: Secure, isolated command execution in Docker containers with no network access
- **Containerized I/O**: All file read/write operations execute in isolated containers for enhanced security
- **Semantic Search**: AI-powered code search using Ollama's local embedding models
- **Audit Logging**: Tracks all operations with timestamps and results
- **Multiple Modes**: Supports pipe, interactive, and file-based operation
- **Configurable Security**: Exclude sensitive paths, whitelist commands, set resource limits

## Container Architecture

llm-runtime uses TWO separate container types for security isolation:

### 1. I/O Container (`llm-runtime-io:latest`)
- **Purpose**: Handles `<open>` and `<write>` commands
- **Built from**: `Dockerfile.io` (Alpine + coreutils only)
- **Features**: Container pooling for performance (5-10x faster)
- **Security**: Minimal attack surface, isolated file operations

### 2. Exec Container (`python-go`)
- **Purpose**: Handles `<exec>` commands  
- **Image**: Prebuilt `python-go` image (has Python and Go)
- **How to get**: `docker pull python-go`
- **Included**: Python, Go, and common development tools

## Installation

### Prerequisites
- Go 1.21 or later
- **Docker** (required - all I/O and exec operations are containerized)
- Ollama with nomic-embed-text model (for search feature - optional)

### Quick Setup

```bash
git clone https://github.com/computerscienceiscool/llm-runtime.git
cd llm-runtime

# STEP 1: Build the I/O container (REQUIRED for file operations)
make build-io-image

# STEP 2: Pull the exec container (has Python and Go)
docker pull python-go

# STEP 3: Build the binary
make build
```

### Verify Installation

```bash
# Check that I/O container exists
make check-io-image

# Check that exec container exists
docker image inspect python-go >/dev/null 2>&1 && echo "python-go image found" || echo "Run: docker pull python-go"

# Check Docker availability
make check-docker
```

## Project Structure
```
llm-runtime/
├── cmd/llm-runtime/       # Entry point
├── pkg/                   # Public API (importable)
│   ├── app/               # Application bootstrap
│   ├── cli/               # Command-line handling
│   ├── config/            # Configuration loading
│   ├── evaluator/         # Command execution
│   ├── sandbox/           # Security, Docker isolation
│   ├── scanner/           # Command parsing
│   ├── search/            # Semantic search (Ollama)
│   └── session/           # Session management
├── internal/              # Internal packages
│   └── core/              # Core internal logic
├── Dockerfile.io          # I/O container definition (Alpine + coreutils)
└── docs/                  # Documentation
    ├── .index/            # Documentation index
    └── examples/          # Example workflows
```


## Available Commands

### 1. Read Files: `<open filepath>`
```
Let me check the main file <open main.go>
```

### 2. Write/Create Files: `<write filepath>content</write>`
```
<write src/new.go>
package main

import "fmt"

func main() {
    fmt.Println("Hello World!")
}
</write>
```

### 3. Execute Commands: `<exec command arguments>`
```
Let me run the tests <exec go test ./...>
Now build the project <exec make build>
```

**Important**: The exec container (`python-go`) includes Python and Go. For other languages or tools, you may need to use a different image or build a custom one.

### 4. Semantic Search: `<search query>`
```
Find files related to authentication <search user login auth>
Find database code <search database connection query>
```

**Search Setup:**
```bash
# Install and start Ollama
curl -fsSL https://ollama.com/install.sh | sh

# Pull the embedding model
ollama pull nomic-embed-text

# Build search index
./llm-runtime --reindex
```



## Usage

### Basic Usage (Pipe Mode)

```bash
echo "Let me check the main file <open main.go>" | ./llm-runtime
```

### With Exec Commands Enabled

**The exec container (`python-go`) includes Python and Go**, so you can run these commands immediately:

```bash
echo "Run Go tests: <exec go test ./...>" | ./llm-runtime
echo "Run Python script: <exec python3 script.py>" | ./llm-runtime 
```

**For other languages** (Node.js, Rust, etc.), you can specify a different Docker image in the config:

```yaml
commands:
  exec:
    enabled: true
    container_image: "your-custom-image"  # Replace python-go
```

### Interactive Mode

```bash
./llm-runtime --interactive 
```

In interactive mode, the tool continuously processes input and executes commands as they appear.

### File Mode

```bash
./llm-runtime --input llm_output.txt --output results.txt 
```

## Command Line Options

### Basic Options
- `--root PATH`: Repository root directory (default: current directory)
- `--max-size BYTES`: Maximum file size in bytes (default: 1048576 = 1MB)
- `--interactive`: Run in interactive mode
- `--input FILE`: Read from file instead of stdin
- `--output FILE`: Write to file instead of stdout
- `--verbose`: Enable verbose output

### Write Command Options
- `--max-write-size BYTES`: Maximum write file size (default: 100KB)
- `--backup`: Create backup before overwriting files (default: true)
- `--allowed-extensions`: Comma-separated list of allowed file extensions
- `--force`: Force write even if conflicts exist

### Exec Command Options
- `--exec-timeout DURATION`: Timeout for exec commands (default: 30s)
- `--exec-memory LIMIT`: Memory limit for containers (default: 512m)
- `--exec-cpu LIMIT`: CPU limit for containers (default: 2)
- `--exec-image IMAGE`: Docker image for exec commands (default: ubuntu:22.04)
- `--exec-whitelist`: Comma-separated list of allowed commands

### I/O Container Options
- `--io-image IMAGE`: Docker image for I/O operations (default: llm-runtime-io:latest)
- `--io-timeout DURATION`: Timeout for I/O operations (default: 60s)
- `--io-memory LIMIT`: Memory limit for I/O containers (default: 256m)
- `--io-cpu LIMIT`: CPU limit for I/O containers (default: 1)

### Security Options
- `--exclude PATTERNS`: Comma-separated list of excluded paths (default: ".git,.env,*.key,*.pem")

## Security Features

### Path Validation
- Canonicalizes all paths using OS-native functions
- Resolves symlinks and verifies final destination
- Prevents directory traversal attempts (../)
- Ensures all accessed files are within repository bounds


### Exec Command Security:
- **Docker Isolation**: Commands run in isolated Docker containers
- **No Network Access**: Containers have no network connectivity (`--network none`)
- **Read-Only Repository**: Repository mounted read-only at `/workspace`
- **Temporary Writes**: Separate writable directory at `/tmp/workspace`
- **Resource Limits**: Memory (512M), CPU (2 cores), and time (30s) limits
- **Command Whitelisting**: Only pre-approved commands allowed
- **Non-Root Execution**: Commands run as unprivileged user (1000:1000)
- **Security Options**: Capabilities dropped, no new privileges
- **Configurable**: Control via `commands.exec.enabled` and whitelist

### I/O Containerization:
- **File Reads**: Execute via `cat` in minimal Alpine container
- **File Writes**: Atomic operations via temp files in container
- **Path Isolation**: Container provides additional layer beyond path validation
- **Resource Limits**: Configurable memory (256M), CPU (1 core), timeout (60s)
- **Minimal Attack Surface**: Alpine-based image with only coreutils



### Default Whitelisted Commands

These commands work with the `python-go` exec container:

```yaml
whitelist:
  # Go (included in python-go)
  - "go test"
  - "go build"
  - "go run"
  - "go mod"
  - "go version"
  
  # Python (included in python-go)
  - "python"
  - "python3"
  - "python -m pytest"
  - "pip install"
  
  # Build tools (included in python-go)
  - "make"
  - "make test"
  - "make build"
  
  # System utilities (included in python-go)
  - "ls"
  - "cat"
  - "grep"
  - "find"
  - "head"
  - "tail"
  - "wc"
```

### Additional Languages

For Node.js, Rust, or other languages, use a different Docker image:

```yaml
commands:
  exec:
    container_image: "node:alpine"  # For Node.js
    # or
    container_image: "rust:alpine"  # For Rust
```

### Excluded Paths
By default, the following are excluded:
- `.git` directory
- `.env` files
- `*.key` files
- `*.pem` files
- `node_modules`
- `__pycache__`
- `*.sqlite`, `*.db`

### Audit Logging
All operations are logged to `audit.log` with:
- ISO 8601 timestamp
- Session ID
- Command type
- File path or command
- Success/failure status
- Execution details (exit codes, duration)
- Error messages (if any)

Example audit log entries:
```
2025-11-22T10:30:45Z|session:1234567890|open|src/main.go|success|
2025-11-22T10:30:46Z|session:1234567890|exec|go test|success|exit_code:0,duration:1.234s
2025-11-22T10:30:47Z|session:1234567890|exec|rm -rf /|failed|EXEC_VALIDATION: command not in whitelist: rm
```

## Example LLM Integration

### System Prompt for LLM
```
You have access to a file system tool that allows you to explore repositories, create/modify files, 
and execute commands in a secure environment. You can use these commands in your responses:

- <open filepath> - Read file contents
- <write filepath>content</write> - Create or update files  
- <exec command args> - Execute commands in Docker container
- <search query> - Search for files using semantic search

IMPORTANT NOTES:
- All file operations happen in isolated Docker containers
- Exec commands run in isolated containers with no network access
- Only whitelisted exec commands are allowed
- The exec container must have required tools (go, node, python, etc.) installed
- File size limits apply (1MB for reads, 100KB for writes by default)

Use exec to run tests, build projects, and validate changes. Always verify the exec 
container has the necessary tools before using language-specific commands.
```

### Example Session

**User**: "Help me understand this Go project and make sure it works"

**LLM**: "I'll explore the project comprehensively and verify it works correctly.

<open go.mod>

Let me check the main entry point:

<open cmd/main.go>

Now I'll run the tests to verify functionality:

<exec go test ./...>

Let me also build the project to ensure it compiles:

<exec go build -o bin/app cmd/main.go>

Finally, let me check the code quality:

<exec go vet ./...>
<exec go fmt ./...>"

**Tool Output**:
```
=== LLM TOOL START ===
I'll explore the project comprehensively and verify it works correctly.

<open go.mod>
=== COMMAND: <open go.mod> ===
=== FILE: go.mod ===
module github.com/example/project

go 1.21

require (
    github.com/gin-gonic/gin v1.9.0
)
=== END FILE ===
=== END COMMAND ===

[...file contents and command execution results...]

=== COMMAND: <exec go test ./...> ===
=== EXEC SUCCESSFUL: go test ./... ===
Exit code: 0
Duration: 2.150s
Output:
?       github.com/example/project/cmd  [no test files]
ok      github.com/example/project/pkg  0.123s
=== END EXEC ===

=== LLM TOOL COMPLETE ===
Commands executed: 6
Time elapsed: 3.45s
=== END ===
```

## Docker Setup for Containers

### I/O Container (Required)

The I/O container is minimal and built from source:

```dockerfile
# Dockerfile.io
FROM golang:1.22.2-alpine

# Create non-root user
RUN addgroup -g 1000 llmuser && \
    adduser -D -u 1000 -G llmuser llmuser

# Install only essential tools
RUN apk add --no-cache coreutils

# Set working directory
WORKDIR /workspace

# Switch to non-root user
USER llmuser

CMD ["/bin/sh"]
```

Build it with:
```bash
make build-io-image
```

### Exec Container (python-go)

The exec container is a prebuilt image that includes Python and Go:

```bash
# Pull the image
docker pull python-go
```

This image contains:
- Python (python3)
- Go
- Common development tools (git, make, etc.)

**To use a different exec image:**

Edit `llm-runtime.config.yaml`:
```yaml
commands:
  exec:
    enabled: true
    container_image: "node:alpine"  # Or any other image
```

Popular alternatives:
- `node:alpine` - For Node.js projects
- `rust:alpine` - For Rust projects
- `ubuntu:22.04` - Minimal Ubuntu (won't have Go/Python/Node by default)

### Runtime Security
```bash
docker run \
    --rm \
    --network none \
    --user 1000:1000 \
    --cap-drop ALL \
    --security-opt no-new-privileges \
    --read-only \
    --tmpfs /tmp \
    --memory 512m \
    --cpus 2 \
    -v /repo:/workspace:ro \
    -v /temp:/tmp/workspace:rw \
    --workdir /workspace \
    <container-image> \
    sh -c "command"
```

## Testing

### Unit Tests
```bash
make test
```

### Comprehensive Test Suite
```bash
make test-suite
```

### Individual Feature Tests
```bash
make test-write      # Test write functionality
make test-exec       # Test exec functionality (requires Docker)
make quick-test      # Quick smoke test
```

### Security Tests
```bash
./security_test.sh
```

## Demos

### Basic Demo
```bash
make demo
```

### Exec Command Demo
```bash
make exec-demo
```

### Write Command Demo
```bash
./write_demo.sh
```

### Example Usage
```bash
make example
```

## I/O Container Management

### Build I/O Container Image
```bash
make build-io-image
```

### Verify I/O Image
```bash
make check-io-image
```

### Test I/O Operations
```bash
make test-io-container
```

### Clean I/O Image
```bash
make clean-io-image
```

## Performance

- Command parsing: <1ms for typical input
- Path validation: <1ms per path
- File read (1MB): <10ms on SSD
- Docker container startup: 1-3s (cached images)
- Total overhead: ~1-5s per exec command


### Container Pooling

By default, each file operation creates a new Docker container. Container pooling pre-creates containers and reuses them across operations.

**Performance improvement:** 5-10x faster for workflows with multiple operations (reduces per-operation overhead from 1-3 seconds to ~100-200ms).

**Enable in llm-runtime.config.yaml:**
```yaml
container_pool:
  enabled: true
  size: 5
  max_uses_per_container: 100
  idle_timeout: 5m
  health_check_interval: 30s
  startup_containers: 2
```

The pool automatically handles health checks, container recycling, and cleanup.

**Note**: Container pooling only applies to I/O operations (`<open>` and `<write>`), not `<exec>` commands.

## Configuration

Edit `llm-runtime.config.yaml` to customize:

```yaml
# Repository settings
repository:
  root: "."
  excluded_paths:
    - ".git"
    - ".env"
    - "*.key"
    - "secrets/"

commands:
  # Open command (uses I/O container)
  open:
    enabled: true
    max_file_size: 1048576  # 1MB
    
  # Write command (uses I/O container)
  write:
    enabled: true
    max_file_size: 102400  # 100KB
    backup_before_write: true
    allowed_extensions:
      - ".go"
      - ".py"
      - ".js"
      - ".md"

  # Exec command (uses exec container)
  exec:
    enabled: true
    container_image: "python-go"  # Has Python and Go
    timeout_seconds: 30
    memory_limit: "512m"
    cpu_limit: 2
    network_enabled: false  # Keep disabled for security
    whitelist:
      - "go test"
      - "go build"
      - "python3"
      - "make test"
      # Add more commands as needed

  # I/O Containerization settings
  io:
    container_image: "llm-runtime-io:latest"
    timeout: "60s"
    memory_limit: "256m"
    cpu_limit: 1

  # Search command
  search:
    enabled: false  # Set to true to enable
    ollama_url: "http://localhost:11434"
    embedding_model: "nomic-embed-text"
    max_results: 10

# Container pool (optional - for I/O operations only)
container_pool:
  enabled: true
  size: 5
  max_uses_per_container: 100
  idle_timeout: 5m
  health_check_interval: 30s
  startup_containers: 2

security:
  excluded_paths:
    - ".git"
    - "*.key"
    - "secrets/"
```

## Search Feature Setup

### Requirements
- Ollama installed and running
- nomic-embed-text model

### Installation Steps

```bash
# 1. Install Ollama
curl -fsSL https://ollama.com/install.sh | sh

# 2. Pull the embedding model
ollama pull nomic-embed-text

# 3. Verify Ollama is running
./llm-runtime check-ollama

# 4. Update configuration
# Edit llm-runtime.config.yaml:
#   commands.search.enabled: true
#   commands.search.embedding_model: "nomic-embed-text"
#   commands.search.ollama_url: "http://localhost:11434"

# 5. Build initial search index
./llm-runtime --reindex
```

### Search Management Commands

```bash
./llm-runtime reindex              # Full reindex
./llm-runtime search-status        # Show index stats
./llm-runtime search-validate      # Validate index
./llm-runtime search-cleanup       # Clean deleted files
./llm-runtime search-update        # Incremental update
./llm-runtime check-ollama         # Verify Ollama setup
```

### Search Usage

```bash
echo "Find authentication code <search user login auth>" | ./llm-runtime
```

## Documentation

For comprehensive documentation, see the [docs/](docs/) directory:

- **[Installation Guide](docs/installation.md)** - Complete setup instructions
- **[Configuration Reference](docs/configuration.md)** - All configuration options
- **[Feature Guides](docs/)** - Detailed guides for each command type
- **[Troubleshooting](docs/troubleshooting.md)** - Common issues and solutions
- **[FAQ](docs/faq.md)** - Frequently asked questions

## Troubleshooting

### Docker Issues
```bash
# Check Docker availability
make check-docker

# Pull required images
docker pull ubuntu:22.04

# Build I/O container
make build-io-image

# Test Docker permissions
docker run --rm ubuntu:22.04 echo "Docker working"
```

### I/O Container Missing
```
Error: I/O container image not found: llm-runtime-io:latest
```

**Solution:**
```bash
make build-io-image
make check-io-image
```

### Exec Commands Failing
```
Error: container image not available
```

**Solution:** Pull the python-go image:
```bash
docker pull python-go

# Verify it's available
docker image inspect python-go
```

If you need different tools, use a different image:
```bash
docker pull node:alpine  # For Node.js
# Then update config: container_image: "node:alpine"
```

### Permission Issues
```bash
# Make sure user is in docker group (Linux)
sudo usermod -aG docker $USER
newgrp docker
```

### Command Whitelist
If a command is blocked:
1. Check if it's in the whitelist
2. Verify the exec container has that tool installed
3. Add it to `llm-runtime.config.yaml` or use `--exec-whitelist`
4. Consider security implications

### Search/Ollama Issues
```bash
# Check Ollama is running
curl http://localhost:11434/api/tags

# Verify model is pulled
ollama list

# Pull the model if missing
ollama pull nomic-embed-text
```

## Future Enhancements

Planned features for future versions:

- **Git Integration**: `<git command>` for version control operations
- **Enhanced Containers**: Language-specific optimized images
- **Streaming Output**: Real-time command output streaming
- **Resource Monitoring**: Container resource usage tracking
- **Plugin System**: Custom command extensions
- **MCP Integration**: Model Context Protocol support

## Security Considerations

1. **Never run as root**: The tool should run with minimal privileges
2. **Docker security**: Ensure Docker daemon is properly configured
3. **Restrict repository access**: Only expose repositories you trust the LLM to access
4. **Monitor audit logs**: Regularly review audit.log for suspicious patterns
5. **Resource limits**: Adjust limits based on your security requirements
6. **Network isolation**: Exec commands have no network access by design
7. **Command whitelisting**: Only allow necessary commands for your use case
8. **Container isolation**: Both I/O and exec operations run in isolated containers
9. **Custom exec images**: Carefully vet tools included in custom exec containers
10. **Keep containers updated**: Regularly rebuild containers with security patches

## Contributing

1. Ensure all tests pass
2. Add tests for new features
3. Update documentation
4. Follow Go best practices
5. Test with Docker security in mind

## License

[License information]

## Support

- **Issues**: [Report bugs](https://github.com/computerscienceiscool/llm-runtime/issues)
- **Discussions**: [Community discussions](https://github.com/computerscienceiscool/llm-runtime/discussions)
- **Documentation**: [Complete docs](docs/)
- **Quick Start Guides**: [Getting started](docs/quick-reference.md)
- **Examples**: [Example projects](examples/) This will help users get started quickly and will be done in future updates(coming soon).

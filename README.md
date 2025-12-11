# LLM File Access Tool

A secure tool that enables Large Language Models to autonomously explore and work with local repositories through embedded commands in their responses. Now includes secure command execution capabilities!

## Features

- **Secure Path Validation**: Prevents directory traversal and access outside repository boundaries
- **Command Parsing**: Extracts and executes `<open filename>`, `<write filename>content</write>`, and `<exec command>` commands from LLM output
- **Docker-based Execution**: Secure, isolated command execution in Docker containers with no network access
- **Audit Logging**: Tracks all operations with timestamps and results
- **Multiple Modes**: Supports pipe, interactive, and file-based operation
- **Configurable Security**: Exclude sensitive paths, whitelist commands, set resource limits

## Installation

### Prerequisites
- Go 1.21 or later
- Docker (for exec commands)
- Ollama with nomic-embed-text model (for search feature)

### Build from source
```bash
git clone https://github.com/computerscienceiscool/llm-runtime.git
cd llm-runtime
make build
```

Or build directly:
```bash
go build -o llm-runtime main.go
```

### Quick setup
```bash
chmod +x setup.sh
./setup.sh
```

## Project Structure

```
llm-runtime/
├── cmd/llm-runtime/       # Entry point
├── internal/              # App-specific code
│   ├── app/               # Application bootstrap
│   ├── cli/               # Command-line handling
│   ├── config/            # Configuration loading
│   ├── search/            # Semantic search (Ollama)
│   └── session/           # Session management
├── pkg/                   # Public API (importable)
│   ├── evaluator/         # Command execution
│   ├── sandbox/           # Security, Docker isolation
│   └── scanner/           # Command parsing
└── docs/                  # Documentation
    └── examples/          # Example workflows (coming soon)

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
### 4. Semantic Search: `<search query>`
```
Find files related to authentication <search user login auth>
Find database code <search database connection query>
```

**Search Setup:**
```bash
# Install and start Ollama
curl -fsSL https://ollama.com/install.sh | sh
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

```bash
echo "Check code and run tests: <open main.go> <exec go test>" | ./llm-runtime --exec-enabled
```

### Interactive Mode

```bash
./llm-runtime --interactive --exec-enabled
```

In interactive mode, the tool continuously processes input and executes commands as they appear.

### File Mode

```bash
./llm-runtime --input llm_output.txt --output results.txt --exec-enabled
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
- `--exec-enabled`: Enable exec command (default: false)
- `--exec-timeout DURATION`: Timeout for exec commands (default: 30s)
- `--exec-memory LIMIT`: Memory limit for containers (default: 512m)
- `--exec-cpu LIMIT`: CPU limit for containers (default: 2)
- `--exec-image IMAGE`: Docker image for exec commands (default: ubuntu:22.04)
- `--exec-whitelist`: Comma-separated list of allowed commands

### Security Options
- `--exclude PATTERNS`: Comma-separated list of excluded paths (default: ".git,.env,*.key,*.pem")

## Security Features

### Path Validation
- Canonicalizes all paths using OS-native functions
- Resolves symlinks and verifies final destination
- Prevents directory traversal attempts (../)
- Ensures all accessed files are within repository bounds

### Exec Command Security
- **Docker Isolation**: Commands run in isolated Docker containers
- **No Network Access**: Containers have no network connectivity (`--network none`)
- **Read-Only Repository**: Repository mounted read-only at `/workspace`
- **Temporary Writes**: Separate writable directory at `/tmp/workspace`
- **Resource Limits**: Memory (512M), CPU (2 cores), and time (30s) limits
- **Command Whitelisting**: Only pre-approved commands allowed
- **Non-Root Execution**: Commands run as unprivileged user
- **Security Options**: Capabilities dropped, no new privileges

### Default Whitelisted Commands
```
Go:        go test, go build, go run, go mod
Node.js:   npm test, npm run build, npm install, node
Python:    python, python3, python -m pytest, pip install
Build:     make, make test, make build
Rust:      cargo build, cargo test, cargo run
System:    ls, cat, grep, find, head, tail, wc
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

All exec commands run in isolated containers with no network access and only whitelisted 
commands are allowed. Use exec to run tests, build projects, and validate changes.
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

## Docker Setup for Exec Commands

The tool automatically uses Docker to create secure, isolated environments for command execution. The default setup:

### Container Configuration
```dockerfile
# Based on ubuntu:22.04 with common tools pre-installed
FROM ubuntu:22.04

RUN apt-get update && apt-get install -y \
    golang-go \
    python3 \
    python3-pip \
    nodejs \
    npm \
    make \
    gcc \
    git \
    && rm -rf /var/lib/apt/lists/*
```

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
    ubuntu:22.04 \
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

## Performance

- Command parsing: <1ms for typical input
- Path validation: <1ms per path
- File read (1MB): <10ms on SSD
- Docker container startup: 1-3s (cached images)
- Total overhead: ~1-5s per exec command

## Configuration

Edit `llm-runtime.config.yaml` to customize:

```yaml
commands:
  exec:
    enabled: true
    whitelist:
      - "go test"
      - "npm build"
      - "python3 -m pytest"
    timeout_seconds: 30
    memory_limit: "512m"
    cpu_limit: 2
    container_image: "ubuntu:22.04"

security:
  excluded_paths:
    - ".git"
    - "*.key"
    - "secrets/"
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

# Pull required image
docker pull ubuntu:22.04

# Test Docker permissions
docker run --rm ubuntu:22.04 echo "Docker working"
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
2. Add it to `llm-runtime.config.yaml` or use `--exec-whitelist`
3. Consider security implications

## Future Enhancements

Planned features for future versions:

- **Vector Search**: `<search query>` for semantic file search
- **Git Integration**: `<git command>` for version control operations
- **Enhanced Containers**: Language-specific optimized images
- **Streaming Output**: Real-time command output streaming
- **Resource Monitoring**: Container resource usage tracking
- **Plugin System**: Custom command extensions

## Security Considerations

1. **Never run as root**: The tool should run with minimal privileges
2. **Docker security**: Ensure Docker daemon is properly configured
3. **Restrict repository access**: Only expose repositories you trust the LLM to access
4. **Monitor audit logs**: Regularly review audit.log for suspicious patterns
5. **Resource limits**: Adjust limits based on your security requirements
6. **Network isolation**: Exec commands have no network access by design
7. **Command whitelisting**: Only allow necessary commands for your use case

## Contributing

1. Ensure all tests pass: `make test-suite`
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
- **Examples**: [Example projects](examples/) This will help users get started quickly and will be done in future updates.

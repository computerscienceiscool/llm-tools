# Quick Reference Guide

Fast reference for llm-runtime commands, configurations, and common patterns.

## Command Syntax

### File Reading
```
<open path/to/file>
```

**Examples:**
```
<open main.go>
<open src/components/App.tsx>
<open config/settings.yaml>
<open .github/workflows/ci.yml>
```

### File Writing
```
<write path/to/file>
content here
can span multiple lines
</write>
```

**Examples:**
```
<write config.yaml>
server:
  port: 8080
</write>

<write main.go>
package main

func main() {
    println("Hello")
}
</write>
```

### Command Execution
```
<exec command arguments>
```

**Examples:**
```
<exec go test>
<exec npm build>
<exec python -m pytest>
<exec make clean>
```

### Semantic Search
```
<search query terms>
```

**Examples:**
```
<search user authentication>
<search database connection>
<search error handling>
<search main entry point>
```

## CLI Usage

### Basic Invocation
```bash
# Interactive mode (stdin)
./llm-runtime

# Single command
echo "<open README.md>" | ./llm-runtime

# From file
cat commands.txt | ./llm-runtime
```

### Common Flags
```bash
# Configuration
./llm-runtime --config custom.yaml

# Logging
./llm-runtime --log-level debug
./llm-runtime --audit-log logs/audit.log

# Search
./llm-runtime --reindex
./llm-runtime --search-db custom.db

# Timeouts
./llm-runtime --exec-timeout 60s
./llm-runtime --io-timeout 30s

# Resource limits
./llm-runtime --exec-memory 1g
./llm-runtime --exec-cpu 4
./llm-runtime --io-memory 256m

# Container images
./llm-runtime --exec-container golang:1.21
./llm-runtime --io-container alpine:latest

# Whitelist
./llm-runtime --exec-whitelist "go test,npm build,make"
```

### Help & Information
```bash
./llm-runtime --help
./llm-runtime --version
```

## Configuration File

### Location
Default: `./llm-runtime.config.yaml`

Custom: `./llm-runtime --config /path/to/config.yaml`

### Structure

```yaml
# Logging
logging:
  level: "info"  # debug, info, warn, error
  format: "text" # text or json

# Audit logging
audit:
  enabled: true
  file: "audit.log"

# Search configuration
search:
  enabled: true
  database_path: "embeddings.db"
  ollama_url: "http://localhost:11434"
  ollama_model: "nomic-embed-text"
  ollama_timeout: 30s
  chunk_size: 1000
  chunk_overlap: 200

# Command execution
commands:
  exec:
    container_image: "python-go"
    timeout_seconds: 30
    memory_limit: "512m"
    cpu_limit: 2
    network_enabled: false
    whitelist:
      - "go test"
      - "go build"
      - "go run"
      - "go mod tidy"
      - "npm test"
      - "npm run build"
      - "npm install"
      - "node"
      - "python"
      - "python3"
      - "python -m pytest"
      - "pip install"
      - "make"
      - "make test"
      - "make build"
      - "cargo build"
      - "cargo test"
      - "cargo run"
      - "ls"
      - "cat"
      - "grep"
      - "find"
      - "head"
      - "tail"
      - "wc"

  # I/O containerization
  io:
    container_image: "llm-runtime-io:latest"
    timeout_seconds: 10
    memory_limit: "128m"
    cpu_limit: 1
    fallback_image: "alpine:latest"
```

## Docker Setup

### Required Images
```bash
# Build I/O container
make build-io-image

# Or pull Alpine fallback
docker pull alpine:latest

# Pull exec container (if not using custom)
docker pull python-go:latest
```

### Verify Images
```bash
docker images | grep -E "llm-runtime-io|python-go|alpine"
```

### Container Management
```bash
# List running containers
docker ps

# Remove stopped containers
docker container prune

# Remove unused images
docker image prune -a

# Full cleanup
docker system prune -a
```

## Ollama Setup (for Search)

### Installation
```bash
# Linux
curl -fsSL https://ollama.com/install.sh | sh

# macOS
brew install ollama

# Windows - download from ollama.com
```

### Model Setup
```bash
# Pull embedding model
ollama pull nomic-embed-text

# Verify installation
ollama list

# Start Ollama (if not running)
ollama serve
```

### Build Search Index
```bash
# Build index
./llm-runtime --reindex

# Verify index
ls -lh embeddings.db

# Query index
sqlite3 embeddings.db "SELECT COUNT(*) FROM embeddings;"
```

## Common Workflows

### Code Analysis
```
Let me examine the authentication system:

<search authentication>

<open internal/auth/auth.go>

<open internal/auth/middleware.go>

Now I understand the structure.
```

### Making Changes
```
I'll update the configuration:

<open config/settings.yaml>

<write config/settings.yaml>
server:
  port: 3000
  host: localhost
</write>

<exec go test ./...>

Changes applied and tested successfully.
```

### Testing & Building
```
<exec go test ./...>

<exec go build -o bin/app .>

<exec ls -la bin/>

Build completed successfully.
```

### Debugging
```
<exec go test -v -run TestAuth>

<open internal/auth/auth_test.go>

<write internal/auth/auth.go>
// Fixed implementation
</write>

<exec go test ./internal/auth>
```

### Multi-Step Workflow
```
<search database connection>

<open internal/database/connection.go>

<write internal/database/connection.go>
// Updated implementation
</write>

<exec go test ./internal/database>

<exec go build ./...>

All tests passing, build successful.
```

## Default Whitelisted Commands

### Go
- `go test`
- `go build`
- `go run`
- `go mod tidy`
- `go vet`
- `go fmt`

### Node.js
- `npm test`
- `npm run build`
- `npm install`
- `node`

### Python
- `python`
- `python3`
- `python -m pytest`
- `pip install`

### Build Tools
- `make`
- `make test`
- `make build`
- `make clean`

### Rust
- `cargo build`
- `cargo test`
- `cargo run`

### System Commands
- `ls`
- `cat`
- `grep`
- `find`
- `head`
- `tail`
- `wc`

## Error Codes & Messages

| Error Type | Meaning | Common Fix |
|------------|---------|------------|
| `DOCKER_UNAVAILABLE` | Docker not running | Start Docker daemon |
| `EXEC_VALIDATION` | Command not whitelisted | Add to whitelist |
| `EXEC_TIMEOUT` | Command too slow | Increase timeout |
| `EXEC_FAILED` | Command returned error | Fix underlying issue |
| `READ_VALIDATION` | Invalid file path | Use relative path |
| `READ_FAILED` | File not found | Check file exists |
| `WRITE_VALIDATION` | Invalid file path | Use relative path |
| `WRITE_FAILED` | Write permission issue | Check permissions |
| `SEARCH_FAILED` | Search error | Check Ollama/index |
| `CONFIG_INVALID` | Bad config syntax | Validate YAML |
| `IO_TIMEOUT` | I/O operation too slow | Increase timeout |

## Makefile Targets

```bash
# Build
make build          # Build binary
make clean          # Remove build artifacts
make rebuild        # Clean + build

# Testing
make test           # Run tests
make test-verbose   # Verbose tests
make test-coverage  # Coverage report

# Docker
make build-io-image     # Build I/O container
make check-io-image     # Verify I/O image exists
make pull-images        # Pull all required images

# Search
make reindex        # Rebuild search index

# Installation
make install        # Install to $GOPATH/bin
make uninstall      # Remove from $GOPATH/bin

# Development
make fmt            # Format code
make lint           # Run linters
make vet            # Run go vet
```

## File Paths

### Tool Files
- **Binary**: `./llm-runtime`
- **Config**: `./llm-runtime.config.yaml`
- **Audit log**: `./audit.log`
- **Search index**: `./embeddings.db`

### Docker
- **Dockerfile (I/O)**: `./Dockerfile.io`
- **Dockerfile (exec)**: Custom (or use public images)

### Documentation
- **Main README**: `./README.md`
- **Docs directory**: `./docs/`
- **Guides**: `./docs/*-guide.md`

## Environment Variables

```bash
# Docker
DOCKER_HOST           # Docker daemon address
DOCKER_CONFIG         # Docker config directory

# Ollama
OLLAMA_HOST          # Ollama server URL

# Tool behavior
LLM_RUNTIME_CONFIG   # Override default config path
```

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | General error |
| 2 | Configuration error |
| 3 | Docker unavailable |
| 4 | Validation error |

## Performance Tips

### Speed Up Container Operations
```bash
# Pre-pull images
make pull-images
make build-io-image

# Use SSD for Docker storage

# Increase resource limits
./llm-runtime --exec-memory 1g --exec-cpu 4
```

### Optimize Search
```bash
# Rebuild index periodically
./llm-runtime --reindex

# Exclude unnecessary files (.gitignore)

# Use specific search terms
```

### Reduce Overhead
```bash
# Use smaller container images
./llm-runtime --io-container alpine:latest

# Limit command execution when not needed

# Disable features you don't use
```

## Security Checklist

- [ ] Docker installed and configured
- [ ] Running as non-root user
- [ ] Whitelist properly configured (minimal commands)
- [ ] Network disabled in containers
- [ ] Audit logging enabled
- [ ] File paths validated
- [ ] Resource limits configured
- [ ] Regular security updates

## Debugging Commands

### Check System Status
```bash
# Docker
docker --version
docker info
docker ps
docker images

# Ollama
ollama list
curl http://localhost:11434/api/tags

# Tool
./llm-runtime --version
ls -lh embeddings.db
cat audit.log
```

### Test Components
```bash
# Docker I/O
docker run --rm llm-runtime-io:latest cat /etc/os-release

# Docker exec
docker run --rm python-go:latest go version

# Basic Alpine
docker run --rm alpine:latest echo "test"

# File operations
echo "<open README.md>" | ./llm-runtime

# Search
echo "<search main>" | ./llm-runtime
```

### Enable Verbose Logging
```bash
./llm-runtime --log-level debug --verbose
```

## Quick Troubleshooting

| Problem | Quick Fix |
|---------|-----------|
| Docker not found | `sudo systemctl start docker` |
| Permission denied | `sudo usermod -aG docker $USER` |
| Image not found | `make build-io-image` |
| Ollama not running | `ollama serve` |
| Model missing | `ollama pull nomic-embed-text` |
| Index missing | `./llm-runtime --reindex` |
| Config invalid | Check YAML syntax |
| Command not whitelisted | Add to config |

## Integration Examples

### With LLM API
```python
import subprocess

def run_command(cmd):
    result = subprocess.run(
        ['./llm-runtime'],
        input=cmd,
        capture_output=True,
        text=True
    )
    return result.stdout

# Use in LLM workflow
response = run_command("<search authentication>")
```

### With Shell Script
```bash
#!/bin/bash

# Test and build
echo "<exec go test ./...>" | ./llm-runtime
echo "<exec go build .>" | ./llm-runtime
```

### In CI/CD
```yaml
# GitHub Actions example
- name: Run LLM-assisted tests
  run: |
    echo "<exec go test ./...>" | ./llm-runtime
    echo "<exec go build .>" | ./llm-runtime
```

## Resource Limits

### Defaults
- **Exec container**: 512MB RAM, 2 CPUs, 30s timeout
- **I/O container**: 128MB RAM, 1 CPU, 10s timeout
- **Search**: No specific limits (uses Ollama resources)

### Customization
```yaml
commands:
  exec:
    memory_limit: "1g"
    cpu_limit: 4
    timeout_seconds: 60
  io:
    memory_limit: "256m"
    cpu_limit: 2
    timeout_seconds: 30
```

## Audit Log Format

```
TIMESTAMP|SESSION_ID|OPERATION|TARGET|STATUS|METADATA
```

**Example:**
```
2025-12-15T10:30:45Z|abc123|read|config.yaml|success|size:1234
2025-12-15T10:30:46Z|abc123|exec|go test|success|exit:0,duration:2.5s
2025-12-15T10:30:47Z|abc123|write|main.go|success|bytes:5678
2025-12-15T10:30:48Z|abc123|search|authentication|success|results:5
```

## Common Patterns

### Read → Analyze → Update
```
<open config/app.yaml>
<write config/app.yaml>
updated: content
</write>
<exec go test ./...>
```

### Search → Read → Modify
```
<search database connection>
<open internal/db/connection.go>
<write internal/db/connection.go>
// improvements
</write>
```

### Test → Fix → Verify
```
<exec go test -v>
<open failed_test_file.go>
<write failed_test_file.go>
// fix
</write>
<exec go test -v>
```

---

**For detailed information**, see:
- [llm-runtime-overview.md](llm-runtime-overview.md)
- [configuration.md](configuration.md)
- [troubleshooting.md](troubleshooting.md)
- Command-specific guides in `docs/`

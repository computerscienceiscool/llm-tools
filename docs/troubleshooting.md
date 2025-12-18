# Troubleshooting Guide

This guide helps diagnose and resolve common issues with llm-runtime.

## Table of Contents

- [Docker Issues](#docker-issues)
- [Search Issues](#search-issues)
- [File Operation Issues](#file-operation-issues)
- [Command Execution Issues](#command-execution-issues)
- [Configuration Issues](#configuration-issues)
- [Performance Issues](#performance-issues)
- [Error Messages](#error-messages)

## Docker Issues

### Docker Daemon Not Running

**Symptoms:**
```
=== ERROR: DOCKER_UNAVAILABLE ===
Message: docker daemon not running
```

**Solutions:**
```bash
# Check Docker status
systemctl status docker

# Start Docker (Linux)
sudo systemctl start docker

# macOS - start Docker Desktop application

# Windows - start Docker Desktop application

# Verify Docker is accessible
docker --version
docker info
```

### Docker Permission Denied

**Symptoms:**
```
=== ERROR: DOCKER_UNAVAILABLE ===
Message: permission denied while trying to connect to Docker daemon
```

**Solutions:**
```bash
# Add user to docker group (Linux)
sudo usermod -aG docker $USER
newgrp docker

# Or use sudo (not recommended for regular use)
sudo ./llm-runtime

# Verify group membership
groups $USER | grep docker

# Test Docker access
docker run --rm hello-world
```

### Container Image Not Found

**Symptoms:**
```
=== ERROR: DOCKER_UNAVAILABLE ===
Message: image not found: llm-runtime-io:latest
```

**Solutions:**
```bash
# Build I/O container image
make build-io-image

# Or pull Alpine fallback
docker pull alpine:latest

# Verify image exists
docker images | grep llm-runtime-io

# Check both I/O and exec images
docker images | grep -E "llm-runtime-io|python-go"
```

### Container Startup Failure

**Symptoms:**
```
=== ERROR: DOCKER_UNAVAILABLE ===
Message: failed to start container
```

**Solutions:**
```bash
# Check Docker disk space
docker system df

# Clean up unused containers/images
docker system prune -a

# Check Docker logs
docker logs $(docker ps -aq | head -1)

# Test basic container functionality
docker run --rm alpine:latest echo "test"
```

### Network Issues in Containers

**Symptoms:**
- Containers cannot access network resources
- DNS resolution fails

**Solution:**
This is expected behavior. Containers run with `--network none` for security. If you need network access, the tool must be modified to allow it (not recommended for security).

### I/O Container Build Issues

**Symptoms:**
```
make build-io-image
Error: failed to build image
```

**Solutions:**
```bash
# Check Dockerfile.io exists
ls -la Dockerfile.io

# Build manually with verbose output
docker build -t llm-runtime-io:latest -f Dockerfile.io . --progress=plain

# Check for Docker build errors
docker build --no-cache -t llm-runtime-io:latest -f Dockerfile.io .

# Fallback: Use Alpine directly
# (tool will automatically use alpine:latest if custom image unavailable)
docker pull alpine:latest
```

## Search Issues

### Ollama Not Running

**Symptoms:**
```
=== ERROR: SEARCH_FAILED ===
Message: failed to connect to Ollama: connection refused
```

**Solutions:**
```bash
# Check if Ollama is running
curl http://localhost:11434/api/tags

# Start Ollama service (Linux)
ollama serve

# Start Ollama (macOS - run Ollama app)

# Check Ollama status
ps aux | grep ollama

# Test Ollama manually
ollama list
```

### Embedding Model Not Downloaded

**Symptoms:**
```
=== ERROR: SEARCH_FAILED ===
Message: model not found: nomic-embed-text
```

**Solutions:**
```bash
# Pull the embedding model
ollama pull nomic-embed-text

# Verify model is downloaded
ollama list

# Check model size (should be ~274MB)
ollama show nomic-embed-text

# Test embedding generation
curl http://localhost:11434/api/embeddings -d '{
  "model": "nomic-embed-text",
  "prompt": "test"
}'
```

### Search Index Missing

**Symptoms:**
```
=== ERROR: SEARCH_FAILED ===
Message: search index not found: embeddings.db
```

**Solutions:**
```bash
# Build search index
./llm-runtime --reindex

# Check index file exists
ls -lh embeddings.db

# If file exists but corrupt, rebuild
rm embeddings.db
./llm-runtime --reindex

# Verify index has content
sqlite3 embeddings.db "SELECT COUNT(*) FROM embeddings;"
```

### Search Index Out of Date

**Symptoms:**
- Search returns old/deleted files
- New files not appearing in results

**Solutions:**
```bash
# Rebuild index from scratch
./llm-runtime --reindex

# Or rebuild with fresh start
rm embeddings.db
./llm-runtime --reindex

# Check index timestamp
ls -lh embeddings.db
```

### Slow Search Performance

**Symptoms:**
- Search takes several seconds
- High CPU usage during search

**Solutions:**
```bash
# Check index size
ls -lh embeddings.db

# For large repositories, consider:
# 1. Add files/directories to .gitignore
# 2. Exclude large binary files
# 3. Limit search to relevant directories

# Rebuild with optimizations
./llm-runtime --reindex

# Check Ollama performance
time ollama run nomic-embed-text "test query"
```

### Search Returns No Results

**Symptoms:**
```
=== SEARCH SUCCESSFUL: query ===
No results found
=== END SEARCH ===
```

**Solutions:**
```bash
# Verify index has content
sqlite3 embeddings.db "SELECT COUNT(*) FROM embeddings;"

# Try broader search terms
# Instead of: <search very specific technical term>
# Try: <search general concept>

# Rebuild index if needed
./llm-runtime --reindex

# Check if search worked during indexing
./llm-runtime --reindex --verbose
```

### Ollama Connection Timeout

**Symptoms:**
```
=== ERROR: SEARCH_FAILED ===
Message: timeout waiting for Ollama response
```

**Solutions:**
```bash
# Increase timeout in config
# Edit llm-runtime.config.yaml:
search:
  ollama_timeout: 60s  # increase from 30s

# Check Ollama resource usage
top -p $(pgrep ollama)

# Restart Ollama
pkill ollama
ollama serve

# Test Ollama responsiveness
time curl http://localhost:11434/api/tags
```

## File Operation Issues

### Path Outside Repository

**Symptoms:**
```
=== ERROR: READ_VALIDATION ===
Message: path outside repository
```

**Solutions:**
```bash
# Use relative paths from repository root
<open config/settings.yaml>  # ✅ Correct

# Not absolute paths
<open /etc/passwd>  # ❌ Wrong

# Not paths outside repository
<open ../../../etc/hosts>  # ❌ Wrong
```

### File Not Found

**Symptoms:**
```
=== ERROR: READ_FAILED ===
Message: file not found
```

**Solutions:**
```bash
# Check file exists
ls -la path/to/file

# Verify case-sensitive path
find . -name "filename"

# List directory contents
ls -la directory/

# Check from repository root
pwd
ls -la config/settings.yaml
```

### Write Permission Denied

**Symptoms:**
```
=== ERROR: WRITE_FAILED ===
Message: permission denied
```

**Solutions:**
```bash
# Check file permissions
ls -la path/to/file

# Fix file permissions if needed
chmod 644 path/to/file

# Check directory permissions
ls -ld path/to/

# Verify Docker has write access
docker run --rm -v $(pwd):/workspace alpine sh -c 'touch /workspace/test && rm /workspace/test'
```

### I/O Container Timeout

**Symptoms:**
```
=== ERROR: IO_TIMEOUT ===
Message: I/O operation exceeded timeout
```

**Solutions:**
```bash
# Increase I/O timeout
./llm-runtime --io-timeout 30s

# Or in config file:
commands:
  io:
    timeout_seconds: 30

# For very large files, read selectively
<exec head -n 100 large_file.log>
<exec grep "ERROR" large_file.log>
```

### I/O Container Memory Issues

**Symptoms:**
```
=== ERROR: DOCKER_UNAVAILABLE ===
Message: container out of memory
```

**Solutions:**
```bash
# Increase I/O container memory limit
./llm-runtime --io-memory 256m

# Or in config file:
commands:
  io:
    memory_limit: "256m"

# Avoid reading huge files
# Use exec with head/tail/grep instead
```

## Command Execution Issues

### Command Not in Whitelist

**Symptoms:**
```
=== ERROR: EXEC_VALIDATION ===
Message: command not in whitelist: rm
```

**Solutions:**
```bash
# Add command to whitelist in config
commands:
  exec:
    whitelist:
      - "go test"
      - "npm build"
      - "rm"  # Add your command

# Or use CLI flag
./llm-runtime --exec-whitelist "go test,npm build,rm"

# Check current whitelist
cat llm-runtime.config.yaml | grep -A 20 "whitelist:"
```

### Exec Timeout

**Symptoms:**
```
=== ERROR: EXEC_TIMEOUT ===
Message: command exceeded timeout (30s)
```

**Solutions:**
```bash
# Increase exec timeout
./llm-runtime --exec-timeout 60s

# Or in config file:
commands:
  exec:
    timeout_seconds: 60

# Optimize long-running commands
<exec go test -short ./...>  # Skip long tests
<exec make quick-build>       # Use faster build
```

### Exec Container Image Missing

**Symptoms:**
```
=== ERROR: DOCKER_UNAVAILABLE ===
Message: exec container image not found: python-go
```

**Solutions:**
```bash
# Pull or build the exec container image
docker pull python-go:latest

# Or use alternative image
./llm-runtime --exec-container golang:1.21

# Verify image exists
docker images | grep python-go
```

### Exec Command Failed

**Symptoms:**
```
=== EXEC SUCCESSFUL: go test ===
Exit code: 1
Output:
FAIL: TestExample
```

**Note:** This is NOT an error with llm-runtime. The command executed successfully, but the command itself failed (e.g., tests failed, build errors).

**Solutions:**
- Review the command output
- Fix the actual issue (failing tests, code errors, etc.)
- Re-run after fixing

## Configuration Issues

### Config File Not Found

**Symptoms:**
```
Warning: config file not found, using defaults
```

**Solutions:**
```bash
# Create config file
cp llm-runtime.config.yaml.example llm-runtime.config.yaml

# Or specify custom location
./llm-runtime --config /path/to/config.yaml

# Verify config file location
ls -la llm-runtime.config.yaml
```

### Invalid YAML Syntax

**Symptoms:**
```
=== ERROR: CONFIG_INVALID ===
Message: failed to parse config file
```

**Solutions:**
```bash
# Validate YAML syntax
python3 -c "import yaml; yaml.safe_load(open('llm-runtime.config.yaml'))"

# Or use online validator: https://www.yamllint.com/

# Check for common issues:
# - Tabs instead of spaces
# - Incorrect indentation
# - Missing colons
# - Unclosed quotes

# Reset to default config
cp llm-runtime.config.yaml.example llm-runtime.config.yaml
```

### Configuration Override Not Working

**Symptoms:**
- CLI flags seem ignored
- Config file settings not applying

**Solutions:**
```bash
# Check flag syntax
./llm-runtime --help

# Verify config file is being loaded
./llm-runtime --verbose

# CLI flags override config file
# Use correct flag names (check --help)
./llm-runtime --exec-timeout 60s  # ✅ Correct
./llm-runtime --timeout 60s       # ❌ Wrong flag name

# Check log output for config loading
./llm-runtime --log-level debug
```

### Audit Log Issues

**Symptoms:**
- Audit log not being created
- Audit log permission denied

**Solutions:**
```bash
# Check audit log path
ls -la audit.log

# Create directory if needed
mkdir -p logs
./llm-runtime --audit-log logs/audit.log

# Fix permissions
chmod 644 audit.log

# Verify audit logging in config
commands:
  audit:
    enabled: true
    file: "audit.log"
```

## Performance Issues

### Slow Startup Time

**Symptoms:**
- Tool takes several seconds to start

**Solutions:**
```bash
# Pre-pull Docker images
docker pull alpine:latest
docker pull python-go:latest

# Build I/O image in advance
make build-io-image

# Check disk I/O
iostat -x 1

# Disable search if not needed
# (removes index loading time)
./llm-runtime # search only loads on demand
```

### High Memory Usage

**Symptoms:**
- Tool consuming excessive RAM
- System becomes slow

**Solutions:**
```bash
# Check memory usage
ps aux | grep llm-runtime

# Reduce container memory limits
./llm-runtime --io-memory 64m --exec-memory 256m

# Or in config:
commands:
  io:
    memory_limit: "64m"
  exec:
    memory_limit: "256m"

# Rebuild search index with smaller chunks
./llm-runtime --reindex
```

### Container Overhead

**Symptoms:**
- Operations slower than expected
- High Docker overhead

**Solutions:**
```bash
# Use faster storage for Docker
# Move Docker to SSD if on HDD

# Pre-pull images to avoid download time
docker pull alpine:latest
docker pull python-go:latest
make build-io-image

# Use smaller images
./llm-runtime --io-container alpine:latest

# Optimize Docker daemon
# Edit /etc/docker/daemon.json
{
  "storage-driver": "overlay2",
  "storage-opts": ["overlay2.override_kernel_check=true"]
}

# Restart Docker
sudo systemctl restart docker
```

### Disk Space Issues

**Symptoms:**
```
no space left on device
```

**Solutions:**
```bash
# Check disk space
df -h

# Clean Docker system
docker system prune -a

# Remove old containers
docker container prune

# Remove unused images
docker image prune -a

# Check Docker space usage
docker system df
```

## Error Messages

### DOCKER_UNAVAILABLE

**Cause:** Docker not running or not accessible

**Solutions:**
1. Start Docker daemon/desktop
2. Check Docker permissions
3. Verify Docker installation

### EXEC_VALIDATION

**Cause:** Command not in whitelist

**Solutions:**
1. Add command to whitelist
2. Verify command spelling
3. Check whitelist configuration

### EXEC_TIMEOUT

**Cause:** Command exceeded time limit

**Solutions:**
1. Increase timeout
2. Optimize command
3. Split into smaller operations

### EXEC_FAILED

**Cause:** Command executed but returned error

**Solutions:**
1. Check command output
2. Fix underlying issue (not llm-runtime issue)
3. Verify command syntax

### READ_VALIDATION / WRITE_VALIDATION

**Cause:** Invalid file path

**Solutions:**
1. Use relative paths
2. Stay within repository bounds
3. Check path syntax

### READ_FAILED / WRITE_FAILED

**Cause:** File operation failed

**Solutions:**
1. Verify file exists (read)
2. Check permissions
3. Ensure directory exists (write)

### SEARCH_FAILED

**Cause:** Search operation failed

**Solutions:**
1. Verify Ollama running
2. Check embedding model downloaded
3. Rebuild search index

### CONFIG_INVALID

**Cause:** Configuration file syntax error

**Solutions:**
1. Validate YAML syntax
2. Check indentation
3. Use example config as template

### IO_TIMEOUT

**Cause:** File I/O operation too slow

**Solutions:**
1. Increase timeout
2. Avoid large files
3. Use exec for filtering

## Getting More Help

### Enable Verbose Logging

```bash
./llm-runtime --log-level debug --verbose
```

### Check Version

```bash
./llm-runtime --version
```

### Review Audit Log

```bash
tail -f audit.log
```

### Inspect Configuration

```bash
cat llm-runtime.config.yaml
```

### Test Docker Manually

```bash
# Test I/O container
docker run --rm llm-runtime-io:latest cat /etc/os-release

# Test exec container
docker run --rm python-go:latest go version

# Test basic Alpine
docker run --rm alpine:latest sh -c 'echo test'
```

### Verify Tool Build

```bash
# Rebuild from scratch
make clean
make build

# Run tests
make test

# Check dependencies
go mod verify
go mod tidy
```

### Common Debug Commands

```bash
# Check all Docker images
docker images

# List running containers
docker ps

# Check Docker logs
docker logs $(docker ps -lq)

# System info
./llm-runtime --version
go version
docker --version
ollama --version

# Environment check
env | grep -i docker
```

## Still Having Issues?

1. **Check existing issues**: Look for similar issues in project repository
2. **Create detailed bug report**: Include error messages, logs, configuration
3. **Provide reproduction steps**: Exact commands that cause the issue
4. **Include environment info**: OS, Docker version, Go version, etc.
5. **Check documentation**: Review relevant guides in `docs/` directory

## Quick Diagnostic Checklist

```bash
# Checklist
[ ] Docker installed and running
[ ] User in docker group (Linux)
[ ] Docker images available (python-go, llm-runtime-io, alpine)
[ ] Ollama installed and running (for search)
[ ] Embedding model downloaded (nomic-embed-text)
[ ] Search index built (./llm-runtime --reindex)
[ ] Configuration file valid YAML
[ ] Sufficient disk space
[ ] Correct file permissions
[ ] Repository path correct
```

Run this diagnostic:
```bash
echo "=== Docker ==="
docker --version && docker info | head -5

echo "=== Images ==="
docker images | grep -E "python-go|llm-runtime-io|alpine"

echo "=== Ollama ==="
ollama list 2>/dev/null || echo "Ollama not running"

echo "=== Search Index ==="
ls -lh embeddings.db 2>/dev/null || echo "No search index"

echo "=== Config ==="
ls -la llm-runtime.config.yaml 2>/dev/null || echo "No config file"

echo "=== Build ==="
./llm-runtime --version
```

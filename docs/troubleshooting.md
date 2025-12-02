# Troubleshooting Guide

Solutions for common issues and problems with the LLM File Access Tool.

## General Debugging

### Enable Verbose Mode
```bash
# Get detailed output
./llm-runtime --verbose

# Check current configuration
./llm-runtime --help
```

### Check Log Files
```bash
# Application logs
tail -f llm-runtime.log

# Audit logs
tail -f audit.log

# Search for specific errors
grep ERROR llm-runtime.log
```

## File Access Issues

### FILE_NOT_FOUND Errors
**Symptom**: `=== ERROR: FILE_NOT_FOUND ===`

**Causes & Solutions**:
```bash
# 1. Check file exists
ls -la filename

# 2. Verify path is relative to repository root
./llm-runtime --root /path/to/repo

# 3. Check for typos in filename
find . -name "*partial-name*"
```

### PATH_SECURITY Errors
**Symptom**: `=== ERROR: PATH_SECURITY ===`

**Common causes**:
- **Directory traversal**: `<open ../../../etc/passwd>`
- **Excluded paths**: `<open .git/config>` or `<open .env>`
- **Outside repository**: Absolute paths outside repo

**Solutions**:
```bash
# Check excluded paths in config
grep excluded_paths llm-runtime.config.yaml

# Modify exclusions if needed
./llm-runtime --exclude ".git,*.key"

# Use relative paths only
./llm-runtime --root /correct/repo/path
```

### RESOURCE_LIMIT Errors
**Symptom**: File too large errors

**Solutions**:
```bash
# Check file size
ls -lh filename

# Increase limits
./llm-runtime --max-size 2097152  # 2MB
./llm-runtime --max-write-size 204800  # 200KB

# Or edit config
```

```yaml
commands:
  open:
    max_file_size: 2097152
  write:
    max_file_size: 204800
```

## Exec Command Issues

### DOCKER_UNAVAILABLE Errors
**Symptom**: `=== ERROR: DOCKER_UNAVAILABLE ===`

**Diagnostics**:
```bash
# Check if Docker is installed
docker --version

# Check if Docker daemon is running
docker info

# Test basic Docker functionality
docker run --rm hello-world
```

**Solutions**:
```bash
# Linux: Start Docker service
sudo systemctl start docker

# Add user to docker group (avoid sudo)
sudo usermod -aG docker $USER
newgrp docker

# macOS: Start Docker Desktop
open -a Docker

# Windows: Start Docker Desktop
```

### EXEC_VALIDATION Errors
**Symptom**: `command not in whitelist`

**Check current whitelist**:
```bash
grep -A 20 "whitelist:" llm-runtime.config.yaml
```

**Add commands to whitelist**:
```yaml
commands:
  exec:
    whitelist:
      - "your-command"
      - "another-command"
```

**Temporary whitelist override**:
```bash
./llm-runtime --exec-whitelist "go test,npm build,your-command"
```

### EXEC_TIMEOUT Errors
**Symptom**: Commands timing out

**Solutions**:
```bash
# Increase timeout
./llm-runtime --exec-timeout 60s

# Or in config
```
```yaml
commands:
  exec:
    timeout_seconds: 60
```

**Optimize slow commands**:
```bash
# Instead of slow command
<exec go test ./...>

# Try faster subset
<exec go test -short ./...>
```

### EXEC_FAILED Errors
**Symptom**: Commands fail with non-zero exit codes

**Common issues**:
- **Missing dependencies**: Package not installed in container
- **Wrong working directory**: Command assumes different location
- **File permissions**: Container user can't access files

**Solutions**:
```bash
# Check what's available in container
<exec ls -la>
<exec which go>
<exec whoami>

# Use full paths if needed
<exec /usr/bin/go version>

# Install missing tools (if allowed)
<exec apt update && apt install tool-name>
```

## Search Issues

### SEARCH_DISABLED Errors
**Symptom**: `search feature is not enabled`

**Enable search**:
```yaml
commands:
  search:
    enabled: true
```

**Check Python dependencies**:
```bash
./llm-runtime --check-python-setup
```

### SEARCH_INIT_FAILED Errors
**Symptom**: Python dependencies not available

**Install dependencies**:
```bash
# Basic installation
pip install sentence-transformers

# With specific Python version
python3 -m pip install sentence-transformers

# In virtual environment (recommended)
python3 -m venv llm-env
source llm-env/bin/activate
pip install sentence-transformers

# Update config with correct Python path
```
```yaml
commands:
  search:
    python_path: "/path/to/python3"
```

### No Search Results
**Possible causes**:

**Index not built**:
```bash
# Build initial index
./llm-runtime --reindex

# Check index status
./llm-runtime --search-status
```

**Files not indexed**:
```bash
# Check which extensions are indexed
grep -A 10 "index_extensions" llm-runtime.config.yaml

# Add more file types
```
```yaml
commands:
  search:
    index_extensions:
      - ".go"
      - ".py"
      - ".js"
      - ".your-extension"
```

**Similarity threshold too high**:
```yaml
commands:
  search:
    min_similarity_score: 0.3  # Lower = more permissive
```

### Outdated Search Results
**Solution**:
```bash
# Update index for changed files
./llm-runtime --search-update

# Full rebuild if needed
./llm-runtime --reindex

# Clean up deleted files
./llm-runtime --search-cleanup
```

## Build and Installation Issues

### Go Version Issues
**Symptom**: Build fails with version errors

**Check Go version**:
```bash
go version
# Should show go1.21 or later
```

**Update Go**:
```bash
# Linux/macOS: Download from https://go.dev/dl/
wget https://go.dev/dl/go1.21.0.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.21.0.linux-amd64.tar.gz

# Update PATH
export PATH=$PATH:/usr/local/go/bin
```

### Dependency Issues
**Symptom**: Missing module errors

**Solutions**:
```bash
# Download dependencies
go mod download

# Clean module cache
go clean -modcache

# Tidy dependencies
go mod tidy

# Rebuild
go build -o llm-runtime main.go
```

### Permission Issues
**Linux/macOS**:
```bash
# Make tool executable
chmod +x llm-runtime

# Fix ownership
sudo chown $USER:$USER llm-runtime

# Docker permissions
sudo usermod -aG docker $USER
newgrp docker
```

## Configuration Issues

### Config File Not Found
**Check config file locations**:
```bash
# Current directory
ls -la llm-runtime.config.yaml

# Home directory
ls -la ~/.llm-runtime.config.yaml

# Specify config explicitly
./llm-runtime --config /path/to/config.yaml
```

### Invalid YAML Syntax
**Symptom**: Config parsing errors

**Validate YAML**:
```bash
# Check syntax with Python
python3 -c "import yaml; yaml.safe_load(open('llm-runtime.config.yaml'))"

# Or online validator
# Copy config to https://yamlvalidator.com/
```

**Common YAML issues**:
- Inconsistent indentation (use spaces, not tabs)
- Missing quotes around special characters
- Wrong list format

```yaml
# Wrong
excluded_paths: .git,*.key

# Right
excluded_paths:
  - ".git"
  - "*.key"
```

## Performance Issues

### Slow File Operations
**Large files**:
```bash
# Check file sizes
find . -name "*.go" -exec ls -lh {} \;

# Increase limits if needed
./llm-runtime --max-size 5242880  # 5MB
```

**Many files**:
```bash
# Check file count
find . -type f | wc -l

# Optimize exclusions
./llm-runtime --exclude ".git,node_modules,*.log,dist/"
```

### Slow Docker Performance
**Image optimization**:
```bash
# Pre-pull images
docker pull ubuntu:22.04

# Use smaller images
```
```yaml
commands:
  exec:
    container_image: "alpine:latest"  # Smaller but may lack tools
```

**Container reuse**:
```bash
# Commands start containers fresh each time
# This is intentional for security but affects performance
# Consider grouping commands: <exec cmd1 && cmd2 && cmd3>
```

### Slow Search Performance
**Large index**:
```bash
# Check index size
./llm-runtime --search-status

# Clean up old entries
./llm-runtime --search-cleanup

# Reduce indexed file types
```
```yaml
commands:
  search:
    index_extensions: [".go", ".py"]  # Fewer types
    max_file_size: 524288  # 512KB limit
```

## Memory and Resource Issues

### Out of Memory Errors
**Large files**:
```bash
# Reduce file size limits
./llm-runtime --max-size 524288  # 512KB

# Check available memory
free -h  # Linux
vm_stat  # macOS
```

**Docker memory limits**:
```yaml
commands:
  exec:
    memory_limit: "256m"  # Reduce from 512m
```

### High CPU Usage
**Docker CPU limits**:
```yaml
commands:
  exec:
    cpu_limit: 1  # Reduce from 2
```

**Search optimization**:
```yaml
commands:
  search:
    max_results: 5  # Reduce from 10
```

## Debug Mode

### Enable Debug Logging
```yaml
logging:
  level: "debug"
  format: "text"  # Easier to read
```

### Trace Execution
```bash
# Add timestamps to commands
./llm-runtime --verbose 2>&1 | while read line; do echo "$(date): $line"; done

# Monitor system resources
top -p $(pgrep llm-runtime)
```

## Getting Help

### Collect Debug Information
```bash
# System information
uname -a
go version
docker version
python3 --version

# Configuration
./llm-runtime --help
cat llm-runtime.config.yaml

# Recent logs
tail -20 llm-runtime.log
tail -20 audit.log
```

### Report Issues
When reporting bugs, include:
1. **Command that failed** (exact command)
2. **Error message** (complete output)
3. **Configuration** (relevant config sections)
4. **Environment** (OS, Go version, Docker version)
5. **Steps to reproduce**

### Common Error Patterns
```bash
# Search for known issues
grep "your-error" audit.log
grep "PATTERN" llm-runtime.log

# Check for resource exhaustion
dmesg | grep -i "out of memory"
```

Most issues can be resolved by checking configuration, verifying dependencies, or adjusting resource limits. When in doubt, start with `--verbose` mode for detailed diagnostic information.

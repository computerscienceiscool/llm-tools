# Troubleshooting Guide

Solutions for common issues with llm-runtime.

## Quick Diagnostics

```bash
# Check tool runs
./llm-runtime --help

# Test file reading
echo "<open README.md>" | ./llm-runtime

# Test Docker (for exec)
docker run --rm hello-world

# Test Ollama (for search)
curl http://localhost:11434/api/tags
```

---

## File Access Issues

### FILE_NOT_FOUND

**Symptom:** `=== ERROR: FILE_NOT_FOUND ===`

**Solutions:**
```bash
# Check file exists
ls -la filename

# Verify path is relative to repo root
./llm-runtime --root /path/to/repo

# Check for typos
find . -name "*partial*"
```

### PATH_SECURITY

**Symptom:** `=== ERROR: PATH_SECURITY ===`

**Common causes:**
- Directory traversal: `<open ../../../etc/passwd>`
- Excluded paths: `<open .git/config>` or `<open .env>`
- Absolute paths outside repo

**Solutions:**
```bash
# Check excluded paths
grep excluded_paths llm-runtime.config.yaml

# Use relative paths only
echo "<open src/main.go>" | ./llm-runtime
```

### RESOURCE_LIMIT

**Symptom:** File too large errors

**Solutions:**
```bash
# Check file size
ls -lh filename

# Increase limit via CLI
./llm-runtime --max-size 2097152  # 2MB

# Or in config
```
```yaml
commands:
  open:
    max_file_size: 2097152
```

---

## Exec Command Issues

### DOCKER_UNAVAILABLE

**Symptom:** `=== ERROR: DOCKER_UNAVAILABLE ===`

**Diagnostics:**
```bash
# Check Docker installed
docker --version

# Check Docker running
docker info

# Test Docker works
docker run --rm hello-world
```

**Solutions:**
```bash
# Linux: Start Docker
sudo systemctl start docker

# Linux: Fix permissions
sudo usermod -aG docker $USER
newgrp docker

# macOS: Start Docker Desktop
open -a Docker
```

### EXEC_VALIDATION

**Symptom:** `command not in whitelist`

**Solutions:**
```bash
# Check current whitelist
grep -A 20 "whitelist:" llm-runtime.config.yaml

# Add command to config
```
```yaml
commands:
  exec:
    whitelist:
      - "go test"
      - "your-command-here"
```

### EXEC_TIMEOUT

**Symptom:** Commands timing out

**Solutions:**
```bash
# Increase timeout
./llm-runtime --exec-timeout 60s
```
```yaml
commands:
  exec:
    timeout_seconds: 60
```

**Optimize slow commands:**
```bash
# Instead of full test suite
<exec go test ./...>

# Run specific tests
<exec go test -short ./...>
```

### EXEC_FAILED

**Symptom:** Non-zero exit code

**Common causes:**
- Missing dependencies in container
- Wrong working directory
- File permissions

**Debug:**
```bash
# Check what's in container
echo "<exec ls -la>" | ./llm-runtime
echo "<exec which go>" | ./llm-runtime
echo "<exec whoami>" | ./llm-runtime
```

---

## Search Issues

### Search Not Working

**Symptom:** Search returns no results or errors

**Step 1: Check Ollama is running**
```bash
# Check Ollama status
ollama list

# If not running, start it
ollama serve

# Or as systemd service (Linux)
sudo systemctl start ollama
```

**Step 2: Check embedding model**
```bash
# List models
ollama list

# Pull if missing
ollama pull nomic-embed-text
```

**Step 3: Test Ollama API**
```bash
curl http://localhost:11434/api/tags
```

**Step 4: Build/rebuild index**
```bash
./llm-runtime --reindex
```

### Ollama Connection Refused

**Symptom:** `connection refused` errors

**Solutions:**
```bash
# Start Ollama
ollama serve

# Check it's listening
curl http://localhost:11434/api/tags

# Check port isn't blocked
lsof -i :11434
```

### Index Out of Date

**Symptom:** Search missing recent files

**Solution:**
```bash
# Rebuild index
./llm-runtime --reindex
```

### Search Returns Irrelevant Results

**Try:**
- More specific search terms
- Different phrasing
- Check indexed file extensions in config

```yaml
commands:
  search:
    index_extensions:
      - ".go"
      - ".py"
      - ".js"
      - ".md"
```

---

## Build Issues

### Go Version

**Symptom:** Build fails with version errors

```bash
# Check version
go version
# Need 1.21+

# Update Go
# Download from https://go.dev/dl/
```

### Missing Dependencies

```bash
# Download deps
go mod download

# Clean and retry
go clean -modcache
go mod tidy
go build ./cmd/llm-runtime
```

### Permission Denied

```bash
# Make executable
chmod +x llm-runtime

# Docker permissions (Linux)
sudo usermod -aG docker $USER
newgrp docker
```

---

## Configuration Issues

### Config Not Found

The tool looks for config in:
1. `./llm-runtime.config.yaml` (current directory)
2. `~/.llm-runtime.config.yaml` (home directory)

```bash
# Check config exists
ls -la llm-runtime.config.yaml

# Specify config explicitly
./llm-runtime --config /path/to/config.yaml
```

### Invalid YAML

**Validate syntax:**
```bash
# Using Python
python3 -c "import yaml; yaml.safe_load(open('llm-runtime.config.yaml'))"
```

**Common YAML mistakes:**
```yaml
# WRONG - no quotes around special chars
excluded_paths: .git,*.key

# RIGHT - use list format
excluded_paths:
  - ".git"
  - "*.key"

# WRONG - tabs
commands:
	exec:  # <- tab character

# RIGHT - spaces only
commands:
  exec:   # <- spaces
```

---

## Performance Issues

### Slow Docker Startup

**Pre-pull the image:**
```bash
docker pull ubuntu:22.04
```

**Combine commands:**
```bash
# Instead of multiple exec calls
<exec go build && go test && go vet>
```

### Slow Search

**Reduce index size:**
```yaml
commands:
  search:
    index_extensions: [".go", ".py"]  # Fewer types
    max_results: 5                     # Fewer results
```

### High Memory Usage

**Reduce limits:**
```yaml
commands:
  open:
    max_file_size: 524288  # 512KB
  exec:
    memory_limit: "256m"
```

---

## Error Reference

| Error | Cause | Solution |
|-------|-------|----------|
| `FILE_NOT_FOUND` | File doesn't exist | Check path |
| `PATH_SECURITY` | Path outside repo or excluded | Use relative paths |
| `RESOURCE_LIMIT` | File too large | Increase limit |
| `EXEC_VALIDATION` | Command not whitelisted | Add to whitelist |
| `DOCKER_UNAVAILABLE` | Docker not running | Start Docker |
| `EXEC_TIMEOUT` | Command took too long | Increase timeout |
| `EXEC_FAILED` | Non-zero exit | Check command output |

---

## Getting Help

### Collect Debug Info

```bash
# System info
uname -a
go version
docker version
ollama --version

# Config
cat llm-runtime.config.yaml

# Test commands
echo "<open README.md>" | ./llm-runtime --verbose
```

### Enable Verbose Mode

```bash
./llm-runtime --verbose
```

### Check Logs

```bash
tail -f llm-runtime.log
tail -f audit.log
```

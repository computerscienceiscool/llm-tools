# Quick Reference Guide

## Command Syntax

### File Reading
```
<open filepath>
```
Examples:
- `<open README.md>`
- `<open src/main.go>`
- `<open config/settings.yaml>`

### File Writing
```
<write filepath>
content goes here
</write>
```
Examples:
- `<write config.json>{"port": 8080}</write>`
- `<write src/util.go>package util</write>`

### Command Execution
```
<exec command arguments>
```
Examples:
- `<exec go test>`
- `<exec npm build>`
- `<exec make clean>`

### Semantic Search
```
<search query terms>
```
Examples:
- `<search authentication logic>`
- `<search error handling>`
- `<search database connection>`

---

## CLI Usage

### Basic
```bash
# Pipe mode
echo "<open README.md>" | ./llm-runtime

# Interactive mode
./llm-runtime --interactive

# With custom timeout
./llm-runtime --exec-timeout 60s
```

### Common Flags
```bash
--root /path/to/repo      # Set repository root
--interactive             # Interactive mode
--reindex                 # Rebuild search index
--verbose                 # Detailed output
--config file.yaml        # Custom config file
--exec-timeout 60s        # Exec timeout
--exec-memory 1g          # Exec memory limit
```

---

## Setup Checklist

### Minimum (file operations)
```bash
git clone https://github.com/computerscienceiscool/llm-runtime.git
cd llm-runtime
make build
echo "<open README.md>" | ./llm-runtime
```

### With Exec (Docker)
```bash
# Install Docker
docker run hello-world

# Test exec
echo "<exec echo hello>" | ./llm-runtime
```

### With Search (Ollama)
```bash
# Install Ollama
curl -fsSL https://ollama.com/install.sh | sh
ollama pull nomic-embed-text

# Build index and test
./llm-runtime --reindex
echo "<search main function>" | ./llm-runtime
```

---

## Default Whitelisted Commands

### Go
```
go test, go build, go run, go mod tidy, go vet, go fmt
```

### Node.js
```
npm test, npm run build, npm install, node
```

### Python
```
python, python3, python -m pytest, pip install
```

### Build Tools
```
make, make test, make build, make clean
```

### System
```
ls, cat, grep, find, head, tail, wc, echo
```

---

## Common Workflows

### Explore a Project
```
<open README.md>
<open go.mod>
<search main entry point>
<open cmd/main.go>
```

### Code Review
```
<search authentication>
<open src/auth/middleware.go>
<exec go test ./src/auth/...>
```

### Make Changes
```
<open src/util.go>
<write src/util.go>
// updated content
</write>
<exec go test>
```

### Find and Fix
```
<search error handling>
<open src/errors.go>
<write src/errors.go>
// fixed code
</write>
<exec go build>
```

---

## Configuration

### Minimal Config
```yaml
repository:
  root: "."
  excluded_paths: [".git", ".env", "*.key"]

commands:
  open:
    enabled: true
  write:
    enabled: true
  exec:
    timeout_seconds: 30
    whitelist:
      - "go test"
      - "npm test"
  search:
    enabled: true
```

### Read-Only Config
```yaml
commands:
  open:
    enabled: true
  write:
    enabled: false
  exec:
    whitelist: []  # No exec commands
  search:
    enabled: true
```

---

## Error Quick Fixes

| Error | Fix |
|-------|-----|
| `FILE_NOT_FOUND` | Check path exists |
| `PATH_SECURITY` | Use relative paths |
| `EXEC_VALIDATION` | Add command to whitelist |
| `DOCKER_UNAVAILABLE` | Start Docker |
| `EXEC_TIMEOUT` | Increase timeout or optimize command |
| Search not working | Run `ollama serve` then `--reindex` |

---

## File Locations

| File | Purpose |
|------|---------|
| `llm-runtime.config.yaml` | Main configuration |
| `embeddings.db` | Search index (SQLite) |
| `audit.log` | Operation log |

---

## Tips

1. **Start with open** — Read README.md first to understand any project
2. **Search before writing** — Check if similar code exists
3. **Test after changes** — Use exec to verify modifications
4. **Combine exec commands** — `<exec go build && go test>`
5. **Use specific searches** — "JWT token validation" beats "auth"

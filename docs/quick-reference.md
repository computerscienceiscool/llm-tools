# Quick Reference Guide

## Command Syntax

### File Reading
```
<open filepath>
```
**Examples:**
- `<open README.md>`
- `<open src/main.go>`
- `<open config/database.yaml>`

### File Writing
```
<write filepath>
content goes here
</write>
```
**Examples:**
- `<write config.json>{"port": 8080}</write>`
- `<write src/new.go>package main\n\nfunc main() {}</write>`

### Command Execution
```
<exec command arguments>
```
**Examples:**
- `<exec go test>`
- `<exec npm build>`
- `<exec make clean>`

### Semantic Search
```
<search query terms>
```
**Examples:**
- `<search authentication logic>`
- `<search error handling>`
- `<search database connection>`

## Command Line Usage

### Basic Usage
```bash
# Pipe mode (most common)
echo "Commands here <open file>" | ./llm-runtime

# Interactive mode
./llm-runtime --interactive

# File input/output
./llm-runtime --input input.txt --output output.txt
```

### Enable Features
```bash
# Enable exec commands
./llm-runtime --exec-enabled

# Enable search (requires Python setup)
./llm-runtime --search-enabled

# Enable all features
./llm-runtime --exec-enabled --search-enabled
```

## Common Flags

### File Operations
```bash
--root /path/to/repo           # Set repository root
--max-size 2097152            # Max file read size (2MB)
--max-write-size 204800       # Max file write size (200KB)
--exclude ".git,*.key,secrets/" # Excluded paths
```

### Exec Commands
```bash
--exec-enabled                # Enable exec functionality
--exec-timeout 60s           # Command timeout
--exec-memory 1g             # Container memory limit
--exec-cpu 4                 # Container CPU limit
--exec-whitelist "go test,npm build" # Allowed commands
```

### General
```bash
--verbose                    # Detailed output
--config custom.yaml         # Custom config file
--help                       # Show all options
```

## Quick Setup

### 1. Build Tool
```bash
git clone https://github.com/computerscienceiscool/llm-runtime.git
cd llm-runtime
make build
```

### 2. Test Basic Functionality
```bash
echo "Test file access <open README.md>" | ./llm-runtime
```

### 3. Enable Docker (for exec)
```bash
# Test Docker
docker run hello-world

# Enable exec commands
echo "Test exec <exec echo 'Hello'>" | ./llm-runtime --exec-enabled
```

### 4. Enable Search (optional)
```bash
# Install Python dependencies
pip install sentence-transformers

# Build search index
./llm-runtime --reindex

# Test search
echo "Test search <search configuration>" | ./llm-runtime
```

## Default Whitelisted Commands

### Go
```
go test, go build, go run, go mod tidy, go vet, go fmt
```

### Node.js
```
npm test, npm run build, npm install, node, npm start
```

### Python
```
python, python3, python -m pytest, pip install
```

### Build Tools
```
make, make test, make build, make clean
```

### Rust
```
cargo build, cargo test, cargo run, cargo check
```

### System
```
ls, cat, grep, find, head, tail, wc, echo
```

## Common Workflows

### Project Exploration
```
<open README.md>
<open package.json>
<search main entry point>
<open src/main.js>
<exec npm test>
```

### Code Review
```
<search authentication>
<open src/auth/middleware.js>
<exec npm test auth>
<search error handling>
<open src/utils/errors.js>
```

### Feature Development
```
<search similar feature>
<open existing/feature.js>
<write new/feature.js>
// implementation
</write>
<write tests/feature.test.js>
// tests
</write>
<exec npm test tests/feature.test.js>
```

### Bug Investigation
```
<search error message>
<open problematic/file.js>
<exec npm test -- --grep "failing test">
<search similar patterns>
```

## Error Codes Quick Reference

### File Access Errors
- **FILE_NOT_FOUND** - File doesn't exist
- **PATH_SECURITY** - Path outside repository or excluded
- **RESOURCE_LIMIT** - File too large
- **PERMISSION_DENIED** - Cannot access file

### Exec Errors
- **EXEC_VALIDATION** - Command not whitelisted
- **DOCKER_UNAVAILABLE** - Docker not installed/running
- **EXEC_TIMEOUT** - Command took too long
- **EXEC_FAILED** - Command returned non-zero exit code

### Search Errors
- **SEARCH_DISABLED** - Search not enabled in config
- **SEARCH_INIT_FAILED** - Python dependencies missing
- **INDEX_NOT_FOUND** - Run `--reindex` first

## Configuration Quick Setup

### Minimal Config (llm-runtime.config.yaml)
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
    enabled: true
    whitelist: ["go test", "npm build", "make"]
  search:
    enabled: true
```

### Security-First Config
```yaml
repository:
  excluded_paths: 
    - ".git"
    - ".env*"
    - "*.key"
    - "*.pem"
    - "secrets/"

commands:
  open:
    max_file_size: 1048576  # 1MB
  write:
    enabled: false  # Read-only mode
  exec:
    timeout_seconds: 30
    memory_limit: "256m"
    whitelist: ["go test", "go build"]

security:
  log_all_operations: true
```

### Development Config
```yaml
commands:
  open:
    max_file_size: 2097152  # 2MB
  write:
    backup_before_write: true
  exec:
    timeout_seconds: 60
    memory_limit: "1g"
    whitelist: 
      - "go test"
      - "npm test"
      - "python -m pytest"
      - "make"
  search:
    max_results: 15

logging:
  level: "debug"
```

## Troubleshooting Quick Fixes

### Docker Issues
```bash
# Start Docker
sudo systemctl start docker  # Linux
open -a Docker              # macOS

# Fix permissions
sudo usermod -aG docker $USER
newgrp docker

# Test Docker
docker run --rm hello-world
```

### Python/Search Issues
```bash
# Install dependencies
pip install sentence-transformers

# Build index
./llm-runtime --reindex

# Check setup
./llm-runtime --check-python-setup
```

### Permission Issues
```bash
# Make executable
chmod +x llm-runtime

# Check repository access
ls -la target/file
```

### File Size Issues
```bash
# Check file size
ls -lh large-file.txt

# Increase limits
./llm-runtime --max-size 5242880  # 5MB
```

## Performance Tips

### Optimize File Operations
- Use search to find files before reading
- Read only necessary files
- Combine multiple operations in one LLM response

### Optimize Exec Commands
- Pre-pull Docker images: `docker pull ubuntu:22.04`
- Combine commands: `<exec cmd1 && cmd2 && cmd3>`
- Use specific commands: `go test ./pkg` vs `go test ./...`

### Optimize Search
- Use specific queries for better results
- Update index incrementally: `--search-update`
- Clean up periodically: `--search-cleanup`

## Integration Examples

### Claude System Prompt
```
You have access to a file system tool. Use these commands:
- <open filepath> - Read files
- <write filepath>content</write> - Create/update files
- <exec command> - Run commands (Docker isolated, no network)
- <search query> - Semantic search for relevant files

Start by reading README.md, then explore systematically.
Always test changes with exec commands.
```

### ChatGPT System Prompt
```
You can interact with a local repository using these commands:
<open file> - Read file contents
<write file>content</write> - Create/modify files  
<exec command> - Execute commands safely
<search terms> - Find relevant files

Explore the codebase systematically and verify changes by running tests.
```

## Common Use Cases

### Code Analysis
1. `<open README.md>` - Understand project
2. `<search main entry>` - Find starting point
3. `<open entry/file>` - Examine code
4. `<exec tests>` - Verify functionality

### Bug Fixing
1. `<search error message>` - Find related code
2. `<open problematic/file>` - Examine issue
3. `<exec reproduce command>` - Reproduce bug
4. `<write fixed/file>fix</write>` - Apply fix
5. `<exec test command>` - Verify fix

### Feature Addition
1. `<search similar feature>` - Find patterns
2. `<open existing/implementation>` - Study approach
3. `<write new/feature>code</write>` - Implement
4. `<write test/file>tests</write>` - Add tests
5. `<exec test new feature>` - Validate

This quick reference covers the most common usage patterns. For detailed information, see the full documentation guides.

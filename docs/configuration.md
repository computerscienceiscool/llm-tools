# Configuration Guide

Complete reference for configuring the LLM File Access Tool via `llm-runtime.config.yaml` and command-line options.

## Configuration File Location

The tool looks for configuration files in this order:
1. `./llm-runtime.config.yaml` (current directory)
2. `~/.llm-runtime.config.yaml` (home directory)
3. Built-in defaults

## Complete Configuration Reference

### Basic Configuration Structure
```yaml
repository:
  root: "."
  excluded_paths:
    - ".git"
    - ".env"
    - "*.key"

commands:
  open:
    enabled: true
    max_file_size: 1048576
    
  write:
    enabled: true
    max_file_size: 102400
    
  exec:
    enabled: false
    timeout_seconds: 30
    
  search:
    enabled: false
    vector_db_path: "./embeddings.db"

security:
  log_all_operations: true
  audit_log_path: "./audit.log"
```

## Repository Settings

### `repository.root`
**Default**: `"."`  
**Description**: Root directory of the repository to explore  
**Example**: 
```yaml
repository:
  root: "/path/to/project"
```

### `repository.excluded_paths`
**Default**: `[".git", ".env", "*.key", "*.pem"]`  
**Description**: Paths and patterns blocked from access  
**Examples**:
```yaml
repository:
  excluded_paths:
    - ".git"              # Git directory
    - ".env*"             # All env files
    - "*.key"             # Private keys
    - "*.pem"             # Certificates
    - "node_modules"      # Dependencies
    - "secrets/"          # Secrets folder
    - "__pycache__"       # Python cache
    - "*.sqlite"          # Database files
```

## Open Command Configuration

### `commands.open.enabled`
**Default**: `true`  
**Description**: Enable/disable file reading  

### `commands.open.max_file_size`
**Default**: `1048576` (1MB)  
**Description**: Maximum file size for reading in bytes  
```yaml
commands:
  open:
    max_file_size: 2097152  # 2MB
```

### `commands.open.allowed_extensions`
**Default**: `[".go", ".py", ".js", ".md", ".txt", ".json", ".yaml"]`  
**Description**: File extensions allowed for reading  
```yaml
commands:
  open:
    allowed_extensions:
      - ".go"
      - ".py"
      - ".js"
      - ".ts"
      - ".md"
      - ".txt"
      - ".json"
      - ".yaml"
      - ".yml"
      - ".toml"
```

## Write Command Configuration

### `commands.write.enabled`
**Default**: `true`  
**Description**: Enable/disable file writing  

### `commands.write.max_file_size`
**Default**: `102400` (100KB)  
**Description**: Maximum file size for writing in bytes  

### `commands.write.backup_before_write`
**Default**: `true`  
**Description**: Create backup before overwriting files  

### `commands.write.allowed_extensions`
**Description**: Restrict write operations to specific file types  
```yaml
commands:
  write:
    allowed_extensions:
      - ".go"
      - ".py"
      - ".js"
      - ".md"
      - ".txt"
      - ".json"
      - ".yaml"
```

## Exec Command Configuration

### `commands.exec.enabled`
**Default**: `false`  
**Description**: Enable/disable command execution (requires Docker)  

### `commands.exec.container_image`
**Default**: `"ubuntu:22.04"`  
**Description**: Docker image for command execution  
```yaml
commands:
  exec:
    container_image: "node:18-alpine"  # For Node.js projects
    # OR
    container_image: "golang:1.21"     # For Go projects
    # OR
    container_image: "python:3.11"     # For Python projects
```

### `commands.exec.timeout_seconds`
**Default**: `30`  
**Description**: Maximum execution time in seconds  

### `commands.exec.memory_limit`
**Default**: `"512m"`  
**Description**: Memory limit for containers  
**Options**: `"256m"`, `"512m"`, `"1g"`, `"2g"`

### `commands.exec.cpu_limit`
**Default**: `2`  
**Description**: CPU cores limit for containers  

### `commands.exec.network_enabled`
**Default**: `false`  
**Description**: Allow network access in containers (NOT recommended)  

### `commands.exec.whitelist`
**Default**: Go, Node.js, Python, Make, System commands  
**Description**: Commands allowed for execution  
```yaml
commands:
  exec:
    whitelist:
      # Go commands
      - "go test"
      - "go build"
      - "go run"
      - "go mod tidy"
      
      # Node.js commands  
      - "npm test"
      - "npm run build"
      - "npm install"
      - "node"
      
      # Python commands
      - "python"
      - "python3"
      - "python -m pytest"
      - "pip install"
      
      # Build tools
      - "make"
      - "make test"
      - "make build"
      
      # Rust commands
      - "cargo build"
      - "cargo test"
      - "cargo run"
      
      # System commands
      - "ls"
      - "cat"
      - "grep"
      - "find"
      - "head"
      - "tail"
      - "wc"
      - "echo"
```

## Search Command Configuration

### `commands.search.enabled`
**Default**: `false`  
**Description**: Enable semantic search (requires Python dependencies)  

### `commands.search.vector_db_path`
**Default**: `"./embeddings.db"`  
**Description**: SQLite database path for storing embeddings  

### `commands.search.embedding_model`
**Default**: `"all-MiniLM-L6-v2"`  
**Description**: Sentence transformer model for embeddings  
**Options**: 
- `"all-MiniLM-L6-v2"` (fast, 384 dimensions)
- `"all-mpnet-base-v2"` (slower, better quality, 768 dimensions)

### `commands.search.max_results`
**Default**: `10`  
**Description**: Maximum search results to return  

### `commands.search.min_similarity_score`
**Default**: `0.5`  
**Description**: Minimum similarity score (0.0-1.0) to include results  

### `commands.search.max_preview_length`
**Default**: `100`  
**Description**: Maximum length of file preview in search results  

### `commands.search.chunk_size`
**Default**: `1000`  
**Description**: Text chunk size for processing large files  

### `commands.search.python_path`
**Default**: `"python3"`  
**Description**: Python interpreter path  
**Examples**: `"python"`, `"python3"`, `"/usr/bin/python3"`

### `commands.search.index_extensions`
**Default**: `[".go", ".py", ".js", ".md", ".txt", ".yaml", ".json"]`  
**Description**: File extensions to include in search index  
```yaml
commands:
  search:
    index_extensions:
      - ".go"
      - ".py"
      - ".js"
      - ".ts"
      - ".jsx"
      - ".tsx"
      - ".md"
      - ".txt"
      - ".yaml"
      - ".yml"
      - ".json"
      - ".toml"
      - ".rs"
      - ".java"
      - ".c"
      - ".cpp"
      - ".h"
      - ".hpp"
```

### `commands.search.max_file_size`
**Default**: `1048576` (1MB)  
**Description**: Maximum file size to index for search  

## Security Configuration

### `security.rate_limit_per_minute`
**Default**: `100`  
**Description**: Maximum operations per minute  

### `security.log_all_operations`
**Default**: `true`  
**Description**: Enable comprehensive audit logging  

### `security.audit_log_path`
**Default**: `"./audit.log"`  
**Description**: Path for audit log file  

### `security.follow_symlinks`
**Default**: `true`  
**Description**: Whether to follow symbolic links  

### `security.allow_hidden_files`
**Default**: `false`  
**Description**: Allow access to hidden files (starting with .)  

## Output Configuration

### `output.show_summaries`
**Default**: `true`  
**Description**: Show command execution summaries  

### `output.show_execution_time`
**Default**: `true`  
**Description**: Display execution time for operations  

### `output.truncate_large_outputs`
**Default**: `true`  
**Description**: Truncate very large command outputs  

### `output.max_output_lines`
**Default**: `1000`  
**Description**: Maximum lines to show in command output  

## Logging Configuration

### `logging.level`
**Default**: `"info"`  
**Options**: `"debug"`, `"info"`, `"warn"`, `"error"`  
**Description**: Logging verbosity level  

### `logging.file`
**Default**: `"./llm-runtime.log"`  
**Description**: Log file path  

### `logging.format`
**Default**: `"json"`  
**Options**: `"json"`, `"text"`  
**Description**: Log format  

## Example Configurations

### Development Environment
```yaml
repository:
  root: "."
  excluded_paths: [".git", "node_modules", ".env"]

commands:
  open:
    enabled: true
    max_file_size: 2097152  # 2MB
  write:
    enabled: true
    backup_before_write: true
  exec:
    enabled: true
    timeout_seconds: 60
    whitelist: ["go test", "npm test", "make", "python3"]
  search:
    enabled: true

security:
  log_all_operations: true

logging:
  level: "debug"
```

### Production Environment
```yaml
repository:
  root: "/app/src"
  excluded_paths: 
    - ".git"
    - ".env*"
    - "*.key"
    - "*.pem"
    - "secrets/"
    - "credentials/"

commands:
  open:
    enabled: true
    max_file_size: 1048576  # 1MB
  write:
    enabled: false  # Disable writes in production
  exec:
    enabled: true
    timeout_seconds: 30
    memory_limit: "256m"
    whitelist: ["go test", "go build"]
  search:
    enabled: true

security:
  rate_limit_per_minute: 50
  log_all_operations: true

logging:
  level: "info"
  format: "json"
```

### Search-Optimized Setup
```yaml
commands:
  search:
    enabled: true
    embedding_model: "all-mpnet-base-v2"  # Better quality
    max_results: 20
    min_similarity_score: 0.3  # More permissive
    index_extensions:
      - ".go"
      - ".py"
      - ".js"
      - ".ts"
      - ".md"
      - ".txt"
      - ".yaml"
      - ".json"
      - ".rs"
      - ".java"
    max_file_size: 2097152  # 2MB
```

## Command Line Override

All configuration options can be overridden via command line:

```bash
# Override repository root
./llm-runtime --root /path/to/project

# Override file size limits
./llm-runtime --max-size 2097152 --max-write-size 204800

# Enable exec with custom settings
./llm-runtime --exec-enabled --exec-timeout 60s --exec-memory 1g

# Custom excluded paths
./llm-runtime --exclude ".git,node_modules,*.secret"

# Verbose logging
./llm-runtime --verbose
```

## Configuration Validation

Test your configuration:
```bash
# Check current configuration
./llm-runtime --help

# Validate Docker setup (if exec enabled)
./llm-runtime --check-docker

# Validate Python setup (if search enabled)  
./llm-runtime --check-python-setup

# Test with specific config file
./llm-runtime --config my-config.yaml
```

## Best Practices

1. **Start minimal** - Enable features as needed
2. **Test thoroughly** - Validate each feature after enabling
3. **Monitor logs** - Keep audit logging enabled
4. **Secure by default** - Keep network disabled for exec
5. **Backup configs** - Version control your configuration
6. **Environment-specific** - Use different configs for dev/prod

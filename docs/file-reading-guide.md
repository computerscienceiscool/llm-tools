# File Reading Guide - `<read>` Command

## Overview

The `<read>` command allows LLMs to read file contents from the repository in a secure, containerized manner. Files are accessed through isolated Docker containers to ensure read operations cannot compromise the host system.

**Security**: All file reads are executed in minimal Alpine Linux containers with read-only access to the repository, providing defense-in-depth security even for simple read operations.

## How It Works

When an LLM includes a `<read>` command, the tool:
1. **Validates the path** - Ensures path is within repository bounds
2. **Creates I/O container** - Spins up minimal Alpine container (default: llm-runtime-io:latest)
3. **Mounts repository read-only** - Repository mounted at `/workspace`
4. **Reads file via cat** - Executes `cat` command in container
5. **Returns content** - File contents returned to LLM
6. **Cleans up** - Container automatically removed

## Basic Syntax

```
<read path/to/file.ext>
```

**Examples:**
- `<read main.go>` - Read Go source file
- `<read README.md>` - Read documentation
- `<read config/settings.yaml>` - Read configuration
- `<read .gitignore>` - Read hidden files
- `<read src/components/Button.tsx>` - Read nested files

## Common Use Cases

### **Understanding Codebase**
```
Let me examine the main entry point:

<read cmd/server/main.go>

And check the configuration structure:

<read internal/config/config.go>

Now let me look at the API handlers:

<read internal/handlers/api.go>
```

### **Reviewing Configuration**
```
Let me check the application configuration:

<read config/app.yaml>

And the database settings:

<read config/database.yaml>

Also the environment template:

<read .env.example>
```

### **Analyzing Dependencies**
```
Let me check the Go module dependencies:

<read go.mod>

And see the locked versions:

<read go.sum>

For Node projects:

<read package.json>

<read package-lock.json>
```

### **Documentation Review**
```
Let me read the project README:

<read README.md>

Check the contributing guidelines:

<read CONTRIBUTING.md>

And review the changelog:

<read CHANGELOG.md>
```

### **Test Analysis**
```
Let me examine the test files:

<read internal/auth/auth_test.go>

<read test/integration/api_test.go>

<read test/fixtures/sample_data.json>
```

### **Build & CI Configuration**
```
Let me check the Makefile:

<read Makefile>

Review the CI pipeline:

<read .github/workflows/ci.yml>

And Docker configuration:

<read Dockerfile>

<read docker-compose.yml>
```

## Output Format

### **Successful Read**
```
=== READ SUCCESSFUL: config/settings.yaml ===
server:
  port: 8080
  host: localhost
  
database:
  connection: postgres://localhost:5432/myapp
  max_connections: 100
  
logging:
  level: info
  format: json
=== END READ ===
```

### **File Not Found**
```
=== ERROR: READ_FAILED ===
Message: READ_FAILED: file not found: nonexistent.txt
Path: <read nonexistent.txt>
=== END ERROR ===
```

### **Path Outside Repository**
```
=== ERROR: READ_VALIDATION ===
Message: READ_VALIDATION: path outside repository: ../../../etc/passwd
Path: <read ../../../etc/passwd>
=== END ERROR ===
```

## Security Model

### **Containerized Reads**
All file reads execute in isolated containers:

```bash
docker run \
    --rm \
    --network none \
    --user 1000:1000 \
    --cap-drop ALL \
    --security-opt no-new-privileges \
    --read-only \
    --memory 128m \
    --cpus 1 \
    -v /repo:/workspace:ro \
    llm-runtime-io:latest \
    cat /workspace/file.txt
```

**Security Features:**
- **No network access**: Container completely isolated
- **Read-only mount**: Cannot modify repository files
- **Minimal image**: Alpine Linux (~5MB) reduces attack surface
- **Resource limits**: 128MB RAM, 1 CPU core, 10-second timeout
- **Non-root user**: Runs as unprivileged user (1000:1000)
- **No new privileges**: Cannot escalate permissions

### **Path Validation**
```
✅ Allowed:
<read main.go>
<read src/components/App.tsx>
<read .github/workflows/ci.yml>
<read internal/../../README.md>  # resolves to README.md

❌ Blocked:
<read ../../../etc/passwd>        # outside repository
<read /etc/hosts>                 # absolute path outside repo
<read ~/.ssh/id_rsa>              # home directory access
```

### **Defense in Depth**
Even though reads are "safe" operations, containerization provides:
- **Exploit mitigation**: Vulnerabilities in cat/shell cannot escape container
- **Resource isolation**: Runaway reads cannot consume host resources  
- **Audit trail**: All reads logged with container execution details
- **Consistent security**: Same model as write/exec operations

## Error Types

### **READ_VALIDATION**
```
<read ../outside/repository.txt>
```
**Cause**: Path points outside repository boundaries
**Solution**: Use paths relative to repository root

### **READ_FAILED**
```
<read nonexistent_file.txt>
```
**Cause**: File does not exist at specified path
**Solution**: Verify file path and name

### **DOCKER_UNAVAILABLE**
```
<read config.yaml>
```
**Cause**: Docker not running or I/O container image missing
**Solution**: Ensure Docker running and build I/O image

### **IO_TIMEOUT**
```
<read very_large_file.bin>
```
**Cause**: Read operation exceeded timeout (default: 10 seconds)
**Solution**: Increase timeout or avoid reading extremely large files

## Configuration

### **Command Line Options**
```bash
# Run with custom I/O settings
./llm-runtime --io-timeout 30s \
              --io-memory 256m \
              --io-cpu 2
```

### **Configuration File**
```yaml
commands:
  io:
    # I/O containerization settings for read/write operations
    container_image: "llm-runtime-io:latest"  # Minimal Alpine image
    timeout_seconds: 10
    memory_limit: "128m"
    cpu_limit: 1
    fallback_image: "alpine:latest"  # If custom image unavailable
```

### **Custom I/O Container**
```bash
# Build the default I/O container image
make build-io-image

# Or use standard Alpine
./llm-runtime --io-container alpine:latest
```

## File Size Considerations

### **Small Files (< 1MB)**
```
<read config.yaml>
<read main.go>
<read package.json>
```
**Performance**: Instant (< 100ms including container overhead)

### **Medium Files (1-10MB)**
```
<read data/large_config.json>
<read logs/application.log>
```
**Performance**: Fast (< 1 second)
**Consideration**: May hit LLM context limits

### **Large Files (> 10MB)**
```
<read database/dump.sql>
<read dist/bundle.js>
```
**Performance**: Slower (several seconds)
**Consideration**: Will likely exceed LLM context window
**Solution**: Use grep/head/tail to extract relevant sections

## Best Practices

### **Read Only What's Needed**
```
# ❌ Avoid reading entire files unnecessarily
<read large_log_file.log>

# ✅ Better: Use exec to filter first
<exec head -n 100 large_log_file.log>
<exec grep "ERROR" application.log>
<exec tail -n 50 debug.log>
```

### **Check File Existence**
```
Let me verify the file exists first:

<exec ls config/settings.yaml>

Now read it:

<read config/settings.yaml>
```

### **Progressive Reading**
```
Let me start with the README:

<read README.md>

Based on that, let me examine the main code:

<read cmd/server/main.go>

Now I'll look at specific modules mentioned:

<read internal/auth/auth.go>
```

### **Combine with Search**
```
Let me search for authentication logic:

<search authentication>

Based on the results, let me read the relevant file:

<read internal/auth/handler.go>
```

## Advanced Usage

### **Reading Multiple Related Files**
```
Let me examine the authentication system:

<read internal/auth/auth.go>

<read internal/auth/middleware.go>

<read internal/auth/token.go>

<read internal/auth/auth_test.go>

Now I understand the complete authentication flow.
```

### **Configuration Review**
```
Let me review all configuration files:

<read config/app.yaml>

<read config/database.yaml>

<read config/logging.yaml>

<read .env.example>

These settings look properly configured.
```

### **Dependency Analysis**
```
For a Go project:

<read go.mod>

<read go.sum>

For a Node project:

<read package.json>

<read package-lock.json>

For a Python project:

<read requirements.txt>

<read setup.py>
```

### **Documentation Gathering**
```
<read README.md>

<read docs/architecture.md>

<read docs/api.md>

<read CONTRIBUTING.md>

<read LICENSE>
```

## Reading Special Files

### **Hidden Files**
```
<read .gitignore>

<read .env.example>

<read .dockerignore>

<read .github/workflows/ci.yml>
```

### **Configuration Files**
```
# YAML
<read config.yaml>
<read .gitlab-ci.yml>

# JSON
<read package.json>
<read tsconfig.json>

# TOML
<read Cargo.toml>
<read pyproject.toml>

# INI
<read .editorconfig>
<read setup.cfg>
```

### **Build Files**
```
<read Makefile>

<read Dockerfile>

<read docker-compose.yml>

<read CMakeLists.txt>
```

### **Test Files**
```
<read test/integration_test.go>

<read tests/test_auth.py>

<read spec/models/user_spec.rb>

<read __tests__/component.test.tsx>
```

## Troubleshooting

### **File Not Found**
```bash
# Check file exists
ls -la path/to/file

# Check exact filename (case-sensitive)
find . -name "filename.ext"

# List directory contents
ls -la directory/
```

### **Path Issues**
```
# ❌ Wrong
<read /absolute/path/file.txt>

# ✅ Correct - relative to repository root
<read relative/path/file.txt>

# ❌ Wrong
<read ~/home/file.txt>

# ✅ Correct
<read file.txt>
```

### **Container Issues**
```bash
# Build I/O container image
make build-io-image

# Or pull Alpine fallback
docker pull alpine:latest

# Verify image exists
docker images | grep llm-runtime-io

# Test container manually
docker run --rm llm-runtime-io:latest cat /etc/os-release
```

### **Permission Issues**
```bash
# Check Docker daemon
systemctl status docker

# Ensure user in docker group
groups $USER | grep docker

# Test Docker access
docker run --rm alpine:latest echo "Hello"
```

### **Large File Issues**
```
# Instead of reading entire file
<read large_file.log>

# Use exec to extract relevant parts
<exec head -n 100 large_file.log>

<exec grep "ERROR" large_file.log | head -n 50>

<exec tail -n 200 large_file.log>
```

## Performance Optimization

### **Container Caching**
```bash
# Pre-pull I/O image to avoid delays
docker pull llm-runtime-io:latest

# Or build custom image
make build-io-image

# Verify image available
docker image inspect llm-runtime-io:latest
```

### **Read Patterns**
```
# ❌ Inefficient - many separate reads
<read file1.go>
<read file2.go>
<read file3.go>
<read file4.go>
<read file5.go>

# ✅ Better - batch related files in context
Let me examine the authentication module:

<read internal/auth/auth.go>
<read internal/auth/middleware.go>
<read internal/auth/token.go>

Now I'll look at the handlers...
```

### **Selective Reading**
```
# ❌ Reading unnecessary files
<read vendor/third_party/huge_lib.go>

# ✅ Focus on relevant project files
<read internal/core/business_logic.go>
```

## Integration with Other Commands

### **Read → Analyze → Write**
```
<read config/settings.yaml>

I see the port is set to 8080. Let me update it:

<write config/settings.yaml>
server:
  port: 3000
  host: localhost
</write>
```

### **Read → Test**
```
<read internal/auth/auth.go>

Let me verify this works:

<exec go test ./internal/auth>
```

### **Search → Read**
```
<search database connection>

Based on the search, let me read the database config:

<read internal/database/connection.go>
```

### **Read → Exec → Verify**
```
<read go.mod>

I see you're using Go 1.21. Let me verify:

<exec go version>

<exec go mod verify>
```

## Audit Trail

All read operations are logged:

```
2025-12-15T10:30:45Z|session:abc123|read|config/settings.yaml|success|size:1234
2025-12-15T10:30:46Z|session:abc123|read|internal/auth/auth.go|success|size:5678
2025-12-15T10:30:47Z|session:abc123|read|nonexistent.txt|failed|error:file_not_found
```

**Logged Information:**
- Timestamp
- Session ID  
- Operation type (read)
- File path
- Success/failure status
- File size (on success) or error type (on failure)
- Container execution details

## Why Containerize Reads?

**Security Benefits:**
- **Exploit mitigation**: Bugs in file reading cannot escape container
- **Resource isolation**: Cannot consume unlimited host resources
- **Consistent model**: Same security approach as write/exec
- **Defense in depth**: Multiple layers prevent compromise
- **Audit capability**: Complete visibility into file access

**Trade-offs:**
- **Slight overhead**: ~50-100ms per read (vs direct file access)
- **Docker dependency**: Requires Docker for all operations
- **Image requirement**: Needs I/O container image available

**Why It's Worth It:**
The minimal overhead provides significant security benefits. Even simple file reads are isolated from the host system, preventing entire classes of potential exploits.

## Container Image Details

### **Default Image: llm-runtime-io:latest**
```dockerfile
FROM alpine:latest
RUN apk add --no-cache coreutils
USER 1000:1000
```

**Size**: ~5MB
**Tools**: cat, basic shell utilities
**Purpose**: Minimal attack surface for I/O operations

### **Fallback: alpine:latest**
If custom image unavailable, falls back to standard Alpine.

### **Building Custom Image**
```bash
# Build from Makefile
make build-io-image

# Or manually
docker build -t llm-runtime-io:latest -f Dockerfile.io .

# Verify
docker images | grep llm-runtime-io
```

The `<read>` command provides LLMs with secure, containerized access to repository files, enabling thorough code analysis and understanding while maintaining complete isolation from the host system.

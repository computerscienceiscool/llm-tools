# File Reading Guide - `<open>` Command

## Overview

The `<open>` command allows LLMs to read file contents from the repository in a secure, containerized manner. Files are accessed through isolated Docker containers to ensure read operations cannot compromise the host system.

**Security**: All file reads are executed in minimal Alpine Linux containers with read-only access to the repository, providing defense-in-depth security even for simple read operations.

## How It Works

When an LLM includes a `<open>` command, the tool:
1. **Validates the path** - Ensures path is within repository bounds
2. **Creates I/O container** - Spins up minimal Alpine container (default: llm-runtime-io:latest)
3. **Mounts repository read-only** - Repository mounted at `/workspace`
4. **Reads file via cat** - Executes `cat` command in container
5. **Returns content** - File contents returned to LLM
6. **Cleans up** - Container automatically removed

## Basic Syntax

```
<open path/to/file.ext>
```

**Examples:**
- `<open main.go>` - Read Go source file
- `<open README.md>` - Read documentation
- `<open config/settings.yaml>` - Read configuration
- `<open .gitignore>` - Read hidden files
- `<open src/components/Button.tsx>` - Read nested files

## Common Use Cases

### **Understanding Codebase**
```
Let me examine the main entry point:

<open cmd/server/main.go>

And check the configuration structure:

<open internal/config/config.go>

Now let me look at the API handlers:

<open internal/handlers/api.go>
```

### **Reviewing Configuration**
```
Let me check the application configuration:

<open config/app.yaml>

And the database settings:

<open config/database.yaml>

Also the environment template:

<open .env.example>
```

### **Analyzing Dependencies**
```
Let me check the Go module dependencies:

<open go.mod>

And see the locked versions:

<open go.sum>

For Node projects:

<open package.json>

<open package-lock.json>
```

### **Documentation Review**
```
Let me read the project README:

<open README.md>

Check the contributing guidelines:

<open CONTRIBUTING.md>

And review the changelog:

<open CHANGELOG.md>
```

### **Test Analysis**
```
Let me examine the test files:

<open internal/auth/auth_test.go>

<open test/integration/api_test.go>

<open test/fixtures/sample_data.json>
```

### **Build & CI Configuration**
```
Let me check the Makefile:

<open Makefile>

Review the CI pipeline:

<open .github/workflows/ci.yml>

And Docker configuration:

<open Dockerfile>

<open docker-compose.yml>
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
Path: <open nonexistent.txt>
=== END ERROR ===
```

### **Path Outside Repository**
```
=== ERROR: READ_VALIDATION ===
Message: READ_VALIDATION: path outside repository: ../../../etc/passwd
Path: <open ../../../etc/passwd>
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
<open main.go>
<open src/components/App.tsx>
<open .github/workflows/ci.yml>
<open internal/../../README.md>  # resolves to README.md

❌ Blocked:
<open ../../../etc/passwd>        # outside repository
<open /etc/hosts>                 # absolute path outside repo
<open ~/.ssh/id_rsa>              # home directory access
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
<open ../outside/repository.txt>
```
**Cause**: Path points outside repository boundaries
**Solution**: Use paths relative to repository root

### **READ_FAILED**
```
<open nonexistent_file.txt>
```
**Cause**: File does not exist at specified path
**Solution**: Verify file path and name

### **DOCKER_UNAVAILABLE**
```
<open config.yaml>
```
**Cause**: Docker not running or I/O container image missing
**Solution**: Ensure Docker running and build I/O image

### **IO_TIMEOUT**
```
<open very_large_file.bin>
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
<open config.yaml>
<open main.go>
<open package.json>
```
**Performance**: Instant (< 100ms including container overhead)

### **Medium Files (1-10MB)**
```
<open data/large_config.json>
<open logs/application.log>
```
**Performance**: Fast (< 1 second)
**Consideration**: May hit LLM context limits

### **Large Files (> 10MB)**
```
<open database/dump.sql>
<open dist/bundle.js>
```
**Performance**: Slower (several seconds)
**Consideration**: Will likely exceed LLM context window
**Solution**: Use grep/head/tail to extract relevant sections

## Best Practices

### **Read Only What's Needed**
```
# ❌ Avoid reading entire files unnecessarily
<open large_log_file.log>

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

<open config/settings.yaml>
```

### **Progressive Reading**
```
Let me start with the README:

<open README.md>

Based on that, let me examine the main code:

<open cmd/server/main.go>

Now I'll look at specific modules mentioned:

<open internal/auth/auth.go>
```

### **Combine with Search**
```
Let me search for authentication logic:

<search authentication>

Based on the results, let me read the relevant file:

<open internal/auth/handler.go>
```

## Advanced Usage

### **Reading Multiple Related Files**
```
Let me examine the authentication system:

<open internal/auth/auth.go>

<open internal/auth/middleware.go>

<open internal/auth/token.go>

<open internal/auth/auth_test.go>

Now I understand the complete authentication flow.
```

### **Configuration Review**
```
Let me review all configuration files:

<open config/app.yaml>

<open config/database.yaml>

<open config/logging.yaml>

<open .env.example>

These settings look properly configured.
```

### **Dependency Analysis**
```
For a Go project:

<open go.mod>

<open go.sum>

For a Node project:

<open package.json>

<open package-lock.json>

For a Python project:

<open requirements.txt>

<open setup.py>
```

### **Documentation Gathering**
```
<open README.md>

<open docs/architecture.md>

<open docs/api.md>

<open CONTRIBUTING.md>

<open LICENSE>
```

## Reading Special Files

### **Hidden Files**
```
<open .gitignore>

<open .env.example>

<open .dockerignore>

<open .github/workflows/ci.yml>
```

### **Configuration Files**
```
# YAML
<open config.yaml>
<open .gitlab-ci.yml>

# JSON
<open package.json>
<open tsconfig.json>

# TOML
<open Cargo.toml>
<open pyproject.toml>

# INI
<open .editorconfig>
<open setup.cfg>
```

### **Build Files**
```
<open Makefile>

<open Dockerfile>

<open docker-compose.yml>

<open CMakeLists.txt>
```

### **Test Files**
```
<open test/integration_test.go>

<open tests/test_auth.py>

<open spec/models/user_spec.rb>

<open __tests__/component.test.tsx>
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
<open /absolute/path/file.txt>

# ✅ Correct - relative to repository root
<open relative/path/file.txt>

# ❌ Wrong
<open ~/home/file.txt>

# ✅ Correct
<open file.txt>
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
<open large_file.log>

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
<open file1.go>
<open file2.go>
<open file3.go>
<open file4.go>
<open file5.go>

# ✅ Better - batch related files in context
Let me examine the authentication module:

<open internal/auth/auth.go>
<open internal/auth/middleware.go>
<open internal/auth/token.go>

Now I'll look at the handlers...
```

### **Selective Reading**
```
# ❌ Reading unnecessary files
<open vendor/third_party/huge_lib.go>

# ✅ Focus on relevant project files
<open internal/core/business_logic.go>
```

## Integration with Other Commands

### **Read → Analyze → Write**
```
<open config/settings.yaml>

I see the port is set to 8080. Let me update it:

<write config/settings.yaml>
server:
  port: 3000
  host: localhost
</write>
```

### **Read → Test**
```
<open internal/auth/auth.go>

Let me verify this works:

<exec go test ./internal/auth>
```

### **Search → Read**
```
<search database connection>

Based on the search, let me read the database config:

<open internal/database/connection.go>
```

### **Read → Exec → Verify**
```
<open go.mod>

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

The `<open>` command provides LLMs with secure, containerized access to repository files, enabling thorough code analysis and understanding while maintaining complete isolation from the host system.

# Command Execution Guide - `<exec>` Command

## Overview

The `<exec>` command allows LLMs to execute shell commands in secure, isolated Docker containers. This enables LLMs to run tests, build projects, validate changes, and interact with development tools without compromising system security.

**Note**: Exec commands are always enabled via the container-based security model. Access is controlled exclusively through the command whitelist.

## How It Works

When an LLM includes an `<exec>` command, the tool:
1. **Validates the command** - Checks against whitelist of allowed commands
2. **Creates Docker container** - Spins up isolated container (default: python-go image)
3. **Mounts repository** - Repository mounted read-only at `/workspace`
4. **Executes command** - Runs command with strict resource limits
5. **Captures output** - Returns stdout, stderr, and exit code
6. **Cleans up** - Container automatically removed after execution

## Basic Syntax

```
<exec command arguments>
```

**Examples:**
- `<exec go test>` - Run Go tests
- `<exec npm build>` - Build Node.js project  
- `<exec python -m pytest>` - Run Python tests
- `<exec make clean>` - Run make command

## Security Model

### **Docker Isolation**
- **No network access**: `--network none` prevents internet access
- **Read-only repository**: Source code mounted as read-only
- **Temporary workspace**: Separate writable directory for outputs
- **Resource limits**: 512MB RAM, 2 CPU cores, 30-second timeout
- **Non-root execution**: Commands run as unprivileged user (1000:1000)

### **Command Whitelisting** 
Only pre-approved commands are allowed:

**Default Whitelist:**
- **Go**: `go test`, `go build`, `go run`, `go mod tidy`
- **Node.js**: `npm test`, `npm run build`, `npm install`, `node`
- **Python**: `python`, `python3`, `python -m pytest`, `pip install`
- **Build tools**: `make`, `make test`, `make build`
- **Rust**: `cargo build`, `cargo test`, `cargo run`
- **System**: `ls`, `cat`, `grep`, `find`, `head`, `tail`, `wc`

### **Container Security**
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
    python-go:latest
```

## Use Cases

### **Testing & Validation**
```
Let me run the test suite to verify everything works:

<exec go test ./...>

Now let me run just the unit tests:

<exec go test -short ./...>

And check test coverage:

<exec go test -cover ./...>
```

### **Building & Compilation**
```
First, let me build the project:

<exec go build -o bin/app .>

Now let's build for different platforms:

<exec go build -o bin/app-linux ./cmd/main.go>

Let me check if the binary was created:

<exec ls -la bin/>
```

### **Dependency Management**
```
Let me install the project dependencies:

<exec npm install>

And check for outdated packages:

<exec npm outdated>

Let me update the dependencies:

<exec npm update>
```

### **Code Analysis**
```
Let me run the linter:

<exec go vet ./...>

Check code formatting:

<exec gofmt -l .>

Run static analysis:

<exec go run honnef.co/go/tools/cmd/staticcheck ./...>
```

### **Project Information**
```
Let me examine the project structure:

<exec find . -name "*.go" -type f | head -10>

Check the lines of code:

<exec find . -name "*.go" -exec wc -l {} \; | awk '{sum += $1} END {print sum}'>

See what Git files have changed:

<exec git status>
```

### **Debugging & Troubleshooting**
```
Let me check if there are any compilation errors:

<exec go build .>

Run with verbose output:

<exec go test -v -run TestSpecificFunction>

Check for race conditions:

<exec go test -race ./...>
```

## Output Format

### **Successful Execution**
```
=== EXEC SUCCESSFUL: go test ===
Exit code: 0
Duration: 2.150s
Output:
?       github.com/example/project/cmd  [no test files]
ok      github.com/example/project/pkg  0.123s
=== END EXEC ===
```

### **Failed Execution**
```
=== EXEC SUCCESSFUL: go test ===
Exit code: 1
Duration: 1.234s
Output:
STDOUT:
--- FAIL: TestExample (0.00s)
    example_test.go:10: expected true, got false
FAIL
exit status 1

STDERR:
FAIL    github.com/example/project  0.001s
=== END EXEC ===
```

### **Command Validation Error**
```
=== ERROR: EXEC_VALIDATION ===
Message: EXEC_VALIDATION: command not in whitelist: rm
Command: <exec rm -rf />
=== END ERROR ===
```

## Common Error Types

### **EXEC_VALIDATION**
```
<exec rm -rf />
```
**Cause**: Command not in whitelist
**Solution**: Add to whitelist or use allowed alternative

### **DOCKER_UNAVAILABLE**
```
<exec go test>
```
**Cause**: Docker not installed or accessible
**Solution**: Install Docker and ensure it's running

### **EXEC_TIMEOUT**
```
<exec sleep 60>
```
**Cause**: Command took longer than 30 seconds
**Solution**: Optimize command or increase timeout

### **EXEC_FAILED**
```
<exec go test>  # when tests fail
```
**Cause**: Command executed but returned non-zero exit code
**Result**: Still shows output, but indicates failure

## Configuration

### **Command Line Options**
```bash
# Run with custom settings
./llm-runtime --exec-timeout 60s \
              --exec-memory 1g \
              --exec-cpu 4
```

### **Custom Whitelist**
```bash
# Add custom commands
./llm-runtime --exec-whitelist "go test,npm build,python -m pytest,make clean"
```

### **Configuration File**
```yaml
commands:
  exec:
    # Note: Exec is always enabled (container-based security).
    # Access is controlled via whitelist only.
    container_image: "python-go"
    timeout_seconds: 30
    memory_limit: "512m"
    cpu_limit: 2
    network_enabled: false
    whitelist:
      - "go test"
      - "go build"
      - "go run"
      - "npm test"
      - "npm run build"
      - "python -m pytest"
      - "make"
      - "cargo build"
      - "ls"
      - "find"
      - "grep"
      - "cat"
      - "head"
      - "tail"
      - "wc"
```

## Advanced Usage

### **Complex Build Pipelines**
```
I'll run the complete build and test pipeline:

<exec make clean>

<exec make deps>

<exec make lint>

<exec make test>

<exec make build>

<exec make package>
```

### **Conditional Execution**
```
Let me check if this is a Go project:

<exec ls go.mod>

Since it's a Go project, let me run Go-specific commands:

<exec go mod tidy>

<exec go test ./...>
```

### **Multi-Step Workflows**
```
I'll set up and test the development environment:

<exec npm install>

<exec npm run build>

<exec npm test>

<exec npm run lint>

Everything looks good! The project builds and all tests pass.
```

### **File System Operations**
```
Let me explore the project structure:

<exec find . -type f -name "*.go" | wc -l>

<exec find . -type d -name "test*">

<exec ls -la src/>

<exec du -sh .>
```

## Docker Requirements

### **Installation Check**
```bash
# Check if Docker is available
docker --version

# Test Docker functionality  
docker run hello-world

# Check for required image
docker images | grep python-go
```

### **Manual Image Preparation**
```bash
# Pre-pull the image to avoid delays
docker pull python-go:latest

# Or use alternative images
docker pull golang:1.21
docker pull node:18-alpine
docker pull python:3.11

# Verify image is available
docker image inspect python-go:latest
```

### **Custom Images**
You can configure custom Docker images with pre-installed tools:

```yaml
commands:
  exec:
    container_image: "node:18-alpine"  # For Node.js projects
    # or
    container_image: "golang:1.21"     # For Go projects
    # or  
    container_image: "python:3.11"     # For Python projects
```

## Performance Tips

### **Container Startup Time**
- **Pre-pull images**: `docker pull python-go:latest`
- **Use specific tags**: Avoid `latest` tag for consistency
- **Smaller images**: Consider Alpine variants

### **Command Optimization**
```
# Instead of multiple separate commands
<exec go test>
<exec go build>
<exec go vet>

# Combine when possible
<exec go test && go build && go vet>
```

### **Resource Management**
- **Adjust limits**: Increase memory/CPU for heavy operations
- **Cleanup**: Containers auto-remove, but monitor disk space
- **Parallel execution**: Multiple exec commands can run simultaneously

## Best Practices for LLMs

### **Always Verify Before Modifying**
```
Let me run tests before making changes:

<exec go test ./...>

[make changes with write commands]

Now let me verify the changes work:

<exec go test ./...>
```

### **Incremental Testing**
```
Let me test the specific component I'm working on:

<exec go test ./internal/auth>

Now test the integration:

<exec go test ./cmd/server>

Finally, run the full suite:

<exec go test ./...>
```

### **Build Verification**
```
Let me ensure the project builds correctly:

<exec go build ./cmd/server>

And verify there are no linting issues:

<exec go vet ./...>

<exec gofmt -l . | wc -l>
```

### **Environment Information**
```
Let me understand the environment:

<exec go version>

<exec go env GOPATH>

<exec ls -la>
```

## Troubleshooting

### **Docker Issues**
```bash
# Check Docker daemon
systemctl status docker

# Check Docker permissions
groups $USER | grep docker

# Test basic Docker functionality
docker run --rm python-go:latest echo "Hello"
```

### **Command Not Found**
1. **Check whitelist**: Command must be in allowed list
2. **Verify spelling**: Ensure command name is exact
3. **Check image**: Command must exist in the container image

### **Timeout Issues**
1. **Optimize command**: Make operations faster
2. **Increase timeout**: Use `--exec-timeout` flag
3. **Split operations**: Break into smaller commands

### **Permission Denied**
1. **Docker setup**: Ensure Docker is properly installed
2. **User permissions**: Add user to docker group
3. **Image access**: Verify image can be pulled

### **Memory/Resource Issues**
1. **Increase limits**: Use `--exec-memory` and `--exec-cpu`
2. **Optimize operations**: Reduce memory usage
3. **Monitor resources**: Check system resources

## Security Considerations

### **What's Allowed**
- Running tests and builds
- Code analysis and linting  
- File system inspection
- Package management (install/update)
- Development tool execution

### **What's Blocked**
- Network access (no internet)
- System modification commands (`rm -rf`, `sudo`)
- Privileged operations
- Access to host file system outside repository
- Long-running services

### **Audit Trail**
All exec commands are logged with:
- Command executed
- Exit code and duration  
- Success/failure status
- Timestamp and session ID

Example audit log:
```
2025-12-15T10:30:46Z|session:1234567890|exec|go test|success|exit_code:0,duration:1.234s
```

## Container vs Host Execution

**Why containers?**
- **Security**: Complete isolation from host system
- **Consistency**: Same environment regardless of host OS
- **Resource control**: Enforced limits prevent abuse
- **Cleanup**: Automatic removal prevents accumulation
- **No network**: Cannot download malware or exfiltrate data

**Trade-offs:**
- **Startup time**: 1-3 seconds per command (cached images)
- **Resource overhead**: Container management overhead
- **Image size**: Requires pre-pulled images

The `<exec>` command transforms LLMs from read-only code analyzers into active development assistants capable of testing, building, and validating their changes in real-time while maintaining complete security isolation.

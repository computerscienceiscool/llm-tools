# File Writing Guide - `<write>` Command

## Overview

The `<write>` command allows LLMs to create or modify files in the repository using secure, atomic operations executed in isolated Docker containers. All writes are performed via temporary files to ensure data integrity and prevent partial writes.

**Security**: All file writes are executed in minimal Alpine Linux containers with carefully controlled write access, using atomic rename operations to ensure consistency.

## How It Works

When an LLM includes a `<write>` command, the tool:
1. **Validates the path** - Ensures path is within repository bounds
2. **Creates I/O container** - Spins up minimal Alpine container (default: llm-runtime-io:latest)
3. **Mounts repository** - Repository mounted at `/workspace` with write access to workspace
4. **Writes to temp file** - Content written to `.tmp` file first
5. **Atomic rename** - Temporary file renamed to final destination
6. **Verifies write** - Confirms file exists and has correct size
7. **Cleans up** - Container automatically removed

## Basic Syntax

```
<write path/to/file.ext>
file content here
can span multiple lines
</write>
```

**Examples:**

```
<write config/settings.yaml>
server:
  port: 8080
  host: localhost
</write>

<write main.go>
package main

import "fmt"

func main() {
    fmt.Println("Hello, World!")
}
</write>

<write README.md>
# My Project

This is my awesome project.
</write>
```

## Common Use Cases

### **Creating New Files**
```
Let me create a new configuration file:

<write config/database.yaml>
database:
  host: localhost
  port: 5432
  name: myapp_dev
  user: developer
  max_connections: 100
</write>

And add a new module:

<write internal/utils/helpers.go>
package utils

import "strings"

func Capitalize(s string) string {
    if s == "" {
        return s
    }
    return strings.ToUpper(s[:1]) + s[1:]
}
</write>
```

### **Modifying Existing Files**
```
Let me update the main configuration:

<write config/app.yaml>
app:
  name: "My Application"
  version: "2.0.0"
  environment: "production"
  debug: false
  
server:
  port: 8080
  host: "0.0.0.0"
  timeout: 30s
</write>

And fix the bug in the handler:

<write internal/handlers/api.go>
package handlers

import (
    "net/http"
    "encoding/json"
)

func HandleAPI(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]string{
        "status": "ok",
        "version": "2.0.0",
    })
}
</write>
```

### **Adding Tests**
```
Let me add comprehensive tests:

<write internal/auth/auth_test.go>
package auth

import (
    "testing"
)

func TestValidateToken(t *testing.T) {
    tests := []struct {
        name    string
        token   string
        wantErr bool
    }{
        {"valid token", "valid.jwt.token", false},
        {"invalid token", "invalid", true},
        {"empty token", "", true},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := ValidateToken(tt.token)
            if (err != nil) != tt.wantErr {
                t.Errorf("ValidateToken() error = %v, wantErr %v", err, tt.wantErr)
            }
        })
    }
}
</write>
```

### **Documentation Updates**
```
Let me update the README with new features:

<write README.md>
# LLM Runtime

A secure runtime environment for LLM-assisted development.

## Features

- **Semantic Search**: Search codebase using natural language
- **File Operations**: Secure read/write with containerization
- **Command Execution**: Run tests and builds in isolated containers
- **Audit Logging**: Complete audit trail of all operations

## Installation

```bash
go install github.com/example/llm-runtime@latest
```

## Quick Start

```bash
# Initialize search index
./llm-runtime --reindex

# Start interactive session
./llm-runtime
```

## Configuration

See [docs/configuration.md](docs/configuration.md) for details.

## License

MIT License - see LICENSE file.
</write>
```

### **Configuration Management**
```
<write .env.production>
DATABASE_URL=postgres://prod:password@db.example.com:5432/myapp
REDIS_URL=redis://cache.example.com:6379
API_KEY=prod_key_here
LOG_LEVEL=info
</write>

<write config/environments/production.yaml>
environment: production
server:
  port: 8080
  host: 0.0.0.0
database:
  pool_size: 50
  timeout: 30s
cache:
  ttl: 3600
  max_size: 1000
</write>
```

### **Build Configuration**
```
<write Makefile>
.PHONY: build test clean

build:
	go build -o bin/app cmd/server/main.go

test:
	go test ./...

clean:
	rm -rf bin/
</write>

<write Dockerfile>
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o server cmd/server/main.go

FROM alpine:latest
COPY --from=builder /app/server /server
EXPOSE 8080
CMD ["/server"]
</write>
```

## Output Format

### **Successful Write**
```
=== WRITE SUCCESSFUL: config/settings.yaml ===
Bytes written: 156
Path: config/settings.yaml
=== END WRITE ===
```

### **File Creation**
```
=== WRITE SUCCESSFUL: internal/new_module.go ===
Bytes written: 342
Path: internal/new_module.go
File created: true
=== END WRITE ===
```

### **Path Validation Error**
```
=== ERROR: WRITE_VALIDATION ===
Message: WRITE_VALIDATION: path outside repository: ../../../etc/passwd
Path: <write ../../../etc/passwd>
=== END ERROR ===
```

### **Write Failed**
```
=== ERROR: WRITE_FAILED ===
Message: WRITE_FAILED: permission denied
Path: <write /protected/file.txt>
=== END ERROR ===
```

## Security Model

### **Containerized Writes**
All file writes execute in isolated containers:

```bash
docker run \
    --rm \
    --network none \
    --user 1000:1000 \
    --cap-drop ALL \
    --security-opt no-new-privileges \
    --read-only \
    --tmpfs /workspace:exec,mode=1777 \
    --memory 128m \
    --cpus 1 \
    -v /repo:/workspace/repo:rw \
    llm-runtime-io:latest \
    sh -c 'cat > /workspace/file.tmp && mv /workspace/file.tmp /workspace/repo/file.txt'
```

**Security Features:**
- **No network access**: Container completely isolated
- **Limited write access**: Can only write to repository directory
- **Minimal image**: Alpine Linux (~5MB) reduces attack surface
- **Resource limits**: 128MB RAM, 1 CPU core, 10-second timeout
- **Non-root user**: Runs as unprivileged user (1000:1000)
- **No new privileges**: Cannot escalate permissions
- **Atomic writes**: Temp file + rename prevents partial writes

### **Atomic Write Operations**
```
1. Write content to temporary file: file.txt.tmp
2. Verify write completed successfully
3. Atomically rename: file.txt.tmp → file.txt
4. Verify final file exists
```

**Benefits:**
- **No partial writes**: File is either fully written or not at all
- **Concurrent safety**: Atomic rename is filesystem-guaranteed
- **Corruption prevention**: Original file preserved if write fails
- **Crash safety**: Interrupted writes don't leave broken files

### **Path Validation**
```
✅ Allowed:
<write new_file.go>
<write src/components/Button.tsx>
<write config/settings.yaml>
<write .github/workflows/ci.yml>

❌ Blocked:
<write ../../../etc/passwd>        # outside repository
<write /etc/hosts>                 # absolute path outside repo
<write ~/.bashrc>                  # home directory access
```

### **Defense in Depth**
- **Container isolation**: Exploits cannot escape to host
- **Resource limits**: Cannot consume unlimited resources
- **Audit trail**: All writes logged with complete details
- **Consistent security**: Same model as read/exec operations

## Error Types

### **WRITE_VALIDATION**
```
<write ../outside/repo/file.txt>
```
**Cause**: Path points outside repository boundaries
**Solution**: Use paths relative to repository root

### **WRITE_FAILED**
```
<write /protected/system/file>
```
**Cause**: Cannot write to specified location (permissions, etc.)
**Solution**: Check path permissions and validity

### **DOCKER_UNAVAILABLE**
```
<write config.yaml>
```
**Cause**: Docker not running or I/O container image missing
**Solution**: Ensure Docker running and build I/O image

### **IO_TIMEOUT**
```
<write huge_file.bin>
...gigabytes of content...
</write>
```
**Cause**: Write operation exceeded timeout (default: 10 seconds)
**Solution**: Increase timeout or split into smaller writes

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
    container_image: "llm-runtime-io:latest"
    timeout_seconds: 10
    memory_limit: "128m"
    cpu_limit: 1
    fallback_image: "alpine:latest"
```

### **Custom I/O Container**
```bash
# Build the default I/O container image
make build-io-image

# Or use standard Alpine
./llm-runtime --io-container alpine:latest
```

## Best Practices for LLMs

### **Read Before Writing**
```
Let me first check the current configuration:

<open config/settings.yaml>

Now I'll update it with the new values:

<write config/settings.yaml>
server:
  port: 3000
  host: localhost
database:
  connection: postgres://localhost/myapp
</write>
```

### **Test After Writing**
```
Let me create the new handler:

<write internal/handlers/user.go>
package handlers

import "net/http"

func HandleUser(w http.ResponseWriter, r *http.Request) {
    w.Write([]byte("User handler"))
}
</write>

Now let me verify it compiles:

<exec go build ./internal/handlers>

And run the tests:

<exec go test ./internal/handlers>
```

### **Incremental Changes**
```
# ❌ Avoid massive rewrites
<write entire_large_codebase.go>
...thousands of lines...
</write>

# ✅ Better: Make focused, incremental changes
<write internal/auth/middleware.go>
...specific changes to middleware...
</write>

<write internal/auth/middleware_test.go>
...tests for the changes...
</write>
```

### **Explain Changes**
```
I found the bug in the authentication logic. The token validation
was checking the wrong field. Let me fix it:

<write internal/auth/token.go>
package auth

import "errors"

func ValidateToken(token string) error {
    if token == "" {
        return errors.New("token cannot be empty")
    }
    
    // Fixed: now checking token.Valid instead of token.Expired
    if !token.Valid {
        return errors.New("invalid token")
    }
    
    return nil
}
</write>

This ensures we're checking the correct field for token validity.
```

### **Complete File Writes**
```
# ❌ Avoid partial/incomplete writes
<write config.yaml>
server:
  port: 8080
  # TODO: add more config
</write>

# ✅ Better: Complete, functional configurations
<write config.yaml>
server:
  port: 8080
  host: localhost
  timeout: 30s
  
database:
  connection: postgres://localhost:5432/myapp
  pool_size: 10
  
logging:
  level: info
  format: json
</write>
```

## Advanced Usage

### **Multi-File Updates**
```
Let me refactor the authentication system:

<write internal/auth/auth.go>
package auth

type Authenticator struct {
    secret string
}

func NewAuthenticator(secret string) *Authenticator {
    return &Authenticator{secret: secret}
}
</write>

<write internal/auth/middleware.go>
package auth

import "net/http"

func (a *Authenticator) Middleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Authentication logic
        next.ServeHTTP(w, r)
    })
}
</write>

<write internal/auth/auth_test.go>
package auth

import "testing"

func TestNewAuthenticator(t *testing.T) {
    a := NewAuthenticator("secret")
    if a.secret != "secret" {
        t.Error("secret not set correctly")
    }
}
</write>
```

### **Configuration Updates**
```
Let me update all configuration files for production:

<write config/production.yaml>
environment: production
debug: false
server:
  port: 8080
  host: 0.0.0.0
</write>

<write config/database-production.yaml>
database:
  host: prod-db.example.com
  port: 5432
  ssl_mode: require
  pool_size: 50
</write>

<write .env.production>
LOG_LEVEL=warn
API_KEY=prod_key_here
SECRET_KEY=prod_secret_here
</write>
```

### **Documentation Generation**
```
<write docs/api.md>
# API Documentation

## Authentication

All API endpoints require authentication via Bearer token.

## Endpoints

### GET /api/users
Returns list of users.

**Response:**
```json
{
  "users": [
    {"id": 1, "name": "Alice"},
    {"id": 2, "name": "Bob"}
  ]
}
```

### POST /api/users
Creates a new user.

**Request:**
```json
{
  "name": "Charlie",
  "email": "charlie@example.com"
}
```
</write>
```

### **Build File Creation**
```
<write Makefile>
.PHONY: all build test clean install

all: build

build:
	@echo "Building..."
	go build -o bin/app cmd/server/main.go

test:
	@echo "Running tests..."
	go test -v ./...

clean:
	@echo "Cleaning..."
	rm -rf bin/

install:
	@echo "Installing..."
	go install ./cmd/server
</write>

<write .github/workflows/ci.yml>
name: CI

on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
        with:
          go-version: '1.21'
      - run: go test ./...
</write>
```

## Working with Different File Types

### **Go Files**
```
<write cmd/server/main.go>
package main

import (
    "log"
    "net/http"
)

func main() {
    http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        w.Write([]byte("Hello, World!"))
    })
    
    log.Println("Server starting on :8080")
    log.Fatal(http.ListenAndServe(":8080", nil))
}
</write>
```

### **JavaScript/TypeScript**
```
<write src/components/Button.tsx>
import React from 'react';

interface ButtonProps {
  label: string;
  onClick: () => void;
}

export const Button: React.FC<ButtonProps> = ({ label, onClick }) => {
  return (
    <button onClick={onClick} className="btn">
      {label}
    </button>
  );
};
</write>
```

### **Python**
```
<write app/main.py>
from fastapi import FastAPI
from typing import Dict

app = FastAPI()

@app.get("/")
async def root() -> Dict[str, str]:
    return {"message": "Hello World"}

@app.get("/health")
async def health() -> Dict[str, str]:
    return {"status": "ok"}
</write>
```

### **YAML Configuration**
```
<write config/app.yaml>
app:
  name: "My Application"
  version: "1.0.0"
  
server:
  port: 8080
  host: localhost
  
database:
  host: localhost
  port: 5432
  name: myapp
  
logging:
  level: info
  format: json
  output: stdout
</write>
```

### **JSON Configuration**
```
<write package.json>
{
  "name": "my-app",
  "version": "1.0.0",
  "description": "My application",
  "main": "index.js",
  "scripts": {
    "start": "node index.js",
    "test": "jest",
    "build": "webpack"
  },
  "dependencies": {
    "express": "^4.18.0",
    "dotenv": "^16.0.0"
  },
  "devDependencies": {
    "jest": "^29.0.0",
    "webpack": "^5.75.0"
  }
}
</write>
```

### **Markdown Documentation**
```
<write CONTRIBUTING.md>
# Contributing Guide

## Getting Started

1. Fork the repository
2. Clone your fork
3. Create a feature branch
4. Make your changes
5. Run tests
6. Submit a pull request

## Code Style

- Follow existing code style
- Add tests for new features
- Update documentation

## Testing

```bash
make test
```

## Questions?

Open an issue or contact the maintainers.
</write>
```

## Troubleshooting

### **Permission Denied**
```bash
# Check Docker daemon
systemctl status docker

# Ensure user in docker group
groups $USER | grep docker

# Check repository permissions
ls -la /path/to/repo

# Test Docker access
docker run --rm alpine:latest touch /tmp/test
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
docker run --rm llm-runtime-io:latest sh -c 'echo test > /tmp/file && cat /tmp/file'
```

### **Path Issues**
```
# ❌ Wrong - absolute path
<write /etc/config.yaml>

# ✅ Correct - relative to repository
<write config/config.yaml>

# ❌ Wrong - outside repository
<write ../../../etc/passwd>

# ✅ Correct - within repository
<write config/database.yaml>
```

### **Write Verification**
```
After writing, verify the file:

<write config/settings.yaml>
server:
  port: 8080
</write>

<exec cat config/settings.yaml>

<exec ls -la config/settings.yaml>
```

### **Large File Writes**
```
# For very large files, consider:
1. Increasing timeout: --io-timeout 60s
2. Splitting into chunks
3. Using exec to generate file instead
```

## Performance Optimization

### **Batch Related Writes**
```
# ✅ Good - batch related changes
<write internal/models/user.go>
...complete user model...
</write>

<write internal/models/user_test.go>
...complete user tests...
</write>

# ❌ Avoid - many tiny writes
<write file1.go>...</write>
<write file2.go>...</write>
<write file3.go>...</write>
...dozens more...
```

### **Container Image Preparation**
```bash
# Pre-build I/O image to avoid delays
make build-io-image

# Verify it's available
docker images | grep llm-runtime-io

# Test write performance
time docker run --rm llm-runtime-io:latest sh -c 'echo test > /tmp/file'
```

## Audit Trail

All write operations are logged:

```
2025-12-15T10:30:45Z|session:abc123|write|config/settings.yaml|success|bytes:1234|created:false
2025-12-15T10:30:46Z|session:abc123|write|new_file.go|success|bytes:5678|created:true
2025-12-15T10:30:47Z|session:abc123|write|protected.txt|failed|error:permission_denied
```

**Logged Information:**
- Timestamp
- Session ID
- Operation type (write)
- File path
- Success/failure status
- Bytes written (on success)
- Whether file was created (new) or modified (existing)
- Error type (on failure)
- Container execution details

## Why Atomic Writes?

### **Problem: Non-Atomic Writes**
```
# What could go wrong:
1. Start writing file...
2. Process crashes mid-write
3. File left in corrupted state
4. Original content lost
```

### **Solution: Atomic Rename**
```
# How atomic writes work:
1. Write to: config.yaml.tmp
2. Verify write succeeded
3. Atomic rename: config.yaml.tmp → config.yaml
4. Original preserved if anything fails
```

### **Benefits**
- **No corruption**: File is complete or unchanged
- **Concurrent safety**: Other processes see complete file
- **Crash safety**: Interruptions don't break files
- **Rollback possible**: Original preserved until rename

## Integration with Other Commands

### **Read → Modify → Write**
```
<open config/settings.yaml>

I see the port is 8080. Let me update it:

<write config/settings.yaml>
server:
  port: 3000
  host: localhost
</write>
```

### **Write → Test → Verify**
```
<write internal/auth/token.go>
package auth

func ValidateToken(token string) error {
    // implementation
    return nil
}
</write>

<exec go test ./internal/auth>

<exec go build ./internal/auth>
```

### **Write → Search → Validate**
```
<write internal/database/connection.go>
...new database code...
</write>

<search database connection>

The new code is properly indexed and searchable.
```

The `<write>` command enables LLMs to make secure, atomic modifications to repository files, with containerized execution ensuring complete isolation and data integrity through temporary files and atomic rename operations.

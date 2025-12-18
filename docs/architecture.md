# Architecture

This document describes the technical architecture of llm-runtime, including its design patterns, component structure, and implementation details.

## Overview

llm-runtime is designed as a secure bridge between Large Language Models and code repositories, enabling LLMs to read, modify, search, and test code through containerized operations.

## High-Level Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    External LLM                             │
│              (Claude, GPT-4, Local Models)                  │
└────────────────────────┬────────────────────────────────────┘
                         │
                         │ Text Commands via stdin
                         │ <open>, <write>, <exec>, <search>
                         │
┌────────────────────────▼────────────────────────────────────┐
│                   llm-runtime Core                          │
│                                                             │
│  ┌──────────────────────────────────────────────────────┐  │
│  │           Command Parser & Router                     │  │
│  │   • Regex-based command extraction                   │  │
│  │   • Syntax validation                                │  │
│  │   • Command routing                                  │  │
│  └─────┬─────────────┬──────────────┬─────────────┬──────┘  │
│        │             │              │             │         │
│  ┌─────▼─────┐ ┌────▼────┐   ┌─────▼─────┐ ┌────▼─────┐  │
│  │   Read    │ │  Write  │   │   Exec    │ │  Search  │  │
│  │  Handler  │ │ Handler │   │  Handler  │ │ Handler  │  │
│  └─────┬─────┘ └────┬────┘   └─────┬─────┘ └────┬─────┘  │
│        │             │              │             │         │
│  ┌─────▼─────────────▼──────┐   ┌──▼──────────┐ │         │
│  │  I/O Containerization    │   │   Exec      │ │         │
│  │  (Docker Alpine)         │   │   Sandbox   │ │         │
│  │                          │   │  (Docker)   │ │         │
│  │  • Path validation       │   │             │ │         │
│  │  • Container creation    │   │  • Whitelist│ │         │
│  │  • Read via cat          │   │  • Resource │ │         │
│  │  • Atomic writes         │   │    limits   │ │         │
│  │  • Cleanup               │   │  • Isolation│ │         │
│  └──────────────────────────┘   └─────────────┘ │         │
│                                                  │         │
│                                      ┌───────────▼──────┐  │
│                                      │  Search Engine   │  │
│                                      │  (Ollama)        │  │
│                                      │                  │  │
│                                      │  • Embeddings    │  │
│                                      │  • Vector search │  │
│                                      │  • SQLite index  │  │
│                                      └──────────────────┘  │
│                                                             │
│  ┌──────────────────────────────────────────────────────┐  │
│  │               Audit Logger                            │  │
│  │   • All operations logged                            │  │
│  │   • Structured output                                │  │
│  │   • Session tracking                                 │  │
│  └──────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
                         │
                         │ Results via stdout
                         │
                         ▼
                 ┌───────────────┐
                 │   audit.log   │
                 └───────────────┘
```

## Component Details

### 1. Command Parser

**Location:** `pkg/parser/`

**Responsibilities:**
- Extract commands from LLM text output
- Validate command syntax
- Route to appropriate handlers

**Supported Patterns:**
```
<open path/to/file>
<write path/to/file>content</write>
<exec command args>
<search query terms>
```

**Implementation:**
```go
type Parser struct {
    patterns map[string]*regexp.Regexp
}

func (p *Parser) Parse(input string) ([]Command, error)
func (p *Parser) ValidateSyntax(cmd Command) error
```

### 2. I/O Handler (Read/Write)

**Location:** `pkg/io/`

**Phase 5 Implementation - Containerized I/O:**

**Read Operations:**
```go
type ReadHandler struct {
    containerizer *IOContainerizer
    validator     *PathValidator
}

func (h *ReadHandler) Read(path string) ([]byte, error) {
    // 1. Validate path (within repo bounds)
    if err := h.validator.Validate(path); err != nil {
        return nil, err
    }
    
    // 2. Execute read in container
    return h.containerizer.ReadFile(path)
}
```

**Container Read Process:**
1. Create Alpine container
2. Mount repository read-only
3. Execute `cat` command
4. Capture output
5. Destroy container

**Write Operations:**
```go
type WriteHandler struct {
    containerizer *IOContainerizer
    validator     *PathValidator
}

func (h *WriteHandler) Write(path string, content []byte) error {
    // 1. Validate path
    // 2. Write to temp file in container
    // 3. Atomic rename
    // 4. Verify
}
```

**Atomic Write Process:**
1. Create Alpine container
2. Write to `.tmp` file
3. Atomically rename to final path
4. Verify write succeeded
5. Destroy container

**Security Features:**
- Path validation (no directory traversal)
- Container isolation
- Read-only repository mounts (for reads)
- Atomic operations (for writes)
- Resource limits (128MB RAM, 1 CPU, 10s timeout)

### 3. Exec Handler

**Location:** `pkg/exec/`

**Implementation:**
```go
type ExecHandler struct {
    whitelist     []string
    containerizer *ExecContainerizer
    config        *ExecConfig
}

func (h *ExecHandler) Execute(cmd string) (*ExecResult, error) {
    // 1. Validate against whitelist
    if !h.isWhitelisted(cmd) {
        return nil, ErrNotWhitelisted
    }
    
    // 2. Create container
    // 3. Execute command
    // 4. Capture output
    // 5. Cleanup
}
```

**Container Configuration:**
```go
type ExecConfig struct {
    Image       string        // "python-go"
    Memory      string        // "512m"
    CPULimit    int           // 2
    Timeout     time.Duration // 30s
    NetworkMode string        // "none"
}
```

**Execution Flow:**
1. Whitelist validation
2. Docker container creation
3. Repository mounted read-only at `/workspace`
4. Command execution with limits
5. Output capture (stdout/stderr)
6. Container cleanup
7. Result return

**Always Enabled:** Exec is always available. Access controlled via whitelist only.

### 4. Search Handler

**Location:** `pkg/search/`

**Implementation:**
```go
type SearchHandler struct {
    index   *EmbeddingIndex
    ollama  *OllamaClient
    chunker *CodeChunker
}

func (h *SearchHandler) Search(query string, limit int) ([]SearchResult, error) {
    // 1. Generate query embedding via Ollama
    embedding := h.ollama.Embed(query)
    
    // 2. Search index for similar embeddings
    results := h.index.Search(embedding, limit)
    
    // 3. Return ranked results
    return results, nil
}
```

**Indexing Process:**
```go
func (h *SearchHandler) BuildIndex(repoPath string) error {
    // 1. Scan repository for code files
    files := h.scanFiles(repoPath)
    
    // 2. Chunk files into manageable pieces
    chunks := h.chunker.Chunk(files)
    
    // 3. Generate embeddings via Ollama
    for _, chunk := range chunks {
        embedding := h.ollama.Embed(chunk.Content)
        h.index.Store(chunk, embedding)
    }
    
    // 4. Save index to SQLite
    return h.index.Save()
}
```

**Components:**
- **Ollama Client:** Communicates with local Ollama server
- **Embedding Index:** SQLite-based vector storage
- **Code Chunker:** Splits files into searchable chunks
- **Result Ranker:** Orders by cosine similarity

### 5. Path Validator

**Location:** `pkg/security/`

**Implementation:**
```go
type PathValidator struct {
    repoRoot  string
    exclusions []string
}

func (v *PathValidator) Validate(path string) error {
    // 1. Resolve to absolute path
    absPath := filepath.Join(v.repoRoot, path)
    
    // 2. Check if within repository bounds
    if !strings.HasPrefix(absPath, v.repoRoot) {
        return ErrPathOutsideRepo
    }
    
    // 3. Check exclusions (.git, sensitive dirs)
    if v.isExcluded(absPath) {
        return ErrPathExcluded
    }
    
    return nil
}
```

**Validation Rules:**
- Must be within repository root
- No parent directory traversal (`../`)
- Not in exclusion list (.git, .env, etc.)
- No absolute paths outside repo

### 6. Container Manager

**Location:** `pkg/container/`

**I/O Containerizer (Phase 5):**
```go
type IOContainerizer struct {
    image       string // "llm-runtime-io:latest"
    timeout     time.Duration
    memoryLimit string
    cpuLimit    int
}

func (c *IOContainerizer) ReadFile(path string) ([]byte, error) {
    ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
    defer cancel()
    
    // Create container
    container := c.createContainer(ctx, "cat", path)
    
    // Execute
    output, err := container.Run(ctx)
    
    // Cleanup
    container.Remove()
    
    return output, err
}

func (c *IOContainerizer) WriteFile(path string, content []byte) error {
    // Similar but with atomic write logic
}
```

**Exec Containerizer:**
```go
type ExecContainerizer struct {
    image       string // "python-go"
    timeout     time.Duration
    memoryLimit string
    cpuLimit    int
}

func (c *ExecContainerizer) Execute(cmd string) (*Result, error) {
    // Similar pattern to I/O containerizer
    // but with command whitelisting
}
```

**Container Security Configuration:**
```go
containerConfig := &container.Config{
    Image: "llm-runtime-io:latest",
    User:  "1000:1000", // non-root
    Cmd:   []string{"cat", filepath},
}

hostConfig := &container.HostConfig{
    NetworkMode:  "none",              // No network
    ReadonlyRootfs: true,              // Read-only filesystem
    CapDrop:      []string{"ALL"},     // Drop all capabilities
    SecurityOpt:  []string{"no-new-privileges"},
    Resources: container.Resources{
        Memory:   128 * 1024 * 1024,   // 128MB
        NanoCPUs: 1000000000,           // 1 CPU
    },
}
```

### 7. Audit Logger

**Location:** `pkg/audit/`

**Implementation:**
```go
type AuditLogger struct {
    file      *os.File
    sessionID string
}

func (l *AuditLogger) Log(event AuditEvent) error {
    entry := fmt.Sprintf(
        "%s|%s|%s|%s|%s|%s\n",
        event.Timestamp.Format(time.RFC3339),
        l.sessionID,
        event.Operation,
        event.Target,
        event.Status,
        event.Metadata,
    )
    _, err := l.file.WriteString(entry)
    return err
}
```

**Audit Event Types:**
- `read` - File read operations
- `write` - File write operations
- `exec` - Command executions
- `search` - Search queries

**Log Format:**
```
TIMESTAMP|SESSION_ID|OPERATION|TARGET|STATUS|METADATA
```

**Example:**
```
2025-12-15T10:30:45Z|abc123|read|config.yaml|success|size:1234
2025-12-15T10:30:46Z|abc123|exec|go test|success|exit:0,duration:2.5s
2025-12-15T10:30:47Z|abc123|write|main.go|success|bytes:5678
2025-12-15T10:30:48Z|abc123|search|authentication|success|results:5
```

## Development Phases

### Phase 1: Core File Operations ✅
**Completed:** Initial release

- Basic file reading
- File writing
- Path validation
- Error handling
- CLI interface

### Phase 2: Command Execution ✅
**Completed:** Q2 2024

- Docker integration
- Command whitelisting
- Resource limits
- Container isolation
- Security hardening

### Phase 3: Semantic Search ✅
**Completed:** Q3 2024

- Ollama integration
- Embedding generation (nomic-embed-text)
- Vector similarity search
- SQLite index storage
- Reindexing capability

### Phase 4: Enhanced Security & Audit ✅
**Completed:** Q4 2024

- Comprehensive audit logging
- Session tracking
- Structured logging
- Security event monitoring
- Configuration management

### Phase 5: I/O Containerization ✅
**Completed:** Q4 2024

**Key Features:**
- File reads via containerized `cat`
- Atomic writes via temp files in containers
- Minimal Alpine-based containers
- Defense-in-depth security model
- Configurable resource limits
- Separate I/O container image

**Benefits:**
- Additional isolation layer for all file operations
- Protection against file parsing vulnerabilities
- Consistent security model across all operations
- Resource isolation for I/O operations
- Complete audit trail with container details

**Implementation Details:**
```yaml
# I/O Containerization Config
commands:
  io:
    container_image: "llm-runtime-io:latest"
    timeout_seconds: 10
    memory_limit: "128m"
    cpu_limit: 1
    fallback_image: "alpine:latest"
```

### Phase 6: Future Enhancements 🔮
**Planned:** 2025

- Plugin system for custom commands
- Multi-repository workspace support
- Advanced caching strategies
- Real-time collaboration features
- Cloud execution options
- Language server protocol integration

## Data Flow

### File Read Flow

```
LLM Request
    ↓
Command Parser → <open config.yaml>
    ↓
Path Validator → Check: config.yaml within repo? ✓
    ↓
I/O Containerizer → Create Alpine container
    ↓
Docker → Run: cat /workspace/config.yaml
    ↓
Container → Execute, capture output
    ↓
Cleanup → Remove container
    ↓
Audit Logger → Log: read|config.yaml|success|size:156
    ↓
Result → Return contents to LLM
```

### File Write Flow

```
LLM Request
    ↓
Command Parser → <write main.go>package main...</write>
    ↓
Path Validator → Check: main.go within repo? ✓
    ↓
I/O Containerizer → Create Alpine container
    ↓
Docker → Write to: /workspace/main.go.tmp
    ↓
Container → Atomic rename: main.go.tmp → main.go
    ↓
Verify → Check file exists, correct size
    ↓
Cleanup → Remove container
    ↓
Audit Logger → Log: write|main.go|success|bytes:342
    ↓
Result → Return confirmation to LLM
```

### Exec Flow

```
LLM Request
    ↓
Command Parser → <exec go test>
    ↓
Whitelist Check → Is "go test" whitelisted? ✓
    ↓
Exec Containerizer → Create python-go container
    ↓
Docker → Mount repo read-only, run: go test
    ↓
Container → Execute with limits (30s, 512MB, 2 CPU)
    ↓
Capture → Collect stdout, stderr, exit code
    ↓
Cleanup → Remove container
    ↓
Audit Logger → Log: exec|go test|success|exit:0,duration:2.3s
    ↓
Result → Return output to LLM
```

### Search Flow

```
LLM Request
    ↓
Command Parser → <search authentication>
    ↓
Search Handler → Generate query embedding
    ↓
Ollama Client → Call: POST /api/embeddings
    ↓
Ollama → Return: [0.123, -0.456, ...]
    ↓
Index Search → Find similar embeddings (cosine similarity)
    ↓
SQLite → Query: SELECT ... ORDER BY similarity DESC LIMIT 5
    ↓
Result Ranking → Sort by relevance score
    ↓
Audit Logger → Log: search|authentication|success|results:5
    ↓
Result → Return file paths and chunks to LLM
```

## Security Architecture

### Defense in Depth Layers

```
Layer 1: Input Validation
  ├─ Command syntax validation
  ├─ Path boundary checking
  └─ Whitelist enforcement

Layer 2: Container Isolation
  ├─ Network disabled (--network none)
  ├─ Read-only filesystems
  ├─ Non-root execution (uid:1000)
  ├─ Capabilities dropped (--cap-drop ALL)
  └─ No privilege escalation

Layer 3: Resource Limits
  ├─ Memory caps (128MB-512MB)
  ├─ CPU limits (1-2 cores)
  ├─ Execution timeouts (10s-30s)
  └─ Storage quotas

Layer 4: Audit & Monitoring
  ├─ All operations logged
  ├─ Session tracking
  ├─ Anomaly detection (future)
  └─ Alert mechanisms (future)
```

### Security Boundaries

**Container Boundary:**
- Prevents escape to host system
- Isolates network access
- Limits resource consumption
- Enforces read-only mounts

**Path Boundary:**
- Restricts access to repository
- Blocks directory traversal
- Excludes sensitive paths
- Validates all file operations

**Command Boundary:**
- Whitelist-only execution
- No arbitrary shell access
- Controlled tool access
- Blocked destructive commands

## Performance Considerations

### Container Overhead

**Cold Start (first time):**
- Image pull: 30-120s (depending on image size)
- Container creation: 1-3s

**Warm Start (cached images):**
- Container creation: 50-200ms
- Command execution: Actual command time + overhead
- Container cleanup: 50-100ms

**Optimization Strategies:**
1. Pre-pull images during installation
2. Use smaller base images (Alpine)
3. Keep containers minimal
4. Implement container pooling (future)

### Search Performance

**Index Build:**
- Small repo (100 files): 10-30s
- Medium repo (1000 files): 1-3 min
- Large repo (10,000 files): 10-30 min

**Search Query:**
- Embedding generation: 50-200ms
- Index lookup: 10-50ms
- Total: ~100-500ms typical

**Optimization:**
- Incremental indexing (future)
- Caching query embeddings
- Optimized SQLite indices
- Batch embedding generation

### Memory Usage

**Tool Process:**
- Base: ~50MB
- Search index loaded: +10-100MB (varies by repo size)
- Peak: ~150-300MB typical

**Containers:**
- I/O container: 128MB limit
- Exec container: 512MB limit
- Multiple concurrent: Limits per container

## Configuration System

### Configuration Hierarchy

```
1. Built-in Defaults (lowest priority)
   ↓
2. Config File (llm-runtime.config.yaml)
   ↓
3. Environment Variables
   ↓
4. CLI Flags (highest priority)
```

### Configuration Structure

```yaml
# Logging
logging:
  level: "info"
  format: "text"

# Audit
audit:
  enabled: true
  file: "audit.log"

# Search
search:
  enabled: true
  database_path: "embeddings.db"
  ollama_url: "http://localhost:11434"
  ollama_model: "nomic-embed-text"

# Commands
commands:
  exec:
    container_image: "python-go"
    timeout_seconds: 30
    memory_limit: "512m"
    cpu_limit: 2
    whitelist: [...]
  
  io:
    container_image: "llm-runtime-io:latest"
    timeout_seconds: 10
    memory_limit: "128m"
    cpu_limit: 1
```

## Error Handling

### Error Types

```go
// Command errors
ErrCommandNotFound
ErrInvalidSyntax
ErrCommandTimeout

// Path errors
ErrPathOutsideRepo
ErrPathExcluded
ErrFileNotFound
ErrPermissionDenied

// Exec errors
ErrCommandNotWhitelisted
ErrDockerUnavailable
ErrContainerFailed

// Search errors
ErrOllamaUnavailable
ErrIndexNotFound
ErrEmbeddingFailed
```

### Error Response Format

```
=== ERROR: ERROR_TYPE ===
Message: Human-readable description
Details: Additional context
Command: Original command that failed
=== END ERROR ===
```

## Testing Strategy

### Unit Tests
- Individual component testing
- Mock Docker interactions
- Path validation edge cases
- Command parsing variations

### Integration Tests
- Full command flow testing
- Docker container integration
- Ollama integration
- File system operations

### Security Tests
- Path traversal attempts
- Privilege escalation tests
- Resource exhaustion tests
- Container escape attempts

### Performance Tests
- Container startup benchmarks
- Search query performance
- Large file handling
- Concurrent operation tests

## Extensibility

### Adding New Commands

1. Define command pattern in parser
2. Implement handler interface
3. Add to command router
4. Update documentation
5. Add tests

**Example:**
```go
// 1. Add pattern
patterns["git"] = regexp.MustCompile(`<git ([^>]+)>`)

// 2. Implement handler
type GitHandler struct {
    // ...
}

func (h *GitHandler) Handle(cmd Command) (*Result, error) {
    // Implementation
}

// 3. Register in router
router.Register("git", &GitHandler{})
```

### Adding Container Images

1. Create Dockerfile
2. Build and tag image
3. Update configuration
4. Test with commands
5. Document usage

### Adding Search Backends

1. Implement `SearchBackend` interface
2. Add configuration options
3. Update search handler
4. Test integration
5. Document setup

## Dependencies

### Core Dependencies
- **Docker**: Container runtime
- **Go**: Programming language (1.21+)
- **SQLite**: Search index storage
- **Ollama**: Embedding generation

### Optional Dependencies
- **Git**: Version control (for repos)
- **Make**: Build automation

### Docker Images
- **llm-runtime-io:latest**: Minimal Alpine for I/O (Phase 5)
- **python-go**: Python + Go environment for exec
- **alpine:latest**: Fallback for I/O operations

## Deployment Patterns

### Local Development
```bash
./llm-runtime --config dev.yaml
```

### CI/CD Integration
```yaml
- name: Run LLM Tests
  run: |
    docker pull llm-runtime-io:latest
    docker pull python-go:latest
    echo "<exec go test ./...>" | ./llm-runtime
```

### Server Deployment
```bash
# Run as service
./llm-runtime --config production.yaml --log-level info
```

## Monitoring & Observability

### Metrics (Future)
- Command execution counts
- Container startup times
- Search query latency
- Error rates
- Resource utilization

### Logging
- Structured JSON logging
- Audit trail
- Error tracking
- Performance metrics

### Health Checks
- Docker daemon availability
- Ollama service status
- Disk space monitoring
- Index integrity

---

This architecture provides a secure, extensible foundation for LLM-repository interaction while maintaining strong isolation and comprehensive auditability. Phase 5's I/O containerization represents the culmination of the defense-in-depth approach, with all operations—reads, writes, and execution—now fully containerized.

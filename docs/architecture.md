# LLM-Runtime Architecture

## Executive Summary

This document describes the architecture of `llm-runtime`, a command interpreter that enables Large Language Models to interact with local filesystems and execute sandboxed commands. The design draws from established principles in interpreter construction:

1. **Single Code Path** — One unified pipeline for all input modes
2. **State Machine Parsing** — Explicit state transitions for command detection
3. **Separation of Concerns** — Clear boundaries between scanning, parsing, and evaluation
4. **Defense in Depth** — Sandboxing at every layer, not just at execution time

---

## 1. Foundational Principles

### 1.1 The Read-Eval-Print Model

Classical interpreters follow the REPL (Read-Eval-Print-Loop) pattern from Lisp systems:

```
┌─────────────────────────────────────────────────────────┐
│                        REPL LOOP                        │
│                                                         │
│   ┌──────┐    ┌───────┐    ┌──────┐    ┌───────┐       │
│   │ READ │───▶│ PARSE │───▶│ EVAL │───▶│ PRINT │──┐    │
│   └──────┘    └───────┘    └──────┘    └───────┘  │    │
│       ▲                                           │    │
│       └───────────────────────────────────────────┘    │
│                                                         │
└─────────────────────────────────────────────────────────┘
```

A single `Scanner` processes input regardless of source (pipe or interactive), emitting commands one at a time for evaluation.

### 1.2 The Phases of Interpretation

Following Aho, Sethi, and Ullman's compiler design principles, we decompose interpretation into distinct phases:

```
┌─────────────────────────────────────────────────────────────────┐
│                     INTERPRETATION PHASES                        │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  Source    ┌─────────┐   Tokens   ┌────────┐   AST    ┌──────┐ │
│  Text  ───▶│ SCANNER │──────────▶│ PARSER │────────▶│ EVAL │  │
│            │ (Lexer) │            │        │          │      │  │
│            └─────────┘            └────────┘          └──────┘  │
│                 │                      │                  │      │
│                 ▼                      ▼                  ▼      │
│            Character             Command             Execution   │
│            Stream                Structure            Result     │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

| Phase | Responsibility | Implementation |
|-------|---------------|----------------|
| **Scanning** | Character-by-character input, state machine | `pkg/scanner/scanner.go` |
| **Parsing** | Command extraction, argument parsing | `pkg/scanner/parser.go` |
| **Evaluation** | Command execution, side effects | `pkg/evaluator/*.go` |
| **Sandboxing** | Security validation, Docker isolation | `pkg/sandbox/*.go` |

---

## 2. Architecture

### 2.1 Component Diagram

```
┌──────────────────────────────────────────────────────────────────────────┐
│                              llm-runtime                                  │
├──────────────────────────────────────────────────────────────────────────┤
│                                                                           │
│  ┌─────────────┐     ┌─────────────┐     ┌─────────────┐                 │
│  │   INPUT     │     │   CORE      │     │   OUTPUT    │                 │
│  │   LAYER     │     │   ENGINE    │     │   LAYER     │                 │
│  ├─────────────┤     ├─────────────┤     ├─────────────┤                 │
│  │ • stdin     │────▶│ • Scanner   │────▶│ • stdout    │                 │
│  │ • file      │     │ • Parser    │     │ • file      │                 │
│  │             │     │ • Evaluator │     │             │                 │
│  └─────────────┘     └──────┬──────┘     └─────────────┘                 │
│                             │                                             │
│                             ▼                                             │
│  ┌───────────────────────────────────────────────────────────────────┐   │
│  │                        SANDBOX LAYER                               │   │
│  ├───────────────────────────────────────────────────────────────────┤   │
│  │  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────────────┐   │   │
│  │  │ OpenCmd  │  │ WriteCmd │  │ ExecCmd  │  │    SearchCmd     │   │   │
│  │  │ (read)   │  │ (write)  │  │ (docker) │  │    (ollama)      │   │   │
│  │  └────┬─────┘  └────┬─────┘  └────┬─────┘  └────────┬─────────┘   │   │
│  │       │             │             │                  │             │   │
│  │       ▼             ▼             ▼                  ▼             │   │
│  │  ┌──────────────────────────────────────────────────────────────┐ │   │
│  │  │                    SECURITY BOUNDARY                          │ │   │
│  │  │  • Path validation    • Extension whitelist                   │ │   │
│  │  │  • Command whitelist  • Resource limits                       │ │   │
│  │  │  • Container isolation • Audit logging                        │ │   │
│  │  └──────────────────────────────────────────────────────────────┘ │   │
│  └───────────────────────────────────────────────────────────────────┘   │
│                                                                           │
└──────────────────────────────────────────────────────────────────────────┘
```

### 2.2 Package Structure

```
llm-runtime/
├── cmd/llm-runtime/           # Entry point
│   └── main.go
│
├── pkg/                       # Public API (importable)
│   ├── scanner/               # Command detection and parsing
│   │   ├── scanner.go         # State machine scanner
│   │   ├── parser.go          # Command extraction (regex-based)
│   │   └── types.go           # Command type definitions
│   │
│   ├── evaluator/             # Command execution
│   │   ├── executor.go        # Command dispatch
│   │   ├── open.go            # <open> implementation
│   │   ├── write.go           # <write> implementation
│   │   ├── exec.go            # <exec> implementation
│   │   └── search.go          # <search> implementation
│   │
│   └── sandbox/               # Security and isolation
│       ├── path.go            # Path validation
│       ├── extension.go       # Extension whitelist
│       ├── exec_validation.go # Command whitelist
│       ├── audit.go           # Audit logging
│       ├── client.go          # Docker client
│       └── container.go       # Container management
│
├── internal/                  # Private implementation
│   ├── app/                   # Application bootstrap
│   ├── cli/                   # Command-line handling
│   ├── config/                # Configuration loading
│   ├── infrastructure/        # Database, filesystem
│   ├── search/                # Semantic search (Ollama integration)
│   └── session/               # Session management
│
└── docs/                      # Documentation
```

---

## 3. The Scanner (State Machine Design)

### 3.1 Why State Machines Over Regex

The core insight: **regex-based parsing is fragile for nested or context-sensitive structures**.

**The Chomsky Hierarchy** classifies formal languages by computational power:

| Type | Name | Recognizer | Example |
|------|------|------------|---------|
| Type-3 | Regular | Finite Automaton (Regex) | `<open file.txt>` |
| Type-2 | Context-Free | Pushdown Automaton | `<write f>nested</write>` |
| Type-1 | Context-Sensitive | Linear Bounded Automaton | Natural language |
| Type-0 | Recursively Enumerable | Turing Machine | Any computable |

**Problem**: The `<write>` command requires *context-sensitive* parsing because we must track nested content until we see `</write>`. Regular expressions (Type-3) cannot handle this—they lack the "memory" to match arbitrary nesting.

**Example of regex failure:**
```go
// This regex CANNOT correctly handle:
// <write file.txt>content with </write> inside</write>
writeRegex := regexp.MustCompile(`<write\s+([^>]+)>(.*?)</write>`)
// The (.*?) will match minimally, breaking on the first </write>
```

**Solution**: A state machine maintains explicit state, enabling correct handling:

```go
case StateInWriteBody:
    s.buffer.WriteString(line)
    if strings.Contains(line, "</write>") {
        // Only transition when we see the closing tag
        s.state = StateText
        return s.extractWriteCommand()
    }
```

### 3.2 Scanner State Machine

```
┌─────────────────────────────────────────────────────────────────┐
│                     SCANNER STATE MACHINE                        │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│                        ┌──────────┐                              │
│            ┌──────────▶│ SCANNING │◀──────────┐                  │
│            │           └────┬─────┘           │                  │
│            │                │                 │                  │
│            │    see '<'     │                 │                  │
│            │                ▼                 │                  │
│            │           ┌──────────┐           │                  │
│            │           │ TAG_OPEN │           │                  │
│            │           └────┬─────┘           │                  │
│            │                │                 │                  │
│            │    match tag   │    no match     │                  │
│            │       type     │                 │                  │
│            │    ┌───────────┴────────┐        │                  │
│            │    ▼                    ▼        │                  │
│            │ ┌──────┐  ┌───────┐  ┌──────┐   │                  │
│            │ │ OPEN │  │ WRITE │  │ EXEC │   │                  │
│            │ └──┬───┘  └───┬───┘  └──┬───┘   │                  │
│            │    │          │         │        │                  │
│            │    │   ┌──────┴──────┐  │        │                  │
│            │    │   ▼             │  │        │                  │
│            │    │ ┌───────────┐   │  │        │                  │
│            │    │ │WRITE_BODY │   │  │        │                  │
│            │    │ └─────┬─────┘   │  │        │                  │
│            │    │       │         │  │        │                  │
│            │    │  </write>       │  │        │                  │
│            │    │       │         │  │        │                  │
│            │    ▼       ▼         ▼  ▼        │                  │
│            │  ┌─────────────────────────┐     │                  │
│            │  │       COMPLETE          │     │                  │
│            │  └───────────┬─────────────┘     │                  │
│            │              │                   │                  │
│            └──────────────┴───────────────────┘                  │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

**Scanner States:**
- `StateText` — Default state, scanning for command start
- `StateInOpen` — Inside `<open ...>` command
- `StateInWrite` — Inside `<write ...>` tag, before content
- `StateInWriteBody` — Accumulating write content until `</write>`
- `StateInExec` — Inside `<exec ...>` command
- `StateInSearch` — Inside `<search ...>` command

### 3.3 Command Detection: HasPrefix vs Contains

**Anti-pattern** (breaks on mid-line commands):
```go
// WRONG: matches "<open" anywhere in the line
if strings.Contains(line, "<open") { ... }
```

**Correct approach**:
```go
// RIGHT: only matches commands at line start (after whitespace)
func isCommandStart(line string) bool {
    trimmed := strings.TrimLeft(line, " \t")
    return strings.HasPrefix(trimmed, "<open") ||
           strings.HasPrefix(trimmed, "<write") ||
           strings.HasPrefix(trimmed, "<exec") ||
           strings.HasPrefix(trimmed, "<search")
}
```

### 3.4 Command Grammar (BNF)

```bnf
<input>       ::= <segment>*
<segment>     ::= <text> | <command>
<text>        ::= (any character except '<')+
<command>     ::= <open-cmd> | <write-cmd> | <exec-cmd> | <search-cmd>

<open-cmd>    ::= '<open' <ws> <filepath> '>'
<write-cmd>   ::= '<write' <ws> <filepath> '>' <content> '</write>'
<exec-cmd>    ::= '<exec' <ws> <shell-cmd> '>'
<search-cmd>  ::= '<search' <ws> <query> '>'

<filepath>    ::= <path-char>+
<shell-cmd>   ::= [^\>]+
<query>       ::= [^\>]+
<content>     ::= (any character except '</write>')*

<ws>          ::= (' ' | '\t' | '\n')+
<path-char>   ::= [a-zA-Z0-9_./-]
```

### 3.5 Command Ordering

Commands are executed in the order they appear in the input. The parser tracks `StartPos` for each command and sorts before returning:

```go
sort.Slice(commands, func(i, j int) bool {
    return commands[i].StartPos < commands[j].StartPos
})
```

This ensures `<open a.go><exec go build><open b.go>` executes in that exact order.

---

## 4. Command Execution Flow

### 4.1 Execution Pipeline

```
┌─────────────────────────────────────────────────────────────────┐
│                     COMMAND EXECUTION FLOW                       │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  Command                                                         │
│     │                                                            │
│     ▼                                                            │
│  ┌──────────────────┐                                           │
│  │ 1. VALIDATE      │  Security checks before any I/O           │
│  │    - Path safety │                                           │
│  │    - Whitelist   │                                           │
│  │    - Size limits │                                           │
│  └────────┬─────────┘                                           │
│           │                                                      │
│           ▼                                                      │
│  ┌──────────────────┐                                           │
│  │ 2. SANDBOX       │  Establish execution boundary             │
│  │    - Container?  │                                           │
│  │    - Direct?     │                                           │
│  └────────┬─────────┘                                           │
│           │                                                      │
│           ▼                                                      │
│  ┌──────────────────┐                                           │
│  │ 3. EXECUTE       │  Perform the operation                    │
│  │    - Read file   │                                           │
│  │    - Write file  │                                           │
│  │    - Run command │                                           │
│  │    - Query Ollama│                                           │
│  └────────┬─────────┘                                           │
│           │                                                      │
│           ▼                                                      │
│  ┌──────────────────┐                                           │
│  │ 4. AUDIT         │  Log result for security review           │
│  │    - Timestamp   │                                           │
│  │    - Session ID  │                                           │
│  │    - Outcome     │                                           │
│  └────────┬─────────┘                                           │
│           │                                                      │
│           ▼                                                      │
│     Result                                                       │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

### 4.2 Sandboxing Strategy (Defense in Depth)

```
┌─────────────────────────────────────────────────────────────────┐
│                      SANDBOXING LAYERS                           │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  LAYER 1: Input Validation (Host)                               │
│  ├─ Path canonicalization (resolve symlinks, ../)               │
│  ├─ Traversal detection (block ../../etc/passwd)                │
│  └─ Extension whitelist (block .exe, .key, .pem)                │
│                                                                  │
│  LAYER 2: Resource Limits (Host + Container)                    │
│  ├─ File size limits (default: 1MB read, 100KB write)           │
│  ├─ Memory limits (--memory 512m)                               │
│  ├─ CPU limits (--cpus 2)                                       │
│  └─ Time limits (context.WithTimeout, default 30s)              │
│                                                                  │
│  LAYER 3: Container Isolation (exec commands only)              │
│  ├─ No network (--network none)                                  │
│  ├─ Read-only root filesystem (--read-only)                      │
│  ├─ Dropped capabilities (--cap-drop ALL)                        │
│  ├─ Non-root user (--user 1000:1000)                             │
│  └─ No privilege escalation (--security-opt no-new-privileges)   │
│                                                                  │
│  LAYER 4: Audit Trail (Host)                                    │
│  ├─ All operations logged with timestamps                        │
│  ├─ Session tracking for correlation                             │
│  └─ Structured JSON format for analysis                          │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

**Key principle**: Even if one layer fails, others provide protection. A bug in path validation doesn't compromise the system if Docker isolation catches the attack.

---

## 5. Search Architecture

Semantic search uses Ollama for local embedding generation—no external API calls, all processing on your machine.

```
┌─────────────────────────────────────────────────────────────────┐
│                      SEARCH ARCHITECTURE                         │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  ┌──────────┐     ┌──────────┐     ┌──────────────────────┐     │
│  │  Query   │────▶│  Ollama  │────▶│  Query Embedding     │     │
│  │          │     │  API     │     │  (nomic-embed-text)  │     │
│  └──────────┘     └──────────┘     └──────────┬───────────┘     │
│                                                │                 │
│                                                ▼                 │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │                    SQLite Database                        │   │
│  │  ┌─────────────────────────────────────────────────────┐ │   │
│  │  │  embeddings table                                    │ │   │
│  │  │  - filepath (PK)                                     │ │   │
│  │  │  - content_hash                                      │ │   │
│  │  │  - embedding (BLOB)                                  │ │   │
│  │  │  - last_modified                                     │ │   │
│  │  └─────────────────────────────────────────────────────┘ │   │
│  └──────────────────────────────────────────────────────────┘   │
│                                                │                 │
│                                                ▼                 │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │              Cosine Similarity Ranking                    │   │
│  │         similarity = (A · B) / (||A|| × ||B||)           │   │
│  └──────────────────────────────────────────────────────────┘   │
│                                                │                 │
│                                                ▼                 │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │              Top-K Results (default: 10)                  │   │
│  └──────────────────────────────────────────────────────────┘   │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

**Components:**
- **Ollama** — Local LLM server providing `nomic-embed-text` embeddings
- **SQLite** — Stores file embeddings for fast similarity search
- **Indexer** — Walks repository, generates embeddings for each file

**Why local embeddings?**
- Privacy: Code never leaves your machine
- Speed: No network latency after initial model load
- Cost: No API fees
- Offline: Works without internet

---

## 6. Error Handling Strategy

### 6.1 Error Categories

```go
type ErrorCategory string

const (
    ErrSyntax     ErrorCategory = "SYNTAX"      // Malformed command
    ErrValidation ErrorCategory = "VALIDATION"  // Security check failed
    ErrExecution  ErrorCategory = "EXECUTION"   // Runtime failure
    ErrResource   ErrorCategory = "RESOURCE"    // Limit exceeded
    ErrInternal   ErrorCategory = "INTERNAL"    // Bug in llm-runtime
)
```

| Category | Code Examples | Description |
|----------|---------------|-------------|
| Syntax | `SYNTAX_INVALID_COMMAND` | Malformed command |
| Validation | `PATH_SECURITY`, `EXEC_VALIDATION` | Security check failed |
| Execution | `EXEC_FAILED`, `FILE_NOT_FOUND` | Runtime failure |
| Resource | `RESOURCE_LIMIT`, `EXEC_TIMEOUT` | Limit exceeded |

### 6.2 Error Recovery

The interpreter is **resilient**—a single command failure does not crash the session:

```go
for _, cmd := range commands {
    result := executor.Execute(cmd)
    if result.Error != nil {
        printError(result.Error)
        continue  // Keep processing remaining commands
    }
    printResult(result)
}
```

This follows the **robustness principle**: "Be conservative in what you send, be liberal in what you accept."

---

## 7. Testing Strategy

### 7.1 Test Categories

| Category | Purpose | Location |
|----------|---------|----------|
| **Unit** | Test individual functions | `*_test.go` next to source |
| **Integration** | Test component interaction | `tests/integration/` |
| **Security** | Test sandbox boundaries | `tests/security/` |

### 7.2 Security Test Cases

Path traversal attacks that must be blocked:

```go
func TestPathTraversal(t *testing.T) {
    attacks := []string{
        "../../../etc/passwd",
        "..\\..\\..\\windows\\system32\\config\\sam",
        "foo/../../../etc/passwd",
        "/etc/passwd",
        "~/.ssh/id_rsa",
        "file:///etc/passwd",
    }
    
    for _, path := range attacks {
        _, err := security.ValidatePath(path, "/repo", nil)
        if err == nil {
            t.Errorf("path traversal not blocked: %s", path)
        }
    }
}
```

### 7.3 Scanner Test Cases

```go
func TestScanner(t *testing.T) {
    cases := []struct {
        name     string
        input    string
        expected []Command
    }{
        {
            name:  "simple open",
            input: "<open README.md>",
            expected: []Command{{Type: "open", Argument: "README.md"}},
        },
        {
            name:  "write with content",
            input: "<write test.txt>Hello, World!</write>",
            expected: []Command{{Type: "write", Argument: "test.txt", Content: "Hello, World!"}},
        },
        {
            name:  "multiple commands preserve order",
            input: "<open a.go><exec go build><open b.go>",
            expected: []Command{
                {Type: "open", Argument: "a.go"},
                {Type: "exec", Argument: "go build"},
                {Type: "open", Argument: "b.go"},
            },
        },
    }
    // ...
}
```

---

## 8. Performance Considerations

### 8.1 Bottlenecks

| Operation | Typical Time | Notes |
|-----------|--------------|-------|
| Command parsing | <1ms | Already fast |
| Path validation | <1ms | Already fast |
| File read (1MB) | <10ms | SSD dependent |
| Docker startup | 1-3s | Main bottleneck for exec |
| Ollama embedding | 100-500ms | First query slower (model load) |
| Similarity search | <1ms per file | SQLite is fast |

### 8.2 Optimization Opportunities

1. **Container Pool**: Pre-warm Docker containers to eliminate startup latency
2. **Embedding Cache**: Already implemented via SQLite
3. **Streaming Output**: For large command outputs, stream rather than buffer
4. **Batch Commands**: Combine multiple exec commands: `<exec cmd1 && cmd2>`

---

## 9. Configuration

Configuration is loaded from `llm-runtime.config.yaml`:

```yaml
repository:
  root: "."
  excluded_paths: [".git", ".env", "*.key"]

commands:
  open:
    enabled: true
    max_file_size: 1048576      # 1MB
  write:
    enabled: true
    backup_before_write: true
    max_file_size: 102400       # 100KB
  exec:
    enabled: false
    timeout_seconds: 30
    memory_limit: "512m"
    whitelist: ["go test", "go build", "npm test"]
  search:
    enabled: true
    vector_db_path: "./embeddings.db"
```

---

## 10. Quick Reference

### 10.1 Command Syntax

```
<open filepath>                    Read file contents
<write filepath>content</write>    Create or update file
<exec command args>                Execute in container
<search query terms>               Semantic search
```

### 10.2 CLI Flags

```
--root PATH          Repository root (default: .)
--interactive        Enable prompts (default: false)
--exec-enabled       Enable <exec> command (default: false)
--reindex            Rebuild search index
--verbose            Verbose output (default: false)
```

### 10.3 Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | General error |
| 2 | Configuration error |
| 124 | Command timeout |

---

## 11. Future Considerations

### 11.1 Potential Enhancements

| Enhancement | Description | Priority |
|-------------|-------------|----------|
| Container pool | Pre-warm Docker containers for faster exec | Medium |
| Streaming output | Stream large command outputs | Low |
| MCP integration | Model Context Protocol support | Medium |
| Additional commands | `<git>`, `<diff>`, `<tree>` | Low |

### 11.2 Design Principles for Extensions

When extending the system:

1. **Security first** — All new commands must go through validation
2. **Single code path** — Avoid separate implementations for different modes
3. **Fail gracefully** — Errors should not crash the interpreter
4. **Audit everything** — All operations must be logged
5. **State machine for parsing** — Don't rely on regex for complex structures

---

## 12. References

1. Abelson, H. & Sussman, G.J. (1996). *Structure and Interpretation of Computer Programs* (2nd ed.). MIT Press. — The REPL model and interpreter design.

2. Aho, A.V., Sethi, R., & Ullman, J.D. (1986). *Compilers: Principles, Techniques, and Tools*. Addison-Wesley. — Lexical analysis, parsing phases, the Chomsky hierarchy.

3. Pike, R. (2011). "Lexical Scanning in Go." https://go.dev/talks/2011/lex.slide — State machine design for lexers in Go.

4. Docker Security Best Practices. https://docs.docker.com/engine/security/ — Container isolation techniques.

5. Postel, J. (1980). RFC 761: "Robustness Principle" — "Be conservative in what you send, be liberal in what you accept."

---

*Document version: 2.0*  


# LLM-Runtime Architecture


## Executive Summary

This document defines the target architecture for `llm-runtime`, a command interpreter that enables Large Language Models to interact with local filesystems and execute sandboxed commands. The design draws from established principles in interpreter construction, emphasizing:

1. **Single Code Path** — One unified pipeline for all input modes
2. **State Machine Parsing** — Explicit state transitions replace ad-hoc buffer manipulation
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

**Current Problem:** The existing codebase has *two* code paths—`InteractiveMode()` and `processPipeMode()`—that implement this loop differently, leading to:
- Duplicated logic
- Inconsistent behavior (e.g., `<write>` works in pipe mode but not interactive)
- Command reordering bugs in `ParseCommands()`

**Solution:** Unify into a single `ScanInput()` function that processes one command at a time, regardless of input source.

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

| Phase | Responsibility | Current Code | Target Code |
|-------|---------------|--------------|-------------|
| **Scanning** | Character-by-character input, tokenization | `bufio.Scanner` in `InteractiveMode()` | `Scanner` struct with state machine |
| **Parsing** | Structure recognition, command extraction | `ParseCommands()` with regex | `Parser` with grammar-based extraction |
| **Evaluation** | Command execution, side effects | `Executor.Execute()` | `Evaluator` with sandbox delegation |
| **Printing** | Result formatting, output | Scattered `fmt.Print` calls | `Printer` with configurable formats |

---

## 2. Target Architecture

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
│  │ • (future:  │     │ • Evaluator │     │ • JSON      │                 │
│  │    socket)  │     │ • Printer   │     │             │                 │
│  └─────────────┘     └──────┬──────┘     └─────────────┘                 │
│                             │                                             │
│                             ▼                                             │
│  ┌───────────────────────────────────────────────────────────────────┐   │
│  │                        SANDBOX LAYER                               │   │
│  ├───────────────────────────────────────────────────────────────────┤   │
│  │  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────────────┐   │   │
│  │  │ OpenCmd  │  │ WriteCmd │  │ ExecCmd  │  │    SearchCmd     │   │   │
│  │  │ (read)   │  │ (write)  │  │ (docker) │  │   (embeddings)   │   │   │
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

### 2.2 The Unified Scanner (State Machine Design)

The core insight from Steve's review: **the scanner loop needs to be replaced with a state machine loop**.

#### 2.2.1 Scanner States

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
│            │  │       EXECUTE           │     │                  │
│            │  └───────────┬─────────────┘     │                  │
│            │              │                   │                  │
│            └──────────────┴───────────────────┘                  │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

#### 2.2.2 State Definition (Target Implementation)

```go
// ScannerState represents the current parsing state
type ScannerState int

const (
    StateScanning   ScannerState = iota  // Default: scanning for commands
    StateTagOpen                          // Saw '<', determining tag type
    StateOpen                             // Parsing <open filepath>
    StateWrite                            // Parsing <write filepath>
    StateWriteBody                        // Accumulating write content
    StateExec                             // Parsing <exec command>
    StateSearch                           // Parsing <search query>
    StateExecute                          // Ready to execute command
)

// Scanner implements a state-machine based input processor
type Scanner struct {
    state       ScannerState
    buffer      strings.Builder
    currentCmd  *Command
    reader      *bufio.Reader
    
    // Configuration
    showPrompt  bool  // True for interactive mode
}
```

#### 2.2.3 Why This Matters

**Current bug example** (from code review): In `InteractiveMode()`, the code uses `strings.Contains()` to detect commands:

```go
// CURRENT (BROKEN)
if strings.Contains(line, "<open") || strings.Contains(line, "<write") ...
```

This fails when:
- A command appears mid-line after other text
- Multiple commands appear on one line
- The string `<open` appears in a comment or string literal

**State machine approach:**

```go
// TARGET (CORRECT)
func (s *Scanner) ProcessByte(b byte) (Command, bool) {
    switch s.state {
    case StateScanning:
        if b == '<' {
            s.state = StateTagOpen
            s.buffer.WriteByte(b)
        } else {
            // Pass through non-command text
            s.emitText(b)
        }
    case StateTagOpen:
        s.buffer.WriteByte(b)
        if s.matchesTagPrefix() {
            s.transitionToTagState()
        }
    // ... other states
    }
}
```

### 2.3 Command Grammar

Following formal language theory, we define the command grammar in BNF:

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
<shell-cmd>   ::= <cmd-char>+
<query>       ::= <query-char>+
<content>     ::= (any character except '</write>')*

<ws>          ::= (' ' | '\t' | '\n')+
<path-char>   ::= [a-zA-Z0-9_./-]
<cmd-char>    ::= [^\>]
<query-char>  ::= [^\>]
```

**Key insight:** The `<write>` command requires *context-sensitive* parsing because we must track nested content until we see `</write>`. This is why regex-based parsing is fragile—regular expressions cannot handle nested structures (Chomsky hierarchy: Type-3 vs Type-2 languages).

### 2.4 Evaluation Model

#### 2.4.1 Command Execution Flow

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
│  │    - Chroot?     │                                           │
│  │    - Direct?     │                                           │
│  └────────┬─────────┘                                           │
│           │                                                      │
│           ▼                                                      │
│  ┌──────────────────┐                                           │
│  │ 3. EXECUTE       │  Perform the operation                    │
│  │    - Read file   │                                           │
│  │    - Write file  │                                           │
│  │    - Run command │                                           │
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

#### 2.4.2 Sandboxing Strategy

**Principle:** All file operations should eventually execute inside the container, not on the host.

```
┌─────────────────────────────────────────────────────────────────┐
│                      SANDBOXING LAYERS                           │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  LAYER 1: Input Validation (Host)                               │
│  ├─ Path canonicalization                                        │
│  ├─ Traversal detection                                          │
│  └─ Extension whitelist                                          │
│                                                                  │
│  LAYER 2: Resource Limits (Host + Container)                    │
│  ├─ File size limits                                             │
│  ├─ Memory limits (--memory)                                     │
│  ├─ CPU limits (--cpus)                                          │
│  └─ Time limits (context.WithTimeout)                            │
│                                                                  │
│  LAYER 3: Isolation (Container)                                 │
│  ├─ No network (--network none)                                  │
│  ├─ Read-only root filesystem (--read-only)                      │
│  ├─ Dropped capabilities (--cap-drop ALL)                        │
│  ├─ Non-root user (--user 1000:1000)                             │
│  └─ No privilege escalation (--security-opt no-new-privileges)   │
│                                                                  │
│  LAYER 4: Audit (Host)                                          │
│  ├─ All operations logged                                        │
│  ├─ Session tracking                                             │
│  └─ Tamper-evident log format                                    │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

**Current gap:** `<open>` and `<write>` execute on the host. Target: move these inside the container for defense-in-depth.

---

## 3. Refactoring Roadmap

### 3.1 Phase 1: Unify Code Paths (Priority: HIGH)

**Goal:** Eliminate `processPipeMode()` and merge into `ScanInput()`.

```
BEFORE:                              AFTER:
┌─────────────────────┐              ┌─────────────────────┐
│ main()              │              │ main()              │
│   │                 │              │   │                 │
│   ├─▶ Interactive?  │              │   └─▶ ScanInput()   │
│   │   │             │              │         │           │
│   │   ├─▶ YES ──▶ InteractiveMode()│         ├─▶ Scanner │
│   │   │             │              │         │   (state  │
│   │   └─▶ NO ───▶ processPipeMode()│         │   machine)│
│   │                 │              │         │           │
└───┴─────────────────┘              │         └─▶ Process │
                                     │             one cmd │
                                     │             at a    │
                                     │             time    │
                                     └─────────────────────┘
```

**Changes required:**
1. Create `Scanner` struct with state machine
2. `--interactive` flag only controls prompt output
3. Remove `processPipeMode()` entirely
4. `ParseCommands()` returns single command (not slice)

### 3.2 Phase 2: Fix Command Matching (Priority: HIGH)

**Goal:** Replace `strings.Contains()` with `strings.HasPrefix()` and proper state transitions.

```go
// BEFORE (line 25-26 in interactive.go)
if strings.Contains(line, "<open") || strings.Contains(line, "<write") ||
   strings.Contains(line, "<exec") || strings.Contains(line, "<search") {

// AFTER
func (s *Scanner) isCommandStart(line string) bool {
    trimmed := strings.TrimLeft(line, " \t")
    return strings.HasPrefix(trimmed, "<open") ||
           strings.HasPrefix(trimmed, "<write") ||
           strings.HasPrefix(trimmed, "<exec") ||
           strings.HasPrefix(trimmed, "<search")
}
```

### 3.3 Phase 3: Process One Command at a Time (Priority: MEDIUM)

**Goal:** Simplify `ProcessText()` and `ProcessCommands()` to handle single commands.

**Current call tree:**
```
ProcessText()
  └─▶ ParseCommands()      # Returns []Command (ALL commands)
        └─▶ for each cmd:
              └─▶ Execute()
```

**Target call tree:**
```
ScanInput()
  └─▶ Scanner.Next()       # Returns ONE Command
        └─▶ Validate()
        └─▶ Execute()
        └─▶ Print()
        └─▶ (loop)
```

### 3.4 Phase 4: Elevate Write Regex (Priority: MEDIUM)

**Problem:** The regex that correctly handles `<write>` is buried in `ParseCommands()`, but the scanner loop doesn't know about write state.

**Solution:** Move write content accumulation into the state machine:

```go
case StateWrite:
    // We've seen "<write filepath>"
    // Now accumulate until "</write>"
    if s.buffer.String() ends with "</write>" {
        s.currentCmd.Content = extractContent(s.buffer.String())
        s.state = StateExecute
    }
```

### 3.5 Phase 5: Containerize All I/O (Priority: LOW)

**Goal:** Execute `<open>` and `<write>` inside Docker container.

**Rationale:** Even with path validation, a bug in the host code could allow escape. Running inside the container provides defense-in-depth.

**Implementation sketch:**
```go
func (e *Executor) ExecuteOpen(filepath string) Result {
    // Instead of os.ReadFile() on host:
    return docker.RunContainer(ContainerConfig{
        Command: fmt.Sprintf("cat %q", filepath),
        RepoRoot: e.config.RepositoryRoot,
        // ... security options
    })
}
```

### 3.6 Phase 6: Replace Python Embeddings (Priority: LOW)

**Goal:** Use pure Go for vector embeddings to eliminate Python dependency.

**Options:**
| Library | Pros | Cons |
|---------|------|------|
| `go-embeddings` | Direct API to OpenAI/etc | External API dependency |
| `kelindar/search` | Pure Go, no external deps | Less sophisticated models |
| Ollama | Local inference, good models | Requires Ollama daemon |

**Recommendation:** Start with Ollama for local development, with fallback to `go-embeddings` for production.

---

## 4. Package Structure (Target)

```
llm-runtime/
├── cmd/
│   └── llm-runtime/
│       └── main.go              # Entry point only
│
├── pkg/                         # Public API (importable)
│   ├── scanner/
│   │   ├── scanner.go           # State machine scanner
│   │   ├── states.go            # State definitions
│   │   └── scanner_test.go
│   │
│   ├── parser/
│   │   ├── parser.go            # Command parsing
│   │   ├── grammar.go           # Grammar definitions
│   │   └── parser_test.go
│   │
│   ├── evaluator/
│   │   ├── evaluator.go         # Command dispatch
│   │   ├── open.go              # <open> implementation
│   │   ├── write.go             # <write> implementation
│   │   ├── exec.go              # <exec> implementation
│   │   ├── search.go            # <search> implementation
│   │   └── evaluator_test.go
│   │
│   └── sandbox/
│       ├── docker.go            # Docker container management
│       ├── validation.go        # Security validation
│       └── sandbox_test.go
│
├── internal/                    # Private implementation
│   ├── config/                  # Configuration loading
│   ├── audit/                   # Audit logging
│   └── session/                 # Session management
│
└── docs/
    ├── architecture.md          # This document
    └── SYSTEM_PROMPT.md         # LLM integration guide
```

**Key change:** Move reusable components from `internal/` to `pkg/` so other projects can import them. Use lowercase symbols for truly private items.

---

## 5. Error Handling Strategy

### 5.1 Error Categories

```go
type ErrorCategory string

const (
    ErrSyntax     ErrorCategory = "SYNTAX"      // Malformed command
    ErrValidation ErrorCategory = "VALIDATION"  // Security check failed
    ErrExecution  ErrorCategory = "EXECUTION"   // Runtime failure
    ErrResource   ErrorCategory = "RESOURCE"    // Limit exceeded
    ErrInternal   ErrorCategory = "INTERNAL"    // Bug in llm-runtime
)

type RuntimeError struct {
    Category ErrorCategory
    Code     string
    Message  string
    Cause    error
    Context  map[string]string
}
```

### 5.2 Error Recovery

The interpreter should be **resilient**—a single command failure should not crash the session:

```go
func (s *Scanner) Run() {
    for {
        cmd, err := s.Next()
        if err == io.EOF {
            break
        }
        if err != nil {
            s.printer.PrintError(err)
            continue  // Keep processing
        }
        
        result := s.evaluator.Execute(cmd)
        s.printer.PrintResult(result)
    }
}
```

---

## 6. Testing Strategy

### 6.1 Test Categories

| Category | Purpose | Location |
|----------|---------|----------|
| **Unit** | Test individual functions | `*_test.go` next to source |
| **Integration** | Test component interaction | `tests/integration/` |
| **Security** | Test sandbox boundaries | `tests/security/` |
| **Fuzzing** | Find edge cases | `tests/fuzz/` |

### 6.2 Scanner Test Cases

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
            name:  "open with surrounding text",
            input: "Please read this: <open main.go> and tell me about it",
            expected: []Command{{Type: "open", Argument: "main.go"}},
        },
        {
            name:  "write with content",
            input: "<write test.txt>Hello, World!</write>",
            expected: []Command{{Type: "write", Argument: "test.txt", Content: "Hello, World!"}},
        },
        {
            name:  "multiple commands",
            input: "<open a.go><open b.go>",
            expected: []Command{
                {Type: "open", Argument: "a.go"},
                {Type: "open", Argument: "b.go"},
            },
        },
        {
            name:  "command in comment should still parse",
            input: "// <open secret.key>",  // Scanner parses; validator rejects
            expected: []Command{{Type: "open", Argument: "secret.key"}},
        },
    }
    // ...
}
```

### 6.3 Security Test Cases

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

---

## 7. Performance Considerations

### 7.1 Bottlenecks

| Operation | Current | Target | Notes |
|-----------|---------|--------|-------|
| Command parsing | <1ms | <1ms | Already fast |
| Path validation | <1ms | <1ms | Already fast |
| Docker startup | 1-3s | 200ms | Use warm containers |
| Embedding generation | 2-5s | 100ms | Use Go-native or cache |

### 7.2 Optimization Opportunities

1. **Container Pool:** Pre-warm Docker containers to eliminate startup latency
2. **Embedding Cache:** Cache embeddings in SQLite (already implemented)
3. **Streaming Output:** For large command outputs, stream rather than buffer

---

## 8. Future Enhancements

### 8.1 Additional Commands

| Command | Description | Priority |
|---------|-------------|----------|
| `<git ...>` | Version control operations | Medium |
| `<diff file1 file2>` | File comparison | Low |
| `<grep pattern>` | Search without embeddings | Low |
| `<tree>` | Directory structure | Low |

### 8.2 Protocol Enhancements

| Enhancement | Description | Priority |
|-------------|-------------|----------|
| JSON mode | Structured input/output for programmatic use | Medium |
| WebSocket | Real-time streaming for long operations | Low |
| MCP | Model Context Protocol integration | High |

---

## 9. References

1. Abelson, H. & Sussman, G.J. (1996). *Structure and Interpretation of Computer Programs* (2nd ed.). MIT Press. ("The Wizard Book")

2. Aho, A.V., Sethi, R., & Ullman, J.D. (1986). *Compilers: Principles, Techniques, and Tools*. Addison-Wesley. ("The Dragon Book")

3. Kernighan, B.W. & Pike, R. (1984). *The UNIX Programming Environment*. Prentice Hall.

4. Pike, R. (2012). "Lexical Scanning in Go." https://go.dev/talks/2011/lex.slide

5. Docker Security Best Practices. https://docs.docker.com/engine/security/

---

## 10. Appendix: Quick Reference

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

*Document version: 1.0*  
*Last updated: December 5, 2025*  
*Author: JJ Salley*

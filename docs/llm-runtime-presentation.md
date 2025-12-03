class: center, middle

# LLM File Access Tool

**A Secure Tool for LLM-Powered Repository Exploration**

Developed by: JJ

*Team Workshop Presentation*

[https://github.com/computerscienceiscool/llm-runtime](https://github.com/computerscienceiscool/llm-runtime)

---

## What We'll Cover Today

1. What is llm-runtime?
2. Why does this tool exist?
3. The Four Core Commands
4. Live Walkthrough: From Simple to Advanced
5. Security & Best Practices
6. Q&A

By the end, you'll understand how to use this tool and why it matters.

---

## What is llm-runtime?

A **command-line tool** that lets Large Language Models (like Claude or ChatGPT) **explore and work with local code repositories**.

**The Problem it Solves:**
- LLMs have context limits (can't load entire codebases)
- Manual copy-paste is tedious and error-prone
- No way for LLMs to run tests or verify their suggestions

**The Solution:**
- LLMs embed special commands in their responses
- The tool parses and executes those commands safely
- Results are returned so the LLM can continue working

---

## How Does It Work?

**Flow:**
1. You ask an LLM to help with your code
2. LLM includes commands like `<open README.md>` in its response
3. **You copy** that response and pipe it to llm-runtime
4. Tool parses and executes commands safely
5. **You copy** results back to the LLM for further analysis

**Important:** This is currently a manual workflow—you're the bridge between the LLM and your codebase. The "autonomous" part is that the LLM decides what to explore, not that it runs automatically.

---

## The Four Core Commands

| Command | Purpose | Example |
|---------|---------|---------|
| `<open>` | Read files | `<open src/main.go>` |
| `<write>` | Create/modify files | `<write config.yaml>content</write>` |
| `<exec>` | Run commands safely | `<exec go test>` |
| `<search>` | Semantic code search | `<search authentication logic>` |

**Complexity Order:** open → write → exec → search

Let's walk through each one!

---

class: center, middle

# Part 1: Reading Files
## The `<open>` Command

*The foundation of everything*

---

## `<open>` - Reading Files

**Syntax:**
```
<open filepath>
```

**Examples:**
```
<open README.md>
<open src/main.go>
<open config/database.yaml>
```

**What happens:**
1. Tool validates the path is safe
2. Reads the file contents
3. Returns formatted output to the LLM

---

## `<open>` - Demo Walkthrough

**Step 1:** Start with a simple command
```bash
echo "<open README.md>" | ./llm-runtime
```

**Step 2:** Observe the output structure
```
=== LLM TOOL START ===
=== COMMAND: <open README.md> ===
=== FILE: README.md ===
[file contents here]
=== END FILE ===
=== END COMMAND ===
=== LLM TOOL COMPLETE ===
```

---

## `<open>` - Security Features

**Built-in Protections:**

| Protection | What it does |
|------------|--------------|
| Path Validation | Prevents `../../etc/passwd` attacks |
| Excluded Paths | Blocks `.git`, `.env`, `*.key` files |
| Size Limits | Default 1MB max file size |
| Symlink Resolution | Validates final destination |

**Demo:** Try reading outside the repo:
```bash
echo "<open ../../../etc/passwd>" | ./llm-runtime
# Returns: PATH_SECURITY error
```

---

class: center, middle

# Part 2: Writing Files
## The `<write>` Command

*Create and modify with confidence*

---

## `<write>` - Creating & Modifying Files

**Syntax:**
```
<write filepath>
content goes here
</write>
```

**Example:**
```
<write config.yaml>
name: my-project
version: 1.0.0
database:
  host: localhost
  port: 5432
</write>
```

---

## `<write>` - Key Features

**Automatic Protections:**

| Feature | Benefit |
|---------|---------|
| Atomic Writes | Writes to temp file first, then renames |
| Auto-Backup | Creates `.bak` file before overwriting |
| Extension Whitelist | Only allows safe file types |
| Auto-Formatting | Formats Go and JSON files automatically |

**Size limit:** 100KB by default

---

## `<write>` - Demo Walkthrough

**Create a new file:**
```bash
echo '<write test-file.txt>
Hello from llm-runtime!
</write>' | ./llm-runtime
```

**Verify and update:**
```bash
cat test-file.txt

echo '<write test-file.txt>
Updated content!
</write>' | ./llm-runtime

ls -la test-file.txt*  # See the .bak file
```

---

## `<write>` - Auto-Formatting Demo

**Go files get auto-formatted:**
```bash
echo '<write hello.go>
package main
import "fmt"
func main(){fmt.Println("Hello")}
</write>' | ./llm-runtime
```

**Check the result:**
```bash
cat hello.go
# Output is properly formatted!
```

---

class: center, middle

# Part 3: Running Commands
## The `<exec>` Command

*Execute safely with Docker isolation*

---

## `<exec>` - Command Execution

**Syntax:**
```
<exec command arguments>
```

**Examples:**
```
<exec go test>
<exec npm build>
<exec python -m pytest>
<exec make clean>
```

**Important:** Requires Docker and `--exec-enabled` flag!

---

## `<exec>` - Why Docker?

**Security through isolation:**

| Without Docker | With Docker |
|----------------|-------------|
| Commands run on your machine | Commands run in disposable container |
| Full network access | NO network access |
| Could delete files | Repository mounted read-only |
| No resource limits | Memory/CPU/time limits |

**Bottom line:** Even if an LLM tries to run something dangerous, your system is protected.

---

## `<exec>` - How It Works

```
Your Machine                    Docker Container
┌─────────────────┐            ┌─────────────────┐
│                 │            │ Ubuntu 22.04    │
│  llm-runtime    │───────────▶│                 │
│                 │            │ /workspace (RO) │
│  /your/repo     │◀───────────│ go test ./...   │
│                 │   results  │                 │
└─────────────────┘            └─────────────────┘
                                (destroyed after)
```

---

## `<exec>` - Command Whitelist

**Only pre-approved commands allowed:**

| Category | Commands |
|----------|----------|
| Go | `go test`, `go build`, `go run` |
| Node.js | `npm test`, `npm build`, `node` |
| Python | `python`, `python3`, `pytest` |
| Build | `make`, `make test`, `make build` |
| System | `ls`, `cat`, `grep`, `find`, `wc` |

**Not whitelisted = blocked.**

---

## `<exec>` - Demo Walkthrough

**Run a simple command:**
```bash
echo "<exec echo 'Hello from Docker'>" | \
  ./llm-runtime --exec-enabled
```

**Run actual tests:**
```bash
echo "<exec go test ./...>" | \
  ./llm-runtime --exec-enabled
```

**Try a blocked command:**
```bash
echo "<exec rm -rf />" | ./llm-runtime --exec-enabled
# Blocked: command not in whitelist
```

---

## `<exec>` - Combine Commands

**Real workflow example:**
```bash
echo "Let me check and test this project:
<open go.mod>
<exec go build ./...>
<exec go test ./...>" | ./llm-runtime --exec-enabled
```

**The LLM can:**
1. Read the module file
2. Verify the project builds
3. Run the tests

All in one response!

---

class: center, middle

# Putting It All Together

*A complete workflow example*

---

## Complete Workflow Example

**Scenario:** Review a project and run tests

```bash
echo "I'll analyze this project:

<open README.md>
<open go.mod>
<exec go test ./...>
<exec go test -cover ./...>" | ./llm-runtime --exec-enabled
```

---

## Interactive Mode

**For ongoing exploration:**
```bash
./llm-runtime --interactive --exec-enabled
```

**Now you can type commands continuously:**
```
<open README.md>
<exec go test ./...>
```

**Exit with Ctrl+D**

---

class: center, middle

# Part 4: Semantic Search
## The `<search>` Command

*Find code by meaning, not just keywords*

---

## `<search>` - Semantic Code Search

**Syntax:**
```
<search query terms>
```

**Examples:**
```
<search authentication middleware>
<search database connection handling>
<search error handling patterns>
```

**Different from grep:** Understands *meaning*, not just text.

---

## `<search>` - How It's Different

| Traditional (grep) | Semantic (search) |
|-------------------|-------------------|
| `grep "auth"` | `<search user authentication>` |
| Exact match only | Finds "login", "signin", "authenticate" |
| No context | Understands relationships |
| Fast but limited | AI-powered understanding |

**Use case:** Finding related code in unfamiliar codebases.

---

## `<search>` - Demo

**Setup (one-time):**
```bash
pip install sentence-transformers
./llm-runtime --reindex
```

**Search for code:**
```bash
echo "<search error handling>" | ./llm-runtime
```

**Output:**
```
=== SEARCH RESULTS ===
1. internal/executor/exec.go (score: 78.50)
   Preview: "// ExecuteExec handles the exec command..."
2. internal/security/path.go (score: 72.30)
   Preview: "// ValidatePath ensures the path is safe..."
```

---

class: center, middle

# Current Limitations

*Being honest about what this tool doesn't do*

---

## What This Tool Doesn't Do (Yet)

| Limitation | What it means |
|------------|---------------|
| **Manual workflow** | You copy/paste between AI and tool |
| **No IDE integration** | Command-line only, no VS Code plugin |
| **AI still makes mistakes** | More capability, same intelligence |
| **Requires setup** | Docker for exec, Python for search |

**The value:** Gives AI actual access to your code in a controlled way—it can explore, verify, iterate instead of guessing from snippets.

---

class: center, middle

# Security Summary

*Why you can trust this tool*

---

## Security Layers

| Layer | Protection |
|-------|------------|
| Path Validation | No directory traversal |
| Excluded Paths | Sensitive files blocked |
| Extension Whitelist | Only safe file types |
| Command Whitelist | Only approved commands |
| Docker Isolation | No host system access |
| No Network | Containers are offline |
| Resource Limits | Memory, CPU, time caps |
| Audit Logging | Full operation history |

---

## Quick Reference

| Task | Command |
|------|---------|
| Read file | `<open filepath>` |
| Write file | `<write path>content</write>` |
| Run command | `<exec command>` (needs `--exec-enabled`) |
| Search code | `<search query>` (needs index) |
| Interactive | `./llm-runtime --interactive` |
| Build index | `./llm-runtime --reindex` |
| View help | `./llm-runtime --help` |

---

## Common Use Cases

| Use Case | Commands Needed |
|----------|-----------------|
| Code review | `<open>` + `<exec go test>` |
| Bug investigation | `<search>` + `<open>` |
| Documentation | `<open>` + `<write>` |
| Refactoring | All four commands |
| Project onboarding | `<open>` + `<search>` |

---

## Getting Started

**1. Clone and build:**
```bash
git clone https://github.com/computerscienceiscool/llm-runtime
cd llm-runtime
make build
```

**2. Quick test:**
```bash
echo "<open README.md>" | ./llm-runtime
```

**3. Enable exec (optional):**
```bash
./llm-runtime --exec-enabled --interactive
```

---

## Resources & Feedback

- **Repository:** [github.com/computerscienceiscool/llm-runtime](https://github.com/computerscienceiscool/llm-runtime)
- **Documentation:** `docs/` folder in the repo
- **System Prompt:** `docs/SYSTEM_PROMPT.md` (for LLM integration)

**Please provide feedback!**
- Open GitHub issues for bugs or ideas
- Or just grab JJ directly

---

class: center, middle

# Questions?

*Let's discuss!*

---

## Appendix: Troubleshooting

| Issue | Solution |
|-------|----------|
| FILE_NOT_FOUND | Check path is relative to repo root |
| PATH_SECURITY | File is protected or outside repo |
| DOCKER_UNAVAILABLE | Install/start Docker |
| EXEC_VALIDATION | Command not in whitelist |
| SEARCH_DISABLED | Enable search in config |

**Full troubleshooting:** `docs/troubleshooting.md`

---

## Appendix: All CLI Flags

```bash
--root PATH           # Repository root
--max-size BYTES      # Max file read size
--max-write-size      # Max file write size
--exec-enabled        # Enable exec command
--exec-timeout        # Command timeout
--exec-whitelist      # Allowed commands
--interactive         # Interactive mode
--reindex             # Rebuild search index
--verbose             # Detailed output
--help                # Show all options
```

# Frequently Asked Questions (FAQ)

## General

### What does this tool do?
It allows LLMs to explore and work with local code repositories by parsing special commands (`<open>`, `<write>`, `<exec>`, `<search>`) from LLM responses and executing them safely. Instead of uploading entire codebases to an LLM, the LLM can dynamically request files or run commands as needed.

### How is this different from uploading files to ChatGPT/Claude?
- **Scale**: Works with repositories too large for LLM context limits
- **Dynamic**: LLMs explore progressively, not everything upfront
- **Security**: Files stay local with audit logging and access controls
- **Interactive**: Can run tests, build projects, validate changes
- **Efficient**: Only loads relevant files

### Do I need all four features?
No. Enable what you need:
- **Minimum**: Just `<open>` for file reading
- **Documentation**: Add `<write>` for creating/updating files
- **Development**: Add `<exec>` for running tests and builds (controlled by whitelist)
- **Large codebases**: Add `<search>` for semantic discovery

### Is this secure?
Yes, with proper configuration:
- All operations happen locally (no data leaves your machine)
- Path validation prevents directory traversal
- Command whitelisting restricts what can execute
- Docker isolation for exec commands
- Comprehensive audit logging
- Configurable excluded paths

---

## Docker and Exec

### Why do I need Docker?
Security and isolation. Docker ensures:
- Commands can't access your host system
- No network access (completely offline)
- Resource limits prevent runaway processes
- Consistent environment regardless of host OS
- Easy cleanup (containers auto-remove)

### What if I don't have Docker?
The tool works fine without it:
- File reading (`<open>`) works
- File writing (`<write>`) works
- Search (`<search>`) works
- Only `<exec>` requires Docker

### Can dangerous commands run?
No. Multiple safety layers:
- Commands must be whitelisted
- Docker containers are isolated
- Containers run with minimal privileges
- Repository mounted read-only
- No network access

### Why are commands timing out?
Default timeout is 30 seconds. Increase it:
```bash
./llm-runtime --exec-timeout 60s
```
Or in config:
```yaml
commands:
  exec:
    timeout_seconds: 60
```

---

## Search

### What is semantic search?
Search that understands meaning, not just keywords:
- **Traditional**: `grep "login"` finds exact word "login"
- **Semantic**: `<search user authentication>` finds login, auth, signin, etc.

### Why do I need Ollama?
Ollama runs the AI model that generates embeddings locally. Benefits:
- Privacy: Code never leaves your machine
- Speed: No network latency
- Cost: No API fees
- Offline: Works without internet

### How do I set up search?
```bash
# Install Ollama
curl -fsSL https://ollama.com/install.sh | sh

# Pull embedding model
ollama pull nomic-embed-text

# Build search index
./llm-runtime --reindex
```

### How much disk space does the index use?
Roughly 1-2KB per file:
- 1,000 files ≈ 1-2MB
- 10,000 files ≈ 10-20MB

### When should I rebuild the index?
```bash
# After major changes
./llm-runtime --reindex
```

---

## File Operations

### Why can't I write certain files?
Security feature. Check:
- Extension whitelist in config
- Excluded paths
- File size limits

### Where do backups go?
Same directory with timestamp:
```
file.go → file.go.bak.1640995200
```

### Can I modify files outside my project?
No. All operations restricted to repository root.

---

## Configuration

### Where does the config file go?
Tool looks in order:
1. `./llm-runtime.config.yaml` (current directory)
2. `~/.llm-runtime.config.yaml` (home directory)
3. Built-in defaults

### Can I use different configs for different projects?
Yes:
```bash
./llm-runtime --config project-config.yaml
```

### How do I see what commands are whitelisted?
```bash
grep -A 20 "whitelist:" llm-runtime.config.yaml
```

---

## Performance

### What slows things down?
Main bottlenecks:
- **Docker startup**: 1-3s per exec command
- **Large files**: Reading multi-MB files
- **First search**: Ollama model loading (~500ms)

### How can I make it faster?
- Pre-pull Docker images: `docker pull ubuntu:22.04`
- Combine exec commands: `<exec cmd1 && cmd2>`
- Limit file sizes in config
- Keep Ollama running in background

---

## Troubleshooting

### PATH_SECURITY errors?
You're accessing restricted paths:
- Outside repository (`../../etc/passwd`)
- Excluded files (`.env`, `*.key`)
- System directories

Use relative paths within your repo.

### Commands not working?
Check:
1. Command in whitelist?
2. Docker running?
3. Correct syntax? (`<exec go test>` not `<exec> go test</exec>`)

### Search returns nothing?
1. Ollama running? `ollama list`
2. Index built? `./llm-runtime --reindex`
3. File types indexed? Check `index_extensions` in config

---

## Best Practices

### How should I integrate with LLMs?
Use the system prompt in `docs/SYSTEM_PROMPT.md`:
- Start with `<open README.md>`
- Use `<search>` for discovery in large codebases
- Always `<exec>` test after changes
- Be systematic in exploration

### What's a good code review workflow?
```
1. <open README.md>           # Understand project
2. <search main entry>        # Find starting point
3. <open src/main.go>         # Read code
4. <exec go test>             # Run tests
5. [Review specific areas]
6. <exec go build>            # Verify builds
```

### When should I use each command?
- **`<open>`**: Reading code, config, docs
- **`<write>`**: Creating docs, fixing bugs, adding features
- **`<exec>`**: Running tests, building, validation
- **`<search>`**: Finding code in large/unfamiliar codebases

---

## Comparison

### vs GitHub Copilot?
Different use cases:
- **Copilot**: Code completion and generation
- **llm-runtime**: Repository exploration and analysis
- **Together**: Comprehensive AI assistance

### vs just using grep?
| Aspect | grep | `<search>` |
|--------|------|------------|
| Query | Exact pattern | Natural language |
| Finds | Exact matches | Related concepts |
| Setup | Built-in | Requires Ollama |
| Best for | Known patterns | Discovery |

---

## Getting Help

### Where to report issues?
GitHub: https://github.com/computerscienceiscool/llm-runtime/issues

### What info to include?
- Command that failed
- Complete error message
- OS, Go version, Docker version
- Relevant config sections

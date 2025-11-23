# Frequently Asked Questions (FAQ)

Common questions and answers about the LLM File Access Tool.

## General Usage

### **Q: What exactly does this tool do?**
**A:** It allows LLMs to autonomously explore and work with local code repositories by parsing special commands (`<open>`, `<write>`, `<exec>`, `<search>`) from LLM responses and executing them safely. Instead of uploading entire codebases to an LLM, the LLM can dynamically request specific files or run commands as needed.

### **Q: How is this different from just uploading files to ChatGPT/Claude?**
**A:** 
- **Scale**: Works with large repositories that exceed LLM context limits
- **Dynamic**: LLMs can explore progressively rather than processing everything upfront
- **Security**: Files stay local, with audit logging and access controls
- **Interactive**: Can run tests, build projects, and validate changes in real-time
- **Efficient**: Only loads relevant files instead of entire codebases

### **Q: Do I need all four features (open, write, exec, search)?**
**A:** No! You can enable features as needed:
- **Minimum**: Just `<open>` for file reading
- **Documentation work**: Add `<write>` for creating/updating files
- **Development work**: Add `<exec>` for running tests and builds
- **Large codebases**: Add `<search>` for semantic discovery

### **Q: Is this secure to use with sensitive code?**
**A:** Yes, with proper configuration:
- All operations happen locally (no data leaves your machine)
- Path validation prevents directory traversal attacks
- Command whitelisting restricts what can be executed
- Docker isolation for exec commands
- Comprehensive audit logging
- Configurable excluded paths for sensitive files

## Docker and Exec Commands

### **Q: Why do I need Docker for exec commands?**
**A:** Security and isolation. Docker ensures that:
- Commands can't access your host system
- No network access (completely offline)
- Resource limits prevent runaway processes
- Consistent environment regardless of host OS
- Easy cleanup (containers auto-remove)

### **Q: What if I don't have Docker?**
**A:** The tool works fine without Docker:
- File reading (`<open>`) works normally
- File writing (`<write>`) works normally  
- Search (`<search>`) works normally
- Only `<exec>` commands will be disabled

### **Q: Can I run dangerous commands like `rm -rf /`?**
**A:** No. Multiple safety layers prevent this:
- Commands must be whitelisted (default blocks dangerous commands)
- Docker containers are isolated from your host system
- Containers run with minimal privileges
- Repository mounted read-only
- No network access

### **Q: Why are my commands timing out?**
**A:** Default timeout is 30 seconds. For longer operations:
```bash
./llm-tool --exec-timeout 120s
```
Or in config:
```yaml
commands:
  exec:
    timeout_seconds: 120
```

## Search Functionality

### **Q: What is semantic search and why do I need it?**
**A:** Semantic search understands meaning, not just keywords:
- **Traditional**: `grep "login"` finds exact word "login"
- **Semantic**: `<search user authentication>` finds login, auth, signin, etc.
- **Useful for**: Large codebases, unfamiliar code, finding related functionality

### **Q: Why do I need Python for search?**
**A:** The semantic search uses AI models (sentence-transformers) that require Python. The Go tool calls Python scripts to generate embeddings.

### **Q: How much disk space does search indexing use?**
**A:** Roughly 1.5KB per file:
- 1000 files = 1.5MB index
- 10,000 files = 15MB index
- Large codebases (100k files) = 150MB index

### **Q: How often should I rebuild the search index?**
**A:** 
- **Daily development**: Use `--search-update` (incremental)
- **Major changes**: Use `--reindex` (full rebuild)
- **Automatic**: Tool auto-updates index when files change

## File Operations

### **Q: Why can't I write to certain file types?**
**A:** Security feature. By default, only safe extensions are allowed:
```yaml
commands:
  write:
    allowed_extensions: [".go", ".py", ".js", ".md", ".txt"]
```
Add more as needed or remove restriction entirely.

### **Q: Where do backup files go?**
**A:** Same directory as original with timestamp:
```
original.go → original.go.bak.1640995200
```
Disable with `backup_before_write: false` in config.

### **Q: Can I modify files outside my project?**
**A:** No. All file operations are restricted to the repository root for security. This prevents accidental modification of system files.

### **Q: What happens if I write a file that already exists?**
**A:** 
1. Tool creates backup (if enabled)
2. Writes new content atomically (via temp file)
3. Reports "UPDATED" status
4. Original backed up as `.bak.timestamp`

## Configuration

### **Q: Where should I put my config file?**
**A:** Tool looks in this order:
1. `./llm-tool.config.yaml` (current directory) - **Recommended**
2. `~/.llm-tool.config.yaml` (home directory)
3. Built-in defaults

### **Q: Can I use different configs for different projects?**
**A:** Yes! Use project-specific configs:
```bash
# In each project directory
./llm-tool --config project-config.yaml

# Or specify path
./llm-tool --config /path/to/specific-config.yaml
```

### **Q: How do I know what commands are whitelisted?**
**A:** Check your config:
```bash
grep -A 20 "whitelist:" llm-tool.config.yaml
```
Or see defaults in documentation.

## Performance

### **Q: How fast is file access?**
**A:** Very fast for typical usage:
- File reading: <10ms for 1MB files on SSD
- Path validation: <1ms per path
- Command parsing: <1ms for typical input
- Docker startup: 1-3s (main bottleneck)

### **Q: What slows down the tool?**
**A:** Main bottlenecks:
- **Docker startup**: 1-3s per exec command
- **Large files**: Reading multi-MB files
- **Many excluded paths**: Complex pattern matching
- **Search indexing**: First-time index building

### **Q: How can I make it faster?**
**A:** 
- **Pre-pull Docker images**: `docker pull ubuntu:22.04`
- **Optimize exclusions**: More specific patterns
- **Combine exec commands**: `<exec cmd1 && cmd2 && cmd3>`
- **Use smaller Docker images**: Alpine variants
- **Limit file sizes**: Appropriate size limits

## Security

### **Q: Can this tool access my private keys/secrets?**
**A:** Not by default. These are excluded:
- `.env` files
- `*.key` files
- `*.pem` files
- `.git` directory
- `secrets/` directories
- `credentials/` directories

### **Q: What gets logged in audit.log?**
**A:** Every operation:
- Timestamp and session ID
- Command type and arguments
- Success/failure status
- File paths accessed
- Command execution details
- Error messages

### **Q: Is it safe to run this on production servers?**
**A:** Generally no. This tool is designed for development environments. For production:
- Disable write commands
- Strict command whitelisting  
- Read-only repository access
- Comprehensive monitoring

### **Q: What if someone malicious gets access to my llm-tool?**
**A:** Limited damage due to built-in protections:
- Can only access files within repository
- Cannot execute non-whitelisted commands
- Docker isolation prevents host system access
- All operations are logged
- No network access from containers

## Troubleshooting

### **Q: Why am I getting PATH_SECURITY errors?**
**A:** You're trying to access restricted paths:
- Outside repository boundaries (`../../etc/passwd`)
- Excluded files (`.env`, `*.key`)
- System directories (`/etc`, `/usr`)

Solution: Use paths relative to repository root only.

### **Q: Why aren't my commands working?**
**A:** Common issues:
1. **Command not whitelisted**: Check `commands.exec.whitelist`
2. **Docker not running**: `docker info`
3. **Exec not enabled**: `commands.exec.enabled: true`
4. **Wrong syntax**: `<exec go test>` not `<exec> go test</exec>`

### **Q: Search isn't finding anything. Why?**
**A:** Possible causes:
1. **Search not enabled**: `commands.search.enabled: true`
2. **Index not built**: `./llm-tool --reindex`
3. **Python deps missing**: `pip install sentence-transformers`
4. **Wrong file types**: Check `index_extensions` in config
5. **High similarity threshold**: Lower `min_similarity_score`

## Best Practices

### **Q: How should I integrate this with LLMs?**
**A:** 
- Use the provided `SYSTEM_PROMPT.md` as your system prompt
- Start with `<open README.md>` to understand projects
- Use `<search>` for discovery in large codebases
- Always `<exec>` test after making changes
- Be systematic in exploration (don't random walk)

### **Q: What's the best workflow for code review?**
**A:**
```
1. <open README.md> - Understand project
2. <search main entry point> - Find start
3. <open src/main.go> - Read entry point
4. <exec go test> - Run existing tests
5. [Review specific areas]
6. <exec go build> - Verify builds
```

### **Q: How should I organize my excluded paths?**
**A:** By sensitivity level:
```yaml
repository:
  excluded_paths:
    # Security critical
    - ".env*"
    - "*.key"
    - "*.pem" 
    - "secrets/"
    
    # System files
    - ".git"
    - "node_modules"
    - "__pycache__"
    
    # Build artifacts
    - "dist/"
    - "build/"
    - "*.log"
```

### **Q: When should I use each command type?**
**A:**
- **`<open>`**: Understanding code, reading config, examining files
- **`<write>`**: Creating docs, fixing bugs, adding features
- **`<exec>`**: Running tests, building, validation, exploration
- **`<search>`**: Finding relevant code in large/unfamiliar codebases

## Advanced Usage

### **Q: Can I extend this tool with custom commands?**
**A:** Not currently, but you can:
- Add custom scripts to exec whitelist
- Create wrapper scripts that call multiple commands
- Use the tool programmatically in larger workflows

### **Q: Can I run this in CI/CD?**
**A:** Yes, with careful configuration:
- Disable interactive features
- Read-only mode for safety
- Specific command whitelists
- Proper resource limits
- Comprehensive logging

### **Q: How does this compare to GitHub Copilot?**
**A:** Different use cases:
- **Copilot**: Code completion and generation
- **LLM Tool**: Repository exploration and analysis
- **Complementary**: Use both together for comprehensive AI assistance

This tool excels at understanding existing codebases, while Copilot excels at generating new code.

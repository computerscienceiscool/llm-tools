# Frequently Asked Questions (FAQ)

## General Questions

### What is llm-runtime?

llm-runtime is a secure runtime environment that enables Large Language Models (LLMs) to interact with code repositories through containerized commands. It provides semantic search, file operations, and command execution with strong security isolation.

### What makes llm-runtime secure?

Security is built on three pillars:
1. **Container Isolation**: All operations (read, write, exec) run in isolated Docker containers
2. **Command Whitelisting**: Only pre-approved commands can execute
3. **Path Validation**: File operations restricted to repository boundaries

Every operation is logged for audit purposes.

### Where do file operations happen by default?

By default, llm-runtime creates a temporary git repository in `/tmp/dynamic-repo/repo-XXXXXXX/` for all operations. This prevents accidental modification of your working directories.

**To use a specific directory:**
```bash
./llm-runtime --root /path/to/your/project
```

**To preserve temporary repos for debugging:**
```bash
KEEP_TEST_REPOS=true ./llm-runtime
```

This design ensures llm-runtime never pollutes your source code directories with test files or backups.

### Do I need Docker for everything?

Yes. Docker is required for:
- File reading (containerized I/O)
- File writing (containerized I/O)  
- Command execution
- Security isolation

The tool cannot function without Docker.

### What languages/frameworks are supported?

llm-runtime is language-agnostic. It works with any codebase because it operates at the file and command level. Common workflows include:
- Go projects (`go test`, `go build`)
- Node.js projects (`npm test`, `npm build`)
- Python projects (`pytest`, `pip install`)
- Any project with Make, CMake, Cargo, etc.

### Is this production-ready?

llm-runtime is designed for development environments where LLMs assist with coding. It's production-ready for:
- Local development
- CI/CD pipelines (with proper security controls)
- Code review automation
- Testing automation

**Not recommended for:** Exposing directly to untrusted LLMs or running user-provided code without review.

## Installation & Setup

### What are the minimum requirements?

**Required:**
- Go 1.21 or later
- Docker (daemon must be running)
- Git

**Optional:**
- Ollama (for semantic search)

### Why does installation require Docker?

Docker provides the security isolation layer. All operations execute in lightweight containers to prevent:
- Host system modification
- Network access from commands
- Resource exhaustion
- Privilege escalation

### Can I run this on Windows?

Yes, with Docker Desktop. WSL2 is recommended for best performance.

### Can I use Podman instead of Docker?

The tool currently requires Docker. Podman support could be added but requires compatibility testing with the containerization code.

### How much disk space do I need?

**Minimum:**
- ~500MB for Docker images
- ~50MB for tool binaries
- Variable for search index (depends on repository size)

**Typical:**
- 1-2GB for Docker images (base images + custom images)
- Search index: ~1-5MB per 1000 files

### How do I update llm-runtime?

```bash
cd llm-runtime
git pull origin main
make clean
make build
./llm-runtime --reindex  # Update search index
```

## Docker Questions

### Why do I get "Docker daemon not running"?

Docker must be running before using llm-runtime.

**Linux:**
```bash
sudo systemctl start docker
```

**macOS/Windows:**
Start Docker Desktop application

### Why do I get "permission denied" with Docker?

On Linux, add your user to the docker group:
```bash
sudo usermod -aG docker $USER
newgrp docker
```

Verify: `docker run hello-world`

### What Docker images does the tool use?

**Default images:**
- `llm-runtime-io:latest` - Minimal Alpine for file I/O (build with `make build-io-image`)
- `python-go` - Combined Python + Go environment for exec commands
- `alpine:latest` - Fallback for I/O operations

You can configure custom images in `llm-runtime.config.yaml`.

### Can I use my own Docker images?

Yes! Configure in `llm-runtime.config.yaml`:

```yaml
commands:
  exec:
    container_image: "node:18-alpine"  # For Node projects
  io:
    container_image: "alpine:latest"   # For I/O operations
```

### Why are Docker containers slow to start?

**First run:** Images need to be downloaded/built
**Subsequent runs:** Should be fast (~100-500ms)

**Optimization:**
```bash
# Pre-pull images
docker pull alpine:latest
docker pull python-go:latest

# Build I/O image
make build-io-image

# Use faster storage (SSD over HDD)
```

### Do containers have network access?

No. All containers run with `--network none` for security. They cannot:
- Download packages from the internet
- Exfiltrate data
- Make API calls

This is by design.

### How do I clean up Docker resources?

```bash
# Remove old containers
docker container prune

# Remove unused images
docker image prune -a

# Full cleanup (careful!)
docker system prune -a
```

## Search Questions

### How does semantic search work?

llm-runtime uses Ollama with the `nomic-embed-text` model to:
1. Generate embeddings for all code files
2. Store embeddings in SQLite database
3. Search by semantic similarity (not just keyword matching)

### Do I need Python for search?

No. Search uses Ollama (a Go-based tool), not Python. Previous documentation incorrectly mentioned Python sentence-transformers.

### Why is search not finding my files?

**Common causes:**
1. Search index not built: `./llm-runtime --reindex`
2. Ollama not running: `ollama serve`
3. Embedding model not downloaded: `ollama pull nomic-embed-text`
4. Index out of date: Rebuild with `--reindex`

### How often should I rebuild the search index?

Rebuild when:
- Adding new files
- Significantly changing code structure
- Search results seem outdated

**Command:** `./llm-runtime --reindex`

### Can I search binary files?

No. Search only indexes text files (code, docs, configs). Binary files are skipped.

### How big can the search index get?

Typical sizes:
- Small project (100 files): ~500KB
- Medium project (1000 files): ~5MB
- Large project (10,000 files): ~50MB

The index is stored in `embeddings.db`.

### Why is search slow?

**Possible causes:**
1. Large repository (many files)
2. Ollama performance (check `ollama list`)
3. Disk I/O (use SSD)
4. Large index file

**Solutions:**
- Exclude unnecessary files (.gitignore)
- Rebuild index to optimize
- Check Ollama resource usage

### Can I disable search?

Yes. Just don't use `<search>` commands. The search system only activates when needed.

### What's the difference between search and grep?

- **Search (`<search>`)**: Semantic/conceptual similarity (understands meaning)
- **Grep (`<exec grep>`)**: Exact text matching

Example:
- `<search user authentication>` finds auth code even without those exact words
- `<exec grep "login">` finds only lines containing "login"

## File Operations

### Why are file operations containerized?

Defense in depth. Even "safe" operations like reading files are containerized to:
- Prevent exploits in file parsing libraries
- Isolate resource usage
- Maintain consistent security model
- Enable complete audit trail

### Can I read files outside the repository?

No. File paths are validated to prevent access outside the repository boundaries. 

**By default**, operations occur in a temporary repository at `/tmp/dynamic-repo/repo-XXXXXXX/`. To work with your actual project:
```bash
./llm-runtime --root /path/to/your/project
```

Path validation protects:
- System files (`/etc/passwd`)
- User files (`~/.ssh/`)
- Other repositories (`../../other-repo`)
- Paths outside the specified repository root

### What's the maximum file size?

**Practical limits:**
- Read operations: ~10MB (beyond this, LLM context limits apply)
- Write operations: Limited by timeout (default: 10s)

**For large files:**
```bash
# Use exec to filter/extract
<exec head -n 100 large.log>
<exec grep "ERROR" application.log>
<exec tail -n 50 debug.log>
```

### How are writes made atomic?

Writes use a two-step process:
1. Write to temporary file (`file.txt.tmp`)
2. Atomic rename to final name (`file.txt`)

This ensures:
- No partial writes
- No corruption on failure
- Safe concurrent access

### Can I modify multiple files at once?

Yes, but each `<write>` command modifies one file. For multiple files:

```
<write file1.go>
...content...
</write>

<write file2.go>
...content...
</write>
```

## Command Execution

### Is command execution always enabled?

Yes. Execution is always available via the container-based security model. Access is controlled solely by the whitelist.

Note: Older documentation mentioned an `--exec-enabled` flag, but exec is now always enabled.

### How do I allow new commands?

Add to whitelist in `llm-runtime.config.yaml`:

```yaml
commands:
  exec:
    whitelist:
      - "go test"
      - "npm build"
      - "your-new-command"  # Add here
```

Or use CLI flag:
```bash
./llm-runtime --exec-whitelist "go test,npm build,your-command"
```

### Why is my command not allowed?

Commands must be explicitly whitelisted for security. Even common commands like `rm`, `sudo`, `curl` are blocked by default.

**Check whitelist:**
```bash
cat llm-runtime.config.yaml | grep -A 20 "whitelist:"
```

### Can commands access the internet?

No. All containers run with `--network none`. Commands cannot:
- Download packages
- Make API calls
- Access databases
- SSH to other systems

### How do I install dependencies?

Dependencies must be pre-installed in the container image. Modify the Dockerfile for your exec container to include needed tools.

Example:
```dockerfile
FROM python:3.11
RUN pip install pytest requests flask
```

Then configure the tool to use this image.

### Why do commands timeout?

Default timeout is 30 seconds. Increase if needed:

```bash
./llm-runtime --exec-timeout 60s
```

Or in config:
```yaml
commands:
  exec:
    timeout_seconds: 60
```

### Can I run interactive commands?

No. Commands must complete and exit. Interactive commands (prompts, REPLs) will hang and timeout.

### What's the command exit code?

All command outputs include exit code:
```
=== EXEC SUCCESSFUL: go test ===
Exit code: 0  # Success
...
```

Exit code 0 = success, non-zero = failure (but command still ran successfully).

## Configuration

### Where is the config file?

Default location: `./llm-runtime.config.yaml`

Specify custom location:
```bash
./llm-runtime --config /path/to/config.yaml
```

### Do I need a config file?

No. The tool works with defaults. Config is optional for customization.

### What can I configure?

Main configuration areas:
- Command whitelists
- Container images
- Timeouts
- Resource limits (memory, CPU)
- Audit logging
- Search settings

See [configuration.md](configuration.md) for full details.

### CLI flags vs config file?

CLI flags override config file settings. Precedence:
1. CLI flags (highest priority)
2. Config file
3. Built-in defaults (lowest priority)

### How do I validate my config?

```bash
# Test YAML syntax
python3 -c "import yaml; yaml.safe_load(open('llm-runtime.config.yaml'))"

# Or run with verbose output
./llm-runtime --verbose
```

## Security Questions

### Is this safe to use with untrusted LLMs?

llm-runtime provides strong isolation, but:

**Safe:** Local development with known/trusted LLMs
**Caution:** Automated systems where LLM behavior is unpredictable
**Unsafe:** Exposing to arbitrary internet users or malicious actors

**Best practice:** Review LLM-generated commands before execution.

### What prevents malicious commands?

Multiple layers:
1. **Whitelist**: Only approved commands run
2. **Container isolation**: Cannot escape to host
3. **No network**: Cannot download malware
4. **No privileges**: Non-root user, dropped capabilities
5. **Resource limits**: Cannot consume unlimited resources

### Can an LLM delete my files?

No, unless `rm` is in your whitelist (not recommended). The default whitelist excludes destructive commands.

### Can an LLM access my SSH keys?

No. File operations are restricted to the repository directory. Cannot access:
- `~/.ssh/`
- `/etc/`
- Other user files

### What's logged for security?

All operations are logged to audit.log:
- Commands executed
- Files read/written  
- Search queries
- Success/failure status
- Timestamps
- Session IDs

### Should I run as root?

**No.** Never run llm-runtime as root. The tool is designed for unprivileged users.

### Can I use this in CI/CD?

Yes, but ensure:
- Docker available in CI environment
- Proper secret management
- Audit logs captured
- Resource limits enforced
- Whitelist properly configured

## Performance Questions

### Why is the first command slow?

Docker images need to be pulled/built on first use. Subsequent commands use cached images.

**Speed up:**
```bash
# Pre-pull images
make build-io-image
docker pull python-go:latest
docker pull alpine:latest
```

### How can I speed up file operations?

1. Use SSD for Docker storage
2. Pre-build I/O container image
3. Increase resource limits if needed
4. Use smaller container images

### Why is search slow?

- First search: Loads index (~100-500ms)
- Large index: More files = slower
- Ollama performance: Check resource usage

**Optimize:**
```bash
# Rebuild index
./llm-runtime --reindex

# Check index size
ls -lh embeddings.db

# Exclude large/unnecessary files
```

### How much memory does the tool use?

- Tool itself: ~50-100MB
- Docker containers: Configurable (default: 128MB for I/O, 512MB for exec)
- Search index: Loaded on demand

Total: Typically ~200-500MB for normal operation.

### Can I run multiple instances?

Yes, but be aware:
- Each instance uses its own containers
- Search index can be shared (read-only)
- Resource limits apply per container
- Docker overhead multiplies

## Troubleshooting

### Common error: "Docker daemon not running"

**Solution:**
```bash
# Start Docker
sudo systemctl start docker  # Linux
# Or start Docker Desktop  # macOS/Windows
```

### Common error: "command not in whitelist"

**Solution:** Add command to whitelist in config file

### Common error: "Ollama connection failed"

**Solution:**
```bash
# Start Ollama
ollama serve

# Pull embedding model
ollama pull nomic-embed-text
```

### Common error: "search index not found"

**Solution:**
```bash
./llm-runtime --reindex
```

### Where can I find more help?

1. **Documentation:** Check `docs/` directory
2. **Troubleshooting guide:** [troubleshooting.md](troubleshooting.md)
3. **Configuration guide:** [configuration.md](configuration.md)
4. **Command guides:** Individual guides for each command type
5. **Issue tracker:** GitHub issues for bug reports

## Advanced Topics

### Can I extend llm-runtime with plugins?

Not currently. The tool has a fixed set of commands. Extensions would require code changes.

### Can I integrate with my IDE?

Yes, through stdin/stdout interface:
```bash
echo "<search authentication>" | ./llm-runtime
```

Some IDEs might support this via custom commands or extensions.

### Does it work with multi-repo setups?

Each llm-runtime instance operates on one repository. For multiple repositories:
- Run separate instances
- Or build wrapper scripts

### Can I customize the prompt format?

The prompt format (`<command>...</command>`) is fixed. This ensures reliable parsing.

### Does it support streaming responses?

No. Commands complete fully before returning results.

### Can I use this programmatically?

Yes, via stdin/stdout:
```go
cmd := exec.Command("./llm-runtime")
cmd.Stdin = strings.NewReader("<search auth>")
output, err := cmd.Output()
```

### How does it compare to GitHub Copilot?

Different tools:
- **Copilot:** Code completion, inline suggestions
- **llm-runtime:** Repository operations, testing, building

They're complementary - use both together.

## Getting Started

### What should I try first?

```bash
# 1. Build the tool
make build

# 2. Test Docker
echo "<exec echo 'hello'>" | ./llm-runtime

# 3. Try file reading
echo "<open README.md>" | ./llm-runtime

# 4. Setup search (optional)
ollama pull nomic-embed-text
./llm-runtime --reindex
echo "<search main function>" | ./llm-runtime
```

### Where should I go next?

1. Read [llm-runtime-overview.md](llm-runtime-overview.md)
2. Review [configuration.md](configuration.md)
3. Check command-specific guides:
   - [command-execution-guide.md](command-execution-guide.md)
   - [file-reading-guide.md](file-reading-guide.md)
   - [file-writing-guide.md](file-writing-guide.md)
   - [semantic-search-guide.md](semantic-search-guide.md)

### What's the learning curve?

- **Basic usage:** 5-10 minutes
- **Full configuration:** 30-60 minutes
- **Advanced workflows:** Practice with real projects

The tool is designed to be intuitive for both LLMs and humans.

---

**Still have questions?** Check the [troubleshooting guide](troubleshooting.md) or open an issue in the repository.

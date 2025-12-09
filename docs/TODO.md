# llm-runtime Todo List & Ideas

## Completed ✅

### Code Cleanup (December 2024)
- [x] Remove duplicate code between `internal/` and `pkg/`
- [x] Consolidate security code to `pkg/sandbox/`
- [x] Consolidate parser code to `pkg/scanner/`
- [x] Fix parser ordering bug (sort by StartPos)
- [x] Update imports in `pkg/evaluator/`
- [x] Remove dead code (`isCommandStart` function)
- [x] Replace Python embeddings with Ollama

### Documentation Updates
- [x] Update architecture.md to reflect refactoring
- [x] Update installation guide for Ollama
- [x] Update configuration guide for Ollama
- [x] Update semantic search guide for Ollama

---

## In Progress 🔄

### Test Cleanup
- [ ] Fix test files in `temp_tests/` for new import paths
- [ ] Update tests referencing removed `PythonPath` field
- [ ] Fix parser tests expecting mid-line command matching
- [ ] Fix app test nil pointer issues

---

## Future Ideas 💡

### Config Personas System

One tool, multiple configs = different "modes" for the LLM assistant.

| Config File | Purpose | Features Enabled |
|-------------|---------|------------------|
| `reviewer.yaml` | Safe code review | open, search |
| `developer.yaml` | Full-featured dev | open, write, exec, search |
| `explorer.yaml` | Codebase exploration | open, search |
| `ci.yaml` | CI/CD validation | open, exec (test/build only) |
| `docs.yaml` | Documentation work | open, write (*.md only) |

**Directory Structure:**
```
~/.llm-configs/
├── reviewer.yaml
├── developer.yaml
├── explorer.yaml
└── docs.yaml
```

**Shell Aliases:**
```bash
alias llm-review='llm-runtime --config ~/.llm-configs/reviewer.yaml --interactive'
alias llm-dev='llm-runtime --config ~/.llm-configs/developer.yaml --interactive --exec-enabled'
alias llm-explore='llm-runtime --config ~/.llm-configs/explorer.yaml --interactive'
```

### Sample Config Templates

#### Read-Only Reviewer
```yaml
repository:
  excluded_paths: [".git", ".env", "*.key", "*.pem", "secrets/"]
commands:
  open:
    enabled: true
  write:
    enabled: false
  exec:
    enabled: false
  search:
    enabled: true
```

#### Full Developer
```yaml
repository:
  excluded_paths: [".git", ".env", "*.key"]
commands:
  open:
    enabled: true
    max_file_size: 2097152
  write:
    enabled: true
    backup_before_write: true
  exec:
    enabled: true
    timeout_seconds: 60
    whitelist:
      - "go test"
      - "go build"
      - "npm test"
      - "make"
  search:
    enabled: true
```

#### Documentation Writer
```yaml
commands:
  open:
    enabled: true
  write:
    enabled: true
    allowed_extensions: [".md", ".txt", ".rst"]
  exec:
    enabled: false
  search:
    enabled: true
```

---

## Technical Improvements

### Container Pool (Medium Priority)
Pre-warm Docker containers to eliminate 1-3s startup latency.

```go
type ContainerPool struct {
    containers chan *Container
    size       int
}

func (p *ContainerPool) Get() *Container {
    return <-p.containers
}

func (p *ContainerPool) Return(c *Container) {
    p.containers <- c
}
```

### Streaming Output (Low Priority)
Stream large command outputs instead of buffering entire result.

### MCP Integration (Medium Priority)
Model Context Protocol support for standardized LLM tool integration.

### Additional Commands (Low Priority)
- `<git status>`, `<git diff>` — Version control operations
- `<tree>` — Directory structure visualization
- `<grep pattern>` — Fast text search (complement to semantic search)

---

## CLI Enhancements

### Config Management
```bash
# List available configs
llm-runtime --list-configs

# Generate starter config
llm-runtime --init-config developer

# Validate config file
llm-runtime --validate-config myconfig.yaml
```

### Project Detection
Auto-detect project type and suggest config:
```bash
$ llm-runtime --auto
Detected: Go project (go.mod found)
Suggested config: go-developer.yaml
Use this? [Y/n]
```

---

## Documentation Tasks

- [ ] Create example workflows in `docs/examples/`
- [ ] Add troubleshooting for common Ollama issues
- [ ] Document config persona system when implemented
- [ ] Add architecture diagrams as actual images

---

## Notes

- Config is the control plane — treat it as important as code
- Different configs for different trust levels
- Always use `backup_before_write: true` for refactoring work
- Search is safe to enable broadly — it's read-only
- Exec is the most dangerous — whitelist carefully

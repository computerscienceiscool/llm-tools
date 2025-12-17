# llm-runtime Todo List & Ideas

## In Progress 

### Test Cleanup
- [ ] Fix test files in `temp_tests/` for new import paths
- [ ] Update tests referencing removed `PythonPath` field
- [ ] Fix parser tests expecting mid-line command matching
- [ ] Fix app test nil pointer issues

### Testing Sections
Tests for the following packages are passing:
- ./pkg/config
- ./pkg/scanner
- ./pkg/search
- ./pkg/session

---

## Future Ideas 

### Technical Improvements

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

### Project Detection
Auto-detect project type and suggest config:
```bash
$ llm-runtime --auto
Detected: Go project (go.mod found)
Suggested config: go-developer.yaml
Use this? [Y/n]
```

---

#### Pending Discussion
- [ ] **Container Image Validation** - Need to decide approach with boss
  - Currently: No validation on Docker image names (security risk)
  - Options: Whitelist, digest pinning, private registry, or pattern-based
  - Decision needed: Who controls config? Public/private deployment? How often add new images?
  
- [ ] **Command Injection in Shell Exec** - Architectural decision needed
  - Current: Prefix-based whitelist with shell (flexible but insecure)
  - Trade-off: Security vs Functionality
  - Proposed: Configurable security modes (strict/flexible/unrestricted)
  - Questions to answer:
    * What's our primary use case? (Dev tool? CI/CD? Public API?)
    * What's our threat model? (Trusted LLMs? Potential prompt injection?)
    * Should we support different modes for different deployments?
  - See: `issues-summary.md` 

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

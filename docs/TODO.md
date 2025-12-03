# llm-runtime Todo List & Ideas

## Config Personas System

### Core Idea
One tool, multiple configs = different "modes" or "personas" for the LLM assistant.

### Config Library to Create

| Config File | Purpose | Features Enabled |
|-------------|---------|------------------|
| `read-only-reviewer.yaml` | Safe code review for production/sensitive repos | open, search |
| `full-developer.yaml` | Full-featured development assistant | open, write, exec, search |
| `search-explorer.yaml` | Exploring unfamiliar codebases | open, search only |
| `ci-validator.yaml` | CI/CD and testing focus | open, exec (test/build only) |
| `documentation-writer.yaml` | Creating/updating docs | open, write (*.md only) |
| `refactoring-assistant.yaml` | Code changes with safety nets | open, write (with backups), exec (tests only) |

### Directory Structure

```
~/.llm-configs/
├── reviewer.yaml
├── developer.yaml
├── explorer.yaml
├── ci.yaml
├── docs.yaml
└── refactor.yaml
```

Or project-local:

```
configs/
├── conservative.yaml
├── permissive.yaml
└── custom.yaml
```

### Shell Aliases to Add

```bash
# Add to ~/.bashrc or ~/.zshrc
alias llm-review='llm-runtime --config ~/.llm-configs/reviewer.yaml --interactive'
alias llm-dev='llm-runtime --config ~/.llm-configs/developer.yaml --interactive --exec-enabled'
alias llm-explore='llm-runtime --config ~/.llm-configs/explorer.yaml --interactive'
alias llm-docs='llm-runtime --config ~/.llm-configs/docs.yaml --interactive'
alias llm-ci='llm-runtime --config ~/.llm-configs/ci.yaml --exec-enabled'
```

---

## Sample Config Templates

### 1. Read-Only Reviewer (`reviewer.yaml`)

```yaml
# Safe for reviewing any codebase - no modifications possible
repository:
  excluded_paths: [".git", ".env", "*.key", "*.pem", "secrets/"]

commands:
  open:
    enabled: true
    max_file_size: 1048576
  write:
    enabled: false
  exec:
    enabled: false
  search:
    enabled: true
    max_results: 15

security:
  log_all_operations: true
```

### 2. Full Developer (`developer.yaml`)

```yaml
# Full power for trusted projects
repository:
  excluded_paths: [".git", ".env", "*.key"]

commands:
  open:
    enabled: true
    max_file_size: 2097152
  write:
    enabled: true
    backup_before_write: true
    max_file_size: 204800
  exec:
    enabled: true
    timeout_seconds: 60
    memory_limit: "1g"
    whitelist:
      - "go test"
      - "go build"
      - "go run"
      - "go mod tidy"
      - "go vet"
      - "npm test"
      - "npm run build"
      - "npm install"
      - "python -m pytest"
      - "make"
      - "cargo test"
      - "cargo build"
      - "ls"
      - "find"
      - "grep"
      - "cat"
  search:
    enabled: true
    max_results: 20

logging:
  level: "info"
```

### 3. Search Explorer (`explorer.yaml`)

```yaml
# For unfamiliar codebases - read and search only
repository:
  excluded_paths: [".git", ".env", "*.key", "*.pem", "node_modules"]

commands:
  open:
    enabled: true
    max_file_size: 1048576
  write:
    enabled: false
  exec:
    enabled: false
  search:
    enabled: true
    max_results: 20
    min_similarity_score: 0.4  # More permissive for discovery
```

### 4. Documentation Writer (`docs.yaml`)

```yaml
# For creating and updating documentation
repository:
  excluded_paths: [".git", ".env", "*.key"]

commands:
  open:
    enabled: true
  write:
    enabled: true
    backup_before_write: true
    allowed_extensions:
      - ".md"
      - ".txt"
      - ".rst"
      - ".adoc"
  exec:
    enabled: false
  search:
    enabled: true
```

### 5. CI Validator (`ci.yaml`)

```yaml
# For testing and validation only
repository:
  excluded_paths: [".git", ".env", "*.key", "*.pem"]

commands:
  open:
    enabled: true
  write:
    enabled: false
  exec:
    enabled: true
    timeout_seconds: 120
    whitelist:
      - "go test"
      - "go build"
      - "go vet"
      - "npm test"
      - "npm run build"
      - "python -m pytest"
      - "make test"
      - "make build"
      - "cargo test"
      - "cargo build"
  search:
    enabled: false
```

### 6. Refactoring Assistant (`refactor.yaml`)

```yaml
# Write access but only test commands for verification
repository:
  excluded_paths: [".git", ".env", "*.key"]

commands:
  open:
    enabled: true
  write:
    enabled: true
    backup_before_write: true  # Critical for refactoring!
  exec:
    enabled: true
    whitelist:
      - "go test"
      - "go build"
      - "go vet"
      - "npm test"
      - "python -m pytest"
      - "make test"
  search:
    enabled: true
```

---

## Implementation Tasks

### Phase 1: Create Config Templates
- [ ] Create `~/.llm-configs/` directory
- [ ] Write `reviewer.yaml`
- [ ] Write `developer.yaml`
- [ ] Write `explorer.yaml`
- [ ] Write `docs.yaml`
- [ ] Write `ci.yaml`
- [ ] Write `refactor.yaml`

### Phase 2: Shell Integration
- [ ] Add aliases to shell config
- [ ] Test each alias works correctly
- [ ] Document aliases in README

### Phase 3: Documentation Updates
- [ ] Add "Config Personas" section to README
- [ ] Create `docs/config-personas.md` guide
- [ ] Add examples to quick-reference.md

### Phase 4: Nice-to-Haves
- [ ] `llm-runtime --list-configs` to show available configs
- [ ] `llm-runtime --init-config <persona>` to generate starter config
- [ ] Config validation command: `llm-runtime --validate-config`
- [ ] Include sample configs in `configs/` directory in repo

---

## Future Ideas

### Config Inheritance
```yaml
# developer.yaml
extends: base.yaml
commands:
  exec:
    enabled: true  # Override base
```

### Project Detection
Auto-detect project type and suggest config:
```bash
$ llm-runtime --auto
Detected: Go project (go.mod found)
Suggested config: go-developer.yaml
Use this? [Y/n]
```

### Config Sharing
```bash
# Export config for team
llm-runtime --export-config > team-standard.yaml

# Import shared config
llm-runtime --import-config https://example.com/team.yaml
```

---

---

## Technical Improvements

### TODO: Make Python Path Configurable via Environment Variable

**Priority:** Low (current hardcoded path works for demo)

**Problem:** Python path is hardcoded in config, but users may have different virtual environments or Python installations.

**Solution:** Implement cascading fallback for python_path:

1. Check for `LLM_RUNTIME_PYTHON_PATH` environment variable first
2. Fall back to config file value if env var not set
3. Fall back to `python3` if neither is set

**Implementation:**
```go
func getPythonPath(config Config) string {
    if envPath := os.Getenv("LLM_RUNTIME_PYTHON_PATH"); envPath != "" {
        return envPath
    }
    if config.Commands.Search.PythonPath != "" {
        return config.Commands.Search.PythonPath
    }
    return "python3"
}
```

**Example usage:**
```bash
export LLM_RUNTIME_PYTHON_PATH=~/venvs/search-env/bin/python3
./llm-runtime --interactive
```

**Documentation updates needed:**
- [ ] Add env var to installation guide
- [ ] Add to configuration.md
- [ ] Add to troubleshooting.md (Python issues section)
- [ ] Update quick-reference.md

---

## Notes

- Config is the control plane - treat it as important as code
- Different configs for different trust levels
- Always use `backup_before_write: true` for refactoring
- Search is safe to enable broadly - it's read-only
- Exec is the most dangerous - whitelist carefully

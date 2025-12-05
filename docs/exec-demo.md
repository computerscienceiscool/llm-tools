# llm-runtime Exec Feature Demo Cheatsheet

## Setup
```bash
cd /path/to/llm-runtime
make build
```

---

## Pipe Mode Demos

### Basic file reading
```bash
echo "<open README.md>" | ./llm-runtime
```

### Exec blocked without flag
```bash
echo "<exec ls>" | ./llm-runtime
```

### Enable exec, simple command
```bash
echo "<exec ls>" | ./llm-runtime --exec-enabled
```

### List files in workspace
```bash
echo "<exec ls -la>" | ./llm-runtime --exec-enabled
```

### Check Go version in container
```bash
echo "<exec go version>" | ./llm-runtime --exec-enabled
```

### Check Python version
```bash
echo "<exec python3 --version>" | ./llm-runtime --exec-enabled
```

### Run Python code
```bash
echo "<exec python3 -c 'print(2+2)'>" | ./llm-runtime --exec-enabled
```

### View file contents
```bash
echo "<exec cat /etc/os-release>" | ./llm-runtime --exec-enabled
```

### View specific file from repo
```bash
echo "<exec cat go.mod>" | ./llm-runtime --exec-enabled
```

### Count lines in file
```bash
echo "<exec wc -l README.md>" | ./llm-runtime --exec-enabled
```

### Find Go files
```bash
echo "<exec find . -name '*.go'>" | ./llm-runtime --exec-enabled
```

### Head of a file
```bash
echo "<exec head -10 README.md>" | ./llm-runtime --exec-enabled
```

### Tail of a file
```bash
echo "<exec tail -10 README.md>" | ./llm-runtime --exec-enabled
```

### Grep for pattern
```bash
echo "<exec grep -r 'func main' .>" | ./llm-runtime --exec-enabled
```

---

## Security Demos

### Whitelist blocks dangerous commands
```bash
echo "<exec rm -rf />" | ./llm-runtime --exec-enabled
```
**Expected:** Blocked - "command not in whitelist: rm"

### Network is disabled (go test fails - can't download deps)
```bash
echo "<exec go test ./...>" | ./llm-runtime --exec-enabled
```
**Expected:** Fails with "network is unreachable" - proves isolation

---

## Multiple Commands
```bash
echo "<open go.mod> <exec go version>" | ./llm-runtime --exec-enabled
```

```bash
echo "<exec ls> <exec cat go.mod>" | ./llm-runtime --exec-enabled
```

---

## Interactive Mode

### Start interactive mode
```bash
./llm-runtime --interactive --exec-enabled
```

### Commands to type once inside:

```
<exec ls>
```

```
<exec ls -la>
```

```
<exec go version>
```

```
<exec python3 --version>
```

```
<exec python3 -c 'print("Hello from Docker")'>
```

```
<exec cat /etc/os-release>
```

```
<exec cat go.mod>
```

```
<exec head -5 README.md>
```

```
<exec find . -name '*.go' | head -10>
```

```
<exec wc -l README.md>
```

```
<open README.md>
```

### Exit interactive mode
```
Ctrl+D
```

---

## Exploring the Container Manually

### Start container interactively
```bash
docker run -it --rm python-go bash
```

### Start with repo mounted (like llm-runtime does)
```bash
docker run -it --rm -v $(pwd):/workspace:ro -w /workspace python-go bash
```

### Commands to run inside container:
```bash
ls -la
go version
python3 --version
cat /etc/os-release
```

### Exit container
```bash
exit
```

---

## Key Security Points

| Security Feature | Demo Command | Expected Result |
|------------------|--------------|-----------------|
| Whitelist | `<exec rm -rf />` | Blocked: not in whitelist |
| No network | `<exec go test ./...>` | Fails: network unreachable |
| Container isolated | `<exec cat /etc/os-release>` | Shows Ubuntu, not host OS |

---

## Whitelisted Commands (from config)

**Go:** `go test`, `go build`, `go run`, `go version`

**Python:** `python`, `python3`, `python -m pytest`, `python3 -m pytest`

**Node:** `node`, `npm test`, `npm run build`, `npm install`

**Rust:** `cargo build`, `cargo test`, `cargo run`

**Build:** `make`, `make test`, `make build`

**System:** `ls`, `cat`, `grep`, `find`, `head`, `tail`, `wc`

**Package:** `pip install`, `pip3 install`

---

## Files Involved (for code walkthrough)

1. `internal/command/parser.go` — parses `<exec>` from text
2. `internal/executor/executor.go` — routes to exec handler
3. `internal/executor/exec.go` — orchestrates execution
4. `internal/security/exec_validation.go` — whitelist check
5. `internal/docker/client.go` — Docker availability check
6. `internal/docker/container.go` — builds and runs container

---

## Troubleshooting

### Docker not found
```bash
docker version
```

### Image not available
```bash
docker images | grep python-go
```

### Check running containers (should be empty after exec)
```bash
docker ps
```

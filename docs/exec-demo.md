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
echo "<exec echo hello>" | ./llm-runtime
```

### Enable exec, simple command
```bash
echo "<exec echo hello>" | ./llm-runtime --exec-enabled
```

### Show repo is mounted at /workspace
```bash
echo "<exec ls -la>" | ./llm-runtime --exec-enabled
```

### Show working directory
```bash
echo "<exec pwd>" | ./llm-runtime --exec-enabled
```

### Whitelist blocks dangerous commands
```bash
echo "<exec rm -rf />" | ./llm-runtime --exec-enabled
```

### No network access
```bash
echo "<exec curl google.com>" | ./llm-runtime --exec-enabled
```

### Read-only repo (can't write)
```bash
echo "<exec touch /workspace/hacked.txt>" | ./llm-runtime --exec-enabled
```

### Run actual Go tests
```bash
echo "<exec go test ./...>" | ./llm-runtime --exec-enabled
```

### Check Go version in container
```bash
echo "<exec go version>" | ./llm-runtime --exec-enabled
```

### Multiple commands in one pass
```bash
echo "<open go.mod> <exec go version>" | ./llm-runtime --exec-enabled
```

---

## Interactive Mode

### Start interactive mode
```bash
./llm-runtime --interactive --exec-enabled
```

### Commands to type once inside:

```
<open README.md>
```

```
<exec echo "Hello from Docker">
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
<exec go test ./...>
```

```
<open go.mod>
```

```
<exec cat /etc/os-release>
```

### Exit interactive mode
```
Ctrl+D
```

---

## Exploring the Container Manually

### Start container interactively (for poking around)
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
pwd
cat /etc/os-release
go version
python3 --version
whoami
ping google.com      # will fail - no network
touch test.txt       # will fail in /workspace - read-only
```

### Exit container
```bash
exit
```

---

## Key Security Points to Demonstrate

| Security Feature | Demo Command | Expected Result |
|------------------|--------------|-----------------|
| Whitelist | `<exec rm -rf />` | Blocked: not in whitelist |
| No network | `<exec curl google.com>` | Fails: no network |
| Read-only repo | `<exec touch /workspace/x>` | Fails: read-only filesystem |
| Non-root user | `<exec whoami>` | Shows user 1000, not root |
| Isolated environment | `<exec cat /etc/os-release>` | Shows Ubuntu, not host OS |
| Container destroyed | Run any exec | Container gone after (check `docker ps`) |

---

## Troubleshooting

### Docker not found
```bash
docker version
```

### Image not available
```bash
docker images | grep python-go
docker pull python-go  # or build it
```

### Permission denied
```bash
sudo usermod -aG docker $USER
newgrp docker
```

### Check running containers (should be empty after exec)
```bash
docker ps
```

---

## Files Involved (for code walkthrough)

1. `internal/command/parser.go` — parses `<exec>` from text
2. `internal/executor/executor.go` — routes to exec handler
3. `internal/executor/exec.go` — orchestrates execution
4. `internal/security/exec_validation.go` — whitelist check
5. `internal/docker/client.go` — Docker availability check
6. `internal/docker/container.go` — builds and runs container

---

## Quick Reference: Security Flags

```bash
docker run \
    --rm                              # Delete container when done
    --network none                    # No internet access
    --user 1000:1000                  # Non-root user
    --cap-drop ALL                    # Drop all Linux capabilities
    --security-opt no-new-privileges  # No privilege escalation
    --read-only                       # Read-only container filesystem
    --memory 512m                     # Memory limit
    --cpus 1                          # CPU limit
    -v /repo:/workspace:ro            # Mount repo read-only
    python-go                         # Image name
    sh -c "command"                   # Command to run
```


# Docker Cheat Sheet for llm-runtime

A beginner-friendly guide for using Docker with llm-runtime.

## What is Docker?

Docker is a tool that runs applications in isolated "containers." Think of a container as a lightweight virtual computer that:

- Has its own operating system (Ubuntu 22.04 in our case)
- Is completely separate from your main computer
- Gets destroyed after each use
- Cannot access the internet (for security)
- Can only see files you explicitly share with it

## Why Does llm-runtime Use Docker?

**Security.** When an LLM runs commands via `<exec>`, those commands run inside a Docker container, NOT on your actual computer. This means:

| Without Docker | With Docker |
|----------------|-------------|
| `<exec rm -rf />` could destroy your system | Command runs in disposable container - your system is safe |
| Commands have full network access | No network access - can't download malware |
| Commands run as your user | Commands run as unprivileged user |
| No resource limits | Memory and CPU limits enforced |

**In short:** Docker lets LLMs execute code safely without risking your computer.

## First-Time Setup

### 1. Install Docker

**Ubuntu/Debian:**
```bash
curl -fsSL https://get.docker.com | sh
```

**macOS:**
```bash
brew install --cask docker
# Then open Docker Desktop from Applications
```

**Windows:**
Download Docker Desktop from https://docker.com

### 2. Fix Permissions (Linux only)

```bash
sudo usermod -aG docker $USER
newgrp docker
```

### 3. Verify Installation

```bash
docker version
```

You should see version info for both Client and Server. If you see "permission denied," repeat step 2.

### 4. Test Docker

```bash
docker run --rm hello-world
```

You should see "Hello from Docker!" - this means Docker is working.

### 5. Pull the Ubuntu Image

```bash
docker pull ubuntu:22.04
```

This downloads the Ubuntu image that llm-runtime uses. Do this once to avoid delays later.

## Using Docker with llm-runtime

### Basic Command Execution

Run a command inside Docker via llm-runtime:

```bash
echo "<exec ls>" | ./llm-runtime --exec-enabled --exec-cpu 1
```

### Verify It's Running in Docker

```bash
echo "<exec cat /etc/hostname>" | ./llm-runtime --exec-enabled --exec-cpu 1
```

If you see a random string like `b1ad5aa37b4c` (not your computer name), it's running in Docker.

### Check the Container's OS

```bash
echo "<exec cat /etc/os-release>" | ./llm-runtime --exec-enabled --exec-cpu 1
```

Should show "Ubuntu 22.04" regardless of what OS you're actually using.

## Manual Docker Commands

Sometimes you want to explore a container yourself, without llm-runtime.

### Open an Interactive Shell

```bash
docker run -it --rm -v $(pwd):/workspace:ro ubuntu:22.04 bash
```

This gives you a bash prompt inside the container. Your current directory is mounted at `/workspace`.

**What the flags mean:**
- `-it` = interactive terminal
- `--rm` = delete container when you exit
- `-v $(pwd):/workspace:ro` = mount current directory as read-only
- `ubuntu:22.04` = the image to use
- `bash` = the command to run

### Commands to Try Inside the Container

Once inside (you'll see a prompt like `root@abc123:/#`):

```bash
# See your mounted files
ls /workspace

# Check the OS
cat /etc/os-release

# Check the hostname (random container ID)
hostname

# Exit the container
exit
```

### Run a Single Command

```bash
docker run --rm ubuntu:22.04 echo "Hello from Docker"
```

### Run a Command with Your Repo Mounted

```bash
docker run --rm -v $(pwd):/workspace:ro ubuntu:22.04 ls /workspace
```

## Troubleshooting

### "permission denied" Error

**Problem:**
```
permission denied while trying to connect to the Docker daemon socket
```

**Solution:**
```bash
sudo usermod -aG docker $USER
newgrp docker
```

If that doesn't work, log out and log back in.

### "Cannot connect to Docker daemon" Error

**Problem:**
```
Cannot connect to the Docker daemon at unix:///var/run/docker.sock
```

**Solution:** Start Docker:

```bash
# Linux
sudo systemctl start docker

# macOS
open -a Docker
```

### "range of CPUs" Error

**Problem:**
```
range of CPUs is from 0.01 to 1.00, as there are only 1 CPUs available
```

**Solution:** Your machine has 1 CPU. Use:
```bash
./llm-runtime --exec-enabled --exec-cpu 1
```

Or update `llm-runtime.config.yaml`:
```yaml
exec:
  cpu_limit: 1
```

### Container Runs Slowly the First Time

**Problem:** First `<exec>` command takes 30+ seconds.

**Solution:** Pre-pull the image:
```bash
docker pull ubuntu:22.04
```

### "image not found" Error

**Solution:**
```bash
docker pull ubuntu:22.04
```

## Useful Docker Commands

### Check Running Containers

```bash
docker ps
```

### Check All Containers (including stopped)

```bash
docker ps -a
```

### List Downloaded Images

```bash
docker images
```

### Remove Old Images

```bash
docker image prune
```

### Check Docker Disk Usage

```bash
docker system df
```

### Clean Up Everything

```bash
docker system prune -a
```

**Warning:** This removes all unused images, containers, and networks.

## How llm-runtime Uses Docker

When you run:
```bash
echo "<exec go test>" | ./llm-runtime --exec-enabled
```

Behind the scenes, llm-runtime runs:
```bash
docker run \
    --rm \
    --network none \
    --user 1000:1000 \
    --cap-drop ALL \
    --security-opt no-new-privileges \
    --read-only \
    --tmpfs /tmp \
    --memory 512m \
    --cpus 1 \
    -v /your/repo:/workspace:ro \
    ubuntu:22.04 \
    sh -c "go test"
```

**Security flags explained:**
| Flag | Purpose |
|------|---------|
| `--rm` | Delete container after use |
| `--network none` | No internet access |
| `--user 1000:1000` | Run as non-root user |
| `--cap-drop ALL` | Remove all Linux capabilities |
| `--security-opt no-new-privileges` | Prevent privilege escalation |
| `--read-only` | Container filesystem is read-only |
| `--memory 512m` | Limit RAM to 512MB |
| `--cpus 1` | Limit to 1 CPU |
| `-v ...:/workspace:ro` | Mount repo as read-only |

## Quick Reference

| Task | Command |
|------|---------|
| Check Docker is installed | `docker version` |
| Test Docker works | `docker run --rm hello-world` |
| Pull Ubuntu image | `docker pull ubuntu:22.04` |
| Interactive shell | `docker run -it --rm -v $(pwd):/workspace:ro ubuntu:22.04 bash` |
| Run llm-runtime exec | `echo "<exec ls>" \| ./llm-runtime --exec-enabled --exec-cpu 1` |
| List running containers | `docker ps` |
| List images | `docker images` |
| Clean up | `docker system prune` |

## Need Help?

- Docker documentation: https://docs.docker.com
- llm-runtime issues: https://github.com/computerscienceiscool/llm-runtime/issues

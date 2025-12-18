# Docker Cheatsheet

Quick reference for Docker operations related to llm-runtime.

## Required Images

### I/O Container (Phase 5)
```bash
# Build custom I/O container
make build-io-image

# Or manually
docker build -t llm-runtime-io:latest -f Dockerfile.io .

# Verify image exists
docker images | grep llm-runtime-io

# Check image size (should be ~5-10MB)
docker images llm-runtime-io:latest --format "{{.Size}}"

# Inspect image
docker image inspect llm-runtime-io:latest
```

### Exec Container
```bash
# Pull Python+Go combined image
docker pull python-go:latest

# Or use alternative images
docker pull golang:1.21
docker pull node:18-alpine
docker pull python:3.11-alpine

# List all exec-compatible images
docker images | grep -E "python|golang|node"
```

### Fallback Images
```bash
# Pull Alpine (I/O fallback)
docker pull alpine:latest

# Pull Ubuntu (alternative exec)
docker pull ubuntu:22.04
```

## Image Management

### Building Images

**I/O Container (Minimal Alpine):**
```bash
# Using Makefile
make build-io-image

# Manual build
docker build -t llm-runtime-io:latest -f Dockerfile.io .

# Build with no cache
docker build --no-cache -t llm-runtime-io:latest -f Dockerfile.io .

# Build and tag with version
docker build -t llm-runtime-io:1.0.0 -t llm-runtime-io:latest -f Dockerfile.io .
```

**Custom Exec Container:**
```bash
# Example: Build custom Python+Go+Node container
docker build -t my-exec-env:latest -f Dockerfile.exec .
```

### Pulling Images
```bash
# Pull specific version
docker pull alpine:3.19

# Pull all required images
docker pull alpine:latest
docker pull python-go:latest

# Pull with progress
docker pull --progress=plain alpine:latest
```

### Listing Images
```bash
# List all images
docker images

# List llm-runtime images
docker images | grep -E "llm-runtime|python-go|alpine"

# List with sizes
docker images --format "table {{.Repository}}\t{{.Tag}}\t{{.Size}}"

# List image IDs only
docker images -q
```

### Inspecting Images
```bash
# Detailed image info
docker image inspect llm-runtime-io:latest

# Check image layers
docker history llm-runtime-io:latest

# Check image size
docker images llm-runtime-io:latest --format "{{.Size}}"

# View image metadata
docker image inspect llm-runtime-io:latest --format '{{json .Config}}'
```

### Removing Images
```bash
# Remove specific image
docker rmi llm-runtime-io:latest

# Remove with force
docker rmi -f llm-runtime-io:latest

# Remove all unused images
docker image prune -a

# Remove all llm-runtime images
docker images | grep llm-runtime | awk '{print $3}' | xargs docker rmi
```

## Container Management

### Running Containers

**Test I/O Container:**
```bash
# Test read operation
docker run --rm \
  -v $(pwd):/workspace:ro \
  llm-runtime-io:latest \
  cat /workspace/README.md

# Test write operation (to temp location)
docker run --rm \
  -v $(pwd):/workspace \
  llm-runtime-io:latest \
  sh -c 'echo "test" > /workspace/test.txt'

# Test with resource limits
docker run --rm \
  --memory 128m \
  --cpus 1 \
  -v $(pwd):/workspace:ro \
  llm-runtime-io:latest \
  cat /workspace/config.yaml
```

**Test Exec Container:**
```bash
# Test Go commands
docker run --rm \
  -v $(pwd):/workspace:ro \
  python-go:latest \
  go version

# Test Python commands
docker run --rm \
  -v $(pwd):/workspace:ro \
  python-go:latest \
  python3 --version

# Run tests in container
docker run --rm \
  --network none \
  --memory 512m \
  --cpus 2 \
  -v $(pwd):/workspace:ro \
  python-go:latest \
  go test ./...
```

**Interactive Container Session:**
```bash
# Start interactive shell in I/O container
docker run -it --rm \
  -v $(pwd):/workspace:ro \
  llm-runtime-io:latest \
  sh

# Start interactive shell in exec container
docker run -it --rm \
  -v $(pwd):/workspace:ro \
  python-go:latest \
  bash
```

### Listing Containers
```bash
# List running containers
docker ps

# List all containers (including stopped)
docker ps -a

# List llm-runtime containers
docker ps -a | grep llm-runtime

# Show last created container
docker ps -l

# Show container IDs only
docker ps -q
```

### Inspecting Containers
```bash
# Detailed container info
docker inspect <container_id>

# Check container logs
docker logs <container_id>

# Follow container logs
docker logs -f <container_id>

# Check container resource usage
docker stats <container_id>

# View container processes
docker top <container_id>
```

### Stopping Containers
```bash
# Stop specific container
docker stop <container_id>

# Stop all containers
docker stop $(docker ps -q)

# Force stop
docker kill <container_id>

# Stop and remove
docker rm -f <container_id>
```

### Removing Containers
```bash
# Remove specific container
docker rm <container_id>

# Remove all stopped containers
docker container prune

# Remove all containers (careful!)
docker rm -f $(docker ps -aq)

# Remove containers older than 24h
docker container prune --filter "until=24h"
```

## Resource Management

### Disk Space

**Check Usage:**
```bash
# Overall Docker disk usage
docker system df

# Detailed breakdown
docker system df -v

# Check specific component sizes
docker images --format "table {{.Repository}}\t{{.Size}}"
docker ps -s
```

**Cleanup:**
```bash
# Remove unused containers
docker container prune

# Remove unused images
docker image prune

# Remove unused volumes
docker volume prune

# Remove unused networks
docker network prune

# Clean everything (CAREFUL!)
docker system prune -a

# Clean with size limit
docker system prune -a --volumes
```

### Memory and CPU

**Monitor Resource Usage:**
```bash
# Real-time stats for all containers
docker stats

# Stats for specific container
docker stats <container_id>

# One-time snapshot
docker stats --no-stream

# Format output
docker stats --format "table {{.Name}}\t{{.CPUPerc}}\t{{.MemUsage}}"
```

**Set Resource Limits:**
```bash
# Memory limit
docker run --rm --memory 128m alpine:latest

# CPU limit
docker run --rm --cpus 2 alpine:latest

# Both
docker run --rm --memory 512m --cpus 2 alpine:latest

# Memory + swap limit
docker run --rm --memory 512m --memory-swap 1g alpine:latest
```

## Network Operations

### Network Configuration
```bash
# List networks
docker network ls

# Create isolated network
docker network create --driver bridge my-network

# Run container with no network (llm-runtime default)
docker run --rm --network none alpine:latest

# Inspect network
docker network inspect bridge
```

### Port Mapping (Not used in llm-runtime)
```bash
# Map port (for reference)
docker run -p 8080:80 nginx

# Map all ports
docker run -P nginx
```

## Volume Management

### Volume Operations
```bash
# List volumes
docker volume ls

# Create volume
docker volume create my-volume

# Remove volume
docker volume rm my-volume

# Remove all unused volumes
docker volume prune

# Inspect volume
docker volume inspect my-volume
```

### Bind Mounts (Used by llm-runtime)
```bash
# Read-only mount (I/O reads, exec operations)
docker run --rm -v $(pwd):/workspace:ro alpine:latest

# Read-write mount (I/O writes)
docker run --rm -v $(pwd):/workspace:rw alpine:latest

# Multiple mounts
docker run --rm \
  -v $(pwd):/workspace:ro \
  -v /tmp:/tmp:rw \
  alpine:latest
```

## Security Options

### Security Hardening (llm-runtime defaults)
```bash
# Run with all security options
docker run --rm \
  --network none \
  --user 1000:1000 \
  --cap-drop ALL \
  --security-opt no-new-privileges \
  --read-only \
  --tmpfs /tmp \
  --memory 128m \
  --cpus 1 \
  -v $(pwd):/workspace:ro \
  llm-runtime-io:latest \
  cat /workspace/file.txt
```

### User Configuration
```bash
# Run as non-root user
docker run --rm --user 1000:1000 alpine:latest

# Run as specific user:group
docker run --rm --user $(id -u):$(id -g) alpine:latest
```

### Capability Management
```bash
# Drop all capabilities
docker run --rm --cap-drop ALL alpine:latest

# Drop specific capability
docker run --rm --cap-drop NET_RAW alpine:latest

# Add specific capability (not recommended)
docker run --rm --cap-add SYS_ADMIN alpine:latest
```

## Debugging

### Troubleshooting Docker

**Check Docker Status:**
```bash
# Docker daemon status
systemctl status docker

# Docker version
docker --version
docker version

# Docker system info
docker info

# Docker daemon logs
journalctl -u docker -f
```

**Test Docker Functionality:**
```bash
# Basic test
docker run --rm hello-world

# Test with Alpine
docker run --rm alpine:latest echo "Docker works"

# Test with network disabled
docker run --rm --network none alpine:latest echo "Isolated"

# Test with resource limits
docker run --rm --memory 64m --cpus 0.5 alpine:latest echo "Limited"
```

**Permission Issues:**
```bash
# Check if user is in docker group
groups $USER | grep docker

# Add user to docker group
sudo usermod -aG docker $USER

# Reload group membership
newgrp docker

# Test access
docker run --rm alpine:latest echo "Access OK"
```

### Container Debugging

**Enter Running Container:**
```bash
# Execute shell in running container
docker exec -it <container_id> sh

# Run command in running container
docker exec <container_id> ls -la /workspace

# Check environment
docker exec <container_id> env
```

**Inspect Container Filesystem:**
```bash
# Copy file from container
docker cp <container_id>:/path/to/file ./local-file

# Copy file to container
docker cp ./local-file <container_id>:/path/to/file

# View container changes
docker diff <container_id>
```

**Debug Failed Containers:**
```bash
# Keep container after exit
docker run -it alpine:latest sh
# (don't use --rm)

# Check exit code
docker inspect <container_id> --format='{{.State.ExitCode}}'

# View last logs
docker logs --tail 50 <container_id>

# View logs with timestamps
docker logs -t <container_id>
```

## Performance Optimization

### Image Optimization
```bash
# Use multi-stage builds
FROM golang:1.21 AS builder
# ... build steps ...
FROM alpine:latest
COPY --from=builder /app/binary /binary

# Use smaller base images
# alpine:latest (~7MB) vs ubuntu:22.04 (~77MB)

# Clean up in same layer
RUN apk add --no-cache git && \
    git clone ... && \
    apk del git
```

### Container Performance
```bash
# Pre-pull images
docker pull alpine:latest
docker pull python-go:latest

# Check image layers
docker history llm-runtime-io:latest --no-trunc

# Analyze image size
docker images --format "{{.Repository}}:{{.Tag}} {{.Size}}"
```

### Caching
```bash
# Build with cache
docker build -t llm-runtime-io:latest -f Dockerfile.io .

# Build without cache
docker build --no-cache -t llm-runtime-io:latest -f Dockerfile.io .

# Pull to update cache
docker pull alpine:latest
```

## Common llm-runtime Patterns

### I/O Container Operations

**File Reading:**
```bash
# Single file
docker run --rm \
  --network none \
  --user 1000:1000 \
  --cap-drop ALL \
  --security-opt no-new-privileges \
  --read-only \
  --memory 128m \
  --cpus 1 \
  -v $(pwd):/workspace:ro \
  llm-runtime-io:latest \
  cat /workspace/README.md
```

**File Writing:**
```bash
# Atomic write pattern
docker run --rm \
  --network none \
  --user 1000:1000 \
  --cap-drop ALL \
  --security-opt no-new-privileges \
  --read-only \
  --tmpfs /tmp \
  --memory 128m \
  --cpus 1 \
  -v $(pwd):/workspace \
  llm-runtime-io:latest \
  sh -c 'echo "content" > /workspace/file.tmp && mv /workspace/file.tmp /workspace/file.txt'
```

### Exec Container Operations

**Run Tests:**
```bash
docker run --rm \
  --network none \
  --user 1000:1000 \
  --cap-drop ALL \
  --security-opt no-new-privileges \
  --read-only \
  --tmpfs /tmp \
  --memory 512m \
  --cpus 2 \
  -v $(pwd):/workspace:ro \
  python-go:latest \
  go test ./...
```

**Build Project:**
```bash
docker run --rm \
  --network none \
  --user 1000:1000 \
  --cap-drop ALL \
  --security-opt no-new-privileges \
  --read-only \
  --tmpfs /tmp \
  --tmpfs /workspace/bin \
  --memory 1g \
  --cpus 4 \
  -v $(pwd):/workspace:ro \
  python-go:latest \
  go build -o /tmp/app .
```

## Quick Reference Commands

### Daily Operations
```bash
# Check what's running
docker ps

# Clean up
docker system prune

# Check disk usage
docker system df

# Pull latest images
docker pull alpine:latest
docker pull python-go:latest

# Rebuild I/O image
make build-io-image
```

### Troubleshooting
```bash
# Docker not working?
systemctl status docker
docker run --rm hello-world

# Image not found?
docker images | grep llm-runtime-io
make build-io-image

# Permission issues?
groups $USER | grep docker
sudo usermod -aG docker $USER

# Disk space?
docker system df
docker system prune -a
```

### Emergency Cleanup
```bash
# Stop everything
docker stop $(docker ps -q)

# Remove all containers
docker container prune -f

# Remove all images
docker image prune -a -f

# Remove all volumes
docker volume prune -f

# Nuclear option (CAREFUL!)
docker system prune -a -f --volumes
```

## Environment Variables

```bash
# Docker daemon socket
export DOCKER_HOST=unix:///var/run/docker.sock

# Docker config directory
export DOCKER_CONFIG=/home/user/.docker

# Build kit (faster builds)
export DOCKER_BUILDKIT=1

# Compose file
export COMPOSE_FILE=docker-compose.yml
```

## Best Practices

### DO:
✅ Pre-pull images before using llm-runtime
✅ Use `--rm` flag for temporary containers
✅ Set resource limits (--memory, --cpus)
✅ Use `--network none` for security
✅ Run as non-root user (--user 1000:1000)
✅ Clean up regularly (docker system prune)
✅ Use specific image tags (alpine:3.19)
✅ Monitor disk usage (docker system df)

### DON'T:
❌ Run containers as root
❌ Leave unused containers running
❌ Ignore disk space warnings
❌ Use :latest in production
❌ Grant unnecessary capabilities
❌ Allow network access unless needed
❌ Forget to clean up old images
❌ Run without resource limits

## Makefile Integration

```bash
# Build I/O image
make build-io-image

# Check if I/O image exists
make check-io-image

# Test I/O container
make test-io-container

# Clean I/O image
make clean-io-image

# Pull all required images
make pull-images

# Clean all Docker resources
make docker-clean
```

## Additional Resources

- Docker documentation: https://docs.docker.com
- Docker security: https://docs.docker.com/engine/security/
- Best practices: https://docs.docker.com/develop/dev-best-practices/
- Alpine Linux: https://alpinelinux.org/

---

**Quick Start:**
```bash
# 1. Build I/O image
make build-io-image

# 2. Pull exec image
docker pull python-go:latest

# 3. Verify
docker images | grep -E "llm-runtime-io|python-go"

# 4. Test
echo "<open README.md>" | ./llm-runtime
```

# Installation Guide

Complete installation guide for llm-runtime across different operating systems and environments.

## Prerequisites

### Required
- **Go 1.21 or later** - For building the tool
- **Git** - For cloning the repository
- **Docker** - Required for all operations (file I/O and command execution)

### Optional (for specific features)
- **Ollama** - Required for `<search>` semantic search functionality

## Quick Installation

```bash
git clone https://github.com/computerscienceiscool/llm-runtime.git
cd llm-runtime
make build
./llm-runtime --help
```

## Manual Installation

**Important**: Docker is required for all operations (file I/O and command execution). Install Docker first before proceeding.

### 1. Install Go

#### Linux (Ubuntu/Debian)
```bash
# Option 1: Package manager (may be older version)
sudo apt update
sudo apt install golang-go

# Option 2: Official installer (recommended)
wget https://go.dev/dl/go1.21.0.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.21.0.linux-amd64.tar.gz
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
source ~/.bashrc
```

#### macOS
```bash
# Option 1: Homebrew (recommended)
brew install go

# Option 2: Official installer
# Download from https://go.dev/dl/
# Install the .pkg file
```

#### Windows
```bash
# Option 1: Chocolatey
choco install golang

# Option 2: Official installer
# Download from https://go.dev/dl/
# Run the .msi installer
```

### 2. Clone and Build
```bash
# Clone repository
git clone https://github.com/computerscienceiscool/llm-runtime.git
cd llm-runtime

# Download dependencies
go mod download

# Build the tool
go build -o llm-runtime ./cmd/llm-runtime

# Or use make
make build

# Verify build
./llm-runtime --help
```

### 3. Install Docker (required for all operations)

Docker is required for:
- Command execution (`<exec>` commands)
- File I/O operations (containerized reads/writes)
- Enhanced security through isolation

#### Linux (Ubuntu/Debian)
```bash
# Install Docker
curl -fsSL https://get.docker.com | sh

# Add user to docker group (avoid sudo)
sudo usermod -aG docker $USER
newgrp docker

# Test Docker
docker run hello-world
```

#### macOS
```bash
# Install Docker Desktop
brew install --cask docker

# Or download from https://docker.com
# Start Docker Desktop application
```

#### Windows
```bash
# Install Docker Desktop
# Download from https://docker.com
# Ensure WSL2 is enabled
```

#### Build I/O Container Image (Optional)

The tool can use a minimal Alpine-based container for file operations:

```bash
# Build the I/O container image
make build-io-image

# Verify it's built
make check-io-image
```

**Note**: If you don't build this image, the tool will use `alpine:latest` by default, which will be pulled automatically from Docker Hub.

### 4. Install Ollama (for search)

#### Linux
```bash
# Install Ollama
curl -fsSL https://ollama.com/install.sh | sh

# Pull the embedding model
ollama pull nomic-embed-text

# Verify installation
ollama list
```

#### macOS
```bash
# Install via Homebrew
brew install ollama

# Or download from https://ollama.com
# Pull the embedding model
ollama pull nomic-embed-text
```

#### Windows
```bash
# Download from https://ollama.com
# Run installer
# Pull the embedding model
ollama pull nomic-embed-text
```

## Verification

### 1. Test Basic Functionality

**Note**: By default, llm-runtime creates a temporary repository in `/tmp/dynamic-repo/`. To test with your actual project:
```bash
# Test with default dynamic repository (may not have README.md)
echo "<exec echo 'Hello from llm-runtime'>" | ./llm-runtime

# Test with specific repository
echo "<open README.md>" | ./llm-runtime --root /path/to/your/project
```

### 2. Test Docker Integration (if installed)
```bash
echo "<exec echo 'Docker works'>" | ./llm-runtime
```

### 3. Test Search (if Ollama installed)
```bash
# Build search index
./llm-runtime --reindex

# Test search
echo "<search configuration>" | ./llm-runtime
```

### 4. Run Test Suite
```bash
make test
```

## Troubleshooting Installation

### Go Version Issues
```bash
# Check Go version
go version

# Should show go1.21 or later
# If too old, install newer version from https://go.dev/dl/
```

### Permission Issues (Linux/macOS)
```bash
# Make sure tool is executable
chmod +x llm-runtime

# Docker permission issues
sudo usermod -aG docker $USER
newgrp docker
```

### Docker Issues
```bash
# Check Docker is running
docker --version
docker info

# Test basic Docker functionality
docker run --rm ubuntu:22.04 echo "Docker test"

# Pull required image
docker pull ubuntu:22.04
```

### Ollama Issues
```bash
# Check Ollama is running
ollama list

# Start Ollama if needed
ollama serve

# Pull embedding model
ollama pull nomic-embed-text

# Test embedding endpoint
curl http://localhost:11434/api/tags
```

### Build Issues
```bash
# Clean and rebuild
make clean
go mod tidy
make build

# Check for missing dependencies
go mod download
```

## Updating

### Update the Tool
```bash
# Pull latest changes
git pull origin main

# Rebuild
make build

# Update search index if needed
./llm-runtime --reindex
```

## Next Steps

After successful installation:
1. **Build I/O container** (optional): `make build-io-image`
2. **Understand repository behavior**: By default, operations occur in `/tmp/dynamic-repo/`. Use `--root` to specify your project directory
3. **Configure search** (optional): Install Ollama and run `./llm-runtime --reindex --root /path/to/project`
4. Read the [configuration guide](configuration.md) to customize settings
5. Check [troubleshooting](troubleshooting.md) for common issues
6. See [llm-runtime-overview.md](llm-runtime-overview.md) to understand all features
7. Review [SYSTEM_PROMPT.md](SYSTEM_PROMPT.md) for LLM integration

## Feature Summary

| Feature | Requirement | Test Command |
|---------|-------------|--------------|
| File reading | Go + Docker (containerized I/O) | `echo "<open README.md>" \| ./llm-runtime` |
| File writing | Go + Docker (containerized I/O) | `echo "<write test.txt>hello</write>" \| ./llm-runtime` |
| Command execution | Docker | `echo "<exec ls>" \| ./llm-runtime` |
| Semantic search | Ollama | `echo "<search main>" \| ./llm-runtime` |

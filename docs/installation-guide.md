# Installation Guide

Complete installation guide for llm-runtime across different operating systems and environments.

## Prerequisites

### Required
- **Go 1.21 or later** - For building the tool
- **Git** - For cloning the repository

### Optional (for specific features)
- **Docker** - Required for `<exec>` command execution
- **Ollama** - Required for `<search>` semantic search functionality

## Quick Installation

```bash
git clone https://github.com/computerscienceiscool/llm-runtime.git
cd llm-runtime
make build
./llm-runtime --help
```

## Manual Installation

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

### 3. Install Docker (for exec commands)

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
```bash
echo "<open README.md>" | ./llm-runtime
```

### 2. Test Docker Integration (if installed)
```bash
echo "<exec echo 'Docker works'>" | ./llm-runtime --exec-enabled
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

## Configuration

### Create Configuration File
```bash
# Copy example config or create new
cat > llm-runtime.config.yaml << 'EOF'
repository:
  root: "."
  excluded_paths:
    - ".git"
    - ".env"
    - "*.key"

commands:
  open:
    enabled: true
    max_file_size: 1048576
  write:
    enabled: true
    max_file_size: 102400
  exec:
    enabled: false
  search:
    enabled: true
    vector_db_path: "./embeddings.db"
EOF
```

## System Installation (Optional)

### Install to /usr/local/bin
```bash
sudo cp llm-runtime /usr/local/bin/
sudo chmod +x /usr/local/bin/llm-runtime

# Now available globally
llm-runtime --help
```

### Create Shell Alias
```bash
# Add to ~/.bashrc or ~/.zshrc
echo 'alias llm="./llm-runtime"' >> ~/.bashrc
source ~/.bashrc
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
1. Read the [configuration guide](configuration.md) to customize settings
2. Check [troubleshooting](troubleshooting.md) for common issues
3. See [llm-runtime-overview.md](llm-runtime-overview.md) to understand all features
4. Review [SYSTEM_PROMPT.md](SYSTEM_PROMPT.md) for LLM integration

## Feature Summary

| Feature | Requirement | Test Command |
|---------|-------------|--------------|
| File reading | Go (built-in) | `echo "<open README.md>" \| ./llm-runtime` |
| File writing | Go (built-in) | `echo "<write test.txt>hello</write>" \| ./llm-runtime` |
| Command execution | Docker | `echo "<exec ls>" \| ./llm-runtime --exec-enabled` |
| Semantic search | Ollama | `echo "<search main>" \| ./llm-runtime` |

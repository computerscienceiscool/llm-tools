# Installation Guide

Complete installation guide for the LLM File Access Tool across different operating systems and environments.

## Prerequisites

### Required
- **Go 1.21 or later** - For building the tool
- **Git** - For cloning the repository

### Optional (for specific features)
- **Docker** - Required for `<exec>` command execution
- **Python 3.8+** - Required for `<search>` semantic search functionality

## Quick Installation

### Automated Setup (Recommended)
```bash
git clone https://github.com/computerscienceiscool/llm-runtime.git
cd llm-runtime
./setup.sh
```

The setup script will:
1. Check all prerequisites
2. Build the tool
3. Run tests
4. Optionally install to system PATH
5. Run a quick demo

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
go build -o llm-runtime main.go

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

### 4. Install Python Dependencies (for search)

#### All Platforms
```bash
# Install sentence-transformers
pip install sentence-transformers

# Or using conda
conda install -c conda-forge sentence-transformers

# Verify installation
python3 -c "import sentence_transformers; print('Search dependencies ready')"
```

## Verification

### 1. Test Basic Functionality
```bash
echo "Test file access <open README.md>" | ./llm-runtime
```

### 2. Test Docker Integration (if installed)
```bash
echo "Test exec <exec echo 'Docker works'>" | ./llm-runtime --exec-enabled
```

### 3. Test Search (if Python deps installed)
```bash
# Build search index
./llm-runtime --reindex

# Test search
echo "Test search <search configuration>" | ./llm-runtime
```

### 4. Run Test Suite
```bash
# Unit tests
make test

# Comprehensive tests
make test-suite

# Security tests
./security_test.sh
```

## Configuration

### Create Configuration File
```bash
# Copy example config
cp llm-runtime.config.yaml my-config.yaml

# Edit as needed
nano my-config.yaml
```

### Enable Features
```yaml
commands:
  exec:
    enabled: true  # Requires Docker
  search:
    enabled: true  # Requires Python dependencies
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
# If too old, install newer version
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

### Python Issues
```bash
# Check Python version
python3 --version

# Install pip if missing
sudo apt install python3-pip  # Linux
brew install python3          # macOS

# Install in virtual environment (recommended)
python3 -m venv llm-env
source llm-env/bin/activate
pip install sentence-transformers
```

### Build Issues
```bash
# Clean and rebuild
make clean
go mod tidy
go build -o llm-runtime main.go

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

### Update Dependencies
```bash
# Update Go dependencies
go get -u ./...
go mod tidy

# Update Python dependencies
pip install --upgrade sentence-transformers
```

## Platform-Specific Notes

### Linux
- **WSL**: Works well in WSL2 with Docker Desktop
- **ARM64**: Use appropriate Go and Docker builds
- **Alpine**: May need additional packages for CGO

### macOS
- **Apple Silicon (M1/M2)**: Use arm64 builds
- **Rosetta**: x64 builds work but slower
- **Homebrew**: Preferred for dependencies

### Windows
- **WSL2**: Recommended environment
- **PowerShell**: Should work but less tested
- **File paths**: Use forward slashes in config

## Security Considerations

### During Installation
- **Verify downloads** from official sources only
- **Check checksums** for Go and Docker installers
- **Review scripts** before running (including setup.sh)

### Post-Installation
- **Limit repository access** to trusted projects only
- **Configure excluded paths** for sensitive files
- **Review Docker security** settings
- **Enable audit logging** in production

## Next Steps

After successful installation:
1. **Read the [configuration guide](configuration.md)** to customize settings
2. **Check [troubleshooting](troubleshooting.md)** for common issues
3. **See [llm-runtime-overview.md](llm-runtime-overview.md)** to understand all features
4. **Review [SYSTEM_PROMPT.md](SYSTEM_PROMPT.md)** for LLM integration

## Tips

- **Use setup.sh** for automated installation when possible
- **Enable Docker** for full functionality (exec commands)
- **Install Python deps** for semantic search capabilities
- **Run test suite** to verify everything works
- **Start with basic usage** before enabling advanced features

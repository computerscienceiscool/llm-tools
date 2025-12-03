# Presentation Prep Checklist

Complete these steps **before** the presentation to ensure smooth demos.

## 1. Environment Setup (Day Before)

### Docker
```bash
# Verify Docker is running
docker version

# Pull the Ubuntu image (avoids download delay during demo)
docker pull ubuntu:22.04

# Test Docker works with the tool
echo "<exec echo 'Docker ready'>" | ./llm-runtime --exec-enabled
```

### Python/Search (Optional - skip if not demoing search)
```bash
# Check Python
python3 --version

# Install sentence-transformers if needed
pip install sentence-transformers

# Verify
./llm-runtime --check-python-setup

# Build the search index (can take a few minutes)
./llm-runtime --reindex
```

### Build the Tool
```bash
cd /path/to/llm-runtime
make clean
make build

# Verify it runs
./llm-runtime --help
```

## 2. Clean Up Demo Environment (30 min before)

```bash
# Remove any leftover test files from previous runs
rm -f test-file.txt hello.go notes.md *.bak.*

# Clear audit log for clean demo
rm -f audit.log

# Verify clean state
ls -la
```

## 3. Terminal Setup

- [ ] Increase terminal font size (Cmd/Ctrl + to zoom)
- [ ] Use a light background if projecting (better visibility)
- [ ] Close unnecessary tabs/applications
- [ ] Disable notifications (Do Not Disturb mode)
- [ ] Have two terminal windows ready:
  - Window 1: Running commands
  - Window 2: Viewing file contents (`cat`, `ls`)

## 4. Browser Setup (for LLM demo)

- [ ] Open Claude.ai or ChatGPT in a browser tab
- [ ] Start a fresh conversation
- [ ] Have the system prompt ready to paste:
  - Located at: `docs/SYSTEM_PROMPT.md`
  - Or use the short version below

### Short System Prompt for Demo
```
You can explore this repository using these commands in your responses:
- <open filepath> - Read a file
- <write filepath>content</write> - Create/modify a file
- <exec command> - Run a command (Docker isolated)
- <search query> - Semantic search

I will execute these commands and paste the results back to you.
Start by reading README.md to understand the project.
```

## 5. Pre-Run These Commands (Avoid First-Run Delays)

```bash
# "Warm up" Docker by running a quick command
echo "<exec echo 'warm up'>" | ./llm-runtime --exec-enabled

# This caches the container startup, making demos faster
```

## 6. Backup Plan

If something breaks during the demo:

### Docker won't start
- Skip exec demos
- Say: "Docker seems to be having issues, so I'll skip the command execution demo, but let me show you the file operations which work without Docker."

### Search not working
- Skip search demos
- Say: "The search feature requires some Python setup I'll skip today, but the core file operations work great."

### General fallback
- Have screenshots of expected output ready
- Or pre-record a backup video

## 7. Quick Test Script (Run 5 min before)

```bash
#!/bin/bash
echo "=== Testing llm-runtime ==="

echo "1. Testing open..."
echo "<open README.md>" | ./llm-runtime | head -20
echo ""

echo "2. Testing write..."
echo "<write demo-test.txt>test</write>" | ./llm-runtime
rm -f demo-test.txt
echo ""

echo "3. Testing exec..."
echo "<exec echo 'Hello'>" | ./llm-runtime --exec-enabled
echo ""

echo "=== All tests passed ==="
```

Save as `pre-demo-test.sh`, run with `bash pre-demo-test.sh`

## 8. Materials to Have Open

- [ ] This prep checklist
- [ ] The presenter script (`presenter-script.md`)
- [ ] GitHub repo page (to show at end)
- [ ] Terminal in the llm-runtime directory

## 9. Room/Tech Setup

- [ ] Test projector/screen share works
- [ ] Verify audience can see terminal text clearly
- [ ] Have water nearby
- [ ] Know where the bathroom is (before 45 min presentation!)

---

## Quick Reference: Demo Commands

```bash
# Open command
echo "Show me the README: <open README.md>" | ./llm-runtime

# Write command
echo '<write test-file.txt>
Hello from llm-runtime!
</write>' | ./llm-runtime

# Exec command
echo "<exec go test ./...>" | ./llm-runtime --exec-enabled

# Multiple commands
echo "<open go.mod> <exec go version>" | ./llm-runtime --exec-enabled

# Interactive mode
./llm-runtime --interactive --exec-enabled
```

---

**You're ready!** Take a breath, you built this thing, you know it better than anyone.

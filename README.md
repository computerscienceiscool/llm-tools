# LLM File Access Tool

A secure tool that enables Large Language Models to autonomously explore and work with local repositories through embedded commands in their responses.

## Features

- **Secure Path Validation**: Prevents directory traversal and access outside repository boundaries
- **Command Parsing**: Extracts and executes `<open filename>` commands from LLM output
- **Audit Logging**: Tracks all operations with timestamps and results
- **Multiple Modes**: Supports pipe, interactive, and file-based operation
- **Configurable Security**: Exclude sensitive paths, set file size limits

## Installation

```bash
go build -o llm-tool main.go
```

Or install directly:
```bash
go install github.com/llm-tool@latest
```

## Usage

### Basic Usage (Pipe Mode)

```bash
echo "Let me check the main file <open main.go>" | ./llm-tool
```

### Interactive Mode

```bash
./llm-tool --interactive
```

In interactive mode, the tool continuously processes input and executes commands as they appear.

### File Mode

```bash
./llm-tool --input llm_output.txt --output results.txt
```

### With Custom Repository Root

```bash
./llm-tool --root /path/to/repository --input query.txt
```

## Command Line Options

- `--root PATH`: Repository root directory (default: current directory)
- `--max-size BYTES`: Maximum file size in bytes (default: 1048576 = 1MB)
- `--interactive`: Run in interactive mode
- `--input FILE`: Read from file instead of stdin
- `--output FILE`: Write to file instead of stdout
- `--exclude PATTERNS`: Comma-separated list of excluded paths (default: ".git,.env,*.key,*.pem")
- `--verbose`: Enable verbose output
- `--json`: Output results in JSON format (not yet implemented)

## Security Features

### Path Validation
- Canonicalizes all paths using OS-native functions
- Resolves symlinks and verifies final destination
- Prevents directory traversal attempts (../)
- Ensures all accessed files are within repository bounds

### Excluded Paths
By default, the following are excluded:
- `.git` directory
- `.env` files
- `*.key` files
- `*.pem` files

### Audit Logging
All operations are logged to `audit.log` with:
- ISO 8601 timestamp
- Session ID
- Command type
- File path
- Success/failure status
- Error messages (if any)

Example audit log entry:
```
2025-11-22T10:30:45Z|session:1234567890|open|src/main.go|success|
2025-11-22T10:30:46Z|session:1234567890|open|../../etc/passwd|failed|path traversal detected
```

## Example LLM Integration

### System Prompt for LLM
```
You have access to a file system tool that allows you to explore the repository. 
You can use the <open filepath> command in your responses to read files.

The tool will execute these commands and provide the file contents. You can use 
multiple commands to explore the codebase thoroughly. All paths are relative to 
the repository root.

Example:
"Let me examine the configuration: <open config.yaml>
Based on that, I'll check the main entry point: <open cmd/main.go>"
```

### Example Session

**User**: "Help me understand this Go project's structure"

**LLM**: "I'll explore the project to understand its structure. Let me start with:

<open go.mod>

Now let me check the main entry point:

<open cmd/main.go>

Let me also look at the README for documentation:

<open README.md>"

**Tool Output**:
```
=== LLM TOOL START ===
I'll explore the project to understand its structure. Let me start with:

<open go.mod>
=== COMMAND: <open go.mod> ===
=== FILE: go.mod ===
module github.com/example/project

go 1.21

require (
    github.com/gin-gonic/gin v1.9.0
)
=== END FILE ===
=== END COMMAND ===

Now let me check the main entry point:

<open cmd/main.go>
=== COMMAND: <open cmd/main.go> ===
=== FILE: cmd/main.go ===
package main

import (
    "github.com/gin-gonic/gin"
)

func main() {
    r := gin.Default()
    r.GET("/", handleHome)
    r.Run(":8080")
}
=== END FILE ===
=== END COMMAND ===

[continues...]
=== LLM TOOL COMPLETE ===
Commands executed: 3
Time elapsed: 0.15s
=== END ===
```

## Testing

Run the test suite:
```bash
go test -v
```

Run with race detection:
```bash
go test -race
```

Run benchmarks:
```bash
go test -bench=.
```

## Architecture

```
┌─────────────┐     ┌─────────────┐     ┌──────────────┐
│  LLM Output │────▶│   Parser    │────▶│   Validator  │
└─────────────┘     └─────────────┘     └──────────────┘
                                               │
                                               ▼
┌─────────────┐     ┌─────────────┐     ┌──────────────┐
│   Results   │◀────│   Executor  │◀────│ File System  │
└─────────────┘     └─────────────┘     └──────────────┘
                           │
                           ▼
                    ┌─────────────┐
                    │ Audit Logger│
                    └─────────────┘
```

## Performance

- Command parsing: <1ms for typical input
- Path validation: <1ms per path
- File read (1MB): <10ms on SSD
- Total overhead: ~2-5ms per command

## Future Enhancements

The tool is designed to be extended with additional commands:

- `<search query>`: Search for files using vector similarity
- `<write filepath>`: Create or update files
- `<exec command>`: Execute commands in sandboxed environment
- `<diff file1 file2>`: Compare files

## Security Considerations

1. **Never run as root**: The tool should run with minimal privileges
2. **Restrict repository access**: Only expose repositories you trust the LLM to read
3. **Monitor audit logs**: Regularly review audit.log for suspicious patterns
4. **File size limits**: Adjust --max-size based on your security requirements
5. **Network isolation**: When adding exec command, ensure container has no network

## Contributing

1. Ensure all tests pass
2. Add tests for new features
3. Update documentation
4. Follow Go best practices

## License

MIT License - See LICENSE file for details
# llm-tools

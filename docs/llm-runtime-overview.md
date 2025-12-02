# LLM File Access Tool - Overview

## What is the LLM File Access Tool?

The LLM File Access Tool is a secure command-line utility that enables Large Language Models (LLMs) to autonomously explore, read, modify, and analyze local code repositories. Instead of requiring all files to be loaded into the LLM's context upfront, this tool allows LLMs to dynamically discover and interact with files as needed through embedded commands in their responses.

## How It Works

The tool works by:
1. **Parsing LLM Output**: Scans text from LLMs for special XML-like commands
2. **Executing Commands Safely**: Runs the requested operations with built-in security controls
3. **Returning Results**: Provides formatted output that the LLM can use to understand the repository

## Core Features

### **File Reading (`<open>` command)**
Allows LLMs to read any file within the repository boundaries with automatic security validation.
- **Use Case**: Understanding project structure, reading configuration files, examining source code
- **Security**: Path traversal protection, size limits, excluded file patterns
- **Example**: `<open src/main.go>` reads the main.go file

### **File Writing (`<write>` command)**
Enables LLMs to create new files or modify existing ones with automatic backups and validation.
- **Use Case**: Creating documentation, updating configuration, refactoring code
- **Security**: Extension whitelisting, backup creation, atomic writes
- **Example**: `<write README.md>New content here</write>` creates or updates README

### **Command Execution (`<exec>` command)**
Runs commands in secure, isolated Docker containers with no network access.
- **Use Case**: Running tests, building projects, validating changes
- **Security**: Command whitelisting, Docker isolation, resource limits
- **Example**: `<exec go test>` runs tests in an isolated container

### **Semantic Search (`<search>` command)**
Performs AI-powered semantic search across all indexed files in the repository.
- **Use Case**: Finding relevant code, locating similar functions, discovering patterns
- **Technology**: Uses sentence-transformers for semantic understanding
- **Example**: `<search authentication logic>` finds auth-related code

## Typical Workflow

1. **LLM receives a task**: "Help me understand this Go project and add error handling"

2. **LLM explores systematically**:
   - `<open README.md>` - Understand the project
   - `<open go.mod>` - Check dependencies
   - `<search error handling>` - Find existing error patterns
   - `<open src/main.go>` - Examine entry point

3. **LLM analyzes and modifies**:
   - `<exec go test>` - Run existing tests
   - `<write src/errors.go>...new error handling code...</write>` - Add improvements
   - `<exec go test>` - Verify changes work

4. **LLM provides summary**: Explains what was discovered and changed

## Security Features

- **Path Validation**: Prevents access outside repository boundaries
- **Docker Isolation**: Commands run in containerized environments
- **Audit Logging**: All operations are logged with timestamps
- **Resource Limits**: File size and execution time constraints
- **Command Whitelisting**: Only approved commands can be executed
- **Backup Creation**: Automatic backups before file modifications

## Installation & Basic Usage

```bash
# Build the tool
make build

# Basic usage (pipe mode)
echo "Check the main file <open main.go>" | ./llm-runtime

# Interactive mode
./llm-runtime --interactive

# With exec commands enabled
echo "Run tests <exec go test>" | ./llm-runtime --exec-enabled
```

## Configuration

The tool is configured via `llm-runtime.config.yaml`:

```yaml
commands:
  exec:
    enabled: true
    whitelist: ["go test", "npm build", "python -m pytest"]
  search:
    enabled: true
    vector_db_path: "./embeddings.db"
security:
  excluded_paths: [".git", ".env", "*.key"]
```

## Use Cases

### **Code Review & Analysis**
LLMs can systematically explore codebases, understand architecture, and provide insights without needing all files loaded upfront.

### **Documentation Generation** 
Automatically create or update documentation by reading source code and understanding project structure.

### **Automated Testing & Validation**
Run tests, check builds, and verify that changes work correctly before suggesting them.

### **Code Refactoring**
Safely modify multiple files while testing changes incrementally to ensure nothing breaks.

### **Bug Investigation**
Search for similar patterns, examine relevant files, and test fixes in isolation.

## Next Steps

- **[File Reading Guide](file-reading-guide.md)** - Learn how to read and explore files
- **[File Writing Guide](file-writing-guide.md)** - Understand file creation and modification
- **[Command Execution Guide](command-execution-guide.md)** - Execute commands safely
- **[Semantic Search Guide](semantic-search-guide.md)** - Search code semantically

The LLM File Access Tool transforms how LLMs work with codebases, making them capable of autonomous exploration and modification while maintaining security and auditability.

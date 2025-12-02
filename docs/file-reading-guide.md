# File Reading Guide - `<open>` Command

## Overview

The `<open>` command allows LLMs to read the contents of any file within the repository. This is the foundation feature that enables LLMs to explore and understand codebases dynamically, rather than having all files loaded into context at once.

## How It Works

When an LLM includes `<open filename>` in its response, the tool:
1. **Validates the path** - Ensures the file is within repository bounds and not excluded
2. **Checks file size** - Verifies the file isn't too large (default: 1MB limit)
3. **Reads the content** - Loads the file contents securely
4. **Returns formatted output** - Provides the content in a clearly marked section

## Basic Syntax

```
<open filepath>
```

**Examples:**
- `<open README.md>` - Read the project README
- `<open src/main.go>` - Read a Go source file
- `<open config/settings.json>` - Read a JSON configuration file
- `<open docs/api/endpoints.md>` - Read nested documentation

## Use Cases

### **Project Exploration**
```
Let me start by understanding this project structure.

<open README.md>

Now let me check the main entry point:

<open src/main.go>

And examine the configuration:

<open package.json>
```

### **Code Analysis**
```
I'll analyze the authentication system step by step.

First, let me look at the main auth file:

<open src/auth/middleware.go>

Now let me check how it's used:

<open src/routes/api.go>

And examine the tests:

<open src/auth/middleware_test.go>
```

### **Following Dependencies**
```
Let me trace through this import chain.

<open go.mod>

I see it imports custom packages. Let me check them:

<open internal/database/connection.go>

<open internal/handlers/user.go>
```

### **Configuration Review**
```
Let me examine all the configuration files to understand the setup.

<open .env.example>

<open config/database.yaml>

<open docker-compose.yml>
```

## Security Features

### **Path Validation**
- **Prevents directory traversal**: `<open ../../etc/passwd>` is blocked
- **Repository boundaries**: Can only access files within the project
- **Symlink resolution**: Follows symlinks but validates final destination

### **Excluded Files**
By default, these patterns are blocked for security:
- `.git/` - Git repository internals
- `.env` - Environment variables with secrets
- `*.key` - Private key files
- `*.pem` - Certificate files
- `secrets/` - Any secrets directories

### **File Size Limits**
- **Default limit**: 1MB per file
- **Configurable**: Can be adjusted via `--max-size` flag
- **Large file handling**: Shows clear error messages for oversized files

## Output Format

When successful, the tool outputs:
```
=== COMMAND: <open filename> ===
=== FILE: filename ===
[file contents here]
=== END FILE ===
=== END COMMAND ===
```

When it fails:
```
=== ERROR: FILE_NOT_FOUND ===
Message: File not found: nonexistent.txt
Command: <open nonexistent.txt>
=== END ERROR ===
```

## Common Error Types

### **FILE_NOT_FOUND**
```
<open missing.txt>
```
**Cause**: The file doesn't exist
**Solution**: Check the file path and spelling

### **PATH_SECURITY**
```
<open ../../../etc/passwd>
```
**Cause**: Attempting to access files outside the repository
**Solution**: Use paths relative to the repository root

### **RESOURCE_LIMIT**
```
<open huge-dataset.csv>
```
**Cause**: File is larger than the size limit
**Solution**: Process file in chunks or increase the limit

### **PERMISSION_DENIED**
```
<open .git/config>
```
**Cause**: File is in the excluded paths list
**Solution**: Remove from excluded paths if access is needed

## Configuration Options

### Command Line Flags
```bash
# Set custom file size limit (2MB)
./llm-runtime --max-size 2097152

# Custom repository root
./llm-runtime --root /path/to/project

# Custom excluded paths
./llm-runtime --exclude ".git,*.secret,private/"

# Verbose output for debugging
./llm-runtime --verbose
```

### Configuration File (llm-runtime.config.yaml)
```yaml
commands:
  open:
    enabled: true
    max_file_size: 1048576  # 1MB
    allowed_extensions:
      - ".go"
      - ".py" 
      - ".js"
      - ".md"
      - ".txt"
      - ".json"
      - ".yaml"

repository:
  excluded_paths:
    - ".git"
    - ".env"
    - "*.key"
    - "*.pem"
    - "secrets"
```

## Best Practices for LLMs

### **Start with Overview Files**
```
Let me start by understanding this project.

<open README.md>
<open package.json>
<open go.mod>
```

### **Follow Logical Flow**
```
Now let me trace through the application flow:

<open cmd/main.go>
<open internal/server/server.go>
<open internal/handlers/api.go>
```

### **Check Related Files**
```
I see this imports a custom package. Let me examine it:

<open internal/auth/jwt.go>
<open internal/auth/jwt_test.go>
```

### **Multiple Commands in Context**
```
I'll examine the entire authentication system:

<open src/auth/middleware.go>
<open src/auth/tokens.go>
<open src/auth/validators.go>
<open tests/auth_test.go>

Based on these files, I can see the authentication uses JWT tokens...
```

## Advanced Usage

### **Conditional Reading**
```
Let me check if this is a Node.js or Go project:

<open package.json>

Since this is a Node.js project, let me check the main entry:

<open src/index.js>
```

### **Error Handling**
```
Let me try to find the configuration file:

<open config.json>

If that doesn't exist, let me try:

<open config.yaml>
```

### **Pattern Discovery**
```
Let me examine a few source files to understand the patterns:

<open src/users.js>
<open src/posts.js>
<open src/comments.js>

I can see this follows a consistent controller pattern...
```

## Performance Tips

- **Read only what you need**: Don't open large files unless necessary
- **Use search first**: For large codebases, use `<search>` to find relevant files
- **Check file sizes**: Be aware of the 1MB default limit
- **Cache mentally**: Remember file contents to avoid re-reading

## Troubleshooting

### **File Not Found**
1. Verify the file exists: `ls -la filename`
2. Check if you're in the right repository root
3. Ensure the path is relative to repository root

### **Permission Denied**
1. Check if the file is in excluded paths
2. Verify file permissions: `ls -la filename`
3. Consider if the file should be accessible

### **File Too Large**
1. Check file size: `ls -lh filename`
2. Use `head` or `tail` commands via `<exec>` for large files
3. Increase size limit if appropriate

The `<open>` command is the foundation of LLM repository exploration, enabling dynamic and secure file access that scales to any codebase size.

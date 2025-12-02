# File Writing Guide - `<write>` Command

## Overview

The `<write>` command enables LLMs to create new files or modify existing files within the repository. This powerful feature allows LLMs to implement changes, create documentation, and modify codebases while maintaining safety through automatic backups and validation.

## How It Works

When an LLM includes a `<write>` command in its response, the tool:
1. **Validates the file path** - Ensures it's within repository bounds and allowed
2. **Checks content size** - Verifies content isn't too large (default: 100KB limit)  
3. **Creates backup** - Saves existing file with timestamp (if it exists)
4. **Validates extension** - Ensures file type is allowed (if configured)
5. **Writes atomically** - Uses temporary file then renames for safety
6. **Formats content** - Auto-formats supported file types (Go, JSON, etc.)

## Basic Syntax

```
<write filepath>
content goes here
</write>
```

**Note**: All content between the opening and closing tags becomes the file content.

## Examples

### **Creating a New File**
```
I'll create a configuration file for this project:

<write config.yaml>
# Project Configuration
name: my-project
version: 1.0.0

database:
  host: localhost
  port: 5432
  name: myapp

server:
  port: 8080
  host: 0.0.0.0
</write>
```

### **Updating an Existing File**
```
Let me enhance the README with more details:

<write README.md>
# My Project

This project provides a REST API for managing tasks.

## Features
- User authentication
- Task CRUD operations
- Real-time notifications

## Quick Start
1. Install dependencies: `npm install`
2. Start server: `npm start`
3. Visit: http://localhost:8080
</write>
```

### **Creating Source Code**
```
I'll add a new utility function:

<write src/utils/validator.js>
/**
 * Input validation utilities
 */

function validateEmail(email) {
    const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
    return emailRegex.test(email);
}

function validatePassword(password) {
    return password.length >= 8 && 
           /[A-Z]/.test(password) && 
           /[a-z]/.test(password) && 
           /\d/.test(password);
}

module.exports = {
    validateEmail,
    validatePassword
};
</write>
```

## Use Cases

### **Creating New Features**
```
I'll implement a new authentication middleware:

<write src/middleware/auth.js>
const jwt = require('jsonwebtoken');

function authenticateToken(req, res, next) {
    const authHeader = req.headers['authorization'];
    const token = authHeader && authHeader.split(' ')[1];

    if (!token) {
        return res.sendStatus(401);
    }

    jwt.verify(token, process.env.ACCESS_TOKEN_SECRET, (err, user) => {
        if (err) return res.sendStatus(403);
        req.user = user;
        next();
    });
}

module.exports = { authenticateToken };
</write>

Now let me add tests for this middleware:

<write tests/auth.test.js>
const { authenticateToken } = require('../src/middleware/auth');
const jwt = require('jsonwebtoken');

describe('Authentication Middleware', () => {
    test('should reject requests without token', () => {
        const req = { headers: {} };
        const res = { sendStatus: jest.fn() };
        
        authenticateToken(req, res, () => {});
        
        expect(res.sendStatus).toHaveBeenCalledWith(401);
    });
});
</write>
```

### **Documentation Creation**
```
I'll create comprehensive API documentation:

<write docs/api.md>
# API Documentation

## Authentication Endpoints

### POST /auth/login
Authenticate user and return JWT token.

**Request:**
```json
{
    "email": "user@example.com",
    "password": "password123"
}
```

**Response:**
```json
{
    "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "user": {
        "id": 1,
        "email": "user@example.com"
    }
}
```

### POST /auth/register
Create new user account.

**Request:**
```json
{
    "email": "user@example.com", 
    "password": "password123",
    "name": "John Doe"
}
```
</write>
```

### **Configuration Updates**
```
I'll update the package.json with new dependencies and scripts:

<write package.json>
{
  "name": "task-manager",
  "version": "1.2.0",
  "description": "A simple task management API",
  "main": "src/index.js",
  "scripts": {
    "start": "node src/index.js",
    "dev": "nodemon src/index.js", 
    "test": "jest",
    "test:watch": "jest --watch",
    "lint": "eslint src/",
    "lint:fix": "eslint src/ --fix"
  },
  "dependencies": {
    "express": "^4.18.0",
    "jsonwebtoken": "^8.5.1",
    "bcryptjs": "^2.4.3",
    "mongoose": "^6.3.0"
  },
  "devDependencies": {
    "jest": "^28.0.0",
    "nodemon": "^2.0.0",
    "eslint": "^8.15.0"
  }
}
</write>
```

### **Refactoring Code**
```
Let me read the current implementation first:

<open src/user-service.js>

Now I'll refactor it to improve error handling:

<write src/user-service.js>
const User = require('./models/User');

class UserService {
    async createUser(userData) {
        try {
            const existingUser = await User.findOne({ email: userData.email });
            if (existingUser) {
                throw new Error('User with this email already exists');
            }
            
            const user = new User(userData);
            await user.save();
            return user;
        } catch (error) {
            console.error('Error creating user:', error);
            throw error;
        }
    }
    
    async findUserById(id) {
        try {
            const user = await User.findById(id);
            if (!user) {
                throw new Error('User not found');
            }
            return user;
        } catch (error) {
            console.error('Error finding user:', error);
            throw error;
        }
    }
}

module.exports = UserService;
</write>
```

## Security Features

### **Path Validation**
- **Repository boundaries**: Can only write within the project directory
- **Traversal prevention**: `<write ../../etc/passwd>content</write>` is blocked
- **Symlink safety**: Validates final destination of symlinks

### **Extension Whitelisting**
Configure allowed file extensions:
```yaml
commands:
  write:
    allowed_extensions:
      - ".js"
      - ".md" 
      - ".json"
      - ".yaml"
```

Attempts to write `.exe`, `.bat`, or other dangerous extensions are blocked.

### **Automatic Backups**
- **Timestamp backups**: `file.js.bak.1640995200`
- **Before every write**: Existing files are backed up automatically
- **Configurable**: Can be disabled with `backup: false` in config

### **Atomic Writes**
- **Temporary files**: Content written to `.tmp` file first
- **Atomic rename**: Only moved to final location if write succeeds
- **Rollback safety**: Failed writes don't corrupt existing files

## File Size Limits

### **Default Limits**
- **Write limit**: 100KB per file (configurable)
- **Prevents abuse**: Stops accidental large file creation

### **Configuration**
```bash
# Increase write size limit to 1MB
./llm-runtime --max-write-size 1048576
```

```yaml
commands:
  write:
    max_file_size: 102400  # 100KB
```

## Content Formatting

The tool automatically formats certain file types:

### **Go Files (.go)**
```
<write main.go>
package main
import "fmt"
func main(){fmt.Println("Hello")}
</write>
```

**Becomes:**
```go
package main

import "fmt"

func main() {
	fmt.Println("Hello")
}
```

### **JSON Files (.json)**
```
<write config.json>
{"name":"app","version":"1.0","settings":{"debug":true}}
</write>
```

**Becomes:**
```json
{
  "name": "app",
  "version": "1.0",
  "settings": {
    "debug": true
  }
}
```

## Output Format

### **Successful Write (New File)**
```
=== WRITE SUCCESSFUL: config.yaml ===
Action: CREATED
Bytes written: 245
=== END WRITE ===
```

### **Successful Write (Updated File)**
```
=== WRITE SUCCESSFUL: README.md ===
Action: UPDATED
Bytes written: 1205
Backup: README.md.bak.1640995200
=== END WRITE ===
```

### **Error Examples**
```
=== ERROR: EXTENSION_DENIED ===
Message: File extension not allowed: .exe
Command: <write malicious.exe>content</write>
=== END ERROR ===
```

## Common Error Types

### **PATH_SECURITY**
```
<write ../../../etc/evil.txt>content</write>
```
**Cause**: Attempting to write outside repository
**Solution**: Use paths relative to repository root

### **EXTENSION_DENIED**
```
<write script.exe>content</write>
```
**Cause**: File extension not in allowed list  
**Solution**: Check allowed extensions or modify configuration

### **RESOURCE_LIMIT**
```
<write huge.txt>
[100KB+ of content]
</write>
```
**Cause**: Content exceeds size limit
**Solution**: Split content or increase limit

### **DIRECTORY_CREATION_FAILED**
```
<write /readonly/newfile.txt>content</write>
```
**Cause**: Cannot create parent directory
**Solution**: Ensure write permissions and valid path

## Configuration Options

### **Command Line Flags**
```bash
# Custom write size limit
./llm-runtime --max-write-size 1048576

# Disable backups  
./llm-runtime --backup=false

# Allow all extensions
./llm-runtime --allowed-extensions=""

# Force writes even with conflicts
./llm-runtime --force
```

### **Configuration File**
```yaml
commands:
  write:
    enabled: true
    max_file_size: 102400
    backup_before_write: true
    allowed_extensions:
      - ".js"
      - ".py"
      - ".go" 
      - ".md"
      - ".json"
      - ".yaml"
      - ".yml"
      - ".txt"
      - ".html"
      - ".css"
    format_code: true
    atomic_writes: true
```

## Best Practices for LLMs

### **Always Read Before Writing**
```
Let me first check the current configuration:

<open package.json>

Now I'll update it with the new dependencies:

<write package.json>
[updated content]
</write>
```

### **Create Related Files Together**
```
I'll create a new feature with its test file:

<write src/features/notifications.js>
[implementation]
</write>

<write tests/notifications.test.js>
[tests]
</write>
```

### **Explain Your Changes**
```
I'll add proper error handling to the user service:

<write src/services/user.js>
[improved code with error handling]
</write>

The changes I made:
1. Added try-catch blocks around database operations
2. Added specific error messages for different failure cases  
3. Added logging for debugging purposes
```

### **Use Appropriate File Sizes**
- Keep files focused and reasonably sized
- Split large implementations into multiple files
- Use the 100KB limit as a guide for file organization

## Advanced Features

### **Backup Management**
```bash
# List backup files
find . -name "*.bak.*" | head -10

# Restore from backup
cp file.js.bak.1640995200 file.js
```

### **Batch Operations**
```
I'll create several related configuration files:

<write .env.example>
DATABASE_URL=postgres://localhost/myapp
JWT_SECRET=your-secret-key
PORT=3000
</write>

<write .gitignore>
node_modules/
.env
*.log
dist/
</write>

<write .dockerignore>
node_modules/
.git/
*.md
</write>
```

### **Template Generation**
```
I'll create a template for new API routes:

<write templates/route-template.js>
const express = require('express');
const router = express.Router();

// GET /api/RESOURCE
router.get('/', async (req, res) => {
    try {
        // Implementation here
        res.json({ message: 'Success' });
    } catch (error) {
        res.status(500).json({ error: error.message });
    }
});

// POST /api/RESOURCE  
router.post('/', async (req, res) => {
    try {
        // Implementation here
        res.status(201).json({ message: 'Created' });
    } catch (error) {
        res.status(500).json({ error: error.message });
    }
});

module.exports = router;
</write>
```

## Troubleshooting

### **Write Fails Silently**
1. Check file extension against allowed list
2. Verify path is within repository
3. Check file size limits
4. Review audit log for errors

### **Content Not Formatted**
1. Ensure file has correct extension
2. Check if formatting is enabled in config
3. Verify content is valid for the file type

### **Backup Files Accumulating**
1. Clean up old backups periodically
2. Consider disabling backups for temporary work
3. Use version control for important changes

The `<write>` command empowers LLMs to make meaningful changes to codebases while maintaining safety and auditability through comprehensive validation and backup systems.

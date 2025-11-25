#!/bin/bash

# Example: LLM exploring a Go web application
# This demonstrates how an LLM would use the tool to understand a codebase

echo "=== Example: LLM Analyzing a Go Web Application ==="
echo
echo "Creating a sample Go web application..."

# Create a sample Go web app structure
SAMPLE_APP="/tmp/sample-go-webapp"
rm -rf "$SAMPLE_APP"
mkdir -p "$SAMPLE_APP/cmd/server"
mkdir -p "$SAMPLE_APP/internal/handlers"
mkdir -p "$SAMPLE_APP/internal/models"
mkdir -p "$SAMPLE_APP/internal/database"
mkdir -p "$SAMPLE_APP/pkg/middleware"
mkdir -p "$SAMPLE_APP/configs"

# Create go.mod
cat > "$SAMPLE_APP/go.mod" << 'EOF'
module github.com/example/webapp

go 1.21

require (
    github.com/gorilla/mux v1.8.0
    github.com/joho/godotenv v1.5.1
    gorm.io/driver/postgres v1.5.2
    gorm.io/gorm v1.25.4
)
EOF

# Create README
cat > "$SAMPLE_APP/README.md" << 'EOF'
# Sample Web Application

A REST API built with Go for managing a simple task list.

## Features
- RESTful API endpoints
- PostgreSQL database
- JWT authentication
- Rate limiting middleware

## Getting Started
1. Install dependencies: `go mod download`
2. Set up environment variables
3. Run: `go run cmd/server/main.go`

## API Endpoints
- GET /api/tasks - List all tasks
- POST /api/tasks - Create a new task
- GET /api/tasks/{id} - Get task by ID
- PUT /api/tasks/{id} - Update task
- DELETE /api/tasks/{id} - Delete task
EOF

# Create main.go
cat > "$SAMPLE_APP/cmd/server/main.go" << 'EOF'
package main

import (
    "log"
    "net/http"
    "os"
    
    "github.com/example/webapp/internal/database"
    "github.com/example/webapp/internal/handlers"
    "github.com/example/webapp/pkg/middleware"
    "github.com/gorilla/mux"
    "github.com/joho/godotenv"
)

func main() {
    // Load environment variables
    if err := godotenv.Load(); err != nil {
        log.Println("No .env file found")
    }
    
    // Initialize database
    db := database.InitDB()
    
    // Create handler with db
    h := handlers.NewTaskHandler(db)
    
    // Setup routes
    r := mux.NewRouter()
    
    // Apply middleware
    r.Use(middleware.LoggingMiddleware)
    r.Use(middleware.RateLimitMiddleware)
    
    // Task routes
    api := r.PathPrefix("/api").Subrouter()
    api.HandleFunc("/tasks", h.ListTasks).Methods("GET")
    api.HandleFunc("/tasks", h.CreateTask).Methods("POST")
    api.HandleFunc("/tasks/{id}", h.GetTask).Methods("GET")
    api.HandleFunc("/tasks/{id}", h.UpdateTask).Methods("PUT")
    api.HandleFunc("/tasks/{id}", h.DeleteTask).Methods("DELETE")
    
    port := os.Getenv("PORT")
    if port == "" {
        port = "8080"
    }
    
    log.Printf("Server starting on port %s", port)
    log.Fatal(http.ListenAndServe(":"+port, r))
}
EOF

# Create task handler
cat > "$SAMPLE_APP/internal/handlers/tasks.go" << 'EOF'
package handlers

import (
    "encoding/json"
    "net/http"
    
    "github.com/example/webapp/internal/models"
    "github.com/gorilla/mux"
    "gorm.io/gorm"
)

type TaskHandler struct {
    db *gorm.DB
}

func NewTaskHandler(db *gorm.DB) *TaskHandler {
    return &TaskHandler{db: db}
}

func (h *TaskHandler) ListTasks(w http.ResponseWriter, r *http.Request) {
    var tasks []models.Task
    h.db.Find(&tasks)
    
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(tasks)
}

func (h *TaskHandler) CreateTask(w http.ResponseWriter, r *http.Request) {
    var task models.Task
    if err := json.NewDecoder(r.Body).Decode(&task); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }
    
    h.db.Create(&task)
    
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusCreated)
    json.NewEncoder(w).Encode(task)
}

func (h *TaskHandler) GetTask(w http.ResponseWriter, r *http.Request) {
    vars := mux.Vars(r)
    id := vars["id"]
    
    var task models.Task
    if err := h.db.First(&task, id).Error; err != nil {
        http.Error(w, "Task not found", http.StatusNotFound)
        return
    }
    
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(task)
}

func (h *TaskHandler) UpdateTask(w http.ResponseWriter, r *http.Request) {
    // Implementation here
}

func (h *TaskHandler) DeleteTask(w http.ResponseWriter, r *http.Request) {
    // Implementation here
}
EOF

# Create task model
cat > "$SAMPLE_APP/internal/models/task.go" << 'EOF'
package models

import (
    "time"
    "gorm.io/gorm"
)

type Task struct {
    ID          uint           `json:"id" gorm:"primaryKey"`
    Title       string         `json:"title" gorm:"not null"`
    Description string         `json:"description"`
    Completed   bool           `json:"completed" gorm:"default:false"`
    CreatedAt   time.Time      `json:"created_at"`
    UpdatedAt   time.Time      `json:"updated_at"`
    DeletedAt   gorm.DeletedAt `json:"-" gorm:"index"`
}
EOF

# Create database connection
cat > "$SAMPLE_APP/internal/database/connection.go" << 'EOF'
package database

import (
    "fmt"
    "log"
    "os"
    
    "github.com/example/webapp/internal/models"
    "gorm.io/driver/postgres"
    "gorm.io/gorm"
)

func InitDB() *gorm.DB {
    dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable",
        os.Getenv("DB_HOST"),
        os.Getenv("DB_USER"),
        os.Getenv("DB_PASSWORD"),
        os.Getenv("DB_NAME"),
        os.Getenv("DB_PORT"),
    )
    
    db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
    if err != nil {
        log.Fatal("Failed to connect to database:", err)
    }
    
    // Auto migrate models
    db.AutoMigrate(&models.Task{})
    
    return db
}
EOF

# Create middleware
cat > "$SAMPLE_APP/pkg/middleware/logging.go" << 'EOF'
package middleware

import (
    "log"
    "net/http"
    "time"
)

func LoggingMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        start := time.Now()
        
        next.ServeHTTP(w, r)
        
        log.Printf(
            "%s %s %s",
            r.Method,
            r.RequestURI,
            time.Since(start),
        )
    })
}

func RateLimitMiddleware(next http.Handler) http.Handler {
    // Simple rate limiting implementation
    return next
}
EOF

echo "Sample application created at: $SAMPLE_APP"
echo
echo "=== Now simulating an LLM exploring this codebase ==="
echo

# Build the tool if needed
if [ ! -f "./llm-tool" ]; then
    go build -o llm-tool main.go
fi

# Simulate LLM exploration
cat << 'EOFLM' | ./llm-tool --root "$SAMPLE_APP"
I've been asked to analyze this Go web application. Let me explore it systematically to understand its architecture and implementation.

First, let me start with the README to get an overview:
<open README.md>

Good! This is a REST API for task management. Now let me check the project structure by looking at go.mod:
<open go.mod>

I can see it uses Gorilla Mux for routing, godotenv for configuration, and GORM with PostgreSQL for the database. Let me examine the main entry point:
<open cmd/server/main.go>

Excellent! The application structure is clear:
- It loads environment variables
- Initializes a database connection
- Sets up HTTP routes with middleware
- Serves on a configurable port

Now let me look at the task handler to understand the API implementation:
<open internal/handlers/tasks.go>

I can see the CRUD operations for tasks. Let me check the Task model:
<open internal/models/task.go>

Good! The task model includes soft deletes with GORM. Now let me examine the database configuration:
<open internal/database/connection.go>

Finally, let me check the middleware:
<open pkg/middleware/logging.go>

## Analysis Summary

Based on my exploration, here's the architecture of this Go web application:

### Architecture Overview
- **Pattern**: Clean Architecture with clear separation of concerns
- **Framework**: REST API using Gorilla Mux router
- **Database**: PostgreSQL with GORM ORM
- **Structure**: Well-organized with cmd/, internal/, and pkg/ directories

### Key Components

1. **Entry Point** (`cmd/server/main.go`)
   - Loads environment configuration
   - Initializes database
   - Sets up routing and middleware
   - Starts HTTP server

2. **API Layer** (`internal/handlers/`)
   - RESTful endpoints for CRUD operations
   - JSON request/response handling
   - Error handling for not found cases

3. **Data Layer** (`internal/models/` and `internal/database/`)
   - GORM models with soft deletes
   - PostgreSQL connection management
   - Automatic migration on startup

4. **Middleware** (`pkg/middleware/`)
   - Request logging
   - Rate limiting (placeholder implementation)

### Strengths
- Clean separation of concerns
- Standard Go project layout
- Database migrations handled automatically
- Environment-based configuration
- Middleware pipeline for cross-cutting concerns

### Areas for Improvement
1. **Error Handling**: More comprehensive error responses needed
2. **Validation**: Input validation missing in handlers
3. **Testing**: No test files found
4. **Authentication**: JWT auth mentioned in README but not implemented
5. **Rate Limiting**: Currently just a placeholder

### Security Considerations
- Environment variables used for sensitive data (good)
- No authentication implemented yet (mentioned in README)
- SQL injection protected by GORM parameterized queries

This is a solid foundation for a Go web API that follows Go best practices and clean architecture principles.
EOFLM

echo
echo "=== Example Complete ==="
echo "This demonstrates how an LLM can autonomously explore and analyze a codebase"
echo "using the file access tool to understand architecture, identify patterns, and"
echo "provide insights without needing all files loaded upfront."
echo
echo "Sample app location: $SAMPLE_APP"
echo "To clean up: rm -rf $SAMPLE_APP"

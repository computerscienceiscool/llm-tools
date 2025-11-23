package core

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/computerscienceiscool/llm-tools/internal/handlers"
)

// MockSession for testing
type MockSession struct {
	mock.Mock
}

func (m *MockSession) GetConfig() *Config {
	args := m.Called()
	return args.Get(0).(*Config)
}

func (m *MockSession) GetID() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockSession) LogAudit(command, argument string, success bool, errorMsg string) {
	m.Called(command, argument, success, errorMsg)
}

func (m *MockSession) IncrementCommandsRun() {
	m.Called()
}

func (m *MockSession) GetCommandsRun() int {
	args := m.Called()
	return args.Int(0)
}

func (m *MockSession) GetStartTime() time.Time {
	args := m.Called()
	return args.Get(0).(time.Time)
}

// MockFileHandler for testing
type MockFileHandler struct {
	mock.Mock
}

func (m *MockFileHandler) OpenFile(filePath string, maxSize int64, repoRoot string) (string, error) {
	args := m.Called(filePath, maxSize, repoRoot)
	return args.String(0), args.Error(1)
}

func (m *MockFileHandler) WriteFile(filePath, content string, maxSize int64, repoRoot string, allowedExts []string, backup bool) (handlers.WriteResult, error) {
	args := m.Called(filePath, content, maxSize, repoRoot, allowedExts, backup)
	return args.Get(0).(handlers.WriteResult), args.Error(1)
}

// MockExecHandler for testing
type MockExecHandler struct {
	mock.Mock
}

func (m *MockExecHandler) ExecuteCommand(command string, config handlers.ExecConfig) (handlers.ExecResult, error) {
	args := m.Called(command, config)
	return args.Get(0).(handlers.ExecResult), args.Error(1)
}

// MockSearchHandler for testing
type MockSearchHandler struct {
	mock.Mock
}

func (m *MockSearchHandler) Search(query string) ([]handlers.SearchResult, error) {
	args := m.Called(query)
	return args.Get(0).([]handlers.SearchResult), args.Error(1)
}

// MockPathValidator for testing
type MockPathValidator struct {
	mock.Mock
}

func (m *MockPathValidator) ValidatePath(requestedPath, repositoryRoot string, excludedPaths []string) (string, error) {
	args := m.Called(requestedPath, repositoryRoot, excludedPaths)
	return args.String(0), args.Error(1)
}

func (m *MockPathValidator) ValidateWriteExtension(filepath string, allowedExtensions []string) error {
	args := m.Called(filepath, allowedExtensions)
	return args.Error(0)
}

// TestNewCommandExecutor tests executor creation
func TestNewCommandExecutor(t *testing.T) {
	session := &MockSession{}
	fileHandler := &MockFileHandler{}
	execHandler := &MockExecHandler{}
	searchHandler := &MockSearchHandler{}
	validator := &MockPathValidator{}

	executor := NewCommandExecutor(session, fileHandler, execHandler, searchHandler, validator)

	assert.NotNil(t, executor)
	assert.IsType(t, &DefaultCommandExecutor{}, executor)
}

// TestExecuteOpen tests open command execution
func TestExecuteOpen(t *testing.T) {
	tests := []struct {
		name            string
		filepath        string
		setupMocks      func(*MockSession, *MockFileHandler, *MockPathValidator)
		expectedSuccess bool
		expectedError   string
		expectedResult  string
	}{
		{
			name:     "successful open",
			filepath: "test.txt",
			setupMocks: func(session *MockSession, fileHandler *MockFileHandler, validator *MockPathValidator) {
				config := &Config{
					RepositoryRoot: "/repo",
					MaxFileSize:    1048576,
					ExcludedPaths:  []string{".git"},
				}
				session.On("GetConfig").Return(config)
				session.On("LogAudit", "open", "test.txt", true, "")
				session.On("IncrementCommandsRun")

				validator.On("ValidatePath", "test.txt", "/repo", []string{".git"}).
					Return("/repo/test.txt", nil)

				fileHandler.On("OpenFile", "/repo/test.txt", int64(1048576), "/repo").
					Return("file content", nil)
			},
			expectedSuccess: true,
			expectedResult:  "file content",
		},
		{
			name:     "path validation error",
			filepath: "../etc/passwd",
			setupMocks: func(session *MockSession, fileHandler *MockFileHandler, validator *MockPathValidator) {
				config := &Config{
					RepositoryRoot: "/repo",
					ExcludedPaths:  []string{".git"},
				}
				session.On("GetConfig").Return(config)
				session.On("LogAudit", "open", "../etc/passwd", false, "PATH_SECURITY: path traversal detected")

				validator.On("ValidatePath", "../etc/passwd", "/repo", []string{".git"}).
					Return("", fmt.Errorf("path traversal detected"))
			},
			expectedSuccess: false,
			expectedError:   "PATH_SECURITY",
		},
		{
			name:     "file handler error",
			filepath: "missing.txt",
			setupMocks: func(session *MockSession, fileHandler *MockFileHandler, validator *MockPathValidator) {
				config := &Config{
					RepositoryRoot: "/repo",
					MaxFileSize:    1048576,
					ExcludedPaths:  []string{".git"},
				}
				session.On("GetConfig").Return(config)
				session.On("LogAudit", "open", "missing.txt", false, "file not found")

				validator.On("ValidatePath", "missing.txt", "/repo", []string{".git"}).
					Return("/repo/missing.txt", nil)

				fileHandler.On("OpenFile", "/repo/missing.txt", int64(1048576), "/repo").
					Return("", fmt.Errorf("file not found"))
			},
			expectedSuccess: false,
			expectedError:   "file not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			session := &MockSession{}
			fileHandler := &MockFileHandler{}
			execHandler := &MockExecHandler{}
			searchHandler := &MockSearchHandler{}
			validator := &MockPathValidator{}

			tt.setupMocks(session, fileHandler, validator)

			executor := NewCommandExecutor(session, fileHandler, execHandler, searchHandler, validator)
			result := executor.ExecuteOpen(tt.filepath)

			assert.Equal(t, tt.expectedSuccess, result.Success)
			if tt.expectedError != "" {
				assert.Contains(t, result.Error.Error(), tt.expectedError)
			}
			if tt.expectedResult != "" {
				assert.Equal(t, tt.expectedResult, result.Result)
			}
			assert.Positive(t, result.ExecutionTime)
			assert.Equal(t, "open", result.Command.Type)
			assert.Equal(t, tt.filepath, result.Command.Argument)

			session.AssertExpectations(t)
			fileHandler.AssertExpectations(t)
			validator.AssertExpectations(t)
		})
	}
}

// TestExecuteWrite tests write command execution
func TestExecuteWrite(t *testing.T) {
	tests := []struct {
		name            string
		filepath        string
		content         string
		setupMocks      func(*MockSession, *MockFileHandler, *MockPathValidator)
		expectedSuccess bool
		expectedError   string
		expectedAction  string
	}{
		{
			name:     "successful write - create",
			filepath: "output.txt",
			content:  "Hello, World!",
			setupMocks: func(session *MockSession, fileHandler *MockFileHandler, validator *MockPathValidator) {
				config := &Config{
					RepositoryRoot:    "/repo",
					MaxWriteSize:      1024,
					AllowedExtensions: []string{".txt"},
					BackupBeforeWrite: true,
					ExcludedPaths:     []string{".git"},
				}
				session.On("GetConfig").Return(config)
				session.On("LogAudit", "write", "output.txt", true, "")
				session.On("IncrementCommandsRun")

				validator.On("ValidatePath", "output.txt", "/repo", []string{".git"}).
					Return("/repo/output.txt", nil)
				validator.On("ValidateWriteExtension", "output.txt", []string{".txt"}).
					Return(nil)

				writeResult := handlers.WriteResult{
					Action:       "CREATED",
					BytesWritten: 13,
					BackupFile:   "",
				}
				fileHandler.On("WriteFile", "/repo/output.txt", "Hello, World!", int64(1024), "/repo", []string{".txt"}, true).
					Return(writeResult, nil)
			},
			expectedSuccess: true,
			expectedAction:  "CREATED",
		},
		{
			name:     "extension validation error",
			filepath: "script.exe",
			content:  "malicious content",
			setupMocks: func(session *MockSession, fileHandler *MockFileHandler, validator *MockPathValidator) {
				config := &Config{
					RepositoryRoot:    "/repo",
					AllowedExtensions: []string{".txt", ".go"},
					ExcludedPaths:     []string{".git"},
				}
				session.On("GetConfig").Return(config)
				session.On("LogAudit", "write", "script.exe", false, "EXTENSION_DENIED: file extension not allowed: .exe")

				validator.On("ValidatePath", "script.exe", "/repo", []string{".git"}).
					Return("/repo/script.exe", nil)
				validator.On("ValidateWriteExtension", "script.exe", []string{".txt", ".go"}).
					Return(fmt.Errorf("file extension not allowed: .exe"))
			},
			expectedSuccess: false,
			expectedError:   "EXTENSION_DENIED",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			session := &MockSession{}
			fileHandler := &MockFileHandler{}
			execHandler := &MockExecHandler{}
			searchHandler := &MockSearchHandler{}
			validator := &MockPathValidator{}

			tt.setupMocks(session, fileHandler, validator)

			executor := NewCommandExecutor(session, fileHandler, execHandler, searchHandler, validator)
			result := executor.ExecuteWrite(tt.filepath, tt.content)

			assert.Equal(t, tt.expectedSuccess, result.Success)
			if tt.expectedError != "" {
				assert.Contains(t, result.Error.Error(), tt.expectedError)
			}
			if tt.expectedAction != "" {
				assert.Equal(t, tt.expectedAction, result.Action)
			}
			assert.Equal(t, "write", result.Command.Type)
			assert.Equal(t, tt.filepath, result.Command.Argument)
			assert.Equal(t, tt.content, result.Command.Content)

			session.AssertExpectations(t)
			fileHandler.AssertExpectations(t)
			validator.AssertExpectations(t)
		})
	}
}

// TestExecuteExec tests exec command execution
func TestExecuteExec(t *testing.T) {
	tests := []struct {
		name            string
		command         string
		setupMocks      func(*MockSession, *MockExecHandler)
		expectedSuccess bool
		expectedError   string
		expectedOutput  string
	}{
		{
			name:    "successful exec",
			command: "echo hello",
			setupMocks: func(session *MockSession, execHandler *MockExecHandler) {
				config := &Config{
					ExecEnabled:        true,
					ExecWhitelist:      []string{"echo"},
					ExecTimeout:        30 * time.Second,
					ExecMemoryLimit:    "512m",
					ExecCPULimit:       2,
					ExecContainerImage: "ubuntu:22.04",
					RepositoryRoot:     "/repo",
				}
				session.On("GetConfig").Return(config)
				session.On("LogAudit", "exec", "echo hello", true, "exit_code:0,duration:0.100s")
				session.On("IncrementCommandsRun")

				execConfig := handlers.ExecConfig{
					Enabled:        true,
					Whitelist:      []string{"echo"},
					Timeout:        30 * time.Second,
					MemoryLimit:    "512m",
					CPULimit:       2,
					ContainerImage: "ubuntu:22.04",
					RepoRoot:       "/repo",
				}

				execResult := handlers.ExecResult{
					ExitCode: 0,
					Stdout:   "hello\n",
					Stderr:   "",
					Duration: time.Millisecond * 100,
				}
				execHandler.On("ExecuteCommand", "echo hello", execConfig).Return(execResult, nil)
			},
			expectedSuccess: true,
			expectedOutput:  "hello\n",
		},
		{
			name:    "exec command validation error",
			command: "rm -rf /",
			setupMocks: func(session *MockSession, execHandler *MockExecHandler) {
				config := &Config{
					ExecEnabled:    true,
					ExecWhitelist:  []string{"echo", "go test"},
					RepositoryRoot: "/repo",
				}
				session.On("GetConfig").Return(config)
				session.On("LogAudit", "exec", "rm -rf /", false, "EXEC_VALIDATION: command not in whitelist: rm")

				execConfig := handlers.ExecConfig{
					Enabled:   true,
					Whitelist: []string{"echo", "go test"},
					RepoRoot:  "/repo",
				}

				execHandler.On("ExecuteCommand", "rm -rf /", execConfig).
					Return(handlers.ExecResult{}, fmt.Errorf("EXEC_VALIDATION: command not in whitelist: rm"))
			},
			expectedSuccess: false,
			expectedError:   "EXEC_VALIDATION",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			session := &MockSession{}
			fileHandler := &MockFileHandler{}
			execHandler := &MockExecHandler{}
			searchHandler := &MockSearchHandler{}
			validator := &MockPathValidator{}

			tt.setupMocks(session, execHandler)

			executor := NewCommandExecutor(session, fileHandler, execHandler, searchHandler, validator)
			result := executor.ExecuteExec(tt.command)

			assert.Equal(t, tt.expectedSuccess, result.Success)
			if tt.expectedError != "" {
				assert.Contains(t, result.Error.Error(), tt.expectedError)
			}
			if tt.expectedOutput != "" {
				assert.Contains(t, result.Result, strings.TrimSpace(tt.expectedOutput))
			}
			assert.Equal(t, "exec", result.Command.Type)
			assert.Equal(t, tt.command, result.Command.Argument)

			session.AssertExpectations(t)
			execHandler.AssertExpectations(t)
		})
	}
}

// TestExecuteSearch tests search command execution
func TestExecuteSearch(t *testing.T) {
	tests := []struct {
		name            string
		query           string
		setupMocks      func(*MockSession, *MockSearchHandler)
		expectedSuccess bool
		expectedError   string
		expectedResults int
	}{
		{
			name:  "successful search",
			query: "authentication",
			setupMocks: func(session *MockSession, searchHandler *MockSearchHandler) {
				session.On("LogAudit", "search", "authentication", true, "results:2,duration:0.050s")
				session.On("IncrementCommandsRun")

				searchResults := []handlers.SearchResult{
					{
						FilePath: "/repo/auth.go",
						Score:    0.95,
						Lines:    100,
						Size:     2048,
						Preview:  "authentication handler implementation",
						ModTime:  time.Now(),
					},
					{
						FilePath: "/repo/login.go",
						Score:    0.80,
						Lines:    50,
						Size:     1024,
						Preview:  "user authentication logic",
						ModTime:  time.Now(),
					},
				}
				searchHandler.On("Search", "authentication").Return(searchResults, nil)
			},
			expectedSuccess: true,
			expectedResults: 2,
		},
		{
			name:  "search error",
			query: "invalid query",
			setupMocks: func(session *MockSession, searchHandler *MockSearchHandler) {
				session.On("LogAudit", "search", "invalid query", false, "search index not available")

				searchHandler.On("Search", "invalid query").
					Return([]handlers.SearchResult{}, fmt.Errorf("search index not available"))
			},
			expectedSuccess: false,
			expectedError:   "search index not available",
		},
		{
			name:  "empty search results",
			query: "nonexistent",
			setupMocks: func(session *MockSession, searchHandler *MockSearchHandler) {
				session.On("LogAudit", "search", "nonexistent", true, "results:0,duration:0.020s")
				session.On("IncrementCommandsRun")

				searchHandler.On("Search", "nonexistent").Return([]handlers.SearchResult{}, nil)
			},
			expectedSuccess: true,
			expectedResults: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			session := &MockSession{}
			fileHandler := &MockFileHandler{}
			execHandler := &MockExecHandler{}
			searchHandler := &MockSearchHandler{}
			validator := &MockPathValidator{}

			tt.setupMocks(session, searchHandler)

			executor := NewCommandExecutor(session, fileHandler, execHandler, searchHandler, validator)
			result := executor.ExecuteSearch(tt.query)

			assert.Equal(t, tt.expectedSuccess, result.Success)
			if tt.expectedError != "" {
				assert.Contains(t, result.Error.Error(), tt.expectedError)
			}
			if result.Success {
				// Verify search result formatting
				assert.Contains(t, result.Result, "SEARCH:")
				assert.Contains(t, result.Result, tt.query)
				if tt.expectedResults == 0 {
					assert.Contains(t, result.Result, "No files found")
				}
			}
			assert.Equal(t, "search", result.Command.Type)
			assert.Equal(t, tt.query, result.Command.Argument)

			session.AssertExpectations(t)
			searchHandler.AssertExpectations(t)
		})
	}
}

// TestFormatSearchResults tests search result formatting
func TestFormatSearchResults(t *testing.T) {
	executor := &DefaultCommandExecutor{}

	tests := []struct {
		name            string
		query           string
		results         []handlers.SearchResult
		duration        time.Duration
		expectedContent []string
	}{
		{
			name:     "empty results",
			query:    "missing",
			results:  []handlers.SearchResult{},
			duration: time.Millisecond * 50,
			expectedContent: []string{
				"SEARCH: missing",
				"No files found matching query",
				"END SEARCH",
			},
		},
		{
			name:  "single result",
			query: "test",
			results: []handlers.SearchResult{
				{
					FilePath: "test.go",
					Score:    0.95,
					Lines:    100,
					Size:     2048,
					Preview:  "test function implementation",
					ModTime:  time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
				},
			},
			duration: time.Millisecond * 25,
			expectedContent: []string{
				"SEARCH: test",
				"1. test.go (score: 0.95)",
				"Lines: 100",
				"Size: 2.0 KB",
				"Modified: 2024-01-01",
				"Preview: \"test function implementation\"",
				"END SEARCH",
			},
		},
		{
			name:  "multiple results",
			query: "auth",
			results: []handlers.SearchResult{
				{
					FilePath: "auth.go",
					Score:    0.95,
					Lines:    200,
					Size:     4096,
					Preview:  "authentication handler",
				},
				{
					FilePath: "login.go",
					Score:    0.80,
					Lines:    150,
					Size:     3072,
					Preview:  "login form validation",
				},
			},
			duration: time.Millisecond * 75,
			expectedContent: []string{
				"SEARCH: auth",
				"1. auth.go (score: 0.95)",
				"2. login.go (score: 0.80)",
				"authentication handler",
				"login form validation",
				"END SEARCH",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := executor.formatSearchResults(tt.query, tt.results, tt.duration)

			for _, expected := range tt.expectedContent {
				assert.Contains(t, result, expected)
			}
		})
	}
}

// TestFormatFileSize tests file size formatting
func TestFormatFileSize(t *testing.T) {
	tests := []struct {
		size     int64
		expected string
	}{
		{0, "0 B"},
		{512, "512 B"},
		{1024, "1.0 KB"},
		{1536, "1.5 KB"},
		{1048576, "1.0 MB"},
		{1073741824, "1.0 GB"},
		{1099511627776, "1.0 TB"},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("size_%d", tt.size), func(t *testing.T) {
			result := formatFileSize(tt.size)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestExecutorErrorHandling tests error handling across all commands
func TestExecutorErrorHandling(t *testing.T) {
	t.Run("nil session", func(t *testing.T) {
		// Test behavior with nil session (should not panic)
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("NewCommandExecutor panicked with nil session: %v", r)
			}
		}()

		executor := NewCommandExecutor(nil, &MockFileHandler{}, &MockExecHandler{},
			&MockSearchHandler{}, &MockPathValidator{})
		assert.NotNil(t, executor)
	})

	t.Run("execution timing", func(t *testing.T) {
		session := &MockSession{}
		fileHandler := &MockFileHandler{}
		validator := &MockPathValidator{}

		config := &Config{RepositoryRoot: "/repo", MaxFileSize: 1048576, ExcludedPaths: []string{}}
		session.On("GetConfig").Return(config)
		session.On("LogAudit", mock.Anything, mock.Anything, mock.Anything, mock.Anything)

		validator.On("ValidatePath", mock.Anything, mock.Anything, mock.Anything).
			Return("/repo/test.txt", nil)
		fileHandler.On("OpenFile", mock.Anything, mock.Anything, mock.Anything).
			Return("content", nil)
		session.On("IncrementCommandsRun")

		executor := NewCommandExecutor(session, fileHandler, &MockExecHandler{},
			&MockSearchHandler{}, validator)

		start := time.Now()
		result := executor.ExecuteOpen("test.txt")
		elapsed := time.Since(start)

		assert.True(t, result.Success)
		assert.Positive(t, result.ExecutionTime)
		assert.True(t, result.ExecutionTime <= elapsed+time.Millisecond) // Allow for small timing variance
	})
}

// BenchmarkExecuteOpen benchmarks open command execution
func BenchmarkExecuteOpen(b *testing.B) {
	session := &MockSession{}
	fileHandler := &MockFileHandler{}
	validator := &MockPathValidator{}

	config := &Config{RepositoryRoot: "/repo", MaxFileSize: 1048576, ExcludedPaths: []string{}}
	session.On("GetConfig").Return(config)
	session.On("LogAudit", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	session.On("IncrementCommandsRun")

	validator.On("ValidatePath", mock.Anything, mock.Anything, mock.Anything).
		Return("/repo/test.txt", nil)
	fileHandler.On("OpenFile", mock.Anything, mock.Anything, mock.Anything).
		Return("content", nil)

	executor := NewCommandExecutor(session, fileHandler, &MockExecHandler{},
		&MockSearchHandler{}, validator)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = executor.ExecuteOpen("test.txt")
	}
}

// BenchmarkFormatSearchResults benchmarks search result formatting
func BenchmarkFormatSearchResults(b *testing.B) {
	executor := &DefaultCommandExecutor{}

	results := make([]handlers.SearchResult, 10)
	for i := range results {
		results[i] = handlers.SearchResult{
			FilePath: fmt.Sprintf("file%d.go", i),
			Score:    0.5 + float64(i)*0.05,
			Lines:    100 + i*10,
			Size:     int64(1024 + i*512),
			Preview:  fmt.Sprintf("preview content for file %d", i),
			ModTime:  time.Now(),
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = executor.formatSearchResults("test query", results, time.Millisecond*100)
	}
}

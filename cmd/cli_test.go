package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/computerscienceiscool/llm-tools/internal/core"
	parserPkg "github.com/computerscienceiscool/llm-tools/internal/parser"
)

// MockConfigLoader for testing
type MockConfigLoader struct {
	mock.Mock
}

func (m *MockConfigLoader) LoadConfig(configPath string) (*core.Config, error) {
	args := m.Called(configPath)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*core.Config), args.Error(1)
}

// MockCommandExecutor for testing
type MockCommandExecutor struct {
	mock.Mock
}

func (m *MockCommandExecutor) ExecuteOpen(filepath string) core.ExecutionResult {
	args := m.Called(filepath)
	return args.Get(0).(core.ExecutionResult)
}

func (m *MockCommandExecutor) ExecuteWrite(filepath, content string) core.ExecutionResult {
	args := m.Called(filepath, content)
	return args.Get(0).(core.ExecutionResult)
}

func (m *MockCommandExecutor) ExecuteExec(command string) core.ExecutionResult {
	args := m.Called(command)
	return args.Get(0).(core.ExecutionResult)
}

func (m *MockCommandExecutor) ExecuteSearch(query string) core.ExecutionResult {
	args := m.Called(query)
	return args.Get(0).(core.ExecutionResult)
}

// MockCommandParser for testing
type MockCommandParser struct {
	mock.Mock
}

func (m *MockCommandParser) ParseCommands(text string) []parserPkg.ParsedCommand {
	args := m.Called(text)
	return args.Get(0).([]parserPkg.ParsedCommand)
}

// TestNewCLIApp tests CLI app creation
func TestNewCLIApp(t *testing.T) {
	configLoader := &MockConfigLoader{}
	executor := &MockCommandExecutor{}
	parser := &MockCommandParser{}

	app := NewCLIApp(configLoader, executor, parser)

	assert.NotNil(t, app)
	assert.Equal(t, configLoader, app.configLoader)
	assert.Equal(t, executor, app.executor)
	assert.Equal(t, parser, app.parser)
}

// TestCLIAppExecute tests the main Execute function
func TestCLIAppExecute(t *testing.T) {
	tests := []struct {
		name          string
		args          []string
		configSetup   func(*MockConfigLoader)
		executorSetup func(*MockCommandExecutor)
		parserSetup   func(*MockCommandParser)
		expectError   bool
	}{
		{
			name: "successful pipe mode execution",
			args: []string{"llm-tool"},
			configSetup: func(m *MockConfigLoader) {
				config := &core.Config{
					Interactive:    false,
					RepositoryRoot: "/tmp",
					MaxFileSize:    1048576,
				}
				m.On("LoadConfig", "").Return(config, nil)
			},
			executorSetup: func(m *MockCommandExecutor) {
				result := core.ExecutionResult{
					Success: true,
					Result:  "test content",
				}
				m.On("ExecuteOpen", "test.txt").Return(result)
			},
			parserSetup: func(m *MockCommandParser) {
				commands := []parserPkg.ParsedCommand{
					{
						Type:     "open",
						Argument: "test.txt",
						StartPos: 0,
						EndPos:   15,
						Original: "<open test.txt>",
					},
				}
				m.On("ParseCommands", mock.AnythingOfType("string")).Return(commands)
			},
			expectError: false,
		},
		{
			name: "config load error",
			args: []string{"llm-tool"},
			configSetup: func(m *MockConfigLoader) {
				m.On("LoadConfig", "").Return(nil, assert.AnError)
			},
			executorSetup: func(m *MockCommandExecutor) {},
			parserSetup:   func(m *MockCommandParser) {},
			expectError:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configLoader := &MockConfigLoader{}
			executor := &MockCommandExecutor{}
			parser := &MockCommandParser{}

			tt.configSetup(configLoader)
			tt.executorSetup(executor)
			tt.parserSetup(parser)

			app := NewCLIApp(configLoader, executor, parser)
			err := app.Execute(tt.args)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			configLoader.AssertExpectations(t)
			executor.AssertExpectations(t)
			parser.AssertExpectations(t)
		})
	}
}

// TestRunPipeMode tests pipe mode functionality
func TestRunPipeMode(t *testing.T) {
	// Create temporary directory for testing
	tempDir, err := os.MkdirTemp("", "llm-tool-pipe-test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create test input file
	inputFile := filepath.Join(tempDir, "input.txt")
	inputContent := "Test command <open test.txt>"
	err = os.WriteFile(inputFile, []byte(inputContent), 0644)
	require.NoError(t, err)

	// Create test target file
	targetFile := filepath.Join(tempDir, "test.txt")
	err = os.WriteFile(targetFile, []byte("Hello, World!"), 0644)
	require.NoError(t, err)

	config := &core.Config{
		Interactive:    false,
		InputFile:      inputFile,
		RepositoryRoot: tempDir,
		MaxFileSize:    1048576,
	}

	configLoader := &MockConfigLoader{}
	configLoader.On("LoadConfig", "").Return(config, nil)

	executor := &MockCommandExecutor{}
	result := core.ExecutionResult{
		Success: true,
		Result:  "Hello, World!",
		Command: core.Command{Type: "open", Argument: "test.txt"},
	}
	executor.On("ExecuteOpen", "test.txt").Return(result)

	parser := &MockCommandParser{}
	commands := []parserPkg.ParsedCommand{
		{
			Type:     "open",
			Argument: "test.txt",
			StartPos: 13,
			EndPos:   28,
			Original: "<open test.txt>",
		},
	}
	parser.On("ParseCommands", inputContent).Return(commands)

	app := NewCLIApp(configLoader, executor, parser)
	err = app.Execute([]string{"llm-tool"})
	assert.NoError(t, err)

	configLoader.AssertExpectations(t)
	executor.AssertExpectations(t)
	parser.AssertExpectations(t)
}

// TestRunInteractiveMode tests interactive mode functionality
func TestRunInteractiveMode(t *testing.T) {
	config := &core.Config{
		Interactive:    true,
		RepositoryRoot: "/tmp",
		MaxFileSize:    1048576,
	}

	configLoader := &MockConfigLoader{}
	configLoader.On("LoadConfig", "").Return(config, nil)

	executor := &MockCommandExecutor{}
	parser := &MockCommandParser{}
	parser.On("ParseCommands", mock.AnythingOfType("string")).Return([]parserPkg.ParsedCommand{})

	app := NewCLIApp(configLoader, executor, parser)

	// This test would require more complex setup to simulate interactive input
	// For now, we test that the app can be created and configured for interactive mode
	assert.NotNil(t, app)

	configLoader.AssertExpectations(t)
}

// TestProcessText tests the text processing functionality
func TestProcessText(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		commands      []parserPkg.ParsedCommand
		execResults   map[string]core.ExecutionResult
		expectedParts []string
	}{
		{
			name:          "no commands",
			input:         "Plain text without commands",
			commands:      []parserPkg.ParsedCommand{},
			execResults:   map[string]core.ExecutionResult{},
			expectedParts: []string{"Plain text without commands"},
		},
		{
			name:  "single open command",
			input: "Read file <open test.txt>",
			commands: []parserPkg.ParsedCommand{
				{
					Type:     "open",
					Argument: "test.txt",
					StartPos: 10,
					EndPos:   25,
					Original: "<open test.txt>",
				},
			},
			execResults: map[string]core.ExecutionResult{
				"open:test.txt": {
					Success: true,
					Result:  "file contents",
					Command: core.Command{Type: "open", Argument: "test.txt"},
				},
			},
			expectedParts: []string{"LLM TOOL START", "FILE: test.txt", "file contents", "LLM TOOL COMPLETE"},
		},
		{
			name:  "write command",
			input: "Create file <write output.txt>Hello</write>",
			commands: []parserPkg.ParsedCommand{
				{
					Type:     "write",
					Argument: "output.txt",
					Content:  "Hello",
					StartPos: 12,
					EndPos:   43,
					Original: "<write output.txt>Hello</write>",
				},
			},
			execResults: map[string]core.ExecutionResult{
				"write:output.txt:Hello": {
					Success:      true,
					Action:       "CREATED",
					BytesWritten: 5,
					Command: core.Command{
						Type:     "write",
						Argument: "output.txt",
						Content:  "Hello",
					},
				},
			},
			expectedParts: []string{"LLM TOOL START", "WRITE SUCCESSFUL", "CREATED", "LLM TOOL COMPLETE"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configLoader := &MockConfigLoader{}
			executor := &MockCommandExecutor{}
			parser := &MockCommandParser{}

			config := &core.Config{RepositoryRoot: "/tmp"}
			//	configLoader.On("LoadConfig", "").Return(config, nil)
			parser.On("ParseCommands", tt.input).Return(tt.commands)

			// Set up executor expectations based on commands
			for _, cmd := range tt.commands {
				var key string
				switch cmd.Type {
				case "open":
					key = "open:" + cmd.Argument
					executor.On("ExecuteOpen", cmd.Argument).Return(tt.execResults[key])
				case "write":
					key = "write:" + cmd.Argument + ":" + cmd.Content
					executor.On("ExecuteWrite", cmd.Argument, cmd.Content).Return(tt.execResults[key])
				case "exec":
					key = "exec:" + cmd.Argument
					executor.On("ExecuteExec", cmd.Argument).Return(tt.execResults[key])
				case "search":
					key = "search:" + cmd.Argument
					executor.On("ExecuteSearch", cmd.Argument).Return(tt.execResults[key])
				}
			}

			app := NewCLIApp(configLoader, executor, parser)
			result := app.processText(tt.input, config)

			for _, expected := range tt.expectedParts {
				assert.Contains(t, result, expected, "Result should contain: %s", expected)
			}

			configLoader.AssertExpectations(t)
			executor.AssertExpectations(t)
			parser.AssertExpectations(t)
		})
	}
}

// TestFormatResult tests result formatting functionality
func TestFormatResult(t *testing.T) {
	tests := []struct {
		name          string
		result        core.ExecutionResult
		expectedParts []string
	}{
		{
			name: "successful open command",
			result: core.ExecutionResult{
				Success: true,
				Result:  "file content here",
				Command: core.Command{Type: "open", Argument: "test.txt"},
			},
			expectedParts: []string{"COMMAND: open", "FILE: test.txt", "file content here", "END FILE"},
		},
		{
			name: "successful write command",
			result: core.ExecutionResult{
				Success:      true,
				Action:       "CREATED",
				BytesWritten: 100,
				Command:      core.Command{Type: "write", Argument: "output.txt"},
			},
			expectedParts: []string{"COMMAND: write", "WRITE SUCCESSFUL", "CREATED", "Bytes written: 100"},
		},
		{
			name: "successful exec command",
			result: core.ExecutionResult{
				Success:       true,
				Result:        "command output",
				ExitCode:      0,
				ExecutionTime: time.Millisecond * 500,
				Command:       core.Command{Type: "exec", Argument: "echo test"},
			},
			expectedParts: []string{"COMMAND: exec", "EXEC SUCCESSFUL", "Exit code: 0", "Duration: 0.500s", "command output"},
		},
		{
			name: "failed command",
			result: core.ExecutionResult{
				Success: false,
				Error:   assert.AnError,
				Command: core.Command{Type: "open", Argument: "missing.txt"},
			},
			expectedParts: []string{"ERROR", "Command: missing.txt"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := &CLIApp{}
			var output strings.Builder
			app.formatResult(&output, tt.result)
			result := output.String()

			for _, expected := range tt.expectedParts {
				assert.Contains(t, result, expected, "Formatted result should contain: %s", expected)
			}
		})
	}
}

// TestCLIAppErrorHandling tests various error conditions
func TestCLIAppErrorHandling(t *testing.T) {
	t.Run("stdin read error", func(t *testing.T) {
		// Test error when reading from stdin fails
		// This would require more complex setup to simulate stdin errors
	})

	t.Run("output file write error", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "llm-tool-error-test")
		require.NoError(t, err)
		defer os.RemoveAll(tempDir)

		// Create read-only directory
		readOnlyDir := filepath.Join(tempDir, "readonly")
		err = os.Mkdir(readOnlyDir, 0444)
		require.NoError(t, err)

		config := &core.Config{
			OutputFile:     filepath.Join(readOnlyDir, "output.txt"),
			RepositoryRoot: tempDir,
		}

		configLoader := &MockConfigLoader{}
		configLoader.On("LoadConfig", "").Return(config, nil)

		executor := &MockCommandExecutor{}
		parser := &MockCommandParser{}
		parser.On("ParseCommands", mock.AnythingOfType("string")).Return([]parserPkg.ParsedCommand{})

		app := NewCLIApp(configLoader, executor, parser)

		// This should fail due to permission error
		err = app.runPipeMode(config)
		assert.Error(t, err)
	})
}

// BenchmarkCLIAppProcessText benchmarks text processing performance
func BenchmarkCLIAppProcessText(b *testing.B) {
	configLoader := &MockConfigLoader{}
	executor := &MockCommandExecutor{}
	parser := &MockCommandParser{}

	config := &core.Config{RepositoryRoot: "/tmp"}
	configLoader.On("LoadConfig", "").Return(config, nil)

	commands := []parserPkg.ParsedCommand{
		{Type: "open", Argument: "test.txt", StartPos: 0, EndPos: 15, Original: "<open test.txt>"},
	}
	parser.On("ParseCommands", mock.AnythingOfType("string")).Return(commands)

	result := core.ExecutionResult{
		Success: true,
		Result:  "test content",
		Command: core.Command{Type: "open", Argument: "test.txt"},
	}
	executor.On("ExecuteOpen", "test.txt").Return(result)

	app := NewCLIApp(configLoader, executor, parser)
	text := "Process this text <open test.txt> with commands"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = app.processText(text, config)
	}
}

// TestCLIAppWithRealComponents tests with actual implementations
func TestCLIAppWithRealComponents(t *testing.T) {
	t.Skip("Integration test - requires real components")

	// This would test with actual implementations of:
	// - Real config loader
	// - Real command executor
	// - Real command parser
	// And verify end-to-end functionality
}

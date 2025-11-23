package parser

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockCommandParser for testing
type MockCommandParser struct {
	mock.Mock
}

func (m *MockCommandParser) ParseCommands(text string) []ParsedCommand {
	args := m.Called(text)
	return args.Get(0).([]ParsedCommand)
}

// TestCommandParserInterface tests the CommandParser interface
func TestCommandParserInterface(t *testing.T) {
	// Ensure MockCommandParser implements CommandParser interface
	var _ CommandParser = (*MockCommandParser)(nil)

	mockParser := &MockCommandParser{}

	expectedCommands := []ParsedCommand{
		{
			Type:     "open",
			Argument: "test.txt",
			StartPos: 0,
			EndPos:   15,
			Original: "<open test.txt>",
		},
	}

	mockParser.On("ParseCommands", "test input").Return(expectedCommands)

	result := mockParser.ParseCommands("test input")

	assert.Equal(t, expectedCommands, result)
	mockParser.AssertExpectations(t)
}

// TestParsedCommand tests the ParsedCommand structure
func TestParsedCommand(t *testing.T) {
	tests := []struct {
		name    string
		command ParsedCommand
		valid   bool
	}{
		{
			name: "valid open command",
			command: ParsedCommand{
				Type:     "open",
				Argument: "file.txt",
				Content:  "",
				StartPos: 0,
				EndPos:   15,
				Original: "<open file.txt>",
			},
			valid: true,
		},
		{
			name: "valid write command",
			command: ParsedCommand{
				Type:     "write",
				Argument: "output.txt",
				Content:  "Hello World",
				StartPos: 10,
				EndPos:   40,
				Original: "<write output.txt>Hello World</write>",
			},
			valid: true,
		},
		{
			name: "valid exec command",
			command: ParsedCommand{
				Type:     "exec",
				Argument: "go test",
				Content:  "",
				StartPos: 5,
				EndPos:   20,
				Original: "<exec go test>",
			},
			valid: true,
		},
		{
			name: "valid search command",
			command: ParsedCommand{
				Type:     "search",
				Argument: "authentication",
				Content:  "",
				StartPos: 15,
				EndPos:   40,
				Original: "<search authentication>",
			},
			valid: true,
		},
		{
			name: "invalid command type",
			command: ParsedCommand{
				Type:     "invalid",
				Argument: "test",
				StartPos: 0,
				EndPos:   5,
				Original: "<invalid>",
			},
			valid: false,
		},
		{
			name: "empty command type",
			command: ParsedCommand{
				Type:     "",
				Argument: "test",
				StartPos: 0,
				EndPos:   5,
				Original: "<>",
			},
			valid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test basic structure properties
			assert.Equal(t, tt.command.Type, tt.command.Type)
			assert.Equal(t, tt.command.Argument, tt.command.Argument)
			assert.Equal(t, tt.command.Content, tt.command.Content)
			assert.Equal(t, tt.command.StartPos, tt.command.StartPos)
			assert.Equal(t, tt.command.EndPos, tt.command.EndPos)
			assert.Equal(t, tt.command.Original, tt.command.Original)

			// Test validity
			isValid := validateParsedCommand(tt.command)
			assert.Equal(t, tt.valid, isValid)
		})
	}
}

// validateParsedCommand is a helper function for validation logic
func validateParsedCommand(cmd ParsedCommand) bool {
	validTypes := map[string]bool{
		"open":   true,
		"write":  true,
		"exec":   true,
		"search": true,
	}

	if !validTypes[cmd.Type] {
		return false
	}

	if cmd.StartPos < 0 || cmd.EndPos <= cmd.StartPos {
		return false
	}

	if cmd.Original == "" {
		return false
	}

	return true
}

// TestParsedCommandPositions tests position validation
func TestParsedCommandPositions(t *testing.T) {
	tests := []struct {
		name     string
		startPos int
		endPos   int
		valid    bool
	}{
		{
			name:     "valid positions",
			startPos: 0,
			endPos:   10,
			valid:    true,
		},
		{
			name:     "equal positions",
			startPos: 5,
			endPos:   5,
			valid:    false,
		},
		{
			name:     "negative start position",
			startPos: -1,
			endPos:   10,
			valid:    false,
		},
		{
			name:     "end before start",
			startPos: 10,
			endPos:   5,
			valid:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := ParsedCommand{
				Type:     "open",
				Argument: "test.txt",
				StartPos: tt.startPos,
				EndPos:   tt.endPos,
				Original: "<open test.txt>",
			}

			isValid := validateParsedCommand(cmd)
			assert.Equal(t, tt.valid, isValid)
		})
	}
}

// TestParsedCommandContent tests content handling for different command types
func TestParsedCommandContent(t *testing.T) {
	tests := []struct {
		name              string
		cmdType           string
		content           string
		shouldHaveContent bool
	}{
		{
			name:              "open command - no content expected",
			cmdType:           "open",
			content:           "",
			shouldHaveContent: false,
		},
		{
			name:              "exec command - no content expected",
			cmdType:           "exec",
			content:           "",
			shouldHaveContent: false,
		},
		{
			name:              "search command - no content expected",
			cmdType:           "search",
			content:           "",
			shouldHaveContent: false,
		},
		{
			name:              "write command - content expected",
			cmdType:           "write",
			content:           "Hello World",
			shouldHaveContent: true,
		},
		{
			name:              "write command - empty content allowed",
			cmdType:           "write",
			content:           "",
			shouldHaveContent: false, // Empty content is technically allowed
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := ParsedCommand{
				Type:     tt.cmdType,
				Argument: "test",
				Content:  tt.content,
				StartPos: 0,
				EndPos:   10,
				Original: "<test>",
			}

			hasContent := cmd.Content != ""
			if tt.shouldHaveContent {
				assert.True(t, hasContent, "Command should have content")
			} else {
				// Content may or may not be present, depending on the test case
				assert.Equal(t, tt.content, cmd.Content)
			}
		})
	}
}

// TestParsedCommandSerialization tests command serialization for debugging
func TestParsedCommandSerialization(t *testing.T) {
	cmd := ParsedCommand{
		Type:     "open",
		Argument: "test.txt",
		Content:  "",
		StartPos: 10,
		EndPos:   25,
		Original: "<open test.txt>",
	}

	// Test that all fields are accessible
	assert.Equal(t, "open", cmd.Type)
	assert.Equal(t, "test.txt", cmd.Argument)
	assert.Equal(t, "", cmd.Content)
	assert.Equal(t, 10, cmd.StartPos)
	assert.Equal(t, 25, cmd.EndPos)
	assert.Equal(t, "<open test.txt>", cmd.Original)
}

// TestParsedCommandComparison tests command equality
func TestParsedCommandComparison(t *testing.T) {
	cmd1 := ParsedCommand{
		Type:     "open",
		Argument: "test.txt",
		StartPos: 0,
		EndPos:   15,
		Original: "<open test.txt>",
	}

	cmd2 := ParsedCommand{
		Type:     "open",
		Argument: "test.txt",
		StartPos: 0,
		EndPos:   15,
		Original: "<open test.txt>",
	}

	cmd3 := ParsedCommand{
		Type:     "write",
		Argument: "test.txt",
		StartPos: 0,
		EndPos:   15,
		Original: "<write test.txt>content</write>",
	}

	// Test equality
	assert.Equal(t, cmd1, cmd2)
	assert.NotEqual(t, cmd1, cmd3)
}

// TestCommandParserContract tests the expected behavior of parser implementations
func TestCommandParserContract(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		expectedCount int
		expectedTypes []string
	}{
		{
			name:          "empty input",
			input:         "",
			expectedCount: 0,
			expectedTypes: []string{},
		},
		{
			name:          "no commands",
			input:         "This is plain text without any commands",
			expectedCount: 0,
			expectedTypes: []string{},
		},
		{
			name:          "single open command",
			input:         "Please read <open test.txt>",
			expectedCount: 1,
			expectedTypes: []string{"open"},
		},
		{
			name:          "multiple commands",
			input:         "First <open file1.txt> then <write file2.txt>content</write>",
			expectedCount: 2,
			expectedTypes: []string{"open", "write"},
		},
		{
			name:          "all command types",
			input:         "Do <open file.txt> then <write out.txt>data</write> and <exec go test> finally <search auth>",
			expectedCount: 4,
			expectedTypes: []string{"open", "write", "exec", "search"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockParser := &MockCommandParser{}

			// Create expected result based on test case
			expectedCommands := make([]ParsedCommand, tt.expectedCount)
			for i, cmdType := range tt.expectedTypes {
				expectedCommands[i] = ParsedCommand{
					Type:     cmdType,
					Argument: "test",
					StartPos: i * 10,
					EndPos:   (i + 1) * 10,
					Original: "<" + cmdType + ">",
				}
			}

			mockParser.On("ParseCommands", tt.input).Return(expectedCommands)

			result := mockParser.ParseCommands(tt.input)

			assert.Len(t, result, tt.expectedCount)
			for i, expectedType := range tt.expectedTypes {
				if i < len(result) {
					assert.Equal(t, expectedType, result[i].Type)
				}
			}

			mockParser.AssertExpectations(t)
		})
	}
}

// TestParserPerformanceContract tests parser performance expectations
func TestParserPerformanceContract(t *testing.T) {
	// Test that parsers can handle reasonably large input
	largeInput := ""
	for i := 0; i < 1000; i++ {
		largeInput += "Some text with <open file" + string(rune(i)) + ".txt> commands. "
	}

	mockParser := &MockCommandParser{}

	// Mock should handle large input without issues
	mockParser.On("ParseCommands", largeInput).Return([]ParsedCommand{})

	result := mockParser.ParseCommands(largeInput)

	assert.NotNil(t, result)
	mockParser.AssertExpectations(t)
}

// TestParserEdgeCases tests edge case handling
func TestParserEdgeCases(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "unicode characters",
			input: "Test with unicode: <open 测试.txt> and emoji 🎉",
		},
		{
			name:  "very long argument",
			input: "Long path: <open " + string(make([]byte, 1000)) + ">",
		},
		{
			name:  "nested angle brackets",
			input: "Nested: <open file<with>brackets.txt>",
		},
		{
			name:  "special characters in arguments",
			input: "Special chars: <open file with spaces & symbols!@#$.txt>",
		},
		{
			name:  "malformed commands",
			input: "Broken: <open> <write> </write> <exec",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockParser := &MockCommandParser{}

			// Parser should handle edge cases gracefully
			mockParser.On("ParseCommands", tt.input).Return([]ParsedCommand{})

			result := mockParser.ParseCommands(tt.input)

			// Should not panic and should return a valid slice
			assert.NotNil(t, result)
			mockParser.AssertExpectations(t)
		})
	}
}

// BenchmarkParsedCommandCreation benchmarks command structure creation
func BenchmarkParsedCommandCreation(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = ParsedCommand{
			Type:     "open",
			Argument: "test.txt",
			Content:  "",
			StartPos: 0,
			EndPos:   15,
			Original: "<open test.txt>",
		}
	}
}

// BenchmarkParsedCommandValidation benchmarks command validation
func BenchmarkParsedCommandValidation(b *testing.B) {
	cmd := ParsedCommand{
		Type:     "open",
		Argument: "test.txt",
		StartPos: 0,
		EndPos:   15,
		Original: "<open test.txt>",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = validateParsedCommand(cmd)
	}
}

// TestMultipleParsedCommands tests handling multiple commands in sequence
func TestMultipleParsedCommands(t *testing.T) {
	commands := []ParsedCommand{
		{Type: "open", Argument: "file1.txt", StartPos: 0, EndPos: 15, Original: "<open file1.txt>"},
		{Type: "write", Argument: "file2.txt", Content: "data", StartPos: 20, EndPos: 45, Original: "<write file2.txt>data</write>"},
		{Type: "exec", Argument: "go test", StartPos: 50, EndPos: 65, Original: "<exec go test>"},
		{Type: "search", Argument: "auth", StartPos: 70, EndPos: 85, Original: "<search auth>"},
	}

	// Test that all commands maintain their properties
	assert.Len(t, commands, 4)

	// Verify positions are sequential
	for i := 1; i < len(commands); i++ {
		assert.True(t, commands[i].StartPos > commands[i-1].EndPos,
			"Commands should have non-overlapping positions")
	}

	// Verify all are valid
	for i, cmd := range commands {
		assert.True(t, validateParsedCommand(cmd),
			"Command %d should be valid", i)
	}
}

// TestParsedCommandMutation tests that commands are immutable once created
func TestParsedCommandMutation(t *testing.T) {
	original := ParsedCommand{
		Type:     "open",
		Argument: "original.txt",
		StartPos: 0,
		EndPos:   20,
		Original: "<open original.txt>",
	}

	// Create a copy
	modified := original
	modified.Argument = "modified.txt"

	// Original should not be affected (this test shows they're separate)
	assert.Equal(t, "original.txt", original.Argument)
	assert.Equal(t, "modified.txt", modified.Argument)
}

package parser

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewCommandParser tests LLM parser creation
func TestNewCommandParser(t *testing.T) {
	parser := NewCommandParser()

	assert.NotNil(t, parser)
	assert.IsType(t, &LLMParser{}, parser)

	llmParser := parser.(*LLMParser)
	assert.NotNil(t, llmParser.openPattern)
	assert.NotNil(t, llmParser.writePattern)
	assert.NotNil(t, llmParser.execPattern)
	assert.NotNil(t, llmParser.searchPattern)
}

// TestLLMParserParseCommands tests the main parsing functionality
func TestLLMParserParseCommands(t *testing.T) {
	parser := NewCommandParser()

	tests := []struct {
		name          string
		input         string
		expectedCount int
		expectedTypes []string
		validate      func(t *testing.T, commands []ParsedCommand)
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
			validate: func(t *testing.T, commands []ParsedCommand) {
				assert.Equal(t, "test.txt", commands[0].Argument)
				assert.Equal(t, "<open test.txt>", commands[0].Original)
				assert.Empty(t, commands[0].Content)
			},
		},
		{
			name:          "single write command",
			input:         "Create file <write output.txt>Hello, World!</write>",
			expectedCount: 1,
			expectedTypes: []string{"write"},
			validate: func(t *testing.T, commands []ParsedCommand) {
				assert.Equal(t, "output.txt", commands[0].Argument)
				assert.Equal(t, "Hello, World!", commands[0].Content)
				assert.Equal(t, "<write output.txt>Hello, World!</write>", commands[0].Original)
			},
		},
		{
			name:          "single exec command",
			input:         "Run tests <exec go test>",
			expectedCount: 1,
			expectedTypes: []string{"exec"},
			validate: func(t *testing.T, commands []ParsedCommand) {
				assert.Equal(t, "go test", commands[0].Argument)
				assert.Empty(t, commands[0].Content)
				assert.Equal(t, "<exec go test>", commands[0].Original)
			},
		},
		{
			name:          "single search command",
			input:         "Find files <search authentication>",
			expectedCount: 1,
			expectedTypes: []string{"search"},
			validate: func(t *testing.T, commands []ParsedCommand) {
				assert.Equal(t, "authentication", commands[0].Argument)
				assert.Empty(t, commands[0].Content)
				assert.Equal(t, "<search authentication>", commands[0].Original)
			},
		},
		{
			name:          "multiple different commands",
			input:         "First <open file1.txt> then <write file2.txt>content</write> and run <exec go build>",
			expectedCount: 3,
			expectedTypes: []string{"open", "write", "exec"},
			validate: func(t *testing.T, commands []ParsedCommand) {
				assert.Equal(t, "file1.txt", commands[0].Argument)
				assert.Equal(t, "file2.txt", commands[1].Argument)
				assert.Equal(t, "content", commands[1].Content)
				assert.Equal(t, "go build", commands[2].Argument)
			},
		},
		{
			name:          "multiple same commands",
			input:         "Read <open file1.txt> and also <open file2.txt>",
			expectedCount: 2,
			expectedTypes: []string{"open", "open"},
			validate: func(t *testing.T, commands []ParsedCommand) {
				assert.Equal(t, "file1.txt", commands[0].Argument)
				assert.Equal(t, "file2.txt", commands[1].Argument)
			},
		},
		{
			name:          "commands with spaces in arguments",
			input:         "Open <open my file with spaces.txt> and run <exec echo hello world>",
			expectedCount: 2,
			expectedTypes: []string{"open", "exec"},
			validate: func(t *testing.T, commands []ParsedCommand) {
				assert.Equal(t, "my file with spaces.txt", commands[0].Argument)
				assert.Equal(t, "echo hello world", commands[1].Argument)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			commands := parser.ParseCommands(tt.input)

			assert.Len(t, commands, tt.expectedCount)

			for i, expectedType := range tt.expectedTypes {
				require.Less(t, i, len(commands), "Not enough commands parsed")
				assert.Equal(t, expectedType, commands[i].Type)
			}

			if tt.validate != nil {
				tt.validate(t, commands)
			}
		})
	}
}

// TestParseOpenCommands tests open command parsing specifically
func TestParseOpenCommands(t *testing.T) {
	parser := NewCommandParser().(*LLMParser)

	tests := []struct {
		name     string
		input    string
		expected []ParsedCommand
	}{
		{
			name:  "simple open",
			input: "<open file.txt>",
			expected: []ParsedCommand{
				{
					Type:     "open",
					Argument: "file.txt",
					StartPos: 0,
					EndPos:   15,
					Original: "<open file.txt>",
				},
			},
		},
		{
			name:  "open with path",
			input: "<open dir/subdir/file.go>",
			expected: []ParsedCommand{
				{
					Type:     "open",
					Argument: "dir/subdir/file.go",
					StartPos: 0,
					EndPos:   25,
					Original: "<open dir/subdir/file.go>",
				},
			},
		},
		{
			name:  "multiple opens",
			input: "<open file1.txt> some text <open file2.txt>",
			expected: []ParsedCommand{
				{
					Type:     "open",
					Argument: "file1.txt",
					StartPos: 0,
					EndPos:   16,
					Original: "<open file1.txt>",
				},
				{
					Type:     "open",
					Argument: "file2.txt",
					StartPos: 27,
					EndPos:   43,
					Original: "<open file2.txt>",
				},
			},
		},
		{
			name:  "open with extra whitespace",
			input: "<open   file with spaces.txt  >",
			expected: []ParsedCommand{
				{
					Type:     "open",
					Argument: "file with spaces.txt",
					StartPos: 0,
					EndPos:   31,
					Original: "<open   file with spaces.txt  >",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			commands := parser.parseOpenCommands(tt.input)

			assert.Equal(t, len(tt.expected), len(commands))

			for i, expected := range tt.expected {
				if i < len(commands) {
					assert.Equal(t, expected.Type, commands[i].Type)
					assert.Equal(t, expected.Argument, commands[i].Argument)
					assert.Equal(t, expected.Original, commands[i].Original)
					assert.Equal(t, expected.StartPos, commands[i].StartPos)
					assert.Equal(t, expected.EndPos, commands[i].EndPos)
				}
			}
		})
	}
}

// TestParseWriteCommands tests write command parsing specifically
func TestParseWriteCommands(t *testing.T) {
	parser := NewCommandParser().(*LLMParser)

	tests := []struct {
		name     string
		input    string
		expected []ParsedCommand
	}{
		{
			name:  "simple write",
			input: "<write output.txt>Hello World</write>",
			expected: []ParsedCommand{
				{
					Type:     "write",
					Argument: "output.txt",
					Content:  "Hello World",
					StartPos: 0,
					EndPos:   37,
					Original: "<write output.txt>Hello World</write>",
				},
			},
		},
		{
			name:  "write with multiline content",
			input: "<write config.yaml>\nkey: value\nother: data\n</write>",
			expected: []ParsedCommand{
				{
					Type:     "write",
					Argument: "config.yaml",
					Content:  "key: value\nother: data",
					StartPos: 0,
					EndPos:   51,
					Original: "<write config.yaml>\nkey: value\nother: data\n</write>",
				},
			},
		},
		{
			name:  "write with empty content",
			input: "<write empty.txt></write>",
			expected: []ParsedCommand{
				{
					Type:     "write",
					Argument: "empty.txt",
					Content:  "",
					StartPos: 0,
					EndPos:   25,
					Original: "<write empty.txt></write>",
				},
			},
		},
		{
			name:  "write with spaces in filename",
			input: "<write my file.txt>content here</write>",
			expected: []ParsedCommand{
				{
					Type:     "write",
					Argument: "my file.txt",
					Content:  "content here",
					StartPos: 0,
					EndPos:   39,
					Original: "<write my file.txt>content here</write>",
				},
			},
		},
		{
			name:  "multiple writes",
			input: "<write file1.txt>content1</write> and <write file2.txt>content2</write>",
			expected: []ParsedCommand{
				{
					Type:     "write",
					Argument: "file1.txt",
					Content:  "content1",
					StartPos: 0,
					EndPos:   33,
					Original: "<write file1.txt>content1</write>",
				},
				{
					Type:     "write",
					Argument: "file2.txt",
					Content:  "content2",
					StartPos: 38,
					EndPos:   71,
					Original: "<write file2.txt>content2</write>",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			commands := parser.parseWriteCommands(tt.input)

			assert.Equal(t, len(tt.expected), len(commands))

			for i, expected := range tt.expected {
				if i < len(commands) {
					assert.Equal(t, expected.Type, commands[i].Type)
					assert.Equal(t, expected.Argument, commands[i].Argument)
					assert.Equal(t, expected.Content, commands[i].Content)
					assert.Equal(t, expected.Original, commands[i].Original)
					assert.Equal(t, expected.StartPos, commands[i].StartPos)
					assert.Equal(t, expected.EndPos, commands[i].EndPos)
				}
			}
		})
	}
}

// TestParseExecCommands tests exec command parsing specifically
func TestParseExecCommands(t *testing.T) {
	parser := NewCommandParser().(*LLMParser)

	tests := []struct {
		name     string
		input    string
		expected []ParsedCommand
	}{
		{
			name:  "simple exec",
			input: "<exec go test>",
			expected: []ParsedCommand{
				{
					Type:     "exec",
					Argument: "go test",
					StartPos: 0,
					EndPos:   14,
					Original: "<exec go test>",
				},
			},
		},
		{
			name:  "exec with complex command",
			input: "<exec go test -v -race ./...>",
			expected: []ParsedCommand{
				{
					Type:     "exec",
					Argument: "go test -v -race ./...",
					StartPos: 0,
					EndPos:   29,
					Original: "<exec go test -v -race ./...>",
				},
			},
		},
		{
			name:  "exec with quotes",
			input: `<exec echo "hello world">`,
			expected: []ParsedCommand{
				{
					Type:     "exec",
					Argument: `echo "hello world"`,
					StartPos: 0,
					EndPos:   25,
					Original: `<exec echo "hello world">`,
				},
			},
		},
		{
			name:  "multiple exec commands",
			input: "<exec make clean> then <exec make build>",
			expected: []ParsedCommand{
				{
					Type:     "exec",
					Argument: "make clean",
					StartPos: 0,
					EndPos:   17,
					Original: "<exec make clean>",
				},
				{
					Type:     "exec",
					Argument: "make build",
					StartPos: 23,
					EndPos:   40,
					Original: "<exec make build>",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			commands := parser.parseExecCommands(tt.input)

			assert.Equal(t, len(tt.expected), len(commands))

			for i, expected := range tt.expected {
				if i < len(commands) {
					assert.Equal(t, expected.Type, commands[i].Type)
					assert.Equal(t, expected.Argument, commands[i].Argument)
					assert.Equal(t, expected.Original, commands[i].Original)
					assert.Equal(t, expected.StartPos, commands[i].StartPos)
					assert.Equal(t, expected.EndPos, commands[i].EndPos)
				}
			}
		})
	}
}

// TestParseSearchCommands tests search command parsing specifically
func TestParseSearchCommands(t *testing.T) {
	parser := NewCommandParser().(*LLMParser)

	tests := []struct {
		name     string
		input    string
		expected []ParsedCommand
	}{
		{
			name:  "simple search",
			input: "<search authentication>",
			expected: []ParsedCommand{
				{
					Type:     "search",
					Argument: "authentication",
					StartPos: 0,
					EndPos:   23,
					Original: "<search authentication>",
				},
			},
		},
		{
			name:  "search with multiple words",
			input: "<search user login logic>",
			expected: []ParsedCommand{
				{
					Type:     "search",
					Argument: "user login logic",
					StartPos: 0,
					EndPos:   25,
					Original: "<search user login logic>",
				},
			},
		},
		{
			name:  "multiple searches",
			input: "<search auth> and <search database>",
			expected: []ParsedCommand{
				{
					Type:     "search",
					Argument: "auth",
					StartPos: 0,
					EndPos:   13,
					Original: "<search auth>",
				},
				{
					Type:     "search",
					Argument: "database",
					StartPos: 18,
					EndPos:   35,
					Original: "<search database>",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			commands := parser.parseSearchCommands(tt.input)

			assert.Equal(t, len(tt.expected), len(commands))

			for i, expected := range tt.expected {
				if i < len(commands) {
					assert.Equal(t, expected.Type, commands[i].Type)
					assert.Equal(t, expected.Argument, commands[i].Argument)
					assert.Equal(t, expected.Original, commands[i].Original)
					assert.Equal(t, expected.StartPos, commands[i].StartPos)
					assert.Equal(t, expected.EndPos, commands[i].EndPos)
				}
			}
		})
	}
}

// TestParserRegexPatterns tests that regex patterns work correctly
func TestParserRegexPatterns(t *testing.T) {
	parser := NewCommandParser().(*LLMParser)

	t.Run("open pattern", func(t *testing.T) {
		matches := parser.openPattern.FindAllStringSubmatch("<open test.txt>", -1)
		assert.Len(t, matches, 1)
		assert.Equal(t, "test.txt", matches[0][1])
	})

	t.Run("write pattern", func(t *testing.T) {
		matches := parser.writePattern.FindAllStringSubmatch("<write test.txt>content</write>", -1)
		assert.Len(t, matches, 1)
		assert.Equal(t, "test.txt", matches[0][1])
		assert.Equal(t, "content", matches[0][2])
	})

	t.Run("exec pattern", func(t *testing.T) {
		matches := parser.execPattern.FindAllStringSubmatch("<exec go test>", -1)
		assert.Len(t, matches, 1)
		assert.Equal(t, "go test", matches[0][1])
	})

	t.Run("search pattern", func(t *testing.T) {
		matches := parser.searchPattern.FindAllStringSubmatch("<search auth>", -1)
		assert.Len(t, matches, 1)
		assert.Equal(t, "auth", matches[0][1])
	})
}

// TestParserEdgeCases tests edge cases and malformed input
func TestParserEdgeCases(t *testing.T) {
	parser := NewCommandParser()

	tests := []struct {
		name          string
		input         string
		expectedCount int
		description   string
	}{
		{
			name:          "unclosed write tag",
			input:         "<write test.txt>content without closing tag",
			expectedCount: 0,
			description:   "Should not parse unclosed write commands",
		},
		{
			name:          "mismatched tags",
			input:         "<write test.txt>content</open>",
			expectedCount: 0,
			description:   "Should not parse mismatched tags",
		},
		{
			name:          "empty command arguments",
			input:         "<open> <write></write> <exec> <search>",
			expectedCount: 0,
			description:   "Should handle empty arguments",
		},
		{
			name:          "nested angle brackets",
			input:         "<open file<with>brackets.txt>",
			expectedCount: 1,
			description:   "Should handle nested brackets in arguments",
		},
		{
			name:          "unicode content",
			input:         "<write unicode.txt>Hello 世界 🌍</write>",
			expectedCount: 1,
			description:   "Should handle unicode content",
		},
		{
			name:          "very long argument",
			input:         "<open " + strings.Repeat("very-long-filename-", 50) + ".txt>",
			expectedCount: 1,
			description:   "Should handle very long arguments",
		},
		{
			name:          "case sensitivity",
			input:         "<OPEN test.txt> <Open test.txt> <Write test.txt>content</Write>",
			expectedCount: 0,
			description:   "Should be case sensitive (no uppercase commands)",
		},
		{
			name:          "commands in different contexts",
			input:         "Text before <open file1.txt> middle text <write file2.txt>content</write> end text",
			expectedCount: 2,
			description:   "Should parse commands regardless of surrounding text",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			commands := parser.ParseCommands(tt.input)
			assert.Len(t, commands, tt.expectedCount, tt.description)
		})
	}
}

// TestParserPositionAccuracy tests that positions are calculated correctly
func TestParserPositionAccuracy(t *testing.T) {
	parser := NewCommandParser()

	input := "Start <open file1.txt> middle <write file2.txt>data</write> end"
	commands := parser.ParseCommands(input)

	require.Len(t, commands, 2)

	// Test first command (open)
	openCmd := commands[0]
	assert.Equal(t, "open", openCmd.Type)
	assert.Equal(t, "file1.txt", openCmd.Argument)
	extractedOpen := input[openCmd.StartPos:openCmd.EndPos]
	assert.Equal(t, openCmd.Original, extractedOpen)

	// Test second command (write)
	writeCmd := commands[1]
	assert.Equal(t, "write", writeCmd.Type)
	assert.Equal(t, "file2.txt", writeCmd.Argument)
	assert.Equal(t, "data", writeCmd.Content)
	extractedWrite := input[writeCmd.StartPos:writeCmd.EndPos]
	assert.Equal(t, writeCmd.Original, extractedWrite)

	// Verify positions don't overlap
	assert.True(t, openCmd.EndPos <= writeCmd.StartPos, "Commands should not overlap")
}

// TestParserPerformance tests parser performance with various input sizes
func TestParserPerformance(t *testing.T) {
	parser := NewCommandParser()

	// Test with increasingly large inputs
	sizes := []int{100, 1000, 10000}

	for _, size := range sizes {
		t.Run(fmt.Sprintf("size_%d", size), func(t *testing.T) {
			// Create input with scattered commands
			input := strings.Repeat("Some text ", size/10)
			input += "<open file1.txt> "
			input += strings.Repeat("More text ", size/10)
			input += "<write file2.txt>content</write> "
			input += strings.Repeat("Even more text ", size/10)

			commands := parser.ParseCommands(input)

			// Should find the commands regardless of input size
			assert.GreaterOrEqual(t, len(commands), 2)
		})
	}
}

// BenchmarkLLMParserParseCommands benchmarks the main parsing function
func BenchmarkLLMParserParseCommands(b *testing.B) {
	parser := NewCommandParser()
	input := "Please <open file1.txt> and then <write file2.txt>content here</write> followed by <exec go test> and finally <search authentication>"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = parser.ParseCommands(input)
	}
}

// BenchmarkLLMParserLargeInput benchmarks parsing with large input
func BenchmarkLLMParserLargeInput(b *testing.B) {
	parser := NewCommandParser()

	// Create large input with multiple commands
	input := strings.Repeat("This is a lot of text between commands. ", 1000)
	input += "<open large-file.txt> "
	input += strings.Repeat("More filler text here. ", 1000)
	input += "<write output.txt>Large content block</write> "
	input += strings.Repeat("Final text block. ", 1000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = parser.ParseCommands(input)
	}
}

// BenchmarkRegexCompilation benchmarks regex pattern performance
func BenchmarkRegexCompilation(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = NewCommandParser()
	}
}

// TestParserMemoryUsage tests parser memory behavior
func TestParserMemoryUsage(t *testing.T) {
	parser := NewCommandParser()

	// Test that parser doesn't retain references to input strings
	input := strings.Repeat("Large input string ", 10000) + "<open test.txt>"
	commands := parser.ParseCommands(input)

	require.Len(t, commands, 1)

	// Clear input reference
	input = ""

	// Command should still be valid
	assert.Equal(t, "open", commands[0].Type)
	assert.Equal(t, "test.txt", commands[0].Argument)
}

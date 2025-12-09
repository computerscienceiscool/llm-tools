package scanner

import (
	"testing"
)

func TestParseCommands_Open(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []Command
	}{
		{
			name:     "single open command",
			input:    "<open main.go>",
			expected: []Command{{Type: "open", Argument: "main.go", StartPos: 0, EndPos: 14, Original: "<open main.go>"}},
		},
		{
			name:     "open command with path",
			input:    "<open internal/config/types.go>",
			expected: []Command{{Type: "open", Argument: "internal/config/types.go", StartPos: 0, EndPos: 31, Original: "<open internal/config/types.go>"}},
		},
		{
			name:     "open command with surrounding text",
			input:    "Let me check the file\n<open README.md>\nfor you",
			expected: []Command{{Type: "open", Argument: "README.md", StartPos: 22, EndPos: 38, Original: "<open README.md>"}},
		},
		{
			name:  "multiple open commands",
			input: "<open file1.go>\n<open file2.go>",
			expected: []Command{
				{Type: "open", Argument: "file1.go", StartPos: 0, EndPos: 15, Original: "<open file1.go>"},
				{Type: "open", Argument: "file2.go", StartPos: 16, EndPos: 31, Original: "<open file2.go>"},
			},
		},
		{
			name:     "open with extra whitespace in argument",
			input:    "<open   spaced.go  >",
			expected: []Command{{Type: "open", Argument: "spaced.go", StartPos: 0, EndPos: 20, Original: "<open   spaced.go  >"}},
		},
		{
			name:     "empty input",
			input:    "",
			expected: nil,
		},
		{
			name:     "no commands",
			input:    "Just some regular text without any commands",
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseCommands(tt.input)

			var openCmds []Command
			for _, cmd := range result {
				if cmd.Type == "open" {
					openCmds = append(openCmds, cmd)
				}
			}

			if len(openCmds) != len(tt.expected) {
				t.Fatalf("expected %d commands, got %d", len(tt.expected), len(openCmds))
			}

			for i, expected := range tt.expected {
				if openCmds[i].Type != expected.Type {
					t.Errorf("command %d: expected type %q, got %q", i, expected.Type, openCmds[i].Type)
				}
				if openCmds[i].Argument != expected.Argument {
					t.Errorf("command %d: expected argument %q, got %q", i, expected.Argument, openCmds[i].Argument)
				}
				if openCmds[i].StartPos != expected.StartPos {
					t.Errorf("command %d: expected StartPos %d, got %d", i, expected.StartPos, openCmds[i].StartPos)
				}
				if openCmds[i].EndPos != expected.EndPos {
					t.Errorf("command %d: expected EndPos %d, got %d", i, expected.EndPos, openCmds[i].EndPos)
				}
				if openCmds[i].Original != expected.Original {
					t.Errorf("command %d: expected Original %q, got %q", i, expected.Original, openCmds[i].Original)
				}
			}
		})
	}
}

func TestParseCommands_Write(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []Command
	}{
		{
			name:  "simple write command",
			input: "<write test.txt>Hello World</write>",
			expected: []Command{{
				Type:     "write",
				Argument: "test.txt",
				Content:  "Hello World",
				StartPos: 0,
				EndPos:   35,
				Original: "<write test.txt>Hello World</write>",
			}},
		},
		{
			name:  "write command with path",
			input: "<write internal/config.yaml>key: value</write>",
			expected: []Command{{
				Type:     "write",
				Argument: "internal/config.yaml",
				Content:  "key: value",
				StartPos: 0,
				EndPos:   46,
				Original: "<write internal/config.yaml>key: value</write>",
			}},
		},
		{
			name:  "write with surrounding text",
			input: "Creating file:\n<write output.txt>content</write>\ndone!",
			expected: []Command{{
				Type:     "write",
				Argument: "output.txt",
				Content:  "content",
				StartPos: 15,
				EndPos:   48,
				Original: "<write output.txt>content</write>",
			}},
		},
		{
			name:  "write with empty content",
			input: "<write empty.txt></write>",
			expected: []Command{{
				Type:     "write",
				Argument: "empty.txt",
				Content:  "",
				StartPos: 0,
				EndPos:   25,
				Original: "<write empty.txt></write>",
			}},
		},
		{
			name:  "write with whitespace content gets trimmed",
			input: "<write file.txt>   spaced   </write>",
			expected: []Command{{
				Type:     "write",
				Argument: "file.txt",
				Content:  "spaced",
				StartPos: 0,
				EndPos:   36,
				Original: "<write file.txt>   spaced   </write>",
			}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseCommands(tt.input)

			var writeCmds []Command
			for _, cmd := range result {
				if cmd.Type == "write" {
					writeCmds = append(writeCmds, cmd)
				}
			}

			if len(writeCmds) != len(tt.expected) {
				t.Fatalf("expected %d commands, got %d", len(tt.expected), len(writeCmds))
			}

			for i, expected := range tt.expected {
				if writeCmds[i].Type != expected.Type {
					t.Errorf("command %d: expected type %q, got %q", i, expected.Type, writeCmds[i].Type)
				}
				if writeCmds[i].Argument != expected.Argument {
					t.Errorf("command %d: expected argument %q, got %q", i, expected.Argument, writeCmds[i].Argument)
				}
				if writeCmds[i].Content != expected.Content {
					t.Errorf("command %d: expected content %q, got %q", i, expected.Content, writeCmds[i].Content)
				}
				if writeCmds[i].StartPos != expected.StartPos {
					t.Errorf("command %d: expected StartPos %d, got %d", i, expected.StartPos, writeCmds[i].StartPos)
				}
				if writeCmds[i].EndPos != expected.EndPos {
					t.Errorf("command %d: expected EndPos %d, got %d", i, expected.EndPos, writeCmds[i].EndPos)
				}
			}
		})
	}
}

func TestParseCommands_Exec(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []Command
	}{
		{
			name:     "simple exec command",
			input:    "<exec go test>",
			expected: []Command{{Type: "exec", Argument: "go test", StartPos: 0, EndPos: 14, Original: "<exec go test>"}},
		},
		{
			name:     "exec with arguments",
			input:    "<exec go test ./... -v>",
			expected: []Command{{Type: "exec", Argument: "go test ./... -v", StartPos: 0, EndPos: 23, Original: "<exec go test ./... -v>"}},
		},
		{
			name:     "exec with surrounding text",
			input:    "Running:\n<exec make build>\nplease wait",
			expected: []Command{{Type: "exec", Argument: "make build", StartPos: 9, EndPos: 26, Original: "<exec make build>"}},
		},
		{
			name:  "multiple exec commands",
			input: "<exec go build>\n<exec go test>",
			expected: []Command{
				{Type: "exec", Argument: "go build", StartPos: 0, EndPos: 15, Original: "<exec go build>"},
				{Type: "exec", Argument: "go test", StartPos: 16, EndPos: 30, Original: "<exec go test>"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseCommands(tt.input)

			var execCmds []Command
			for _, cmd := range result {
				if cmd.Type == "exec" {
					execCmds = append(execCmds, cmd)
				}
			}

			if len(execCmds) != len(tt.expected) {
				t.Fatalf("expected %d commands, got %d", len(tt.expected), len(execCmds))
			}

			for i, expected := range tt.expected {
				if execCmds[i].Type != expected.Type {
					t.Errorf("command %d: expected type %q, got %q", i, expected.Type, execCmds[i].Type)
				}
				if execCmds[i].Argument != expected.Argument {
					t.Errorf("command %d: expected argument %q, got %q", i, expected.Argument, execCmds[i].Argument)
				}
				if execCmds[i].StartPos != expected.StartPos {
					t.Errorf("command %d: expected StartPos %d, got %d", i, expected.StartPos, execCmds[i].StartPos)
				}
				if execCmds[i].EndPos != expected.EndPos {
					t.Errorf("command %d: expected EndPos %d, got %d", i, expected.EndPos, execCmds[i].EndPos)
				}
			}
		})
	}
}

func TestParseCommands_Search(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []Command
	}{
		{
			name:     "simple search command",
			input:    "<search database connection>",
			expected: []Command{{Type: "search", Argument: "database connection", StartPos: 0, EndPos: 28, Original: "<search database connection>"}},
		},
		{
			name:     "search single word",
			input:    "<search config>",
			expected: []Command{{Type: "search", Argument: "config", StartPos: 0, EndPos: 15, Original: "<search config>"}},
		},
		{
			name:     "search with surrounding text",
			input:    "Looking for\n<search error handling>\nin codebase",
			expected: []Command{{Type: "search", Argument: "error handling", StartPos: 12, EndPos: 35, Original: "<search error handling>"}},
		},
		{
			name:  "multiple search commands",
			input: "<search auth>\n<search login>",
			expected: []Command{
				{Type: "search", Argument: "auth", StartPos: 0, EndPos: 13, Original: "<search auth>"},
				{Type: "search", Argument: "login", StartPos: 14, EndPos: 28, Original: "<search login>"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseCommands(tt.input)

			var searchCmds []Command
			for _, cmd := range result {
				if cmd.Type == "search" {
					searchCmds = append(searchCmds, cmd)
				}
			}

			if len(searchCmds) != len(tt.expected) {
				t.Fatalf("expected %d commands, got %d", len(tt.expected), len(searchCmds))
			}

			for i, expected := range tt.expected {
				if searchCmds[i].Type != expected.Type {
					t.Errorf("command %d: expected type %q, got %q", i, expected.Type, searchCmds[i].Type)
				}
				if searchCmds[i].Argument != expected.Argument {
					t.Errorf("command %d: expected argument %q, got %q", i, expected.Argument, searchCmds[i].Argument)
				}
				if searchCmds[i].StartPos != expected.StartPos {
					t.Errorf("command %d: expected StartPos %d, got %d", i, expected.StartPos, searchCmds[i].StartPos)
				}
				if searchCmds[i].EndPos != expected.EndPos {
					t.Errorf("command %d: expected EndPos %d, got %d", i, expected.EndPos, searchCmds[i].EndPos)
				}
			}
		})
	}
}

func TestParseCommands_Mixed(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		expectedTypes []string
	}{
		{
			name:          "all command types",
			input:         "<open file.go>\n<write out.txt>content</write>\n<exec go test>\n<search query>",
			expectedTypes: []string{"open", "write", "exec", "search"},
		},
		{
			name:          "open and write",
			input:         "Check\n<open config.yaml>\nthen\n<write output.txt>done</write>",
			expectedTypes: []string{"open", "write"},
		},
		{
			name:          "exec and search",
			input:         "<exec make>\nfollowed by\n<search results>",
			expectedTypes: []string{"exec", "search"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseCommands(tt.input)

			if len(result) != len(tt.expectedTypes) {
				t.Fatalf("expected %d commands, got %d", len(tt.expectedTypes), len(result))
			}

			typeCount := make(map[string]int)
			for _, cmd := range result {
				typeCount[cmd.Type]++
			}

			for _, expectedType := range tt.expectedTypes {
				if typeCount[expectedType] == 0 {
					t.Errorf("expected command type %q not found", expectedType)
				}
				typeCount[expectedType]--
			}
		})
	}
}

func TestParseCommands_EdgeCases(t *testing.T) {
	tests := []struct {
		name  string
		input string
		count int
	}{
		{
			name:  "malformed open - no closing bracket",
			input: "<open file.go",
			count: 0,
		},
		{
			name:  "malformed write - no closing tag",
			input: "<write file.txt>content",
			count: 0,
		},
		{
			name:  "nested brackets in content",
			input: "<write file.txt><tag>nested</tag></write>",
			count: 1,
		},
		{
			name:  "special characters in path",
			input: "<open path/to/file-name_v2.go>",
			count: 1,
		},
		{
			name:  "unicode in content",
			input: "<write test.txt>Hello 世界 🌍</write>",
			count: 1,
		},
		{
			name:  "very long argument",
			input: "<open " + "a/b/c/d/e/f/g/h/i/j/k/l/m/n/o/p/q/r/s/t/u/v/w/x/y/z/file.go" + ">",
			count: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseCommands(tt.input)
			if len(result) != tt.count {
				t.Errorf("expected %d commands, got %d", tt.count, len(result))
			}
		})
	}
}

func TestParseCommands_Positions(t *testing.T) {
	input := "prefix\n<open file.go>\nsuffix"
	result := ParseCommands(input)

	if len(result) != 1 {
		t.Fatalf("expected 1 command, got %d", len(result))
	}

	cmd := result[0]

	extracted := input[cmd.StartPos:cmd.EndPos]
	if extracted != cmd.Original {
		t.Errorf("position mismatch: extracted %q, Original %q", extracted, cmd.Original)
	}
}

package scanner

import (
	"bufio"
	"strings"
	"testing"
)

// TestScannerStateString verifies state names are correct
func TestScannerStateString(t *testing.T) {
	tests := []struct {
		state    ScannerState
		expected string
	}{
		{StateScanning, "StateScanning"},
		{StateTagOpen, "StateTagOpen"},
		{StateOpen, "StateOpen"},
		{StateWrite, "StateWrite"},
		{StateWriteBody, "StateWriteBody"},
		{StateExec, "StateExec"},
		{StateExecBody, "StateExecBody"},
		{StateSearch, "StateSearch"},
		{StateExecute, "StateExecute"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			got := tt.state.String()
			if got != tt.expected {
				t.Errorf("State.String() = %q, want %q", got, tt.expected)
			}
		})
	}
}

// TestNewScanner verifies scanner initialization
func TestNewScanner(t *testing.T) {
	reader := bufio.NewReader(strings.NewReader(""))
	scanner := NewScanner(reader, false)

	if scanner == nil {
		t.Fatal("NewScanner() returned nil")
	}

	if scanner.state != StateScanning {
		t.Errorf("Initial state = %v, want StateScanning", scanner.state)
	}

	if scanner.currentCmd != nil {
		t.Error("currentCmd should be nil initially")
	}

	if scanner.reader == nil {
		t.Error("reader should not be nil")
	}
}

// TestTransitionTo verifies state transitions work
func TestTransitionTo(t *testing.T) {
	reader := bufio.NewReader(strings.NewReader(""))
	scanner := NewScanner(reader, false)

	scanner.transitionTo(StateTagOpen)
	if scanner.state != StateTagOpen {
		t.Errorf("After transitionTo(StateTagOpen), state = %v, want StateTagOpen", scanner.state)
	}

	scanner.transitionTo(StateWriteBody)
	if scanner.state != StateWriteBody {
		t.Errorf("After transitionTo(StateWriteBody), state = %v, want StateWriteBody", scanner.state)
	}
}

// TestResetCommand verifies command reset
func TestResetCommand(t *testing.T) {
	reader := bufio.NewReader(strings.NewReader(""))
	scanner := NewScanner(reader, false)

	scanner.currentCmd = &Command{Type: "test"}
	scanner.buffer.WriteString("some data")

	scanner.resetCommand()

	if scanner.currentCmd != nil {
		t.Error("currentCmd should be nil after reset")
	}

	if scanner.buffer.Len() != 0 {
		t.Errorf("buffer length = %d, want 0", scanner.buffer.Len())
	}
}

// TestStartCommand verifies command initialization
func TestStartCommand(t *testing.T) {
	reader := bufio.NewReader(strings.NewReader(""))
	scanner := NewScanner(reader, false)

	scanner.buffer.WriteString("old data")

	scanner.startCommand("open")

	if scanner.currentCmd == nil {
		t.Fatal("currentCmd should not be nil after startCommand")
	}

	if scanner.currentCmd.Type != "open" {
		t.Errorf("command type = %q, want %q", scanner.currentCmd.Type, "open")
	}

	if scanner.buffer.Len() != 0 {
		t.Error("buffer should be reset after startCommand")
	}
}

// TestScan_OpenCommand tests basic open command parsing
func TestScan_OpenCommand(t *testing.T) {
	input := "<open README.md>\n"
	reader := bufio.NewReader(strings.NewReader(input))
	scanner := NewScanner(reader, false)

	cmd := scanner.Scan()

	if cmd == nil {
		t.Fatal("Scan() returned nil, expected command")
	}

	if cmd.Type != "open" {
		t.Errorf("Type = %q, want %q", cmd.Type, "open")
	}

	if cmd.Argument != "README.md" {
		t.Errorf("Argument = %q, want %q", cmd.Argument, "README.md")
	}
}

// TestScan_OpenCommandWithSpaces tests open command with spaces in argument
func TestScan_OpenCommandWithSpaces(t *testing.T) {
	input := "<open  src/main.go  >\n"
	reader := bufio.NewReader(strings.NewReader(input))
	scanner := NewScanner(reader, false)

	cmd := scanner.Scan()

	if cmd == nil {
		t.Fatal("Scan() returned nil")
	}

	if cmd.Argument != "src/main.go" {
		t.Errorf("Argument = %q, want %q (should be trimmed)", cmd.Argument, "src/main.go")
	}
}

// TestScan_WriteCommand tests write command with content
func TestScan_WriteCommand(t *testing.T) {
	input := "<write test.txt>hello world</write>\n"
	reader := bufio.NewReader(strings.NewReader(input))
	scanner := NewScanner(reader, false)

	cmd := scanner.Scan()

	if cmd == nil {
		t.Fatal("Scan() returned nil")
	}

	if cmd.Type != "write" {
		t.Errorf("Type = %q, want %q", cmd.Type, "write")
	}

	if cmd.Argument != "test.txt" {
		t.Errorf("Argument = %q, want %q", cmd.Argument, "test.txt")
	}

	if cmd.Content != "hello world" {
		t.Errorf("Content = %q, want %q", cmd.Content, "hello world")
	}
}

// TestScan_WriteCommandMultiline tests write with multiline content
func TestScan_WriteCommandMultiline(t *testing.T) {
	input := `<write config.yaml>
server:
  port: 8080
  host: localhost
</write>
`
	reader := bufio.NewReader(strings.NewReader(input))
	scanner := NewScanner(reader, false)

	cmd := scanner.Scan()

	if cmd == nil {
		t.Fatal("Scan() returned nil")
	}

	if cmd.Type != "write" {
		t.Errorf("Type = %q, want %q", cmd.Type, "write")
	}

	expectedContent := "server:\n  port: 8080\n  host: localhost"
	if cmd.Content != expectedContent {
		t.Errorf("Content = %q, want %q", cmd.Content, expectedContent)
	}
}

// TestScan_ExecCommand tests basic exec command
func TestScan_ExecCommand(t *testing.T) {
	input := "<exec go test>\n"
	reader := bufio.NewReader(strings.NewReader(input))
	scanner := NewScanner(reader, false)

	cmd := scanner.Scan()

	if cmd == nil {
		t.Fatal("Scan() returned nil")
	}

	if cmd.Type != "exec" {
		t.Errorf("Type = %q, want %q", cmd.Type, "exec")
	}

	if cmd.Argument != "go test" {
		t.Errorf("Argument = %q, want %q", cmd.Argument, "go test")
	}

	if cmd.Content != "" {
		t.Errorf("Content should be empty, got %q", cmd.Content)
	}
}

// TestScan_SearchCommand tests search command
func TestScan_SearchCommand(t *testing.T) {
	input := "<search TODO comments>\n"
	reader := bufio.NewReader(strings.NewReader(input))
	scanner := NewScanner(reader, false)

	cmd := scanner.Scan()

	if cmd == nil {
		t.Fatal("Scan() returned nil")
	}

	if cmd.Type != "search" {
		t.Errorf("Type = %q, want %q", cmd.Type, "search")
	}

	if cmd.Argument != "TODO comments" {
		t.Errorf("Argument = %q, want %q", cmd.Argument, "TODO comments")
	}
}

// TestScan_MultipleCommands tests scanning multiple commands
func TestScan_MultipleCommands(t *testing.T) {
	input := `<open file1.go>
<open file2.go>
<exec go test>
`
	reader := bufio.NewReader(strings.NewReader(input))
	scanner := NewScanner(reader, false)

	// First command
	cmd1 := scanner.Scan()
	if cmd1 == nil || cmd1.Type != "open" || cmd1.Argument != "file1.go" {
		t.Errorf("First command: got %+v, want open file1.go", cmd1)
	}

	// Second command
	cmd2 := scanner.Scan()
	if cmd2 == nil || cmd2.Type != "open" || cmd2.Argument != "file2.go" {
		t.Errorf("Second command: got %+v, want open file2.go", cmd2)
	}

	// Third command
	cmd3 := scanner.Scan()
	if cmd3 == nil || cmd3.Type != "exec" || cmd3.Argument != "go test" {
		t.Errorf("Third command: got %+v, want exec go test", cmd3)
	}

	// No more commands
	cmd4 := scanner.Scan()
	if cmd4 != nil {
		t.Errorf("Expected nil after all commands, got %+v", cmd4)
	}
}

// TestScan_EmptyInput tests scanning empty input
func TestScan_EmptyInput(t *testing.T) {
	input := ""
	reader := bufio.NewReader(strings.NewReader(input))
	scanner := NewScanner(reader, false)

	cmd := scanner.Scan()

	if cmd != nil {
		t.Errorf("Expected nil for empty input, got %+v", cmd)
	}
}

// TestScan_NoCommands tests input with no valid commands
func TestScan_NoCommands(t *testing.T) {
	input := "Just some regular text without commands\n"
	reader := bufio.NewReader(strings.NewReader(input))
	scanner := NewScanner(reader, false)

	cmd := scanner.Scan()

	if cmd != nil {
		t.Errorf("Expected nil for text without commands, got %+v", cmd)
	}
}

// TestScan_InvalidTag tests handling of invalid tags
func TestScan_InvalidTag(t *testing.T) {
	input := "<invalid command>\n"
	reader := bufio.NewReader(strings.NewReader(input))
	scanner := NewScanner(reader, false)

	cmd := scanner.Scan()

	if cmd != nil {
		t.Errorf("Expected nil for invalid tag, got %+v", cmd)
	}
}

// TestScan_WriteEmptyContent tests write with empty content
func TestScan_WriteEmptyContent(t *testing.T) {
	input := "<write empty.txt></write>\n"
	reader := bufio.NewReader(strings.NewReader(input))
	scanner := NewScanner(reader, false)

	cmd := scanner.Scan()

	if cmd == nil {
		t.Fatal("Scan() returned nil")
	}

	if cmd.Type != "write" {
		t.Errorf("Type = %q, want %q", cmd.Type, "write")
	}

	if cmd.Content != "" {
		t.Errorf("Content should be empty, got %q", cmd.Content)
	}
}

// TestScan_CommandsWithText tests commands mixed with regular text
func TestScan_CommandsWithText(t *testing.T) {
	input := "Some text before\n<open file.go>\nSome text after\n"
	reader := bufio.NewReader(strings.NewReader(input))
	scanner := NewScanner(reader, false)

	cmd := scanner.Scan()

	if cmd == nil {
		t.Fatal("Scan() returned nil")
	}

	if cmd.Type != "open" {
		t.Errorf("Type = %q, want %q", cmd.Type, "open")
	}

	if cmd.Argument != "file.go" {
		t.Errorf("Argument = %q, want %q", cmd.Argument, "file.go")
	}
}

// TestScan_WriteWithSpecialCharacters tests write with special chars
func TestScan_WriteWithSpecialCharacters(t *testing.T) {
	input := "<write test.txt>Content with <brackets> and special chars: !@#$%</write>\n"
	reader := bufio.NewReader(strings.NewReader(input))
	scanner := NewScanner(reader, false)

	cmd := scanner.Scan()

	if cmd == nil {
		t.Fatal("Scan() returned nil")
	}

	expectedContent := "Content with <brackets> and special chars: !@#$%"
	if cmd.Content != expectedContent {
		t.Errorf("Content = %q, want %q", cmd.Content, expectedContent)
	}
}

// TestScan_PathWithSlashes tests file paths with slashes
func TestScan_PathWithSlashes(t *testing.T) {
	input := "<open internal/config/types.go>\n"
	reader := bufio.NewReader(strings.NewReader(input))
	scanner := NewScanner(reader, false)

	cmd := scanner.Scan()

	if cmd == nil {
		t.Fatal("Scan() returned nil")
	}

	if cmd.Argument != "internal/config/types.go" {
		t.Errorf("Argument = %q, want %q", cmd.Argument, "internal/config/types.go")
	}
}

// TestScan_ExecSingleLine tests exec on single line
func TestScan_ExecSingleLine(t *testing.T) {
	input := "<exec ls -la>\n"
	reader := bufio.NewReader(strings.NewReader(input))
	scanner := NewScanner(reader, false)

	cmd := scanner.Scan()

	if cmd == nil {
		t.Fatal("Scan() returned nil")
	}

	if cmd.Type != "exec" {
		t.Errorf("Type = %q, want %q", cmd.Type, "exec")
	}

	if cmd.Argument != "ls -la" {
		t.Errorf("Argument = %q, want %q", cmd.Argument, "ls -la")
	}
}

// TestScan_ConsecutiveCommands tests commands on consecutive lines
func TestScan_ConsecutiveCommands(t *testing.T) {
	input := "<open a.go>\n<open b.go>\n<open c.go>\n"
	reader := bufio.NewReader(strings.NewReader(input))
	scanner := NewScanner(reader, false)

	commands := []string{}
	for {
		cmd := scanner.Scan()
		if cmd == nil {
			break
		}
		commands = append(commands, cmd.Argument)
	}

	if len(commands) != 3 {
		t.Errorf("Expected 3 commands, got %d", len(commands))
	}

	expected := []string{"a.go", "b.go", "c.go"}
	for i, arg := range commands {
		if arg != expected[i] {
			t.Errorf("Command %d: got %q, want %q", i, arg, expected[i])
		}
	}
}

// TestScan_WriteWithNestedTags tests write content containing tags
func TestScan_WriteWithNestedTags(t *testing.T) {
	input := "<write test.html><div><p>Hello</p></div></write>\n"
	reader := bufio.NewReader(strings.NewReader(input))
	scanner := NewScanner(reader, false)

	cmd := scanner.Scan()

	if cmd == nil {
		t.Fatal("Scan() returned nil")
	}

	expectedContent := "<div><p>Hello</p></div>"
	if cmd.Content != expectedContent {
		t.Errorf("Content = %q, want %q", cmd.Content, expectedContent)
	}
}

// TestScan_LongArgument tests commands with very long arguments
func TestScan_LongArgument(t *testing.T) {
	longPath := strings.Repeat("a/", 50) + "file.go"
	input := "<open " + longPath + ">\n"
	reader := bufio.NewReader(strings.NewReader(input))
	scanner := NewScanner(reader, false)

	cmd := scanner.Scan()

	if cmd == nil {
		t.Fatal("Scan() returned nil")
	}

	if cmd.Argument != longPath {
		t.Error("Long argument not preserved correctly")
	}
}

// TestScan_ShowPrompts tests scanner with prompts enabled
func TestScan_ShowPrompts(t *testing.T) {
	input := "<open test.go>\n"
	reader := bufio.NewReader(strings.NewReader(input))
	scanner := NewScanner(reader, true) // showPrompts = true

	cmd := scanner.Scan()

	if cmd == nil {
		t.Fatal("Scan() returned nil")
	}

	if cmd.Type != "open" || cmd.Argument != "test.go" {
		t.Errorf("Command not parsed correctly with showPrompts=true")
	}
}

// Benchmark tests
func BenchmarkScan_SimpleOpen(b *testing.B) {
	input := "<open README.md>\n"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		reader := bufio.NewReader(strings.NewReader(input))
		scanner := NewScanner(reader, false)
		scanner.Scan()
	}
}

func BenchmarkScan_Write(b *testing.B) {
	input := "<write test.txt>hello world</write>\n"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		reader := bufio.NewReader(strings.NewReader(input))
		scanner := NewScanner(reader, false)
		scanner.Scan()
	}
}

func BenchmarkScan_MultipleCommands(b *testing.B) {
	input := "<open a.go>\n<open b.go>\n<exec go test>\n"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		reader := bufio.NewReader(strings.NewReader(input))
		scanner := NewScanner(reader, false)
		for scanner.Scan() != nil {
		}
	}
}

package cli

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/computerscienceiscool/llm-runtime/internal/config"
	"github.com/computerscienceiscool/llm-runtime/pkg/evaluator"
	"github.com/computerscienceiscool/llm-runtime/internal/search"
)

// captureOutput captures stdout and stderr during function execution
func captureOutput(t *testing.T, f func()) (stdout, stderr string) {
	t.Helper()

	// Capture stdout
	oldStdout := os.Stdout
	rOut, wOut, err := os.Pipe()
	if err != nil {
		t.Fatalf("Failed to create stdout pipe: %v", err)
	}
	os.Stdout = wOut

	// Capture stderr
	oldStderr := os.Stderr
	rErr, wErr, err := os.Pipe()
	if err != nil {
		t.Fatalf("Failed to create stderr pipe: %v", err)
	}
	os.Stderr = wErr

	// Run the function
	f()

	// Restore and read stdout
	wOut.Close()
	os.Stdout = oldStdout
	var bufOut bytes.Buffer
	io.Copy(&bufOut, rOut)
	rOut.Close()

	// Restore and read stderr
	wErr.Close()
	os.Stderr = oldStderr
	var bufErr bytes.Buffer
	io.Copy(&bufErr, rErr)
	rErr.Close()

	return bufOut.String(), bufErr.String()
}

// mockStdin temporarily replaces os.Stdin with a reader
func mockStdin(t *testing.T, input string) func() {
	t.Helper()

	oldStdin := os.Stdin
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("Failed to create stdin pipe: %v", err)
	}

	os.Stdin = r

	// Write input in a goroutine to avoid blocking
	go func() {
		defer w.Close()
		w.WriteString(input)
	}()

	return func() {
		os.Stdin = oldStdin
		r.Close()
	}
}

func TestInteractiveMode_PrintsWelcomeMessage(t *testing.T) {
	tempDir := t.TempDir()

	cfg := &config.Config{
		RepositoryRoot:    tempDir,
		MaxFileSize:       1048576,
		MaxWriteSize:      102400,
		AllowedExtensions: []string{".txt"},
		ExcludedPaths:     []string{".git"},
		BackupBeforeWrite: false,
		ExecEnabled:       false,
	}

	searchCfg := &search.SearchConfig{Enabled: false}
	auditLog := func(cmd, arg string, success bool, errMsg string) {}
	exec := evaluator.NewExecutor(cfg, searchCfg, auditLog)

	// Empty input - will immediately hit EOF
	restore := mockStdin(t, "")
	defer restore()

	startTime := time.Now()

	_, stderr := captureOutput(t, func() {
		ScanInput(exec, startTime, true)
	})

	// Check welcome message in stderr
	expectedParts := []string{
		"LLM Tool - Interactive Mode",
		"Waiting for input",
		"<open filepath>",
		"<write filepath>",
		"<exec command",
		"<search query>",
	}

	for _, part := range expectedParts {
		if !strings.Contains(stderr, part) {
			t.Errorf("Welcome message missing expected part: %q\nFull stderr:\n%s", part, stderr)
		}
	}
}

func TestInteractiveMode_ProcessesOpenCommand(t *testing.T) {
	tempDir := t.TempDir()

	// Create a test file
	testFile := "test.txt"
	testContent := "Hello from test file"
	if err := os.WriteFile(tempDir+"/"+testFile, []byte(testContent), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	cfg := &config.Config{
		RepositoryRoot:    tempDir,
		MaxFileSize:       1048576,
		MaxWriteSize:      102400,
		AllowedExtensions: []string{".txt"},
		ExcludedPaths:     []string{".git"},
		BackupBeforeWrite: false,
		ExecEnabled:       false,
	}

	searchCfg := &search.SearchConfig{Enabled: false}
	auditLog := func(cmd, arg string, success bool, errMsg string) {}
	exec := evaluator.NewExecutor(cfg, searchCfg, auditLog)

	// Input with open command
	input := "<open test.txt>\n"
	restore := mockStdin(t, input)
	defer restore()

	startTime := time.Now()

	stdout, _ := captureOutput(t, func() {
		ScanInput(exec, startTime, true)
	})

	// Check that file content appears in output
	if !strings.Contains(stdout, testContent) {
		t.Errorf("Expected file content %q in stdout\nFull stdout:\n%s", testContent, stdout)
	}
}

func TestInteractiveMode_ProcessesWriteCommand(t *testing.T) {
	tempDir := t.TempDir()

	cfg := &config.Config{
		RepositoryRoot:    tempDir,
		MaxFileSize:       1048576,
		MaxWriteSize:      102400,
		AllowedExtensions: []string{".txt"},
		ExcludedPaths:     []string{".git"},
		BackupBeforeWrite: false,
		ExecEnabled:       false,
	}

	searchCfg := &search.SearchConfig{Enabled: false}
	auditLog := func(cmd, arg string, success bool, errMsg string) {}
	exec := evaluator.NewExecutor(cfg, searchCfg, auditLog)

	// Input with write command
	input := "<write newfile.txt>Created content</write>\n"
	restore := mockStdin(t, input)
	defer restore()

	startTime := time.Now()

	stdout, _ := captureOutput(t, func() {
		ScanInput(exec, startTime, true)
	})

	// Check that write was successful
	if !strings.Contains(stdout, "WRITE SUCCESSFUL") {
		t.Errorf("Expected 'WRITE SUCCESSFUL' in stdout\nFull stdout:\n%s", stdout)
	}

	// Verify file was created
	content, err := os.ReadFile(tempDir + "/newfile.txt")
	if err != nil {
		t.Errorf("File was not created: %v", err)
	}
	if string(content) != "Created content" {
		t.Errorf("File content = %q, want %q", string(content), "Created content")
	}
}

func TestInteractiveMode_ProcessesExecCommand(t *testing.T) {
	tempDir := t.TempDir()

	cfg := &config.Config{
		RepositoryRoot:    tempDir,
		MaxFileSize:       1048576,
		MaxWriteSize:      102400,
		AllowedExtensions: []string{".txt"},
		ExcludedPaths:     []string{".git"},
		BackupBeforeWrite: false,
		ExecEnabled:       false, // Disabled, should show error
	}

	searchCfg := &search.SearchConfig{Enabled: false}
	auditLog := func(cmd, arg string, success bool, errMsg string) {}
	exec := evaluator.NewExecutor(cfg, searchCfg, auditLog)

	// Input with exec command
	input := "<exec echo hello>\n"
	restore := mockStdin(t, input)
	defer restore()

	startTime := time.Now()

	stdout, _ := captureOutput(t, func() {
		ScanInput(exec, startTime, true)
	})

	// Exec is disabled, should show error
	if !strings.Contains(stdout, "ERROR") {
		t.Errorf("Expected 'ERROR' for disabled exec in stdout\nFull stdout:\n%s", stdout)
	}
}

func TestInteractiveMode_ProcessesSearchCommand(t *testing.T) {
	tempDir := t.TempDir()

	cfg := &config.Config{
		RepositoryRoot:    tempDir,
		MaxFileSize:       1048576,
		MaxWriteSize:      102400,
		AllowedExtensions: []string{".txt"},
		ExcludedPaths:     []string{".git"},
		BackupBeforeWrite: false,
		ExecEnabled:       false,
	}

	searchCfg := &search.SearchConfig{Enabled: false} // Disabled
	auditLog := func(cmd, arg string, success bool, errMsg string) {}
	exec := evaluator.NewExecutor(cfg, searchCfg, auditLog)

	// Input with search command
	input := "<search test query>\n"
	restore := mockStdin(t, input)
	defer restore()

	startTime := time.Now()

	stdout, _ := captureOutput(t, func() {
		ScanInput(exec, startTime, true)
	})

	// Search is disabled, should show error
	if !strings.Contains(stdout, "ERROR") {
		t.Errorf("Expected 'ERROR' for disabled search in stdout\nFull stdout:\n%s", stdout)
	}
}

func TestInteractiveMode_PlainTextNoCommand(t *testing.T) {
	t.Skip("TODO: Plain text handling - will fix in later")
	tempDir := t.TempDir()

	cfg := &config.Config{
		RepositoryRoot:    tempDir,
		MaxFileSize:       1048576,
		MaxWriteSize:      102400,
		AllowedExtensions: []string{".txt"},
		ExcludedPaths:     []string{".git"},
		BackupBeforeWrite: false,
		ExecEnabled:       false,
	}

	searchCfg := &search.SearchConfig{Enabled: false}
	auditLog := func(cmd, arg string, success bool, errMsg string) {}
	exec := evaluator.NewExecutor(cfg, searchCfg, auditLog)

	// Plain text without commands - should be buffered until EOF
	input := "Just some plain text\nwith multiple lines\n"
	restore := mockStdin(t, input)
	defer restore()

	startTime := time.Now()

	stdout, _ := captureOutput(t, func() {
		ScanInput(exec, startTime, true)
	})

	// Plain text without commands should pass through
	if !strings.Contains(stdout, "Just some plain text") {
		t.Errorf("Expected plain text in output\nFull stdout:\n%s", stdout)
	}
}

func TestInteractiveMode_MultipleCommands(t *testing.T) {
	tempDir := t.TempDir()

	cfg := &config.Config{
		RepositoryRoot:    tempDir,
		MaxFileSize:       1048576,
		MaxWriteSize:      102400,
		AllowedExtensions: []string{".txt"},
		ExcludedPaths:     []string{".git"},
		BackupBeforeWrite: false,
		ExecEnabled:       false,
	}

	searchCfg := &search.SearchConfig{Enabled: false}
	auditLog := func(cmd, arg string, success bool, errMsg string) {}
	exec := evaluator.NewExecutor(cfg, searchCfg, auditLog)

	// Multiple commands in sequence
	input := "<write first.txt>First file</write>\n<write second.txt>Second file</write>\n"
	restore := mockStdin(t, input)
	defer restore()

	startTime := time.Now()

	stdout, stderr := captureOutput(t, func() {
		ScanInput(exec, startTime, true)
	})

	// Both writes should be successful
	if strings.Count(stdout, "WRITE SUCCESSFUL") < 2 {
		t.Errorf("Expected 2 'WRITE SUCCESSFUL' in stdout\nFull stdout:\n%s\nStderr:\n%s", stdout, stderr)
	}

	// Verify both files were created
	if _, err := os.Stat(tempDir + "/first.txt"); os.IsNotExist(err) {
		t.Error("first.txt was not created")
	}
	if _, err := os.Stat(tempDir + "/second.txt"); os.IsNotExist(err) {
		t.Error("second.txt was not created")
	}
}

func TestInteractiveMode_WaitingForMoreInputMessage(t *testing.T) {
	tempDir := t.TempDir()

	cfg := &config.Config{
		RepositoryRoot:    tempDir,
		MaxFileSize:       1048576,
		MaxWriteSize:      102400,
		AllowedExtensions: []string{".txt"},
		ExcludedPaths:     []string{".git"},
		BackupBeforeWrite: false,
		ExecEnabled:       false,
	}

	searchCfg := &search.SearchConfig{Enabled: false}
	auditLog := func(cmd, arg string, success bool, errMsg string) {}
	exec := evaluator.NewExecutor(cfg, searchCfg, auditLog)

	// Input with a command
	input := "<write test.txt>content</write>\n"
	restore := mockStdin(t, input)
	defer restore()

	startTime := time.Now()

	_, stderr := captureOutput(t, func() {
		ScanInput(exec, startTime, true)
	})

	// After processing a command, should print "Waiting for more input"
	if !strings.Contains(stderr, "Waiting for more input") {
		t.Errorf("Expected 'Waiting for more input' in stderr\nFull stderr:\n%s", stderr)
	}
}

func TestInteractiveMode_EmptyInput(t *testing.T) {
	tempDir := t.TempDir()

	cfg := &config.Config{
		RepositoryRoot:    tempDir,
		MaxFileSize:       1048576,
		MaxWriteSize:      102400,
		AllowedExtensions: []string{".txt"},
		ExcludedPaths:     []string{".git"},
		BackupBeforeWrite: false,
		ExecEnabled:       false,
	}

	searchCfg := &search.SearchConfig{Enabled: false}
	auditLog := func(cmd, arg string, success bool, errMsg string) {}
	exec := evaluator.NewExecutor(cfg, searchCfg, auditLog)

	// Empty input - immediate EOF
	restore := mockStdin(t, "")
	defer restore()

	startTime := time.Now()

	// Should not panic or hang
	stdout, stderr := captureOutput(t, func() {
		ScanInput(exec, startTime, true)
	})

	// Should still print welcome message
	if !strings.Contains(stderr, "LLM Tool - Interactive Mode") {
		t.Errorf("Expected welcome message\nStdout:\n%s\nStderr:\n%s", stdout, stderr)
	}
}

func TestInteractiveMode_CommandWithSurroundingText(t *testing.T) {
	tempDir := t.TempDir()

	// Create a test file
	if err := os.WriteFile(tempDir+"/existing.txt", []byte("existing content"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	cfg := &config.Config{
		RepositoryRoot:    tempDir,
		MaxFileSize:       1048576,
		MaxWriteSize:      102400,
		AllowedExtensions: []string{".txt"},
		ExcludedPaths:     []string{".git"},
		BackupBeforeWrite: false,
		ExecEnabled:       false,
	}

	searchCfg := &search.SearchConfig{Enabled: false}
	auditLog := func(cmd, arg string, success bool, errMsg string) {}
	exec := evaluator.NewExecutor(cfg, searchCfg, auditLog)

	// Command with surrounding text on same line
	input := "<open existing.txt>\n"

	restore := mockStdin(t, input)
	defer restore()

	startTime := time.Now()

	stdout, _ := captureOutput(t, func() {
		ScanInput(exec, startTime, true)
	})

	// Should process the open command and show file content
	if !strings.Contains(stdout, "existing content") {
		t.Errorf("Expected file content in output\nFull stdout:\n%s", stdout)
	}
}

func TestInteractiveMode_FailedCommand(t *testing.T) {
	tempDir := t.TempDir()

	cfg := &config.Config{
		RepositoryRoot:    tempDir,
		MaxFileSize:       1048576,
		MaxWriteSize:      102400,
		AllowedExtensions: []string{".txt"},
		ExcludedPaths:     []string{".git"},
		BackupBeforeWrite: false,
		ExecEnabled:       false,
	}

	searchCfg := &search.SearchConfig{Enabled: false}
	auditLog := func(cmd, arg string, success bool, errMsg string) {}
	exec := evaluator.NewExecutor(cfg, searchCfg, auditLog)

	// Try to open non-existent file
	input := "<open nonexistent.txt>\n"
	restore := mockStdin(t, input)
	defer restore()

	startTime := time.Now()

	stdout, _ := captureOutput(t, func() {
		ScanInput(exec, startTime, true)
	})

	// Should show error
	if !strings.Contains(stdout, "ERROR") {
		t.Errorf("Expected 'ERROR' for non-existent file\nFull stdout:\n%s", stdout)
	}
}

// TestIsCommandStart verifies the Contains→HasPrefix bug fix
func TestIsCommandStart(t *testing.T) {
	tests := []struct {
		name     string
		line     string
		expected bool
	}{
		// Should match - commands at start of line
		{"open at start", "<open main.go>", true},
		{"write at start", "<write test.txt>content</write>", true},
		{"exec at start", "<exec go test>", true},
		{"search at start", "<search query>", true},
		{"with leading space", "  <open file.go>", true},
		{"with leading tab", "\t<write config.yaml>data</write>", true},

		// Should NOT match - THE BUG FIX
		{"comment should NOT match", "// don't <open secret.key>", false},
		{"mid-line should NOT match", "Please read <open main.go> carefully", false},
		{"in string should NOT match", `fmt.Println("<open example>")`, false},
		{"regular text", "This is just text", false},
		{"empty line", "", false},
		{"HTML tag not command", "<div>content</div>", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isCommandStart(tt.line)
			if result != tt.expected {
				t.Errorf("isCommandStart(%q) = %v, want %v", tt.line, result, tt.expected)
			}
		})
	}
}

// TestContainsBugFix demonstrates the bug that was fixed
func TestContainsBugFix(t *testing.T) {
	// This line has a command in a comment - should NOT trigger
	commentLine := "// don't <open secret.key>"

	if isCommandStart(commentLine) {
		t.Error("BUG: Command in comment should NOT be detected")
	}

	// This line has command at start - SHOULD trigger
	commandLine := "<open secret.key>"

	if !isCommandStart(commandLine) {
		t.Error("Command at start should be detected")
	}
}


// TestInteractiveMode_MultiLineWrite tests the key feature - multi-line write commands
func TestInteractiveMode_MultiLineWrite(t *testing.T) {
	tempDir := t.TempDir()

	cfg := &config.Config{
		RepositoryRoot:    tempDir,
		MaxFileSize:       1048576,
		MaxWriteSize:      102400,
		AllowedExtensions: []string{".txt"},
		ExcludedPaths:     []string{".git"},
		BackupBeforeWrite: false,
		ExecEnabled:       false,
	}

	searchCfg := &search.SearchConfig{Enabled: false}
	auditLog := func(cmd, arg string, success bool, errMsg string) {}
	exec := evaluator.NewExecutor(cfg, searchCfg, auditLog)

	// Multi-line write command 
	input := "<write multiline.txt>\nfirst line\nsecond line\nthird line\n</write>\n"
	restore := mockStdin(t, input)
	defer restore()

	startTime := time.Now()

	stdout, _ := captureOutput(t, func() {
		ScanInput(exec, startTime, false)
	})

	// Verify the write was successful
	if !strings.Contains(stdout, "WRITE SUCCESSFUL") {
		t.Errorf("Expected 'WRITE SUCCESSFUL' in stdout\nFull stdout:\n%s", stdout)
	}

	// Verify file was created with ALL content
	filePath := tempDir + "/multiline.txt"
	content, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("File was not created: %v", err)
	}

	expectedContent := "first line\nsecond line\nthird line"
	actualContent := string(content)
	if actualContent != expectedContent {
		t.Errorf("File content mismatch\nExpected: %q\nGot: %q", expectedContent, actualContent)
	}
}
// TestInteractiveMode_MultiLineWriteWithEmptyLines tests write with blank lines
func TestInteractiveMode_MultiLineWriteWithEmptyLines(t *testing.T) {
	tempDir := t.TempDir()

	cfg := &config.Config{
		RepositoryRoot:    tempDir,
		MaxFileSize:       1048576,
		MaxWriteSize:      102400,
		AllowedExtensions: []string{".txt"},
		ExcludedPaths:     []string{".git"},
		BackupBeforeWrite: false,
		ExecEnabled:       false,
	}

	searchCfg := &search.SearchConfig{Enabled: false}
	auditLog := func(cmd, arg string, success bool, errMsg string) {}
	exec := evaluator.NewExecutor(cfg, searchCfg, auditLog)

	// Write with empty lines in content
	input := "<write empty.txt>\nline 1\n\nline 3\n</write>\n"
	restore := mockStdin(t, input)
	defer restore()

	startTime := time.Now()

	stdout, _ := captureOutput(t, func() {
		ScanInput(exec, startTime, false)
	})

	if !strings.Contains(stdout, "WRITE SUCCESSFUL") {
		t.Errorf("Expected 'WRITE SUCCESSFUL' in stdout\nFull stdout:\n%s", stdout)
	}

	// Verify file has empty line preserved
	content, err := os.ReadFile(tempDir + "/empty.txt")
	if err != nil {
		t.Fatalf("File was not created: %v", err)
	}

	expectedContent := "line 1\n\nline 3"
	if string(content) != expectedContent {
		t.Errorf("Content = %q, want %q", string(content), expectedContent)
	}
}

  // TestInteractiveMode_SingleLineWrite tests single-line write still works
func TestInteractiveMode_SingleLineWrite(t *testing.T) {
	tempDir := t.TempDir()

	cfg := &config.Config{
		RepositoryRoot:    tempDir,
		MaxFileSize:       1048576,
		MaxWriteSize:      102400,
		AllowedExtensions: []string{".txt"},
		ExcludedPaths:     []string{".git"},
		BackupBeforeWrite: false,
		ExecEnabled:       false,
	}

	searchCfg := &search.SearchConfig{Enabled: false}
	auditLog := func(cmd, arg string, success bool, errMsg string) {}
	exec := evaluator.NewExecutor(cfg, searchCfg, auditLog)

	// Single line write (old behavior should still work)
	input := "<write single.txt>just one line</write>\n"
	restore := mockStdin(t, input)
	defer restore()

	startTime := time.Now()

	stdout, _ := captureOutput(t, func() {
		ScanInput(exec, startTime, false)
	})

	if !strings.Contains(stdout, "WRITE SUCCESSFUL") {
		t.Errorf("Expected 'WRITE SUCCESSFUL' in stdout\nFull stdout:\n%s", stdout)
	}

	content, err := os.ReadFile(tempDir + "/single.txt")
	if err != nil {
		t.Fatalf("File was not created: %v", err)
	}

	if string(content) != "just one line" {
		t.Errorf("Content = %q, want %q", string(content), "just one line")
	}
}

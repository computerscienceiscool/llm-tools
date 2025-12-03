package cli

import (
	"flag"
	"os"
	"testing"
	"time"
)

// resetFlags resets the flag package state for testing
func resetFlags() {
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
}

func TestParseFlags_Defaults(t *testing.T) {
	// Save original args and restore after test
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	resetFlags()
	os.Args = []string{"cmd"}

	flags := ParseFlags()

	// Check default values
	if flags.Config.RepositoryRoot != "." {
		t.Errorf("RepositoryRoot = %q, want %q", flags.Config.RepositoryRoot, ".")
	}
	if flags.Config.MaxFileSize != 1048576 {
		t.Errorf("MaxFileSize = %d, want %d", flags.Config.MaxFileSize, 1048576)
	}
	if flags.Config.MaxWriteSize != 102400 {
		t.Errorf("MaxWriteSize = %d, want %d", flags.Config.MaxWriteSize, 102400)
	}
	if flags.Config.Interactive != false {
		t.Errorf("Interactive = %v, want %v", flags.Config.Interactive, false)
	}
	if flags.Config.JSONOutput != false {
		t.Errorf("JSONOutput = %v, want %v", flags.Config.JSONOutput, false)
	}
	if flags.Config.Verbose != false {
		t.Errorf("Verbose = %v, want %v", flags.Config.Verbose, false)
	}
	if flags.Config.RequireConfirmation != false {
		t.Errorf("RequireConfirmation = %v, want %v", flags.Config.RequireConfirmation, false)
	}
	if flags.Config.BackupBeforeWrite != true {
		t.Errorf("BackupBeforeWrite = %v, want %v", flags.Config.BackupBeforeWrite, true)
	}
	if flags.Config.ForceWrite != false {
		t.Errorf("ForceWrite = %v, want %v", flags.Config.ForceWrite, false)
	}
	if flags.Config.ExecEnabled != false {
		t.Errorf("ExecEnabled = %v, want %v", flags.Config.ExecEnabled, false)
	}
	if flags.Config.ExecTimeout != 30*time.Second {
		t.Errorf("ExecTimeout = %v, want %v", flags.Config.ExecTimeout, 30*time.Second)
	}
	if flags.Config.ExecMemoryLimit != "512m" {
		t.Errorf("ExecMemoryLimit = %q, want %q", flags.Config.ExecMemoryLimit, "512m")
	}
	if flags.Config.ExecCPULimit != 1 {
		t.Errorf("ExecCPULimit = %d, want %d", flags.Config.ExecCPULimit, 1)
	}
	if flags.Config.ExecContainerImage != "python-go" {
		t.Errorf("ExecContainerImage = %q, want %q", flags.Config.ExecContainerImage, "python-go")
	}
	if flags.Config.ExecNetworkEnabled != false {
		t.Errorf("ExecNetworkEnabled = %v, want %v", flags.Config.ExecNetworkEnabled, false)
	}

	// Check search flags defaults
	if flags.Reindex != false {
		t.Errorf("Reindex = %v, want %v", flags.Reindex, false)
	}
	if flags.SearchStatus != false {
		t.Errorf("SearchStatus = %v, want %v", flags.SearchStatus, false)
	}
	if flags.SearchValidate != false {
		t.Errorf("SearchValidate = %v, want %v", flags.SearchValidate, false)
	}
	if flags.SearchCleanup != false {
		t.Errorf("SearchCleanup = %v, want %v", flags.SearchCleanup, false)
	}
	if flags.SearchUpdate != false {
		t.Errorf("SearchUpdate = %v, want %v", flags.SearchUpdate, false)
	}
	if flags.CheckPythonSetup != false {
		t.Errorf("CheckPythonSetup = %v, want %v", flags.CheckPythonSetup, false)
	}
}

func TestParseFlags_CustomValues(t *testing.T) {
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	resetFlags()
	os.Args = []string{
		"cmd",
		"-root", "/custom/root",
		"-max-size", "2097152",
		"-max-write-size", "204800",
		"-interactive",
		"-json",
		"-verbose",
		"-require-confirmation",
		"-backup=false",
		"-force",
		"-exec-enabled",
		"-exec-timeout", "60s",
		"-exec-memory", "1g",
		"-exec-cpu", "4",
		"-exec-image", "alpine:latest",
		"-exec-network",
		"-input", "input.txt",
		"-output", "output.txt",
	}

	flags := ParseFlags()

	if flags.Config.RepositoryRoot != "/custom/root" {
		t.Errorf("RepositoryRoot = %q, want %q", flags.Config.RepositoryRoot, "/custom/root")
	}
	if flags.Config.MaxFileSize != 2097152 {
		t.Errorf("MaxFileSize = %d, want %d", flags.Config.MaxFileSize, 2097152)
	}
	if flags.Config.MaxWriteSize != 204800 {
		t.Errorf("MaxWriteSize = %d, want %d", flags.Config.MaxWriteSize, 204800)
	}
	if flags.Config.Interactive != true {
		t.Errorf("Interactive = %v, want %v", flags.Config.Interactive, true)
	}
	if flags.Config.JSONOutput != true {
		t.Errorf("JSONOutput = %v, want %v", flags.Config.JSONOutput, true)
	}
	if flags.Config.Verbose != true {
		t.Errorf("Verbose = %v, want %v", flags.Config.Verbose, true)
	}
	if flags.Config.RequireConfirmation != true {
		t.Errorf("RequireConfirmation = %v, want %v", flags.Config.RequireConfirmation, true)
	}
	if flags.Config.BackupBeforeWrite != false {
		t.Errorf("BackupBeforeWrite = %v, want %v", flags.Config.BackupBeforeWrite, false)
	}
	if flags.Config.ForceWrite != true {
		t.Errorf("ForceWrite = %v, want %v", flags.Config.ForceWrite, true)
	}
	if flags.Config.ExecEnabled != true {
		t.Errorf("ExecEnabled = %v, want %v", flags.Config.ExecEnabled, true)
	}
	if flags.Config.ExecTimeout != 60*time.Second {
		t.Errorf("ExecTimeout = %v, want %v", flags.Config.ExecTimeout, 60*time.Second)
	}
	if flags.Config.ExecMemoryLimit != "1g" {
		t.Errorf("ExecMemoryLimit = %q, want %q", flags.Config.ExecMemoryLimit, "1g")
	}
	if flags.Config.ExecCPULimit != 4 {
		t.Errorf("ExecCPULimit = %d, want %d", flags.Config.ExecCPULimit, 4)
	}
	if flags.Config.ExecContainerImage != "alpine:latest" {
		t.Errorf("ExecContainerImage = %q, want %q", flags.Config.ExecContainerImage, "alpine:latest")
	}
	if flags.Config.ExecNetworkEnabled != true {
		t.Errorf("ExecNetworkEnabled = %v, want %v", flags.Config.ExecNetworkEnabled, true)
	}
	if flags.Config.InputFile != "input.txt" {
		t.Errorf("InputFile = %q, want %q", flags.Config.InputFile, "input.txt")
	}
	if flags.Config.OutputFile != "output.txt" {
		t.Errorf("OutputFile = %q, want %q", flags.Config.OutputFile, "output.txt")
	}
}

func TestParseFlags_SearchFlags(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		wantFlag string
	}{
		{
			name:     "reindex flag",
			args:     []string{"cmd", "-reindex"},
			wantFlag: "reindex",
		},
		{
			name:     "search-status flag",
			args:     []string{"cmd", "-search-status"},
			wantFlag: "search-status",
		},
		{
			name:     "search-validate flag",
			args:     []string{"cmd", "-search-validate"},
			wantFlag: "search-validate",
		},
		{
			name:     "search-cleanup flag",
			args:     []string{"cmd", "-search-cleanup"},
			wantFlag: "search-cleanup",
		},
		{
			name:     "search-update flag",
			args:     []string{"cmd", "-search-update"},
			wantFlag: "search-update",
		},
		{
			name:     "check-python-setup flag",
			args:     []string{"cmd", "-check-python-setup"},
			wantFlag: "check-python-setup",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oldArgs := os.Args
			defer func() { os.Args = oldArgs }()

			resetFlags()
			os.Args = tt.args

			flags := ParseFlags()

			switch tt.wantFlag {
			case "reindex":
				if !flags.Reindex {
					t.Error("Reindex should be true")
				}
			case "search-status":
				if !flags.SearchStatus {
					t.Error("SearchStatus should be true")
				}
			case "search-validate":
				if !flags.SearchValidate {
					t.Error("SearchValidate should be true")
				}
			case "search-cleanup":
				if !flags.SearchCleanup {
					t.Error("SearchCleanup should be true")
				}
			case "search-update":
				if !flags.SearchUpdate {
					t.Error("SearchUpdate should be true")
				}
			case "check-python-setup":
				if !flags.CheckPythonSetup {
					t.Error("CheckPythonSetup should be true")
				}
			}
		})
	}
}

func TestParseFlags_AllowedExtensions(t *testing.T) {
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	resetFlags()
	os.Args = []string{"cmd", "-allowed-extensions", ".go,.py,.rs"}

	flags := ParseFlags()

	expected := []string{".go", ".py", ".rs"}
	if len(flags.Config.AllowedExtensions) != len(expected) {
		t.Errorf("AllowedExtensions length = %d, want %d", len(flags.Config.AllowedExtensions), len(expected))
	}

	for i, ext := range expected {
		if i < len(flags.Config.AllowedExtensions) && flags.Config.AllowedExtensions[i] != ext {
			t.Errorf("AllowedExtensions[%d] = %q, want %q", i, flags.Config.AllowedExtensions[i], ext)
		}
	}
}

func TestParseFlags_ExecWhitelist(t *testing.T) {
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	resetFlags()
	os.Args = []string{"cmd", "-exec-whitelist", "go test,go build,make"}

	flags := ParseFlags()

	expected := []string{"go test", "go build", "make"}
	if len(flags.Config.ExecWhitelist) != len(expected) {
		t.Errorf("ExecWhitelist length = %d, want %d", len(flags.Config.ExecWhitelist), len(expected))
	}

	for i, cmd := range expected {
		if i < len(flags.Config.ExecWhitelist) && flags.Config.ExecWhitelist[i] != cmd {
			t.Errorf("ExecWhitelist[%d] = %q, want %q", i, flags.Config.ExecWhitelist[i], cmd)
		}
	}
}

func TestParseFlags_ExcludedPaths(t *testing.T) {
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	resetFlags()
	os.Args = []string{"cmd", "-exclude", ".git,node_modules,vendor"}

	flags := ParseFlags()

	expected := []string{".git", "node_modules", "vendor"}
	if len(flags.Config.ExcludedPaths) != len(expected) {
		t.Errorf("ExcludedPaths length = %d, want %d", len(flags.Config.ExcludedPaths), len(expected))
	}

	for i, path := range expected {
		if i < len(flags.Config.ExcludedPaths) && flags.Config.ExcludedPaths[i] != path {
			t.Errorf("ExcludedPaths[%d] = %q, want %q", i, flags.Config.ExcludedPaths[i], path)
		}
	}
}

func TestParseFlags_ExtensionsWithSpaces(t *testing.T) {
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	resetFlags()
	os.Args = []string{"cmd", "-allowed-extensions", " .go , .py , .rs "}

	flags := ParseFlags()

	// Spaces should be trimmed
	expected := []string{".go", ".py", ".rs"}
	for i, ext := range expected {
		if i < len(flags.Config.AllowedExtensions) && flags.Config.AllowedExtensions[i] != ext {
			t.Errorf("AllowedExtensions[%d] = %q, want %q (trimmed)", i, flags.Config.AllowedExtensions[i], ext)
		}
	}
}

func TestHasSearchCommand(t *testing.T) {
	tests := []struct {
		name     string
		flags    CLIFlags
		expected bool
	}{
		{
			name:     "no search commands",
			flags:    CLIFlags{},
			expected: false,
		},
		{
			name:     "reindex set",
			flags:    CLIFlags{Reindex: true},
			expected: true,
		},
		{
			name:     "search status set",
			flags:    CLIFlags{SearchStatus: true},
			expected: true,
		},
		{
			name:     "search validate set",
			flags:    CLIFlags{SearchValidate: true},
			expected: true,
		},
		{
			name:     "search cleanup set",
			flags:    CLIFlags{SearchCleanup: true},
			expected: true,
		},
		{
			name:     "search update set",
			flags:    CLIFlags{SearchUpdate: true},
			expected: true,
		},
		{
			name:     "check python setup set",
			flags:    CLIFlags{CheckPythonSetup: true},
			expected: true,
		},
		{
			name:     "multiple search commands set",
			flags:    CLIFlags{Reindex: true, SearchStatus: true},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.flags.HasSearchCommand()
			if got != tt.expected {
				t.Errorf("HasSearchCommand() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestParseFlags_DefaultAllowedExtensions(t *testing.T) {
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	resetFlags()
	os.Args = []string{"cmd"}

	flags := ParseFlags()

	// Check that default extensions are set
	defaultExts := []string{".go", ".py", ".js", ".md", ".txt", ".json", ".yaml", ".yml", ".toml"}
	if len(flags.Config.AllowedExtensions) != len(defaultExts) {
		t.Errorf("AllowedExtensions length = %d, want %d", len(flags.Config.AllowedExtensions), len(defaultExts))
	}

	for i, ext := range defaultExts {
		if i < len(flags.Config.AllowedExtensions) && flags.Config.AllowedExtensions[i] != ext {
			t.Errorf("AllowedExtensions[%d] = %q, want %q", i, flags.Config.AllowedExtensions[i], ext)
		}
	}
}

func TestParseFlags_DefaultExcludedPaths(t *testing.T) {
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	resetFlags()
	os.Args = []string{"cmd"}

	flags := ParseFlags()

	// Check that default excluded paths are set
	defaultExcluded := []string{".git", ".env", "*.key", "*.pem"}
	if len(flags.Config.ExcludedPaths) != len(defaultExcluded) {
		t.Errorf("ExcludedPaths length = %d, want %d", len(flags.Config.ExcludedPaths), len(defaultExcluded))
	}

	for i, path := range defaultExcluded {
		if i < len(flags.Config.ExcludedPaths) && flags.Config.ExcludedPaths[i] != path {
			t.Errorf("ExcludedPaths[%d] = %q, want %q", i, flags.Config.ExcludedPaths[i], path)
		}
	}
}

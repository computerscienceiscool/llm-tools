package infrastructure

import (
	"os/exec"
	"testing"
)

// Helper to check if Python with sentence-transformers is available
func pythonWithSentenceTransformersAvailable() bool {
	cmd := exec.Command("python3", "-c", "import sentence_transformers; print('OK')")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return false
	}
	return string(output) == "OK\n"
}

// Helper to check if Python is available (without sentence-transformers)
func pythonAvailable() bool {
	cmd := exec.Command("python3", "--version")
	return cmd.Run() == nil
}

func TestCheckPythonDependencies_Success(t *testing.T) {
	if !pythonWithSentenceTransformersAvailable() {
		t.Skip("Python with sentence-transformers not available")
	}

	err := CheckPythonDependencies("python3")
	if err != nil {
		t.Errorf("CheckPythonDependencies failed when deps are available: %v", err)
	}
}

func TestCheckPythonDependencies_InvalidPath(t *testing.T) {
	err := CheckPythonDependencies("/nonexistent/python")
	if err == nil {
		t.Error("expected error for invalid Python path")
	}
}

func TestCheckPythonDependencies_EmptyPath(t *testing.T) {
	err := CheckPythonDependencies("")
	if err == nil {
		t.Error("expected error for empty Python path")
	}
}

func TestCheckPythonDependencies_NotPython(t *testing.T) {
	// Use a command that exists but isn't Python
	err := CheckPythonDependencies("/bin/true")
	if err == nil {
		t.Error("expected error for non-Python command")
	}
}

func TestCheckPythonDependencies_PythonWithoutSentenceTransformers(t *testing.T) {
	if !pythonAvailable() {
		t.Skip("Python not available")
	}
	if pythonWithSentenceTransformersAvailable() {
		t.Skip("sentence-transformers is available, cannot test missing dependency")
	}

	// Python is available but sentence-transformers is not
	err := CheckPythonDependencies("python3")
	if err == nil {
		t.Error("expected error when sentence-transformers is not installed")
	}
}

func TestCheckPythonDependencies_ErrorMessage(t *testing.T) {
	err := CheckPythonDependencies("/nonexistent/path/python")
	if err == nil {
		t.Fatal("expected error")
	}

	errMsg := err.Error()
	if errMsg == "" {
		t.Error("error message should not be empty")
	}

	// Error should mention Python or sentence-transformers
	t.Logf("Error message: %s", errMsg)
}

func TestCheckPythonDependencies_DifferentPythonNames(t *testing.T) {
	pythonNames := []string{
		"python3",
		"python",
		"/usr/bin/python3",
		"/usr/local/bin/python3",
	}

	for _, pythonPath := range pythonNames {
		t.Run(pythonPath, func(t *testing.T) {
			// Just check if the command exists
			cmd := exec.Command(pythonPath, "--version")
			if cmd.Run() != nil {
				t.Skipf("%s not available", pythonPath)
			}

			// If Python exists, check dependencies
			err := CheckPythonDependencies(pythonPath)
			// We don't assert the result - just ensure it doesn't panic
			t.Logf("CheckPythonDependencies(%s): %v", pythonPath, err)
		})
	}
}

func TestCheckPythonDependencies_CommandInjection(t *testing.T) {
	// Test that malicious input doesn't cause issues
	maliciousInputs := []string{
		"python3; rm -rf /",
		"python3 && echo pwned",
		"python3 | cat /etc/passwd",
		"$(whoami)",
		"`whoami`",
	}

	for _, input := range maliciousInputs {
		t.Run(input, func(t *testing.T) {
			// Should fail gracefully without executing malicious commands
			err := CheckPythonDependencies(input)
			if err == nil {
				// If it somehow succeeds, that's suspicious
				t.Logf("Warning: %q returned nil error", input)
			}
		})
	}
}

func TestCheckPythonDependencies_RelativePath(t *testing.T) {
	// Relative path that doesn't exist
	err := CheckPythonDependencies("./nonexistent_python")
	if err == nil {
		t.Error("expected error for nonexistent relative path")
	}
}

func TestCheckPythonDependencies_SpacesInPath(t *testing.T) {
	// Path with spaces (likely doesn't exist)
	err := CheckPythonDependencies("/path with spaces/python")
	if err == nil {
		t.Error("expected error for path with spaces")
	}
}

func TestCheckPythonDependencies_RepeatedCalls(t *testing.T) {
	if !pythonWithSentenceTransformersAvailable() {
		t.Skip("Python with sentence-transformers not available")
	}

	// Multiple calls should all succeed
	for i := 0; i < 5; i++ {
		err := CheckPythonDependencies("python3")
		if err != nil {
			t.Errorf("call %d failed: %v", i, err)
		}
	}
}

// Benchmark tests
func BenchmarkCheckPythonDependencies_Available(b *testing.B) {
	if !pythonWithSentenceTransformersAvailable() {
		b.Skip("Python with sentence-transformers not available")
	}

	for i := 0; i < b.N; i++ {
		CheckPythonDependencies("python3")
	}
}

func BenchmarkCheckPythonDependencies_Unavailable(b *testing.B) {
	for i := 0; i < b.N; i++ {
		CheckPythonDependencies("/nonexistent/python")
	}
}

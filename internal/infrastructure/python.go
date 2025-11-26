package infrastructure

import (
	"fmt"
	"os/exec"
	"strings"
)

// CheckPythonDependencies verifies that Python and required packages are available
func CheckPythonDependencies(pythonPath string) error {
	// Test Python availability
	cmd := exec.Command(pythonPath, "-c", "import sentence_transformers; print('OK')")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("Python or sentence-transformers not available: %w\nOutput: %s", err, output)
	}

	if !strings.Contains(string(output), "OK") {
		return fmt.Errorf("sentence-transformers import failed")
	}

	return nil
}

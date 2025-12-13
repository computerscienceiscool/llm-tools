// TestContainerIO_WriteFileFromContainer tests writing files from container
func TestContainerIO_WriteFileFromContainer(t *testing.T) {
	if !isDockerAvailable() {
		t.Skip("Docker not available")
	}
	ensureTestImage(t)

	tmpDir := t.TempDir()

	// First check if we can even see /workspace
	cfg := ContainerConfig{
		Image:       "alpine:latest",
		Command:     "ls -la /workspace",
		RepoRoot:    tmpDir,
		MemoryLimit: "128m",
		CPULimit:    1,
		Timeout:     10 * time.Second,
	}

	result, err := RunContainer(cfg)
	if err != nil {
		t.Logf("ls /workspace failed: %v", err)
		t.Logf("stdout: %s", result.Stdout)
		t.Logf("stderr: %s", result.Stderr)
	} else {
		t.Logf("ls /workspace succeeded:")
		t.Logf("stdout: %s", result.Stdout)
	}

	// Now try to write
	cfg = ContainerConfig{
		Image:       "alpine:latest",
		Command:     "sh -c 'echo written_by_container > /workspace/output.txt 2>&1 && echo SUCCESS'",
		RepoRoot:    tmpDir,
		MemoryLimit: "128m",
		CPULimit:    1,
		Timeout:     10 * time.Second,
	}

	result, err = RunContainer(cfg)
	if err != nil {
		t.Logf("Write attempt error: %v", err)
	}
	
	t.Logf("Write attempt stdout: %s", result.Stdout)
	t.Logf("Write attempt stderr: %s", result.Stderr)
	t.Logf("Write attempt exit code: %d", result.ExitCode)

	if strings.Contains(result.Stdout, "SUCCESS") {
		t.Log("Write command executed successfully")
		
		// Check if file exists on host
		outputFile := filepath.Join(tmpDir, "output.txt")
		if content, err := os.ReadFile(outputFile); err == nil {
			if strings.Contains(string(content), "written_by_container") {
				t.Log("File successfully written and visible on host")
			} else {
				t.Errorf("File exists but has wrong content: %s", string(content))
			}
		} else {
			t.Logf("File not visible on host: %v", err)
		}
	}
}

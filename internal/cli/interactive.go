package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/computerscienceiscool/llm-tools/internal/executor"
)

// InteractiveMode handles continuous input/output
func InteractiveMode(exec *executor.Executor, startTime time.Time) {
	scanner := bufio.NewScanner(os.Stdin)
	var buffer strings.Builder

	fmt.Fprintln(os.Stderr, "LLM Tool - Interactive Mode")
	fmt.Fprintln(os.Stderr, "Waiting for input (send EOF with Ctrl+D to process)...")
	fmt.Fprintln(os.Stderr, "Supports commands: <open filepath>, <write filepath>content</write>, <exec command args>, <search query>")

	for scanner.Scan() {
		line := scanner.Text()
		buffer.WriteString(line)
		buffer.WriteString("\n")

		// Check if line contains a command
		if strings.Contains(line, "<open") || strings.Contains(line, "<write") ||
			strings.Contains(line, "<exec") || strings.Contains(line, "<search") {
			// Process accumulated text
			result := ProcessText(buffer.String(), exec, startTime)
			fmt.Print(result)
			buffer.Reset()

			fmt.Fprintln(os.Stderr, "\nWaiting for more input...")
		}
	}

	// Process any remaining text
	if buffer.Len() > 0 {
		result := ProcessText(buffer.String(), exec, startTime)
		fmt.Print(result)
	}
}

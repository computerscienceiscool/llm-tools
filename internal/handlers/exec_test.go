package handlers

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestCommandExecution tests command execution structure
func TestCommandExecution(t *testing.T) {
	exec := CommandExecution{
		Command:   "go test ./...",
		ExitCode:  0,
		Stdout:    "ok\ttest\t0.123s",
		Stderr:    "",
		Duration:  2 * time.Second,
		Timestamp: time.Now(),
	}

	assert.Equal(t, "go test ./...", exec.Command)
	assert.Equal(t, 0, exec.ExitCode)
	assert.Contains(t, exec.Stdout, "ok")
	assert.Equal(t, 2*time.Second, exec.Duration)
}

// TestExecutionStats tests execution statistics
func TestExecutionStats(t *testing.T) {
	stats := ExecutionStats{
		TotalCommands:  10,
		SuccessfulRuns: 8,
		FailedRuns:     2,
		AverageRunTime: 1500 * time.Millisecond,
		TotalRunTime:   15 * time.Second,
	}

	assert.Equal(t, 10, stats.TotalCommands)
	assert.Equal(t, 8, stats.SuccessfulRuns)
	assert.Equal(t, 2, stats.FailedRuns)
	assert.Equal(t, 0.8, stats.SuccessRate())
}

// Placeholder structures for testing
type CommandExecution struct {
	Command   string
	ExitCode  int
	Stdout    string
	Stderr    string
	Duration  time.Duration
	Timestamp time.Time
}

type ExecutionStats struct {
	TotalCommands  int
	SuccessfulRuns int
	FailedRuns     int
	AverageRunTime time.Duration
	TotalRunTime   time.Duration
}

func (s ExecutionStats) SuccessRate() float64 {
	if s.TotalCommands == 0 {
		return 0
	}
	return float64(s.SuccessfulRuns) / float64(s.TotalCommands)
}

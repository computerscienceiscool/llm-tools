package scanner

import (
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
	scanner := NewScanner(nil, false)
	
	if scanner == nil {
		t.Fatal("NewScanner() returned nil")
	}
	
	if scanner.state != StateScanning {
		t.Errorf("Initial state = %v, want StateScanning", scanner.state)
	}
	
	if scanner.currentCmd != nil {
		t.Error("currentCmd should be nil initially")
	}
}

// TestTransitionTo verifies state transitions work
func TestTransitionTo(t *testing.T) {
	scanner := NewScanner(nil, false)
	
	scanner.transitionTo(StateTagOpen)
	if scanner.state != StateTagOpen {
		t.Errorf("After transitionTo(StateTagOpen), state = %v, want StateTagOpen", scanner.state)
	}
	
	scanner.transitionTo(StateWriteBody)
	if scanner.state != StateWriteBody {
		t.Errorf("After transitionTo(StateWriteBody), state = %v, want StateWriteBody", scanner.state)
	}
}

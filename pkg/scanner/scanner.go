package scanner

import (
	"bufio"
	"strings"

)

// ScannerState represents the current parsing state
type ScannerState int

const (
	StateScanning  ScannerState = iota // Default: scanning for commands or plain text
	StateTagOpen                        // Saw '<', determining tag type
	StateOpen                           // Parsing <open filepath>
	StateWrite                          // Parsing <write filepath>
	StateWriteBody                      // Accumulating write content until </write>
	StateExec                           // Parsing <exec command>
	StateSearch                         // Parsing <search query>
	StateExecute                        // Ready to execute command
)

// String returns the name of the state (for debugging)
func (s ScannerState) String() string {
	switch s {
	case StateScanning:
		return "StateScanning"
	case StateTagOpen:
		return "StateTagOpen"
	case StateOpen:
		return "StateOpen"
	case StateWrite:
		return "StateWrite"
	case StateWriteBody:
		return "StateWriteBody"
	case StateExec:
		return "StateExec"
	case StateSearch:
		return "StateSearch"
	case StateExecute:
		return "StateExecute"
	default:
		return "StateUnknown"
	}
}

// Scanner implements a state-machine based input processor
type Scanner struct {
	state      ScannerState
	buffer     strings.Builder
	currentCmd *Command
	reader     *bufio.Reader
	showPrompts bool
}

// NewScanner creates a new state-machine scanner
func NewScanner(reader *bufio.Reader, showPrompts bool) *Scanner {
	return &Scanner{
		state:       StateScanning,
		reader:      reader,
		showPrompts: showPrompts,
	}
}

// transitionTo changes state
func (s *Scanner) transitionTo(newState ScannerState) {
	s.state = newState
}

// resetCommand clears the current command and buffer
func (s *Scanner) resetCommand() {
	s.currentCmd = nil
	s.buffer.Reset()
}

// startCommand initializes a new command
func (s *Scanner) startCommand(cmdType string) {
	s.currentCmd = &Command{
		Type: cmdType,
	}
	s.buffer.Reset()
}




// Scan reads input and returns the next complete command
// Returns nil when EOF or no command found
func (s *Scanner) Scan() *Command {
	for {
		line, err := s.reader.ReadString('\n')
		if err != nil {
			// EOF - return any incomplete write command as nil
			if line == "" {
				return nil
			}
		}

		// Process the line based on current state
		for i := 0; i < len(line); i++ {
			ch := line[i]

			switch s.state {
			case StateScanning:
				if ch == '<' {
					s.transitionTo(StateTagOpen)
					s.buffer.Reset()
					s.buffer.WriteByte(ch)
				}


			case StateTagOpen:
				s.buffer.WriteByte(ch)
				buffered := s.buffer.String()
				
				// Wait until we have enough characters to determine command type
				if ch == ' ' || ch == '>' {
					if strings.HasPrefix(buffered, "<open") {
						s.startCommand("open")
						s.transitionTo(StateOpen)
						s.buffer.Reset()
					} else if strings.HasPrefix(buffered, "<write") {
						s.startCommand("write")
						s.transitionTo(StateWrite)
						s.buffer.Reset()
					} else if strings.HasPrefix(buffered, "<exec") {
						s.startCommand("exec")
						s.transitionTo(StateExec)
						s.buffer.Reset()
					} else if strings.HasPrefix(buffered, "<search") {
						s.startCommand("search")
						s.transitionTo(StateSearch)
						s.buffer.Reset()
					} else {
						// Not a valid command, go back to scanning
						s.transitionTo(StateScanning)
						s.buffer.Reset()
					}
				}


			case StateOpen:
				if ch == '>' {
					s.currentCmd.Argument = strings.TrimSpace(s.buffer.String())
					s.transitionTo(StateScanning)
					cmd := s.currentCmd
					s.resetCommand()
					return cmd
				} else {
					s.buffer.WriteByte(ch)
				}

			case StateWrite:
				if ch == '>' {
					s.currentCmd.Argument = strings.TrimSpace(s.buffer.String())
					s.transitionTo(StateWriteBody)
					s.buffer.Reset()
				} else {
					s.buffer.WriteByte(ch)
				}

			case StateWriteBody:
				// KEY STATE: accumulate everything until </write>
				s.buffer.WriteByte(ch)
				
				buffered := s.buffer.String()
				if strings.Contains(buffered, "</write>") {
					idx := strings.Index(buffered, "</write>")
					content := buffered[:idx]
					s.currentCmd.Content = strings.TrimSpace(content)
					s.transitionTo(StateScanning)
					cmd := s.currentCmd
					s.resetCommand()
					return cmd
				}

			case StateExec:
				if ch == '>' {
					s.currentCmd.Argument = strings.TrimSpace(s.buffer.String())
					s.transitionTo(StateScanning)
					cmd := s.currentCmd
					s.resetCommand()
					return cmd
				} else {
					s.buffer.WriteByte(ch)
				}

			case StateSearch:
				if ch == '>' {
					s.currentCmd.Argument = strings.TrimSpace(s.buffer.String())
					s.transitionTo(StateScanning)
					cmd := s.currentCmd
					s.resetCommand()
					return cmd
				} else {
					s.buffer.WriteByte(ch)
				}
			}
		}
	}
}

package parser

// CommandParser parses LLM output for commands
type CommandParser interface {
	ParseCommands(text string) []ParsedCommand
}

// ParsedCommand represents a parsed command
type ParsedCommand struct {
	Type     string
	Argument string
	Content  string
	StartPos int
	EndPos   int
	Original string
}

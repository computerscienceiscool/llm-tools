package parser

import (
	"regexp"
	"strings"
)

// LLMParser implements CommandParser for parsing LLM output
type LLMParser struct {
	openPattern   *regexp.Regexp
	writePattern  *regexp.Regexp
	execPattern   *regexp.Regexp
	searchPattern *regexp.Regexp
}

// NewCommandParser creates a new LLM command parser
func NewCommandParser() CommandParser {
	return &LLMParser{
		openPattern:   regexp.MustCompile(`<open\s+([^>]+)>`),
		writePattern:  regexp.MustCompile(`<write\s+([^>]+)>\s*(.*?)</write>`),
		execPattern:   regexp.MustCompile(`<exec\s+([^>]+)>`),
		searchPattern: regexp.MustCompile(`<search\s+([^>]+)>`),
	}
}

// ParseCommands extracts commands from LLM output
func (p *LLMParser) ParseCommands(text string) []ParsedCommand {
	var commands []ParsedCommand

	// Parse open commands
	commands = append(commands, p.parseOpenCommands(text)...)
	
	// Parse write commands
	commands = append(commands, p.parseWriteCommands(text)...)
	
	// Parse exec commands
	commands = append(commands, p.parseExecCommands(text)...)
	
	// Parse search commands
	commands = append(commands, p.parseSearchCommands(text)...)

	return commands
}

func (p *LLMParser) parseOpenCommands(text string) []ParsedCommand {
	var commands []ParsedCommand
	
	matches := p.openPattern.FindAllStringSubmatchIndex(text, -1)
	for _, match := range matches {
		if len(match) >= 4 {
			cmd := ParsedCommand{
				Type:     "open",
				Argument: strings.TrimSpace(text[match[2]:match[3]]),
				StartPos: match[0],
				EndPos:   match[1],
				Original: text[match[0]:match[1]],
			}
			commands = append(commands, cmd)
		}
	}
	
	return commands
}

func (p *LLMParser) parseWriteCommands(text string) []ParsedCommand {
	var commands []ParsedCommand
	
	matches := p.writePattern.FindAllStringSubmatchIndex(text, -1)
	for _, match := range matches {
		if len(match) >= 6 {
			content := strings.TrimSpace(text[match[4]:match[5]])
			cmd := ParsedCommand{
				Type:     "write",
				Argument: strings.TrimSpace(text[match[2]:match[3]]),
				Content:  content,
				StartPos: match[0],
				EndPos:   match[1],
				Original: text[match[0]:match[1]],
			}
			commands = append(commands, cmd)
		}
	}
	
	return commands
}

func (p *LLMParser) parseExecCommands(text string) []ParsedCommand {
	var commands []ParsedCommand
	
	matches := p.execPattern.FindAllStringSubmatchIndex(text, -1)
	for _, match := range matches {
		if len(match) >= 4 {
			cmd := ParsedCommand{
				Type:     "exec",
				Argument: strings.TrimSpace(text[match[2]:match[3]]),
				StartPos: match[0],
				EndPos:   match[1],
				Original: text[match[0]:match[1]],
			}
			commands = append(commands, cmd)
		}
	}
	
	return commands
}

func (p *LLMParser) parseSearchCommands(text string) []ParsedCommand {
	var commands []ParsedCommand
	
	matches := p.searchPattern.FindAllStringSubmatchIndex(text, -1)
	for _, match := range matches {
		if len(match) >= 4 {
			cmd := ParsedCommand{
				Type:     "search",
				Argument: strings.TrimSpace(text[match[2]:match[3]]),
				StartPos: match[0],
				EndPos:   match[1],
				Original: text[match[0]:match[1]],
			}
			commands = append(commands, cmd)
		}
	}
	
	return commands
}

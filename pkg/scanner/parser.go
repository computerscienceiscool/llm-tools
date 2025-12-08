package scanner

import (
	"regexp"
	"strings"
)

// ParseCommands extracts commands from LLM output
func ParseCommands(text string) []Command {
	var commands []Command

	// Pattern for <open filepath> commands
	openPattern := regexp.MustCompile(`(?m)^\s*<open\s+([^>]+)>`)

	matches := openPattern.FindAllStringSubmatchIndex(text, -1)
	for _, match := range matches {
		if len(match) >= 4 {
			cmd := Command{
				Type:     "open",
				Argument: strings.TrimSpace(text[match[2]:match[3]]),
				StartPos: match[0],
				EndPos:   match[1],
				Original: text[match[0]:match[1]],
			}
			commands = append(commands, cmd)
		}
	}

	// Pattern for <write filepath>content</write> commands
	writePattern := regexp.MustCompile(`(?ms)^\s*<write\s+([^>]+)>\s*(.*?)</write>`)

	// Find write commands
	writeMatches := writePattern.FindAllStringSubmatchIndex(text, -1)
	for _, match := range writeMatches {
		if len(match) >= 6 {
			content := strings.TrimSpace(text[match[4]:match[5]])
			cmd := Command{
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

	// Pattern for <exec command arguments> commands
	execPattern := regexp.MustCompile(`(?m)^\s*<exec\s+([^>]+)>`)

	execMatches := execPattern.FindAllStringSubmatchIndex(text, -1)
	for _, match := range execMatches {
		if len(match) >= 4 {
			cmd := Command{
				Type:     "exec",
				Argument: strings.TrimSpace(text[match[2]:match[3]]),
				StartPos: match[0],
				EndPos:   match[1],
				Original: text[match[0]:match[1]],
			}
			commands = append(commands, cmd)
		}
	}

	// Pattern for <search query terms> commands
	searchPattern := regexp.MustCompile(`(?m)^\s*<search\s+([^>]+)>`)

	searchMatches := searchPattern.FindAllStringSubmatchIndex(text, -1)
	for _, match := range searchMatches {
		if len(match) >= 4 {
			cmd := Command{
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

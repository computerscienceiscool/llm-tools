package evaluator

import (
	"fmt"
	"strings"
	"time"

	"github.com/computerscienceiscool/llm-runtime/pkg/config"
	"github.com/computerscienceiscool/llm-runtime/pkg/scanner"
	"github.com/computerscienceiscool/llm-runtime/pkg/search"
)

// ExecuteSearch handles the "search" command
func ExecuteSearch(query string, cfg *config.Config, searchCfg *search.SearchConfig, auditLog func(cmd, arg string, success bool, errMsg string)) scanner.ExecutionResult {
	startTime := time.Now()
	result := scanner.ExecutionResult{
		Command: scanner.Command{Type: "search", Argument: query},
	}

	// Check if search is enabled
	if searchCfg == nil || !searchCfg.Enabled {
		result.Success = false
		fullError := fmt.Errorf("SEARCH_DISABLED: search feature is not enabled")
		result.Error = SanitizeError(fullError) // Sanitized for LLM
		result.ExecutionTime = time.Since(startTime)
		if auditLog != nil {
			auditLog("search", query, false, fullError.Error()) // Full error to audit
		}
		return result
	}

	// Initialize search engine
	searchEngine, err := search.NewSearchEngine(searchCfg, cfg.RepositoryRoot)
	if err != nil {
		result.Success = false
		fullError := fmt.Errorf("SEARCH_INIT_FAILED: %w", err)
		result.Error = SanitizeError(fullError) // Sanitized for LLM
		result.ExecutionTime = time.Since(startTime)
		if auditLog != nil {
			auditLog("search", query, false, fullError.Error()) // Full error to audit
		}
		return result
	}
	defer searchEngine.Close()

	// Execute search
	searchResults, err := searchEngine.Search(query)
	if err != nil {
		result.Success = false
		fullError := fmt.Errorf("SEARCH_FAILED: %w", err)
		result.Error = SanitizeError(fullError) // Sanitized for LLM
		result.ExecutionTime = time.Since(startTime)
		if auditLog != nil {
			auditLog("search", query, false, fullError.Error()) // Full error to audit
		}
		return result
	}

	// Format results
	result.Success = true
	result.Result = formatSearchOutput(query, searchResults, searchCfg.MaxResults, time.Since(startTime))
	result.ExecutionTime = time.Since(startTime)

	// Log successful search
	if auditLog != nil {
		auditLog("search", query, true, fmt.Sprintf("results:%d,duration:%.3fs",
			len(searchResults), result.ExecutionTime.Seconds()))
	}

	return result
}

// formatSearchOutput formats search results for output
func formatSearchOutput(query string, results []search.SearchResult, maxResults int, duration time.Duration) string {
	var output strings.Builder

	output.WriteString(fmt.Sprintf("=== SEARCH: %s ===\n", query))
	output.WriteString(fmt.Sprintf("=== SEARCH RESULTS (%.2fs) ===\n", duration.Seconds()))

	if len(results) == 0 {
		output.WriteString("No files found matching query.\n")
		output.WriteString("Try broader search terms or check if files are indexed.\n")
		output.WriteString("=== END SEARCH ===\n")
		return output.String()
	}

	for i, result := range results {
		output.WriteString(fmt.Sprintf("%d. %s (score: %.2f)\n",
			i+1, result.FilePath, result.Score*100))

		// File metadata
		output.WriteString(fmt.Sprintf("   Lines: %d | Size: %s",
			result.LineCount, formatFileSizeForSearch(result.FileSize)))
		output.WriteString("\n")

		// Preview
		if result.Preview != "" {
			output.WriteString(fmt.Sprintf("   Preview: \"%s\"\n", result.Preview))
		}

		output.WriteString("\n")
	}

	// Show additional results count
	if len(results) >= maxResults {
		output.WriteString(fmt.Sprintf("[Showing top %d results]\n", maxResults))
	}

	output.WriteString("=== END SEARCH ===\n")
	return output.String()
}

// formatFileSizeForSearch formats file size in human-readable format
func formatFileSizeForSearch(size int64) string {
	const unit = 1024
	if size < unit {
		return fmt.Sprintf("%d B", size)
	}

	div, exp := int64(unit), 0
	for n := size / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}

	return fmt.Sprintf("%.1f %cB", float64(size)/float64(div), "KMGTPE"[exp])
}

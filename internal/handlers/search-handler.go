package handlers

import (
	"fmt"
)

// DefaultSearchHandler implements SearchHandler
type DefaultSearchHandler struct{}

// NewSearchHandler creates a new search handler
func NewSearchHandler() SearchHandler {
	return &DefaultSearchHandler{}
}

// Search performs a search operation (stub implementation)
func (h *DefaultSearchHandler) Search(query string) ([]SearchResult, error) {
	// For now, return an error indicating search is not implemented
	// In the full implementation, this would integrate with the search engine
	return nil, fmt.Errorf("SEARCH_DISABLED: search functionality not implemented in this refactored version")
}

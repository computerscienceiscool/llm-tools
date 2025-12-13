package cli

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestCheckOllamaAvailability_Success tests successful Ollama connection
func TestCheckOllamaAvailability_Success(t *testing.T) {
	// Create a test server that mimics Ollama
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/tags" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"models":[]}`))
		}
	}))
	defer server.Close()

	err := checkOllamaAvailability(server.URL)
	if err != nil {
		t.Errorf("checkOllamaAvailability() unexpected error: %v", err)
	}
}

// TestCheckOllamaAvailability_ServerDown tests when Ollama is not running
func TestCheckOllamaAvailability_ServerDown(t *testing.T) {
	err := checkOllamaAvailability("http://localhost:99999")
	if err == nil {
		t.Error("checkOllamaAvailability() expected error when server is down")
	}
}

// TestCheckOllamaAvailability_WrongStatus tests non-200 response
func TestCheckOllamaAvailability_WrongStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	err := checkOllamaAvailability(server.URL)
	if err == nil {
		t.Error("checkOllamaAvailability() expected error for non-200 status")
	}
}

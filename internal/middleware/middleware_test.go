package middleware

import (
	"bytes"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestLogging(t *testing.T) {
	// Create a buffer to capture log output
	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer log.SetOutput(nil)

	// Create a test handler
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Wrap the test handler with the Logging middleware
	loggingHandler := Logging(testHandler)

	// Create a test request
	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()

	// Serve the request
	loggingHandler.ServeHTTP(rr, req)

	// Check the response
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	// Check the log output
	logOutput := buf.String()
	expectedParts := []string{"Started GET /test", "Completed GET /test in"}
	for _, part := range expectedParts {
		if !strings.Contains(logOutput, part) {
			t.Errorf("log output doesn't contain expected part: %s", part)
		}
	}
}

func TestJSONContentType(t *testing.T) {
	// Create a test handler
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Wrap the test handler with the JSONContentType middleware
	jsonHandler := JSONContentType(testHandler)

	// Create a test request
	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()

	// Serve the request
	jsonHandler.ServeHTTP(rr, req)

	// Check the response
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	// Check the Content-Type header
	contentType := rr.Header().Get("Content-Type")
	expectedContentType := "application/json"
	if contentType != expectedContentType {
		t.Errorf("handler returned wrong Content-Type: got %v want %v", contentType, expectedContentType)
	}
}

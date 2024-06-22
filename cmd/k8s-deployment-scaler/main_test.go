package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHealthCheck(t *testing.T) {
	req, err := http.NewRequest("GET", "/healthz", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(healthCheck)

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	expected := `{"status":"OK"}`
	if strings.TrimSpace(rr.Body.String()) != expected {
		t.Errorf("handler returned unexpected body: got %v want %v", rr.Body.String(), expected)
	}

	contentType := rr.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("handler returned wrong content type: got %v want %v", contentType, "application/json")
	}
}

func TestHandleReplicaCount(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		url            string
		body           string
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "GET all replica counts",
			method:         "GET",
			url:            "/replica-count",
			expectedStatus: http.StatusOK,
			expectedBody:   `{"replicaCounts":{"another-namespace/another-deployment":5,"default/my-deployment":3}}`,
		},
		{
			name:           "GET specific deployment replica count",
			method:         "GET",
			url:            "/replica-count?namespace=default&deployment=my-deployment",
			expectedStatus: http.StatusOK,
			expectedBody:   `{"replicaCount":3}`,
		},
		{
			name:           "GET missing parameters",
			method:         "GET",
			url:            "/replica-count?namespace=default",
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Both namespace and deployment must be specified together\n",
		},
		{
			name:           "POST update replica count",
			method:         "POST",
			url:            "/replica-count?namespace=default&deployment=my-deployment",
			body:           `{"replicas": 5}`,
			expectedStatus: http.StatusOK,
			expectedBody:   `{"replicaCount":5}`,
		},
		{
			name:           "POST invalid replica count",
			method:         "POST",
			url:            "/replica-count?namespace=default&deployment=my-deployment",
			body:           `{"replicas": -1}`,
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Replica count must be non-negative\n",
		},
		{
			name:           "POST missing parameters",
			method:         "POST",
			url:            "/replica-count?namespace=default",
			body:           `{"replicas": 5}`,
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Missing query parameters\n",
		},
		{
			name:           "Invalid method",
			method:         "PUT",
			url:            "/replica-count",
			expectedStatus: http.StatusMethodNotAllowed,
			expectedBody:   "Method not allowed\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest(tt.method, tt.url, strings.NewReader(tt.body))
			if err != nil {
				t.Fatal(err)
			}

			rr := httptest.NewRecorder()
			handler := http.HandlerFunc(handleReplicaCount)

			handler.ServeHTTP(rr, req)

			if status := rr.Code; status != tt.expectedStatus {
				t.Errorf("handler returned wrong status code: got %v want %v", status, tt.expectedStatus)
			}

			if strings.TrimSpace(rr.Body.String()) != strings.TrimSpace(tt.expectedBody) {
				t.Errorf("handler returned unexpected body: got %v want %v", rr.Body.String(), tt.expectedBody)
			}
		})
	}
}

func TestListDeployments(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		url            string
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "List all deployments",
			url:            "/deployments",
			expectedStatus: http.StatusOK,
			expectedBody:   `{"deployments":["default/my-deployment","another-namespace/another-deployment"]}`,
		},
		{
			name:           "List deployments for specific namespace",
			url:            "/deployments?namespace=test-namespace",
			expectedStatus: http.StatusOK,
			expectedBody:   `{"deployments":["test-namespace/my-deployment"]}`,
		},
		{
			name:           "Invalid method",
			method:         "POST",
			url:            "/deployments",
			expectedStatus: http.StatusMethodNotAllowed,
			expectedBody:   "Method not allowed\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest(tt.method, tt.url, nil)
			if err != nil {
				t.Fatal(err)
			}

			rr := httptest.NewRecorder()
			handler := http.HandlerFunc(listDeployments)

			handler.ServeHTTP(rr, req)

			if status := rr.Code; status != tt.expectedStatus {
				t.Errorf("handler returned wrong status code: got %v want %v", status, tt.expectedStatus)
			}

			if strings.TrimSpace(rr.Body.String()) != strings.TrimSpace(tt.expectedBody) {
				t.Errorf("handler returned unexpected body: got %v want %v", rr.Body.String(), tt.expectedBody)
			}
		})
	}
}

func TestEncodeAndWriteJSON(t *testing.T) {
	rr := httptest.NewRecorder()
	data := map[string]string{"key": "value"}

	err := encodeAndWriteJSON(rr, data)
	if err != nil {
		t.Fatalf("encodeAndWriteJSON returned an error: %v", err)
	}

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	var result map[string]string
	err = json.NewDecoder(rr.Body).Decode(&result)
	if err != nil {
		t.Fatalf("Error decoding JSON response: %v", err)
	}

	expected := map[string]string{"key": "value"}
	if result["key"] != expected["key"] {
		t.Errorf("handler returned unexpected body: got %v want %v", result, expected)
	}
}

func TestLoggingMiddleware(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req, err := http.NewRequest("GET", "/test", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	loggingMiddleware(handler).ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}
}

func TestSetupHandlers(t *testing.T) {
	handler := setupHandlers()

	testCases := []struct {
		path           string
		expectedStatus int
	}{
		{"/healthz", http.StatusOK},
		{"/replica-count", http.StatusOK},
		{"/deployments", http.StatusOK},
		{"/nonexistent", http.StatusNotFound},
	}
	for _, tc := range testCases {
		req, err := http.NewRequest("GET", tc.path, nil)
		if err != nil {
			t.Fatal(err)
		}

		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		if status := rr.Code; status != tc.expectedStatus {
			t.Errorf("handler returned wrong status code for %s: got %v want %v", tc.path, status, tc.expectedStatus)
		}
	}
}

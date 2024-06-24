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
	handler := setupHandlers()

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

func TestHandleGetReplicaCount(t *testing.T) {
	tests := []struct {
		name           string
		url            string
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "GET all replica counts",
			url:            "/replica-count",
			expectedStatus: http.StatusOK,
			expectedBody:   `{"replicaCounts":{"another-namespace/another-deployment":5,"default/my-deployment":3}}`,
		},
		{
			name:           "GET specific deployment replica count",
			url:            "/replica-count?namespace=default&deployment=my-deployment",
			expectedStatus: http.StatusOK,
			expectedBody:   `{"replicaCount":3}`,
		},
		{
			name:           "GET missing parameters",
			url:            "/replica-count?namespace=default",
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `{"message":"Both namespace and deployment must be specified together","code":400}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest("GET", tt.url, nil)
			if err != nil {
				t.Fatal(err)
			}

			rr := httptest.NewRecorder()
			handler := jsonContentTypeMiddleware(http.HandlerFunc(handleGetReplicaCount))

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

func TestHandlePostReplicaCount(t *testing.T) {
	tests := []struct {
		name           string
		url            string
		body           string
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "POST update replica count",
			url:            "/replica-count?namespace=default&deployment=my-deployment",
			body:           `{"replicas": 5}`,
			expectedStatus: http.StatusOK,
			expectedBody:   `{"replicaCount":5}`,
		},
		{
			name:           "POST invalid replica count",
			url:            "/replica-count?namespace=default&deployment=my-deployment",
			body:           `{"replicas": -1}`,
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `{"message":"Replica count must be non-negative","code":400}`,
		},
		{
			name:           "POST missing parameters",
			url:            "/replica-count?namespace=default",
			body:           `{"replicas": 5}`,
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `{"message":"Missing query parameters","code":400}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest("POST", tt.url, strings.NewReader(tt.body))
			if err != nil {
				t.Fatal(err)
			}

			rr := httptest.NewRecorder()
			handler := jsonContentTypeMiddleware(http.HandlerFunc(handlePostReplicaCount))

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
			method:         "GET",
			url:            "/deployments",
			expectedStatus: http.StatusOK,
			expectedBody:   `{"deployments":["default/my-deployment","another-namespace/another-deployment"]}`,
		},
		{
			name:           "List deployments for specific namespace",
			method:         "GET",
			url:            "/deployments?namespace=test-namespace",
			expectedStatus: http.StatusOK,
			expectedBody:   `{"deployments":["test-namespace/my-deployment"]}`,
		},
		{
			name:           "Invalid method",
			method:         "POST",
			url:            "/deployments",
			expectedStatus: http.StatusMethodNotAllowed,
			expectedBody:   "Method Not Allowed\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest(tt.method, tt.url, nil)
			if err != nil {
				t.Fatal(err)
			}

			rr := httptest.NewRecorder()
			handler := setupHandlers()

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

func TestJSONContentTypeMiddleware(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("test"))
	})

	req, err := http.NewRequest("GET", "/test", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	jsonContentTypeMiddleware(handler).ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	contentType := rr.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("middleware did not set correct Content-Type: got %v want %v", contentType, "application/json")
	}

	expected := "test"
	if rr.Body.String() != expected {
		t.Errorf("handler returned unexpected body: got %v want %v", rr.Body.String(), expected)
	}
}

func TestSetupHandlers(t *testing.T) {
	handler := setupHandlers()

	testCases := []struct {
		method         string
		path           string
		expectedStatus int
	}{
		{"GET", "/healthz", http.StatusOK},
		{"GET", "/replica-count", http.StatusOK},
		{"POST", "/replica-count", http.StatusBadRequest}, // Expects query parameters
		{"GET", "/deployments", http.StatusOK},
		{"GET", "/nonexistent", http.StatusNotFound},
		{"POST", "/healthz", http.StatusMethodNotAllowed},
		{"PUT", "/replica-count", http.StatusMethodNotAllowed},
		{"DELETE", "/deployments", http.StatusMethodNotAllowed},
	}
	for _, tc := range testCases {
		req, err := http.NewRequest(tc.method, tc.path, nil)
		if err != nil {
			t.Fatal(err)
		}

		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		if status := rr.Code; status != tc.expectedStatus {
			t.Errorf("handler returned wrong status code for %s %s: got %v want %v", tc.method, tc.path, status, tc.expectedStatus)
		}
	}
}

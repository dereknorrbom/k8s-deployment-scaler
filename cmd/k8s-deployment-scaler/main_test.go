package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes/fake"
)

func TestHealthCheck(t *testing.T) {
	// Initialize the clientset with a fake client
	clientset = fake.NewSimpleClientset()

	// Set up the fake informer
	factory := informers.NewSharedInformerFactory(clientset, 0)
	deploymentInformer = factory.Apps().V1().Deployments().Informer()

	// Start the informer and wait for the cache to sync
	stopCh := make(chan struct{})
	factory.Start(stopCh)
	factory.WaitForCacheSync(stopCh)
	defer close(stopCh)

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
	// Create a fake clientset with a deployment
	fakeClientset := fake.NewSimpleClientset(&appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-deployment",
			Namespace: "default",
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: int32Ptr(3),
		},
	})

	// Set up the fake informer
	factory := informers.NewSharedInformerFactory(fakeClientset, 0)
	deploymentInformer = factory.Apps().V1().Deployments().Informer()

	// Start the informer and wait for the cache to sync
	stopCh := make(chan struct{})
	factory.Start(stopCh)
	factory.WaitForCacheSync(stopCh)
	defer close(stopCh)

	tests := []struct {
		name           string
		url            string
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "GET specific deployment replica count",
			url:            "/replica-count?namespace=default&deployment=my-deployment",
			expectedStatus: http.StatusOK,
			expectedBody:   `{"replicaCount":3}`,
			// ... (rest of the test cases)
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
	// Create a fake clientset
	fakeClientset := fake.NewSimpleClientset(&appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-deployment",
			Namespace: "default",
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: int32Ptr(3),
		},
	})

	// Set up the fake informer
	factory := informers.NewSharedInformerFactory(fakeClientset, 0)
	deploymentInformer = factory.Apps().V1().Deployments().Informer()

	// Start the informer and wait for the cache to sync
	stopCh := make(chan struct{})
	factory.Start(stopCh)
	factory.WaitForCacheSync(stopCh)
	defer close(stopCh)

	// Set the global clientset to our fake clientset
	clientset = fakeClientset

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
		{
			name:           "POST deployment not found in Kubernetes",
			url:            "/replica-count?namespace=default&deployment=non-existent",
			body:           `{"replicas": 5}`,
			expectedStatus: http.StatusNotFound,
			expectedBody:   `{"message":"Deployment not found in cache","code":404}`,
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

			if tt.expectedStatus == http.StatusOK {
				var result map[string]int32
				err := json.Unmarshal(rr.Body.Bytes(), &result)
				if err != nil {
					t.Fatalf("Error unmarshaling JSON response: %v", err)
				}

				if replicaCount, ok := result["replicaCount"]; !ok || replicaCount != 5 {
					t.Errorf("handler returned unexpected replicaCount: got %v want %v", replicaCount, 5)
				}
			} else {
				if strings.TrimSpace(rr.Body.String()) != strings.TrimSpace(tt.expectedBody) {
					t.Errorf("handler returned unexpected body: got %v want %v", rr.Body.String(), tt.expectedBody)
				}
			}
		})
	}
}

func TestListDeployments(t *testing.T) {
	// Create a fake clientset with multiple deployments
	fakeClientset := fake.NewSimpleClientset(
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "my-deployment",
				Namespace: "default",
			},
		},
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "another-deployment",
				Namespace: "another-namespace",
			},
		},
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "my-deployment",
				Namespace: "test-namespace",
			},
		},
	)

	// Set up the fake informer
	factory := informers.NewSharedInformerFactory(fakeClientset, 0)
	deploymentInformer = factory.Apps().V1().Deployments().Informer()

	// Start the informer and wait for the cache to sync
	stopCh := make(chan struct{})
	factory.Start(stopCh)
	factory.WaitForCacheSync(stopCh)
	defer close(stopCh)

	tests := []struct {
		name                string
		method              string
		url                 string
		expectedStatus      int
		expectedDeployments []string
		expectedBody        string
	}{
		{
			name:                "List all deployments",
			method:              "GET",
			url:                 "/deployments",
			expectedStatus:      http.StatusOK,
			expectedDeployments: []string{"default/my-deployment", "another-namespace/another-deployment", "test-namespace/my-deployment"},
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

			if tt.expectedDeployments != nil {
				var result map[string][]string
				err := json.Unmarshal(rr.Body.Bytes(), &result)
				if err != nil {
					t.Fatalf("Error unmarshaling JSON response: %v", err)
				}

				if deployments, ok := result["deployments"]; ok {
					for _, expectedDeployment := range tt.expectedDeployments {
						found := false
						for _, actualDeployment := range deployments {
							if actualDeployment == expectedDeployment {
								found = true
								break
							}
						}
						if !found {
							t.Errorf("Expected deployment %s not found in response", expectedDeployment)
						}
					}
				} else {
					t.Errorf("Response does not contain 'deployments' key")
				}
			} else if tt.expectedBody != "" {
				if strings.TrimSpace(rr.Body.String()) != strings.TrimSpace(tt.expectedBody) {
					t.Errorf("handler returned unexpected body: got %v want %v", rr.Body.String(), tt.expectedBody)
				}
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

	// Create a fake clientset with a deployment
	fakeClientset := fake.NewSimpleClientset(&appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-deployment",
			Namespace: "default",
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: int32Ptr(3),
		},
	})

	// Set up the fake informer
	factory := informers.NewSharedInformerFactory(fakeClientset, 0)
	deploymentInformer = factory.Apps().V1().Deployments().Informer()

	// Start the informer and wait for the cache to sync
	stopCh := make(chan struct{})
	factory.Start(stopCh)
	factory.WaitForCacheSync(stopCh)
	defer close(stopCh)

	testCases := []struct {
		method         string
		path           string
		expectedStatus int
	}{
		{"GET", "/healthz", http.StatusOK},
		{"GET", "/replica-count?namespace=default&deployment=test-deployment", http.StatusOK},
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

func TestGetDeploymentFromCache(t *testing.T) {
	// Create a fake clientset with a deployment
	fakeClientset := fake.NewSimpleClientset(&appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "existing-deployment",
			Namespace: "default",
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: int32Ptr(3),
		},
	})

	// Set up the fake informer
	factory := informers.NewSharedInformerFactory(fakeClientset, 0)
	deploymentInformer = factory.Apps().V1().Deployments().Informer()

	// Start the informer and wait for the cache to sync
	stopCh := make(chan struct{})
	factory.Start(stopCh)
	factory.WaitForCacheSync(stopCh)
	defer close(stopCh)

	tests := []struct {
		name             string
		namespace        string
		deploymentName   string
		expectedFound    bool
		expectedReplicas int32
	}{
		{
			name:             "Existing deployment",
			namespace:        "default",
			deploymentName:   "existing-deployment",
			expectedFound:    true,
			expectedReplicas: 3,
		},
		{
			name:           "Non-existing deployment",
			namespace:      "default",
			deploymentName: "non-existing-deployment",
			expectedFound:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info, found := getDeploymentFromCache(tt.namespace, tt.deploymentName)
			if found != tt.expectedFound {
				t.Errorf("getDeploymentFromCache() found = %v, want %v", found, tt.expectedFound)
			}
			if found && info.Replicas != tt.expectedReplicas {
				t.Errorf("getDeploymentFromCache() replicas = %v, want %v", info.Replicas, tt.expectedReplicas)
			}
		})
	}
}

// Helper function to create a pointer to an int32
func int32Ptr(i int32) *int32 {
	return &i
}

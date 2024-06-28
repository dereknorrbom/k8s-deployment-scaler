package handlers

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	appslisters "k8s.io/client-go/listers/apps/v1"
)

// Mock DeploymentLister
type mockDeploymentLister struct {
	deployments map[string]*appsv1.Deployment
}

func (m *mockDeploymentLister) List(selector labels.Selector) (ret []*appsv1.Deployment, err error) {
	panic("not implemented")
}

func (m *mockDeploymentLister) Deployments(namespace string) appslisters.DeploymentNamespaceLister {
	return &mockDeploymentNamespaceLister{
		namespace:   namespace,
		deployments: m.deployments,
	}
}

type mockDeploymentNamespaceLister struct {
	namespace   string
	deployments map[string]*appsv1.Deployment
}

func (m *mockDeploymentNamespaceLister) List(selector labels.Selector) (ret []*appsv1.Deployment, err error) {
	panic("not implemented")
}

func (m *mockDeploymentNamespaceLister) Get(name string) (*appsv1.Deployment, error) {
	key := fmt.Sprintf("%s/%s", m.namespace, name)
	if deployment, ok := m.deployments[key]; ok {
		return deployment, nil
	}
	return nil, errors.NewNotFound(schema.GroupResource{Group: "apps", Resource: "deployments"}, name)
}

func TestGetDeploymentFromCache(t *testing.T) {
	// Create a mock deployment lister
	mockLister := &mockDeploymentLister{
		deployments: map[string]*appsv1.Deployment{
			"test-namespace/test-deployment": {
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-deployment",
					Namespace: "test-namespace",
				},
			},
		},
	}

	tests := []struct {
		name           string
		namespace      string
		deploymentName string
		wantFound      bool
	}{
		{
			name:           "Existing deployment",
			namespace:      "test-namespace",
			deploymentName: "test-deployment",
			wantFound:      true,
		},
		{
			name:           "Non-existing deployment",
			namespace:      "test-namespace",
			deploymentName: "non-existing",
			wantFound:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, found := getDeploymentFromCache(tt.namespace, tt.deploymentName, mockLister)
			if found != tt.wantFound {
				t.Errorf("getDeploymentFromCache() found = %v, want %v", found, tt.wantFound)
			}
		})
	}
}

func TestValidateQueryParams(t *testing.T) {
	tests := []struct {
		name           string
		url            string
		wantNamespace  string
		wantDeployment string
		wantError      bool
	}{
		{
			name:           "Valid params",
			url:            "/api?namespace=test-ns&deployment=test-dep",
			wantNamespace:  "test-ns",
			wantDeployment: "test-dep",
			wantError:      false,
		},
		{
			name:      "Missing namespace",
			url:       "/api?deployment=test-dep",
			wantError: true,
		},
		{
			name:      "Missing deployment",
			url:       "/api?namespace=test-ns",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, _ := http.NewRequest("GET", tt.url, nil)
			namespace, deployment, apiErr := validateQueryParams(req)

			if tt.wantError {
				if apiErr == nil {
					t.Errorf("validateQueryParams() error = nil, wantError %v", tt.wantError)
				}
			} else {
				if apiErr != nil {
					t.Errorf("validateQueryParams() unexpected error = %v", apiErr)
				}
				if namespace != tt.wantNamespace {
					t.Errorf("validateQueryParams() namespace = %v, want %v", namespace, tt.wantNamespace)
				}
				if deployment != tt.wantDeployment {
					t.Errorf("validateQueryParams() deployment = %v, want %v", deployment, tt.wantDeployment)
				}
			}
		})
	}
}

func TestEncodeAndWriteJSON(t *testing.T) {
	tests := []struct {
		name         string
		data         interface{}
		expectedBody string
	}{
		{
			name:         "Encode struct",
			data:         struct{ Message string }{"Hello"},
			expectedBody: `{"Message":"Hello"}`,
		},
		{
			name:         "Encode map",
			data:         map[string]int{"a": 1, "b": 2},
			expectedBody: `{"a":1,"b":2}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			err := encodeAndWriteJSON(w, tt.data)
			if err != nil {
				t.Fatalf("encodeAndWriteJSON() error = %v", err)
			}
			if w.Body.String() != tt.expectedBody+"\n" {
				t.Errorf("encodeAndWriteJSON() body = %v, want %v", w.Body.String(), tt.expectedBody)
			}
		})
	}
}

func TestWriteJSONError(t *testing.T) {
	tests := []struct {
		name         string
		apiErr       apiError
		expectedCode int
		expectedBody string
	}{
		{
			name:         "Bad Request",
			apiErr:       apiError{Message: "Bad Request", Code: http.StatusBadRequest},
			expectedCode: http.StatusBadRequest,
			expectedBody: `{"message":"Bad Request","code":400}`,
		},
		{
			name:         "Not Found",
			apiErr:       apiError{Message: "Not Found", Code: http.StatusNotFound},
			expectedCode: http.StatusNotFound,
			expectedBody: `{"message":"Not Found","code":404}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			writeJSONError(w, tt.apiErr)
			if w.Code != tt.expectedCode {
				t.Errorf("writeJSONError() status code = %v, want %v", w.Code, tt.expectedCode)
			}
			if w.Body.String() != tt.expectedBody+"\n" {
				t.Errorf("writeJSONError() body = %v, want %v", w.Body.String(), tt.expectedBody)
			}
			if w.Header().Get("Content-Type") != "application/json" {
				t.Errorf("writeJSONError() Content-Type = %v, want application/json", w.Header().Get("Content-Type"))
			}
		})
	}
}

func TestWriteInternalServerError(t *testing.T) {
	w := httptest.NewRecorder()
	writeInternalServerError(w, fmt.Errorf("test error"))

	expectedCode := http.StatusInternalServerError
	expectedBody := `{"message":"Internal server error","code":500}`

	if w.Code != expectedCode {
		t.Errorf("writeInternalServerError() status code = %v, want %v", w.Code, expectedCode)
	}
	if w.Body.String() != expectedBody+"\n" {
		t.Errorf("writeInternalServerError() body = %v, want %v", w.Body.String(), expectedBody)
	}
	if w.Header().Get("Content-Type") != "application/json" {
		t.Errorf("writeInternalServerError() Content-Type = %v, want application/json", w.Header().Get("Content-Type"))
	}
}

package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	appslisters "k8s.io/client-go/listers/apps/v1"
)

// apiError represents an error response in JSON format
type apiError struct {
	Message string `json:"message"`
	Code    int    `json:"code"`
}

// getDeploymentFromCache retrieves a deployment from the lister
func getDeploymentFromCache(namespace, name string, deploymentLister appslisters.DeploymentLister) (*appsv1.Deployment, bool) {
	deployment, err := deploymentLister.Deployments(namespace).Get(name)
	if err != nil {
		if errors.IsNotFound(err) {
			return nil, false
		}
		log.Printf("Error getting deployment: %v", err)
		return nil, false
	}
	return deployment, true
}

// validateQueryParams checks if both namespace and deployment are provided
func validateQueryParams(r *http.Request) (string, string, *apiError) {
	namespace := r.URL.Query().Get("namespace")
	deploymentName := r.URL.Query().Get("deployment")

	if namespace == "" || deploymentName == "" {
		return "", "", &apiError{
			Message: "Both namespace and deployment must be specified",
			Code:    http.StatusBadRequest,
		}
	}

	return namespace, deploymentName, nil
}

// encodeAndWriteJSON serializes the given data to JSON and writes it to the http.ResponseWriter.
func encodeAndWriteJSON(w http.ResponseWriter, data interface{}) error {
	err := json.NewEncoder(w).Encode(data)
	if err != nil {
		log.Printf("Error encoding JSON response: %v", err)
		return fmt.Errorf("JSON encoding failed: %w", err)
	}
	return nil
}

// writeJSONError writes an error response in JSON format
func writeJSONError(w http.ResponseWriter, err apiError) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(err.Code)
	json.NewEncoder(w).Encode(err)
}

// writeInternalServerError writes an internal server error response in JSON format
func writeInternalServerError(w http.ResponseWriter, err error) {
	log.Printf("Internal server error: %v", err)
	writeJSONError(w, apiError{
		Message: "Internal server error",
		Code:    http.StatusInternalServerError,
	})
}

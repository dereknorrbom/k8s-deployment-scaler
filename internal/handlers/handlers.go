package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	autoscalingv1 "k8s.io/api/autoscaling/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
	appslisters "k8s.io/client-go/listers/apps/v1"
)

// Define the global clientset variable using kubernetes.Interface
var clientset kubernetes.Interface

// SetClientset sets the global clientset
func SetClientset(cs kubernetes.Interface) {
	clientset = cs
}

// healthCheck handles the /healthz endpoint for health checks
func HealthCheck(w http.ResponseWriter, r *http.Request) {
	// Check Kubernetes connectivity
	_, err := clientset.Discovery().ServerVersion()
	if err != nil {
		log.Printf("Kubernetes connectivity check failed: %v", err)
		writeJSONError(w, apiError{
			Message: "Kubernetes connectivity check failed",
			Code:    http.StatusServiceUnavailable,
		})
		return
	}

	if err := encodeAndWriteJSON(w, map[string]string{"status": "OK"}); err != nil {
		writeInternalServerError(w, err)
	}
}

// handleGetReplicaCount handles the /replica-count endpoint for GET requests
func GetReplicaCount(w http.ResponseWriter, r *http.Request, deploymentLister appslisters.DeploymentLister) {
	namespace, deploymentName, err := validateQueryParams(r)
	if err != nil {
		writeJSONError(w, *err)
		return
	}

	deployment, exists := getDeploymentFromCache(namespace, deploymentName, deploymentLister)
	if !exists {
		writeJSONError(w, apiError{
			Message: "Deployment not found",
			Code:    http.StatusNotFound,
		})
		return
	}

	response := map[string]interface{}{
		"replicaCount": *deployment.Spec.Replicas,
	}
	if err := encodeAndWriteJSON(w, response); err != nil {
		writeInternalServerError(w, err)
	}
}

// handlePostReplicaCount handles the /replica-count endpoint for POST requests
func PostReplicaCount(w http.ResponseWriter, r *http.Request) {
	namespace, deploymentName, apiErr := validateQueryParams(r)
	if apiErr != nil {
		writeJSONError(w, *apiErr)
		return
	}

	var reqBody struct {
		Replicas int32 `json:"replicas"`
	}

	// Decode the request body
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		writeJSONError(w, apiError{
			Message: "Invalid request body",
			Code:    http.StatusBadRequest,
		})
		return
	}

	// Validate the replica count
	if reqBody.Replicas < 0 {
		writeJSONError(w, apiError{
			Message: "Replica count must be non-negative",
			Code:    http.StatusBadRequest,
		})
		return
	}

	// Create the scale object
	scale := &autoscalingv1.Scale{
		ObjectMeta: metav1.ObjectMeta{
			Name:      deploymentName,
			Namespace: namespace,
		},
		Spec: autoscalingv1.ScaleSpec{
			Replicas: reqBody.Replicas,
		},
	}

	// Update the deployment scale
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	_, err := clientset.AppsV1().Deployments(namespace).UpdateScale(ctx, deploymentName, scale, metav1.UpdateOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			writeJSONError(w, apiError{
				Message: "Deployment not found",
				Code:    http.StatusNotFound,
			})
		} else {
			log.Printf("Failed to update deployment scale: %v", err)
			writeJSONError(w, apiError{
				Message: "Failed to update deployment scale",
				Code:    http.StatusInternalServerError,
			})
		}
		return
	}

	// Return the response
	response := map[string]interface{}{
		"replicaCount": reqBody.Replicas,
	}
	if err := encodeAndWriteJSON(w, response); err != nil {
		writeInternalServerError(w, err)
	}
}

// listDeployments handles the /deployments endpoint to list deployments
func ListDeployments(w http.ResponseWriter, r *http.Request, deploymentLister appslisters.DeploymentLister) {
	namespace := r.URL.Query().Get("namespace")

	list, err := deploymentLister.Deployments(namespace).List(labels.Everything())
	if err != nil {
		log.Printf("Error listing deployments: %v", err)
		writeJSONError(w, apiError{
			Message: "Failed to list deployments",
			Code:    http.StatusInternalServerError,
		})
		return
	}

	deployments := make([]string, 0, len(list))
	for _, deployment := range list {
		deployments = append(deployments, fmt.Sprintf("%s/%s", deployment.Namespace, deployment.Name))
	}

	response := map[string]interface{}{
		"deployments": deployments,
	}

	if err := encodeAndWriteJSON(w, response); err != nil {
		writeInternalServerError(w, err)
	}
}

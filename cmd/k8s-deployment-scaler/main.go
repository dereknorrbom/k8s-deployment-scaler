package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	appslisters "k8s.io/client-go/listers/apps/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
)

// Define the global clientset variable using kubernetes.Interface
var clientset kubernetes.Interface

func getKubernetesConfig() (*rest.Config, error) {
	return clientcmd.BuildConfigFromFlags("", os.Getenv("KUBECONFIG"))
}

// main initializes and runs the server, setting up TLS configuration, handling graceful shutdown, and defining HTTP routes.
func main() {
	// Initialize Kubernetes client
	config, err := getKubernetesConfig()
	if err != nil {
		log.Fatalf("Error building kubeconfig: %v", err)
	}

	clientset, err = kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatalf("Error creating Kubernetes client: %v", err)
	}

	// Set up deployment informer and lister
	factory := informers.NewSharedInformerFactory(clientset, time.Minute*10)
	deploymentInformer := factory.Apps().V1().Deployments()
	deploymentLister := deploymentInformer.Lister()
	deploymentsSynced := deploymentInformer.Informer().HasSynced

	// Start all informers
	stopCh := make(chan struct{})
	defer close(stopCh)
	factory.Start(stopCh)

	// Wait for the deployment cache to sync
	if !cache.WaitForCacheSync(stopCh, deploymentsSynced) {
		log.Fatal("Failed to sync deployment informer")
	}

	tlsConfig, err := setupTLSConfig()
	if err != nil {
		log.Fatalf("Failed to set up TLS config: %v", err)
	}

	port := os.Getenv("SERVER_PORT")
	if port == "" {
		port = "8443"
	}

	server := &http.Server{
		Addr:      fmt.Sprintf(":%s", port),
		TLSConfig: tlsConfig,
		Handler:   loggingMiddleware(setupHandlers(deploymentLister)),
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	go func() {
		log.Printf("Server starting on :%s...\n", port)
		if err := server.ListenAndServeTLS("", ""); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	<-stop

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server shutdown failed: %v", err)
	}
	log.Println("Server gracefully stopped")
}

// setupTLSConfig loads certificates and sets up TLS configuration.
func setupTLSConfig() (*tls.Config, error) {
	serverCert, err := tls.LoadX509KeyPair("certs/server-cert.pem", "certs/server-key.pem")
	if err != nil {
		return nil, fmt.Errorf("loading server certificate: %v", err)
	}

	caCert, err := os.ReadFile("certs/ca-cert.pem")
	if err != nil {
		return nil, fmt.Errorf("loading CA certificate: %v", err)
	}
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)

	return &tls.Config{
		Certificates: []tls.Certificate{serverCert},
		ClientCAs:    caCertPool,
		ClientAuth:   tls.RequireAndVerifyClientCert,
		MinVersion:   tls.VersionTLS13,
		MaxVersion:   tls.VersionTLS13,
		CipherSuites: []uint16{
			tls.TLS_AES_256_GCM_SHA384,
			tls.TLS_CHACHA20_POLY1305_SHA256,
			tls.TLS_AES_128_GCM_SHA256,
		},
	}, nil
}

// loggingMiddleware logs information about incoming requests
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		log.Printf("Started %s %s", r.Method, r.URL.Path)

		next.ServeHTTP(w, r)

		log.Printf("Completed %s %s in %v", r.Method, r.URL.Path, time.Since(start))
	})
}

// jsonContentTypeMiddleware sets the Content-Type header to application/json for all requests
func jsonContentTypeMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		next.ServeHTTP(w, r)
	})
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

// setupHandlers configures and returns the HTTP request multiplexer
func setupHandlers(deploymentLister appslisters.DeploymentLister) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", jsonContentTypeMiddleware(http.HandlerFunc(healthCheck)).ServeHTTP)
	mux.HandleFunc("GET /replica-count", jsonContentTypeMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handleGetReplicaCount(w, r, deploymentLister)
	})).ServeHTTP)
	mux.HandleFunc("POST /replica-count", jsonContentTypeMiddleware(http.HandlerFunc(handlePostReplicaCount)).ServeHTTP)
	mux.HandleFunc("GET /deployments", jsonContentTypeMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		listDeployments(w, r, deploymentLister)
	})).ServeHTTP)
	return mux
}

// apiError represents an error response in JSON format
type apiError struct {
	Message string `json:"message"`
	Code    int    `json:"code"`
}

// healthCheck handles the /healthz endpoint for health checks
func healthCheck(w http.ResponseWriter, r *http.Request) {
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

// getDeploymentFromCache retrieves a deployment from the lister
func getDeploymentFromCache(namespace, name string, deploymentLister appslisters.DeploymentLister) (*appsv1.Deployment, bool) {
	deployment, err := deploymentLister.Deployments(namespace).Get(name)
	if err != nil {
		if errors.IsNotFound(err) {
			return nil, false
		}
		log.Printf("Error getting deployment from cache: %v", err)
		return nil, false
	}
	return deployment, true
}

// handleGetReplicaCount handles the /replica-count endpoint for GET requests
func handleGetReplicaCount(w http.ResponseWriter, r *http.Request, deploymentLister appslisters.DeploymentLister) {
	namespace := r.URL.Query().Get("namespace")
	deploymentName := r.URL.Query().Get("deployment")

	// Validate the query parameters
	if namespace == "" || deploymentName == "" {
		writeJSONError(w, apiError{
			Message: "Both namespace and deployment must be specified",
			Code:    http.StatusBadRequest,
		})
		return
	}

	deployment, exists := getDeploymentFromCache(namespace, deploymentName, deploymentLister)
	if !exists {
		writeJSONError(w, apiError{
			Message: "Deployment not found in cache",
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
func handlePostReplicaCount(w http.ResponseWriter, r *http.Request) {
	namespace := r.URL.Query().Get("namespace")
	deploymentName := r.URL.Query().Get("deployment")

	// Validate the query parameters
	if namespace == "" || deploymentName == "" {
		writeJSONError(w, apiError{
			Message: "Missing query parameters",
			Code:    http.StatusBadRequest,
		})
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
func listDeployments(w http.ResponseWriter, r *http.Request, deploymentLister appslisters.DeploymentLister) {
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

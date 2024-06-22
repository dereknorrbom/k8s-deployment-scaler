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
)

// main initializes and runs the server, setting up TLS configuration, handling graceful shutdown, and defining HTTP routes.
func main() {
	tlsConfig, err := setupTLSConfig()
	if err != nil {
		log.Fatalf("Failed to set up TLS config: %v", err)
	}

	port := os.Getenv("SERVER_PORT")
	if port == "" {
		port = "8080"
	}

	server := &http.Server{
		Addr:      fmt.Sprintf(":%s", port),
		TLSConfig: tlsConfig,
		Handler:   loggingMiddleware(setupHandlers()),
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

// setupHandlers sets up the HTTP handlers for the server
func setupHandlers() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", healthCheck)
	mux.HandleFunc("/replica-count", handleReplicaCount)
	mux.HandleFunc("/deployments", listDeployments)
	return mux
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

// healthCheck handles the /healthz endpoint for health checks
func healthCheck(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := encodeAndWriteJSON(w, map[string]string{"status": "OK"}); err != nil {
		log.Printf("Error in healthCheck: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// handleReplicaCount handles the /replica-count endpoint for both GET and POST requests.
func handleReplicaCount(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	namespace := r.URL.Query().Get("namespace")
	deployment := r.URL.Query().Get("deployment")

	switch r.Method {
	case http.MethodGet:
		handleGetReplicaCount(w, namespace, deployment)
	case http.MethodPost:
		handlePostReplicaCount(w, r, namespace, deployment)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func handleGetReplicaCount(w http.ResponseWriter, namespace, deployment string) {
	if namespace == "" && deployment == "" {
		response := map[string]interface{}{
			"replicaCounts": map[string]int{
				"default/my-deployment":                3,
				"another-namespace/another-deployment": 5,
			},
		}
		if err := encodeAndWriteJSON(w, response); err != nil {
			log.Printf("Error in handleGetReplicaCount: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		}
	} else if namespace != "" && deployment != "" {
		response := map[string]interface{}{
			"replicaCount": 3, // Placeholder response
		}
		if err := encodeAndWriteJSON(w, response); err != nil {
			log.Printf("Error in handleGetReplicaCount: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		}
	} else {
		http.Error(w, "Both namespace and deployment must be specified together", http.StatusBadRequest)
	}
}

func handlePostReplicaCount(w http.ResponseWriter, r *http.Request, namespace, deployment string) {
	if namespace == "" || deployment == "" {
		http.Error(w, "Missing query parameters", http.StatusBadRequest)
		return
	}

	var reqBody struct {
		Replicas int `json:"replicas"`
	}

	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		log.Printf("Error parsing request body: %v", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if reqBody.Replicas < 0 {
		http.Error(w, "Replica count must be non-negative", http.StatusBadRequest)
		return
	}

	log.Printf("Setting replica count for %s/%s to %d", namespace, deployment, reqBody.Replicas)

	response := map[string]interface{}{
		"replicaCount": reqBody.Replicas,
	}
	if err := encodeAndWriteJSON(w, response); err != nil {
		log.Printf("Error in handlePostReplicaCount: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// listDeployments handles the /deployments endpoint to list deployments
func listDeployments(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	namespace := r.URL.Query().Get("namespace")

	var response map[string]interface{}
	if namespace == "" {
		response = map[string]interface{}{
			"deployments": []string{"default/my-deployment", "another-namespace/another-deployment"},
		}
	} else {
		response = map[string]interface{}{
			"deployments": []string{fmt.Sprintf("%s/my-deployment", namespace)},
		}
	}

	if err := encodeAndWriteJSON(w, response); err != nil {
		log.Printf("Error in listDeployments: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

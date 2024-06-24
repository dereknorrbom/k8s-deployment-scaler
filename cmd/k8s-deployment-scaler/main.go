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
		port = "8443"
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
func setupHandlers() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", jsonContentTypeMiddleware(http.HandlerFunc(healthCheck)).ServeHTTP)
	mux.HandleFunc("GET /replica-count", jsonContentTypeMiddleware(http.HandlerFunc(handleGetReplicaCount)).ServeHTTP)
	mux.HandleFunc("POST /replica-count", jsonContentTypeMiddleware(http.HandlerFunc(handlePostReplicaCount)).ServeHTTP)
	mux.HandleFunc("GET /deployments", jsonContentTypeMiddleware(http.HandlerFunc(listDeployments)).ServeHTTP)
	return mux
}

// apiError represents an error response in JSON format
type apiError struct {
	Message string `json:"message"`
	Code    int    `json:"code"`
}

// healthCheck handles the /healthz endpoint for health checks
func healthCheck(w http.ResponseWriter, r *http.Request) {
	if err := encodeAndWriteJSON(w, map[string]string{"status": "OK"}); err != nil {
		writeInternalServerError(w, err)
	}
}

// handleGetReplicaCount handles the /replica-count endpoint for GET requests
func handleGetReplicaCount(w http.ResponseWriter, r *http.Request) {
	namespace := r.URL.Query().Get("namespace")
	deployment := r.URL.Query().Get("deployment")

	if namespace == "" && deployment == "" {
		response := map[string]interface{}{
			"replicaCounts": map[string]int{
				"default/my-deployment":                3,
				"another-namespace/another-deployment": 5,
			},
		}
		if err := encodeAndWriteJSON(w, response); err != nil {
			writeInternalServerError(w, err)
		}
	} else if namespace != "" && deployment != "" {
		response := map[string]interface{}{
			"replicaCount": 3, // Placeholder response
		}
		if err := encodeAndWriteJSON(w, response); err != nil {
			writeInternalServerError(w, err)
		}
	} else {
		writeJSONError(w, apiError{
			Message: "Both namespace and deployment must be specified together",
			Code:    http.StatusBadRequest,
		})
	}
}

// handlePostReplicaCount handles the /replica-count endpoint for POST requests
func handlePostReplicaCount(w http.ResponseWriter, r *http.Request) {
	namespace := r.URL.Query().Get("namespace")
	deployment := r.URL.Query().Get("deployment")

	if namespace == "" || deployment == "" {
		writeJSONError(w, apiError{
			Message: "Missing query parameters",
			Code:    http.StatusBadRequest,
		})
		return
	}

	var reqBody struct {
		Replicas int `json:"replicas"`
	}

	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		writeJSONError(w, apiError{
			Message: "Invalid request body",
			Code:    http.StatusBadRequest,
		})
		return
	}

	if reqBody.Replicas < 0 {
		writeJSONError(w, apiError{
			Message: "Replica count must be non-negative",
			Code:    http.StatusBadRequest,
		})
		return
	}

	log.Printf("Setting replica count for %s/%s to %d", namespace, deployment, reqBody.Replicas)

	response := map[string]interface{}{
		"replicaCount": reqBody.Replicas,
	}
	if err := encodeAndWriteJSON(w, response); err != nil {
		writeInternalServerError(w, err)
	}
}

// listDeployments handles the /deployments endpoint to list deployments
func listDeployments(w http.ResponseWriter, r *http.Request) {
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
		writeInternalServerError(w, err)
	}
}

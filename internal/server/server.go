package server

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"k8s-deployment-scaler/internal/handlers"
	"k8s-deployment-scaler/internal/middleware"

	appslisters "k8s.io/client-go/listers/apps/v1"
)

type Server struct {
	*http.Server
}

type customLogger struct {
	logger *log.Logger
}

func (l *customLogger) Write(p []byte) (n int, err error) {
	message := string(p)
	if strings.Contains(message, "EOF") {
		return len(p), nil // Suppress EOF errors
	}
	return l.logger.Writer().Write(p)
}

// New creates and returns a new Server instance
func New(deploymentLister appslisters.DeploymentLister, enableTLS bool) (*Server, error) {
	var handler http.Handler = setupHandlers(deploymentLister)
	var srv *http.Server

	if enableTLS {
		tlsConfig, err := setupTLSConfig()
		if err != nil {
			return nil, fmt.Errorf("failed to set up TLS config: %v", err)
		}
		srv = &http.Server{
			Addr:      ":8443",
			TLSConfig: tlsConfig,
			ErrorLog:  log.New(&customLogger{logger: log.Default()}, "", 0),
			Handler:   handler,
		}
	} else {
		srv = &http.Server{
			Addr:     ":8443",
			ErrorLog: log.New(&customLogger{logger: log.Default()}, "", 0),
			Handler:  handler,
		}
	}

	return &Server{Server: srv}, nil
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

// setupHandlers configures and returns the HTTP request multiplexer
func setupHandlers(deploymentLister appslisters.DeploymentLister) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", middleware.JSONContentType(http.HandlerFunc(handlers.HealthCheck)).ServeHTTP)
	mux.HandleFunc("GET /replica-count", middleware.JSONContentType(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlers.GetReplicaCount(w, r, deploymentLister)
	})).ServeHTTP)
	mux.HandleFunc("POST /replica-count", middleware.JSONContentType(http.HandlerFunc(handlers.PostReplicaCount)).ServeHTTP)
	mux.HandleFunc("GET /deployments", middleware.JSONContentType(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlers.ListDeployments(w, r, deploymentLister)
	})).ServeHTTP)
	return mux
}

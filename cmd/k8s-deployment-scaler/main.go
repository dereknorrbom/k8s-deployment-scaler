package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"k8s-deployment-scaler/internal/handlers"
	"k8s-deployment-scaler/internal/kubernetes"
	"k8s-deployment-scaler/internal/server"

	"k8s.io/client-go/informers"
	"k8s.io/client-go/tools/cache"
)

func main() {
	// Initialize Kubernetes client
	clientset, err := kubernetes.NewClientset()
	if err != nil {
		log.Fatalf("Error creating Kubernetes client: %v", err)
	}

	handlers.SetClientset(clientset)

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

	// Create and configure the server
	srv, err := server.New(deploymentLister, true)
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}

	// Start the server
	go func() {
		log.Printf("Server starting on %s...\n", srv.Addr)
		if err := srv.ListenAndServeTLS("", ""); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	// Set up graceful shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	<-stop

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server shutdown failed: %v", err)
	}
	log.Println("Server gracefully stopped")
}

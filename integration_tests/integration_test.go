package integration_tests

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	namespace      = "default"
	deploymentName = "k8s-deployment-scaler"
	baseURL        = "https://localhost:8443"
	certPath       = "../certs/client-cert.pem"
	keyPath        = "../certs/client-key.pem"
	caPath         = "../certs/ca-cert.pem"
)

var client *kubernetes.Clientset

func TestMain(m *testing.M) {
	// Set up the Kubernetes client
	kubeconfig := os.Getenv("KUBECONFIG")
	if kubeconfig == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			fmt.Printf("Error getting user home dir: %v\n", err)
			os.Exit(1)
		}
		kubeconfig = filepath.Join(home, ".kube", "config")
	}

	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		fmt.Printf("Error building kubeconfig: %v\n", err)
		os.Exit(1)
	}

	client, err = kubernetes.NewForConfig(config)
	if err != nil {
		fmt.Printf("Error creating Kubernetes client: %v\n", err)
		os.Exit(1)
	}

	// Run the tests
	code := m.Run()

	// Exit
	os.Exit(code)
}

func TestHealthCheck(t *testing.T) {
	resp, err := makeRequest("GET", "/healthz", nil)
	if err != nil {
		t.Fatalf("Error making request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status OK, got %v", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Error reading response body: %v", err)
	}

	var result map[string]string
	err = json.Unmarshal(body, &result)
	if err != nil {
		t.Fatalf("Error unmarshaling JSON: %v", err)
	}

	if result["status"] != "OK" {
		t.Errorf("Expected status 'OK', got %v", result["status"])
	}
}

func TestGetReplicaCount(t *testing.T) {
	resp, err := makeRequest("GET", fmt.Sprintf("/replica-count?namespace=%s&deployment=%s", namespace, deploymentName), nil)
	if err != nil {
		t.Fatalf("Error making request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status OK, got %v", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Error reading response body: %v", err)
	}

	var result map[string]int
	err = json.Unmarshal(body, &result)
	if err != nil {
		t.Fatalf("Error unmarshaling JSON: %v", err)
	}

	if result["replicaCount"] <= 0 {
		t.Errorf("Expected positive replica count, got %v", result["replicaCount"])
	}
}

func TestSetReplicaCount(t *testing.T) {
	newReplicaCount := 3
	payload := fmt.Sprintf(`{"replicas": %d}`, newReplicaCount)

	resp, err := makeRequest("POST", fmt.Sprintf("/replica-count?namespace=%s&deployment=%s", namespace, deploymentName), []byte(payload))
	if err != nil {
		t.Fatalf("Error making request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status OK, got %v", resp.Status)
	}

	// Wait for the deployment to update
	time.Sleep(5 * time.Second)

	// Verify the replica count was updated
	deployment, err := client.AppsV1().Deployments(namespace).Get(context.TODO(), deploymentName, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("Error getting deployment: %v", err)
	}

	if *deployment.Spec.Replicas != int32(newReplicaCount) {
		t.Errorf("Expected %d replicas, got %d", newReplicaCount, *deployment.Spec.Replicas)
	}
}

func TestListDeployments(t *testing.T) {
	resp, err := makeRequest("GET", "/deployments", nil)
	if err != nil {
		t.Fatalf("Error making request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status OK, got %v", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Error reading response body: %v", err)
	}

	var result map[string][]string
	err = json.Unmarshal(body, &result)
	if err != nil {
		t.Fatalf("Error unmarshaling JSON: %v", err)
	}

	if len(result["deployments"]) == 0 {
		t.Errorf("Expected at least one deployment, got none")
	}

	found := false
	for _, deployment := range result["deployments"] {
		if deployment == fmt.Sprintf("%s/%s", namespace, deploymentName) {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("Expected to find deployment %s/%s, but it was not in the list", namespace, deploymentName)
	}
}

func makeRequest(method, path string, payload []byte) (*http.Response, error) {
	cert, err := tls.LoadX509KeyPair(certPath, keyPath)
	if err != nil {
		return nil, fmt.Errorf("error loading client cert: %v", err)
	}

	caCert, err := os.ReadFile(caPath)
	if err != nil {
		return nil, fmt.Errorf("error reading CA cert: %v", err)
	}
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)

	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				RootCAs:      caCertPool,
				Certificates: []tls.Certificate{cert},
			},
		},
	}

	req, err := http.NewRequest(method, baseURL+path, bytes.NewBuffer(payload))
	if err != nil {
		return nil, err
	}

	if method == "POST" {
		req.Header.Set("Content-Type", "application/json")
	}

	return client.Do(req)
}

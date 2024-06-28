package kubernetes

import (
	"fmt"
	"os"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// NewClientset creates a new Kubernetes clientset
func NewClientset() (kubernetes.Interface, error) {
	config, err := getKubernetesConfig()
	if err != nil {
		return nil, fmt.Errorf("error building kubeconfig: %v", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("error creating Kubernetes client: %v", err)
	}

	return clientset, nil
}

// getKubernetesConfig returns a Kubernetes rest.Config, using in-cluster config if running in cluster,
// or kubeconfig if running outside the cluster
func getKubernetesConfig() (*rest.Config, error) {
	// Try in-cluster config first
	config, err := rest.InClusterConfig()
	if err == nil {
		return config, nil
	}

	// If in-cluster config fails, try kubeconfig
	kubeconfigPath := os.Getenv("KUBECONFIG")
	if kubeconfigPath == "" {
		kubeconfigPath = os.ExpandEnv("$HOME/.kube/config")
	}

	config, err = clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	if err != nil {
		return nil, fmt.Errorf("error building kubeconfig: %v", err)
	}

	return config, nil
}

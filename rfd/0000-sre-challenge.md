### Design Document for Kubernetes Deployment Scaler

---
authors: Derek Norrbom (dereknorrbom@gmail.com)
state: draft
---

## What

This document outlines the proposed design and implementation plan for a Kubernetes Deployment Scaler API. The API will provide functionality to get and set the replica count of a Kubernetes deployment, utilizing mTLS for secure communication and caching for efficient data retrieval.

## Why

The purpose of this project is to implement a scalable, secure API for managing Kubernetes deployments. This project will demonstrate proficiency in Go, Kubernetes, Docker, and security best practices (mTLS). It will also highlight the ability to design, build, and document an application that can be easily extended and maintained.

## How

### API Structure

The API will provide the following endpoints:

- **GET /healthz**: Health check endpoint that returns "OK" if the server is running and Kubernetes connectivity is verified.
- **GET /replica-count?namespace={namespace}&deployment={deployment}**: Returns the current replica count of the specified deployment.
- **POST /replica-count?namespace={namespace}&deployment={deployment}**: Sets the replica count of the specified deployment.
- **GET /deployments?namespace={namespace}**: Lists all deployments in the specified namespace.

Each endpoint will be secured using mutual TLS (mTLS) to ensure secure communication between the client and the server.

#### Example JSON Responses

- **GET /healthz**
   ##### Response:
   ```json
   {
      "status": "OK"
   }
   ```

- **GET /replica-count?namespace=default&deployment=my-deployment**
   ##### Response:
   ```json
   {
      "replicaCount": 3
   }
   ```

- **POST /replica-count?namespace=default&deployment=my-deployment**
   ##### Request Body
   ```json
   {
      "replicas": 3
   }
   ```
   ##### Response
   ```json
   {
      "replicaCount": 3
   }
   ```

- **GET /deployments?namespace=default**
   ##### Response
   ```json
   {
      "deployments": ["default/my-deployment"]
   }
   ```

### Developer Workflow

1. **Clone the Repository**: 
   ```
   git clone <repository-url>
   cd k8s-deployment-scaler
   ```

2. **Run the Setup Script**:
   A `setup.sh` script will be provided to automate the detection and installation of the required dependencies (Docker, KIND, Helm, kubectl (Kubernetes command-line tool)). The setup script will target x86 64-bit Linux and MacOS environments.
   ```
   make setup
   ```

3. **Build and Deploy**: 
   Use a Makefile to automate the entire setup and deployment process. The `make all` command will generate certificates, build the Docker image, create the KIND cluster, load the Docker image into the cluster, deploy the Helm chart, and set up port forwarding.
   ```
   make all
   ```

### Makefile Design

The Makefile will streamline the development and deployment workflow by providing a single command to perform all necessary tasks. This will improve developer efficiency and ensure consistency in the setup process.

Proposed Makefile:

```
.PHONY: all docker-build kind-create kind-delete kind-recreate load-image deploy clean port-forward generate-certs teardown

# The default target that builds the Docker image, recreates the kind cluster, loads the image, deploys the application, and sets up port forwarding
all: docker-build generate-certs kind-recreate load-image deploy port-forward

# Build the Docker image for the application
docker-build:
	@echo "Building Docker image..."
	docker build -t k8s-deployment-scaler:latest .

# Create a new kind cluster
kind-create:
	@echo "Creating kind cluster..."
	kind create cluster --name k8s-deployment-scaler

# Delete the existing kind cluster
kind-delete:
	@echo "Deleting kind cluster..."
	kind delete cluster --name k8s-deployment-scaler || true

# Recreate the kind cluster by deleting and then creating it
kind-recreate: kind-delete kind-create

# Load the Docker image into the kind cluster
load-image:
	@echo "Loading Docker image into kind cluster..."
	kind load docker-image k8s-deployment-scaler:latest --name k8s-deployment-scaler

# Deploy the application using Helm
deploy:
	@echo "Deploying application with Helm..."
	helm upgrade --install k8s-deployment-scaler ./helm/k8s-deployment-scaler

# Set up port forwarding to access the application locally
port-forward:
	@echo "Setting up port forwarding..."
	kubectl wait --for=condition=ready pod -l app=k8s-deployment-scaler --timeout=120s
	kubectl port-forward svc/k8s-deployment-scaler 8080:80

# Clean up by removing the certs directory
clean:
	@echo "Cleaning up..."
	rm -rf ./certs

# Tear down by deleting the kind cluster and removing the Docker image
teardown: clean
	kind delete cluster --name k8s-deployment-scaler
	docker rmi k8s-deployment-scaler:latest

# Generate certificates by copying them from the Docker image to the local certs directory
generate-certs:
	@echo "Generating certificates..."
	docker create --name temp-container k8s-deployment-scaler:latest
	docker cp temp-container:/app/certs ./
	docker rm temp-container
```

### Ease of Contributing to the Project from a Fresh Clone

The repository will include comprehensive setup instructions and scripts to automate environment setup, certificate generation, and deployment processes. This ensures that new developers can quickly get up to speed and contribute effectively.

### Ease of Building, Running, and Testing the Server

- **Building**: The Dockerfile will automate the build process, ensuring that the server can be built consistently across different environments.
- **Running**: The provided Makefile and Helm charts will simplify the deployment process, allowing developers to quickly deploy and test the server in a Kubernetes cluster.
- **Testing**: The server will include endpoints that can be tested using curl or any other HTTP client. The use of mTLS will ensure that tests also verify the security of the communication.

### Build and Release

The build process will be automated using Docker. The Dockerfile will ensure that the server is built consistently, and the Helm charts will automate the deployment process. This setup will allow for easy integration into CI/CD pipelines for automated builds and releases.

### Caching, mTLS, and Delivery

#### Caching

The API will use a caching mechanism to store and retrieve the replica count of the Kubernetes deployment. This will reduce the number of API calls to the Kubernetes API server, improving performance and efficiency.

##### Implementation Details:
1. **In-Memory Cache**: Use an in-memory map to store the replica counts.
2. **Synchronization**: Use a `sync.RWMutex` to manage concurrent access to the cache.
3. **Kubernetes Informers**: Use Kubernetes informers to watch for changes to deployments and update the cache accordingly.
   - **AddFunc**: Triggered when a deployment is added. It updates the cache.
   - **UpdateFunc**: Triggered when a deployment is updated. It updates the cache.
   - **DeleteFunc**: Triggered when a deployment is deleted. It removes the entry from the cache.

#### mTLS

The API will use mutual TLS (mTLS) for secure communication. This involves the following steps:
- Generating a CA certificate and key.
- Generating server and client certificates signed by the CA.
- Configuring the server to require client certificates for authentication.
- Configuring the client to use its certificate and the CA certificate to verify the server.

### Security

The chosen cipher suites for mTLS will be:
- TLS_AES_128_GCM_SHA256
- TLS_AES_256_GCM_SHA384
- TLS_CHACHA20_POLY1305_SHA256

Additional security measures include:
- Strict validation of client certificates.

### Testing

Testing will be a critical part of ensuring the robustness and security of the API. The following testing strategies will be implemented:

1. **Unit Tests**: These tests will cover individual functions and components of the Go application, ensuring they work as expected in isolation.
2. **Integration Tests**: These tests will validate the interactions between the API endpoints and the Kubernetes cluster. They will be run in the KIND cluster to simulate a real Kubernetes environment.
3. **End-to-End Tests**: These tests will cover the full workflow from deploying the application, interacting with the API, and verifying the expected outcomes. mTLS configurations will be tested to ensure secure communication.
4. **Happy Path Tests**: These tests will cover scenarios where the API behaves as expected:
   - Successfully retrieving the replica count.
   - Successfully setting the replica count.
5. **Unhappy Path Tests**: These tests will cover scenarios where the API encounters errors:
   - Attempting to set an invalid replica count.
   - Failing to retrieve the replica count due to missing deployment.

### Delivery

The API will be delivered as a Docker image, which can be deployed to any Kubernetes cluster using Helm charts. This ensures that the API can be easily deployed and managed in different environments.

#### Kubernetes Resources Deployed by Helm

The Helm chart will deploy the following Kubernetes resources:

- **Deployment**: Manages the deployment of the API Docker container, ensuring the specified number of replicas are running.
- **Service**: Exposes the API deployment within the Kubernetes cluster, allowing it to be accessed by other services or through port forwarding.
- **ClusterRole and ClusterRoleBinding**: Grants cluster-wide permissions necessary for the API to interact with Kubernetes resources.
- **Role and RoleBinding**: Grants namespace-specific permissions for the API to interact with Kubernetes resources.

### Extending to Multiple Target Kubernetes Clusters

While the current implementation targets a single Kubernetes cluster, extending it to support multiple clusters involves several key considerations. Note that this section explains how this might be implemented if it were part of the project scope:

1. **Configuration Management**:
   - Use a configuration file or environment variables to specify the contexts for multiple Kubernetes clusters.
   - Example configuration file (`clusters.yaml`):
```yaml
clusters:
  - name: cluster1
    context: /path/to/cluster1/kubeconfig
  - name: cluster2
    context: /path/to/cluster2/kubeconfig
```
2. **Context Switching**:
   - Modify the API to switch contexts based on the target cluster specified in the request.
   - Use the `client-go` library to dynamically load and switch between different kubeconfig files.

3. **API Changes**:
   - Update the API endpoints to accept a cluster identifier as a parameter.
   - Example:
```sh
GET /{cluster}/replica-count
POST /{cluster}/set-replica-count
GET /{cluster}/deployments
```
4. **Concurrency and Caching**:
   - Implement separate caches for each cluster to avoid conflicts and ensure data consistency.
   - Use goroutines and synchronization mechanisms to handle concurrent requests to different clusters.

5. **Security**:
   - Ensure that mTLS is configured for each cluster context.
   - Manage and distribute certificates securely for each cluster.

## Conclusion

This design document outlines the structure, workflow, and security considerations for the Kubernetes Deployment Scaler API. The project aims to demonstrate the ability to design, implement, and document a secure, scalable application that can be easily extended and maintained. The use of mTLS for secure communication and caching for efficient data retrieval ensures that the API will be both secure and performant.

This design document provides a comprehensive overview of the project, covering the key aspects of API structure, developer workflow, ease of contribution, build and release processes, caching, mTLS, and testing strategies.

# Kubernetes Deployment Scaler API

This project implements a secure API to manage Kubernetes deployments. The API is built with Go and uses mutual TLS (mTLS) for secure communication. It provides endpoints to manage deployment replica counts and list deployments, with efficient caching for read operations.

## Table of Contents
1. [Features](#features)
2. [Prerequisites](#prerequisites)
3. [Setup](#setup)
4. [Running the Application](#running-the-application)
5. [API Endpoints](#api-endpoints)
6. [Testing](#testing)
   - [Integration Tests](#integration-tests)
7. [Cleanup](#cleanup)
8. [Makefile Commands](#makefile-commands)
9. [Security](#security)
10. [Caching](#caching)
11. [Helm Chart](#helm-chart)
12. [Scripts](#scripts)
13. [Contributing](#contributing)

## Features

- Health check endpoint verifying Kubernetes connectivity
- Get and set the replica count of a deployment
- List all deployments in a namespace or across all namespaces
- Secure mTLS communication
- Efficient caching of deployment information
- Graceful shutdown handling
- Comprehensive test suite including integration tests
- Helm chart for easy deployment
- Makefile for streamlined development and deployment processes

## Prerequisites

- Docker
- Kind (Kubernetes in Docker)
- kubectl
- Helm

## Setup

1. **Clone the Repository**:
   ```sh
   git clone https://github.com/dereknorrbom/k8s-deployment-scaler.git
   cd k8s-deployment-scaler
   ```

2. **Install Dependencies**:
   ```sh
   make setup
   ```

3. **Build and Deploy**:
   Use the provided Makefile to build the Docker image, generate certificates, create a Kind cluster, and deploy the application:
   ```sh
   make all
   ```

   This command will:
   - Build the Docker image
   - Generate necessary certificates
   - Create a Kind cluster
   - Load the image into the cluster
   - Deploy the application using Helm
   - Set up port forwarding

4. **Manual Setup (if needed)**:
   If you prefer to set up manually or need to perform steps individually:

   a. Build the Docker image:
   ```sh
   make docker-build
   ```

   b. Generate certificates:
   ```sh
   make generate-certs
   ```

   c. Create a Kind cluster:
   ```sh
   make kind-create
   ```

   d. Load the image and deploy:
   ```sh
   make kind-load
   make deploy
   ```

   e. Set up port forwarding:
   ```sh
   make port-forward
   ```

## Running the Application

Once setup is complete and port forwarding is active, the API will be available at `https://localhost:8443`.

## API Endpoints

- **Health Check**: `GET /healthz`
  - **Example:** 
    ```sh
    curl -X GET "https://localhost:8443/healthz" --cert ./certs/client-cert.pem --key ./certs/client-key.pem --cacert ./certs/ca-cert.pem
    ```
- **Get Replica Count**: `GET /replica-count?namespace=<namespace>&deployment=<deployment>`
  - **Example:** 
    ```sh
    curl -X GET "https://localhost:8443/replica-count?namespace=k8s-deployment-scaler&deployment=k8s-deployment-scaler" --cert ./certs/client-cert.pem --key ./certs/client-key.pem --cacert ./certs/ca-cert.pem
    ```
- **Set Replica Count**: `POST /replica-count?namespace=<namespace>&deployment=<deployment>`
  - **Example:** 
    ```sh
    curl -X POST -H "Content-Type: application/json" -d '{"replicas": 5}' "https://localhost:8443/replica-count?namespace=k8s-deployment-scaler&deployment=k8s-deployment-scaler" --cert ./certs/client-cert.pem --key ./certs/client-key.pem --cacert ./certs/ca-cert.pem
    ```
- **List Deployments**: `GET /deployments?namespace=<namespace>` (namespace is optional)
  - **Example:** 
    ```sh
    curl -X GET "https://localhost:8443/deployments?namespace=k8s-deployment-scaler" --cert ./certs/client-cert.pem --key ./certs/client-key.pem --cacert ./certs/ca-cert.pem
    ```

## Testing
Run the Go test suite, including unit tests for API endpoints, middleware, and helper functions:
```sh
make test
```

### Integration Tests

Integration tests are located in the `integration_tests` directory. These tests verify the functionality of the API endpoints and ensure proper interaction with the Kubernetes cluster. These tests must be run after the application is deployed.

Run integration tests:
```sh
make integration-test
```

Individual endpoint tests:
```sh
make test-health
make test-get-replica-count
make test-set-replica-count
make test-get-deployments
```

## Cleanup

To remove the Kind cluster and clean up resources:
```sh
make teardown
```

## Makefile Commands

- `make all`: Build, generate certs, create cluster, and deploy
- `make setup`: Install dependencies
- `make docker-build`: Build the Docker image
- `make generate-certs`: Generate necessary certificates
- `make kind-create`: Create a Kind cluster
- `make kind-load`: Load the Docker image into the Kind cluster
- `make deploy`: Deploy the application using Helm
- `make helm-upgrade`: Upgrade the Helm chart
- `make port-forward`: Set up port forwarding
- `make clean`: Remove generated files and certificates
- `make teardown`: Full cleanup including Kind cluster deletion
- `make test`: Run the Go test suite
- `make integration-test`: Run integration tests

## Security

This application uses mTLS for secure communication. Certificates are generated during the build process and stored in the `certs` directory.

## Caching

The application implements an efficient caching mechanism using Kubernetes informers to keep deployment information up-to-date and serve read requests quickly without querying the Kubernetes API for every request.

## Helm Chart

The application is deployed using a Helm chart located in the `helm/k8s-deployment-scaler` directory. The chart includes:

- **Deployment:** Defines the deployment configuration for the application pods.
- **Service:** Exposes the application's API endpoints through a Kubernetes service.
- **ServiceAccount:** Provides a dedicated service account for the application to interact with the Kubernetes API.
- **ClusterRole and ClusterRoleBinding:** Defines the permissions required for the application to access and manage deployments.

## Scripts

The `scripts` directory contains helper scripts for setting up the environment and generating certificates:

- `setup.sh`: Installs prerequisites (kubectl, Helm, Docker)
- `generate_certs.sh`: Generates necessary certificates for mTLS

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

- **Fork the repository.**
- **Create a new branch for your feature or bug fix.**
- **Make your changes and commit them.**
- **Push your changes to your fork.**
- **Submit a pull request to the main repository.**

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
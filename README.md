# Kubernetes Deployment Scaler API

This project implements a secure API to manage Kubernetes deployments. The API is built with Go and uses mutual TLS (mTLS) for secure communication. It provides endpoints to manage deployment replica counts and list deployments, with efficient caching for read operations.

## Features

- Health check endpoint verifying Kubernetes connectivity
- Get and set the replica count of a deployment
- List all deployments in a namespace or across all namespaces
- Secure mTLS communication
- Efficient caching of deployment information
- Graceful shutdown handling
- Comprehensive test suite

## Prerequisites

- Docker
- Kind (Kubernetes in Docker)
- kubectl

## Setup

1. **Clone the Repository**:
   ```sh
   git clone https://github.com/dereknorrbom/k8s-deployment-scaler.git
   cd k8s-deployment-scaler
   ```

2. **Build and Deploy**:
   Use the provided Makefile to build the Docker image, generate certificates, create a Kind cluster, and deploy the application:
   ```sh
   make all
   ```

   This command will:
   - Build the Docker image
   - Generate necessary certificates
   - Create a Kind cluster
   - Load the image into the cluster
   - Deploy the application
   - Set up port forwarding

3. **Manual Setup (if needed)**:
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
   make kind-deploy
   ```

   e. Set up port forwarding:
   ```sh
   make port-forward
   ```

## Running the Application

Once setup is complete and port forwarding is active, the API will be available at `https://localhost:8443`.

## API Endpoints

- **Health Check**: `GET /healthz`
- **Get Replica Count**: `GET /replica-count?namespace=<namespace>&deployment=<deployment>`
- **Set Replica Count**: `POST /replica-count?namespace=<namespace>&deployment=<deployment>`
- **List Deployments**: `GET /deployments?namespace=<namespace>` (namespace is optional)

## Testing

Run the test suite:
```sh
make test
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
- `make docker-build`: Build the Docker image
- `make generate-certs`: Generate necessary certificates
- `make kind-create`: Create a Kind cluster
- `make kind-load`: Load the Docker image into the Kind cluster
- `make kind-deploy`: Deploy the application to the Kind cluster
- `make port-forward`: Set up port forwarding
- `make clean`: Remove generated files and certificates
- `make teardown`: Full cleanup including Kind cluster deletion
- `make test`: Run the Go test suite

## Security

This application uses mTLS for secure communication. Certificates are generated during the build process and stored in the `certs` directory.

## Caching

The application implements an efficient caching mechanism using Kubernetes informers to keep deployment information up-to-date and serve read requests quickly without querying the Kubernetes API for every request.
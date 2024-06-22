# Kubernetes Deployment Scaler API

This project implements a simple API to manage Kubernetes deployments. The API is built with Go and uses mutual TLS (mTLS) for secure communication. The current implementation includes the following features:

- Health check endpoint
- Get and set the replica count of a deployment
- List all deployments in a namespace

## Table of Contents

- [Prerequisites](#prerequisites)
- [Setup](#setup)
- [Running the Application](#running-the-application)
- [API Endpoints](#api-endpoints)
  - [Health Check](#health-check)
  - [Get Replica Count](#get-replica-count)
  - [Set Replica Count](#set-replica-count)
  - [List Deployments](#list-deployments)
- [Example cURL Commands](#example-curl-commands)

## Prerequisites

- Docker

## Setup

1. **Clone the Repository**:
    ```sh
    git clone <repository-url>
    cd k8s-deployment-scaler
    ```

2. **Build and Run the Docker Container**:
    Use the provided Makefile to build and run the Docker container.
    ```sh
    make all
    ```

3. **Generate Certificates**:
    Certificates are generated automatically during the build process within the Docker container.

## Running the Application

Once the Docker container is running, the API will be available at `https://localhost:8080`.

## API Endpoints

### Health Check

- **Endpoint**: `/healthz`
- **Method**: `GET`
- **Response**:
    ```json
    {
        "status": "OK"
    }
    ```

### Get Replica Count

- **Endpoint**: `/replica-count`
- **Method**: `GET`
- **Query Parameters**:
    - `namespace`: The namespace of the deployment
    - `deployment`: The name of the deployment
- **Response**:
    ```json
    {
        "replicaCount": 3
    }
    ```

### Set Replica Count

- **Endpoint**: `/replica-count`
- **Method**: `POST`
- **Query Parameters**:
    - `namespace`: The namespace of the deployment
    - `deployment`: The name of the deployment
- **Request Body**:
    ```json
    {
        "replicas": 3
    }
    ```
- **Response**:
    ```json
    {
        "replicas": 3
    }
    ```

### List Deployments

- **Endpoint**: `/deployments`
- **Method**: `GET`
- **Query Parameters**:
    - `namespace` (optional): The namespace to list deployments from
- **Response**:
    ```json
    {
        "deployments": ["default/my-deployment", "another-namespace/another-deployment"]
    }
    ```

## Example cURL Commands

### Health Check
```sh
curl https://localhost:8080/healthz \
    --cert certs/client-cert.pem \
    --key certs/client-key.pem \
    --cacert certs/ca-cert.pem
```

### Get Replica Count
```sh
curl "https://localhost:8080/replica-count?namespace=namespace&deployment=deployment" \
    --cert certs/client-cert.pem \
    --key certs/client-key.pem \
    --cacert certs/ca-cert.pem
```
### Set Replica Count
```sh
curl -X POST -H "Content-Type: application/json" -d '{"replicas": 3}' \
    "https://localhost:8080/replica-count?namespace=namespace&deployment=deployment" \
    --cert certs/client-cert.pem \
    --key certs/client-key.pem \
    --cacert certs/ca-cert.pem
```

### List Deployments
```sh
curl "https://localhost:8080/deployments?namespace=namespace" \
    --cert certs/client-cert.pem \
    --key certs/client-key.pem \
    --cacert certs/ca-cert.pem
```
```sh
curl "https://localhost:8080/deployments" \
    --cert certs/client-cert.pem \
    --key certs/client-key.pem \
    --cacert certs/ca-cert.pem
```

### Makefile
The Makefile includes several useful commands to streamline the development and deployment process:

- `make all`: Build the Docker image, generate the certificates, and run the application.
- `make docker-build`: Build the Docker image for the application.
- `make generate-certs`: Generate certificates by copying them from the Docker image to the local certs directory.
- `make run`: Run the Docker image.
- `make clean`: Clean up by removing the certs directory.
- `make teardown`: Tear down by removing the Docker image.
- `make test`: Run the test suite.
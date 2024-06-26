.PHONY: all docker-build generate-certs run teardown clean test-health test-get-replica-count test-set-replica-count test-get-deployments test

all: docker-build generate-certs kind-create kind-load kind-deploy port-forward

# Build the Docker image for the application
docker-build:
	@echo "Building Docker image..."
	docker build --target builder --tag k8s-deployment-scaler-builder:latest -f build/package/Dockerfile .
	docker build --tag k8s-deployment-scaler:latest -f build/package/Dockerfile .

# Generate certificates by copying them from the builder image to the local certs directory
generate-certs:
	@echo "Generating certificates..."
	docker create --name temp-container k8s-deployment-scaler-builder:latest
	docker cp temp-container:/app/certs ./
	docker rm temp-container

# Create a KIND cluster
kind-create:
	@echo "Creating KIND cluster..."
	kind create cluster --name k8s-deployment-scaler

# Load the Docker image into KIND
kind-load:
	@echo "Loading Docker image into KIND..."
	kind load docker-image k8s-deployment-scaler:latest --name k8s-deployment-scaler

kind-deploy:
	@echo "Deploying RBAC..."
	kubectl apply -f rbac.yaml
	@echo "Deploying application to KIND..."
	kubectl apply -f k8s-deployment-scaler.yaml
	@echo "Waiting for deployment to be ready..."
	kubectl wait --for=condition=available --timeout=60s deployment/k8s-deployment-scaler

port-forward:
	@echo "Waiting for pod to be ready..."
	kubectl wait --for=condition=ready pod -l app=k8s-deployment-scaler --timeout=120s
	kubectl port-forward svc/k8s-deployment-scaler 8443:8443

# Delete the KIND cluster
kind-delete:
	@echo "Deleting KIND cluster..."
	kind delete cluster --name k8s-deployment-scaler

# Clean up by removing the certs directory
clean:
	@echo "Cleaning up..."
	rm -rf ./certs
	rm -f ./k8s-deployment-scaler

# Tear down by removing the Docker image
teardown: clean kind-delete
	@echo "Tearing down..."
	@docker stop $$(docker ps -q --filter ancestor=k8s-deployment-scaler:latest) 2>/dev/null || true
	@docker rm $$(docker ps -aq --filter ancestor=k8s-deployment-scaler:latest) 2>/dev/null || true
	@docker rm temp-container 2>/dev/null || true
	docker rmi k8s-deployment-scaler:latest

# Check the health of the application
test-health:
	@echo "Running test to check health..."
	curl https://localhost:8443/healthz \
    --cert certs/client-cert.pem \
    --key certs/client-key.pem \
    --cacert certs/ca-cert.pem

# Get the replica count of the deployment
test-get-replica-count:
	@echo "Running test to get replica count..."
	curl "https://localhost:8443/replica-count?namespace=default&deployment=k8s-deployment-scaler" \
    --cert certs/client-cert.pem \
    --key certs/client-key.pem \
    --cacert certs/ca-cert.pem

# Set the replica count of the deployment
test-set-replica-count:
	@echo "Running test to set replica count..."
	curl -X POST -H "Content-Type: application/json" -d '{"replicas": 3}' \
    "https://localhost:8443/replica-count?namespace=default&deployment=k8s-deployment-scaler" \
    --cert certs/client-cert.pem \
    --key certs/client-key.pem \
    --cacert certs/ca-cert.pem

# Get the deployments
test-get-deployments:
	@echo "Running test to get deployments..."
	curl "https://localhost:8443/deployments" \
    --cert certs/client-cert.pem \
    --key certs/client-key.pem \
    --cacert certs/ca-cert.pem

# Run the test suite
test:
	@echo "Running test suite..."
	go test -v ./cmd/k8s-deployment-scaler
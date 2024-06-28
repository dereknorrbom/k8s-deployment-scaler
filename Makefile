# Define common variables
APP_NAME := k8s-deployment-scaler
KUBE_NAMESPACE := $(APP_NAME)
DOCKER_IMAGE := $(APP_NAME)
DOCKER_TAG := latest
BUILDER_IMAGE := $(DOCKER_IMAGE)-builder:$(DOCKER_TAG)
FINAL_IMAGE := $(DOCKER_IMAGE):$(DOCKER_TAG)
KIND_CLUSTER_NAME := $(DOCKER_IMAGE)
CERT_DIR := ./certs
CURL_CERT_ARGS := --cert $(CERT_DIR)/client-cert.pem --key $(CERT_DIR)/client-key.pem --cacert $(CERT_DIR)/ca-cert.pem
BASE_URL := https://localhost:8443

.PHONY: all docker-build generate-certs kind-create kind-load deploy port-forward test-health test-get-replica-count test-set-replica-count test-get-deployments integration-test test

setup:
	@echo "Setting up dependencies..."
	@chmod +x scripts/setup.sh
	@bash ./scripts/setup.sh

# Build and deploy the application
all: docker-build generate-certs kind-create kind-load deploy port-forward

# Build the Docker image for the application
docker-build:
	@echo "Building Docker image..."
	docker build --target builder --tag $(BUILDER_IMAGE) -f build/package/Dockerfile .
	docker build --tag $(FINAL_IMAGE) -f build/package/Dockerfile .

# Generate certificates by copying them from the builder image to the local certs directory
generate-certs:
	@echo "Generating certificates..."
	docker create --name temp-container $(BUILDER_IMAGE)
	docker cp temp-container:/app/certs ./
	docker rm temp-container

# Create a KIND cluster
kind-create:
	@echo "Creating KIND cluster..."
	kind create cluster --name $(KIND_CLUSTER_NAME)

# Load the Docker image into KIND
kind-load:
	@echo "Loading Docker image into KIND..."
	kind load docker-image $(FINAL_IMAGE) --name $(KIND_CLUSTER_NAME)

# Deploy the Helm chart
deploy:
	@echo "Deploying Helm chart..."
	helm upgrade --install $(DOCKER_IMAGE) ./helm/$(APP_NAME) --namespace $(KUBE_NAMESPACE) --create-namespace
	@echo "Waiting for deployment to be ready..."
	kubectl rollout status deployment/$(APP_NAME) --namespace $(KUBE_NAMESPACE) --timeout=120s

# Upgrade the Helm chart
helm-upgrade:
	@echo "Upgrading Helm chart..."
	helm upgrade $(DOCKER_IMAGE) ./helm/$(APP_NAME)

# Port forward the service to the local machine
port-forward:
	@echo "Waiting for pod to be ready..."
	kubectl wait --for=condition=ready pod -l app.kubernetes.io/name=$(APP_NAME) --namespace $(KUBE_NAMESPACE) --timeout=120s
	POD_NAME=$$(kubectl get pods -l app.kubernetes.io/name=$(APP_NAME) -o jsonpath="{.items[0].metadata.name}" --namespace $(KUBE_NAMESPACE)) && \
	kubectl port-forward pod/$$POD_NAME 8443:8443 --namespace $(KUBE_NAMESPACE)

# Delete the KIND cluster
kind-delete:
	@echo "Deleting KIND cluster..."
	kind delete cluster --name $(KIND_CLUSTER_NAME)

# Clean up by removing the certs directory
clean:
	@echo "Cleaning up..."
	rm -rf ./certs
	rm -f ./$(APP_NAME)

# Tear down by removing the Docker image
teardown: clean kind-delete
	@echo "Tearing down..."
	@docker stop $$(docker ps -q --filter ancestor=$(FINAL_IMAGE)) 2>/dev/null || true
	@docker rm $$(docker ps -aq --filter ancestor=$(FINAL_IMAGE)) 2>/dev/null || true
	@docker rm temp-container 2>/dev/null || true
	docker rmi $(FINAL_IMAGE)

# Define a function for curl commands
define curl_command
	curl -X $(1) $(BASE_URL)$(2) \
		$(CURL_CERT_ARGS)
endef

# Check the health of the application
test-health:
	@echo "Running test to check health..."
	$(call curl_command,GET,/healthz)

# Get the replica count of the deployment
test-get-replica-count:
	@echo "Running test to get replica count..."
	$(call curl_command,GET,/replica-count?namespace=$(KUBE_NAMESPACE)\&deployment=$(APP_NAME))

# Set the replica count of the deployment
test-set-replica-count:
	@echo "Running test to set replica count..."
	$(call curl_command,POST,/replica-count?namespace=$(KUBE_NAMESPACE)\&deployment=$(APP_NAME) -H "Content-Type: application/json" -d '{"replicas": 3}')

# Get the deployments
test-get-deployments:
	@echo "Running test to get deployments..."
	$(call curl_command,GET,/deployments)

# Run all integration tests
integration-test:
	@echo "Running Go integration tests..."
	kind get kubeconfig --name $(KIND_CLUSTER_NAME) > ~/.kube/config
	KUBECONFIG=$(HOME)/.kube/config go test -v ./integration_tests

# Run the test suite
test:
	@echo "Running test suite..."
	go test -v ./...
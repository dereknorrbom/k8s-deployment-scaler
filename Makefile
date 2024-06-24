.PHONY: all docker-build generate-certs run teardown clean test

all: docker-build generate-certs run

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

# Run the Docker image
run:
	@echo "Running Docker image..."
	docker run -p 8443:8443 k8s-deployment-scaler:latest

# Clean up by removing the certs directory
clean:
	@echo "Cleaning up..."
	rm -rf ./certs
	rm -f ./k8s-deployment-scaler

# Tear down by removing the Docker image
teardown: clean
	@echo "Tearing down..."
	@docker stop $$(docker ps -q --filter ancestor=k8s-deployment-scaler:latest) 2>/dev/null || true
	@docker rm $$(docker ps -aq --filter ancestor=k8s-deployment-scaler:latest) 2>/dev/null || true
	@docker rm temp-container 2>/dev/null || true
	docker rmi k8s-deployment-scaler:latest

# Run the test suite
test:
	@echo "Running test suite..."
	go test -v ./cmd/k8s-deployment-scaler
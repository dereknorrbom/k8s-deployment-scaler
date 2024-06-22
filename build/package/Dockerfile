# syntax=docker/dockerfile:1

FROM golang:1.22-alpine

# Set the working directory inside the container
WORKDIR /app

# Install necessary packages including OpenSSL and CA certificates
RUN apk --no-cache add openssl ca-certificates

# Copy the go.mod and go.sum files to the work directory
COPY go.mod go.sum ./

# Generate certificates
COPY scripts ./scripts
RUN chmod +x ./scripts/generate_certs.sh
RUN sh ./scripts/generate_certs.sh

# Download dependencies
RUN go mod tidy

# Copy the source code
COPY cmd/k8s-deployment-scaler .

# Build the Go application
RUN go build -o k8s-deployment-scaler

# Command to run the built binary
CMD ["/app/k8s-deployment-scaler"]
# syntax=docker/dockerfile:1

# Stage 1: Build the Go binary
FROM golang:1.22-alpine as builder

# Set the working directory inside the container
WORKDIR /app

# Install necessary packages including OpenSSL
RUN apk --no-cache add openssl

# Copy the go.mod and go.sum files to the work directory
COPY go.mod go.sum ./

# Download dependencies
RUN go mod tidy

# Copy the source code
COPY cmd/k8s-deployment-scaler .

# Build the Go application
RUN go build -o k8s-deployment-scaler

# Generate certificates
COPY scripts ./scripts
RUN chmod +x ./scripts/generate_certs.sh
RUN sh ./scripts/generate_certs.sh

# Stage 2: Create the final image
FROM scratch

WORKDIR /app

# Copy only the necessary server certificates from the builder stage
COPY --from=builder /app/certs/server-cert.pem /app/certs/server-key.pem /app/certs/ca-cert.pem ./certs/

# Copy the built Go binary from the builder stage
COPY --from=builder /app/k8s-deployment-scaler .

# Command to run the built binary
CMD ["/app/k8s-deployment-scaler"]

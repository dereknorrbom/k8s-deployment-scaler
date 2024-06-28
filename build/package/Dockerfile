# syntax=docker/dockerfile:1

# Stage 1: Build the Go binary
FROM golang:1.22-alpine AS builder

WORKDIR /app

# Install necessary packages
RUN apk add --no-cache openssl ca-certificates

# Copy go.mod and go.sum first to leverage Docker cache
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the project
COPY . .

# Generate certificates using the script
RUN chmod +x ./scripts/generate_certs.sh && \
    sh ./scripts/generate_certs.sh && \
    chmod 644 /app/certs/server-cert.pem /app/certs/ca-cert.pem && \
    chmod 600 /app/certs/server-key.pem

# Build the Go application with static linking
RUN CGO_ENABLED=0 GOOS=linux go build -o k8s-deployment-scaler ./cmd/k8s-deployment-scaler

# Stage 2: Create the final image
FROM gcr.io/distroless/static-debian11:nonroot
WORKDIR /app

# Copy the necessary files from the builder stage
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder --chown=nonroot:nonroot /app/certs/server-cert.pem /app/certs/server-key.pem /app/certs/ca-cert.pem ./certs/
COPY --from=builder /app/k8s-deployment-scaler .

EXPOSE 8443

CMD ["/app/k8s-deployment-scaler"]
#!/bin/bash

# Define directory and file paths
CERTS_DIR="certs"
CA_KEY="$CERTS_DIR/ca-key.pem"
CA_CERT="$CERTS_DIR/ca-cert.pem"
SERVER_KEY="$CERTS_DIR/server-key.pem"
SERVER_CERT="$CERTS_DIR/server-cert.pem"
CLIENT_KEY="$CERTS_DIR/client-key.pem"
CLIENT_CERT="$CERTS_DIR/client-cert.pem"

# Create certs directory if it doesn't exist
mkdir -p $CERTS_DIR

# Check if CA key and certificate exist
if [ ! -f "$CA_KEY" ] || [ ! -f "$CA_CERT" ]; then
    echo "Generating CA key and certificate..."
    openssl genpkey -algorithm RSA -out $CA_KEY
    openssl req -x509 -new -key $CA_KEY -out $CA_CERT -days 365 -subj "/C=US/ST=State/L=City/O=Organization/OU=Unit/CN=example.com"
else
    echo "CA key and certificate already exist."
fi

# Check if server key and certificate exist
if [ ! -f "$SERVER_KEY" ] || [ ! -f "$SERVER_CERT" ]; then
    echo "Generating server key and certificate..."
    openssl genpkey -algorithm RSA -out $SERVER_KEY
    openssl req -new -key $SERVER_KEY -out $CERTS_DIR/server.csr -subj "/C=US/ST=State/L=City/O=Organization/OU=Unit/CN=localhost"
    openssl x509 -req -in $CERTS_DIR/server.csr -CA $CA_CERT -CAkey $CA_KEY -CAcreateserial -out $SERVER_CERT -days 365 -extfile <(printf "subjectAltName=DNS:localhost,IP:127.0.0.1")
    rm $CERTS_DIR/server.csr
else
    echo "Server key and certificate already exist."
fi

# Check if client key and certificate exist
if [ ! -f "$CLIENT_KEY" ] || [ ! -f "$CLIENT_CERT" ]; then
    echo "Generating client key and certificate..."
    openssl genpkey -algorithm RSA -out $CLIENT_KEY
    openssl req -new -key $CLIENT_KEY -out $CERTS_DIR/client.csr -subj "/C=US/ST=State/L=City/O=Organization/OU=Unit/CN=client"
    openssl x509 -req -in $CERTS_DIR/client.csr -CA $CA_CERT -CAkey $CA_KEY -CAcreateserial -out $CLIENT_CERT -days 365 -extfile <(printf "subjectAltName=DNS:client,IP:127.0.0.1")
    rm $CERTS_DIR/client.csr
else
    echo "Client key and certificate already exist."
fi

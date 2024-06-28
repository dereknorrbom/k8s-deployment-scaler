#!/bin/bash

set -e

# Function to check if a command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Function to install kubectl
install_kubectl() {
    echo "Installing kubectl..."
    curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/$(uname -s | tr '[:upper:]' '[:lower:]')/amd64/kubectl"
    chmod +x ./kubectl
    sudo mv ./kubectl /usr/local/bin/kubectl
}

# Function to install Helm
install_helm() {
    echo "Installing Helm..."
    curl https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 | bash
}

# Function to install Docker
install_docker() {
    if [[ "$OSTYPE" == "linux-gnu"* ]]; then
        echo "Installing Docker on Linux..."
        sudo apt-get update
        sudo apt-get install -y docker.io
        sudo systemctl start docker
        sudo systemctl enable docker
    elif [[ "$OSTYPE" == "darwin"* ]]; then
        echo "Installing Docker on macOS..."
        if command_exists brew; then
            brew install docker
            echo "Docker installed. To start the Docker daemon, run: "
            echo "docker-machine create --driver virtualbox default"
            echo "eval \$(docker-machine env default)"
        else
            echo "Homebrew is not installed. Please install Homebrew first or Docker Desktop manually."
            echo "Homebrew: https://brew.sh/"
            echo "Docker Desktop: https://www.docker.com/products/docker-desktop"
            exit 1
        fi
    else
        echo "Unsupported operating system"
        exit 1
    fi
}

# Check and install kubectl
if ! command_exists kubectl; then
    install_kubectl
else
    echo "kubectl is already installed"
fi

# Check and install Helm
if ! command_exists helm; then
    install_helm
else
    echo "Helm is already installed"
fi

# Check and install Docker
if ! command_exists docker; then
    install_docker
else
    echo "Docker is already installed"
fi

echo "Setup complete. kubectl, Helm, and Docker are now installed."
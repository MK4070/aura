#!/usr/bin/env bash

set -Eeuo pipefail

IMAGE_NAME="aura-ingestion:latest"
OLLAMA_CONTAINER="aura-ollama"
LLM_MODEL="llama3.2:3b"
EMBEDDING_MODEL="nomic-embed-text"

cleanup() {
    # Un-register the trap immediately so it only fires once
    trap - EXIT INT TERM

    echo ""
    echo "========================================"
    echo "Stopping Python backend cleanly..."
    
    # Give FastAPI/Uvicorn 3 seconds to close its connections
    sleep 3

    echo "Tearing down Aura Stack..."
    make -s down-obs || true
    make -s down-core || true
    
    echo "========================================"
    echo "Aura Stack teardown complete."
}

trap cleanup EXIT INT TERM

check_requirements() {
    command -v docker >/dev/null || {
        echo "Docker is not installed."
        exit 1
    }

    command -v make >/dev/null || {
        echo "make is not installed."
        exit 1
    }

    command -v uv >/dev/null || {
        echo "uv (Python package manager) is not installed."
        echo "Please install it by running: curl -LsSf https://astral.sh/uv/install.sh | sh"
        exit 1
    }
}

check_docker() {
    echo "Checking Docker daemon status..."

    if ! docker info >/dev/null 2>&1; then
        echo "Docker is not running."
        exit 1
    fi

    echo "Docker is running."
}

build_ingestion() {
    if docker image inspect "$IMAGE_NAME" >/dev/null 2>&1 \
        && [ "${FORCE_BUILD:-0}" != "1" ]; then
        echo "Using existing image: $IMAGE_NAME"
    else
        echo "Building ingestion image..."
        (cd services/ingestion && make -s docker-build)
    fi
}

start_services() {
    make -s up-core

    echo "Checking Ollama container models..."
    
    # Wait for the Ollama daemon to become responsive inside the container
    local retry_count=0
    until docker exec "$OLLAMA_CONTAINER" ollama list >/dev/null 2>&1; do
        if [ $retry_count -eq 0 ]; then
            echo "Waiting for Ollama daemon to initialize..."
        fi
        sleep 2
        retry_count=$((retry_count + 1))
        if [ $retry_count -gt 10 ]; then
            echo "Error: Ollama daemon failed to start within 20 seconds."
            exit 1
        fi
    done

    # Verify if both required models exist in the container
    if docker exec "$OLLAMA_CONTAINER" ollama list | grep -q "$EMBEDDING_MODEL" && \
       docker exec "$OLLAMA_CONTAINER" ollama list | grep -q "$LLM_MODEL"; then
        echo "Required models are already present."
    else
        echo "Models missing or incomplete. Initializing pull sequence..."
        make -s pull-models
    fi

    make -s up-obs
}

start_backend() {
    echo "Starting Python backend..."
    echo "Press Ctrl+C to stop the Python server and tear down the stack."
    echo "========================================"
    (cd services/query && make -s dev)
}

main() {
    echo "========================================"
    echo "Booting Aura Stack..."
    echo "========================================"

    check_requirements
    check_docker
    build_ingestion
    start_services
    start_backend
}

main "$@"
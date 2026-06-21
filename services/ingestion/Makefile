BUILD_DIR := bin
# APP_NAME := uploader
# MAIN_PKG := ./cmd/api
APP_NAME := processor
MAIN_PKG := ./cmd/worker
IMAGE_NAME := aura-ingestion
IMAGE_TAG := latest

.PHONY: all docker-build build run test clean fmt vet tidy help

# Default target when you just type 'make'
all: help

docker-build: ## Builds the multi-stage Docker image
	docker build -t $(IMAGE_NAME):$(IMAGE_TAG) .

build: ## Compile the Go binary
	@echo "Building $(APP_NAME)..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 go build -o $(BUILD_DIR)/$(APP_NAME) $(MAIN_PKG)

run: tidy fmt vet ## Run the application directly without building the binary
	@echo "Running the server..."
	go run $(MAIN_PKG)

test: ## Run all Go tests in the project
	@echo "Running tests..."
	go test -v ./...

clean: ## Remove build artifacts
	@echo "Cleaning..."
	$(GO_CLEAN)
	rm -rf $(BUILD_DIR)

fmt: ## Run go fmt against code
	@echo "Formatting code..."
	go fmt ./...

vet: ## Run go vet against code to catch common errors
	@echo "Vetting code..."
	go vet ./...

tidy: ## Format go.mod and remove unused dependencies
	@echo "Tidying module requirements..."
	go mod tidy

help: ## Show this help menu
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'
# Specify the compose files and environment files
COMPOSE_CORE := docker-compose --env-file deploy/.env.core -f deploy/docker-compose.yml
COMPOSE_OBS  := docker-compose --env-file deploy/.env.obs -f deploy/docker-compose.obs.yml

# Ollama models to pull
LLM_MODEL := llama3.2:3b
EMBEDDING_MODEL := nomic-embed-text

all: help
# ==============================================================================
# Core Infrastructure

.PHONY: up-core
up-core: ## Start the core infrastructure in the background
	@echo "Starting core infrastructure..."
	$(COMPOSE_CORE) up -d

.PHONY: down-core
down-core: ## Stop and remove core infrastructure containers
	@echo "Stopping core infrastructure..."
	$(COMPOSE_CORE) down

.PHONY: logs-core
logs-core: ## Tail the logs for the core infrastructure
	$(COMPOSE_CORE) logs -f

# ==============================================================================
# Observability Infrastructure (OpenTelemetry, Prometheus, Grafana)

.PHONY: up-obs
up-obs: ## Start the observability stack in the background
	@echo "Starting observability stack..."
	$(COMPOSE_OBS) up -d

.PHONY: down-obs
down-obs: ## Stop and remove observability containers
	@echo "Stopping observability stack..."
	$(COMPOSE_OBS) down

.PHONY: logs-obs
logs-obs: ## Tail the logs for the observability stack
	$(COMPOSE_OBS) logs -f

# ==============================================================================
# AI Model Management

.PHONY: pull-models
pull-models: ## Pull the required LLM and Embedding models into Ollama
	@echo "Pulling embedding model: $(EMBEDDING_MODEL)..."
	docker exec -it aura-ollama ollama pull $(EMBEDDING_MODEL)
	@echo "Pulling generation model: $(LLM_MODEL)..."
	docker exec -it aura-ollama ollama pull $(LLM_MODEL)
	@echo "Models pulled successfully."

.PHONY: init
init: up-core ## Spin up core infra and pull models (Wait a few seconds for Ollama to boot before pulling)
	@echo "Waiting 10 seconds for Ollama to initialize before pulling models..."
	@sleep 10
	@$(MAKE) pull-models
	@echo "Initialization complete. Ready for development."

.PHONY: down-all
down-all: down-obs down-core ## Tear down everything
	@echo "All infrastructure stopped."

# ==============================================================================
.PHONY: clean
clean: ## Tear down everything AND remove named volumes (WARNING: destroys data)
	@echo "Destroying infrastructure and volumes..."
	$(COMPOSE_OBS) down -v
	$(COMPOSE_CORE) down -v

.PHONY: help
help: ## Show this help menu
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'
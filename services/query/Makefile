.PHONY: setup dev lint format typecheck test clean

all: help

setup: ## Install dependencies and sync the uv virtual environment
	uv sync --all-extras

dev: ## Run the FastAPI server with hot-reload
	uv run uvicorn app.main:app --reload --host 0.0.0.0 --port 8000

lint: ## Run the Ruff linter
	uv run ruff check .

format: ## Auto-format code using Ruff
	uv run ruff format .
	uv run ruff check --select I --fix .  # Fix import sorting

typecheck: ## Run static type checking
	uv run mypy app/

test: ## Run the test suite
	uv run pytest tests/ -v

clean: ## Clean up cache directories
	find . -type d -name "__pycache__" -exec rm -rf {} +
	find . -type d -name ".ruff_cache" -exec rm -rf {} +
	find . -type d -name ".mypy_cache" -exec rm -rf {} +
	find . -type d -name ".pytest_cache" -exec rm -rf {} +

help: ## Show this help menu
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'
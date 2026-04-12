.PHONY: dev build clean sqlc lint typecheck test test-unit test-e2e hooks hooks-fallback help

# ========================
# Development
# ========================

dev: ## Start Wails dev mode with hot-reload
	wails dev

build: ## Build production binary
	wails build

clean: ## Remove build artifacts
	rm -rf build/bin
	rm -rf frontend/dist
	rm -f oversite

# ========================
# Code Generation
# ========================

sqlc: ## Regenerate Go code from SQL queries
	go tool sqlc generate

# ========================
# Quality
# ========================

lint: ## Run all linters
	go tool golangci-lint run ./...
	cd frontend && pnpm lint

typecheck: ## Run TypeScript type checking
	cd frontend && pnpm typecheck

# ========================
# Testing
# ========================

test: test-unit test-e2e ## Run all tests

test-unit: ## Run Go + TS unit tests
	go test -race ./...
	cd frontend && pnpm test

test-e2e: ## Run Playwright E2E tests
	cd e2e && npx playwright test

# ========================
# Git Hooks
# ========================

hooks: ## Install pre-commit hooks via lefthook
	go tool lefthook install

hooks-fallback: ## Install pre-commit hooks (no extra tools)
	git config core.hooksPath .githooks
	@echo "Pre-commit hooks activated via core.hooksPath"

# ========================
# Help
# ========================

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

.DEFAULT_GOAL := help

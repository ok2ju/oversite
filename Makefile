.PHONY: dev build clean sqlc migrate-create lint typecheck test test-go test-fe test-unit test-e2e hooks hooks-fallback help

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

migrate-create: ## Create new migration pair (usage: make migrate-create name=<name>)
	@test -n "$(name)" || (echo "Usage: make migrate-create name=<migration_name>" && exit 1)
	@next=$$(printf "%03d" $$(($$(ls migrations/*.up.sql 2>/dev/null | wc -l | tr -d ' ') + 1))); \
	touch "migrations/$${next}_$(name).up.sql" "migrations/$${next}_$(name).down.sql"; \
	echo "Created migrations/$${next}_$(name).up.sql"; \
	echo "Created migrations/$${next}_$(name).down.sql"

# ========================
# Quality
# ========================

lint: ## Run all linters
	golangci-lint run ./...
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

test-go: ## Run Go tests only
	go test -race ./...

test-fe: ## Run frontend tests only
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

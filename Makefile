.PHONY: up down dev logs certs migrate-up migrate-down migrate-create sqlc lint test test-unit test-integration test-e2e build clean hooks hooks-fallback help

# ========================
# Docker
# ========================

up: certs ## Start all services in background
	docker compose up -d

down: ## Stop all services
	docker compose down

dev: certs ## Start with hot-reload (foreground)
	docker compose -f docker-compose.yml -f docker-compose.dev.yml up

logs: ## Tail logs (use: make logs s=api)
	docker compose logs -f $(s)

ps: ## Show running services
	docker compose ps

restart: ## Restart a service (use: make restart s=api)
	docker compose restart $(s)

# ========================
# TLS Certs
# ========================

certs: ## Generate local TLS certs (requires mkcert)
	@if [ ! -f nginx/certs/localhost.pem ]; then \
		mkdir -p nginx/certs; \
		mkcert -install 2>/dev/null || true; \
		mkcert -cert-file nginx/certs/localhost.pem -key-file nginx/certs/localhost-key.pem localhost 127.0.0.1 ::1; \
		echo "TLS certs generated in nginx/certs/"; \
	else \
		echo "TLS certs already exist, skipping."; \
	fi

# ========================
# Database
# ========================

migrate-up: ## Run all pending migrations
	docker compose exec api /app/oversite migrate up

migrate-down: ## Rollback the last migration
	docker compose exec api /app/oversite migrate down

migrate-create: ## Create new migration files
	@read -p "Migration name: " name; \
	docker compose exec api /app/oversite migrate create $$name

sqlc: ## Regenerate Go code from SQL queries
	cd backend && go tool sqlc generate

# ========================
# Quality
# ========================

lint: ## Run all linters
	cd backend && go tool golangci-lint run ./...
	cd frontend && pnpm lint

typecheck: ## Run TypeScript type checking
	cd frontend && pnpm tsc --noEmit

hooks: ## Install pre-commit hooks
	go -C backend tool lefthook install

hooks-fallback: ## Install pre-commit hooks (no extra tools)
	git config core.hooksPath .githooks
	@echo "Pre-commit hooks activated via core.hooksPath"

# ========================
# Testing
# ========================

test: test-unit test-integration test-e2e ## Run all tests

test-unit: ## Run Go + TS unit tests
	cd backend && go test ./...
	cd frontend && pnpm test

test-integration: ## Run Go integration tests (requires Docker)
	cd backend && go test -tags integration -count=1 ./...

test-e2e: ## Run Playwright E2E tests
	cd e2e && npx playwright test

# ========================
# Build
# ========================

build: ## Build all artifacts
	cd backend && go build -o bin/oversite ./cmd/oversite
	cd frontend && pnpm build

clean: ## Remove build artifacts
	rm -rf backend/bin backend/tmp
	rm -rf frontend/.next frontend/out

# ========================
# Help
# ========================

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

.DEFAULT_GOAL := help

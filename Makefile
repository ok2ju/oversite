.PHONY: dev build clean sqlc migrate-create gen-appicon lint typecheck test test-go test-fe test-unit test-e2e hooks hooks-fallback db-reset help

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

gen-appicon: ## Regenerate build/appicon.png from the reticle mark
	go run ./cmd/gen-appicon

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
	lefthook install

hooks-fallback: ## Install pre-commit hooks (no extra tools)
	git config core.hooksPath .githooks
	@echo "Pre-commit hooks activated via core.hooksPath"

# ========================
# Database
# ========================

db-reset: ## Wipe all data from the local SQLite DB (keeps schema and migration history)
	@case "$$(uname -s)" in \
	  Darwin) db="$$HOME/Library/Application Support/oversite/oversite.db" ;; \
	  Linux)  db="$${XDG_DATA_HOME:-$$HOME/.local/share}/oversite/oversite.db" ;; \
	  *)      echo "Unsupported OS: $$(uname -s)"; exit 1 ;; \
	esac; \
	if [ ! -f "$$db" ]; then echo "No database at $$db"; exit 0; fi; \
	if pgrep -x oversite >/dev/null 2>&1; then echo "Refusing: oversite is running. Quit the app first."; exit 1; fi; \
	echo "Wiping data from $$db"; \
	stmts=$$(sqlite3 "$$db" "SELECT 'DELETE FROM ' || quote(name) || ';' FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%' AND name <> 'schema_migrations';"); \
	sqlite3 "$$db" "PRAGMA foreign_keys=OFF; BEGIN; $$stmts COMMIT; VACUUM;"; \
	echo "Done."

# ========================
# Help
# ========================

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

.DEFAULT_GOAL := help

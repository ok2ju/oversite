.PHONY: up down dev logs migrate-up migrate-down migrate-create sqlc lint test build

# Docker
up:
	docker compose up -d

down:
	docker compose down

dev:
	docker compose -f docker-compose.yml -f docker-compose.dev.yml up

logs:
	docker compose logs -f $(s)

# Database
migrate-up:
	docker compose exec api /app/oversite migrate up

migrate-down:
	docker compose exec api /app/oversite migrate down

migrate-create:
	@read -p "Migration name: " name; \
	docker compose exec api /app/oversite migrate create $$name

sqlc:
	cd backend && sqlc generate

# Quality
lint:
	cd backend && golangci-lint run ./...
	cd frontend && pnpm lint

test:
	cd backend && go test ./...
	cd frontend && pnpm test

build:
	cd backend && go build -o cmd/oversite/oversite ./cmd/oversite
	cd frontend && pnpm build

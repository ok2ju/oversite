# Oversite

CS2 2D demo viewer and analytics platform for Faceit players. Upload demos, watch top-down playback, generate heatmaps, collaborate on strategies, and track Faceit stats -- all in the browser.

## Prerequisites

| Tool | Version | Install |
|------|---------|---------|
| [Docker](https://www.docker.com/) | 24+ | [Docker Desktop](https://www.docker.com/products/docker-desktop/) |
| [Docker Compose](https://docs.docker.com/compose/) | 2.20+ | Included with Docker Desktop |
| [Go](https://go.dev/) | 1.22+ | `brew install go` |
| [Node.js](https://nodejs.org/) | 20+ | `brew install node` |
| [pnpm](https://pnpm.io/) | 9+ | `corepack enable && corepack prepare pnpm@latest --activate` |
| [mkcert](https://github.com/FiloSottile/mkcert) | -- | `brew install mkcert` |

You also need a [Faceit Developer](https://developers.faceit.com) account with an OAuth app registered.

> **Why mkcert?** Faceit requires HTTPS redirect URIs, even for localhost. mkcert generates locally-trusted TLS certificates so `https://localhost` works without browser warnings. Certs are auto-generated on first `make dev`.

## Quick Start

### 1. Clone and configure

```bash
git clone git@github.com:ok2ju/oversite.git
cd oversite

# Install frontend dependencies
cd frontend && pnpm install && cd ..

# Install pre-commit hooks
make hooks
```

### 2. Set up environment

```bash
cp .env.example .env
```

Edit `.env` and fill in your Faceit credentials:

```
FACEIT_CLIENT_ID=your_client_id
FACEIT_CLIENT_SECRET=your_client_secret
FACEIT_REDIRECT_URI=https://localhost/api/v1/auth/faceit/callback
FACEIT_API_KEY=your_api_key
```

> Get these by creating an app at [developers.faceit.com](https://developers.faceit.com).
> Set the OAuth redirect URI to `https://localhost/api/v1/auth/faceit/callback`.

### 3. Start the stack

```bash
make dev
```

This automatically generates local TLS certificates (via mkcert) on first run, then starts all services with hot-reload:

| Service | URL | Description |
|---------|-----|-------------|
| nginx | [localhost](https://localhost) | Reverse proxy -- HTTPS entry point |
| web | localhost:3000 | Next.js frontend |
| api | localhost:8080 | Go REST API |
| ws | localhost:8081 | Go WebSocket server |
| worker | -- | Background job processor |
| postgres | localhost:5432 | TimescaleDB database |
| redis | localhost:6379 | Sessions, cache, job queue |
| minio | [localhost:9001](http://localhost:9001) | S3-compatible demo storage (console) |

### 4. Run database migrations

In a separate terminal:

```bash
make migrate-up
```

### 5. Open the app

Visit **https://localhost** and click **Sign in with Faceit**.

## Development Workflows

### Full stack (Docker)

```bash
make dev              # Start everything with hot-reload (foreground)
make up               # Start everything in background
make down             # Stop all services
make logs             # Tail all logs
make logs s=api       # Tail logs for a specific service
make ps               # Show service status
make restart s=api    # Restart a specific service
```

### Frontend only (faster HMR)

Run the backend in Docker and Next.js locally for faster iteration:

```bash
make up                  # Start backend services in background
cd frontend && pnpm dev  # Start Next.js dev server
```

Visit **http://localhost:3000**. API requests are proxied to the Go backend via Next.js rewrites.

> When running this way, update `.env`:
> `FACEIT_REDIRECT_URI=https://localhost/api/v1/auth/faceit/callback`
> The OAuth callback always goes through nginx (HTTPS), even when developing on `:3000`.

### Database

```bash
make migrate-up       # Run all pending migrations
make migrate-down     # Rollback last migration
make migrate-create   # Create new migration files (interactive)
make sqlc             # Regenerate Go code from SQL queries
```

### Testing

```bash
make test             # Run all tests (unit + integration + e2e)
make test-unit        # Go + TypeScript unit tests
make test-integration # Go integration tests (requires Docker)
make test-e2e         # Playwright E2E tests
```

### Linting

```bash
make lint             # Lint Go + TypeScript
make typecheck        # TypeScript type checking
```

## Project Structure

```
oversite/
├── backend/          # Go API, WebSocket server, worker
├── frontend/         # Next.js App Router frontend
├── e2e/              # Playwright E2E tests
├── nginx/            # Reverse proxy config
├── docs/             # PRD, architecture, implementation plan
├── docker-compose.yml
├── docker-compose.dev.yml
└── Makefile
```

See [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) for system design and data flows.

## Tech Stack

| Layer | Technology |
|-------|-----------|
| Frontend | Next.js, TypeScript, PixiJS v8, shadcn/ui, Tailwind, Zustand, TanStack Query |
| Backend | Go, chi router, gorilla/websocket |
| Demo Parsing | demoinfocs-golang v5 |
| Database | PostgreSQL 16 + TimescaleDB |
| Cache/Queue | Redis 7 (sessions, cache, Streams job queue) |
| Storage | MinIO (S3-compatible) |
| Collaboration | Yjs CRDT |
| Auth | Faceit OAuth 2.0 + PKCE |
| Infra | Docker Compose, nginx |

# Oversite

CS2 2D demo viewer and analytics platform for Faceit players. Upload demos, watch top-down playback, generate heatmaps, collaborate on strategies, and track Faceit stats -- all in the browser.

## Prerequisites

Install the following before starting development:

| Tool | Version | Install |
|------|---------|---------|
| [Go](https://go.dev/) | 1.26+ | `brew install go` |
| [Node.js](https://nodejs.org/) | 20+ | `brew install node` |
| [pnpm](https://pnpm.io/) | 9+ | `corepack enable && corepack prepare pnpm@latest --activate` |
| [Docker](https://www.docker.com/) | 24+ | [Docker Desktop](https://www.docker.com/products/docker-desktop/) |
| [Docker Compose](https://docs.docker.com/compose/) | 2.20+ | Included with Docker Desktop |

### Setup

```bash
# Clone the repo
git clone git@github.com:ok2ju/oversite.git
cd oversite

# Install frontend dependencies
cd frontend && pnpm install && cd ..

# Install pre-commit hooks (tools auto-resolved from go.mod)
make hooks

# Start all services (Postgres, Redis, MinIO, etc.)
make up

# Run database migrations
make migrate-up

# Start frontend dev server
cd frontend && pnpm dev
```

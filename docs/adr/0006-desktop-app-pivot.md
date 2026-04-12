# ADR-0006: Pivot from Web Application to Desktop Application

**Date:** 2026-04-12
**Status:** Accepted

## Context

The web-based Oversite application requires 8 Docker services (nginx, Next.js, Go API, Go WebSocket, Go Worker, PostgreSQL + TimescaleDB, Redis, MinIO) to serve even a single user. This imposes significant operational burden:

- Hosting PostgreSQL, Redis, and MinIO costs money and requires maintenance
- Demo upload latency is significant for 100 MB+ files -- users must upload, wait for parsing, then stream tick data back over HTTP
- The multi-user collaboration features (Yjs strategy board, WebSocket viewer sync) saw minimal demand in early feedback; the primary audience is solo grinders reviewing their own demos
- The target audience is CS2 gamers, who universally have capable desktops with GPUs -- a web deployment model underutilizes their hardware

A desktop application eliminates all infrastructure costs, removes upload latency entirely (demos are already local), and simplifies the architecture from 8 services to a single binary.

### Alternatives considered

| Approach | Why rejected |
|----------|-------------|
| **Keep web app, optimize hosting** | Still requires PostgreSQL/Redis/MinIO hosting. Upload latency is fundamental to the web model. Doesn't solve the cost problem for a side project. |
| **Hybrid: desktop parsing + web dashboard** | Two deployment targets to maintain. Sync between local and cloud adds complexity. Doesn't simplify enough to justify the split. |
| **PWA with local-first storage** | IndexedDB has storage limits and performance issues with 1.28M rows per demo. No native filesystem access for `.dem` files without File System Access API (limited browser support). |

## Decision

Pivot Oversite to a native desktop application. Key decisions:

- **Single-user, local-first**: All data stays on the user's machine. No server required.
- **Eliminate cloud infrastructure**: Remove Docker, nginx, Redis, MinIO, PostgreSQL. Replace with SQLite (see [ADR-0008](0008-sqlite-local-database.md)).
- **Retain core stack**: Keep Go backend, React frontend, PixiJS v8 rendering, demo parser (demoinfocs-golang), Faceit OAuth + PKCE, shadcn/ui, Tailwind, Zustand, TanStack Query.
- **Defer collaboration**: Strategy board becomes single-user drawing tool. No Yjs, no WebSocket relay. Collaboration may return in v2 via peer-to-peer or cloud sync.
- **Target platforms**: macOS (12+), Windows (10+), Linux (Ubuntu 22.04+).

## Consequences

### Positive

- Zero infrastructure cost -- no servers to host, monitor, or maintain
- Instant demo parsing -- no upload step; parser reads directly from local filesystem
- Lower latency -- tick data served from local SQLite, not over HTTP
- Simpler architecture -- single binary, single process, no inter-service communication
- Simpler deployment -- distribute a binary, not a Docker Compose stack
- Full access to user's filesystem for demo management (drag-and-drop, auto-scan directories)

### Negative

- No real-time collaboration in v1 -- strategy board is single-user only
- No web sharing -- can't share strategy boards or heatmaps via URL
- Cross-platform testing burden -- must test on macOS, Windows, and Linux
- Requires an auto-updater mechanism for distributing new versions
- Larger initial download (~30 MB binary) vs. zero-install web app

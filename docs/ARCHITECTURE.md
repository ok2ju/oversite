# Oversite -- Architecture Documentation

> **Version:** 1.0
> **Last Updated:** 2026-03-31
> **Format:** arc42

---

## Table of Contents

1. [Introduction & Goals](#1-introduction--goals)
2. [System Context (C4 Level 1)](#2-system-context-c4-level-1)
3. [Container Diagram (C4 Level 2)](#3-container-diagram-c4-level-2)
4. [Component Diagrams (C4 Level 3)](#4-component-diagrams-c4-level-3)
5. [Data Flow Diagrams](#5-data-flow-diagrams)
6. [REST API Specification](#6-rest-api-specification)
7. [WebSocket Protocol](#7-websocket-protocol)
8. [Database Schema](#8-database-schema)
9. [Docker Compose Topology](#9-docker-compose-topology)
10. [CRDT Decision: Yjs vs OT](#10-crdt-decision-yjs-vs-ot)
11. [Cross-Cutting Concerns](#11-cross-cutting-concerns)
12. [Monorepo Directory Structure](#12-monorepo-directory-structure)
13. [Cloud Migration Path](#13-cloud-migration-path)
14. [Testing Architecture](#14-testing-architecture)

---

## 1. Introduction & Goals

### 1.1 Requirements Overview

Oversite is a web-based 2D demo viewer and analytics platform for CS2 Faceit players. Key quality goals:

| Priority | Quality Goal | Motivation |
|----------|-------------|------------|
| 1 | **Performance** | 60 FPS canvas rendering; < 30s demo parse; < 200ms API p95 |
| 2 | **Real-time Collaboration** | < 200ms strat board sync; conflict-free concurrent editing |
| 3 | **Developer Experience** | Monorepo with hot reload, type-safe SQL, shared Docker env |
| 4 | **Cloud Readiness** | Docker Compose now, Kubernetes-ready architecture |

### 1.2 Stakeholders

| Role | Concern |
|------|---------|
| Solo developer | Productive monorepo DX; manageable complexity |
| End users (Faceit players) | Fast, reliable demo review and team collaboration |
| Future contributors | Clear architecture boundaries; documented APIs |

---

## 2. System Context (C4 Level 1)

```
┌─────────────────────────────────────────────────────────┐
│                    External Systems                      │
│                                                         │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐  │
│  │  Faceit API   │  │ Steam (demo  │  │   Browser    │  │
│  │  (OAuth +     │  │  downloads)  │  │   (User)     │  │
│  │   Data API)   │  │              │  │              │  │
│  └──────┬───────┘  └──────┬───────┘  └──────┬───────┘  │
│         │                 │                  │          │
└─────────┼─────────────────┼──────────────────┼──────────┘
          │                 │                  │
          ▼                 ▼                  ▼
┌─────────────────────────────────────────────────────────┐
│                                                         │
│                    O V E R S I T E                       │
│                                                         │
│    Web application for CS2 demo review, analytics,      │
│    real-time strategy collaboration, and Faceit          │
│    stats tracking.                                      │
│                                                         │
└─────────────────────────────────────────────────────────┘
```

### External System Interfaces

| System | Protocol | Purpose |
|--------|----------|---------|
| **Faceit OAuth** | HTTPS (OAuth 2.0 + PKCE) | User authentication |
| **Faceit Data API** | HTTPS REST | Player stats, match history, ELO data |
| **Faceit / Steam** | HTTPS | Demo file downloads (`.dem` URLs from match data) |
| **User Browser** | HTTPS + WSS | All UI interactions, real-time features |

---

## 3. Container Diagram (C4 Level 2)

```
                           ┌──────────────┐
                           │   Browser     │
                           │  (Next.js +   │
                           │   PixiJS)     │
                           └──────┬───────┘
                                  │ HTTPS / WSS
                                  ▼
                           ┌──────────────┐
                           │    nginx      │
                           │   (reverse    │
                           │    proxy)     │
                           └──────┬───────┘
                         ┌────────┼────────┐
                         │        │        │
                    ┌────▼───┐ ┌──▼───┐ ┌──▼──────┐
                    │Frontend│ │ API  │ │   WS    │
                    │Next.js │ │Server│ │ Server  │
                    │  :3000 │ │(Go)  │ │  (Go)   │
                    │        │ │:8080 │ │  :8081  │
                    └────────┘ └──┬───┘ └──┬──────┘
                                  │        │
                    ┌─────────────┼────────┼──────────┐
                    │             │        │          │
               ┌────▼───┐  ┌─────▼──┐  ┌──▼───┐  ┌──▼───┐
               │Worker  │  │Postgres│  │Redis │  │MinIO │
               │(Go)    │  │+ Timesc│  │  7   │  │(S3)  │
               │        │  │aleDB   │  │:6379 │  │:9000 │
               └────────┘  │:5432   │  └──────┘  └──────┘
                           └────────┘
```

### Container Descriptions

| Container | Technology | Responsibility |
|-----------|-----------|---------------|
| **nginx** | nginx 1.25+ | TLS termination, reverse proxy, static asset caching, WebSocket upgrade |
| **Frontend** | Next.js 14+ (Node 20) | Server-rendered pages, PixiJS canvas, Zustand state, Yjs client |
| **API Server** | Go 1.22+ / chi | REST API, authentication, business logic, Faceit API client |
| **WS Server** | Go 1.22+ / gorilla/websocket | WebSocket connections for viewer sync and Yjs relay |
| **Worker** | Go 1.22+ | Background jobs: demo parsing, Faceit sync, heatmap generation |
| **PostgreSQL** | PostgreSQL 16 + TimescaleDB | Relational data + time-series tick data in hypertables |
| **Redis** | Redis 7 | Session store, API cache, job queue (Redis Streams) |
| **MinIO** | MinIO (latest) | S3-compatible object storage for `.dem` files and exported assets |

### Container Communication

| From | To | Protocol | Notes |
|------|----|----------|-------|
| nginx | Frontend | HTTP | Proxied on `/` path |
| nginx | API Server | HTTP | Proxied on `/api/` path |
| nginx | WS Server | WebSocket | Proxied on `/ws/` path; `Upgrade: websocket` |
| API Server | PostgreSQL | TCP (pg) | sqlc-generated queries |
| API Server | Redis | TCP (RESP) | Session read/write, cache |
| API Server | MinIO | HTTP (S3) | Presigned URLs for upload/download |
| API Server | Redis (Streams) | TCP | Enqueue background jobs |
| Worker | Redis (Streams) | TCP | Dequeue and process jobs |
| Worker | PostgreSQL | TCP (pg) | Write parsed demo data |
| Worker | MinIO | HTTP (S3) | Read `.dem` files for parsing |
| WS Server | Redis (Pub/Sub) | TCP | Cross-instance message broadcast |
| Frontend | API Server | HTTP | REST calls via TanStack Query |
| Frontend | WS Server | WebSocket | Viewer sync + Yjs strat board |

---

## 4. Component Diagrams (C4 Level 3)

### 4.1 Go Backend Components

```
┌─────────────────────────────────────────────────────┐
│                   API Server (Go)                    │
│                                                     │
│  ┌─────────────┐  ┌──────────────┐  ┌───────────┐  │
│  │   Router     │  │  Middleware   │  │  Handlers │  │
│  │   (chi)      │─▶│  - Auth      │─▶│  - Demo   │  │
│  │              │  │  - CORS      │  │  - Round   │  │
│  │              │  │  - RateLimit │  │  - Strat   │  │
│  │              │  │  - Logging   │  │  - Faceit  │  │
│  │              │  │  - Recovery  │  │  - Lineup  │  │
│  └─────────────┘  └──────────────┘  │  - Auth    │  │
│                                     └─────┬─────┘  │
│                                           │        │
│  ┌──────────────┐  ┌──────────────┐  ┌────▼─────┐  │
│  │ Faceit Client │  │  S3 Client   │  │ Services │  │
│  │ (HTTP)        │  │  (MinIO SDK) │  │  - Demo  │  │
│  └──────────────┘  └──────────────┘  │  - User  │  │
│                                      │  - Strat │  │
│  ┌──────────────┐  ┌──────────────┐  │  - Stats │  │
│  │ Session Store │  │  Job Queue   │  └────┬─────┘  │
│  │ (Redis)       │  │ (Redis Strm) │       │        │
│  └──────────────┘  └──────────────┘  ┌────▼─────┐  │
│                                      │   Store   │  │
│                                      │  (sqlc)   │  │
│                                      └──────────┘  │
└─────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────┐
│                   Worker (Go)                        │
│                                                     │
│  ┌──────────────┐  ┌──────────────┐  ┌───────────┐  │
│  │ Stream        │  │ Demo Parser  │  │  Faceit   │  │
│  │ Consumer      │─▶│ (demoinfocs) │  │  Syncer   │  │
│  │ (Redis)       │  └──────────────┘  └───────────┘  │
│  └──────────────┘                                   │
│                    ┌──────────────┐  ┌───────────┐  │
│                    │  Heatmap     │  │  Store    │  │
│                    │  Generator   │  │  (sqlc)   │  │
│                    └──────────────┘  └───────────┘  │
└─────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────┐
│                 WS Server (Go)                       │
│                                                     │
│  ┌──────────────┐  ┌──────────────┐  ┌───────────┐  │
│  │ Upgrade       │  │ Viewer Hub   │  │  Strat    │  │
│  │ Handler       │─▶│ (demo rooms) │  │  Relay    │  │
│  │ (gorilla)     │  └──────────────┘  │  (Yjs)   │  │
│  └──────────────┘                     └───────────┘  │
│                    ┌──────────────┐                  │
│                    │ Redis PubSub │                  │
│                    │ (broadcast)  │                  │
│                    └──────────────┘                  │
└─────────────────────────────────────────────────────┘
```

### 4.2 Next.js Frontend Components

```
┌──────────────────────────────────────────────────────────┐
│                  Next.js Frontend                         │
│                                                          │
│  ┌─────────────────────────────────────────────────────┐ │
│  │                    App Shell                         │ │
│  │  Sidebar  │  Header  │  Content Area                │ │
│  └─────────────────────────────────────────────────────┘ │
│                                                          │
│  ┌───────────┐  ┌───────────┐  ┌───────────────────────┐ │
│  │  Pages     │  │  Stores    │  │  Providers            │ │
│  │            │  │ (Zustand)  │  │                       │ │
│  │ - Viewer   │  │            │  │ - AuthProvider        │ │
│  │ - Heatmap  │  │ - viewer   │  │ - QueryProvider       │ │
│  │ - Strats   │  │ - strat    │  │ - WebSocketProvider   │ │
│  │ - Dashboard│  │ - ui       │  │ - ThemeProvider       │ │
│  │ - Lineups  │  │ - faceit   │  │                       │ │
│  └─────┬─────┘  └─────┬─────┘  └───────────────────────┘ │
│        │              │                                   │
│  ┌─────▼──────────────▼──────────────────────────────┐   │
│  │                 Canvas Layer                        │   │
│  │                                                    │   │
│  │  ┌──────────────┐  ┌──────────────┐               │   │
│  │  │  PixiJS App   │  │  Yjs Doc +    │               │   │
│  │  │  (Viewer)     │  │  Awareness    │               │   │
│  │  │               │  │  (Strat Board)│               │   │
│  │  │ - MapLayer    │  │               │               │   │
│  │  │ - PlayerLayer │  │ - DrawLayer   │               │   │
│  │  │ - EventLayer  │  │ - TokenLayer  │               │   │
│  │  │ - UILayer     │  │ - CursorLayer │               │   │
│  │  └──────────────┘  └──────────────┘               │   │
│  └────────────────────────────────────────────────────┘   │
│                                                          │
│  ┌────────────────────────────────────────────────────┐   │
│  │                  UI Components (shadcn/ui)          │   │
│  │  Button │ Dialog │ Tabs │ Select │ Slider │ ...    │   │
│  └────────────────────────────────────────────────────┘   │
└──────────────────────────────────────────────────────────┘
```

### Key Frontend Patterns

| Pattern | Implementation |
|---------|---------------|
| **PixiJS outside React** | PixiJS Application instantiated in a `useEffect`; React renders a container `<div>`, PixiJS manages its own render loop. Zustand store bridges React UI controls ↔ PixiJS state. |
| **Zustand stores** | Separate stores per domain: `viewerStore` (playback state, current tick), `stratStore` (board state), `uiStore` (sidebar, modals), `faceitStore` (profile, matches). |
| **TanStack Query** | All API data fetched via `useQuery`/`useMutation`. Stale-while-revalidate for demo lists, match history. |
| **Yjs integration** | Yjs `Doc` + `WebsocketProvider` for strat board. Drawing operations are Yjs map/array mutations. Awareness protocol for cursor positions. |

---

## 5. Data Flow Diagrams

### 5.1 Demo Upload & Parse

```
User                Frontend           API Server        MinIO          Redis Streams      Worker            PostgreSQL
 │                    │                    │                │                │                │                │
 │  Upload .dem       │                    │                │                │                │                │
 │───────────────────▶│                    │                │                │                │                │
 │                    │  POST /api/v1/demos│                │                │                │                │
 │                    │───────────────────▶│                │                │                │                │
 │                    │                    │  Validate file │                │                │                │
 │                    │                    │  (magic bytes, │                │                │                │
 │                    │                    │   size check)  │                │                │                │
 │                    │                    │                │                │                │                │
 │                    │                    │  PUT object    │                │                │                │
 │                    │                    │───────────────▶│                │                │                │
 │                    │                    │      OK        │                │                │                │
 │                    │                    │◀───────────────│                │                │                │
 │                    │                    │                │                │                │                │
 │                    │                    │  INSERT demo   │                │                │                │
 │                    │                    │  (status=      │                │                │                │
 │                    │                    │   uploaded)     │                │                │          ─────▶│
 │                    │                    │                │                │                │                │
 │                    │                    │  XADD parse_job│                │                │                │
 │                    │                    │───────────────────────────────▶│                │                │
 │                    │                    │                │                │                │                │
 │                    │  202 Accepted      │                │                │                │                │
 │                    │  {id, status:      │                │                │                │                │
 │                    │   uploaded}        │                │                │                │                │
 │                    │◀───────────────────│                │                │                │                │
 │  Show "parsing"    │                    │                │                │                │                │
 │◀───────────────────│                    │                │                │                │                │
 │                    │                    │                │  XREAD         │                │                │
 │                    │                    │                │  parse_job     │                │                │
 │                    │                    │                │◀───────────────────────────────│                │
 │                    │                    │                │                │                │                │
 │                    │                    │                │                │  GET .dem file │                │
 │                    │                    │                │◀───────────────────────────────│                │
 │                    │                    │                │  .dem bytes    │                │                │
 │                    │                    │                │───────────────────────────────▶│                │
 │                    │                    │                │                │                │                │
 │                    │                    │                │                │  Parse with    │                │
 │                    │                    │                │                │  demoinfocs    │                │
 │                    │                    │                │                │                │                │
 │                    │                    │                │                │  Batch INSERT  │                │
 │                    │                    │                │                │  ticks, events,│                │
 │                    │                    │                │                │  rounds, stats │                │
 │                    │                    │                │                │───────────────▶│                │
 │                    │                    │                │                │                │                │
 │                    │                    │                │                │  UPDATE demo   │                │
 │                    │                    │                │                │  status=ready  │                │
 │                    │                    │                │                │───────────────▶│                │
 │                    │                    │                │                │                │                │
 │  Poll status       │                    │                │                │                │                │
 │───────────────────▶│  GET /demos/:id    │                │                │                │                │
 │                    │───────────────────▶│                │                │                │                │
 │                    │  {status: ready}   │                │                │                │                │
 │                    │◀───────────────────│                │                │                │                │
 │  Show "ready"      │                    │                │                │                │                │
 │◀───────────────────│                    │                │                │                │                │
```

### 5.2 Faceit Sync

```
User logs in
     │
     ▼
API Server ──▶ Faceit OAuth ──▶ Access token stored in session
     │
     ▼
API Server ──▶ Redis Streams: XADD faceit_sync_job {user_id, faceit_id}
     │
     ▼
Worker reads job
     │
     ├──▶ GET Faceit Data API /players/{id}/history
     │    (paginate through recent matches)
     │
     ├──▶ For each new match:
     │    INSERT INTO faceit_matches (...)
     │
     └──▶ Optionally: download demo URL → enqueue parse job
```

### 5.3 Strategy Board Collaboration (Yjs)

```
User A (Browser)          WS Server              User B (Browser)
     │                        │                        │
     │  WS Connect            │                        │
     │  /ws/strat/:id         │    WS Connect          │
     │───────────────────────▶│◀───────────────────────│
     │                        │                        │
     │  Yjs sync step 1       │                        │
     │  (state vector)        │                        │
     │───────────────────────▶│                        │
     │                        │  Relay sync step 1     │
     │                        │───────────────────────▶│
     │                        │                        │
     │                        │  Yjs sync step 2       │
     │                        │  (missing updates)     │
     │                        │◀───────────────────────│
     │  Relay sync step 2     │                        │
     │◀───────────────────────│                        │
     │                        │                        │
     │  ── Synced state ──    │   ── Synced state ──   │
     │                        │                        │
     │  User A draws arrow    │                        │
     │  (Yjs update)          │                        │
     │───────────────────────▶│                        │
     │                        │  Broadcast update      │
     │                        │───────────────────────▶│
     │                        │                        │  Arrow appears
     │                        │                        │
     │                        │  Awareness update      │
     │                        │  (User B cursor pos)   │
     │                        │◀───────────────────────│
     │  Broadcast awareness   │                        │
     │◀───────────────────────│                        │
     │  See User B's cursor   │                        │
     │                        │                        │
     │  ── On disconnect ──   │                        │
     │                        │  Persist Yjs state     │
     │                        │──▶ PostgreSQL          │
     │                        │   (yjs_state BYTEA)    │
```

### 5.4 Grenade Extraction Pipeline

```
Worker (during demo parse)
     │
     ├──▶ Listen for grenade_throw events from demoinfocs parser
     │    Extract: thrower SteamID, position (x,y,z), aim angles, grenade type
     │
     ├──▶ Listen for grenade_detonate events
     │    Extract: detonation position (x,y,z), tick
     │
     ├──▶ Correlate throw → detonate by entity ID and tick proximity
     │
     ├──▶ INSERT INTO game_events (event_type='grenade_throw', ...)
     │    INSERT INTO game_events (event_type='grenade_detonate', ...)
     │
     └──▶ INSERT INTO grenade_lineups (auto-generated entries)
          - map_name, grenade_type, throw position, landing position
          - Linked to demo_id and tick for "view in demo"
```

---

## 6. REST API Specification

### 6.1 Authentication

All endpoints except `/api/v1/auth/*` and health checks require a valid session cookie.

#### `GET /api/v1/auth/faceit`

Redirects to Faceit OAuth authorization URL with PKCE challenge.

#### `GET /api/v1/auth/faceit/callback`

| Parameter | Source | Description |
|-----------|--------|-------------|
| code | Query | Authorization code from Faceit |
| state | Query | CSRF state parameter |

Exchanges code for tokens, creates/updates user, sets session cookie, redirects to `/dashboard`.

#### `POST /api/v1/auth/logout`

Invalidates Redis session. Clears session cookie.

#### `GET /api/v1/auth/me`

Returns current user profile. **Response 200:**

```json
{
  "data": {
    "id": "uuid",
    "nickname": "PlayerOne",
    "avatar_url": "https://...",
    "faceit_elo": 2100,
    "faceit_level": 9,
    "country": "SE"
  }
}
```

### 6.2 Demos

#### `POST /api/v1/demos`

Multipart file upload. Max 500 MB.

**Response 202:**

```json
{
  "data": {
    "id": "uuid",
    "status": "uploaded",
    "file_size": 104857600,
    "created_at": "2026-03-31T12:00:00Z"
  }
}
```

#### `GET /api/v1/demos`

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| page | int | 1 | Page number |
| per_page | int | 20 | Items per page (max 100) |
| map | string | - | Filter by map name |
| status | string | - | Filter by status |
| sort | string | -created_at | Sort field (prefix `-` for desc) |

#### `GET /api/v1/demos/:id/ticks`

| Parameter | Type | Description |
|-----------|------|-------------|
| start_tick | int | Start of tick range (required) |
| end_tick | int | End of tick range (required) |
| steam_ids | string | Comma-separated filter (optional) |

Returns array of tick data points. Max range: 6400 ticks (100s at 64 tick).

#### `GET /api/v1/demos/:id/events`

| Parameter | Type | Description |
|-----------|------|-------------|
| round | int | Filter by round number |
| type | string | Filter by event type |
| steam_id | string | Filter by player |

### 6.3 Heatmaps

#### `POST /api/v1/heatmaps/aggregate`

**Request:**

```json
{
  "demo_ids": ["uuid1", "uuid2"],
  "type": "kills",
  "filters": {
    "side": "CT",
    "weapon_category": "rifle",
    "player_steam_id": "76561198..."
  },
  "resolution": 256
}
```

**Response 200:** Array of `{x, y, intensity}` data points for client-side KDE rendering.

### 6.4 Strategy Boards

#### `POST /api/v1/strats`

```json
{
  "title": "Mirage A Execute",
  "map_name": "de_mirage"
}
```

#### `PUT /api/v1/strats/:id`

```json
{
  "title": "Updated title",
  "share_mode": "read_only"
}
```

#### `POST /api/v1/strats/:id/export`

Returns PNG image of the board's current state.

### 6.5 Grenade Lineups

#### `GET /api/v1/lineups`

| Parameter | Type | Description |
|-----------|------|-------------|
| map | string | Filter by map |
| type | string | smoke / flash / he / molotov |
| search | string | Full-text search on title/description |
| favorites | bool | Only show favorites |
| page | int | Pagination |

### 6.6 Faceit

#### `GET /api/v1/faceit/elo-history`

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| days | int | 30 | Number of days to fetch |

Returns array of `{date, elo, match_id}`.

---

## 7. WebSocket Protocol

### 7.1 Viewer WebSocket (`/ws/viewer/:demoId`)

Used for synchronized demo viewing (future multi-user spectating). Currently serves as efficient tick data streaming.

#### Client → Server Messages

```json
{"type": "subscribe", "round": 5}
{"type": "seek", "tick": 12800}
{"type": "set_speed", "speed": 2.0}
```

#### Server → Client Messages

```json
{"type": "tick_batch", "ticks": [...], "start_tick": 12800, "end_tick": 12864}
{"type": "events", "events": [...]}
{"type": "round_info", "round": 5, "start_tick": 12000, "end_tick": 15000}
```

### 7.2 Strategy Board WebSocket (`/ws/strat/:stratId`)

Implements the Yjs WebSocket protocol. The Go server acts as a "dumb relay":

1. **Connection**: Client connects, sends Yjs sync step 1 (state vector)
2. **Sync**: Server loads persisted Yjs state from PostgreSQL, sends sync step 2 (missing updates)
3. **Updates**: Client sends Yjs updates (binary); server broadcasts to all other clients in the room
4. **Awareness**: Yjs Awareness protocol messages relayed for cursor positions and user presence
5. **Persistence**: On last client disconnect (or periodic interval), server encodes current Yjs doc state and writes to PostgreSQL

#### Authentication

WebSocket connections authenticated via session cookie (same as REST). The upgrade request is validated by the auth middleware before the connection is established.

#### Binary Protocol

Yjs messages are binary (Uint8Array). The WS server does not interpret the content -- it simply relays between connected clients and handles persistence.

Message types (first byte):

| Byte | Type |
|------|------|
| 0 | Yjs sync |
| 1 | Yjs awareness |
| 2 | Yjs auth (unused -- we use cookie auth) |

---

## 8. Database Schema

### 8.1 PostgreSQL + TimescaleDB DDL

```sql
-- Extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "timescaledb";

-- ===================
-- Users
-- ===================
CREATE TABLE users (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    faceit_id       VARCHAR(64) NOT NULL UNIQUE,
    nickname        VARCHAR(64) NOT NULL,
    avatar_url      TEXT,
    faceit_elo      INTEGER DEFAULT 0,
    faceit_level    SMALLINT DEFAULT 1,
    country         VARCHAR(2),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_users_faceit_id ON users (faceit_id);

-- ===================
-- Demos
-- ===================
CREATE TABLE demos (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    faceit_match_id VARCHAR(64),
    map_name        VARCHAR(32) NOT NULL,
    file_path       TEXT NOT NULL,
    file_size       BIGINT NOT NULL,
    status          VARCHAR(16) NOT NULL DEFAULT 'uploaded',
    total_ticks     INTEGER,
    tick_rate       REAL,
    duration_secs   INTEGER,
    match_date      TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_demos_user_id ON demos (user_id);
CREATE INDEX idx_demos_status ON demos (status);
CREATE INDEX idx_demos_map ON demos (map_name);

-- ===================
-- Rounds
-- ===================
CREATE TABLE rounds (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    demo_id         UUID NOT NULL REFERENCES demos(id) ON DELETE CASCADE,
    round_number    SMALLINT NOT NULL,
    start_tick      INTEGER NOT NULL,
    end_tick        INTEGER NOT NULL,
    winner_side     VARCHAR(2) NOT NULL,
    win_reason      VARCHAR(32) NOT NULL,
    ct_score        SMALLINT NOT NULL,
    t_score         SMALLINT NOT NULL,

    UNIQUE (demo_id, round_number)
);

CREATE INDEX idx_rounds_demo_id ON rounds (demo_id);

-- ===================
-- Player Rounds
-- ===================
CREATE TABLE player_rounds (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    round_id        UUID NOT NULL REFERENCES rounds(id) ON DELETE CASCADE,
    steam_id        VARCHAR(20) NOT NULL,
    player_name     VARCHAR(64) NOT NULL,
    team_side       VARCHAR(2) NOT NULL,
    kills           SMALLINT NOT NULL DEFAULT 0,
    deaths          SMALLINT NOT NULL DEFAULT 0,
    assists         SMALLINT NOT NULL DEFAULT 0,
    damage          INTEGER NOT NULL DEFAULT 0,
    headshot_kills  SMALLINT NOT NULL DEFAULT 0,
    first_kill      BOOLEAN NOT NULL DEFAULT FALSE,
    first_death     BOOLEAN NOT NULL DEFAULT FALSE,
    clutch_kills    SMALLINT NOT NULL DEFAULT 0,

    UNIQUE (round_id, steam_id)
);

CREATE INDEX idx_player_rounds_round_id ON player_rounds (round_id);
CREATE INDEX idx_player_rounds_steam_id ON player_rounds (steam_id);

-- ===================
-- Tick Data (TimescaleDB Hypertable)
-- ===================
CREATE TABLE tick_data (
    time            TIMESTAMPTZ NOT NULL,
    demo_id         UUID NOT NULL,
    tick            INTEGER NOT NULL,
    steam_id        VARCHAR(20) NOT NULL,
    x               REAL NOT NULL,
    y               REAL NOT NULL,
    z               REAL NOT NULL,
    yaw             REAL NOT NULL,
    health          SMALLINT NOT NULL,
    armor           SMALLINT NOT NULL,
    is_alive        BOOLEAN NOT NULL,
    weapon          VARCHAR(32)
);

SELECT create_hypertable('tick_data', 'time');

CREATE INDEX idx_tick_data_demo_tick ON tick_data (demo_id, tick);
CREATE INDEX idx_tick_data_steam_id ON tick_data (steam_id, time DESC);

-- ===================
-- Game Events
-- ===================
CREATE TABLE game_events (
    id                  UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    demo_id             UUID NOT NULL REFERENCES demos(id) ON DELETE CASCADE,
    round_id            UUID REFERENCES rounds(id) ON DELETE CASCADE,
    tick                INTEGER NOT NULL,
    event_type          VARCHAR(32) NOT NULL,
    attacker_steam_id   VARCHAR(20),
    victim_steam_id     VARCHAR(20),
    weapon              VARCHAR(32),
    x                   REAL,
    y                   REAL,
    z                   REAL,
    extra_data          JSONB
);

CREATE INDEX idx_game_events_demo_id ON game_events (demo_id);
CREATE INDEX idx_game_events_type ON game_events (event_type);
CREATE INDEX idx_game_events_demo_round ON game_events (demo_id, round_id);

-- ===================
-- Strategy Boards
-- ===================
CREATE TABLE strategy_boards (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title           VARCHAR(128) NOT NULL,
    map_name        VARCHAR(32) NOT NULL,
    yjs_state       BYTEA,
    share_mode      VARCHAR(16) NOT NULL DEFAULT 'private',
    share_token     VARCHAR(64) UNIQUE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_strat_boards_user_id ON strategy_boards (user_id);
CREATE INDEX idx_strat_boards_share_token ON strategy_boards (share_token);

-- ===================
-- Grenade Lineups
-- ===================
CREATE TABLE grenade_lineups (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    demo_id         UUID REFERENCES demos(id) ON DELETE SET NULL,
    tick            INTEGER,
    map_name        VARCHAR(32) NOT NULL,
    grenade_type    VARCHAR(16) NOT NULL,
    throw_x         REAL NOT NULL,
    throw_y         REAL NOT NULL,
    throw_z         REAL NOT NULL,
    throw_yaw       REAL NOT NULL,
    throw_pitch     REAL NOT NULL,
    land_x          REAL NOT NULL,
    land_y          REAL NOT NULL,
    land_z          REAL NOT NULL,
    title           VARCHAR(128) NOT NULL,
    description     TEXT,
    tags            TEXT[],
    is_favorite     BOOLEAN NOT NULL DEFAULT FALSE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_lineups_user_id ON grenade_lineups (user_id);
CREATE INDEX idx_lineups_map_type ON grenade_lineups (map_name, grenade_type);

-- ===================
-- Faceit Matches
-- ===================
CREATE TABLE faceit_matches (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    faceit_match_id VARCHAR(64) NOT NULL,
    map_name        VARCHAR(32) NOT NULL,
    score_team      SMALLINT NOT NULL,
    score_opponent  SMALLINT NOT NULL,
    result          VARCHAR(4) NOT NULL,
    elo_before      INTEGER,
    elo_after       INTEGER,
    kills           SMALLINT,
    deaths          SMALLINT,
    assists         SMALLINT,
    demo_url        TEXT,
    demo_id         UUID REFERENCES demos(id) ON DELETE SET NULL,
    played_at       TIMESTAMPTZ NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE (user_id, faceit_match_id)
);

CREATE INDEX idx_faceit_matches_user_id ON faceit_matches (user_id, played_at DESC);

-- ===================
-- Sessions (managed by Redis, schema for reference)
-- ===================
-- Redis key: session:{token}
-- Redis value: JSON {user_id, faceit_access_token, faceit_refresh_token, created_at, expires_at}
-- TTL: 7 days
```

### 8.2 TimescaleDB Configuration

```sql
-- Hypertable chunk interval: 1 day (grouping demos by date)
SELECT set_chunk_time_interval('tick_data', INTERVAL '1 day');

-- Compression policy: compress chunks older than 7 days
ALTER TABLE tick_data SET (
    timescaledb.compress,
    timescaledb.compress_segmentby = 'demo_id',
    timescaledb.compress_orderby = 'tick ASC, steam_id ASC'
);

SELECT add_compression_policy('tick_data', INTERVAL '7 days');

-- Retention policy: drop chunks older than 365 days (optional)
-- SELECT add_retention_policy('tick_data', INTERVAL '365 days');
```

### 8.3 Redis Key Patterns

| Pattern | Type | TTL | Purpose |
|---------|------|-----|---------|
| `session:{token}` | String (JSON) | 7d | User session |
| `user:{id}:profile` | String (JSON) | 5m | Cached user profile |
| `faceit:profile:{faceit_id}` | String (JSON) | 15m | Cached Faceit profile |
| `demo:{id}:status` | String | 1h | Demo parse status (fast polling) |
| `ratelimit:{ip}:{endpoint}` | String (counter) | 1m | Rate limiting |
| `stream:parse_jobs` | Stream | - | Demo parse job queue |
| `stream:faceit_sync_jobs` | Stream | - | Faceit sync job queue |
| `strat:{id}:clients` | Set | - | Active WebSocket clients per board |

---

## 9. Docker Compose Topology

### 9.1 Services

```yaml
# docker-compose.yml (simplified)
services:
  nginx:
    image: nginx:1.25-alpine
    ports: ["80:80", "443:443"]
    networks: [frontend]
    depends_on: [web, api, ws]

  web:
    build: ./frontend
    expose: ["3000"]
    networks: [frontend]
    environment:
      NEXT_PUBLIC_API_URL: /api
      NEXT_PUBLIC_WS_URL: /ws

  api:
    build: ./backend
    command: ["./oversite", "serve"]
    expose: ["8080"]
    networks: [frontend, backend]
    depends_on: [postgres, redis, minio]
    environment:
      DATABASE_URL: postgres://oversite:oversite@postgres:5432/oversite
      REDIS_URL: redis://redis:6379
      MINIO_ENDPOINT: minio:9000

  ws:
    build: ./backend
    command: ["./oversite", "ws"]
    expose: ["8081"]
    networks: [frontend, backend]
    depends_on: [postgres, redis]

  worker:
    build: ./backend
    command: ["./oversite", "worker"]
    networks: [backend]
    depends_on: [postgres, redis, minio]

  postgres:
    image: timescale/timescaledb:latest-pg16
    volumes: ["pgdata:/var/lib/postgresql/data"]
    networks: [backend]
    environment:
      POSTGRES_DB: oversite
      POSTGRES_USER: oversite
      POSTGRES_PASSWORD: oversite

  redis:
    image: redis:7-alpine
    volumes: ["redisdata:/data"]
    networks: [backend]

  minio:
    image: minio/minio:latest
    command: server /data --console-address ":9001"
    volumes: ["miniodata:/data"]
    networks: [backend]
    environment:
      MINIO_ROOT_USER: minioadmin
      MINIO_ROOT_PASSWORD: minioadmin

networks:
  frontend:
  backend:

volumes:
  pgdata:
  redisdata:
  miniodata:
```

### 9.2 Network Segmentation

| Network | Services | Purpose |
|---------|----------|---------|
| **frontend** | nginx, web, api, ws | Handles user-facing traffic |
| **backend** | api, ws, worker, postgres, redis, minio | Internal service communication |

- `nginx` is the only service with published ports
- `postgres`, `redis`, `minio` are only reachable from the `backend` network
- `api` and `ws` bridge both networks (receive requests from nginx, connect to backend services)

### 9.3 nginx Configuration (Key Routes)

```nginx
upstream frontend { server web:3000; }
upstream api      { server api:8080; }
upstream ws       { server ws:8081;  }

server {
    listen 80;

    # Frontend
    location / {
        proxy_pass http://frontend;
    }

    # REST API
    location /api/ {
        proxy_pass http://api;
        client_max_body_size 500m;  # demo uploads
    }

    # WebSocket
    location /ws/ {
        proxy_pass http://ws;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_read_timeout 86400s;
    }
}
```

---

## 10. CRDT Decision: Yjs vs OT

### Why Yjs (CRDT) over Operational Transformation

| Factor | Yjs (CRDT) | OT (e.g., ShareJS) |
|--------|-----------|---------------------|
| **Server complexity** | Dumb relay -- server just broadcasts binary messages | Server must transform operations; complex state machine |
| **Conflict resolution** | Automatic, mathematical guarantee | Requires correct transformation functions |
| **Go compatibility** | Server is a WebSocket relay; no Yjs logic in Go | Would need Go OT implementation (none mature) |
| **Offline support** | Built-in; changes merge on reconnect | Requires additional buffering logic |
| **Awareness protocol** | Built-in cursor/presence support | Must implement separately |
| **Ecosystem** | Active JS ecosystem; y-websocket reference impl | Fewer maintained libraries |
| **Performance** | Sub-linear merge; compact binary encoding | Linear in operation count |

### Trade-offs Accepted

- **Binary protocol**: Server cannot inspect or validate drawing operations (must trust client)
- **State size**: Yjs document grows with edit history (mitigated by periodic GC / snapshot)
- **No server-side rendering**: Cannot generate strat board thumbnails without a JS runtime (future: headless browser sidecar)

---

## 11. Cross-Cutting Concerns

### 11.1 Authentication Flow

```
Browser                    API Server              Faceit OAuth
  │                            │                       │
  │  GET /api/v1/auth/faceit   │                       │
  │───────────────────────────▶│                       │
  │                            │  Generate PKCE pair   │
  │                            │  Store in Redis       │
  │  302 → Faceit authorize URL│                       │
  │◀───────────────────────────│                       │
  │                            │                       │
  │  User authorizes on Faceit │                       │
  │───────────────────────────────────────────────────▶│
  │                            │                       │
  │  302 → /callback?code=...  │                       │
  │───────────────────────────▶│                       │
  │                            │  Exchange code+PKCE   │
  │                            │──────────────────────▶│
  │                            │  {access, refresh}    │
  │                            │◀──────────────────────│
  │                            │                       │
  │                            │  GET /me (Faceit API) │
  │                            │──────────────────────▶│
  │                            │  {id, nickname, elo}  │
  │                            │◀──────────────────────│
  │                            │                       │
  │                            │  Upsert user in DB    │
  │                            │  Create Redis session │
  │  Set-Cookie: session=...   │                       │
  │  302 → /dashboard          │                       │
  │◀───────────────────────────│                       │
```

### 11.2 Authorization Rules

| Resource | Rule |
|----------|------|
| Demos | Users can only access their own demos |
| Strategy boards | Owner: full access. Shared read_only: view only. Shared editable: view + edit |
| Lineups | Users can only CRUD their own lineups. Browsing shows all (future: community) |
| Faceit data | Users can only view their own Faceit data |
| Admin | No admin role in v1. Single-tenant per-user model |

### 11.3 Error Handling Strategy

| Layer | Approach |
|-------|----------|
| **Go handlers** | Return structured `{error: {code, message}}` JSON. Map domain errors to HTTP status codes. |
| **Go services** | Return `error` values with sentinel errors (`ErrNotFound`, `ErrForbidden`, etc.) |
| **Worker** | On parse failure: set demo status to `error`, log error, continue processing queue |
| **Frontend** | TanStack Query `onError` callbacks. Toast notifications for user-facing errors. Error boundaries for component crashes. |
| **WebSocket** | Send error frame + close with appropriate status code (4001 = auth, 4004 = not found) |

### 11.4 Observability

| Component | Tool | Details |
|-----------|------|---------|
| **Structured logging** | `slog` (Go stdlib) | JSON format, request ID correlation |
| **HTTP metrics** | Prometheus via chi middleware | Request count, latency histograms, error rates |
| **Health checks** | `/healthz` (liveness), `/readyz` (readiness) | Checks DB, Redis, MinIO connectivity |
| **Frontend errors** | Console + future Sentry integration | Error boundaries report to error tracking |

---

## 12. Monorepo Directory Structure

```
oversite/
├── .github/
│   └── workflows/
│       ├── ci.yml                 # Lint, test, build
│       └── docker.yml             # Build & push images
├── backend/
│   ├── cmd/
│   │   └── oversite/
│   │       └── main.go            # CLI entry (serve, ws, worker, migrate)
│   ├── internal/
│   │   ├── auth/                  # OAuth, session, middleware
│   │   ├── config/                # Env-based configuration
│   │   ├── demo/                  # Demo service + parser integration
│   │   ├── faceit/                # Faceit API client + sync
│   │   ├── handler/               # HTTP handlers (chi)
│   │   ├── lineup/                # Grenade lineup service
│   │   ├── middleware/            # CORS, rate limit, logging, recovery
│   │   ├── model/                 # Domain types (shared across packages)
│   │   ├── store/                 # sqlc-generated + custom DB queries
│   │   ├── strat/                 # Strategy board service
│   │   ├── websocket/             # WS hub, viewer, Yjs relay
│   │   ├── worker/                # Background job processor
│   │   └── testutil/              # Test helpers, containers, mocks
│   ├── migrations/                # SQL migration files (golang-migrate)
│   ├── queries/                   # sqlc SQL files
│   ├── testdata/                  # Test fixtures and golden files
│   │   ├── demos/                 # Small .dem fixture files
│   │   └── golden/                # Golden file snapshots
│   ├── sqlc.yaml                  # sqlc configuration
│   ├── go.mod
│   ├── go.sum
│   ├── Makefile                   # Build, test, lint, generate
│   └── Dockerfile
├── frontend/
│   ├── src/
│   │   ├── app/                   # Next.js App Router pages
│   │   ├── components/
│   │   │   ├── ui/                # shadcn/ui components
│   │   │   ├── viewer/            # PixiJS viewer components
│   │   │   ├── strat/             # Strategy board components
│   │   │   └── layout/            # Shell, sidebar, header
│   │   ├── hooks/                 # Custom React hooks
│   │   ├── lib/
│   │   │   ├── api.ts             # API client (fetch wrapper)
│   │   │   ├── ws.ts              # WebSocket client
│   │   │   ├── pixi/              # PixiJS setup, layers, renderers
│   │   │   ├── yjs/               # Yjs doc, provider, awareness
│   │   │   └── maps/              # Map radar images + calibration data
│   │   ├── stores/                # Zustand stores
│   │   ├── types/                 # TypeScript types
│   │   ├── utils/                 # Utility functions
│   │   └── test/                  # Test infrastructure
│   │       ├── msw/               # MSW handlers and server setup
│   │       ├── render.tsx          # Custom renderWithProviders helper
│   │       └── setup.ts           # Vitest global setup
│   ├── public/
│   │   └── maps/                  # Radar images (de_dust2.png, etc.)
│   ├── next.config.js
│   ├── tailwind.config.ts
│   ├── tsconfig.json
│   ├── package.json
│   └── Dockerfile
├── docker-compose.yml
├── docker-compose.dev.yml         # Dev overrides (hot reload, debug ports)
├── nginx/
│   └── nginx.conf
├── docs/
│   ├── PRD.md
│   ├── ARCHITECTURE.md
│   ├── IMPLEMENTATION_PLAN.md
│   └── TASK_BREAKDOWN.md
├── e2e/                            # Playwright E2E tests
│   └── tests/                      # E2E test specs
├── CLAUDE.md
├── Makefile                       # Root-level commands (up, down, logs, etc.)
└── README.md
```

---

## 13. Cloud Migration Path

The architecture is designed for Docker Compose locally but structured for straightforward cloud migration:

| Local (Docker Compose) | Cloud Equivalent | Migration Notes |
|------------------------|-----------------|-----------------|
| nginx container | Cloud Load Balancer / CDN (e.g., Cloudflare, AWS ALB) | TLS termination moves to LB; nginx may remain for routing |
| Next.js container | Vercel / Cloud Run / ECS | Static export or server mode depending on SSR needs |
| Go API container | Cloud Run / ECS / K8s Deployment | Stateless; scales horizontally |
| Go WS container | Cloud Run (WebSocket-enabled) / ECS | Sticky sessions via Redis PubSub for multi-instance |
| Go Worker container | Cloud Run Jobs / ECS Task | Scaled by queue depth |
| PostgreSQL + TimescaleDB | Timescale Cloud / RDS + TimescaleDB AMI | Managed service preferred |
| Redis | Elasticache / Upstash / Memorystore | Managed service preferred |
| MinIO | AWS S3 / GCS / R2 | Swap MinIO SDK endpoint to S3 endpoint; API compatible |

### Migration Checklist

1. Replace Docker network service discovery with environment-variable-based URLs
2. Add health check endpoints for cloud LB (already implemented: `/healthz`, `/readyz`)
3. Configure Redis PubSub for multi-instance WS server (already designed for this)
4. Set up managed database with connection pooling (PgBouncer or built-in)
5. Move MinIO bucket to S3 -- only change: endpoint URL in config
6. Add CI/CD pipeline for container image builds and deployments
7. Configure secrets management (AWS Secrets Manager, GCP Secret Manager, etc.)

---

## 14. Testing Architecture

The project follows **Test-Driven Development (TDD)**: write a failing test first (RED), implement just enough to pass (GREEN), then refactor (REFACTOR). This section defines the testing strategy across every layer.

### 14.1 Test Pyramid Strategy

```
                    ╱╲
                   ╱ E2E ╲             ~10 tests per phase
                  ╱────────╲           Playwright + Docker Compose
                 ╱Integration╲         ~100 tests total
                ╱──────────────╲       testcontainers, MSW, httptest
               ╱   Unit Tests   ╲      ~500+ tests total
              ╱──────────────────╲     go test, Vitest, RTL
             ╱════════════════════╲
```

| Tier | Scope | Runtime Budget | Trigger | Go Tools | Frontend Tools |
|------|-------|---------------|---------|----------|---------------|
| **Unit** | Single function/component, no I/O | < 30s total | Every save, pre-commit | `go test`, table-driven | Vitest, RTL |
| **Integration** | Real DB, real Redis, real API mocks | < 3m total | Pre-push, CI | testcontainers-go, httptest | MSW, Playwright components |
| **E2E** | Full Docker Compose stack, browser | < 10m total | CI only | N/A | Playwright |

### 14.2 Go Backend Testing Strategy

#### 14.2.1 Table-Driven Tests

All Go tests follow the canonical table-driven pattern:

```go
func TestFunctionName(t *testing.T) {
    tests := []struct {
        name     string
        input    InputType
        expected OutputType
        wantErr  bool
    }{
        {"valid input", validInput, expectedOutput, false},
        {"empty input", emptyInput, zeroValue, true},
        // ...
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := FunctionName(tt.input)
            if tt.wantErr {
                require.Error(t, err)
                return
            }
            require.NoError(t, err)
            assert.Equal(t, tt.expected, got)
        })
    }
}
```

#### 14.2.2 Interface-Based Dependency Injection

All services accept interfaces for their dependencies, enabling unit testing with mocks and integration testing with real implementations.

| Interface | Methods | Real Implementation | Mock Usage |
|-----------|---------|-------------------|------------|
| `Store` | CRUD for all entities | sqlc-generated queries | Unit tests for handlers and services |
| `S3Client` | PutObject, GetObject, RemoveObject, PresignedURL | MinIO SDK wrapper | Unit tests for upload/download logic |
| `SessionStore` | Create, Get, Delete, Refresh | Redis client | Unit tests for auth middleware |
| `JobQueue` | Enqueue, Consume, Ack | Redis Streams | Unit tests for API handlers producing jobs |
| `FaceitAPI` | GetProfile, GetMatches, GetMatchDetail | HTTP client | Unit tests for sync worker |

#### 14.2.3 Integration Tests with testcontainers-go

Integration tests use ephemeral containers per test suite, not shared databases:

```go
//go:build integration

func TestMain(m *testing.M) {
    ctx := context.Background()
    pg := testutil.StartPostgres(ctx)    // TimescaleDB container
    redis := testutil.StartRedis(ctx)     // Redis container
    defer pg.Terminate(ctx)
    defer redis.Terminate(ctx)

    testutil.RunMigrations(pg.ConnectionString())
    os.Exit(m.Run())
}
```

- **Build tag**: `//go:build integration` separates integration tests from fast unit tests
- **Run unit only**: `go test ./...`
- **Run integration**: `go test -tags integration ./...`
- **Run all**: `go test -tags integration ./...`
- **Container reuse**: One container per test suite (`TestMain`), not per test function

#### 14.2.4 HTTP Handler Tests

Handler tests use `httptest.NewRecorder` with the real chi router but mock service-layer dependencies:

```go
func TestDemoUploadHandler(t *testing.T) {
    mockStore := &mocks.MockStore{}
    mockS3 := &mocks.MockS3Client{}
    mockQueue := &mocks.MockJobQueue{}

    handler := handler.NewDemoHandler(mockStore, mockS3, mockQueue)
    router := chi.NewRouter()
    router.Post("/api/v1/demos", handler.Upload)

    // Create multipart request...
    req := httptest.NewRequest("POST", "/api/v1/demos", body)
    rec := httptest.NewRecorder()
    router.ServeHTTP(rec, req)

    assert.Equal(t, http.StatusAccepted, rec.Code)
}
```

This tests routing, middleware, request parsing, and response serialization without network or database I/O.

#### 14.2.5 Demo Parser Tests with Fixture Files

The demo parser (highest-risk task in the project) uses **golden file testing**:

- **Fixture files**: Small, real CS2 `.dem` files (< 5 MB) stored in `backend/testdata/demos/`
- **Golden files**: Expected parse output stored as `.golden.json` in `backend/testdata/golden/`
- **Update flag**: Run `go test -update` to regenerate golden files after intentional changes
- **Coverage**: At least 3 fixture demos covering normal match, overtime, and bot/disconnect edge cases

```go
func TestParseDemo(t *testing.T) {
    result := parser.Parse("../../testdata/demos/small_match.dem")
    golden := testutil.GoldenFile(t, "small_match_events", result)
    assert.Equal(t, golden, result)
}
```

Unit tests separately cover edge case functions: warmup detection, overtime detection, bot filtering, tick sampling rate.

#### 14.2.6 sqlc Query Tests

sqlc queries are tested against a real PostgreSQL + TimescaleDB instance via testcontainers:

- Run the actual generated Go code against a real database
- Seed test data, execute queries, assert results
- Catches SQL regressions when `backend/queries/*.sql` files are modified
- Particularly important for complex queries: tick range retrieval, heatmap aggregation, cross-demo stats

### 14.3 Frontend Testing Strategy

#### 14.3.1 Vitest Configuration

```typescript
// vitest.config.ts
export default defineConfig({
  test: {
    environment: 'jsdom',
    setupFiles: ['./src/test/setup.ts'],
    globals: true,
    css: false,
  },
  resolve: { alias: { '@': '/src' } },
})
```

#### 14.3.2 React Testing Library

Custom `renderWithProviders()` helper wraps components in all required providers:

```typescript
// src/test/render.tsx
export function renderWithProviders(
  ui: React.ReactElement,
  options?: { initialRoute?: string }
) {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return render(ui, {
    wrapper: ({ children }) => (
      <QueryClientProvider client={queryClient}>
        <MemoryRouter initialEntries={[options?.initialRoute ?? '/']}>
          {children}
        </MemoryRouter>
      </QueryClientProvider>
    ),
  })
}
```

#### 14.3.3 MSW (Mock Service Worker)

API mocks intercept `fetch` at the network level for realistic testing of TanStack Query hooks:

```typescript
// src/test/msw/handlers.ts
export const handlers = [
  http.get('/api/v1/auth/me', () => HttpResponse.json({ data: mockUser })),
  http.get('/api/v1/demos', () => HttpResponse.json({ data: mockDemos, meta: { page: 1, total: 5 } })),
  // ...per-feature handlers added as features are built
]
```

#### 14.3.4 Zustand Store Tests

Stores are tested as pure units -- no React rendering needed:

```typescript
import { useViewerStore } from '@/stores/viewer'

beforeEach(() => useViewerStore.setState(useViewerStore.getInitialState()))

test('setSpeed updates playback speed', () => {
  useViewerStore.getState().setSpeed(2.0)
  expect(useViewerStore.getState().speed).toBe(2.0)
})
```

#### 14.3.5 PixiJS / Canvas Layer Testing

PixiJS code uses a **split testing strategy**:

| Aspect | Approach | Example |
|--------|----------|---------|
| **Logic** (transforms, interpolation, state) | Classical TDD with unit tests | `worldToPixel(x, y, calibration)` tested with known coordinate pairs |
| **Visual rendering** (draw calls, sprites) | Screenshot tests with Playwright | Render PlayerLayer with mock data, compare to reference screenshot |

The "brain" is TDD'd; the "eyes" are screenshot-tested. This avoids forcing classical TDD on raw PixiJS draw calls where visual output is the real specification.

#### 14.3.6 TanStack Query Hook Tests

```typescript
test('useDemos fetches demo list', async () => {
  const { result } = renderHook(() => useDemos(), { wrapper: createWrapper() })
  await waitFor(() => expect(result.current.isSuccess).toBe(true))
  expect(result.current.data).toHaveLength(5)
})
```

#### 14.3.7 Yjs Collaboration Tests

Yjs tests use in-memory providers -- no WebSocket server needed:

```typescript
test('two docs converge on shared map', () => {
  const doc1 = new Y.Doc()
  const doc2 = new Y.Doc()

  // Sync mechanism: apply updates from one to the other
  doc1.on('update', (update) => Y.applyUpdate(doc2, update))
  doc2.on('update', (update) => Y.applyUpdate(doc1, update))

  doc1.getMap('board').set('title', 'A Site Execute')
  expect(doc2.getMap('board').get('title')).toBe('A Site Execute')
})
```

### 14.4 Test Database Strategy

- Tests **never** use a shared database. Each integration test suite gets an ephemeral container via testcontainers-go
- Migrations run automatically during `TestMain` setup
- Frontend tests use MSW (no database needed)
- Test data is inserted per test or per suite, never persisted between runs
- `TRUNCATE` between tests within a suite for isolation without container restart overhead

### 14.5 Fixture Management

| Fixture Type | Location | Strategy |
|--------------|----------|----------|
| `.dem` files (small, < 5 MB) | `backend/testdata/demos/` | Committed to repo |
| `.dem` files (large, > 5 MB) | Downloaded in CI | `make download-test-fixtures` |
| API response mocks | `frontend/src/test/msw/handlers.ts` | MSW handlers, co-evolve with API |
| Golden files (parser output) | `backend/testdata/golden/` | JSON snapshots, `-update` flag |
| Golden files (sqlc queries) | `backend/testdata/golden/queries/` | Expected result sets |
| Map calibration test data | `frontend/src/lib/maps/__tests__/fixtures/` | Known world→pixel coordinate pairs |
| Radar images (test) | `frontend/public/maps/` | Same as production (small PNGs) |

### 14.6 CI Test Pipeline Flow

```
┌──────┐    ┌────────────┐    ┌─────────────────┐    ┌───────┐    ┌────────────┐
│ Lint │───▶│ Unit Tests │───▶│Integration Tests│───▶│ Build │───▶│ E2E Tests  │
│      │    │ go test    │    │ testcontainers  │    │       │    │ Playwright │
│      │    │ pnpm test  │    │ MSW             │    │       │    │ Docker     │
└──────┘    └────────────┘    └─────────────────┘    └───────┘    └────────────┘
                                                          │
                                                     Each stage
                                                     gates the next
```

- **Lint** fails → nothing else runs
- **Unit tests** fail → no integration tests
- **Integration tests** fail → no build artifacts
- **Build** fails → no E2E
- **E2E** tests fail → PR cannot merge

### 14.7 Test File Conventions

| Codebase | Pattern | Location | Example |
|----------|---------|----------|---------|
| Go unit test | `*_test.go` (same package) | Colocated with source | `backend/internal/auth/session_test.go` |
| Go integration test | `*_integration_test.go` + `//go:build integration` | Colocated with source | `backend/internal/store/demo_integration_test.go` |
| Frontend unit/component | `*.test.ts` / `*.test.tsx` | Colocated with source | `frontend/src/stores/viewer.test.ts` |
| Frontend test helpers | `src/test/*` | Centralized | `frontend/src/test/render.tsx` |
| E2E test | `*.spec.ts` | `e2e/tests/` | `e2e/tests/demo-upload.spec.ts` |
| Golden files | `*.golden.json` | `backend/testdata/golden/` | `backend/testdata/golden/small_match_events.golden.json` |

---

*Cross-references: [PRD.md](PRD.md) for feature requirements, [IMPLEMENTATION_PLAN.md](IMPLEMENTATION_PLAN.md) for delivery phases, [TASK_BREAKDOWN.md](TASK_BREAKDOWN.md) for granular tasks.*

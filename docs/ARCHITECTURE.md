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
│   │   └── worker/                # Background job processor
│   ├── migrations/                # SQL migration files (golang-migrate)
│   ├── queries/                   # sqlc SQL files
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
│   │   └── utils/                 # Utility functions
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

*Cross-references: [PRD.md](PRD.md) for feature requirements, [IMPLEMENTATION_PLAN.md](IMPLEMENTATION_PLAN.md) for delivery phases, [TASK_BREAKDOWN.md](TASK_BREAKDOWN.md) for granular tasks.*

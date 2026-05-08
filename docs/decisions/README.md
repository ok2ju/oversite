# Architecture Decision Records

This directory captures key architectural decisions for Oversite. Each ADR records the context, decision, and consequences of a significant technical choice.

**How to add a new ADR:** Copy [template.md](template.md), number it sequentially, and add it to the table below.

| # | Decision | Status |
|---|----------|--------|
| [0001](0001-pixijs-outside-react.md) | Run PixiJS outside the React render tree | Accepted |
| [0002](0002-yjs-dumb-relay.md) | Use Yjs with a dumb WebSocket relay for strategy board collaboration | Deprecated |
| [0003](0003-redis-streams-job-queue.md) | Use Redis Streams as the job queue | Deprecated |
| [0004](0004-timescaledb-tick-data.md) | Use TimescaleDB hypertable for tick-level player position data | Superseded by ADR-0008 |
| [0005](0005-faceit-oauth-pkce.md) | Faceit OAuth 2.0 + PKCE as sole authentication method | Superseded by ADR-0014 |
| [0006](0006-desktop-app-pivot.md) | Pivot from web application to desktop application | Accepted |
| [0007](0007-wails-framework.md) | Use Wails as the desktop application framework | Accepted |
| [0008](0008-sqlite-local-database.md) | Use SQLite as the local database | Accepted |
| [0009](0009-loopback-oauth-desktop.md) | Loopback OAuth flow for desktop authentication | Superseded by ADR-0014 |
| [0010](0010-sqlc-type-safe-sql.md) | sqlc for type-safe SQL generation | Accepted |
| [0011](0011-zustand-state-management.md) | Zustand for frontend state management | Accepted |
| [0012](0012-tanstack-query-wails-bindings.md) | TanStack Query for Wails binding responses | Accepted |
| [0013](0013-logging.md) | Logging strategy | Accepted |
| [0014](0014-remove-faceit-integration.md) | Remove Faceit integration; ship as single-tenant local tool | Accepted |
| [0015](0015-streaming-parse-ingest-pipeline.md) | Streaming parse → ingest pipeline (`WithTickSink` channel) | Accepted |
| [0016](0016-sqlite-multi-connection-pool.md) | Multi-connection SQLite pool with `busy_timeout` | Accepted |
| [0017](0017-parser-defense-in-depth.md) | Parser defense-in-depth — independent watchdog + entity-panic opt-in | Accepted |

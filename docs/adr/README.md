# Architecture Decision Records

This directory captures key architectural decisions for Oversite. Each ADR records the context, decision, and consequences of a significant technical choice.

**How to add a new ADR:** Copy [template.md](template.md), number it sequentially, and add it to the table below.

| # | Decision | Status |
|---|----------|--------|
| [0001](0001-pixijs-outside-react.md) | Run PixiJS outside the React render tree | Accepted |
| [0002](0002-yjs-dumb-relay.md) | Use Yjs with a dumb WebSocket relay for strategy board collaboration | Accepted |
| [0003](0003-redis-streams-job-queue.md) | Use Redis Streams as the job queue | Accepted |
| [0004](0004-timescaledb-tick-data.md) | Use TimescaleDB hypertable for tick-level player position data | Accepted |
| [0005](0005-faceit-oauth-pkce.md) | Faceit OAuth 2.0 + PKCE as sole authentication method | Accepted |

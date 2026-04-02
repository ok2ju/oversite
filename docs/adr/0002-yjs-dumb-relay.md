# ADR-0002: Use Yjs with a Dumb WebSocket Relay for Strategy Board Collaboration

**Date:** 2026-03-31
**Status:** Accepted

## Context

The strategy board feature requires real-time collaborative editing — multiple users drawing arrows, placing icons, and adding notes on a shared CS2 map canvas simultaneously. The server needs to relay changes between clients and persist board state.

The key constraint: the backend is Go. Any server-side document logic must either be implemented in Go or avoided entirely.

### Alternatives considered

| Approach | Why rejected |
|----------|-------------|
| **Operational Transformation (ShareJS/OT)** | Requires the server to understand and transform operations. No mature Go OT library exists. Would need to implement a complex state machine in Go or run a Node.js sidecar. |
| **Yjs with server-side parsing** | Go would need to decode/encode Yjs binary format to validate or transform operations. No Go Yjs implementation exists; would couple Go to Yjs internals. |
| **Firebase Realtime DB / Firestore** | Adds external dependency and vendor lock-in. Pricing scales with operations, problematic for high-frequency drawing updates. |

## Decision

Use Yjs (CRDT) in the browser for all document logic. The Go WebSocket server acts as a dumb relay: it receives binary Yjs sync/awareness messages from one client and broadcasts them to all others in the same room. On disconnect (last client leaves), the server encodes the full Yjs document state to binary and stores it in `strategy_boards.yjs_state` (BYTEA column).

The server never inspects, validates, or transforms Yjs message contents.

## Consequences

### Positive

- Go server stays simple — just WebSocket room management and binary message forwarding
- Conflict resolution is mathematically guaranteed by CRDT properties
- Offline editing works out of the box — changes merge on reconnect
- Cursor/presence awareness is built into Yjs's awareness protocol
- No need for a Node.js sidecar or Go Yjs library

### Negative

- Server cannot validate drawing operations (must trust client input)
- Yjs document grows with edit history; requires periodic garbage collection or snapshots
- Cannot generate server-side thumbnails of strategy boards without a JS runtime (future: headless browser sidecar)
- Binary protocol makes server-side debugging harder — messages are opaque blobs

# ADR-0003: Use Redis Streams as the Job Queue

**Date:** 2026-03-31
**Status:** Deprecated (desktop app runs parsing in-process; no Redis)

## Context

Demo parsing and Faceit data syncing are CPU-intensive, long-running operations (parsing a single demo takes 10-30 seconds). These must run asynchronously — the API server enqueues a job, a separate worker process consumes it.

Redis is already in the stack for session storage and caching, so a Redis-based queue avoids adding another infrastructure dependency.

### Alternatives considered

| Approach | Why rejected |
|----------|-------------|
| **RabbitMQ** | Proven message broker, but adds a new service to Docker Compose, requires its own management/monitoring, and is overkill for the current scale (single worker, 2 job types). |
| **AWS SQS / GCP Pub/Sub** | Vendor lock-in. Adds cloud dependency to a Docker Compose-first project. Would complicate local development. |
| **PostgreSQL LISTEN/NOTIFY + polling** | No built-in consumer groups, no message persistence after consumption, requires custom retry/dead-letter logic. |
| **Go channels (in-process)** | Jobs lost on crash. Couples API server and worker into a single process, breaking the separate-scaling model. |

## Decision

Use Redis Streams with consumer groups. The API server produces jobs via `XADD` to named streams (`demo_parse_jobs`, `faceit_sync_jobs`). The worker process reads via `XREADGROUP`, acknowledges on success (`XACK`), and retries on failure (max 3 attempts). After max retries, jobs are moved to a dead-letter stream.

## Consequences

### Positive

- Zero new infrastructure — Redis is already required for sessions/cache
- Consumer groups give exactly-once delivery semantics within the group
- Built-in message persistence — unacknowledged messages survive Redis restarts (with AOF)
- Easy to scale horizontally by adding workers to the consumer group
- `XINFO` commands provide queue depth, pending count, and consumer lag for monitoring

### Negative

- Redis Streams is less feature-rich than RabbitMQ (no routing, no priority queues, no delayed messages out of the box)
- If Redis goes down, both sessions and job queue are unavailable (single point of failure — acceptable for v1, mitigated by Redis persistence)
- Dead-letter handling is manual (custom logic, not a broker-native feature)

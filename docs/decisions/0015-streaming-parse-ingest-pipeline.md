# ADR-0015: Streaming Parse → Ingest Pipeline

**Date:** 2026-05-08
**Status:** Accepted

## Context

Demo parsing buffered the full `ParseResult.Ticks` slice (800K+ rows × ~150 B = 100 MB+) before ingest started. Combined with retained event state, peak heap during a 30-round import sat near the `maxHeapBytes = 4 GiB` watchdog ceiling, and the watchdog was doing load-bearing work — it was the only thing standing between a borderline demo and an OOM kill of the Wails process.

The goals of this change:

- Drop peak heap on the tick path from "tens to hundreds of MB" to "small constant".
- Keep `maxHeapBytes` as a true safety net (corrupt-demo defense), not a normal-path backstop.
- Don't break the post-processing passes that rely on the full event list.

### Alternatives considered

| Approach | Why rejected |
|----------|--------------|
| **Status quo (buffer all ticks then ingest)** | Linear and simple, but peak heap is unbounded with demo size and stresses the GC. The watchdog stays load-bearing. |
| **Stream both ticks and events** | `dropKnifeRounds`, `pairShotsWithImpacts`, and `ExtractGrenadeLineups` all walk events in tick order while retaining cross-event state (e.g. shot pairing keeps `lastShotIdx[attackerSteamID]` across the whole match). An earlier streaming-events attempt broke knife-round detection silently — symptoms only showed up on demos with mid-round reconnects. The risk-to-reward isn't worth it. |
| **Callback-style parser API** (`Parse(r, onTick, onEvent, onRound, ...)`) | Invasive — touches every call site, and the callback discipline (no blocking, no allocation) is hard to enforce. Channels give us the same shape with explicit backpressure. |
| **Bounded buffer with periodic explicit flushes from parser** | Reproduces a channel poorly. Adds a flush cadence parameter we'd have to tune. |

## Decision

Introduce a tick-only streaming pipeline between parser and ingester.

- **Parser opt-in via `WithTickSink(chan<- TickSnapshot)`.** When set, the parser pushes `TickSnapshot` values into the channel via a `pushTick` helper that selects on `ctx.Done()`. The channel is closed by the parser via `defer` so the ingester's `range` terminates cleanly on parse end (or on ctx cancellation). Without `WithTickSink`, behavior is unchanged.
- **`IngestStream(ctx, demoID, ticksIn <-chan TickSnapshot)` is the canonical ingest entrypoint.** It owns the single ingest transaction, ctx-cancel, and `recover()`. Slice-based `Ingest(demoID, ticks)` is now a thin fan-into-channel adapter so batch logic only lives in one place.
- **`app.go parseDemo` runs parse + ingest concurrently via `errgroup.WithContext`** over a buffered channel (`DefaultTickSinkBuffer = 5000`). Either side returning an error cancels the other.
- **Events stay in `state.events`.** The post-processing passes (knife-round filtering, shot pairing, grenade lineups) continue to walk a complete in-memory event list. Streaming events is explicitly out of scope for this ADR — see "Alternatives considered".
- **`maxHeapBytes = 4 GiB` watchdog is retained.** The tick path is now bounded, but `demoinfocs`'s internal protobuf and entity-table state still grows on corrupt demos, and that's where the watchdog earns its keep.

## Consequences

### Positive

- Peak tick-path heap drops from ~120 MB to ~4 MB on a 30-round demo (~30×).
- Channel-based pipeline naturally backpressures the parser when the ingest tx is the bottleneck — no ad-hoc rate limiting needed.
- Cancellation flows cleanly: ctx cancel from `Shutdown` propagates through `errgroup.WithContext` to both goroutines.
- The watchdog returns to its intended role as a safety net rather than a normal-path floor.
- Slice-based `Ingest` retained as a thin adapter, so existing tests and any non-streaming caller still work without change.

### Negative

- Two-goroutine pipeline is more complex than buffer-then-ingest; debugging requires reasoning about channel closure ordering and `errgroup` cancellation semantics.
- `DefaultTickSinkBuffer = 5000` is a tuning constant. Too small starves the ingester; too large gives back the heap savings. Not a hot-tuned number — chosen empirically as "enough to absorb sqlc tx commit latency without filling on a fast parser".
- Event path is still buffered in memory. Full memory bound during parse is now dominated by parser-internal state, not our slice — that's why the watchdog stays.
- `Parse` signature changed (now takes `ctx`); test fixtures and any external callers had to update.

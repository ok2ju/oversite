# ADR-0018: Auto-Retry Parse with Entity-Panic Tolerance on `ErrCorruptEntityTable`

**Date:** 2026-05-08
**Status:** Accepted

Builds on [ADR-0017](0017-parser-defense-in-depth.md). That ADR made `IgnorePacketEntitiesPanic` opt-in and exposed `SetTolerateEntityErrors` as a Wails binding for users who needed the previous swallow-and-continue behavior. This ADR closes the gap where the binding has no UI surface, leaving users dead-ended on the very first corrupt-entity demo.

## Context

A user testing the v0.1.8 release on Windows reported a parse failure on a 316 MB demo:

```
parsing demo: unable to find existing entity 1647731180
```

The error fires ~25 ms into parsing — the entity table is damaged from byte zero, almost certainly genuine corruption (an entity ID of 1.6B is far outside the valid CS2 range). The parser correctly recovered the panic and surfaced `ErrCorruptEntityTable` per ADR-0017. The user-facing message reads:

> parse failed: demo has a corrupt entity table; parsing was stopped to avoid running out of memory

ADR-0017 documented the escape hatch as: *"a user who needs partial-parse tolerance for their specific demo can flip the bool via the new `SetTolerateEntityErrors` Wails binding."* The binding does exist in `frontend/wailsjs/go/main/App.d.ts`, but no UI calls it — `grep -r SetTolerateEntityErrors frontend/src` returns nothing. The Settings surface that ADR-0017 anticipated was never built, so the documented opt-in is unreachable. The user has no path forward.

The motivating failure here is not memory safety — that's well-handled by the heap watchdog. It's a UX dead-end created by the gap between "binding exists" and "user can reach it."

The goals of this change:

- A user with a demo that fails on the entity-panic recovery should successfully import without intervention, when memory safety can be guaranteed by the existing watchdog.
- The default fail-fast posture from ADR-0017 should still apply on the *first* parse attempt — the heap watchdog is engineered to catch corruption early, and we want that signal to fire before we hand the parser more rope.
- The retry must be safe from data-corruption side effects (no half-committed tick rows surviving into the second attempt).
- The decision is reversible without rebuilding — the existing `SetTolerateEntityErrors` binding still works for callers who want to skip the first attempt entirely.

### Alternatives considered

| Approach | Why rejected |
|----------|--------------|
| **Status quo (fail with `ErrCorruptEntityTable`, surface to UI)** | Documented opt-in path is unreachable; users dead-end. Re-implementing user-facing recovery requires a Settings UI surface that hasn't been built and is not on the immediate roadmap. |
| **Default `WithIgnoreEntityPanics(true)`** | Reverts the core safety call from ADR-0017. The first-attempt fail-fast path is a deliberate safety check — corrupt demos that *do* drive runaway state get bounded by the watchdog at fail-fast latency, not at ~3 GiB after the watchdog has to step in. Auto-retry preserves that safety on attempt 1 and only opts into tolerance after the demo has demonstrably tripped the recoverable case. |
| **Build a Settings UI for `SetTolerateEntityErrors`** | Solves the dead-end but pushes a UX burden onto users who shouldn't need to know about parser internals. "I imported a demo and it failed; I should toggle a setting that says 'tolerate entity errors' to retry" is poor product. The setting is most useful as a power-user override for batches of known-corrupt demos, not as the standard recovery path. |
| **Retry N times with widening tolerance** | Only one knob meaningfully changes between attempts (`IgnorePacketEntitiesPanic`). A second retry with the same flag would be cargo-cult. |
| **Surface a "Retry with tolerance" button per demo** | Requires per-demo state tracking, button rendering in the demo card, and another Wails binding. Worth doing later if the auto-retry signal turns out to be too aggressive — punted to a follow-up if telemetry shows it. |

## Decision

`app.go parseDemo` runs the parse pipeline twice, conditionally:

1. **First attempt** with the user-configured tolerance (default `false` per ADR-0017).
2. **Auto-retry once** if the first attempt returns an error matching `errors.Is(err, demo.ErrCorruptEntityTable)` *and* the user had not already enabled tolerance.

Implementation:

- The streaming parse+ingest pipeline (parser + ingester + errgroup) is extracted into a new helper `runParsePipeline(demoID, f, tolerateEntityErrors, emitProgress)` so both attempts share the errgroup wiring and don't drift.
- Between attempts, the file is rewound with `f.Seek(0, io.SeekStart)` and the progress bar is reset to 0% via `emitProgress("parsing", 0)`.
- Retry path emits `slog.Warn("parseDemo: retrying with entity-panic tolerance", "first_err", err.Error())` so `errors.txt` records when the second attempt was triggered. This gives us the telemetry to revisit the decision: if every other parse trips the retry, we'd flip the default; if it almost never trips, we'd consider removing the layer.

The retry is safe without DB-level cleanup because:

- `IngestStream` runs everything in a single transaction with `defer tx.Rollback()` (`ingest.go:100`), so a failed first attempt commits zero rows.
- Even if a previous import had committed rows for this `demoID`, the next call's `DeleteTickDataByDemoID` (`ingest.go:103`) wipes them as the first step of the new transaction.
- The heap watchdog (ADR-0017) and the tick/event caps (`parser.go:34-37`) are both still active on the retry. The flag flip changes only how demoinfocs handles entity-update panics — it does not disable the safety net that motivated ADR-0017's defaults.

The `SetTolerateEntityErrors` Wails binding is retained. It now serves a refined purpose: skip the first (likely-failing) attempt for users who are re-importing a known-corrupt batch and don't want to pay the wasted parse time. It's no longer the primary recovery path.

## Consequences

### Positive

- Users with corrupt-entity demos succeed without ever touching settings — the previous dead-end is closed.
- The first-attempt fail-fast posture from ADR-0017 is preserved: the watchdog still gets to fire on the cheaper, no-tolerance path before we give the parser more rope.
- Telemetry is built in. The `parseDemo: retrying with entity-panic tolerance` log line gives us a count of how often the second attempt is needed in the wild — feeding the next "should we flip the default?" decision with data instead of speculation.
- The pipeline extraction into `runParsePipeline` removes ~45 lines of nested goroutine logic from `parseDemo` and centralizes it in one helper. Future changes to the streaming pattern (a third attempt, a different ingestion backend) live in one place.
- All layers stay reversible. `SetTolerateEntityErrors(true)` skips the first attempt; deleting the retry block reverts to ADR-0017 behavior; the helper extraction is pure refactor.

### Negative

- Worst-case parse time on corrupt demos doubles (one full failed attempt + one full successful attempt). The retry only runs when the first attempt errored on a recovered panic, not when it ran to completion — for the user's 316 MB demo the failed attempt aborts in ~25 ms, so the doubled time is dominated by the successful pass. Acceptable.
- Two parse passes mean two pprof dumps if the heap watchdog trips on both attempts — bounded by the existing 5-file cap in `pruneOldProfiles`.
- A demo that genuinely should fail (e.g., a non-CS2 file misidentified as a demo) now produces an error path with two log entries instead of one. The `slog.Warn` retry line makes the doubled output unambiguous.
- This decision narrows ADR-0017's intent: that ADR positioned the entity-panic-off default as a hard fail-fast safety check; we've now layered a recovery on top. The combined story is "fail fast on attempt 1, tolerate on attempt 2." If a future failure mode demonstrates that the retry hands the runaway too much rope despite the watchdog, the retry is the layer to remove — not the default flag.
- The auto-retry can mask a regression in the heap watchdog. If a future change makes the watchdog less aggressive, a runaway on the retry path could go undiagnosed for longer because the surface error would be the same as today's recovery. Mitigated by the per-attempt pprof dumps and the `first_err` field in the warn log.

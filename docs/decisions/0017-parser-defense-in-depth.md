# ADR-0017: Parser Defense-in-Depth — Independent Watchdog + Entity-Panic Opt-In

**Date:** 2026-05-08
**Status:** Accepted

Builds on [ADR-0015](0015-streaming-parse-ingest-pipeline.md) (streaming pipeline that bounded the tick path) by hardening the remaining failure mode: parser-internal state growth on corrupt demos. Does not supersede ADR-0015 — that ADR's heap watchdog is one of the layers retained here.

## Context

A user reported that importing a 325 MB CS2 demo on a 16 GB Windows host drove `oversite.exe` to 13 GB of active private working set before the parser was cancelled. The same demo imports cleanly on macOS at ~120 MB. Two structural defects let the runaway happen:

1. **The heap watchdog was tied to `events.FrameDone`.** Its limit-check ran inside the FrameDone handler at `parser.go:797-820`, every 10K frames. The demoinfocs library does substantial pre-frame work — string-table parsing, entity baselines, DataTable decoding, SendTable bootstrap — before the first FrameDone fires. On a corrupt demo where the entity table is damaged, that pre-frame phase is exactly where the library accumulates state, but the watchdog cannot fire because the dispatcher hasn't yielded yet. The user's heartbeat log confirmed: zero heartbeat lines before the blow-up.
2. **`config.IgnorePacketEntitiesPanic = true` was unconditional** (set in commit `2329a1d` to stop crashes on certain Windows POV demos). With it on, demoinfocs swallows "unable to find existing entity" panics and continues parsing. On a pathological demo that produces an unbounded internal accumulation loop. The `2329a1d` fix traded a visible crash for a silent 13 GB working-set blow-up.

`GOMEMLIMIT` (set by ADR-0015's sysinfo work) is a soft GC target, not allocation rejection. When the data is genuinely live, GC has nothing to free and the heap grows past the target — `GOMEMLIMIT` cannot be load-bearing for this failure mode.

The goals of this change:

- Catch parser-memory blow-ups regardless of which demoinfocs phase is running (including pre-frame).
- Default to fail-fast on entity-table corruption, but keep tolerance available behind an explicit user opt-in.
- Produce a triage artifact (heap pprof) on every kill so we can attribute the next blow-up to a specific upstream issue (string tables vs. entity tables vs. our own code).
- Keep all the changes reversible and behind options/flags so a user-reported regression is one toggle away from the previous behavior.

### Alternatives considered

| Approach | Why rejected |
|----------|--------------|
| **Status quo (FrameDone heartbeat only)** | The premise of the heartbeat — that the dispatcher yields between work units — doesn't hold during pre-frame setup. Demonstrably failed in production: 13 GB working set, zero heartbeat lines logged. |
| **Raise the heap limit / `GOMEMLIMIT`** | Doesn't solve anything. The user's Windows host has 16 GB total; the parser was using 13 GB of it. There is no headroom to give. The actionable fix is to *catch the runaway earlier*, not give it more rope. |
| **Drop `IgnorePacketEntitiesPanic` entirely (back to ADR-0015 pre-`2329a1d` crash behavior)** | Some real Windows POV demos still rely on it to parse at all. We don't want to regress users who imported successfully before. The opt-in path keeps that escape valve available. |
| **Subprocess parsing** (cmd/oversite-parse + `JOB_OBJECT_LIMIT_PROCESS_MEMORY` on Windows / `RLIMIT_AS` on Unix) | The bulletproof option — parent stays at ~80 MB regardless of what the child does. Rejected for now: ~10% IPC overhead, two binaries to ship via Wails `Resources`, and substantially more involved testing. We documented it as Phase 3 of the plan. Revisit if the watchdog telemetry shows the issue recurring across diverse demos rather than a single pathological file. |
| **Bigger `maxGameEvent` or `maxTickRows` cap** | Wrong direction. The slice caps already work on healthy demos; the blow-up is in demoinfocs-internal state that those caps don't see. |
| **Drop `runtime.GC()` and rely on `GOMEMLIMIT`** | `GOMEMLIMIT` triggers more frequent GC, not allocation rejection. The runtime is also slow to scavenge pages back to the OS on Windows specifically, so even when the heap shrinks, the working set stays high. We need an explicit `debug.FreeOSMemory()` to force the page release. |

## Decision

Three layered defenses plus diagnostic plumbing. Each layer is independently rollback-able.

### 1. Independent goroutine heap watchdog — `internal/demo/heap_watchdog.go` (new)

A goroutine started in `Parse()` (`go wd.Run(); defer wd.Stop()`) that polls `runtime.ReadMemStats` every 500 ms regardless of whether the demoinfocs dispatcher has yielded. Trip conditions:

- `mem.HeapAlloc > limit` (Go-heap), **or**
- `procMem.WorkingSetSize > limit + limit/2` (Windows OS-reported working set, when `sysinfo.ProcessMemory()` returns non-zero).

On trip:

1. Writes a heap pprof profile to `{AppData}/oversite/profiles/heap-{demoID}-{ts}.pprof` via `pprof.Lookup("heap").WriteTo(f, 0)`.
2. Calls a stop callback that sets `state.limitExceeded` and `Cancel()`s the parser.
3. Logs a single WARN line containing the dump path plus `MemStats` fields (`HeapAlloc`, `HeapSys`, `HeapIdle`, `HeapInuse`, `StackInuse`, `Sys`, `NextGC`, `NumGC`, `PauseTotalNs`) and the OS-reported counters (`WorkingSetSize`, `PrivateUsage`).
4. Stops itself; does not loop. `sync.Once` guards both the callback and the dump so a tight ticker can't spam.

Soft warning at 50% of the limit also fires once per parse — a breadcrumb for users who run close to the ceiling without tripping it.

The existing FrameDone heartbeat at `parser.go:797-820` is **retained**. It still does the right thing on healthy parses (cheap to evaluate, gives us in-context tick/event counts in errors.txt) and it doubles up the kill-switch check.

### 2. Make `IgnorePacketEntitiesPanic` opt-in — new `WithIgnoreEntityPanics(bool)` option

Default **false**. The existing `recover()` in `Parse` checks the panic message: if it contains `"unable to find existing entity"`, it returns the new sentinel error `ErrCorruptEntityTable` with a user-facing message ("Demo has a corrupt entity table. Parsing was stopped to avoid running out of memory."). A user who needs partial-parse tolerance for their specific demo can flip the bool via the new `SetTolerateEntityErrors` Wails binding — that re-enables the swallow behavior, accepting the higher peak-memory risk.

`config.IgnoreErrBombsiteIndexNotFound = true` stays unconditional — that flag skips a malformed bomb event without accumulating internal state.

### 3. Force OS page release after parse — `debug.FreeOSMemory()` in `app.go`

Called once after `result.Events = nil` / `result.Lineups = nil` / `result.Rounds = nil`. This is the only reliable way to drop the Windows working set after a memory-heavy operation; Go's runtime is slow to madvise/decommit unused pages there. Cost: ~50–200 ms once per import. Mostly a no-op on macOS/Linux where the runtime is already aggressive.

### Diagnostic plumbing

- `internal/sysinfo/procmem*.go` — new `ProcessMemory()` that returns `WorkingSetSize`/`PrivateUsage` from `psapi!GetProcessMemoryInfo` on Windows (via `NewLazySystemDLL`, same pattern as `GlobalMemoryStatusEx`); zeros on other platforms. Lets the watchdog log Windows-specific page retention alongside `MemStats`.
- `internal/database.ProfilesDir()` — parallel to `DemosDir()`. Creates `{AppData}/oversite/profiles/` and prunes oldest *.pprof files keeping the last 5, so a reproduction loop doesn't fill the disk.
- New Wails bindings on `App`: `ProfilesDir()`, `OpenProfilesFolder()`, `GetTolerateEntityErrors()`, `SetTolerateEntityErrors(bool)` — exposed for a Settings UI surface.
- `maxGameEvent` cap dropped 500K → 100K. The 500K headroom existed to absorb runaway events under the old swallow-and-continue regime; with the watchdog catching pathological cases earlier, 100K is still 2–3× the worst legitimate match.

## Consequences

### Positive

- A pathological demo aborts within ~1 second of crossing the heap ceiling instead of running away to 13 GB. Verified locally.
- Every kill produces a heap pprof dump on disk that the user can attach to a bug report. Top frames pinpoint whether the blow-up was in demoinfocs string tables, entity tables, or our own state — drives the next decision (Phase 3 subprocess vs. upstream demoinfocs PR vs. our own fix) without needing the user's demo file.
- Default fail-fast on entity-table corruption surfaces the underlying issue cleanly instead of swallowing it. Users who can't import a specific demo can opt back into tolerance via Settings — no rebuild required.
- The kill-switch is now genuinely load-bearing across the *whole* parse, not just the FrameDone phase. The previous design implicitly assumed pre-frame work was small and bounded; that assumption is no longer required.
- `debug.FreeOSMemory()` measurably drops the Windows working set after a healthy parse — Task Manager actually shows the parse memory disappearing, not held forever.
- All three defenses are independently reversible: `WithIgnoreEntityPanics(true)` restores `2329a1d` behavior, the watchdog can be disabled by passing `WithHeapLimit(0)` (or equivalent), `debug.FreeOSMemory()` is a single-line delete. No DB schema changes.

### Negative

- Two watchdog code paths (independent goroutine + FrameDone heartbeat) to maintain. The duplication is intentional — different blind spots — but a future refactor must remember why both exist.
- A demo that previously imported successfully under unconditional `IgnorePacketEntitiesPanic = true` and that triggers a "find existing entity" panic now fails by default. Mitigation: the error message tells the user to enable `Tolerate Entity Errors` in Settings, and the pprof dump tells us which demoinfocs path is responsible. Acceptable trade for surfacing the silent-failure mode.
- pprof dumps add disk usage (bounded to 5 files via `pruneOldProfiles`, but each can be tens of MB on a tripped 3 GiB heap). On a constrained drive this could matter; the bound keeps it under ~250 MB worst case.
- `sysinfo.ProcessMemory()` adds another `NewLazySystemDLL` lookup on Windows (`psapi.dll`). Cost is amortized — the proc handle is resolved once via `LazySystemDLL` and reused on every poll.
- The watchdog interval (500 ms) is a tuning constant. Too tight wastes CPU; too loose lets the heap climb further before kill. 500 ms gives ~30 µs/poll measured against a `runtime.ReadMemStats` plus the Windows syscall — negligible against a multi-second parse.
- `debug.FreeOSMemory()` blocks the calling goroutine for ~50–200 ms once per import. On macOS/Linux the cost is mostly wasted (the runtime already returned the pages); we eat it for portability rather than gating the call on `runtime.GOOS`.

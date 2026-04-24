# ADR-0013: File-based Logging with Dev-only Network Capture

**Date:** 2026-04-18
**Status:** Accepted

## Context

Oversite ships as a single-binary Wails desktop app. Users run it outside any server infrastructure, so stderr is discarded as soon as the app exits. Before this change the codebase logged through a mix of `log.Printf` and `slog.Info` calls, all going to stderr. Diagnosing a crash or a failed Faceit sync after the fact required asking the user to re-run from a terminal -- not realistic for end users.

A secondary concern: the `debugTransport` in `internal/auth/debug_transport.go` dumped HTTP request/response pairs when `OVERSITE_DEBUG_HTTP=1` was set. Useful during development, but it also wrote to stderr, so its output was just as ephemeral.

We need:

1. A persistent record of warnings and errors that survives across app restarts.
2. A richer HTTP capture that only runs in dev builds, without relying on an env-var flag that developers have to remember.

## Decision

Introduce a small `internal/logging/` package that owns both concerns.

- **`errors.txt`** -- always on. WARN+ records go through `slog.TextHandler` into `{AppDataDir}/logs/errors.txt` (and also to stderr). Rotated at 5MB with 3 backups via `gopkg.in/natefinch/lumberjack.v2` (already a transitive dep, now direct). A stdlib `log.SetOutput` bridge captures bare `log.Printf` calls as WARN records so un-migrated call sites are not lost.
- **`network.txt`** -- dev only. Detected via `wailsRuntime.Environment(ctx).BuildType == "dev"` in `App.Startup`. Lifts the existing `debugTransport` logic into `logging.NewTransport` and writes to a rotated `network.txt` (same 5MB/3 backup policy). The transport is injected into the Faceit client, the demo download client, and the OAuth token exchange so every outbound HTTP call is captured in dev.
- The `OVERSITE_DEBUG_HTTP` env flag and `debug_transport.go` are removed.

Files live under `{AppDataDir}/logs/` alongside the SQLite DB.

## Consequences

### Positive

- Users can attach `errors.txt` to bug reports without having to reproduce from a terminal.
- HTTP traces during development no longer require setting an env var before launch.
- `log.Printf` call sites that haven't been converted still produce useful output instead of going to `/dev/null`.
- Rotation is automatic and bounded -- the app will never accumulate more than ~20MB of log files.

### Negative

- Introduces one more direct dependency (lumberjack).
- Process-global logger state (the package uses package-level vars) -- acceptable because a desktop app has exactly one main process.
- Frontend console logs are deliberately out of scope; crashes in the Wails webview still require DevTools.

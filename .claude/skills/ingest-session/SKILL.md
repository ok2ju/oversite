---
name: ingest-session
description: Summarize the current Claude Code session into docs/log.md and propose updates to relevant docs/knowledge/ wiki pages. Trigger manually when you want to capture what was learned in a session.
disable-model-invocation: true
---

# Ingest Session

Summarize the current Claude Code session and persist the non-obvious learnings into the project's Obsidian vault. The goal is a durable project record, not a minute-by-minute log.

## Scope

**In scope:**
- Append a one-line entry to `docs/log.md` describing what happened
- Identify which `docs/knowledge/*.md` pages the session's work affects
- Propose concrete edits (as diffs) for those knowledge pages — new facts, corrected assumptions, patterns learned

**Out of scope — never auto-edit these:**
- `docs/product/*` (PRD — canonical spec)
- `docs/architecture/*` (Arc42 — canonical spec)
- `docs/tasks.md`, `docs/roadmap.md` (canonical trackers)
- `docs/decisions/*` (ADRs — only the user creates/edits these)

If the session made a material architectural decision, **suggest** a new ADR using `docs/decisions/template.md` — do not create one silently.

## Workflow

1. **Scan the session.** Review the conversation for files touched, features added, bugs fixed, decisions made, and learnings (things that surprised you or weren't obvious from the code).

2. **Draft the log entry.** One line in `docs/log.md` at the top, under today's date header if it exists (create one if not). Format:
   ```
   YYYY-MM-DD — <summary> (files: [[...]], refs: [[knowledge/page]])
   ```
   Keep it under 150 characters. The log is an index, not a diary.

3. **Identify affected knowledge pages.** Match file touches against wiki stems:
   - Edits to `frontend/src/components/viewer/` or PixiJS code → `knowledge/pixijs-viewer.md`
   - Edits to `app.go` bindings or `runtime.EventsEmit` → `knowledge/wails-bindings.md`
   - Edits to `internal/database/` or schema pragmas → `knowledge/sqlite-wal.md`
   - Edits to `queries/*.sql` or `internal/store/` → `knowledge/sqlc-workflow.md`
   - Edits to `migrations/` → `knowledge/migrations.md`
   - Edits to `internal/auth/` or OAuth flow → `knowledge/faceit-oauth.md`
   - Edits to `frontend/src/lib/maps/` or calibration logic → `knowledge/coordinate-calibration.md`
   - Edits to `internal/demo/` or parser logic → `knowledge/demo-parser.md`
   - Edits to `*_test.go`, `*.test.tsx`, `internal/testutil/`, `src/test/` → `knowledge/testing.md`

   If nothing in the session maps to a wiki page, say so explicitly: **"No wiki updates needed for this session."** Do not invent reasons to edit.

4. **Propose edits as diffs.** For each affected page, show the user exactly what lines you'd add or change. Prefer additions to the "Gotchas" / "Known bug" / "Don't" sections over rewrites.

5. **Wait for confirmation.** Do not write to `docs/log.md` or any knowledge page until the user approves. Present the proposed changes, then ask "Apply these changes?"

6. **Consider an ADR.** If the session made a material architectural decision (new library, changed data flow, new binding pattern, new storage approach), tell the user: *"This looks like an ADR candidate — consider creating docs/decisions/NNNN-<name>.md from the template."* Do not create it yourself.

## Principles

- **Capture what's non-obvious.** Not "touched file X" — that's in git. Capture *why it was hard* or *what future-me needs to know*.
- **One ingestion per session.** If the user runs `/ingest-session` twice, check whether today's log entry already exists and update rather than duplicate.
- **Short is better than complete.** The wiki shouldn't absorb everything; it should absorb the bits worth remembering.
- **Never invent.** If you're not confident a learning actually happened in this session, omit it. Drift kills the wiki's trustworthiness faster than gaps do.

## What counts as "worth capturing"

Good candidates:
- A library version incompatibility you worked around
- A subtle invariant (e.g., "this event must fire before that one or X breaks")
- A failed approach that future-you would otherwise retry
- A gotcha that isn't in the README
- A pattern that emerged across multiple files

Skip:
- Routine CRUD additions
- Code style / formatting changes
- Anything the commit message already captures
- Test refactors that don't change test approach

## Example output

```
## Session summary

Fixed incendiary/molotov grenade extraction (P2-T06). Added `FireGrenadeStart` handler
in `internal/demo/parser.go` and `"fire_start"` to `detonationTypes` in the extractor.
Orphan grenade rate dropped from ~25% to ~2% (DecoyExpired remaining).

## Proposed log entry (docs/log.md)

2026-04-24 — Fixed incendiary/molotov grenade extraction; orphan rate 25% → 2% (see [[knowledge/demo-parser]])

## Proposed knowledge updates

docs/knowledge/demo-parser.md — update "Known bug" section:

  - **Fix (P2-T06):**
  + **Fixed in P2-T06 (2026-04-24):**
    1. Register `events.FireGrenadeStart` in `parser.go`, emit `"fire_start"` event type.
    2. Add `"fire_start": true` to `detonationTypes` in `grenade_extractor.go`.
  - 3. `DecoyExpired` is also unmatched, but the volume is small — lower priority.
  + Post-fix orphan rate: ~2% (remaining: `DecoyExpired`).

## ADR candidate

None — this was a bug fix, not a decision.

## Apply these changes? (y/n)
```

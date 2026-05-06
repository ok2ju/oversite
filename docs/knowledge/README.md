# Knowledge Wiki

LLM-maintained wiki of implementation entities and patterns. These pages capture the bits that aren't in the spec but matter day-to-day: gotchas, working conventions, rationale not captured by ADRs. Each page is short and topical; they grow from real friction, not from copying the PRD.

**Curated by:** the `/ingest-session` slash command after each Claude Code session.

## Entities

- [[pixijs-viewer]] — PixiJS application lifecycle and Zustand bridge
- [[wails-bindings]] — Go method → TS function conventions, progress events
- [[sqlite-wal]] — WAL mode, single connection rationale, modernc driver
- [[sqlc-workflow]] — queries/ → generate → store/ roundtrip
- [[migrations]] — golang-migrate embedding, up/down discipline
- [[coordinate-calibration]] — world-space ↔ pixel-space transform
- [[demo-parser]] — demoinfocs-golang findings + ongoing edge cases
- [[testing]] — shared test utilities by layer

## Conventions

- Keep pages short (< 200 lines). If a page grows, split it.
- Link freely with `[[page-name]]` wikilinks; Obsidian renders both.
- When in doubt whether something belongs here or in an ADR: if it's a **decision we made**, it's an ADR. If it's **knowledge we gained**, it's a wiki page.
- When in doubt whether it belongs here or in product/: product/ is the *spec*; knowledge/ is the *implementation notes*.

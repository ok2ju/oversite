# Project Log

Append-only chronological record of notable project activity. New entries at the top. Maintained via the `/ingest-session` slash command.

Format: `YYYY-MM-DD — <summary>` with links to affected pages.

---

## 2026-05-06 — Faceit integration removed

The app pivots away from the Faceit-account-tied desktop client to a single-tenant local tool. Removed:

- Backend: `internal/auth/`, `internal/faceit/`, `vars.go`, all auth/sync/download bindings in `app.go`
- Database: migration 005 drops `users`, `faceit_matches`, and the `demos.user_id` / `demos.faceit_match_id` columns; `queries/users.sql` and `queries/faceit_matches.sql` deleted; `demos.sql` no longer filters by user
- Frontend: dashboard, auth provider, login/callback/match-detail routes, all faceit hooks/store/types, Faceit MSW handlers and binding mocks; index now redirects to `/demos`
- CI: Faceit secrets and `LDFLAGS` injection removed from `.github/workflows/release.yml` (delete the `FACEIT_*` repo secrets manually)
- Docs: added [[decisions/0014-remove-faceit-integration|ADR-0014]] superseding [[decisions/0005-faceit-oauth-pkce|ADR-0005]] and [[decisions/0009-loopback-oauth-desktop|ADR-0009]] (both retained as historical superseded records); deleted `knowledge/faceit-oauth.md` and `plans/p4-faceit-heatmaps.md`; de-Faceit-ified the rest
- Known straggler: `internal/demo/compress.go:67` still uses `"faceit-demo-*.dem"` as the zstd-decompression temp-file prefix — cosmetic, unrelated to the integration; rename in a follow-up

The 2D demo player and per-demo analytics remain. Comprehensive per-player statistics built on `player_rounds` / `game_events` are a follow-up plan. Plan: `~/.claude/plans/i-ve-changed-the-original-frolicking-bengio.md`.

## 2026-04-24 — Docs vault reorganization

Restructured `docs/` into an Obsidian vault with split monoliths, a knowledge wiki, and an entry-point MOC.

- Split [[product/vision|PRD]] into 7 topical pages under `product/`
- Split [[architecture/overview|ARCHITECTURE]] into 8 Arc42-aligned pages under `architecture/`
- Renamed `adr/` → `decisions/`, `TASK_BREAKDOWN.md` → `tasks.md`, `IMPLEMENTATION_PLAN.md` → `roadmap.md`
- Lowercased phase plan filenames
- Seeded [[knowledge/README|knowledge wiki]] with 9 entity/pattern pages
- Absorbed `spike-parser-findings.md` into [[knowledge/demo-parser]]
- Added [[index]] as MOC and this log
- Updated root `README.md` to reference the new vault layout (MOC + subfolder links)

Rationale: the flat `docs/` folder with four 800–1250 line monoliths was hard to navigate and edit. Splits give cleaner URLs, smaller LLM edit surfaces, and surface-area for a curated wiki layer. See `/Users/sundayfunday/.claude/plans/im-thinking-about-having-binary-clover.md` for the plan that drove this change.

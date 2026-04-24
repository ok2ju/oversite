# Project Log

Append-only chronological record of notable project activity. New entries at the top. Maintained via the `/ingest-session` slash command.

Format: `YYYY-MM-DD — <summary>` with links to affected pages.

---

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

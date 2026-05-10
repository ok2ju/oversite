---
name: expand-slice
description: Expand a single slice from a slices.md file (produced by /slice-plan) into a detailed ticket file with acceptance criteria, files-to-touch, test plan, TDD workflow, and Definition of Done. Use after /slice-plan when you're about to start work on a specific slice. Just-in-time only — expand the next slice, not all at once. Outputs to .claude/tasks/{NNN}/{NN}-{slug}.md.
disable-model-invocation: true
---

# Expand Slice

Expand one slice from an existing `.claude/tasks/{NNN}/slices.md` into a working ticket. The slice already says **what** to ship and **why**; this skill adds **how** — acceptance criteria, files (verified against the codebase), tests keyed to project conventions, and a Definition of Done.

This is a **just-in-time** skill. Expand the slice you're about to start, not the whole backlog. Tickets go stale the moment a predecessor slice teaches you something new.

## Arguments

`$ARGUMENTS` may be:

- `<NNN> <N>` — expand slice `N` in task folder `NNN` (e.g. `001 3`)
- `<N>` — expand slice `N` in the **most recent** task folder under `.claude/tasks/`
- empty — expand the **next un-expanded** slice in the most recent task folder
- `--force` (combined with the above) — bypass the stale-predecessor guard

If multiple task folders exist and no `NNN` is given, list them and ask which one. Don't guess.

## Workflow

### 1. Resolve target task folder and slice number

1. `ls .claude/tasks/` — pick the folder per the rules above.
2. Read `slices.md`. Parse the `### Slice N — ...` headings into a list. If the requested `N` doesn't exist, error and show the available slice numbers.
3. List existing ticket files (`{NN}-*.md`) in the folder. If `<N>` is omitted, target the **lowest** slice number that has no ticket file yet.

### 2. Stale-predecessor guard

For any slice `N > 1`, check the predecessor ticket `{NN-1}-*.md`:

- If it doesn't exist → warn: *"Slice N-1 hasn't been expanded yet. Expanding ahead is allowed but the AC for slice N may shift after slice N-1 lands."*
- If it exists and its `**Status:**` line is anything other than `Merged` or `Done` → warn: *"Slice N-1 is `{status}`. Tickets expanded ahead of merged predecessors go stale fast."*

Don't refuse — show the warning, then ask whether to proceed. `--force` skips the prompt.

### 3. Read the slice and verify the codebase

Re-read **only the requested slice section** plus the file's `## North star`. Don't re-read every slice — they aren't relevant.

For every path mentioned in the slice's `**In scope:**` bullets:
- If the file exists → record current line numbers for any anchors the slice cites (e.g. *"around line 443 of `app.go`"*). Update stale line numbers in the output.
- If the file is new → mark it `(new)` in the Files-to-touch table.
- If the slice references a symbol (function, table, binding, hook), `grep` for it. If it already exists somewhere unexpected, surface that under Risks.

This step is what earns the ticket its detail. Skip it and you're just reformatting the slice.

### 4. Generate the ticket

Use the template below. Keep it scannable — tables for files and tests, checklists for AC and DoD.

**Acceptance criteria** — derive from the slice's Demo + each user-visible **In scope** bullet. Use Given/When/Then or testable imperatives ("the side panel renders one row per row in `analysis_mistakes` for the current player"). One criterion per behavior, not per file.

**Test plan** — key strictly to this project's conventions (CLAUDE.md → "Test-Writing Discipline"):

| Layer | Convention |
|-------|-----------|
| Go DB tests | `testutil.NewTestQueries(t)` (in-memory SQLite + migrations) |
| Go golden | `testutil.CompareGolden(t, name, got)` + fixtures in `testdata/` |
| Frontend component | `renderWithProviders()` from `src/test/render.tsx` |
| Frontend HTTP mocks | MSW handlers in `src/test/msw/handlers.ts` |
| PixiJS mocks | factories in `src/test/mocks/pixi.ts` |
| Wails binding mocks | `src/test/mocks/bindings.ts` |
| Zustand stores | Vitest, store reset in `beforeEach` |
| E2E | Playwright in `e2e/` (only when the slice changes a top-level user flow) |

If a slice doesn't need a layer (e.g. no PixiJS change), omit that row. Don't pad.

**TDD workflow** — RED / GREEN / REFACTOR with concrete file-level steps. The slice already commits to small scope; the workflow should fit in ~6 bullets. Mark `N/A` if the slice is pure infra/config (rare for vertical slices — push back if you find yourself writing this).

**Files to touch** — one row per file, marked `new` or `edit`. Cross-reference the line anchors verified in step 3. Include test files explicitly.

**Definition of Done** — copy the boilerplate below; add slice-specific items where the slice has unique gotchas (Wails binding regen, migration roll-forward, etc.).

### 5. Write the file

Filename: `.claude/tasks/{NNN}/{NN}-{kebab-slug-of-slice-title}.md`, where `NN` is the slice number zero-padded to 2 digits.

If the file already exists, **don't silently overwrite**. Show the user the current `**Status:**` line and ask: overwrite, append a new "Re-expanded {date}" section, or abort.

### 6. Report back

Three lines max:
- Path written.
- Slice title and effort estimate.
- One sentence on the first concrete step (usually the RED test in TDD).

Don't restate the AC in chat — the file is the source of truth.

## Output template

```markdown
# Slice {N} — {verb-led title}

**Slices file:** [`slices.md`](./slices.md)
**Status:** Not started
**Expanded:** {YYYY-MM-DD}
**Effort:** {½ day | 1 day | 2 days}
**Complexity:** {S | M | L | XL}
**Depends on:** Slice {N-1} ({Merged ✓ | In progress | Not started})

## Demo
{copied verbatim from the slice's `**Demo:**` line}

## North star context
{1 sentence pulled from `slices.md` `## North star`, framed for this slice}

## Acceptance criteria
- [ ] {testable user-visible behavior, one per row}
- [ ] {…}

## Files to touch

| Path | Status | Notes |
|------|--------|-------|
| `internal/demo/analysis/analyzer.go` | new | entry point; called from `app.go:parseDemo` |
| `app.go` | edit | add `GetMistakeTimeline` binding; insert at L{verified-line} |
| `frontend/src/components/viewer/mistake-list.tsx` | new | mounted from `demo-viewer.tsx` |
| `internal/demo/analysis/analyzer_test.go` | new | golden test for the rule |
| `frontend/src/components/viewer/mistake-list.test.tsx` | new | RTL via `renderWithProviders` |

## Test plan

| Layer | File | Approach |
|-------|------|----------|
| Go golden | `internal/demo/analysis/analyzer_test.go` | `testutil.CompareGolden(t, "{rule}", got)`; fixture in `testdata/{rule}/` |
| Frontend RTL | `frontend/src/components/viewer/mistake-list.test.tsx` | `renderWithProviders`; mock `GetMistakeTimeline` via `src/test/mocks/bindings.ts` |
| Wails wire-up | manual | run `wails dev`; confirm binding appears in `frontend/wailsjs/go/main/App.d.ts` after rebuild |

## TDD workflow

1. **RED** — {file}: write failing golden test asserting the analyzer emits {expected shape} for fixture {name}.
2. **RED** — {file}: write failing RTL test asserting the list renders one row per binding result.
3. **GREEN** — implement {analyzer.go}, {persist.go}, the binding, the hook, and the component (minimum to pass).
4. **REFACTOR** — extract helpers if the rule body exceeds ~30 lines; keep tests green.

## Definition of Done
- [ ] All acceptance criteria check out via the test plan above
- [ ] `make test-unit` passes (Go + Vitest)
- [ ] `make lint` clean
- [ ] `make typecheck` clean
- [ ] Manual smoke: `wails dev` → {one-line scenario reproducing the Demo}
- [ ] No regression in {related area, e.g. existing demo-viewer panel}
- [ ] Wails bindings regenerated (`wails dev` once → check `frontend/wailsjs/go/main/`) {include only if a binding was added}
- [ ] Migration applied cleanly on a fresh DB and on `make db-reset` {include only if a migration was added}

## Out of scope (peeled in slices.md)
- {copy `**Deferred:**` bullets verbatim, preserving Killick axis refs}

## Risks / unknowns
- {anything surfaced during the codebase scan that wasn't visible at slicing time, e.g. stale line anchors, conflicting symbol, schema constraint}
- {if the slice's effort estimate looks wrong post-scan, say so explicitly}
```

## Principles

- **Detail comes from verification, not from prose.** A ticket that lists files and line numbers from a real grep is worth more than five paragraphs of restated motivation. Do step 3 (codebase scan) every time — it's the difference between a ticket and a glorified slice copy.
- **JIT or not at all.** If the user asks to expand all 8 slices upfront, push back: tickets 4+ will go stale. Offer to expand only slice 1 (and slice 2 if it's trivially independent).
- **Don't redesign during expansion.** If the slice's plan looks wrong on contact with the codebase, surface the conflict under "Risks" and ask the user — don't quietly rewrite the In scope.
- **Match the project's vocabulary.** This repo uses Complexity (S/M/L/XL), Deps, Test Types, RED/GREEN/REFACTOR, `make` targets, `testutil` helpers, MSW, `renderWithProviders`. Reuse them; don't invent parallel terminology.
- **One ticket file per slice.** Sub-tasks belong inside the file (as TDD steps), not as separate ticket files.
- **Don't lecture in chat.** Three lines back to the user, then silence. The file is the artifact.

## Examples

### Example 1 — first expansion in a fresh task folder

User runs `/expand-slice` (no args) immediately after `/slice-plan` finished writing `.claude/tasks/002-some-epic/slices.md`.

- Resolve: most recent folder is `002-some-epic`; lowest un-expanded slice is `1`.
- No predecessor → no stale guard.
- Scan files in slice 1's In scope → all are new except one binding insertion point in `app.go`. Verify line number.
- Write `.claude/tasks/002-some-epic/01-{slug}.md`.
- Report: path; slice title; "Start with the RED golden test in `internal/demo/.../analyzer_test.go`."

### Example 2 — predecessor not merged

User runs `/expand-slice 3` while slice 2's ticket exists with `**Status:** In progress`.

- Stale guard fires: warn that AC for slice 3 may shift if slice 2 is still moving.
- Ask whether to proceed. On confirm, expand normally; under "Risks", note: *"Slice 2 (`{title}`) is in progress at expansion time; revisit AC if its scope drifts."*

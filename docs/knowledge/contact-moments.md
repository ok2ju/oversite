# Contact moments (v1)

> Status: shipped via the [Timeline Contact Moments plan](../../.claude/plans/timeline-contact-moments/README.md).
> Last updated: 2026-05-12.

## What it is

A **contact moment** is a tick-bounded window where a player is in
confirmed engagement (visibility, shot hit, damage, kill, or
effective flash) with one or more enemies. The system precomputes one
timeline marker per contact and a phase-bucketed list of mistakes
attached to it.

## Pipeline

```
demo file
  └─► parser (Phase 1: persists PlayerSpottersChanged → player_visibility)
        └─► contact builder (Phase 2: signal cluster → contact_moments row)
              └─► detectors (Phase 3: ten v1 detectors → contact_mistakes rows)
                    └─► Wails binding GetContactMoments (Phase 4)
                          └─► useContactMoments hook
                                └─► ContactsLane + ContactTooltip (Phase 4/5)
```

## Tables

- `player_visibility` — `(demo_id, tick, viewer, target, became_visible)`.
- `contact_moments` — per-contact window with outcome and `builder_version`.
- `contact_mistakes` — per-finding row with kind / phase / severity /
  extras / `detector_version`.

Schema files:
- `migrations/019_player_visibility.up.sql`
- `migrations/020_contact_moments.up.sql`
- `migrations/021_contact_mistakes.up.sql`

## Source layout

| Layer | Path |
|-------|------|
| Parser handler (LoS spotters) | `internal/demo/parser.go` |
| Contact builder | `internal/demo/contacts/` |
| Detectors + runner | `internal/demo/contacts/detectors/` |
| Wails bindings | `app.go` (`GetContactMoments`, `GetRoundImportantMoments`) |
| Frontend hook | `frontend/src/hooks/use-contact-moments.ts` |
| Lane builder projection | `frontend/src/lib/timeline/build-lanes.ts` |
| Active-marker derivation | `frontend/src/lib/timeline/contacts.ts` (`findActiveContact`) |
| Lane component | `frontend/src/components/viewer/round-timeline/contacts-lane.tsx` |
| Tooltip | `frontend/src/components/viewer/round-timeline/contact-tooltip.tsx` |

## v1 detector catalog

Severities (1 = low, 2 = medium, 3 = high) come from
`internal/demo/contacts/detectors/catalog.go`. The numbers below
reflect the catalog after the Phase 5 tune.

| Phase | Kind | Category | Default severity |
|-------|------|----------|------------------|
| pre | `slow_reaction` | aim | medium (2) |
| pre | `missed_first_shot` | spray | medium (2) |
| pre | `isolated_peek` | positioning | high (3) |
| pre | `bad_crosshair_height` | aim | medium (2) |
| pre | `peek_while_reloading` | aim | high (3) |
| during | `shot_while_moving` | movement | low (1) — _Phase 5 retune_ |
| during | `aim_while_flashed` | aim | medium (2) |
| during | `lost_hp_advantage` | trade | high (3) |
| post | `no_reposition_after_kill` | positioning | high (3) |
| post | `no_reload_with_cover` | utility | medium (2) |

See [Severity calibration](contact-mistake-severity-calibration.md)
for the corpus-driven tuning protocol and the calibration harness
(`internal/demo/contacts/detectors/calibration_test.go`,
`//go:build calibration`).

## Tooltip density

The Phase 4 tooltip ships top-3 + "+N more". Phase 5 added a
density-stress test (1v3 with 10+ mistakes spread across all three
phases) at
`frontend/src/components/viewer/round-timeline/contact-tooltip.test.tsx`.
Ordering is severity DESC; phase grouping renders Pre / Engagement /
Post sections; the "+N more" button toggles the expanded view.

## Active-marker highlight

While the playhead sits inside a contact's `[tPre, tPost]` window the
matching marker renders with `data-active="true"` plus an
amber ring + pulse (`ring-2 ring-amber-300 ring-offset-1 animate-pulse`).
Derived from `currentTick` via `findActiveContact` — no Zustand state.
Implementation:

- `frontend/src/lib/timeline/contacts.ts:findActiveContact`
- `frontend/src/components/viewer/round-timeline/round-timeline.tsx`
  (useMemo wiring)
- `frontend/src/components/viewer/round-timeline/contacts-lane.tsx`
  (renders the highlight class on the active marker)

## Re-import semantics

- `builder_version` and `detector_version` are independent. Phase 5
  did not bump either; severity edits in `catalog.go` apply to new
  imports immediately and to existing demos only on re-import.
- Force-update procedure:
  ```sql
  DELETE FROM contact_mistakes;
  ```
  …then re-import every demo. `MaxDetectorVersionForDemo` returns 0
  for rows with no `contact_mistakes` and the Phase 3 runner
  re-executes.

## Out of scope (v1)

- Cross-round mistakes (`walked_into_known_angle` etc.). Catalog rows
  exist with `Func: nil` for the v2 set.
- Per-season aggregation.
- A separate contact-deep-link panel on the analysis page.
- Screenshot-regression baseline for the lane (state-based e2e only —
  see `e2e/tests/timeline-contact-moments.spec.ts`).

## Related

- [Timeline Contact Moments plan](../../.claude/plans/timeline-contact-moments/README.md)
- [Severity calibration](contact-mistake-severity-calibration.md)
- [Demo parser](demo-parser.md)

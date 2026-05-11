# ADR-0019: Duel as a first-class analyzer entity

**Date:** 2026-05-11
**Status:** Accepted

## Context

The analyzer (`internal/demo/analysis/`) emits nine `Mistake` kinds. Four are fire-anchored (`shot_while_moving`, `no_counter_strafe`, `missed_first_shot`, `spray_decay`) and run on raw `weapon_fire` events with no concept of a target. The other five kill-anchored kinds (`slow_reaction`, `caught_reloading`, `isolated_peek`, `repeated_death_zone`, `flash_assist`) imply an attacker↔victim relationship, but until now that relationship lived implicitly on each row's `extras_json` blob — never as a first-class join key.

Two concrete problems fell out of the flat model:

1. **False positives on intent-less shots.** A player spraying through smoke at a wall accumulates `shot_while_moving` / `no_counter_strafe` flags even though there was no opponent to hit. The mistake says nothing real about the player's decision-making — they weren't in a duel.
2. **No grouping in the UI.** A duel that produces three mistakes (slow reaction → missed first shot → spray decay) currently surfaces as three independent rows in the mistake list. Coaches asking "what went wrong on that fight?" have to mentally re-stitch the engagement from tick numbers.

A separate table of engagements, with each mistake row carrying a back-reference to the engagement it occurred inside, addresses both: cone-less shots produce no engagement and no mistake, and the UI can render a duel band on the round timeline with its mistakes tooltipped beneath.

## Decision

Introduce `Duel` as a first-class analyzer entity:

- **Domain type** (`internal/demo/analysis/duels.go`): `Duel` carries `(AttackerSteam, VictimSteam, StartTick, EndTick, Outcome, EndReason, HitConfirmed, HurtCount, ShotCount, MutualLocalID)`. Outcome is one of `won | lost | inconclusive | won_then_traded | lost_but_traded`.
- **Detector** runs over the merged `weapon_fire + player_hurt + kill` event stream in tick order. Per-round state machine; targets resolved by `WeaponFireExtra.HitVictimSteamID` (authoritative) or, when missing, single-candidate cone enumeration (15° half-angle, 2200u max distance, 2s recent-activity window). Zero or ambiguous candidates → no duel, no fire-rule mistake. Stale duels (no event in 3s) expire `inconclusive`; A→V kill closes the duel `won`; a follow-up kill of the attacker inside `TradeWindowSeconds` flips the outcome to `won_then_traded`.
- **Schema** (migration 018): `analysis_duels` table with self-referential `mutual_duel_id` FK; `analysis_mistakes.duel_id INTEGER REFERENCES analysis_duels(id) ON DELETE SET NULL`. Cross-duel patterns (`eco_misbuy`, `he_damage`) keep `duel_id = NULL`.
- **Pipeline change**: `analysis.Run()` returns `([]Mistake, []Duel, error)`. After the existing rule list emits mistakes and `EnrichFireMistakes` annotates them, `AttributeMistakesToDuels` walks the mistakes once and resolves each one's `DuelID` based on a kind→role mapping (attacker for fire rules + `slow_reaction`, victim for `caught_reloading` / `isolated_peek` / `repeated_death_zone`, cross-player for `flash_assist`).
- **Persistence** writes duels and mistakes in one transaction. Duels are inserted first to capture rowids; mutual links are backfilled in a second-pass `UPDATE` because both peers need rowids before they can reference each other.
- **`AnalysisVersion`** bumped 1 → 2 so old demos surface the existing Recompute CTA. Old mistakes keep `duel_id = NULL` until the user re-runs analysis.

## Consequences

### Positive

- **Spam-into-smoke is silently dropped** instead of flagged. The cone fallback is intentionally strict (single-candidate only) — under-attribute over mis-attribute. The frontend's "Unattributed" bucket can surface the rare residual.
- **Engagement-level UI surface** unlocks the duels lane on `RoundTimeline` and the tooltip-grouped mistakes inside it. Coaches read "this fight, this outcome, these three mistakes" instead of nine disconnected ticks.
- **Compositional**: new analyzer rules can attach to duels by adding a kind to the attribution map without changing the detector. Existing rules are unchanged — attribution is layered on after `EnrichFireMistakes` the same way the cause-tag classifier was.
- **Persistence is atomic.** Duels and mistakes commit together; a failure mid-write rolls back both. Idempotent re-runs converge on the same rows.

### Negative

- **A new table and a new pass** on every demo import. Detector is O(events) — a single tick-ordered walk with an `active[attacker][victim]` map — but it does add overhead. On a 30-min match this is sub-100 ms vs the existing analyzer's seconds, so the cost is acceptable.
- **Cone false-positive rate is unknown.** We have no real-demo data on how often `pickTarget` returns ambiguous candidates (zero or multiple) for a shot that would have resolved to a clear duel in retrospect. The plan accepts this — instrumentation is deferred until we ingest enough real demos to characterise the rate.
- **No LOS gate through smokes.** `GrenadeDetonateExtra` doesn't carry position today, so a shot through an active smoke at a player visible in the open beyond it can still open a (real) duel. Acceptable for now; a smoke-cylinder gate can be added later if the false-positive rate justifies it.
- **Mutual link is two-pass writes**, not a single insert with foreign keys. The complexity sits in `persist.go`; the detector emits `MutualLocalID` against in-memory ids and the persistence layer translates to rowids. Worked example in [[knowledge/sqlc-workflow]].
- **`AnalysisVersion` bump invalidates every prior demo's analyzer rows** until the user clicks Recompute. The mistakes themselves still render (with `duel_id = NULL`), so the UI degrades gracefully, but the duels lane stays empty for old demos. This is the same staleness contract every prior `AnalysisVersion` bump has used.

## Alternatives considered

- **Materialised view on `analysis_mistakes`** that derives engagements at read time from mistake clustering. Rejected: clustering on tick proximity alone produces false groups when two duels resolve in the same 500 ms window, and the cone-fallback resolution that drives the false-positive reduction can't be deferred to read time without re-walking the event stream.
- **Embedding duel metadata in `extras_json`.** Rejected: the schema would be a Duel "by reference" only — no FK, no ability to query duels independently of their mistakes, and the UI's duels lane can't render a duel that produced zero mistakes (clean kills) without scanning every event row.
- **Symmetric (undirected) duel representation.** Rejected: every existing kill-anchored mistake belongs to exactly one side of the engagement (the dying player has different mistakes than the killing player), and a symmetric model would force every consumer to keep deciding which player's perspective applies. The mutual-link pattern captures the symmetric case (`A↔V crossfire`) as two directed duels with a backpointer.

## Implementation references

- Detector + attribution: `internal/demo/analysis/duels.go`
- Tests: `internal/demo/analysis/duels_test.go` (8 table-driven cases covering edge resolutions)
- Schema: `migrations/018_analysis_duels.{up,down}.sql`
- Queries: `queries/analysis_duels.sql` + `queries/analysis_mistakes.sql` (added `duel_id`)
- Persistence: `internal/demo/analysis/persist.go` (`PersistWithRoundMap` signature change)
- Wails bindings: `app.go` (`ListDuelsForPlayer`, `GetDuelContext`) + `types.go` (`DuelEntry`, `DuelContext`)
- Frontend: `frontend/src/components/viewer/round-timeline/duels-lane.tsx`, `frontend/src/hooks/use-duel-timeline.ts`, `frontend/src/components/viewer/patterns-section.tsx`

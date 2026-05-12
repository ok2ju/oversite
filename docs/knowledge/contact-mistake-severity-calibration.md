# Contact-mistake severity calibration (v1)

> Status: pending operator-run with a real 20–30 demo corpus.
> Tooling: `internal/demo/contacts/detectors/calibration_test.go`
> (`//go:build calibration`).
> Plan: [`.claude/plans/timeline-contact-moments/phase-5/01-calibration-tooling.md`](../../.claude/plans/timeline-contact-moments/phase-5/01-calibration-tooling.md)
> · [`02-severity-tuning.md`](../../.claude/plans/timeline-contact-moments/phase-5/02-severity-tuning.md)

<!-- DO NOT HAND-EDIT THE TABLES BELOW. -->
<!-- They are overwritten on every harness run. -->
<!-- Hand-write rationale inside the per-kind RATIONALE blocks. -->

> ⚠️ **Warning:** The corpus pass has **not yet been run**. This page is a
> placeholder; the harness writes over the tables below on every
> invocation. The Phase 5 commit baked in the one severity decision
> the plan explicitly carved out (`shot_while_moving` →
> low, see §3 below); the other nine v1 detectors keep their Phase 3
> defaults until a calibrated distribution is in hand.

## Corpus

_Populated by the harness._ Run:

```bash
go test -tags=calibration -timeout=30m -v \
    ./internal/demo/contacts/detectors/ \
    -run TestSeverityDistribution \
    -corpus=$(pwd)/testdata/corpus
```

…after dropping 20–30 representative `.dem` files under `testdata/corpus/`
(directory is `.gitignore`d — see Phase 5 plan §A3).

## Per-kind distribution

The harness will emit a table per v1 kind with low/medium/high counts
and a `BEGIN RATIONALE` block the operator fills.

For each block, the operator must record one of:

- "Keep severity = X" with a one-line rationale.
- "Move severity to X" with a one-line rationale (+ optional paired
  threshold edit).
- "Move threshold from A to B" with a rationale rooted in the per-kind
  table.

## Summary

_Populated by the harness._ Targets per the design intent:

| target distribution | range |
|---------------------|-------|
| low ≈ 70%           | 60–80% |
| medium ≈ 25%        | 18–35% |
| high ≈ 5%           |  3–10% |

## 3. Decisions baked into Phase 5

These changes landed in the Phase 5 commit before the corpus run.
They are tracked here for traceability.

| Kind | Severity before | Severity after | Rationale |
|------|-----------------|----------------|-----------|
| `shot_while_moving` | medium (2) | low (1) | A single contact can emit one finding per offending shot (5+ in a sprayed AK while running). At medium, these drown the tooltip's top-3 list. Lowered to low so the higher-impact findings displace them; granularity preserved via the `severity DESC, tick ASC` ordering. See plan `02-severity-tuning.md` §4.2. |

The remaining nine v1 kinds (`slow_reaction`, `missed_first_shot`,
`isolated_peek`, `bad_crosshair_height`, `peek_while_reloading`,
`aim_while_flashed`, `lost_hp_advantage`, `no_reposition_after_kill`,
`no_reload_with_cover`) keep their Phase 3 defaults pending the
corpus-driven tuning loop.

## Operator notes

- **Re-run the harness** after any threshold edit. The harness is
  opt-in (no CI cost) — see `01-calibration-tooling.md` §2.
- **Force-update existing demos** after a severity edit:
  ```sql
  DELETE FROM contact_mistakes;
  ```
  then re-import each demo. `MaxDetectorVersionForDemo` returns 0 for
  rows with no `contact_mistakes` and the Phase 3 runner re-executes.
- **Corpus is not committed.** Place demos under `testdata/corpus/`
  (gitignored). The harness records SHA-256 of each demo so the corpus
  can be reconstructed if needed.

## Open issues

_Filled in by hand once the corpus run completes._ Track:

- Kinds whose distribution falls outside the §3 target range and
  cannot be brought inside with a single threshold/severity edit.
- Tick-rate skews (MM 32 vs Faceit 64).
- `PreviousContactEnd` clamp leaks surfacing as spikes in the
  first-contact-of-the-round bucket.

These become Phase 5+ tickets; they do not block Phase 5's completion.

# ADR-0014: Remove Faceit Integration

**Date:** 2026-05-06
**Status:** Accepted

## Context

Oversite was originally designed as a Faceit-account-tied desktop client. Authentication, match history, ELO tracking, and auto-import of competitive demos all flowed from a user's Faceit account. Two earlier ADRs codified this:

- [ADR-0005](0005-faceit-oauth-pkce.md) — Faceit OAuth 2.0 + PKCE as the sole authentication method
- [ADR-0009](0009-loopback-oauth-desktop.md) — Loopback OAuth flow (RFC 8252) for desktop authentication

In practice, the Faceit-centric scope created a steady drag on the project:

- **Surface area disproportionate to value.** OAuth, secret-bearing CI workflow, keychain storage, sync worker, ELO history table, dashboard, login/callback routes, and a parallel set of `users`/`faceit_matches` schema all existed to support a feature set most users only sampled.
- **External dependency on a third party we don't control.** Faceit can change OAuth behavior, rate-limit the API, or restrict the demo download endpoint at any time. Several rounds of "fix Faceit auth" / "fix sync" had already been needed.
- **Conflict with the desktop app's value proposition.** The strongest, most differentiated capability is local 2D demo playback and per-demo analytics ([ADR-0006](0006-desktop-app-pivot.md), [ADR-0007](0007-wails-framework.md)). These work on **any** `.dem` file, regardless of source. Tying them to a Faceit login excluded matchmaking, FACEIT-Hub, third-party servers, and tournament VODs.
- **Data model complexity.** A single-tenant local app does not need a `users` table or a `user_id` foreign key on every demo. Filtering by user is meaningless when there's exactly one user per install.

### Alternatives considered

| Approach | Why rejected |
|----------|-------------|
| **Keep Faceit, fix the friction** | Doesn't address the core issue — every Faceit-origin entity is a maintenance and API-stability burden whose removal makes the rest of the app simpler, not just faster. |
| **Keep Faceit as optional** | Optionality is the most expensive choice: both code paths must work and be tested. Either Faceit is the primary login (ADR-0005) or it isn't — the in-between costs more than it returns. |
| **Replace with Steam OpenID / generic OAuth** | The app no longer needs an identity at all. Adding a different identity provider only re-introduces the same class of problems for a feature (multi-user accounts) we don't need. |

## Decision

Remove the Faceit integration entirely. Oversite becomes a **single-tenant local tool**: no authentication, no remote account, no API sync. Demos are imported by drag-and-drop or local file picker only.

Concretely:

- **Backend** — delete `internal/auth/`, `internal/faceit/`, `vars.go`, and all auth/sync/download Wails bindings on the `App` struct.
- **Database** — migration `005` drops the `users` and `faceit_matches` tables and removes the `demos.user_id` and `demos.faceit_match_id` columns; `queries/users.sql` and `queries/faceit_matches.sql` are deleted; remaining queries no longer filter by user.
- **Frontend** — remove the dashboard route, auth provider, login/callback/match-detail routes, all Faceit hooks, store, and types; index now redirects to `/demos`. Remove Faceit MSW handlers and Wails binding mocks.
- **CI/release** — remove Faceit OAuth secrets and `LDFLAGS` injection from `.github/workflows/release.yml`. The `FACEIT_*` repository secrets must be deleted manually.
- **Docs** — `knowledge/faceit-oauth.md` and `plans/p4-faceit-heatmaps.md` are deleted; the rest of the vault is de-Faceit-ified. ADRs 0005 and 0009 are kept as historical records and marked **Superseded by this ADR**.

This ADR supersedes [ADR-0005](0005-faceit-oauth-pkce.md) and [ADR-0009](0009-loopback-oauth-desktop.md).

## Consequences

### Positive

- **Smaller, simpler app.** No OAuth, no session/token storage, no keychain, no sync worker, no `users` table, no per-row tenancy filter. Every layer (schema, queries, bindings, routes, tests) shrinks.
- **No third-party API dependency.** The product can no longer be broken by a Faceit API or OAuth change.
- **Larger addressable demo set.** Any `.dem` file plays — matchmaking, third-party hubs, tournament demos, custom servers — not only Faceit-origin matches.
- **Privacy improvement.** Nothing leaves the user's machine. No tokens to revoke if a binary is lost or leaked.
- **Aligns with the desktop pivot ([ADR-0006](0006-desktop-app-pivot.md)).** The "single binary, runs locally" identity becomes literal rather than aspirational.

### Negative

- **No more auto-import** — users must drop demos in manually. For Faceit players this is a real regression.
- **No ELO history view** — ranked progress over time is no longer surfaced. (The underlying value was always upstream Faceit data; we never owned the source of truth.)
- **No cross-device or cloud state** — demo library lives in one SQLite file per machine.
- **Faceit-style demo conventions remain in the parser** (knife rounds, MR12 round counts) — these are properties of the demo files themselves, not of any API integration, so they stay. See `internal/demo/parser.go` comments.
- **Reintroducing remote features later costs more.** If we ever want sync or accounts again, schema, auth, and UI must be rebuilt rather than re-enabled.

# ADR-0005: Faceit OAuth 2.0 + PKCE as Sole Authentication Method

**Date:** 2026-03-31
**Status:** Superseded by [ADR-0014](0014-remove-faceit-integration.md) (2026-05-06)

## Context

Oversite is purpose-built for Faceit CS2 players. Every core feature — ELO tracking, match history, demo auto-import — requires a Faceit account. The question is whether to also support email/password or other identity providers.

### Alternatives considered

| Approach | Why rejected |
|----------|-------------|
| **Email/password + Faceit OAuth** | Two auth systems to maintain. Password hashing, reset flows, email verification — significant surface area for a solo developer. Users would still need to link their Faceit account separately, so email/password alone isn't sufficient. |
| **Steam OAuth** | Many CS2 players have Steam accounts, but Oversite's features depend on Faceit data (ELO, match history). Steam login would still require a separate Faceit linking step. |
| **Auth0 / Clerk / third-party auth** | Adds vendor dependency and cost. Faceit's OAuth is straightforward enough that a dedicated auth service adds complexity without proportional benefit. |
| **OAuth without PKCE** | Standard authorization code flow works, but PKCE protects against authorization code interception attacks. Since the frontend initiates the flow (SPA context), PKCE is a security best practice. |

## Decision

Use Faceit OAuth 2.0 with PKCE as the only authentication method. No email/password, no other identity providers.

Flow:
1. Browser initiates login → API generates PKCE code verifier/challenge, stores in Redis, redirects to Faceit
2. User authorizes on Faceit → callback with authorization code
3. API exchanges code + PKCE verifier for access/refresh tokens
4. API fetches Faceit profile (`/me`), upserts user in database
5. API creates Redis session, sets `HttpOnly` session cookie, redirects to `/dashboard`

Sessions are stored in Redis with a configurable TTL. Refresh tokens are stored encrypted in PostgreSQL for background Faceit API calls (data sync).

## Consequences

### Positive

- Zero password management — no hashing, reset flows, or breach liability
- Every user is guaranteed to have a Faceit account, simplifying the data model
- PKCE prevents authorization code interception without requiring a client secret in the browser
- Single sign-on experience — users click one button to authenticate

### Negative

- Users without Faceit accounts cannot use the platform (acceptable — they're not the target audience)
- Faceit OAuth downtime blocks all logins (mitigated by session TTL keeping existing users active)
- Faceit could change or deprecate their OAuth API (low risk — it's their primary third-party integration mechanism)

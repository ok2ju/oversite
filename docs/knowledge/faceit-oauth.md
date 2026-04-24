# Faceit OAuth

**Related:** [ADR-0005](../decisions/0005-faceit-oauth-pkce.md) · [ADR-0009](../decisions/0009-loopback-oauth-desktop.md) · [architecture/data-flows](../architecture/data-flows.md) §5.3

## Flow: loopback + PKCE

Desktop apps can't use a web redirect URL, so we follow RFC 8252:

1. Go backend starts a temporary HTTP listener on `127.0.0.1:<random-port>` with path `/callback`.
2. Generate PKCE `code_verifier` (random 64 bytes, base64url) and `code_challenge` (SHA-256 of verifier, base64url).
3. Open the system browser to Faceit's `/authorize` URL with `code_challenge`, `code_challenge_method=S256`, `redirect_uri=http://127.0.0.1:<port>/callback`, and a `state` nonce.
4. User authenticates in their browser.
5. Faceit redirects to our loopback listener with `code`.
6. Backend exchanges `code` + `code_verifier` at Faceit's `/token` endpoint for access/refresh tokens.
7. Stop the HTTP listener.

## Token storage

- **Refresh token** → OS keychain via `zalando/go-keyring`. Service name `oversite-faceit-auth`, account = Faceit user ID.
- **Access token** → in-memory only in the Go process. Refreshed lazily when a request 401s.
- **Never** write either to disk or to the SQLite database.

Per-platform backing stores:
- macOS: Keychain Services
- Windows: Credential Manager
- Linux: Secret Service (GNOME Keyring / KWallet)

Testing: `testutil.MockKeyring` in `internal/testutil/mocks.go` stubs the whole interface — use it, don't roll your own mock.

## State nonce

Always validate the `state` parameter on callback. If it doesn't match what was generated in step 2, reject the callback — it's a cross-site attack attempt.

## Session longevity

Access tokens from Faceit typically last ~24h; refresh tokens last ~30 days. We refresh automatically on 401. If the refresh token itself expires, the user has to log in again.

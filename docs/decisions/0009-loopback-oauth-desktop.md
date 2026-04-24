# ADR-0009: Loopback OAuth Flow for Desktop Authentication

**Date:** 2026-04-12
**Status:** Accepted

## Context

Faceit OAuth 2.0 + PKCE remains the authentication method ([ADR-0005](0005-faceit-oauth-pkce.md)), but the desktop app has no persistent HTTP server to receive OAuth callbacks. The web app used a server-side callback endpoint (`/api/v1/auth/faceit/callback`); the desktop app needs an alternative.

### Alternatives considered

| Approach | Why rejected |
|----------|-------------|
| **Custom protocol handler (`oversite://callback`)** | Requires OS-level protocol registration. Fragile across platforms -- macOS requires Info.plist entries, Windows needs registry keys, Linux varies by desktop environment. Some browsers show security warnings for custom protocols. |
| **Device Authorization Grant (RFC 8628)** | User-friendly (enter code on a web page), but Faceit does not support the device authorization grant type. Would require a proxy server. |
| **Embedded WebView login** | Renders Faceit login inside the app's WebView. Security concern: users can't verify they're entering credentials on the real Faceit site. Some OAuth providers block embedded WebView flows. Against OAuth best practices (RFC 8252 recommends system browser). |
| **Manual token paste** | User logs in on Faceit in browser, copies a token, pastes into the app. Terrible UX. |

## Decision

Use the **loopback redirect** pattern (RFC 8252 Section 7.3):

1. App starts a temporary HTTP listener on `http://127.0.0.1:{random_port}/callback`
2. App opens the system browser to Faceit's authorization URL with `redirect_uri=http://127.0.0.1:{port}/callback`
3. User authenticates in their default browser (familiar, trusted environment)
4. Faceit redirects to the loopback address; the temp listener captures the authorization code
5. App exchanges the code for tokens (with PKCE verifier) and shuts down the temp listener
6. Tokens stored securely:
   - **Refresh token**: OS keychain via `zalando/go-keyring` (Keychain on macOS, Credential Manager on Windows, Secret Service on Linux)
   - **Access token**: In-memory only (short-lived, refreshed as needed)

### Faceit redirect URI configuration

Faceit OAuth app must be configured to accept `http://localhost` as a valid redirect URI. Per RFC 8252, the authorization server should allow any port on the loopback address for native apps.

## Consequences

### Positive

- Standard OAuth flow following RFC 8252 best practices for native applications
- User authenticates in their trusted system browser -- no embedded credential entry
- No custom protocol registration -- works across all platforms without OS-level configuration
- OS keychain provides encrypted, OS-managed credential storage (better than plaintext config files)
- PKCE still applies -- same security guarantees as the web version

### Negative

- Random port requires Faceit to accept `http://localhost` with any port (or a wildcard port) as a valid redirect URI -- not all OAuth providers support this
- Brief context switch: user goes from app → browser → back to app. May be confusing on first use (mitigated with clear UI messaging: "Logging in via browser...")
- Local firewall software may block the temporary localhost listener on some corporate/managed machines
- If the user closes the browser tab before completing auth, the temp listener times out -- need graceful error handling and retry UX
- Keychain access may require user permission prompts on first use (especially macOS Keychain Access)

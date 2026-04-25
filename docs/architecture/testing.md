# Architecture — Testing Architecture

> **Siblings:** [overview](overview.md) · [structure](structure.md) · [components](components.md) · [data-flows](data-flows.md) · [wails-bindings](wails-bindings.md) · [database](database.md) · [crosscutting](crosscutting.md)

---

## Testing Architecture

### Test Strategy by Layer

| Layer | Tool | Strategy |
|-------|------|----------|
| Go services | `go test` | Classical TDD; interfaces enable mocking |
| Go bindings | `go test` + httptest-style | Test service methods directly |
| Go demo parser | Golden-file TDD + spike | Spike validates library, then golden tests |
| sqlc queries | `go test` + temp SQLite | `:memory:` or temp file SQLite for each test |
| Zustand stores | Vitest | Classical TDD; pure state + actions |
| React components | Vitest + RTL | Render + interaction tests |
| Wails binding hooks | Vitest + mock bindings | Mock auto-generated binding functions |
| PixiJS rendering | Test-alongside | TDD the logic; screenshot-test the visuals |
| E2E flows | Playwright | Test-alongside; written after features work |

### Test Infrastructure

| Component | Purpose |
|-----------|---------|
| Temp SQLite databases | In-memory (`:memory:`) or temp file per test; run migrations; clean between tests |
| MSW (Mock Service Worker) | Mock Faceit API responses for frontend tests |
| Vitest + React Testing Library | Frontend component and hook testing |
| Playwright | E2E tests against a running Wails dev instance |
| Golden files | Known-good parser output for regression testing |

### Key Differences from Web Test Architecture

| Aspect | Web Version | Desktop Version |
|--------|-------------|-----------------|
| Database tests | testcontainers (PostgreSQL) | Temp SQLite (`:memory:` or temp file) |
| API tests | httptest against chi router | Direct service method calls |
| WebSocket tests | WebSocket test client | Not needed (no WebSocket) |
| Yjs tests | In-memory Yjs docs | Not needed (no Yjs) |
| Auth tests | Mock Redis session store | Mock keyring interface |

---

*See [knowledge/testing.md](../knowledge/testing.md) for practical test-writing patterns and reusable utilities.*

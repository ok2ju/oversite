# Product — Non-Functional Requirements

> **Siblings:** [vision](vision.md) · [personas](personas.md) · [features](features.md) · [user-stories](user-stories.md) · [data-models](data-models.md) · [wails-bindings](wails-bindings.md)

---

## Non-Functional Requirements

### Performance

| Metric | Target |
|--------|--------|
| Demo parse time (avg 100 MB file) | < 10 seconds (local disk, no upload) |
| 2D Viewer frame rate | Stable 60 FPS at 1080p |
| Heatmap render (single demo) | < 2 seconds |
| Heatmap render (10-demo aggregate) | < 5 seconds |
| App startup time (cold) | < 3 seconds to interactive UI |
| Tick data query latency (SQLite) | < 50ms for a 1000-tick range |

### Security

- `.dem` file validation (magic bytes, size limits) before parsing
- No network listeners; the app neither calls out to nor accepts inbound connections at runtime
- SQLite database file permissions: owner read/write only

### Accessibility

- WCAG 2.1 AA compliance for all non-canvas UI
- Keyboard navigation for all controls (playback, menus, forms)
- Screen reader support for UI chrome (aria labels, roles)
- Color-blind-friendly palette option for team colors
- Canvas elements: provide text alternatives where feasible (scoreboard, stats)

### Platform Support

| Platform | Minimum Version | WebView Engine |
|----------|----------------|---------------|
| macOS | 12 (Monterey)+ | WebKit (WKWebView) |
| Windows | 10 (1903)+ | WebView2 (Chromium-based) |
| Linux | Ubuntu 22.04+ | WebKitGTK |

- WebGL 2.0 required for PixiJS rendering (all supported WebView versions support this)
- No internet access required at runtime

### Installation & Distribution

| Dimension | Target |
|-----------|--------|
| Install size | < 30 MB |
| Installer format (macOS) | `.dmg` or `.app` bundle |
| Installer format (Windows) | `.exe` (NSIS) or `.msi` |
| Installer format (Linux) | `.AppImage` or `.deb` |
| Auto-update | Built-in update checker + download |

### Test Coverage & Quality

The project follows **Test-Driven Development (TDD)**. Every feature is developed using the Red-Green-Refactor cycle: write a failing test first, implement the minimum code to pass, then refactor.

| Metric | Target |
|--------|--------|
| Go backend line coverage | >= 80% |
| Go critical-path coverage (parser, SQLite store) | >= 90% |
| Frontend component/hook test coverage | >= 75% |
| Frontend utility/store coverage | >= 90% |
| E2E critical path coverage | 100% of US-01 (install), US-04 (import), US-09 (viewer), US-22 (strat board) — see [user-stories](user-stories.md) |
| CI gate | Zero merge to main without all tests passing |

**Test execution time budgets:**

| Test Tier | Budget |
|-----------|--------|
| Unit tests (Go + TS) | < 30 seconds total |
| Integration tests (temp SQLite, MSW) | < 2 minutes total |
| End-to-end tests (Playwright) | < 10 minutes total |

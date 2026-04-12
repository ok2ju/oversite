# ADR-0007: Use Wails as the Desktop Application Framework

**Date:** 2026-04-12
**Status:** Accepted

## Context

The desktop pivot ([ADR-0006](0006-desktop-app-pivot.md)) requires a framework that embeds a Go backend and a web-based frontend into a single native binary. The existing codebase has a Go backend (chi router, demoinfocs-golang parser, sqlc) and a React frontend (PixiJS, shadcn/ui, Tailwind, Zustand). The chosen framework should maximize reuse of this existing code.

### Alternatives considered

| Approach | Why rejected |
|----------|-------------|
| **Electron** | Bundles Chromium -- 150 MB+ app size. Requires a Node.js bridge layer to call Go code (child process or IPC). No direct Go-to-JS bindings. Massive ecosystem, but the overhead doesn't justify it for this project. |
| **Tauri** | Rust-based; uses system WebView (good). But the backend is Rust -- would require rewriting all Go code in Rust, or running Go as a sidecar process with IPC. Sidecar approach adds complexity and negates the "single binary" benefit. |
| **Fyne** | Go-native UI toolkit, but uses its own rendering engine (not web technologies). Would require rewriting the entire React frontend. No PixiJS, no shadcn/ui, no Tailwind. |
| **Lorca / go-webview** | Minimal wrappers around system WebView. No build tooling, no asset embedding, no cross-compilation support. Would need to build all the glue that Wails provides. |

## Decision

Use **Wails v2** as the desktop framework. Wails provides:

- **Go-to-JS bindings**: Go struct methods are automatically exposed as TypeScript functions callable from the frontend. No REST API layer needed for internal communication.
- **System WebView**: Uses WebKit (macOS), WebView2 (Windows), and WebKitGTK (Linux). No bundled browser -- keeps binary size ~15 MB.
- **Vite integration**: Frontend is built as a Vite SPA (replacing Next.js App Router). Assets are embedded into the Go binary via `embed.FS`.
- **Single binary output**: `wails build` produces one executable per platform.
- **Cross-compilation**: Build for all three platforms from a single machine (with appropriate toolchains).

The frontend migrates from Next.js (App Router, server components) to a Vite + React SPA with `react-router-dom` for client-side routing. Server-side rendering is unnecessary in a desktop app.

## Consequences

### Positive

- Direct Go-to-JS bindings eliminate the HTTP/REST layer for internal communication -- lower latency, simpler code
- ~15 MB binary size vs. 150 MB+ for Electron
- Reuses existing Go backend code with minimal changes (remove HTTP handlers, expose service methods via Wails bindings)
- Reuses existing React frontend code (components, stores, PixiJS layers) with routing migration
- Wails CLI provides dev mode with hot-reload for both Go and frontend
- Single binary distribution simplifies installation

### Negative

- System WebView inconsistencies -- must test PixiJS WebGL rendering on WebKit (macOS), WebView2 (Windows), and WebKitGTK (Linux)
- Smaller ecosystem than Electron -- fewer community plugins and examples
- Wails v2 is less mature than Electron -- may encounter edge cases in WebView interop
- Requires Vite migration from Next.js -- App Router features (server components, file-based routing, API routes) must be replaced
- System WebView version depends on OS version -- older OS versions may have outdated WebView capabilities

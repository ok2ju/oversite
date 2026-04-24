# Architecture — Overview & System Context

> **Version:** 2.0 · **Format:** arc42 · **Siblings:** [structure](structure.md) · [components](components.md) · [data-flows](data-flows.md) · [wails-bindings](wails-bindings.md) · [database](database.md) · [crosscutting](crosscutting.md) · [testing](testing.md)

---

## 1. Introduction & Goals

### 1.1 Requirements Overview

Oversite is a desktop 2D demo viewer and analytics platform for CS2 Faceit players. It runs as a single native binary using Wails (Go backend + system WebView frontend).

| Priority | Quality Goal | Motivation |
|----------|-------------|------------|
| 1 | **Performance** | 60 FPS canvas rendering; < 10s demo parse from local disk; < 50ms tick query |
| 2 | **Simplicity** | Single binary, single process, no external services except Faceit API |
| 3 | **Developer Experience** | Monorepo with hot reload, type-safe SQL, Wails dev mode |
| 4 | **Cross-Platform** | macOS, Windows, Linux from a single codebase |

### 1.2 Stakeholders

| Role | Concern |
|------|---------|
| Solo developer | Productive monorepo DX; manageable complexity |
| End users (Faceit players) | Fast, reliable demo review on their desktop |
| Future contributors | Clear architecture boundaries; documented bindings |

---

## 2. System Context (C4 Level 1)

```
┌─────────────────────────────────────────────────────────┐
│                    External Systems                      │
│                                                         │
│  ┌──────────────┐                   ┌──────────────┐   │
│  │  Faceit API   │                   │ Local         │   │
│  │  (OAuth +     │                   │ Filesystem    │   │
│  │   Data API)   │                   │ (.dem files)  │   │
│  └──────┬───────┘                   └──────┬───────┘   │
│         │                                  │           │
└─────────┼──────────────────────────────────┼───────────┘
          │                                  │
          ▼                                  ▼
┌─────────────────────────────────────────────────────────┐
│                                                         │
│              O V E R S I T E  (Desktop)                  │
│                                                         │
│    Native desktop app for CS2 demo review, analytics,   │
│    strategy planning, and Faceit stats tracking.        │
│                                                         │
│    Single binary: Go backend + WebView frontend         │
│                                                         │
└─────────────────────────────────────────────────────────┘
```

### External System Interfaces

| System | Protocol | Purpose |
|--------|----------|---------|
| **Faceit OAuth** | HTTPS (OAuth 2.0 + PKCE) | User authentication via loopback redirect |
| **Faceit Data API** | HTTPS REST | Player stats, match history, ELO data |
| **Local Filesystem** | OS file I/O | Read `.dem` files, SQLite database, app data |

---

*Cross-references: [product specs](../product/vision.md) · [roadmap](../roadmap.md) · [tasks](../tasks.md)*

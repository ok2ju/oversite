# Architecture — Overview & System Context

> **Version:** 2.1 · **Format:** arc42 · **Siblings:** [structure](structure.md) · [components](components.md) · [data-flows](data-flows.md) · [wails-bindings](wails-bindings.md) · [database](database.md) · [crosscutting](crosscutting.md) · [testing](testing.md)

---

## Introduction & Goals

### Requirements Overview

Oversite is a desktop 2D demo viewer and analytics platform for CS2 players. It runs as a single native binary using Wails (Go backend + system WebView frontend) with no network dependencies — demos are imported from disk, parsed in-process, and stored in a local SQLite database.

| Priority | Quality Goal | Motivation |
|----------|-------------|------------|
| 1 | **Performance** | 60 FPS canvas rendering; < 10s demo parse from local disk; < 50ms tick query |
| 2 | **Simplicity** | Single binary, single process, no external services |
| 3 | **Developer Experience** | Monorepo with hot reload, type-safe SQL, Wails dev mode |
| 4 | **Cross-Platform** | macOS, Windows, Linux from a single codebase |

### Stakeholders

| Role | Concern |
|------|---------|
| Solo developer | Productive monorepo DX; manageable complexity |
| End users (CS2 players) | Fast, reliable demo review on their desktop |
| Future contributors | Clear architecture boundaries; documented bindings |

---

## System Context (C4 Level 1)

```
┌─────────────────────────────────────────────────────────┐
│                                                         │
│              O V E R S I T E  (Desktop)                  │
│                                                         │
│    Native desktop app for CS2 demo review, analytics,   │
│    and strategy planning.                               │
│                                                         │
│    Single binary: Go backend + WebView frontend         │
│                                                         │
└──────────────────────────┬──────────────────────────────┘
                           │
                           ▼
                    ┌──────────────┐
                    │ Local        │
                    │ Filesystem   │
                    │ (.dem files) │
                    └──────────────┘
```

### External System Interfaces

| System | Protocol | Purpose |
|--------|----------|---------|
| **Local Filesystem** | OS file I/O | Read `.dem` files, SQLite database, app data |

There are no remote services. The app neither calls out to nor accepts inbound connections at runtime.

---

*Cross-references: [product specs](../product/vision.md) · [roadmap](../roadmap.md) · [tasks](../tasks.md)*

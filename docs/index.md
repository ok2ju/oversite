# Oversite Docs

CS2 2D demo viewer and analytics platform for Faceit players. Single-binary Wails desktop app. This vault is the single source of truth for product, architecture, and implementation knowledge.

## Product

What we're building. Canonical specs — read top-to-bottom to understand the product.

- [[product/vision|Vision & Goals]] — problem statement, goals, tech stack
- [[product/personas|Personas]] — Kai (solo grinder), Sofia (IGL), Marcus (analyst)
- [[product/features|Features]] — information architecture + feature specs (F1–F7)
- [[product/user-stories|User Stories]] — US-01 through US-33 + empty/error states
- [[product/non-functional|Non-Functional Requirements]] — performance, security, a11y, quality
- [[product/data-models|Data Models]] — business-level entity tables
- [[product/wails-bindings|Wails Bindings]] — binding method catalog

## Architecture

How it's built. Arc42-format system design — read to understand the system.

- [[architecture/overview|Overview]] — intro, goals, system context (C4 L1)
- [[architecture/structure|Structure]] — app structure (C4 L2) + directory layout
- [[architecture/components|Components]] — Go + React components (C4 L3)
- [[architecture/data-flows|Data Flows]] — sequence diagrams for 6 core flows
- [[architecture/wails-bindings|Wails Bindings]] — binding architecture, events, errors
- [[architecture/database|Database]] — DDL + local storage layout
- [[architecture/crosscutting|Crosscutting]] — errors, logging, config, data integrity, calibration, auto-update
- [[architecture/testing|Testing]] — test strategy by layer

## Decisions

ADRs — immutable records of accepted/deprecated/superseded decisions.

- [[decisions/README|ADR Index]] (13 records)

## Delivery

Where we are and where we're going.

- [[roadmap|Roadmap]] — 6-phase delivery plan
- [[tasks|Task Breakdown]] — granular tasks across P1–P6
- Phase plans: [[plans/p1-desktop-foundation|P1]] · [[plans/p2-auth-demo-pipeline|P2]] · [[plans/p3-core-2d-viewer|P3]] · [[plans/p4-faceit-heatmaps|P4]]

## Knowledge Wiki

LLM-maintained notes on implementation entities, patterns, and gotchas. Curated via `/ingest-session`.

- [[knowledge/README|Wiki Index]]

## Log

- [[log|Project Log]] — append-only chronological record

---

*Archive: [[archive/web-app/README|legacy web app docs]]*

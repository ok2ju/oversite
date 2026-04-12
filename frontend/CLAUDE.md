# Frontend — TypeScript / React

## Coding Conventions

- **State**: Zustand stores per domain (`viewerStore`, `stratStore`, `uiStore`, `faceitStore`). Use selector hooks to minimize re-renders.
- **Data fetching**: TanStack Query for all API calls. No raw `fetch` in components.
- **Components**: shadcn/ui for standard UI. Custom components in `components/viewer/`, `components/strat/`.
- **Styling**: Tailwind CSS utility classes. No CSS modules or styled-components.

## Testing

TDD (Red-Green-Refactor). All conventions below apply.

- **Components/hooks**: Vitest + React Testing Library. API mocking via MSW (Mock Service Worker).
- **Zustand stores**: Pure unit tests (no DOM needed).
- **TanStack Query hooks**: `renderHook()` + MSW.
- **PixiJS logic**: Unit-test transforms, interpolation, state. Visual output screenshot-tested with Playwright.


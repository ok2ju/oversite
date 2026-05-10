# Frontend — TypeScript / React

## Coding Conventions

- **State**: Zustand stores per domain (`useDemoStore`, `useViewerStore`, `useStratStore`, `useUiStore`, `useHeatmapStore`, `useTickBufferStore`) in `src/stores/`. Use selector hooks to minimize re-renders.
- **Data fetching**: TanStack Query for all API calls. No raw `fetch` in components.
- **Components**: shadcn/ui for standard UI. Custom components in `components/viewer/`, `components/strat/`.
- **Styling**: Tailwind CSS utility classes. No CSS modules or styled-components.
- **Wails bindings**: Import generated functions from `frontend/wailsjs/go/main/App`. Mock them in tests via `src/test/mocks/bindings.ts`.

## Testing

TDD (Red-Green-Refactor). All conventions below apply.

- **Wrappers**: Render with `renderWithProviders()` from `src/test/render.tsx` (provides QueryClient + Theme). Never instantiate a raw QueryClientProvider.
- **Components/hooks**: Vitest + React Testing Library. API mocking via MSW (Mock Service Worker).
- **Zustand stores**: Pure unit tests (no DOM needed).
- **TanStack Query hooks**: `renderHook()` + MSW.
- **PixiJS logic**: Unit-test transforms, interpolation, state. Visual output screenshot-tested with Playwright.
- **PixiJS mocks**: Use factories from `src/test/mocks/pixi.ts`.


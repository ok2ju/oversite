# Product — Wails Bindings Overview

> **Siblings:** [vision](vision.md) · [personas](personas.md) · [features](features.md) · [user-stories](user-stories.md) · [non-functional](non-functional.md) · [data-models](data-models.md)
>
> **See also:** [architecture/wails-bindings.md](../architecture/wails-bindings.md) for binding architecture, event system, and error handling conventions.

---

## Wails Bindings Overview

Instead of a REST API, the Go backend exposes methods to the frontend via Wails bindings. The frontend calls these as async TypeScript functions.

### Binding Groups

#### Demos

| Method | Signature | Description |
|--------|----------|-------------|
| `ImportDemoFile` | `() -> void` | Open native file picker; copy into the app demos folder, validate, persist, and trigger parse |
| `ImportDemoByPath` | `(path: string) -> void` | Import a `.dem` at the given path (drag-and-drop); copies into the app demos folder |
| `ListDemos` | `(page: number, perPage: number) -> DemoListResult` | Paginated list of imported demos |
| `GetDemoByID` | `(id: string) -> Demo` | Get demo metadata |
| `DeleteDemo` | `(id: number) -> void` | Delete demo data from SQLite |

#### Viewer

| Method | Signature | Description |
|--------|----------|-------------|
| `GetDemoRounds` | `(demoId: string) -> Round[]` | All rounds for a demo |
| `GetDemoEvents` | `(demoId: string) -> GameEvent[]` | All game events for a demo |
| `GetDemoTicks` | `(demoId: string, startTick: number, endTick: number) -> TickData[]` | Tick data for a range |
| `GetRoundRoster` | `(demoId: string, roundNumber: number) -> PlayerRosterEntry[]` | Roster for a specific round |
| `GetScoreboard` | `(demoId: string) -> ScoreboardEntry[]` | Aggregated player stats for a demo |

#### Heatmaps

| Method | Signature | Description |
|--------|----------|-------------|
| `GetHeatmapData` | `(demoIds: number[], weapons: string[], playerSteamID: string, side: string) -> HeatmapPoint[]` | Aggregated kill positions |
| `GetUniqueWeapons` | `(demoIds: number[]) -> string[]` | Distinct kill-event weapons across demos |
| `GetUniquePlayers` | `(demoIds: number[]) -> PlayerInfo[]` | Distinct kill-event players across demos |
| `GetWeaponStats` | `(demoId: string) -> WeaponStat[]` | Per-weapon kill / headshot counts for a demo |

### Frontend Call Pattern

```typescript
import { ListDemos, ImportDemoByPath } from '../../wailsjs/go/main/App';

// Wails bindings are called as regular async functions
await ImportDemoByPath('/path/to/demo.dem');
const demos = await ListDemos(1, 50);
```

TanStack Query wraps these bindings for caching and background refetch:

```typescript
const { data: demos } = useQuery({
  queryKey: ['demos', page, perPage],
  queryFn: () => ListDemos(page, perPage),
});
```

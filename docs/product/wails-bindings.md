# Product — Wails Bindings Overview

> **Siblings:** [vision](vision.md) · [personas](personas.md) · [features](features.md) · [user-stories](user-stories.md) · [non-functional](non-functional.md) · [data-models](data-models.md)
>
> **See also:** [architecture/wails-bindings.md](../architecture/wails-bindings.md) for binding architecture, event system, and error handling conventions.

---

## 10. Wails Bindings Overview

Instead of a REST API, the Go backend exposes methods to the frontend via Wails bindings. The frontend calls these as async TypeScript functions.

### 10.1 Binding Groups

#### Auth

| Method | Signature | Description |
|--------|----------|-------------|
| `StartLogin` | `() -> LoginResult` | Start loopback OAuth; opens system browser |
| `GetCurrentUser` | `() -> User \| null` | Get logged-in user profile |
| `Logout` | `() -> void` | Clear tokens from keychain |
| `RefreshProfile` | `() -> User` | Re-fetch Faceit profile data |

#### Demos

| Method | Signature | Description |
|--------|----------|-------------|
| `ImportDemo` | `(path: string) -> Demo` | Import and parse a single `.dem` file |
| `ImportFolder` | `(path: string) -> Demo[]` | Recursively import `.dem` files from folder |
| `ListDemos` | `(opts: ListOpts) -> Demo[]` | List user's demos (sortable, filterable) |
| `GetDemo` | `(id: number) -> Demo` | Get demo metadata |
| `DeleteDemo` | `(id: number, deleteFile: bool) -> void` | Delete demo data; optionally remove `.dem` |
| `GetRounds` | `(demoId: number) -> Round[]` | Get round summaries for a demo |
| `GetRoundDetail` | `(roundId: number) -> RoundDetail` | Get round detail + player stats |
| `GetTicks` | `(demoId: number, from: number, to: number) -> TickData[]` | Get tick data for a range |
| `GetEvents` | `(demoId: number, filters: EventFilter) -> GameEvent[]` | Get filtered game events |

#### Heatmaps

| Method | Signature | Description |
|--------|----------|-------------|
| `GetHeatmapData` | `(demoIds: number[], filters: HeatmapFilter) -> HeatmapPoint[]` | Aggregated heatmap data |

#### Strategy Boards

| Method | Signature | Description |
|--------|----------|-------------|
| `ListBoards` | `() -> StrategyBoard[]` | List all strategy boards |
| `CreateBoard` | `(title: string, mapName: string) -> StrategyBoard` | Create a new board |
| `GetBoard` | `(id: number) -> StrategyBoard` | Get board with state |
| `SaveBoard` | `(id: number, state: string) -> void` | Save board state (JSON) |
| `DeleteBoard` | `(id: number) -> void` | Delete a board |
| `ExportBoardJSON` | `(id: number) -> string` | Export board as JSON string |
| `ImportBoardJSON` | `(json: string) -> StrategyBoard` | Import board from JSON |

#### Grenade Lineups

| Method | Signature | Description |
|--------|----------|-------------|
| `ListLineups` | `(filters: LineupFilter) -> GrenadeLineup[]` | List/search lineups |
| `GetLineup` | `(id: number) -> GrenadeLineup` | Get lineup detail |
| `UpdateLineup` | `(id: number, data: LineupUpdate) -> GrenadeLineup` | Update title, description, tags |
| `DeleteLineup` | `(id: number) -> void` | Delete a lineup |
| `ToggleFavorite` | `(id: number) -> void` | Toggle favorite status |

#### Faceit

| Method | Signature | Description |
|--------|----------|-------------|
| `GetFaceitProfile` | `() -> FaceitProfile` | Get user's Faceit profile (cached) |
| `GetMatches` | `(opts: MatchListOpts) -> FaceitMatch[]` | Get match history (paginated) |
| `SyncMatches` | `() -> SyncResult` | Trigger manual match sync |
| `ImportMatchDemo` | `(matchId: string) -> Demo` | Download and import demo from Faceit match |

#### System

| Method | Signature | Description |
|--------|----------|-------------|
| `OpenFileDialog` | `() -> string` | Native file picker for `.dem` files |
| `OpenFolderDialog` | `() -> string` | Native folder picker |
| `GetAppInfo` | `() -> AppInfo` | App version, data dir, DB size |
| `CheckForUpdates` | `() -> UpdateInfo \| null` | Check if a newer version is available |

### 10.2 Frontend Call Pattern

```typescript
import { ImportDemo, ListDemos } from '../../wailsjs/go/main/App';

// Wails bindings are called as regular async functions
const demo = await ImportDemo('/path/to/demo.dem');
const demos = await ListDemos({ sortBy: 'date', order: 'desc' });
```

TanStack Query wraps these bindings for caching and background refetch:

```typescript
const { data: demos } = useQuery({
  queryKey: ['demos', filters],
  queryFn: () => ListDemos(filters),
});
```

# Product — Data Models

> **Siblings:** [vision](vision.md) · [personas](personas.md) · [features](features.md) · [user-stories](user-stories.md) · [non-functional](non-functional.md) · [wails-bindings](wails-bindings.md)
>
> **Canonical DDL:** Authoritative `CREATE TABLE` statements with constraints and indexes live in [architecture/database.md](../architecture/database.md). The tables below describe entities at a business level — field purposes and types. If they diverge, the DDL in architecture/database.md is the source of truth.

---

## Data Models

### Core Entities

#### Demo

| Field | Type | Notes |
|-------|------|-------|
| id | INTEGER | Primary key (autoincrement) |
| map_name | TEXT | e.g., "de_dust2" |
| file_path | TEXT | Absolute path to local `.dem` file |
| file_size | INTEGER | Bytes |
| status | TEXT | imported / parsing / ready / failed |
| total_ticks | INTEGER | Set after parsing |
| tick_rate | REAL | Ticks per second |
| duration_secs | INTEGER | Match duration |
| match_date | TEXT | ISO 8601 datetime |
| created_at | TEXT | ISO 8601 datetime |

#### Round

| Field | Type | Notes |
|-------|------|-------|
| id | INTEGER | Primary key (autoincrement) |
| demo_id | INTEGER | FK -> Demo |
| round_number | INTEGER | 1-based |
| start_tick | INTEGER | |
| end_tick | INTEGER | |
| winner_side | TEXT | CT / T |
| win_reason | TEXT | elimination / bomb_exploded / defused / time |
| ct_score | INTEGER | Score after this round |
| t_score | INTEGER | Score after this round |

#### PlayerRound

| Field | Type | Notes |
|-------|------|-------|
| id | INTEGER | Primary key (autoincrement) |
| round_id | INTEGER | FK -> Round |
| steam_id | TEXT | Steam64 ID |
| player_name | TEXT | |
| team_side | TEXT | CT / T |
| kills | INTEGER | |
| deaths | INTEGER | |
| assists | INTEGER | |
| damage | INTEGER | |
| headshot_kills | INTEGER | |
| first_kill | INTEGER | 0/1 boolean |
| first_death | INTEGER | 0/1 boolean |
| clutch_kills | INTEGER | |

#### TickData

| Field | Type | Notes |
|-------|------|-------|
| demo_id | INTEGER | FK -> Demo; part of composite PK |
| tick | INTEGER | Part of composite PK |
| steam_id | TEXT | Part of composite PK |
| x | REAL | World-space X |
| y | REAL | World-space Y |
| z | REAL | World-space Z |
| yaw | REAL | View angle (horizontal) |
| health | INTEGER | |
| armor | INTEGER | |
| is_alive | INTEGER | 0/1 boolean |
| weapon | TEXT | Active weapon |

*Index: `(demo_id, tick)` composite index for range scan queries.*

#### GameEvent

| Field | Type | Notes |
|-------|------|-------|
| id | INTEGER | Primary key (autoincrement) |
| demo_id | INTEGER | FK -> Demo |
| round_id | INTEGER | FK -> Round |
| tick | INTEGER | |
| event_type | TEXT | kill / grenade_throw / grenade_detonate / bomb_plant / bomb_defuse |
| attacker_steam_id | TEXT | Nullable |
| victim_steam_id | TEXT | Nullable |
| weapon | TEXT | Nullable |
| x | REAL | Event position |
| y | REAL | |
| z | REAL | |
| extra_data | TEXT | JSON string for event-specific data (headshot, penetration, flash assist) |

#### StrategyBoard

| Field | Type | Notes |
|-------|------|-------|
| id | INTEGER | Primary key (autoincrement) |
| title | TEXT | |
| map_name | TEXT | |
| board_state | TEXT | JSON serialized board state |
| created_at | TEXT | ISO 8601 datetime |
| updated_at | TEXT | ISO 8601 datetime |

#### GrenadeLineup

| Field | Type | Notes |
|-------|------|-------|
| id | INTEGER | Primary key (autoincrement) |
| demo_id | INTEGER | FK -> Demo (source, nullable) |
| tick | INTEGER | Source tick in demo |
| map_name | TEXT | |
| grenade_type | TEXT | smoke / flash / he / molotov |
| throw_x | REAL | Thrower position |
| throw_y | REAL | |
| throw_z | REAL | |
| throw_yaw | REAL | Aim angle |
| throw_pitch | REAL | Aim angle |
| land_x | REAL | Landing/detonation position |
| land_y | REAL | |
| land_z | REAL | |
| title | TEXT | User-provided or auto-generated |
| description | TEXT | |
| tags | TEXT | JSON array of tags |
| is_favorite | INTEGER | 0/1 boolean; default 0 |
| created_at | TEXT | ISO 8601 datetime |

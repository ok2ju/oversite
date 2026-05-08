package main

import "encoding/json"

// Domain types exposed to the frontend via Wails bindings.
// JSON tags match the TypeScript interfaces in frontend/src/types/.

// Demo represents a parsed demo file (full detail variant, used by
// GetDemoByID for the viewer page).
type Demo struct {
	ID           int64  `json:"id"`
	MapName      string `json:"map_name"`
	FilePath     string `json:"file_path"`
	FileSize     int64  `json:"file_size"`
	Status       string `json:"status"`
	TotalTicks   int    `json:"total_ticks"`
	TickRate     int    `json:"tick_rate"`
	DurationSecs int    `json:"duration_secs"`
	MatchDate    string `json:"match_date"`
	CreatedAt    string `json:"created_at"`
}

// DemoSummary is the list-row variant returned by ListDemos. FilePath is
// replaced with FileName (basename only) — the library table never uses the
// full path anyway and a 100-demo page saves ~10–20 KB on the wire vs Demo.
type DemoSummary struct {
	ID           int64  `json:"id"`
	MapName      string `json:"map_name"`
	FileName     string `json:"file_name"`
	FileSize     int64  `json:"file_size"`
	Status       string `json:"status"`
	TotalTicks   int    `json:"total_ticks"`
	TickRate     int    `json:"tick_rate"`
	DurationSecs int    `json:"duration_secs"`
	MatchDate    string `json:"match_date"`
	CreatedAt    string `json:"created_at"`
}

// DemoListResult is the paginated response for demo listing.
type DemoListResult struct {
	Data []DemoSummary  `json:"data"`
	Meta PaginationMeta `json:"meta"`
}

// PaginationMeta holds pagination metadata.
type PaginationMeta struct {
	Total   int `json:"total"`
	Page    int `json:"page"`
	PerPage int `json:"per_page"`
}

// Round represents a round within a demo.
type Round struct {
	ID            string `json:"id"`
	RoundNumber   int    `json:"round_number"`
	StartTick     int    `json:"start_tick"`
	FreezeEndTick int    `json:"freeze_end_tick"`
	EndTick       int    `json:"end_tick"`
	WinnerSide    string `json:"winner_side"`
	WinReason     string `json:"win_reason"`
	CTScore       int    `json:"ct_score"`
	TScore        int    `json:"t_score"`
	IsOvertime    bool   `json:"is_overtime"`
	CTTeamName    string `json:"ct_team_name"`
	TTeamName     string `json:"t_team_name"`
}

// GameEvent represents a game event in a demo. ExtraData is passed through as
// raw JSON so we can avoid the per-row Unmarshal-then-Marshal round trip in
// storeGameEventToBinding (8K events × 1 map alloc + N key allocs each is the
// dominant cost in the events read path). The wire format on the frontend is
// unchanged for cold fields: a `Record<string, unknown>` is decoded directly
// from the bytes.
//
// Hot fields (Headshot, AssisterSteamID, HealthDamage, AttackerName,
// VictimName, AttackerTeam, VictimTeam) are dedicated columns on game_events
// and travel as top-level fields on this struct (see migration 010). The
// frontend reads them as top-level keys instead of poking through extra_data.
type GameEvent struct {
	ID              string          `json:"id"`
	DemoID          string          `json:"demo_id"`
	RoundID         *string         `json:"round_id"`
	Tick            int             `json:"tick"`
	EventType       string          `json:"event_type"`
	AttackerSteamID *string         `json:"attacker_steam_id"`
	VictimSteamID   *string         `json:"victim_steam_id"`
	Weapon          *string         `json:"weapon"`
	X               *float64        `json:"x"`
	Y               *float64        `json:"y"`
	Z               *float64        `json:"z"`
	Headshot        bool            `json:"headshot"`
	AssisterSteamID *string         `json:"assister_steam_id"`
	HealthDamage    int             `json:"health_damage"`
	AttackerName    string          `json:"attacker_name"`
	VictimName      string          `json:"victim_name"`
	AttackerTeam    string          `json:"attacker_team"`
	VictimTeam      string          `json:"victim_team"`
	ExtraData       json.RawMessage `json:"extra_data"`
}

// TickData represents a player's state at a specific tick.
//
// Inventory used to live here (as a comma-separated string) but was moved to
// per-round storage in migration 011 — see RoundLoadoutEntry and
// GetRoundLoadouts. The viewer team-bars merge live tick fields with the
// round-scoped loadout in the frontend hook.
//
// Coordinate precision: X/Y/Z and Yaw are sent as int16 instead of float64.
// A CS2 unit is ~2.5 cm and map extents fit comfortably in ±32k units; the
// frontend's per-tick interpolation produces sub-unit fractional pixels
// either way (`cur.x + (nxt.x - cur.x) * alpha` runs in JS double precision).
// Yaw is rounded to whole degrees — 1° resolution is below human angular
// perception at typical viewport zoom. Saves ~150 KB per tick chunk on the
// JSON wire (10 chars/float vs 4 chars/int × 64 K numbers).
type TickData struct {
	Tick        int     `json:"tick"`
	SteamID     string  `json:"steam_id"`
	X           int16   `json:"x"`
	Y           int16   `json:"y"`
	Z           int16   `json:"z"`
	Yaw         int16   `json:"yaw"`
	Health      int     `json:"health"`
	Armor       int     `json:"armor"`
	IsAlive     bool    `json:"is_alive"`
	Weapon      *string `json:"weapon"`
	Money       int     `json:"money"`
	HasHelmet   bool    `json:"has_helmet"`
	HasDefuser  bool    `json:"has_defuser"`
	AmmoClip    int     `json:"ammo_clip"`
	AmmoReserve int     `json:"ammo_reserve"`
}

// RoundLoadoutEntry is one player's freeze-end loadout for a specific round.
// Inventory is a comma-separated weapon list (encodeInventory output) the
// frontend splits on receipt. Returned from GetRoundLoadouts as a map keyed
// by round_number → []RoundLoadoutEntry.
type RoundLoadoutEntry struct {
	SteamID   string `json:"steam_id"`
	Inventory string `json:"inventory"`
}

// PlayerRosterEntry represents a player in a round's roster.
type PlayerRosterEntry struct {
	SteamID    string `json:"steam_id"`
	PlayerName string `json:"player_name"`
	TeamSide   string `json:"team_side"`
}

// ScoreboardEntry represents aggregated player stats for a demo.
type ScoreboardEntry struct {
	SteamID      string  `json:"steam_id"`
	PlayerName   string  `json:"player_name"`
	TeamSide     string  `json:"team_side"`
	Kills        int     `json:"kills"`
	Deaths       int     `json:"deaths"`
	Assists      int     `json:"assists"`
	Damage       int     `json:"damage"`
	HSKills      int     `json:"hs_kills"`
	RoundsPlayed int     `json:"rounds_played"`
	HSPercent    float64 `json:"hs_percent"`
	ADR          float64 `json:"adr"`
}

// HeatmapPoint represents a single aggregated kill position.
type HeatmapPoint struct {
	X         float64 `json:"x"`
	Y         float64 `json:"y"`
	KillCount int     `json:"kill_count"`
}

// PlayerInfo identifies a player by Steam ID and name.
type PlayerInfo struct {
	SteamID    string `json:"steam_id"`
	PlayerName string `json:"player_name"`
}

// WeaponStat represents aggregated weapon kill stats for a demo.
type WeaponStat struct {
	Weapon    string `json:"weapon"`
	KillCount int    `json:"kill_count"`
	HSCount   int    `json:"hs_count"`
}

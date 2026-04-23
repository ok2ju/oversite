package main

// Domain types exposed to the frontend via Wails bindings.
// JSON tags match the TypeScript interfaces in frontend/src/types/.

// User represents an authenticated user.
type User struct {
	UserID   string `json:"user_id"`
	FaceitID string `json:"faceit_id"`
	Nickname string `json:"nickname"`
}

// Demo represents a parsed demo file.
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

// DemoListResult is the paginated response for demo listing.
type DemoListResult struct {
	Data []Demo         `json:"data"`
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
}

// GameEvent represents a game event in a demo.
type GameEvent struct {
	ID              string         `json:"id"`
	DemoID          string         `json:"demo_id"`
	RoundID         *string        `json:"round_id"`
	Tick            int            `json:"tick"`
	EventType       string         `json:"event_type"`
	AttackerSteamID *string        `json:"attacker_steam_id"`
	VictimSteamID   *string        `json:"victim_steam_id"`
	Weapon          *string        `json:"weapon"`
	X               *float64       `json:"x"`
	Y               *float64       `json:"y"`
	Z               *float64       `json:"z"`
	ExtraData       map[string]any `json:"extra_data"`
}

// TickData represents a player's state at a specific tick.
type TickData struct {
	Tick    int     `json:"tick"`
	SteamID string  `json:"steam_id"`
	X       float64 `json:"x"`
	Y       float64 `json:"y"`
	Z       float64 `json:"z"`
	Yaw     float64 `json:"yaw"`
	Health  int     `json:"health"`
	Armor   int     `json:"armor"`
	IsAlive bool    `json:"is_alive"`
	Weapon  *string `json:"weapon"`
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

// FaceitProfile represents a user's Faceit profile.
type FaceitProfile struct {
	Nickname      string        `json:"nickname"`
	AvatarURL     *string       `json:"avatar_url"`
	Elo           *int          `json:"elo"`
	Level         *int          `json:"level"`
	Country       *string       `json:"country"`
	MatchesPlayed int           `json:"matches_played"`
	CurrentStreak CurrentStreak `json:"current_streak"`
}

// CurrentStreak represents a win/loss streak.
type CurrentStreak struct {
	Type  string `json:"type"`
	Count int    `json:"count"`
}

// FaceitMatch represents a Faceit match.
type FaceitMatch struct {
	ID            string   `json:"id"`
	FaceitMatchID string   `json:"faceit_match_id"`
	MapName       string   `json:"map_name"`
	ScoreTeam     int      `json:"score_team"`
	ScoreOpponent int      `json:"score_opponent"`
	Result        string   `json:"result"`
	Kills         *int     `json:"kills"`
	Deaths        *int     `json:"deaths"`
	Assists       *int     `json:"assists"`
	ADR           *float64 `json:"adr"`
	DemoURL       *string  `json:"demo_url"`
	DemoID        *string  `json:"demo_id"`
	HasDemo       bool     `json:"has_demo"`
	PlayedAt      string   `json:"played_at"`
}

// FolderImportResult is the response from importing a folder of demos.
type FolderImportResult struct {
	Imported []Demo   `json:"imported"`
	Errors   []string `json:"errors"`
}

// FaceitMatchListResult is the paginated response for Faceit match listing.
type FaceitMatchListResult struct {
	Data []FaceitMatch  `json:"data"`
	Meta PaginationMeta `json:"meta"`
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

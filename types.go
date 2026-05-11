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
	Tick    int    `json:"tick"`
	SteamID string `json:"steam_id"`
	X       int16  `json:"x"`
	Y       int16  `json:"y"`
	Z       int16  `json:"z"`
	Yaw     int16  `json:"yaw"`
	// Pitch is the player's vertical view angle in whole degrees, downward
	// positive (CS2 convention). int16 keeps the wire small (±90° fits trivially)
	// at the same 1° quantization as Yaw — viewer interpolation handles the rest.
	Pitch int16 `json:"pitch"`
	// Crouch is Player.IsDucking() at the sample. Used by the analysis page's
	// crouch-before-shot evidence; viewer renders unchanged on legacy demos
	// (false defaults).
	Crouch      bool    `json:"crouch"`
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

// PlayerMatchStats is the deep-stats payload returned by GetPlayerMatchStats
// for the right-side player panel in the viewer.
type PlayerMatchStats struct {
	SteamID          string              `json:"steam_id"`
	PlayerName       string              `json:"player_name"`
	TeamSide         string              `json:"team_side"`
	RoundsPlayed     int                 `json:"rounds_played"`
	Kills            int                 `json:"kills"`
	Deaths           int                 `json:"deaths"`
	Assists          int                 `json:"assists"`
	Damage           int                 `json:"damage"`
	HSKills          int                 `json:"hs_kills"`
	ClutchKills      int                 `json:"clutch_kills"`
	FirstKills       int                 `json:"first_kills"`
	FirstDeaths      int                 `json:"first_deaths"`
	OpeningWins      int                 `json:"opening_wins"`
	OpeningLosses    int                 `json:"opening_losses"`
	TradeKills       int                 `json:"trade_kills"`
	HSPercent        float64             `json:"hs_percent"`
	ADR              float64             `json:"adr"`
	DamageByWeapon   []DamageByWeapon    `json:"damage_by_weapon"`
	DamageByOpponent []DamageByOpponent  `json:"damage_by_opponent"`
	Rounds           []PlayerRoundDetail `json:"rounds"`
	Movement         MovementStats       `json:"movement"`
	Timings          TimingStats         `json:"timings"`
	Utility          UtilityStats        `json:"utility"`
	HitGroups        []HitGroupBreakdown `json:"hit_groups"`
}

// UtilityStats is the match-level utility profile for a player (Phase 3).
type UtilityStats struct {
	FlashesThrown          int     `json:"flashes_thrown"`
	SmokesThrown           int     `json:"smokes_thrown"`
	HEsThrown              int     `json:"hes_thrown"`
	MolotovsThrown         int     `json:"molotovs_thrown"`
	DecoysThrown           int     `json:"decoys_thrown"`
	FlashAssists           int     `json:"flash_assists"`
	BlindTimeInflictedSecs float64 `json:"blind_time_inflicted_secs"`
	EnemiesFlashed         int     `json:"enemies_flashed"`
}

// MistakeEntry is a single per-player analysis finding (e.g. an untraded
// death). Returned chronologically from GetMistakeTimeline; the viewer side
// panel renders one row per entry. Category / Severity / Title / Suggestion /
// WhyItHurts are resolved server-side from analysis.TemplateForKind so the
// frontend can render rich rows without duplicating the kind→presentation
// mapping.
type MistakeEntry struct {
	ID          int64          `json:"id"`
	Kind        string         `json:"kind"`
	Category    string         `json:"category"`
	Severity    int            `json:"severity"`
	Title       string         `json:"title"`
	Suggestion  string         `json:"suggestion"`
	WhyItHurts  string         `json:"why_it_hurts"`
	RoundNumber int            `json:"round_number"`
	Tick        int64          `json:"tick"`
	SteamID     string         `json:"steam_id"`
	Extras      map[string]any `json:"extras"`
	// DuelID is the analysis_duels row this mistake attaches to. nil for
	// mistakes that don't belong to a duel (eco_misbuy, he_damage) or
	// that fell into the "unattributed" bucket.
	DuelID *int64 `json:"duel_id"`
}

// MistakeCoOccurrence is the lightweight reference shape for a *different*
// mistake the same player committed inside the same fire window. The
// mistake-detail panel renders one chip per co-occurrence so the user can
// pivot to the related play without leaving the detail card.
type MistakeCoOccurrence struct {
	ID    int64  `json:"id"`
	Kind  string `json:"kind"`
	Title string `json:"title"`
	Tick  int64  `json:"tick"`
}

// DuelEntry is a single attacker→victim engagement, returned by
// ListDuelsForPlayer. The viewer renders one band per entry on the
// round-timeline duels lane. HitConfirmed / HurtCount / ShotCount drive
// hit dots in the band; MutualDuelID lets the viewer stack two bands
// when both players fired at each other in overlapping windows.
type DuelEntry struct {
	ID            int64  `json:"id"`
	RoundNumber   int    `json:"round_number"`
	AttackerSteam string `json:"attacker_steam"`
	VictimSteam   string `json:"victim_steam"`
	StartTick     int64  `json:"start_tick"`
	EndTick       int64  `json:"end_tick"`
	Outcome       string `json:"outcome"`
	EndReason     string `json:"end_reason"`
	HitConfirmed  bool   `json:"hit_confirmed"`
	HurtCount     int    `json:"hurt_count"`
	ShotCount     int    `json:"shot_count"`
	MutualDuelID  *int64 `json:"mutual_duel_id"`
}

// DuelContext is the deep-detail variant returned by GetDuelContext —
// one duel plus the mistakes inside it. Powers the tooltip / popover on
// the duels lane.
type DuelContext struct {
	Duel     DuelEntry      `json:"duel"`
	Mistakes []MistakeEntry `json:"mistakes"`
}

// MistakeContext is the deep-detail variant returned by GetMistakeContext.
// Carries everything MistakeEntry does plus the surrounding round window
// metadata and the list of co-occurring mistakes (P2-3) so the analysis-
// detail card can render the play with no extra round-trip.
type MistakeContext struct {
	Entry         MistakeEntry          `json:"entry"`
	RoundStartTck int64                 `json:"round_start_tick"`
	RoundEndTick  int64                 `json:"round_end_tick"`
	FreezeEndTick int64                 `json:"freeze_end_tick"`
	CoOccurring   []MistakeCoOccurrence `json:"co_occurring"`
}

// PlayerAnalysis is the per-(demo, player) summary row read by the viewer's
// overall-score gauge and category cards. OverallScore is a 0–100 composite
// computed by analysis.RunMatchSummary; downstream readers must treat it as
// an opaque composite — the score recipe rebalances across slices.
//
// The wider per-category aggregate columns (added in slice 10) ride alongside
// the legacy TradePct / AvgTradeTicks; the frontend reads them by name from
// the same struct without needing a second binding.
type PlayerAnalysis struct {
	SteamID       string  `json:"steam_id"`
	OverallScore  int     `json:"overall_score"`
	Version       int     `json:"version"`
	TradePct      float64 `json:"trade_pct"`
	AvgTradeTicks float64 `json:"avg_trade_ticks"`

	// Aim
	CrosshairHeightAvgOff float64 `json:"crosshair_height_avg_off"`
	TimeToFireMsAvg       float64 `json:"time_to_fire_ms_avg"`
	FlickCount            int     `json:"flick_count"`
	FlickHitPct           float64 `json:"flick_hit_pct"`

	// Spray
	FirstShotAccPct float64 `json:"first_shot_acc_pct"`
	SprayDecaySlope float64 `json:"spray_decay_slope"`

	// Movement
	StandingShotPct  float64 `json:"standing_shot_pct"`
	CounterStrafePct float64 `json:"counter_strafe_pct"`

	// Utility
	SmokesThrown     int `json:"smokes_thrown"`
	SmokesKillAssist int `json:"smokes_kill_assist"`
	FlashAssists     int `json:"flash_assists"`
	HeDamage         int `json:"he_damage"`
	NadesUnused      int `json:"nades_unused"`

	// Positioning
	IsolatedPeekDeaths int `json:"isolated_peek_deaths"`
	RepeatedDeathZones int `json:"repeated_death_zones"`

	// Economy
	FullBuyADR float64 `json:"full_buy_adr"`
	EcoKills   int     `json:"eco_kills"`

	Extras map[string]any `json:"extras"`
}

// PlayerRoundEntry is one row from the player_round_analysis table — the
// per-(demo, player, round) breakdown that backs the standalone analysis
// page's per-round drilldown. Slice 10 promotes economy + nade-usage +
// shot-accuracy fields to columns; future slices add category columns under
// the same shape. Extras is nullable to mirror MistakeEntry / PlayerAnalysis.
type PlayerRoundEntry struct {
	SteamID     string  `json:"steam_id"`
	RoundNumber int     `json:"round_number"`
	TradePct    float64 `json:"trade_pct"`
	BuyType     string  `json:"buy_type"`
	MoneySpent  int     `json:"money_spent"`
	NadesUsed   int     `json:"nades_used"`
	NadesUnused int     `json:"nades_unused"`
	ShotsFired  int     `json:"shots_fired"`
	ShotsHit    int     `json:"shots_hit"`

	Extras map[string]any `json:"extras"`
}

// MatchInsights is the team-level summary surfaced by GetMatchInsights. It
// rolls up player_match_analysis rows into per-side aggregates plus a small
// list of standout players for the team-comparison surface on the analysis
// page.
type MatchInsights struct {
	DemoID    string            `json:"demo_id"`
	CTSummary TeamSummary       `json:"ct_summary"`
	TSummary  TeamSummary       `json:"t_summary"`
	Standouts []PlayerHighlight `json:"standouts"`
}

// TeamSummary collapses one side's player_match_analysis rows into an
// average-per-metric view. Counts are sums; percentages are simple means.
// The frontend reads this for the head-to-head bar chart.
type TeamSummary struct {
	Side             string  `json:"side"` // "CT" or "T"
	Players          int     `json:"players"`
	AvgOverallScore  float64 `json:"avg_overall_score"`
	AvgTradePct      float64 `json:"avg_trade_pct"`
	AvgStandingShot  float64 `json:"avg_standing_shot_pct"`
	AvgCounterStrafe float64 `json:"avg_counter_strafe_pct"`
	AvgFirstShot     float64 `json:"avg_first_shot_acc_pct"`
	TotalFlashAssist int     `json:"total_flash_assists"`
	TotalSmokesKA    int     `json:"total_smokes_kill_assist"`
	TotalHeDamage    int     `json:"total_he_damage"`
	TotalIsolated    int     `json:"total_isolated_peek_deaths"`
	TotalEcoKills    int     `json:"total_eco_kills"`
	AvgFullBuyADR    float64 `json:"avg_full_buy_adr"`
}

// PlayerHighlight is one entry in the MatchInsights.Standouts list — the top
// performer for a category. Five entries (one per major category) keep the
// surface narrow.
type PlayerHighlight struct {
	SteamID    string  `json:"steam_id"`
	Category   string  `json:"category"`
	MetricName string  `json:"metric_name"`
	MetricVal  float64 `json:"metric_value"`
}

// AnalysisStatus reports whether mechanical-analysis rows exist for a demo.
// Status is one of:
//   - "imported"  — demo row exists, parser hasn't run yet
//   - "parsing"   — parse-and-analyze pipeline is in flight
//   - "failed"    — parse failed; demos list shows the failure
//   - "missing"   — demo is "ready" but no player_match_analysis rows exist
//     (typically a demo imported before slice 1 landed). The viewer panel
//     auto-triggers RecomputeAnalysis on this state.
//   - "ready"     — analyzer rows are present; panel renders the populated
//     header + list.
//
// "missing" is intentionally separate from the demo lifecycle status so a
// future "recomputing" state can slot in without overloading Demo.status.
type AnalysisStatus struct {
	DemoID string `json:"demo_id"`
	Status string `json:"status"`
}

// HabitRow mirrors analysis.HabitRow but with the enums flattened to plain
// strings so the JSON wire encoding stays trivial. The frontend reads `status`
// and `direction` as discriminated-union strings; thresholds are surfaced so
// the row can render its norm line ("≤ 100 ms") without a second binding.
//
// PreviousValue / Delta are populated by GetHabitReport once history is in
// scope (P0-3): nil means "no previous demo to compare against" and the UI
// hides the delta line.
type HabitRow struct {
	Key           string   `json:"key"`
	Label         string   `json:"label"`
	Description   string   `json:"description"`
	Unit          string   `json:"unit"`
	Direction     string   `json:"direction"`
	Value         float64  `json:"value"`
	Status        string   `json:"status"`
	GoodThreshold float64  `json:"good_threshold"`
	WarnThreshold float64  `json:"warn_threshold"`
	GoodMin       float64  `json:"good_min"`
	GoodMax       float64  `json:"good_max"`
	WarnMin       float64  `json:"warn_min"`
	WarnMax       float64  `json:"warn_max"`
	PreviousValue *float64 `json:"previous_value"`
	Delta         *float64 `json:"delta"`
}

// HabitReport is the response shape of GetHabitReport — a list of habit rows
// for one (demo, player), already classified server-side. AsOf is the demo's
// match_date as an RFC3339 string (empty when the demo's match_date is unset)
// so the page header can render "as of YYYY-MM-DD" without a second fetch.
type HabitReport struct {
	DemoID  string     `json:"demo_id"`
	SteamID string     `json:"steam_id"`
	AsOf    string     `json:"as_of"`
	Habits  []HabitRow `json:"habits"`
}

// HistoryPoint is one (demo, value) pair in a player's habit timeline,
// returned by GetHabitHistory. Sorted newest-first by the binding so the
// frontend can render sparklines or trend tables without re-sorting.
type HistoryPoint struct {
	DemoID    string  `json:"demo_id"`
	MatchDate string  `json:"match_date"`
	Value     float64 `json:"value"`
}

// NextDrill mirrors analysis.Drill on the wire. Key is empty when the
// maintenance fallback is returned (no bad/warn habit found) — the
// frontend branches on Key === "" to swap copy for the all-good case.
type NextDrill struct {
	Key      string   `json:"key"`
	Title    string   `json:"title"`
	Why      string   `json:"why"`
	Duration string   `json:"duration"`
	Chips    []string `json:"chips"`
}

// CoachingHabitRow mirrors analysis.CoachingHabitRow — a HabitRow with a
// per-card Trend sequence powering the sparkline. The aggregated Value is
// already classified server-side via the norm catalog.
type CoachingHabitRow struct {
	HabitRow
	Trend []HistoryPoint `json:"trend"`
}

// MistakeKindCount is one row in the coaching errors strip — a fire-related
// mistake kind plus its count across the lookback window. Empty list when no
// fire-related mistakes were flagged for the player.
type MistakeKindCount struct {
	Kind  string `json:"kind"`
	Total int    `json:"total"`
}

// CoachingReport is the response shape of GetCoachingReport. Habits powers the
// 6-card grid; Errors powers the taxonomy strip; LatestDemoID + LastDemoAt let
// the CTA link to the most recent demo's analysis page. Empty steamID and
// "no demos" both yield the empty report (Habits=[], Errors=[],
// LatestDemoID="") so the frontend can render an empty state without
// branching on errors.
type CoachingReport struct {
	SteamID      string             `json:"steam_id"`
	Lookback     int                `json:"lookback"`
	Habits       []CoachingHabitRow `json:"habits"`
	Errors       []MistakeKindCount `json:"errors"`
	LatestDemoID string             `json:"latest_demo_id"`
	LastDemoAt   string             `json:"last_demo_at"`
}

// HitGroupBreakdown is one row in the damage-by-hit-group breakdown.
type HitGroupBreakdown struct {
	HitGroup int    `json:"hit_group"`
	Label    string `json:"label"`
	Damage   int    `json:"damage"`
	Hits     int    `json:"hits"`
}

// PlayerRoundDetail is one round's breakdown for a single player.
type PlayerRoundDetail struct {
	RoundNumber           int      `json:"round_number"`
	TeamSide              string   `json:"team_side"`
	Kills                 int      `json:"kills"`
	Deaths                int      `json:"deaths"`
	Assists               int      `json:"assists"`
	Damage                int      `json:"damage"`
	HSKills               int      `json:"hs_kills"`
	ClutchKills           int      `json:"clutch_kills"`
	FirstKill             bool     `json:"first_kill"`
	FirstDeath            bool     `json:"first_death"`
	TradeKill             bool     `json:"trade_kill"`
	LoadoutValue          int      `json:"loadout_value"`
	DistanceUnits         int      `json:"distance_units"`
	AliveDurationSecs     float64  `json:"alive_duration_secs"`
	TimeToFirstContactSec *float64 `json:"time_to_first_contact_sec"`
}

// MovementStats is the match-level movement profile for a player. Strafe
// percent is approximate (16 Hz sample rate); the panel surfaces this with a
// tooltip.
type MovementStats struct {
	DistanceUnits   int     `json:"distance_units"`
	AvgSpeedUps     float64 `json:"avg_speed_ups"`
	MaxSpeedUps     float64 `json:"max_speed_ups"`
	StrafePercent   float64 `json:"strafe_percent"`
	StationaryRatio float64 `json:"stationary_ratio"`
	WalkingRatio    float64 `json:"walking_ratio"`
	RunningRatio    float64 `json:"running_ratio"`
}

// TimingStats is the match-level timing profile for a player.
type TimingStats struct {
	AvgTimeToFirstContactSecs float64 `json:"avg_time_to_first_contact_secs"`
	AvgAliveDurationSecs      float64 `json:"avg_alive_duration_secs"`
	TimeOnSiteASecs           float64 `json:"time_on_site_a_secs"`
	TimeOnSiteBSecs           float64 `json:"time_on_site_b_secs"`
}

// DamageByWeapon is one row in the damage-by-weapon breakdown.
type DamageByWeapon struct {
	Weapon string `json:"weapon"`
	Damage int    `json:"damage"`
}

// DamageByOpponent is one row in the damage-by-opponent breakdown.
type DamageByOpponent struct {
	SteamID    string `json:"steam_id"`
	PlayerName string `json:"player_name"`
	TeamSide   string `json:"team_side"`
	Damage     int    `json:"damage"`
}

package faceit

// Player represents a Faceit player profile from GET /players/{player_id}.
type Player struct {
	PlayerID  string          `json:"player_id"`
	Nickname  string          `json:"nickname"`
	Avatar    string          `json:"avatar"`
	Country   string          `json:"country"`
	FaceitURL string          `json:"faceit_url"`
	Games     map[string]Game `json:"games"`
}

// Game holds game-specific stats within a player profile.
type Game struct {
	GameID     string `json:"game_id"`
	Region     string `json:"region"`
	SkillLevel int    `json:"skill_level"`
	FaceitElo  int    `json:"faceit_elo"`
}

// MatchHistory is the paginated response from GET /players/{id}/history.
type MatchHistory struct {
	Items []MatchSummary `json:"items"`
	Start int            `json:"start"`
	End   int            `json:"end"`
}

// MatchSummary is a single match entry in match history.
type MatchSummary struct {
	MatchID    string          `json:"match_id"`
	GameID     string          `json:"game_id"`
	GameMode   string          `json:"game_mode"`
	StartedAt  int64           `json:"started_at"`
	FinishedAt int64           `json:"finished_at"`
	Teams      map[string]Team `json:"teams"`
	Results    MatchResults    `json:"results"`
	FaceitURL  string          `json:"faceit_url"`
}

// Team represents a team in a match.
type Team struct {
	TeamID   string       `json:"team_id"`
	Nickname string       `json:"nickname"`
	Players  []TeamPlayer `json:"players"`
}

// TeamPlayer represents a player within a team.
type TeamPlayer struct {
	PlayerID   string `json:"player_id"`
	Nickname   string `json:"nickname"`
	Avatar     string `json:"avatar"`
	FaceitElo  int    `json:"faceit_elo"`
	SkillLevel int    `json:"skill_level"`
}

// MatchResults holds the outcome of a match.
type MatchResults struct {
	Winner string         `json:"winner"`
	Score  map[string]int `json:"score"`
}

// MatchDetails is the full match detail from GET /matches/{match_id}.
type MatchDetails struct {
	MatchID    string          `json:"match_id"`
	GameID     string          `json:"game_id"`
	Region     string          `json:"region"`
	Status     string          `json:"status"`
	Teams      map[string]Team `json:"teams"`
	Results    MatchResults    `json:"results"`
	DemoURL    []string        `json:"demo_url"`
	StartedAt  int64           `json:"started_at"`
	FinishedAt int64           `json:"finished_at"`
	FaceitURL  string          `json:"faceit_url"`
}

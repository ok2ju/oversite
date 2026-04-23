package faceit

import "context"

// FaceitPlayer represents a Faceit player profile.
type FaceitPlayer struct {
	PlayerID   string
	Nickname   string
	Avatar     string
	Country    string
	SkillLevel int
	FaceitElo  int
}

// FaceitMatchHistory is a paginated match history response.
type FaceitMatchHistory struct {
	Items []FaceitMatchSummary
}

// FaceitMatchSummary is a single match entry from the player history endpoint.
// Winner and Score use raw faction IDs from the API (e.g., "faction1").
// Map is not included; it comes from match details (voting data).
type FaceitMatchSummary struct {
	MatchID    string
	StartedAt  int64
	FinishedAt int64
	Winner     string              // Winning faction ID (e.g., "faction1")
	Score      map[string]int      // Score keyed by faction ID
	Teams      map[string][]string // Player IDs keyed by faction ID
}

// FaceitMatchDetails holds full match information.
type FaceitMatchDetails struct {
	MatchID    string
	Map        string
	DemoURL    []string
	StartedAt  int64
	FinishedAt int64
}

// FaceitPlayerMatchStats holds a single player's stats from a match.
type FaceitPlayerMatchStats struct {
	Kills     int
	Deaths    int
	Assists   int
	Headshots int
	ADR       float64
}

// FaceitLifetimeStats holds aggregated lifetime stats for a player in a game.
type FaceitLifetimeStats struct {
	Matches int
}

// FaceitClient abstracts the Faceit Data API. The real implementation
// lives in internal/auth (HTTPFaceitClient); test mocks live in
// internal/testutil (MockFaceitClient).
type FaceitClient interface {
	GetPlayer(ctx context.Context, playerID string) (*FaceitPlayer, error)
	GetPlayerLifetimeStats(ctx context.Context, playerID string) (*FaceitLifetimeStats, error)
	GetPlayerHistory(ctx context.Context, playerID string, offset, limit int) (*FaceitMatchHistory, error)
	GetMatchDetails(ctx context.Context, matchID string) (*FaceitMatchDetails, error)
	GetMatchStats(ctx context.Context, matchID string, playerID string) (*FaceitPlayerMatchStats, error)
}

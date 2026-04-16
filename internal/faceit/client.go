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

// FaceitMatchSummary is a single match entry.
type FaceitMatchSummary struct {
	MatchID    string
	Map        string
	StartedAt  int64
	FinishedAt int64
	Winner     string
	Score      map[string]int
}

// FaceitMatchDetails holds full match information.
type FaceitMatchDetails struct {
	MatchID    string
	Map        string
	DemoURL    []string
	StartedAt  int64
	FinishedAt int64
}

// FaceitClient abstracts the Faceit Data API. The real implementation
// lives in internal/auth (HTTPFaceitClient); test mocks live in
// internal/testutil (MockFaceitClient).
type FaceitClient interface {
	GetPlayer(ctx context.Context, playerID string) (*FaceitPlayer, error)
	GetPlayerHistory(ctx context.Context, playerID string, offset, limit int) (*FaceitMatchHistory, error)
	GetMatchDetails(ctx context.Context, matchID string) (*FaceitMatchDetails, error)
}

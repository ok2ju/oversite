package faceit

import "context"

// MockFaceitClient is a configurable stub for the Faceit API.
// Set the exported fields to control what each method returns.
type MockFaceitClient struct {
	GetPlayerFn              func(ctx context.Context, playerID string) (*FaceitPlayer, error)
	GetPlayerLifetimeStatsFn func(ctx context.Context, playerID string) (*FaceitLifetimeStats, error)
	GetPlayerHistoryFn       func(ctx context.Context, playerID string, offset, limit int) (*FaceitMatchHistory, error)
	GetMatchDetailsFn        func(ctx context.Context, matchID string) (*FaceitMatchDetails, error)
	GetMatchStatsFn          func(ctx context.Context, matchID string, playerID string) (*FaceitPlayerMatchStats, error)
}

func (m *MockFaceitClient) GetPlayer(ctx context.Context, playerID string) (*FaceitPlayer, error) {
	if m.GetPlayerFn != nil {
		return m.GetPlayerFn(ctx, playerID)
	}
	return nil, nil
}

func (m *MockFaceitClient) GetPlayerLifetimeStats(ctx context.Context, playerID string) (*FaceitLifetimeStats, error) {
	if m.GetPlayerLifetimeStatsFn != nil {
		return m.GetPlayerLifetimeStatsFn(ctx, playerID)
	}
	return nil, nil
}

func (m *MockFaceitClient) GetPlayerHistory(ctx context.Context, playerID string, offset, limit int) (*FaceitMatchHistory, error) {
	if m.GetPlayerHistoryFn != nil {
		return m.GetPlayerHistoryFn(ctx, playerID, offset, limit)
	}
	return &FaceitMatchHistory{}, nil
}

func (m *MockFaceitClient) GetMatchDetails(ctx context.Context, matchID string) (*FaceitMatchDetails, error) {
	if m.GetMatchDetailsFn != nil {
		return m.GetMatchDetailsFn(ctx, matchID)
	}
	return nil, nil
}

func (m *MockFaceitClient) GetMatchStats(ctx context.Context, matchID string, playerID string) (*FaceitPlayerMatchStats, error) {
	if m.GetMatchStatsFn != nil {
		return m.GetMatchStatsFn(ctx, matchID, playerID)
	}
	return nil, nil
}

package faceit

import "context"

// MockFaceitClient is a configurable stub for the Faceit API.
// Set the exported fields to control what each method returns.
type MockFaceitClient struct {
	GetPlayerFn        func(ctx context.Context, playerID string) (*FaceitPlayer, error)
	GetPlayerHistoryFn func(ctx context.Context, playerID string, offset, limit int) (*FaceitMatchHistory, error)
	GetMatchDetailsFn  func(ctx context.Context, matchID string) (*FaceitMatchDetails, error)
}

func (m *MockFaceitClient) GetPlayer(ctx context.Context, playerID string) (*FaceitPlayer, error) {
	if m.GetPlayerFn != nil {
		return m.GetPlayerFn(ctx, playerID)
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

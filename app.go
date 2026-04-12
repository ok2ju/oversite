package main

import (
	"context"
	"errors"
)

// App struct holds application state and is bound to the frontend.
type App struct {
	ctx context.Context
}

// NewApp creates a new App instance.
func NewApp() *App {
	return &App{}
}

// Startup is called when the app starts. The context is saved
// so it can be used by other methods.
func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx
}

// Shutdown is called when the app is closing.
func (a *App) Shutdown(_ context.Context) {
}

// Greet returns a greeting for the given name.
// This is a placeholder binding to verify frontend-to-Go communication.
func (a *App) Greet(name string) string {
	return "Hello " + name + ", welcome to Oversite!"
}

// ---------------------------------------------------------------------------
// Binding stubs — wired to the frontend via Wails, implemented in later phases
// ---------------------------------------------------------------------------

var errNotImplemented = errors.New("not implemented")

// GetCurrentUser returns the currently authenticated user.
func (a *App) GetCurrentUser() (*User, error) {
	return nil, errNotImplemented
}

// LoginWithFaceit initiates the Faceit OAuth login flow.
func (a *App) LoginWithFaceit() error {
	return errNotImplemented
}

// ListDemos returns a paginated list of imported demos.
func (a *App) ListDemos(page, perPage int) (*DemoListResult, error) {
	return nil, errNotImplemented
}

// ImportDemoFile opens a native file dialog and imports the selected .dem file.
func (a *App) ImportDemoFile() error {
	return errNotImplemented
}

// DeleteDemo removes a demo by ID.
func (a *App) DeleteDemo(id int64) error {
	return errNotImplemented
}

// GetDemoRounds returns all rounds for a demo.
func (a *App) GetDemoRounds(demoID string) ([]Round, error) {
	return nil, errNotImplemented
}

// GetDemoEvents returns all game events for a demo.
func (a *App) GetDemoEvents(demoID string) ([]GameEvent, error) {
	return nil, errNotImplemented
}

// GetDemoTicks returns player tick data for a range of ticks within a demo.
func (a *App) GetDemoTicks(demoID string, startTick, endTick int) ([]TickData, error) {
	return nil, errNotImplemented
}

// GetRoundRoster returns the player roster for a specific round.
func (a *App) GetRoundRoster(demoID string, roundNumber int) ([]PlayerRosterEntry, error) {
	return nil, errNotImplemented
}

// GetFaceitProfile returns the current user's Faceit profile.
func (a *App) GetFaceitProfile() (*FaceitProfile, error) {
	return nil, errNotImplemented
}

// GetEloHistory returns elo history for the given number of days.
func (a *App) GetEloHistory(days int) ([]EloHistoryPoint, error) {
	return nil, errNotImplemented
}

// GetFaceitMatches returns a paginated, filtered list of Faceit matches.
func (a *App) GetFaceitMatches(page, perPage int, mapName, result string) (*FaceitMatchListResult, error) {
	return nil, errNotImplemented
}

package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"strconv"

	"github.com/ok2ju/oversite/internal/auth"
	"github.com/ok2ju/oversite/internal/database"
	"github.com/ok2ju/oversite/internal/demo"
	"github.com/ok2ju/oversite/internal/store"
	"github.com/ok2ju/oversite/migrations"
	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// App struct holds application state and is bound to the frontend.
type App struct {
	ctx           context.Context
	db            *sql.DB
	queries       *store.Queries
	authService   *auth.AuthService
	importService *demo.ImportService
}

// NewApp creates a new App instance.
func NewApp() *App {
	return &App{}
}

// Startup is called when the app starts. The context is saved
// so it can be used by other methods.
func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx

	dbPath, err := database.DefaultDBPath()
	if err != nil {
		log.Fatalf("failed to get database path: %v", err)
	}

	db, err := database.OpenWithMigrations(dbPath, migrations.FS)
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}
	a.db = db
	a.queries = store.New(db)

	// Demo import service.
	a.importService = demo.NewImportService(a.queries, a.db)

	// Auth service with OS keychain and real Faceit client.
	kr := &auth.RealKeyring{}
	tokens := auth.NewTokenStore(kr)
	faceitClient := auth.NewHTTPFaceitClient()

	oauthCfg := auth.OAuthConfig{
		ClientID:     "5765e820-0e1e-4dd9-9ea7-7a264010e816",
		ClientSecret: faceitClientSecret,
		AuthURL:      "https://accounts.faceit.com/accounts",
		TokenURL:     "https://api.faceit.com/auth/v1/oauth/token",
		RelayURL:     "https://ok2ju.github.io/oversite/oauth/callback",
	}

	a.authService = auth.NewAuthService(
		oauthCfg,
		tokens,
		faceitClient,
		a.queries,
		func(url string) error {
			wailsRuntime.BrowserOpenURL(a.ctx, url)
			return nil
		},
	)
}

// Shutdown is called when the app is closing.
func (a *App) Shutdown(_ context.Context) {
	if a.db != nil {
		a.db.Close()
	}
}

// Greet returns a greeting for the given name.
// This is a placeholder binding to verify frontend-to-Go communication.
func (a *App) Greet(name string) string {
	return "Hello " + name + ", welcome to Oversite!"
}

// ---------------------------------------------------------------------------
// Auth bindings
// ---------------------------------------------------------------------------

// GetCurrentUser returns the currently authenticated user.
func (a *App) GetCurrentUser() (*User, error) {
	u, err := a.authService.GetCurrentUser(a.ctx)
	if err != nil {
		return nil, err
	}
	if u == nil {
		return nil, nil
	}
	return &User{
		UserID:   strconv.FormatInt(u.ID, 10),
		FaceitID: u.FaceitID,
		Nickname: u.Nickname,
	}, nil
}

// LoginWithFaceit initiates the Faceit OAuth login flow.
func (a *App) LoginWithFaceit() error {
	_, err := a.authService.Login(a.ctx)
	return err
}

// Logout clears the authentication state and keychain.
func (a *App) Logout() error {
	return a.authService.Logout()
}

// ---------------------------------------------------------------------------
// Demo bindings
// ---------------------------------------------------------------------------

// ListDemos returns a paginated list of imported demos.
func (a *App) ListDemos(page, perPage int) (*DemoListResult, error) {
	u, err := a.authService.GetCurrentUser(a.ctx)
	if err != nil {
		return nil, err
	}
	if u == nil {
		return &DemoListResult{
			Data: []Demo{},
			Meta: PaginationMeta{Total: 0, Page: page, PerPage: perPage},
		}, nil
	}

	offset := int64((page - 1) * perPage)
	demos, err := a.queries.ListDemosByUserID(a.ctx, store.ListDemosByUserIDParams{
		UserID:    u.ID,
		OffsetVal: offset,
		LimitVal:  int64(perPage),
	})
	if err != nil {
		return nil, fmt.Errorf("listing demos: %w", err)
	}

	total, err := a.queries.CountDemosByUserID(a.ctx, u.ID)
	if err != nil {
		return nil, fmt.Errorf("counting demos: %w", err)
	}

	data := make([]Demo, len(demos))
	for i, d := range demos {
		data[i] = storeDemoToBinding(d)
	}

	return &DemoListResult{
		Data: data,
		Meta: PaginationMeta{
			Total:   int(total),
			Page:    page,
			PerPage: perPage,
		},
	}, nil
}

// ImportDemoFile opens a native file dialog and imports the selected .dem file.
func (a *App) ImportDemoFile() error {
	u, err := a.authService.GetCurrentUser(a.ctx)
	if err != nil {
		return err
	}
	if u == nil {
		return errors.New("not logged in")
	}

	filePath, err := wailsRuntime.OpenFileDialog(a.ctx, wailsRuntime.OpenDialogOptions{
		Title: "Select CS2 Demo File",
		Filters: []wailsRuntime.FileFilter{
			{DisplayName: "CS2 Demo Files (*.dem)", Pattern: "*.dem"},
		},
	})
	if err != nil {
		return fmt.Errorf("file dialog: %w", err)
	}
	if filePath == "" {
		return nil // User cancelled.
	}

	_, err = a.importService.ImportFile(a.ctx, filePath, u.ID)
	return err
}

// ImportDemoFolder opens a native directory dialog and imports all .dem files.
func (a *App) ImportDemoFolder() (*FolderImportResult, error) {
	u, err := a.authService.GetCurrentUser(a.ctx)
	if err != nil {
		return nil, err
	}
	if u == nil {
		return nil, errors.New("not logged in")
	}

	dirPath, err := wailsRuntime.OpenDirectoryDialog(a.ctx, wailsRuntime.OpenDialogOptions{
		Title: "Select Demo Folder",
	})
	if err != nil {
		return nil, fmt.Errorf("directory dialog: %w", err)
	}
	if dirPath == "" {
		return nil, nil // User cancelled.
	}

	result, err := a.importService.ImportFolder(a.ctx, dirPath, u.ID)
	if err != nil {
		return nil, err
	}

	imported := make([]Demo, len(result.Imported))
	for i, d := range result.Imported {
		imported[i] = storeDemoToBinding(*d)
	}

	importErrors := make([]string, len(result.Errors))
	for i, e := range result.Errors {
		importErrors[i] = e.Error()
	}

	return &FolderImportResult{
		Imported: imported,
		Errors:   importErrors,
	}, nil
}

// DeleteDemo removes a demo by ID.
func (a *App) DeleteDemo(id int64) error {
	return a.queries.DeleteDemo(a.ctx, id)
}

// ---------------------------------------------------------------------------
// Stub bindings — implemented in later phases
// ---------------------------------------------------------------------------

var errNotImplemented = errors.New("not implemented")

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

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func storeDemoToBinding(d store.Demo) Demo {
	return Demo{
		ID:           d.ID,
		MapName:      d.MapName,
		FilePath:     d.FilePath,
		FileSize:     d.FileSize,
		Status:       d.Status,
		TotalTicks:   int(d.TotalTicks),
		TickRate:     int(d.TickRate),
		DurationSecs: int(d.DurationSecs),
		MatchDate:    d.MatchDate,
		CreatedAt:    d.CreatedAt,
	}
}

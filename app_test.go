package main

import (
	"context"
	"database/sql"
	"strconv"
	"testing"

	"github.com/ok2ju/oversite/internal/auth"
	"github.com/ok2ju/oversite/internal/faceit"
	"github.com/ok2ju/oversite/internal/store"
	"github.com/ok2ju/oversite/internal/testutil"
)

// newTestApp returns an App backed by an in-memory SQLite database with
// migrations applied. Caller gets the queries handle for seeding test data.
func newTestApp(t *testing.T) (*App, *store.Queries) {
	t.Helper()
	q, db := testutil.NewTestQueries(t)
	app := &App{
		ctx:     context.Background(),
		db:      db,
		queries: q,
	}
	return app, q
}

// seedDemo creates a user + demo and returns the demo.
func seedDemo(t *testing.T, q *store.Queries) store.Demo {
	t.Helper()
	ctx := context.Background()
	user, err := q.CreateUser(ctx, store.CreateUserParams{
		FaceitID: "test-faceit", Nickname: "tester",
		AvatarUrl: "", FaceitElo: 2000, FaceitLevel: 10, Country: "US",
	})
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	demo, err := q.CreateDemo(ctx, store.CreateDemoParams{
		UserID: user.ID, MapName: "de_dust2",
		FilePath: "/demos/test.dem", FileSize: 100_000_000, Status: "imported",
	})
	if err != nil {
		t.Fatalf("CreateDemo: %v", err)
	}
	parsed, err := q.UpdateDemoAfterParse(ctx, store.UpdateDemoAfterParseParams{
		ID: demo.ID, MapName: "de_dust2",
		TotalTicks: 128000, TickRate: 128.0, DurationSecs: 2400,
	})
	if err != nil {
		t.Fatalf("UpdateDemoAfterParse: %v", err)
	}
	return parsed
}

// seedRounds creates rounds for a demo and returns them.
func seedRounds(t *testing.T, q *store.Queries, demoID int64) []store.Round {
	t.Helper()
	ctx := context.Background()
	rounds := []store.CreateRoundParams{
		{DemoID: demoID, RoundNumber: 1, StartTick: 0, EndTick: 3000, WinnerSide: "CT", WinReason: "CTWin", CtScore: 1, TScore: 0},
		{DemoID: demoID, RoundNumber: 2, StartTick: 3001, EndTick: 6000, WinnerSide: "T", WinReason: "TWin", CtScore: 1, TScore: 1},
		{DemoID: demoID, RoundNumber: 25, StartTick: 60000, EndTick: 63000, WinnerSide: "CT", WinReason: "CTWin", CtScore: 13, TScore: 12, IsOvertime: 1},
	}
	var result []store.Round
	for _, rp := range rounds {
		r, err := q.CreateRound(ctx, rp)
		if err != nil {
			t.Fatalf("CreateRound: %v", err)
		}
		result = append(result, r)
	}
	return result
}

// newTestAppWithUser returns an App with an AuthService wired to a real user.
// The returned user has: Nickname="tester", FaceitElo=2000, Level=10, Country="US".
func newTestAppWithUser(t *testing.T) (*App, *store.Queries, store.User) {
	t.Helper()
	q, db := testutil.NewTestQueries(t)
	ctx := context.Background()

	user, err := q.CreateUser(ctx, store.CreateUserParams{
		FaceitID: "test-faceit-id", Nickname: "tester",
		AvatarUrl: "https://example.com/avatar.png", FaceitElo: 2000,
		FaceitLevel: 10, Country: "US",
	})
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}

	kr := testutil.NewMockKeyring()
	tokens := auth.NewTokenStore(kr)
	if err := tokens.SaveUserID(strconv.FormatInt(user.ID, 10)); err != nil {
		t.Fatalf("SaveUserID: %v", err)
	}

	authSvc := auth.NewAuthService(
		auth.OAuthConfig{},
		tokens,
		&faceit.MockFaceitClient{},
		q,
		func(string) error { return nil },
	)

	app := &App{
		ctx:         ctx,
		db:          db,
		queries:     q,
		authService: authSvc,
	}
	return app, q, user
}

// seedFaceitMatches inserts 5 Faceit matches with varied results/elos/maps.
func seedFaceitMatches(t *testing.T, q *store.Queries, userID int64) []store.FaceitMatch {
	t.Helper()
	ctx := context.Background()
	params := []store.CreateFaceitMatchParams{
		{UserID: userID, FaceitMatchID: "match-1", MapName: "de_dust2", ScoreTeam: 16, ScoreOpponent: 10, Result: "win", EloBefore: 1980, EloAfter: 2000, Kills: 20, Deaths: 15, Assists: 5, DemoUrl: "https://example.com/demo1.dem.gz", PlayedAt: "2026-04-10T10:00:00Z"},
		{UserID: userID, FaceitMatchID: "match-2", MapName: "de_mirage", ScoreTeam: 14, ScoreOpponent: 16, Result: "loss", EloBefore: 2000, EloAfter: 1975, Kills: 18, Deaths: 20, Assists: 3, PlayedAt: "2026-04-11T10:00:00Z"},
		{UserID: userID, FaceitMatchID: "match-3", MapName: "de_dust2", ScoreTeam: 16, ScoreOpponent: 8, Result: "win", EloBefore: 1975, EloAfter: 2005, Kills: 25, Deaths: 10, Assists: 7, PlayedAt: "2026-04-12T10:00:00Z"},
		{UserID: userID, FaceitMatchID: "match-4", MapName: "de_inferno", ScoreTeam: 16, ScoreOpponent: 14, Result: "win", EloBefore: 2005, EloAfter: 2020, Kills: 22, Deaths: 18, Assists: 4, PlayedAt: "2026-04-13T10:00:00Z"},
		{UserID: userID, FaceitMatchID: "match-5", MapName: "de_dust2", ScoreTeam: 10, ScoreOpponent: 16, Result: "loss", EloBefore: 2020, EloAfter: 1995, Kills: 12, Deaths: 19, Assists: 2, PlayedAt: "2026-04-14T10:00:00Z"},
	}
	var matches []store.FaceitMatch
	for _, p := range params {
		m, err := q.CreateFaceitMatch(ctx, p)
		if err != nil {
			t.Fatalf("CreateFaceitMatch: %v", err)
		}
		matches = append(matches, m)
	}
	return matches
}

// ---------------------------------------------------------------------------
// GetDemoByID
// ---------------------------------------------------------------------------

func TestGetDemoByID(t *testing.T) {
	app, q := newTestApp(t)
	demo := seedDemo(t, q)

	tests := []struct {
		name    string
		id      string
		wantErr bool
	}{
		{"valid", "1", false},
		{"invalid string", "abc", true},
		{"not found", "9999", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := app.GetDemoByID(tt.id)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.ID != demo.ID {
				t.Errorf("ID = %d, want %d", got.ID, demo.ID)
			}
			if got.MapName != "de_dust2" {
				t.Errorf("MapName = %q, want %q", got.MapName, "de_dust2")
			}
			if got.TotalTicks != 128000 {
				t.Errorf("TotalTicks = %d, want 128000", got.TotalTicks)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// GetDemoRounds
// ---------------------------------------------------------------------------

func TestGetDemoRounds(t *testing.T) {
	app, q := newTestApp(t)
	demo := seedDemo(t, q)
	seedRounds(t, q, demo.ID)

	tests := []struct {
		name    string
		demoID  string
		wantLen int
		wantErr bool
	}{
		{"valid", "1", 3, false},
		{"invalid id", "abc", 0, true},
		{"empty demo", "9999", 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := app.GetDemoRounds(tt.demoID)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(got) != tt.wantLen {
				t.Fatalf("len = %d, want %d", len(got), tt.wantLen)
			}
		})
	}

	// Verify conversion details.
	rounds, _ := app.GetDemoRounds("1")
	if rounds[0].RoundNumber != 1 {
		t.Errorf("RoundNumber = %d, want 1", rounds[0].RoundNumber)
	}
	if rounds[0].WinnerSide != "CT" {
		t.Errorf("WinnerSide = %q, want CT", rounds[0].WinnerSide)
	}
	if rounds[0].IsOvertime {
		t.Error("Round 1 should not be overtime")
	}
	// Round 25 should be overtime.
	if !rounds[2].IsOvertime {
		t.Error("Round 25 should be overtime")
	}
}

// ---------------------------------------------------------------------------
// GetDemoEvents
// ---------------------------------------------------------------------------

func TestGetDemoEvents(t *testing.T) {
	app, q := newTestApp(t)
	demo := seedDemo(t, q)
	rounds := seedRounds(t, q, demo.ID)
	ctx := context.Background()

	_, err := q.CreateGameEvent(ctx, store.CreateGameEventParams{
		DemoID: demo.ID, RoundID: rounds[0].ID, Tick: 100,
		EventType:       "kill",
		AttackerSteamID: sql.NullString{String: "STEAM_A", Valid: true},
		VictimSteamID:   sql.NullString{String: "STEAM_B", Valid: true},
		Weapon:          sql.NullString{String: "ak47", Valid: true},
		X:               100.5, Y: 200.5, Z: 10.0,
		ExtraData: `{"headshot":true}`,
	})
	if err != nil {
		t.Fatalf("CreateGameEvent: %v", err)
	}
	// Event with null optional fields.
	_, err = q.CreateGameEvent(ctx, store.CreateGameEventParams{
		DemoID: demo.ID, RoundID: rounds[0].ID, Tick: 200,
		EventType: "bomb_plant",
		ExtraData: "",
	})
	if err != nil {
		t.Fatalf("CreateGameEvent: %v", err)
	}

	tests := []struct {
		name    string
		demoID  string
		wantLen int
		wantErr bool
	}{
		{"valid", "1", 2, false},
		{"invalid id", "xyz", 0, true},
		{"empty demo", "9999", 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := app.GetDemoEvents(tt.demoID)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(got) != tt.wantLen {
				t.Fatalf("len = %d, want %d", len(got), tt.wantLen)
			}
		})
	}

	// Verify conversion of first event with all fields set.
	events, _ := app.GetDemoEvents("1")
	e := events[0]
	if e.EventType != "kill" {
		t.Errorf("EventType = %q, want kill", e.EventType)
	}
	if e.AttackerSteamID == nil || *e.AttackerSteamID != "STEAM_A" {
		t.Errorf("AttackerSteamID = %v, want STEAM_A", e.AttackerSteamID)
	}
	if e.Weapon == nil || *e.Weapon != "ak47" {
		t.Errorf("Weapon = %v, want ak47", e.Weapon)
	}
	if e.ExtraData == nil {
		t.Fatal("ExtraData should not be nil")
	}
	if hs, ok := e.ExtraData["headshot"]; !ok || hs != true {
		t.Errorf("ExtraData[headshot] = %v, want true", hs)
	}

	// Verify second event with null optional fields.
	e2 := events[1]
	if e2.AttackerSteamID != nil {
		t.Error("expected nil AttackerSteamID for bomb_plant")
	}
	if e2.Weapon != nil {
		t.Error("expected nil Weapon for bomb_plant")
	}
	if e2.ExtraData != nil {
		t.Error("expected nil ExtraData for empty string")
	}
}

// ---------------------------------------------------------------------------
// GetDemoTicks
// ---------------------------------------------------------------------------

func TestGetDemoTicks(t *testing.T) {
	app, q := newTestApp(t)
	demo := seedDemo(t, q)
	ctx := context.Background()

	tickParams := []store.InsertTickDataParams{
		{DemoID: demo.ID, Tick: 100, SteamID: "STEAM_A", X: 1, Y: 2, Z: 3, Yaw: 90, Health: 100, Armor: 100, IsAlive: 1, Weapon: "ak47"},
		{DemoID: demo.ID, Tick: 100, SteamID: "STEAM_B", X: 4, Y: 5, Z: 6, Yaw: 180, Health: 0, Armor: 0, IsAlive: 0, Weapon: ""},
		{DemoID: demo.ID, Tick: 200, SteamID: "STEAM_A", X: 10, Y: 20, Z: 30, Yaw: 45, Health: 80, Armor: 50, IsAlive: 1, Weapon: "m4a1"},
	}
	for _, tp := range tickParams {
		if err := q.InsertTickData(ctx, tp); err != nil {
			t.Fatalf("InsertTickData: %v", err)
		}
	}

	tests := []struct {
		name    string
		demoID  string
		start   int
		end     int
		wantLen int
		wantErr bool
	}{
		{"full range", "1", 0, 300, 3, false},
		{"partial range", "1", 100, 100, 2, false},
		{"empty range", "1", 500, 600, 0, false},
		{"invalid id", "abc", 0, 100, 0, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := app.GetDemoTicks(tt.demoID, tt.start, tt.end)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(got) != tt.wantLen {
				t.Fatalf("len = %d, want %d", len(got), tt.wantLen)
			}
		})
	}

	// Verify alive/dead conversion.
	ticks, _ := app.GetDemoTicks("1", 100, 100)
	if !ticks[0].IsAlive {
		t.Error("STEAM_A should be alive")
	}
	if ticks[0].Weapon == nil || *ticks[0].Weapon != "ak47" {
		t.Errorf("Weapon = %v, want ak47", ticks[0].Weapon)
	}
	if ticks[1].IsAlive {
		t.Error("STEAM_B should be dead")
	}
	if ticks[1].Weapon != nil {
		t.Error("expected nil Weapon for empty weapon string")
	}
}

// ---------------------------------------------------------------------------
// GetRoundRoster
// ---------------------------------------------------------------------------

func TestGetRoundRoster(t *testing.T) {
	app, q := newTestApp(t)
	demo := seedDemo(t, q)
	rounds := seedRounds(t, q, demo.ID)
	ctx := context.Background()

	players := []store.CreatePlayerRoundParams{
		{RoundID: rounds[0].ID, SteamID: "STEAM_A", PlayerName: "Player1", TeamSide: "CT", Kills: 2, Deaths: 0, Assists: 1, Damage: 200, HeadshotKills: 1},
		{RoundID: rounds[0].ID, SteamID: "STEAM_B", PlayerName: "Player2", TeamSide: "T", Kills: 0, Deaths: 1, Assists: 0, Damage: 50},
	}
	for _, pp := range players {
		if _, err := q.CreatePlayerRound(ctx, pp); err != nil {
			t.Fatalf("CreatePlayerRound: %v", err)
		}
	}

	tests := []struct {
		name        string
		demoID      string
		roundNumber int
		wantLen     int
		wantErr     bool
	}{
		{"valid round", "1", 1, 2, false},
		{"non-existent round", "1", 99, 0, false},
		{"invalid id", "abc", 1, 0, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := app.GetRoundRoster(tt.demoID, tt.roundNumber)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(got) != tt.wantLen {
				t.Fatalf("len = %d, want %d", len(got), tt.wantLen)
			}
		})
	}

	roster, _ := app.GetRoundRoster("1", 1)
	if roster[0].SteamID != "STEAM_A" {
		t.Errorf("SteamID = %q, want STEAM_A", roster[0].SteamID)
	}
	if roster[0].PlayerName != "Player1" {
		t.Errorf("PlayerName = %q, want Player1", roster[0].PlayerName)
	}
}

// ---------------------------------------------------------------------------
// GetScoreboard
// ---------------------------------------------------------------------------

func TestGetScoreboard(t *testing.T) {
	app, q := newTestApp(t)
	demo := seedDemo(t, q)
	rounds := seedRounds(t, q, demo.ID)
	ctx := context.Background()

	// Player A: 2 rounds, 5 kills (3 hs), 2 deaths, 300 damage.
	for i, roundIdx := range []int{0, 1} {
		kills := int64(2)
		hs := int64(1)
		dmg := int64(150)
		if i == 1 {
			kills = 3
			hs = 2
		}
		_, err := q.CreatePlayerRound(ctx, store.CreatePlayerRoundParams{
			RoundID: rounds[roundIdx].ID, SteamID: "STEAM_A", PlayerName: "Player1",
			TeamSide: "CT", Kills: kills, Deaths: 1, Assists: 1,
			Damage: dmg, HeadshotKills: hs,
		})
		if err != nil {
			t.Fatalf("CreatePlayerRound: %v", err)
		}
	}

	tests := []struct {
		name    string
		demoID  string
		wantLen int
		wantErr bool
	}{
		{"valid", "1", 1, false},
		{"invalid id", "xyz", 0, true},
		{"empty demo", "9999", 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := app.GetScoreboard(tt.demoID)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(got) != tt.wantLen {
				t.Fatalf("len = %d, want %d", len(got), tt.wantLen)
			}
		})
	}

	board, _ := app.GetScoreboard("1")
	entry := board[0]
	if entry.Kills != 5 {
		t.Errorf("Kills = %d, want 5", entry.Kills)
	}
	if entry.Deaths != 2 {
		t.Errorf("Deaths = %d, want 2", entry.Deaths)
	}
	if entry.HSKills != 3 {
		t.Errorf("HSKills = %d, want 3", entry.HSKills)
	}
	if entry.RoundsPlayed != 2 {
		t.Errorf("RoundsPlayed = %d, want 2", entry.RoundsPlayed)
	}
	// HSPercent = 3/5 * 100 = 60
	if entry.HSPercent != 60 {
		t.Errorf("HSPercent = %f, want 60", entry.HSPercent)
	}
	// ADR = 300/2 = 150
	if entry.ADR != 150 {
		t.Errorf("ADR = %f, want 150", entry.ADR)
	}
}

// ---------------------------------------------------------------------------
// computeStreak
// ---------------------------------------------------------------------------

func TestComputeStreak(t *testing.T) {
	tests := []struct {
		name      string
		results   []string
		wantType  string
		wantCount int
	}{
		{"empty", nil, "none", 0},
		{"single win", []string{"win"}, "win", 1},
		{"single loss", []string{"loss"}, "loss", 1},
		{"win streak", []string{"win", "win", "win", "loss"}, "win", 3},
		{"loss streak", []string{"loss", "loss", "win"}, "loss", 2},
		{"alternating", []string{"win", "loss", "win"}, "win", 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := computeStreak(tt.results)
			if got.Type != tt.wantType {
				t.Errorf("Type = %q, want %q", got.Type, tt.wantType)
			}
			if got.Count != tt.wantCount {
				t.Errorf("Count = %d, want %d", got.Count, tt.wantCount)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// GetFaceitProfile
// ---------------------------------------------------------------------------

func TestGetFaceitProfile(t *testing.T) {
	t.Run("not logged in", func(t *testing.T) {
		q, db := testutil.NewTestQueries(t)
		kr := testutil.NewMockKeyring()
		tokens := auth.NewTokenStore(kr)
		app := &App{
			ctx: context.Background(), db: db, queries: q,
			authService: auth.NewAuthService(auth.OAuthConfig{}, tokens, &faceit.MockFaceitClient{}, q, func(string) error { return nil }),
		}
		_, err := app.GetFaceitProfile()
		if err == nil {
			t.Fatal("expected error for unauthenticated user")
		}
	})

	t.Run("no matches", func(t *testing.T) {
		app, _, _ := newTestAppWithUser(t)
		profile, err := app.GetFaceitProfile()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if profile.Nickname != "tester" {
			t.Errorf("Nickname = %q, want %q", profile.Nickname, "tester")
		}
		if profile.Elo == nil || *profile.Elo != 2000 {
			t.Errorf("Elo = %v, want 2000", profile.Elo)
		}
		if profile.Level == nil || *profile.Level != 10 {
			t.Errorf("Level = %v, want 10", profile.Level)
		}
		if profile.MatchesPlayed != 0 {
			t.Errorf("MatchesPlayed = %d, want 0", profile.MatchesPlayed)
		}
		if profile.CurrentStreak.Type != "none" {
			t.Errorf("Streak.Type = %q, want %q", profile.CurrentStreak.Type, "none")
		}
	})

	t.Run("with matches and streak", func(t *testing.T) {
		app, q, user := newTestAppWithUser(t)
		seedFaceitMatches(t, q, user.ID)

		profile, err := app.GetFaceitProfile()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if profile.MatchesPlayed != 5 {
			t.Errorf("MatchesPlayed = %d, want 5", profile.MatchesPlayed)
		}
		// Matches ordered by played_at DESC: match-5 (loss), match-4 (win), ...
		// So streak should be "loss", count=1.
		if profile.CurrentStreak.Type != "loss" {
			t.Errorf("Streak.Type = %q, want %q", profile.CurrentStreak.Type, "loss")
		}
		if profile.CurrentStreak.Count != 1 {
			t.Errorf("Streak.Count = %d, want 1", profile.CurrentStreak.Count)
		}
		if profile.AvatarURL == nil || *profile.AvatarURL != "https://example.com/avatar.png" {
			t.Errorf("AvatarURL = %v, want avatar URL", profile.AvatarURL)
		}
		if profile.Country == nil || *profile.Country != "US" {
			t.Errorf("Country = %v, want US", profile.Country)
		}
	})
}

// ---------------------------------------------------------------------------
// GetEloHistory
// ---------------------------------------------------------------------------

func TestGetEloHistory(t *testing.T) {
	t.Run("all time", func(t *testing.T) {
		app, q, user := newTestAppWithUser(t)
		seedFaceitMatches(t, q, user.ID)

		points, err := app.GetEloHistory(0)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(points) != 5 {
			t.Fatalf("len = %d, want 5", len(points))
		}
		// Ordered ASC by played_at. First match: elo_after=2000.
		if points[0].Elo == nil || *points[0].Elo != 2000 {
			t.Errorf("points[0].Elo = %v, want 2000", points[0].Elo)
		}
		if points[0].MapName != "de_dust2" {
			t.Errorf("points[0].MapName = %q, want de_dust2", points[0].MapName)
		}
	})

	t.Run("empty", func(t *testing.T) {
		app, _, _ := newTestAppWithUser(t)
		points, err := app.GetEloHistory(30)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(points) != 0 {
			t.Errorf("len = %d, want 0", len(points))
		}
	})

	t.Run("filtered by days", func(t *testing.T) {
		app, q, user := newTestAppWithUser(t)
		seedFaceitMatches(t, q, user.ID)

		// All test matches are in Apr 2026 — 1 day window won't include them.
		points, err := app.GetEloHistory(1)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(points) != 0 {
			t.Errorf("len = %d, want 0 (all matches are in the past)", len(points))
		}
	})
}

// ---------------------------------------------------------------------------
// GetFaceitMatches
// ---------------------------------------------------------------------------

func TestGetFaceitMatches(t *testing.T) {
	app, q, user := newTestAppWithUser(t)
	seedFaceitMatches(t, q, user.ID)

	t.Run("unfiltered", func(t *testing.T) {
		result, err := app.GetFaceitMatches(1, 10, "", "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Meta.Total != 5 {
			t.Errorf("Total = %d, want 5", result.Meta.Total)
		}
		if len(result.Data) != 5 {
			t.Fatalf("len = %d, want 5", len(result.Data))
		}
		// Ordered DESC by played_at, first should be match-5.
		if result.Data[0].FaceitMatchID != "match-5" {
			t.Errorf("first match = %q, want match-5", result.Data[0].FaceitMatchID)
		}
	})

	t.Run("filter by map", func(t *testing.T) {
		result, err := app.GetFaceitMatches(1, 10, "de_dust2", "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Meta.Total != 3 {
			t.Errorf("Total = %d, want 3", result.Meta.Total)
		}
	})

	t.Run("filter by result", func(t *testing.T) {
		result, err := app.GetFaceitMatches(1, 10, "", "win")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Meta.Total != 3 {
			t.Errorf("Total = %d, want 3", result.Meta.Total)
		}
	})

	t.Run("pagination", func(t *testing.T) {
		result, err := app.GetFaceitMatches(1, 2, "", "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result.Data) != 2 {
			t.Fatalf("len = %d, want 2", len(result.Data))
		}
		if result.Meta.Total != 5 {
			t.Errorf("Total = %d, want 5", result.Meta.Total)
		}
		if result.Meta.Page != 1 {
			t.Errorf("Page = %d, want 1", result.Meta.Page)
		}
	})

	t.Run("type mapping", func(t *testing.T) {
		result, err := app.GetFaceitMatches(1, 1, "de_dust2", "win")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result.Data) == 0 {
			t.Fatal("expected at least 1 match")
		}
		m := result.Data[0]
		// Match-3: EloBefore=1975, EloAfter=2005, EloChange=+30.
		if m.EloBefore == nil || *m.EloBefore != 1975 {
			t.Errorf("EloBefore = %v, want 1975", m.EloBefore)
		}
		if m.EloAfter == nil || *m.EloAfter != 2005 {
			t.Errorf("EloAfter = %v, want 2005", m.EloAfter)
		}
		if m.EloChange == nil || *m.EloChange != 30 {
			t.Errorf("EloChange = %v, want 30", m.EloChange)
		}
		if m.DemoURL != nil {
			t.Errorf("DemoURL = %v, want nil (match-3 has no demo_url)", m.DemoURL)
		}
		if m.HasDemo {
			t.Error("HasDemo should be false")
		}
	})
}

// ---------------------------------------------------------------------------
// Heatmap helpers
// ---------------------------------------------------------------------------

// seedKillEvents creates game events with kills for heatmap testing.
func seedKillEvents(t *testing.T, q *store.Queries, demoID int64, rounds []store.Round) {
	t.Helper()
	ctx := context.Background()

	// Also seed player_rounds so heatmap side-filtering can join.
	_, err := q.CreatePlayerRound(ctx, store.CreatePlayerRoundParams{
		RoundID: rounds[0].ID, SteamID: "STEAM_A", PlayerName: "Player1",
		TeamSide: "CT", Kills: 2, Deaths: 0, Assists: 0, Damage: 200, HeadshotKills: 1,
	})
	if err != nil {
		t.Fatalf("CreatePlayerRound: %v", err)
	}
	_, err = q.CreatePlayerRound(ctx, store.CreatePlayerRoundParams{
		RoundID: rounds[0].ID, SteamID: "STEAM_B", PlayerName: "Player2",
		TeamSide: "T", Kills: 1, Deaths: 1, Assists: 0, Damage: 100,
	})
	if err != nil {
		t.Fatalf("CreatePlayerRound: %v", err)
	}

	events := []store.CreateGameEventParams{
		{DemoID: demoID, RoundID: rounds[0].ID, Tick: 100, EventType: "kill",
			AttackerSteamID: sql.NullString{String: "STEAM_A", Valid: true},
			VictimSteamID:   sql.NullString{String: "STEAM_B", Valid: true},
			Weapon:          sql.NullString{String: "AK-47", Valid: true},
			X:               100.5, Y: 200.5, Z: 10.0, ExtraData: `{"headshot":true}`},
		{DemoID: demoID, RoundID: rounds[0].ID, Tick: 200, EventType: "kill",
			AttackerSteamID: sql.NullString{String: "STEAM_A", Valid: true},
			VictimSteamID:   sql.NullString{String: "STEAM_B", Valid: true},
			Weapon:          sql.NullString{String: "AK-47", Valid: true},
			X:               100.5, Y: 200.5, Z: 10.0, ExtraData: `{"headshot":false}`},
		{DemoID: demoID, RoundID: rounds[0].ID, Tick: 300, EventType: "kill",
			AttackerSteamID: sql.NullString{String: "STEAM_B", Valid: true},
			VictimSteamID:   sql.NullString{String: "STEAM_A", Valid: true},
			Weapon:          sql.NullString{String: "AWP", Valid: true},
			X:               300.0, Y: 400.0, Z: 5.0, ExtraData: `{"headshot":true}`},
	}
	for _, ep := range events {
		if _, err := q.CreateGameEvent(ctx, ep); err != nil {
			t.Fatalf("CreateGameEvent: %v", err)
		}
	}
}

// ---------------------------------------------------------------------------
// GetHeatmapData
// ---------------------------------------------------------------------------

func TestGetHeatmapData(t *testing.T) {
	app, q, user := newTestAppWithUser(t)
	ctx := context.Background()

	demo, err := q.CreateDemo(ctx, store.CreateDemoParams{
		UserID: user.ID, MapName: "de_dust2",
		FilePath: "/demos/heatmap.dem", FileSize: 100_000_000, Status: "imported",
	})
	if err != nil {
		t.Fatalf("CreateDemo: %v", err)
	}
	demo, err = q.UpdateDemoAfterParse(ctx, store.UpdateDemoAfterParseParams{
		ID: demo.ID, MapName: "de_dust2", TotalTicks: 128000, TickRate: 128.0, DurationSecs: 2400,
	})
	if err != nil {
		t.Fatalf("UpdateDemoAfterParse: %v", err)
	}
	rounds := seedRounds(t, q, demo.ID)
	seedKillEvents(t, q, demo.ID, rounds)

	t.Run("all kills", func(t *testing.T) {
		points, err := app.GetHeatmapData([]int64{demo.ID}, []string{}, "", "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(points) != 2 {
			t.Fatalf("len = %d, want 2 (two distinct positions)", len(points))
		}
		// Position (100.5, 200.5) has 2 kills.
		found := false
		for _, p := range points {
			if p.X == 100.5 && p.Y == 200.5 {
				if p.KillCount != 2 {
					t.Errorf("KillCount = %d, want 2", p.KillCount)
				}
				found = true
			}
		}
		if !found {
			t.Error("expected point at (100.5, 200.5)")
		}
	})

	t.Run("filter by weapon", func(t *testing.T) {
		points, err := app.GetHeatmapData([]int64{demo.ID}, []string{"AWP"}, "", "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(points) != 1 {
			t.Fatalf("len = %d, want 1", len(points))
		}
		if points[0].X != 300.0 {
			t.Errorf("X = %f, want 300.0", points[0].X)
		}
	})

	t.Run("filter by player", func(t *testing.T) {
		points, err := app.GetHeatmapData([]int64{demo.ID}, []string{}, "STEAM_A", "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(points) != 1 {
			t.Fatalf("len = %d, want 1 (only STEAM_A kills at one position)", len(points))
		}
		if points[0].KillCount != 2 {
			t.Errorf("KillCount = %d, want 2", points[0].KillCount)
		}
	})

	t.Run("filter by side", func(t *testing.T) {
		points, err := app.GetHeatmapData([]int64{demo.ID}, []string{}, "", "CT")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// Only STEAM_A is CT, so only their 2 kills at (100.5, 200.5).
		if len(points) != 1 {
			t.Fatalf("len = %d, want 1", len(points))
		}
		if points[0].KillCount != 2 {
			t.Errorf("KillCount = %d, want 2", points[0].KillCount)
		}
	})

	t.Run("empty for nonexistent demo", func(t *testing.T) {
		points, err := app.GetHeatmapData([]int64{9999}, []string{}, "", "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(points) != 0 {
			t.Errorf("len = %d, want 0", len(points))
		}
	})
}

// ---------------------------------------------------------------------------
// GetUniqueWeapons
// ---------------------------------------------------------------------------

func TestGetUniqueWeapons(t *testing.T) {
	app, q, user := newTestAppWithUser(t)
	ctx := context.Background()

	demo, err := q.CreateDemo(ctx, store.CreateDemoParams{
		UserID: user.ID, MapName: "de_dust2",
		FilePath: "/demos/weapons.dem", FileSize: 100_000_000, Status: "imported",
	})
	if err != nil {
		t.Fatalf("CreateDemo: %v", err)
	}
	demo, err = q.UpdateDemoAfterParse(ctx, store.UpdateDemoAfterParseParams{
		ID: demo.ID, MapName: "de_dust2", TotalTicks: 128000, TickRate: 128.0, DurationSecs: 2400,
	})
	if err != nil {
		t.Fatalf("UpdateDemoAfterParse: %v", err)
	}
	rounds := seedRounds(t, q, demo.ID)
	seedKillEvents(t, q, demo.ID, rounds)

	t.Run("returns distinct weapons", func(t *testing.T) {
		weapons, err := app.GetUniqueWeapons([]int64{demo.ID})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(weapons) != 2 {
			t.Fatalf("len = %d, want 2 (AK-47, AWP)", len(weapons))
		}
	})

	t.Run("empty for no demos", func(t *testing.T) {
		weapons, err := app.GetUniqueWeapons([]int64{9999})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(weapons) != 0 {
			t.Errorf("len = %d, want 0", len(weapons))
		}
	})
}

// ---------------------------------------------------------------------------
// GetUniquePlayers
// ---------------------------------------------------------------------------

func TestGetUniquePlayers(t *testing.T) {
	app, q, user := newTestAppWithUser(t)
	ctx := context.Background()

	demo, err := q.CreateDemo(ctx, store.CreateDemoParams{
		UserID: user.ID, MapName: "de_dust2",
		FilePath: "/demos/players.dem", FileSize: 100_000_000, Status: "imported",
	})
	if err != nil {
		t.Fatalf("CreateDemo: %v", err)
	}
	demo, err = q.UpdateDemoAfterParse(ctx, store.UpdateDemoAfterParseParams{
		ID: demo.ID, MapName: "de_dust2", TotalTicks: 128000, TickRate: 128.0, DurationSecs: 2400,
	})
	if err != nil {
		t.Fatalf("UpdateDemoAfterParse: %v", err)
	}
	rounds := seedRounds(t, q, demo.ID)
	seedKillEvents(t, q, demo.ID, rounds)

	t.Run("returns distinct players", func(t *testing.T) {
		players, err := app.GetUniquePlayers([]int64{demo.ID})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(players) != 2 {
			t.Fatalf("len = %d, want 2", len(players))
		}
		// Check that we get real player info.
		found := false
		for _, p := range players {
			if p.SteamID == "STEAM_A" && p.PlayerName == "Player1" {
				found = true
			}
		}
		if !found {
			t.Error("expected player STEAM_A / Player1")
		}
	})
}

// ---------------------------------------------------------------------------
// GetWeaponStats
// ---------------------------------------------------------------------------

func TestGetWeaponStats(t *testing.T) {
	app, q := newTestApp(t)
	demo := seedDemo(t, q)
	rounds := seedRounds(t, q, demo.ID)
	ctx := context.Background()

	// Seed player_rounds for the join.
	_, err := q.CreatePlayerRound(ctx, store.CreatePlayerRoundParams{
		RoundID: rounds[0].ID, SteamID: "STEAM_A", PlayerName: "Player1",
		TeamSide: "CT", Kills: 2, Deaths: 0, Assists: 0, Damage: 200, HeadshotKills: 1,
	})
	if err != nil {
		t.Fatalf("CreatePlayerRound: %v", err)
	}

	events := []store.CreateGameEventParams{
		{DemoID: demo.ID, RoundID: rounds[0].ID, Tick: 100, EventType: "kill",
			AttackerSteamID: sql.NullString{String: "STEAM_A", Valid: true},
			Weapon:          sql.NullString{String: "AK-47", Valid: true},
			X:               100, Y: 200, Z: 10, ExtraData: `{"headshot":true}`},
		{DemoID: demo.ID, RoundID: rounds[0].ID, Tick: 200, EventType: "kill",
			AttackerSteamID: sql.NullString{String: "STEAM_A", Valid: true},
			Weapon:          sql.NullString{String: "AK-47", Valid: true},
			X:               100, Y: 200, Z: 10, ExtraData: `{"headshot":false}`},
		{DemoID: demo.ID, RoundID: rounds[0].ID, Tick: 300, EventType: "kill",
			AttackerSteamID: sql.NullString{String: "STEAM_A", Valid: true},
			Weapon:          sql.NullString{String: "AWP", Valid: true},
			X:               300, Y: 400, Z: 5, ExtraData: `{"headshot":true}`},
	}
	for _, ep := range events {
		if _, err := q.CreateGameEvent(ctx, ep); err != nil {
			t.Fatalf("CreateGameEvent: %v", err)
		}
	}

	tests := []struct {
		name    string
		demoID  string
		wantLen int
		wantErr bool
	}{
		{"valid", strconv.FormatInt(demo.ID, 10), 2, false},
		{"invalid id", "abc", 0, true},
		{"empty demo", "9999", 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := app.GetWeaponStats(tt.demoID)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(got) != tt.wantLen {
				t.Fatalf("len = %d, want %d", len(got), tt.wantLen)
			}
		})
	}

	// Verify stats ordering and HS counts.
	stats, _ := app.GetWeaponStats(strconv.FormatInt(demo.ID, 10))
	// AK-47 has 2 kills (1 HS), should be first (ordered by kill_count DESC).
	if stats[0].Weapon != "AK-47" {
		t.Errorf("Weapon[0] = %q, want AK-47", stats[0].Weapon)
	}
	if stats[0].KillCount != 2 {
		t.Errorf("KillCount[0] = %d, want 2", stats[0].KillCount)
	}
	if stats[0].HSCount != 1 {
		t.Errorf("HSCount[0] = %d, want 1", stats[0].HSCount)
	}
	// AWP has 1 kill (1 HS).
	if stats[1].Weapon != "AWP" {
		t.Errorf("Weapon[1] = %q, want AWP", stats[1].Weapon)
	}
	if stats[1].KillCount != 1 {
		t.Errorf("KillCount[1] = %d, want 1", stats[1].KillCount)
	}
	if stats[1].HSCount != 1 {
		t.Errorf("HSCount[1] = %d, want 1", stats[1].HSCount)
	}
}

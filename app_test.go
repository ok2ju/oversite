package main

import (
	"context"
	"database/sql"
	"testing"

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
		{DemoID: demoID, RoundNumber: 25, StartTick: 60000, EndTick: 63000, WinnerSide: "CT", WinReason: "CTWin", CtScore: 13, TScore: 12},
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

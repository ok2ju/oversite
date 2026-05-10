package main

import (
	"bytes"
	"context"
	"database/sql"
	"strconv"
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

// seedDemo creates a demo and returns it.
func seedDemo(t *testing.T, q *store.Queries) store.Demo {
	t.Helper()
	ctx := context.Background()
	demo, err := q.CreateDemo(ctx, store.CreateDemoParams{
		MapName:  "de_dust2",
		FilePath: "/demos/test.dem",
		FileSize: 100_000_000,
		Status:   "imported",
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
// GetMinEngagementsForAimCritique
// ---------------------------------------------------------------------------

// TestGetMinEngagementsForAimCritique_DefaultsAndRoundtrip asserts the binding
// reports the slice-8 default of 8 on a fresh App and roundtrips arbitrary
// non-negative values via SetX → GetX. Matches the trivial-default-roundtrip
// pattern; mirrors the SetTolerateEntityErrors binding shape.
func TestGetMinEngagementsForAimCritique_DefaultsAndRoundtrip(t *testing.T) {
	app := &App{minEngagementsForAimCritique: 8}
	if got := app.GetMinEngagementsForAimCritique(); got != 8 {
		t.Errorf("default = %d, want 8", got)
	}

	app.SetMinEngagementsForAimCritique(5)
	if got := app.GetMinEngagementsForAimCritique(); got != 5 {
		t.Errorf("after Set(5) = %d, want 5", got)
	}

	app.SetMinEngagementsForAimCritique(0)
	if got := app.GetMinEngagementsForAimCritique(); got != 0 {
		t.Errorf("after Set(0) = %d, want 0", got)
	}

	// Negative values are clamped to 0 — keeps the gate sensible without
	// surfacing a typed enum to the frontend.
	app.SetMinEngagementsForAimCritique(-3)
	if got := app.GetMinEngagementsForAimCritique(); got != 0 {
		t.Errorf("after Set(-3) = %d, want 0 (clamped)", got)
	}
}

// ---------------------------------------------------------------------------
// GetAnalysisStatus
// ---------------------------------------------------------------------------

func TestGetAnalysisStatus(t *testing.T) {
	app, q := newTestApp(t)
	ctx := context.Background()

	// Helper to create a demo at a given lifecycle status. Bypasses the
	// seedDemo helper because tests below need each row to stay in its
	// originally-imported state without the UpdateDemoAfterParse flip to
	// "ready".
	createDemo := func(t *testing.T, status string) store.Demo {
		t.Helper()
		d, err := q.CreateDemo(ctx, store.CreateDemoParams{
			MapName:  "de_dust2",
			FilePath: "/demos/" + status + ".dem",
			FileSize: 100,
			Status:   status,
		})
		if err != nil {
			t.Fatalf("CreateDemo(%s): %v", status, err)
		}
		return d
	}

	t.Run("ready with rows returns ready", func(t *testing.T) {
		d := createDemo(t, "imported")
		if _, err := q.UpdateDemoAfterParse(ctx, store.UpdateDemoAfterParseParams{
			ID: d.ID, MapName: "de_dust2",
			TotalTicks: 1, TickRate: 64, DurationSecs: 1,
		}); err != nil {
			t.Fatalf("UpdateDemoAfterParse: %v", err)
		}
		if err := q.UpsertPlayerMatchAnalysis(ctx, store.UpsertPlayerMatchAnalysisParams{
			DemoID: d.ID, SteamID: "STEAM_A",
			OverallScore: 75, TradePct: 0.6, AvgTradeTicks: 90, ExtrasJson: "{}",
		}); err != nil {
			t.Fatalf("UpsertPlayerMatchAnalysis: %v", err)
		}

		got, err := app.GetAnalysisStatus(strconv.FormatInt(d.ID, 10))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.Status != "ready" {
			t.Errorf("Status = %q, want ready", got.Status)
		}
		if got.DemoID != strconv.FormatInt(d.ID, 10) {
			t.Errorf("DemoID = %q, want %q", got.DemoID, strconv.FormatInt(d.ID, 10))
		}
	})

	t.Run("ready without rows returns missing", func(t *testing.T) {
		d := createDemo(t, "imported")
		if _, err := q.UpdateDemoAfterParse(ctx, store.UpdateDemoAfterParseParams{
			ID: d.ID, MapName: "de_dust2",
			TotalTicks: 1, TickRate: 64, DurationSecs: 1,
		}); err != nil {
			t.Fatalf("UpdateDemoAfterParse: %v", err)
		}

		got, err := app.GetAnalysisStatus(strconv.FormatInt(d.ID, 10))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.Status != "missing" {
			t.Errorf("Status = %q, want missing", got.Status)
		}
	})

	t.Run("parsing surfaces verbatim", func(t *testing.T) {
		d := createDemo(t, "imported")
		if _, err := q.UpdateDemoStatus(ctx, store.UpdateDemoStatusParams{
			Status: "parsing", ID: d.ID,
		}); err != nil {
			t.Fatalf("UpdateDemoStatus: %v", err)
		}

		got, err := app.GetAnalysisStatus(strconv.FormatInt(d.ID, 10))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.Status != "parsing" {
			t.Errorf("Status = %q, want parsing", got.Status)
		}
	})

	t.Run("failed surfaces verbatim", func(t *testing.T) {
		d := createDemo(t, "imported")
		if _, err := q.UpdateDemoStatus(ctx, store.UpdateDemoStatusParams{
			Status: "failed", ID: d.ID,
		}); err != nil {
			t.Fatalf("UpdateDemoStatus: %v", err)
		}

		got, err := app.GetAnalysisStatus(strconv.FormatInt(d.ID, 10))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.Status != "failed" {
			t.Errorf("Status = %q, want failed", got.Status)
		}
	})

	t.Run("imported surfaces verbatim", func(t *testing.T) {
		d := createDemo(t, "imported")

		got, err := app.GetAnalysisStatus(strconv.FormatInt(d.ID, 10))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.Status != "imported" {
			t.Errorf("Status = %q, want imported", got.Status)
		}
	})

	t.Run("unknown demo id returns error", func(t *testing.T) {
		if _, err := app.GetAnalysisStatus("999999"); err == nil {
			t.Fatal("expected error for unknown demo, got nil")
		}
	})

	t.Run("invalid demo id returns error", func(t *testing.T) {
		if _, err := app.GetAnalysisStatus("abc"); err == nil {
			t.Fatal("expected error for invalid demo id, got nil")
		}
	})
}

// ---------------------------------------------------------------------------
// GetPlayerRoundAnalysis
// ---------------------------------------------------------------------------

func TestGetPlayerRoundAnalysis(t *testing.T) {
	app, q := newTestApp(t)
	d := seedDemo(t, q)
	ctx := context.Background()

	// Seed three rows for steam_a (out-of-order on purpose to verify the
	// query's ORDER BY round_number) and one row for steam_b. extras_json
	// covers the empty default ("{}"), an empty string, and a populated map.
	rowsToSeed := []struct {
		steam    string
		round    int64
		tradePct float64
		extras   string
	}{
		{"STEAM_A", 3, 0.5, "{}"},
		{"STEAM_A", 1, 1.0, ""},
		{"STEAM_A", 2, 0.0, `{"reason":"untraded"}`},
		{"STEAM_B", 1, 0.25, "{}"},
	}
	for _, r := range rowsToSeed {
		if err := q.UpsertPlayerRoundAnalysis(ctx, store.UpsertPlayerRoundAnalysisParams{
			DemoID:      d.ID,
			SteamID:     r.steam,
			RoundNumber: r.round,
			TradePct:    r.tradePct,
			ExtrasJson:  r.extras,
		}); err != nil {
			t.Fatalf("UpsertPlayerRoundAnalysis(%s, round=%d): %v", r.steam, r.round, err)
		}
	}

	t.Run("invalid demo id returns error", func(t *testing.T) {
		if _, err := app.GetPlayerRoundAnalysis("abc", "STEAM_A"); err == nil {
			t.Fatal("expected error for invalid demo id, got nil")
		}
	})

	t.Run("empty steamID returns empty slice", func(t *testing.T) {
		got, err := app.GetPlayerRoundAnalysis(strconv.FormatInt(d.ID, 10), "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(got) != 0 {
			t.Errorf("expected empty slice, got %d entries", len(got))
		}
	})

	t.Run("populated rows ordered by round_number ASC", func(t *testing.T) {
		got, err := app.GetPlayerRoundAnalysis(strconv.FormatInt(d.ID, 10), "STEAM_A")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(got) != 3 {
			t.Fatalf("expected 3 rows, got %d", len(got))
		}
		wantRounds := []int{1, 2, 3}
		for i, want := range wantRounds {
			if got[i].RoundNumber != want {
				t.Errorf("got[%d].RoundNumber = %d, want %d", i, got[i].RoundNumber, want)
			}
		}
		if got[0].TradePct != 1.0 {
			t.Errorf("round 1 TradePct = %v, want 1.0", got[0].TradePct)
		}
	})

	t.Run("empty extras_json yields nil Extras", func(t *testing.T) {
		got, err := app.GetPlayerRoundAnalysis(strconv.FormatInt(d.ID, 10), "STEAM_A")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// Round 3 was seeded with extras_json = "{}" — defensive unmarshal
		// short-circuits and leaves Extras nil.
		if got[2].RoundNumber != 3 {
			t.Fatalf("got[2].RoundNumber = %d, want 3", got[2].RoundNumber)
		}
		if got[2].Extras != nil {
			t.Errorf("got[2].Extras = %v, want nil for {}", got[2].Extras)
		}
		// Round 1 was seeded with extras_json = "" — also short-circuits.
		if got[0].Extras != nil {
			t.Errorf("got[0].Extras = %v, want nil for empty string", got[0].Extras)
		}
	})

	t.Run("populated extras_json yields unmarshalled map", func(t *testing.T) {
		got, err := app.GetPlayerRoundAnalysis(strconv.FormatInt(d.ID, 10), "STEAM_A")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// Round 2 was seeded with extras_json = `{"reason":"untraded"}`.
		if got[1].RoundNumber != 2 {
			t.Fatalf("got[1].RoundNumber = %d, want 2", got[1].RoundNumber)
		}
		if got[1].Extras == nil {
			t.Fatalf("got[1].Extras = nil, want unmarshalled map")
		}
		if got[1].Extras["reason"] != "untraded" {
			t.Errorf("got[1].Extras[reason] = %v, want %q", got[1].Extras["reason"], "untraded")
		}
	})

	t.Run("unknown player returns empty slice", func(t *testing.T) {
		got, err := app.GetPlayerRoundAnalysis(strconv.FormatInt(d.ID, 10), "STEAM_NONE")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(got) != 0 {
			t.Errorf("expected empty slice, got %d entries", len(got))
		}
	})
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

	err := q.CreateGameEvent(ctx, store.CreateGameEventParams{
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
	err = q.CreateGameEvent(ctx, store.CreateGameEventParams{
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
	if !bytes.Contains(e.ExtraData, []byte(`"headshot":true`)) {
		t.Errorf("ExtraData = %s, want to contain headshot:true", e.ExtraData)
	}

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
// GetEventsByTypes
// ---------------------------------------------------------------------------

func TestGetEventsByTypes(t *testing.T) {
	app, q := newTestApp(t)
	demo := seedDemo(t, q)
	rounds := seedRounds(t, q, demo.ID)
	ctx := context.Background()

	events := []store.CreateGameEventParams{
		{DemoID: demo.ID, RoundID: rounds[0].ID, Tick: 100, EventType: "kill",
			AttackerSteamID: sql.NullString{String: "STEAM_A", Valid: true},
			VictimSteamID:   sql.NullString{String: "STEAM_B", Valid: true},
			Weapon:          sql.NullString{String: "ak47", Valid: true},
			ExtraData:       `{"headshot":true}`},
		{DemoID: demo.ID, RoundID: rounds[0].ID, Tick: 200, EventType: "bomb_plant"},
		{DemoID: demo.ID, RoundID: rounds[0].ID, Tick: 300, EventType: "kill",
			AttackerSteamID: sql.NullString{String: "STEAM_A", Valid: true},
			VictimSteamID:   sql.NullString{String: "STEAM_C", Valid: true},
			Weapon:          sql.NullString{String: "awp", Valid: true},
			ExtraData:       `{"headshot":false}`},
		{DemoID: demo.ID, RoundID: rounds[0].ID, Tick: 400, EventType: "smoke_start"},
	}
	for _, ep := range events {
		if err := q.CreateGameEvent(ctx, ep); err != nil {
			t.Fatalf("CreateGameEvent: %v", err)
		}
	}

	tests := []struct {
		name    string
		demoID  string
		types   []string
		wantLen int
		wantErr bool
	}{
		{"single type", "1", []string{"kill"}, 2, false},
		{"multiple types", "1", []string{"kill", "bomb_plant"}, 3, false},
		{"unmatched type", "1", []string{"round_end"}, 0, false},
		{"empty types returns empty", "1", []string{}, 0, false},
		{"invalid id", "abc", []string{"kill"}, 0, true},
		{"empty demo", "9999", []string{"kill"}, 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := app.GetEventsByTypes(tt.demoID, tt.types)
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

	// Confirm extra_data is decoded for kills (the whole point of the binding).
	kills, err := app.GetEventsByTypes("1", []string{"kill"})
	if err != nil {
		t.Fatalf("GetEventsByTypes(kill): %v", err)
	}
	if len(kills) != 2 {
		t.Fatalf("len(kills) = %d, want 2", len(kills))
	}
	if kills[0].EventType != "kill" {
		t.Errorf("EventType = %q, want kill", kills[0].EventType)
	}
	if kills[0].ExtraData == nil {
		t.Fatal("ExtraData should not be nil for kill event")
	}
	if !bytes.Contains(kills[0].ExtraData, []byte(`"headshot":true`)) {
		t.Errorf("ExtraData = %s, want to contain headshot:true", kills[0].ExtraData)
	}
	// Ordered by tick.
	if kills[0].Tick != 100 {
		t.Errorf("kills[0].Tick = %d, want 100", kills[0].Tick)
	}
	if kills[1].Tick != 300 {
		t.Errorf("kills[1].Tick = %d, want 300", kills[1].Tick)
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
	if entry.HSPercent != 60 {
		t.Errorf("HSPercent = %f, want 60", entry.HSPercent)
	}
	if entry.ADR != 150 {
		t.Errorf("ADR = %f, want 150", entry.ADR)
	}
}

// ---------------------------------------------------------------------------
// Heatmap helpers
// ---------------------------------------------------------------------------

// seedKillEvents creates game events with kills for heatmap testing.
func seedKillEvents(t *testing.T, q *store.Queries, demoID int64, rounds []store.Round) {
	t.Helper()
	ctx := context.Background()

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
		if err := q.CreateGameEvent(ctx, ep); err != nil {
			t.Fatalf("CreateGameEvent: %v", err)
		}
	}
}

// ---------------------------------------------------------------------------
// GetHeatmapData
// ---------------------------------------------------------------------------

func TestGetHeatmapData(t *testing.T) {
	app, q := newTestApp(t)
	demo := seedDemo(t, q)
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
	app, q := newTestApp(t)
	demo := seedDemo(t, q)
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
	app, q := newTestApp(t)
	demo := seedDemo(t, q)
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

	_, err := q.CreatePlayerRound(ctx, store.CreatePlayerRoundParams{
		RoundID: rounds[0].ID, SteamID: "STEAM_A", PlayerName: "Player1",
		TeamSide: "CT", Kills: 2, Deaths: 0, Assists: 0, Damage: 200, HeadshotKills: 1,
	})
	if err != nil {
		t.Fatalf("CreatePlayerRound: %v", err)
	}

	// `headshot` was promoted to a real column in migration 010; it no longer
	// rides inside extra_data and the weapon-stats query reads the column
	// directly.
	events := []store.CreateGameEventParams{
		{DemoID: demo.ID, RoundID: rounds[0].ID, Tick: 100, EventType: "kill",
			AttackerSteamID: sql.NullString{String: "STEAM_A", Valid: true},
			Weapon:          sql.NullString{String: "AK-47", Valid: true},
			X:               100, Y: 200, Z: 10, Headshot: 1, ExtraData: `{}`},
		{DemoID: demo.ID, RoundID: rounds[0].ID, Tick: 200, EventType: "kill",
			AttackerSteamID: sql.NullString{String: "STEAM_A", Valid: true},
			Weapon:          sql.NullString{String: "AK-47", Valid: true},
			X:               100, Y: 200, Z: 10, Headshot: 0, ExtraData: `{}`},
		{DemoID: demo.ID, RoundID: rounds[0].ID, Tick: 300, EventType: "kill",
			AttackerSteamID: sql.NullString{String: "STEAM_A", Valid: true},
			Weapon:          sql.NullString{String: "AWP", Valid: true},
			X:               300, Y: 400, Z: 5, Headshot: 1, ExtraData: `{}`},
	}
	for _, ep := range events {
		if err := q.CreateGameEvent(ctx, ep); err != nil {
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

	stats, _ := app.GetWeaponStats(strconv.FormatInt(demo.ID, 10))
	if stats[0].Weapon != "AK-47" {
		t.Errorf("Weapon[0] = %q, want AK-47", stats[0].Weapon)
	}
	if stats[0].KillCount != 2 {
		t.Errorf("KillCount[0] = %d, want 2", stats[0].KillCount)
	}
	if stats[0].HSCount != 1 {
		t.Errorf("HSCount[0] = %d, want 1", stats[0].HSCount)
	}
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

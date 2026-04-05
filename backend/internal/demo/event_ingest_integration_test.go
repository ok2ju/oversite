//go:build integration

package demo_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"

	"github.com/ok2ju/oversite/backend/internal/demo"
	"github.com/ok2ju/oversite/backend/internal/store"
)

// createEventTestDemo creates a user and demo for FK dependencies, returning cleanup func.
func createEventTestDemo(t *testing.T, q *store.Queries) (uuid.UUID, func()) {
	t.Helper()
	ctx := context.Background()

	user, err := q.CreateUser(ctx, store.CreateUserParams{
		FaceitID: "ingest-test-" + uuid.NewString()[:8],
		Nickname: "IngestTestPlayer",
	})
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}

	d, err := q.CreateDemo(ctx, store.CreateDemoParams{
		UserID:   user.ID,
		FilePath: "/demos/ingest-test.dem",
		FileSize: 1024000,
		Status:   "ready",
	})
	if err != nil {
		t.Fatalf("CreateDemo: %v", err)
	}

	cleanup := func() {
		q.DeleteGameEventsByDemoID(ctx, d.ID)
		q.DeleteRoundsByDemoID(ctx, d.ID)
		q.DeleteDemo(ctx, d.ID)
		q.DeleteUser(ctx, user.ID)
	}

	return d.ID, cleanup
}

// createEventTestRound creates a round linked to the demo.
func createEventTestRound(t *testing.T, q *store.Queries, demoID uuid.UUID, roundNum int16) store.Round {
	t.Helper()
	round, err := q.CreateRound(context.Background(), store.CreateRoundParams{
		DemoID:      demoID,
		RoundNumber: roundNum,
		StartTick:   int32(roundNum-1) * 3200,
		EndTick:     int32(roundNum) * 3200,
		WinnerSide:  "CT",
		WinReason:   "TargetBombed",
		CtScore:     roundNum,
		TScore:      0,
	})
	if err != nil {
		t.Fatalf("CreateRound(%d): %v", roundNum, err)
	}
	return round
}

func TestIngestGameEvents_Integration_KillEvents(t *testing.T) {
	ctx := context.Background()
	q := store.New(testDB)
	demoID, cleanup := createEventTestDemo(t, q)
	defer cleanup()

	round := createEventTestRound(t, q, demoID, 1)
	roundMap := map[int]uuid.UUID{1: round.ID}

	events := []demo.GameEvent{
		{
			Tick: 1000, RoundNumber: 1, Type: "kill",
			AttackerSteamID: "76561198012345678",
			VictimSteamID:   "76561198087654321",
			Weapon:          "ak47",
			X:               -512.5, Y: 1024.3, Z: 64.0,
			ExtraData: map[string]interface{}{
				"headshot":      true,
				"penetrated":    false,
				"through_smoke": false,
			},
		},
		{
			Tick: 1200, RoundNumber: 1, Type: "kill",
			AttackerSteamID: "76561198087654321",
			VictimSteamID:   "76561198012345678",
			Weapon:          "awp",
			X:               200.0, Y: 300.0, Z: 0.0,
			ExtraData: map[string]interface{}{
				"headshot": false,
				"no_scope": true,
			},
		},
	}

	count, err := demo.IngestGameEvents(ctx, q, demoID, events, roundMap)
	if err != nil {
		t.Fatalf("IngestGameEvents: %v", err)
	}
	if count != 2 {
		t.Errorf("count = %d, want 2", count)
	}

	// Query back by type
	stored, err := q.GetGameEventsByType(ctx, store.GetGameEventsByTypeParams{
		DemoID:    demoID,
		EventType: "kill",
	})
	if err != nil {
		t.Fatalf("GetGameEventsByType: %v", err)
	}
	if len(stored) != 2 {
		t.Fatalf("stored events = %d, want 2", len(stored))
	}

	// Verify first kill
	e := stored[0]
	if e.Tick != 1000 {
		t.Errorf("tick = %d, want 1000", e.Tick)
	}
	if !e.AttackerSteamID.Valid || e.AttackerSteamID.String != "76561198012345678" {
		t.Errorf("attacker = %v, want '76561198012345678'", e.AttackerSteamID)
	}
	if !e.Weapon.Valid || e.Weapon.String != "ak47" {
		t.Errorf("weapon = %v, want 'ak47'", e.Weapon)
	}

	// Verify ExtraData JSONB
	if !e.ExtraData.Valid {
		t.Fatal("ExtraData should be valid")
	}
	var extra map[string]interface{}
	if err := json.Unmarshal(e.ExtraData.RawMessage, &extra); err != nil {
		t.Fatalf("unmarshalling: %v", err)
	}
	if extra["headshot"] != true {
		t.Errorf("extra[headshot] = %v, want true", extra["headshot"])
	}
}

func TestIngestGameEvents_Integration_GrenadeEvents(t *testing.T) {
	ctx := context.Background()
	q := store.New(testDB)
	demoID, cleanup := createEventTestDemo(t, q)
	defer cleanup()

	round := createEventTestRound(t, q, demoID, 1)
	roundMap := map[int]uuid.UUID{1: round.ID}

	events := []demo.GameEvent{
		{
			Tick: 800, RoundNumber: 1, Type: "grenade_throw",
			AttackerSteamID: "76561198012345678",
			Weapon:          "flashbang",
			X:               100.0, Y: 200.0, Z: 50.0,
		},
		{
			Tick: 900, RoundNumber: 1, Type: "grenade_detonate",
			AttackerSteamID: "76561198012345678",
			Weapon:          "flashbang",
			X:               150.0, Y: 250.0, Z: 48.0,
		},
	}

	count, err := demo.IngestGameEvents(ctx, q, demoID, events, roundMap)
	if err != nil {
		t.Fatalf("IngestGameEvents: %v", err)
	}
	if count != 2 {
		t.Errorf("count = %d, want 2", count)
	}

	stored, err := q.GetGameEventsByDemoID(ctx, demoID)
	if err != nil {
		t.Fatalf("GetGameEventsByDemoID: %v", err)
	}
	if len(stored) != 2 {
		t.Fatalf("stored = %d, want 2", len(stored))
	}

	// Verify positions
	if !stored[0].X.Valid || stored[0].X.Float64 != 100.0 {
		t.Errorf("X = %v, want 100.0", stored[0].X)
	}
	if !stored[0].Weapon.Valid || stored[0].Weapon.String != "flashbang" {
		t.Errorf("weapon = %v, want 'flashbang'", stored[0].Weapon)
	}
	if stored[1].EventType != "grenade_detonate" {
		t.Errorf("event_type = %q, want 'grenade_detonate'", stored[1].EventType)
	}
}

func TestIngestGameEvents_Integration_BombEvents(t *testing.T) {
	ctx := context.Background()
	q := store.New(testDB)
	demoID, cleanup := createEventTestDemo(t, q)
	defer cleanup()

	round := createEventTestRound(t, q, demoID, 1)
	roundMap := map[int]uuid.UUID{1: round.ID}

	events := []demo.GameEvent{
		{
			Tick: 1500, RoundNumber: 1, Type: "bomb_plant",
			AttackerSteamID: "76561198012345678",
			X:               300.0, Y: 400.0, Z: 0.0,
			ExtraData: map[string]interface{}{"site": "A"},
		},
		{
			Tick: 2500, RoundNumber: 1, Type: "bomb_explode",
			X: 300.0, Y: 400.0, Z: 0.0,
			ExtraData: map[string]interface{}{"site": "A"},
		},
	}

	count, err := demo.IngestGameEvents(ctx, q, demoID, events, roundMap)
	if err != nil {
		t.Fatalf("IngestGameEvents: %v", err)
	}
	if count != 2 {
		t.Errorf("count = %d, want 2", count)
	}

	stored, err := q.GetGameEventsByType(ctx, store.GetGameEventsByTypeParams{
		DemoID:    demoID,
		EventType: "bomb_plant",
	})
	if err != nil {
		t.Fatalf("GetGameEventsByType: %v", err)
	}
	if len(stored) != 1 {
		t.Fatalf("stored = %d, want 1", len(stored))
	}

	var extra map[string]interface{}
	if err := json.Unmarshal(stored[0].ExtraData.RawMessage, &extra); err != nil {
		t.Fatalf("unmarshalling: %v", err)
	}
	if extra["site"] != "A" {
		t.Errorf("extra[site] = %v, want 'A'", extra["site"])
	}
}

func TestIngestGameEvents_Integration_RoundLinkage(t *testing.T) {
	ctx := context.Background()
	q := store.New(testDB)
	demoID, cleanup := createEventTestDemo(t, q)
	defer cleanup()

	r1 := createEventTestRound(t, q, demoID, 1)
	r2 := createEventTestRound(t, q, demoID, 2)
	r3 := createEventTestRound(t, q, demoID, 3)
	roundMap := map[int]uuid.UUID{1: r1.ID, 2: r2.ID, 3: r3.ID}

	events := []demo.GameEvent{
		{Tick: 500, RoundNumber: 1, Type: "kill", AttackerSteamID: "s1", VictimSteamID: "s2", Weapon: "ak47", X: 1, Y: 2, Z: 3},
		{Tick: 600, RoundNumber: 1, Type: "kill", AttackerSteamID: "s2", VictimSteamID: "s1", Weapon: "m4a1", X: 4, Y: 5, Z: 6},
		{Tick: 3500, RoundNumber: 2, Type: "kill", AttackerSteamID: "s1", VictimSteamID: "s2", Weapon: "deagle", X: 7, Y: 8, Z: 9},
		{Tick: 7000, RoundNumber: 3, Type: "bomb_plant", AttackerSteamID: "s1", X: 10, Y: 11, Z: 12, ExtraData: map[string]interface{}{"site": "B"}},
	}

	count, err := demo.IngestGameEvents(ctx, q, demoID, events, roundMap)
	if err != nil {
		t.Fatalf("IngestGameEvents: %v", err)
	}
	if count != 4 {
		t.Errorf("count = %d, want 4", count)
	}

	// Query round 1 events
	r1Events, err := q.GetGameEventsByDemoAndRound(ctx, store.GetGameEventsByDemoAndRoundParams{
		DemoID:  demoID,
		RoundID: uuid.NullUUID{UUID: r1.ID, Valid: true},
	})
	if err != nil {
		t.Fatalf("GetGameEventsByDemoAndRound(r1): %v", err)
	}
	if len(r1Events) != 2 {
		t.Errorf("round 1 events = %d, want 2", len(r1Events))
	}

	// Query round 2 events
	r2Events, err := q.GetGameEventsByDemoAndRound(ctx, store.GetGameEventsByDemoAndRoundParams{
		DemoID:  demoID,
		RoundID: uuid.NullUUID{UUID: r2.ID, Valid: true},
	})
	if err != nil {
		t.Fatalf("GetGameEventsByDemoAndRound(r2): %v", err)
	}
	if len(r2Events) != 1 {
		t.Errorf("round 2 events = %d, want 1", len(r2Events))
	}

	// Query round 3 events
	r3Events, err := q.GetGameEventsByDemoAndRound(ctx, store.GetGameEventsByDemoAndRoundParams{
		DemoID:  demoID,
		RoundID: uuid.NullUUID{UUID: r3.ID, Valid: true},
	})
	if err != nil {
		t.Fatalf("GetGameEventsByDemoAndRound(r3): %v", err)
	}
	if len(r3Events) != 1 {
		t.Errorf("round 3 events = %d, want 1", len(r3Events))
	}
}

func TestIngestGameEvents_Integration_MixedEventTypes(t *testing.T) {
	ctx := context.Background()
	q := store.New(testDB)
	demoID, cleanup := createEventTestDemo(t, q)
	defer cleanup()

	round := createEventTestRound(t, q, demoID, 1)
	roundMap := map[int]uuid.UUID{1: round.ID}

	events := []demo.GameEvent{
		{Tick: 100, RoundNumber: 1, Type: "grenade_throw", AttackerSteamID: "s1", Weapon: "flashbang", X: 1, Y: 2, Z: 3},
		{Tick: 200, RoundNumber: 1, Type: "grenade_detonate", AttackerSteamID: "s1", Weapon: "flashbang", X: 4, Y: 5, Z: 6},
		{Tick: 500, RoundNumber: 1, Type: "kill", AttackerSteamID: "s1", VictimSteamID: "s2", Weapon: "ak47", X: 7, Y: 8, Z: 9, ExtraData: map[string]interface{}{"headshot": true}},
		{Tick: 1000, RoundNumber: 1, Type: "bomb_plant", AttackerSteamID: "s1", X: 10, Y: 11, Z: 12, ExtraData: map[string]interface{}{"site": "B"}},
		{Tick: 1500, RoundNumber: 1, Type: "bomb_explode", X: 10, Y: 11, Z: 12, ExtraData: map[string]interface{}{"site": "B"}},
	}

	count, err := demo.IngestGameEvents(ctx, q, demoID, events, roundMap)
	if err != nil {
		t.Fatalf("IngestGameEvents: %v", err)
	}
	if count != 5 {
		t.Errorf("count = %d, want 5", count)
	}

	// Verify total and tick ordering
	stored, err := q.GetGameEventsByDemoID(ctx, demoID)
	if err != nil {
		t.Fatalf("GetGameEventsByDemoID: %v", err)
	}
	if len(stored) != 5 {
		t.Fatalf("stored = %d, want 5", len(stored))
	}

	// Verify ascending tick order
	for i := 1; i < len(stored); i++ {
		if stored[i].Tick < stored[i-1].Tick {
			t.Errorf("events not ordered by tick: %d < %d at index %d", stored[i].Tick, stored[i-1].Tick, i)
		}
	}
}

func TestIngestGameEvents_Integration_NullRoundID(t *testing.T) {
	ctx := context.Background()
	q := store.New(testDB)
	demoID, cleanup := createEventTestDemo(t, q)
	defer cleanup()

	// No rounds created, empty roundMap
	events := []demo.GameEvent{
		{Tick: 100, RoundNumber: 0, Type: "kill", AttackerSteamID: "s1", VictimSteamID: "s2", Weapon: "ak47", X: 1, Y: 2, Z: 3},
		{Tick: 200, RoundNumber: 99, Type: "kill", AttackerSteamID: "s2", VictimSteamID: "s1", Weapon: "m4a1", X: 4, Y: 5, Z: 6},
	}

	count, err := demo.IngestGameEvents(ctx, q, demoID, events, nil)
	if err != nil {
		t.Fatalf("IngestGameEvents: %v", err)
	}
	if count != 2 {
		t.Errorf("count = %d, want 2", count)
	}

	stored, err := q.GetGameEventsByDemoID(ctx, demoID)
	if err != nil {
		t.Fatalf("GetGameEventsByDemoID: %v", err)
	}
	if len(stored) != 2 {
		t.Fatalf("stored = %d, want 2", len(stored))
	}

	for i, e := range stored {
		if e.RoundID.Valid {
			t.Errorf("stored[%d].RoundID should be NULL, got %v", i, e.RoundID)
		}
	}
}

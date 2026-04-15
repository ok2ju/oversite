package demo_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/ok2ju/oversite/internal/demo"
	"github.com/ok2ju/oversite/internal/store"
	"github.com/ok2ju/oversite/internal/testutil"
)

// createDemoForEvents creates a demo in the test database for FK constraints.
func createDemoForEvents(t *testing.T, q *store.Queries, userID int64) store.Demo {
	t.Helper()
	d, err := q.CreateDemo(context.Background(), store.CreateDemoParams{
		UserID:   userID,
		MapName:  "de_dust2",
		FilePath: "/demos/test.dem",
		FileSize: 100_000,
		Status:   "imported",
	})
	if err != nil {
		t.Fatalf("CreateDemo: %v", err)
	}
	return d
}

func TestIngestGameEvents_Basic(t *testing.T) {
	q, db := testutil.NewTestQueries(t)
	ctx := context.Background()

	user := createTestUser(t, q)
	d := createDemoForEvents(t, q, user.ID)

	// Create a round so we can map round numbers to DB IDs.
	round, err := q.CreateRound(ctx, store.CreateRoundParams{
		DemoID:      d.ID,
		RoundNumber: 1,
		StartTick:   0,
		EndTick:     1000,
		WinnerSide:  "CT",
		WinReason:   "ct_win",
		CtScore:     1,
		TScore:      0,
	})
	if err != nil {
		t.Fatalf("CreateRound: %v", err)
	}

	roundMap := map[int]int64{1: round.ID}

	events := []demo.GameEvent{
		{
			Tick:            100,
			RoundNumber:     1,
			Type:            "kill",
			AttackerSteamID: "76561198000000001",
			VictimSteamID:   "76561198000000002",
			Weapon:          "ak47",
			X:               1.0,
			Y:               2.0,
			Z:               3.0,
			ExtraData:       map[string]interface{}{"headshot": true, "weapon": "ak47"},
		},
		{
			Tick:            200,
			RoundNumber:     1,
			Type:            "grenade_throw",
			AttackerSteamID: "76561198000000001",
			Weapon:          "flashbang",
			X:               10.0,
			Y:               20.0,
			Z:               30.0,
			ExtraData:       map[string]interface{}{},
		},
		{
			Tick:            300,
			RoundNumber:     1,
			Type:            "bomb_plant",
			AttackerSteamID: "76561198000000003",
			X:               50.0,
			Y:               60.0,
			Z:               70.0,
			ExtraData:       map[string]interface{}{"site": "A"},
		},
	}

	count, err := demo.IngestGameEvents(ctx, db, d.ID, events, roundMap)
	if err != nil {
		t.Fatalf("IngestGameEvents: %v", err)
	}
	if count != 3 {
		t.Errorf("count = %d, want 3", count)
	}

	// Verify stored events.
	stored, err := q.GetGameEventsByDemoID(ctx, d.ID)
	if err != nil {
		t.Fatalf("GetGameEventsByDemoID: %v", err)
	}
	if len(stored) != 3 {
		t.Fatalf("stored events = %d, want 3", len(stored))
	}

	// Check first event (kill).
	e := stored[0]
	if e.DemoID != d.ID {
		t.Errorf("event[0] DemoID = %d, want %d", e.DemoID, d.ID)
	}
	if e.RoundID != round.ID {
		t.Errorf("event[0] RoundID = %d, want %d", e.RoundID, round.ID)
	}
	if e.Tick != 100 {
		t.Errorf("event[0] Tick = %d, want 100", e.Tick)
	}
	if e.EventType != "kill" {
		t.Errorf("event[0] EventType = %q, want %q", e.EventType, "kill")
	}
	if !e.AttackerSteamID.Valid || e.AttackerSteamID.String != "76561198000000001" {
		t.Errorf("event[0] AttackerSteamID = %v, want 76561198000000001", e.AttackerSteamID)
	}
	if !e.VictimSteamID.Valid || e.VictimSteamID.String != "76561198000000002" {
		t.Errorf("event[0] VictimSteamID = %v, want 76561198000000002", e.VictimSteamID)
	}
	if !e.Weapon.Valid || e.Weapon.String != "ak47" {
		t.Errorf("event[0] Weapon = %v, want ak47", e.Weapon)
	}
	if e.X != 1.0 || e.Y != 2.0 || e.Z != 3.0 {
		t.Errorf("event[0] X/Y/Z = %f/%f/%f, want 1/2/3", e.X, e.Y, e.Z)
	}

	// Check second event (grenade_throw).
	if stored[1].EventType != "grenade_throw" {
		t.Errorf("event[1] EventType = %q, want %q", stored[1].EventType, "grenade_throw")
	}

	// Check third event (bomb_plant).
	if stored[2].EventType != "bomb_plant" {
		t.Errorf("event[2] EventType = %q, want %q", stored[2].EventType, "bomb_plant")
	}
}

func TestIngestGameEvents_Idempotent(t *testing.T) {
	q, db := testutil.NewTestQueries(t)
	ctx := context.Background()

	user := createTestUser(t, q)
	d := createDemoForEvents(t, q, user.ID)

	round, err := q.CreateRound(ctx, store.CreateRoundParams{
		DemoID:      d.ID,
		RoundNumber: 1,
		StartTick:   0,
		EndTick:     1000,
		WinnerSide:  "CT",
		WinReason:   "ct_win",
		CtScore:     1,
		TScore:      0,
	})
	if err != nil {
		t.Fatalf("CreateRound: %v", err)
	}

	roundMap := map[int]int64{1: round.ID}

	events := []demo.GameEvent{
		{
			Tick:        100,
			RoundNumber: 1,
			Type:        "kill",
			Weapon:      "ak47",
		},
		{
			Tick:        200,
			RoundNumber: 1,
			Type:        "bomb_plant",
		},
	}

	// First ingestion.
	count1, err := demo.IngestGameEvents(ctx, db, d.ID, events, roundMap)
	if err != nil {
		t.Fatalf("IngestGameEvents (first): %v", err)
	}
	if count1 != 2 {
		t.Errorf("first count = %d, want 2", count1)
	}

	// Second ingestion (idempotent).
	count2, err := demo.IngestGameEvents(ctx, db, d.ID, events, roundMap)
	if err != nil {
		t.Fatalf("IngestGameEvents (second): %v", err)
	}
	if count2 != 2 {
		t.Errorf("second count = %d, want 2", count2)
	}

	// Verify only 2 events in DB (not 4).
	stored, err := q.GetGameEventsByDemoID(ctx, d.ID)
	if err != nil {
		t.Fatalf("GetGameEventsByDemoID: %v", err)
	}
	if len(stored) != 2 {
		t.Errorf("stored events = %d, want 2 (idempotent)", len(stored))
	}
}

func TestIngestGameEvents_EmptyEvents(t *testing.T) {
	_, db := testutil.NewTestQueries(t)
	ctx := context.Background()

	count, err := demo.IngestGameEvents(ctx, db, 1, nil, nil)
	if err != nil {
		t.Fatalf("IngestGameEvents: %v", err)
	}
	if count != 0 {
		t.Errorf("count = %d, want 0", count)
	}
}

func TestIngestGameEvents_ExtraData(t *testing.T) {
	q, db := testutil.NewTestQueries(t)
	ctx := context.Background()

	user := createTestUser(t, q)
	d := createDemoForEvents(t, q, user.ID)

	round, err := q.CreateRound(ctx, store.CreateRoundParams{
		DemoID:      d.ID,
		RoundNumber: 1,
		StartTick:   0,
		EndTick:     1000,
		WinnerSide:  "CT",
		WinReason:   "ct_win",
		CtScore:     1,
		TScore:      0,
	})
	if err != nil {
		t.Fatalf("CreateRound: %v", err)
	}

	roundMap := map[int]int64{1: round.ID}

	extra := map[string]interface{}{"headshot": true, "weapon": "ak47"}
	events := []demo.GameEvent{
		{
			Tick:        100,
			RoundNumber: 1,
			Type:        "kill",
			ExtraData:   extra,
		},
	}

	_, err = demo.IngestGameEvents(ctx, db, d.ID, events, roundMap)
	if err != nil {
		t.Fatalf("IngestGameEvents: %v", err)
	}

	stored, err := q.GetGameEventsByDemoID(ctx, d.ID)
	if err != nil {
		t.Fatalf("GetGameEventsByDemoID: %v", err)
	}
	if len(stored) != 1 {
		t.Fatalf("stored events = %d, want 1", len(stored))
	}

	// Round-trip: unmarshal the stored JSON and check values.
	var decoded map[string]interface{}
	if err := json.Unmarshal([]byte(stored[0].ExtraData), &decoded); err != nil {
		t.Fatalf("json.Unmarshal ExtraData: %v", err)
	}

	headshot, ok := decoded["headshot"]
	if !ok {
		t.Fatal("missing key 'headshot' in ExtraData")
	}
	if headshot != true {
		t.Errorf("headshot = %v, want true", headshot)
	}

	weapon, ok := decoded["weapon"]
	if !ok {
		t.Fatal("missing key 'weapon' in ExtraData")
	}
	if weapon != "ak47" {
		t.Errorf("weapon = %v, want %q", weapon, "ak47")
	}
}

func TestIngestGameEvents_NullableFields(t *testing.T) {
	q, db := testutil.NewTestQueries(t)
	ctx := context.Background()

	user := createTestUser(t, q)
	d := createDemoForEvents(t, q, user.ID)

	round, err := q.CreateRound(ctx, store.CreateRoundParams{
		DemoID:      d.ID,
		RoundNumber: 1,
		StartTick:   0,
		EndTick:     1000,
		WinnerSide:  "CT",
		WinReason:   "ct_win",
		CtScore:     1,
		TScore:      0,
	})
	if err != nil {
		t.Fatalf("CreateRound: %v", err)
	}

	roundMap := map[int]int64{1: round.ID}

	// Event with empty attacker/victim/weapon — should become NULL in DB.
	events := []demo.GameEvent{
		{
			Tick:            100,
			RoundNumber:     1,
			Type:            "bomb_explode",
			AttackerSteamID: "",
			VictimSteamID:   "",
			Weapon:          "",
			X:               10.0,
			Y:               20.0,
			Z:               30.0,
		},
	}

	_, err = demo.IngestGameEvents(ctx, db, d.ID, events, roundMap)
	if err != nil {
		t.Fatalf("IngestGameEvents: %v", err)
	}

	stored, err := q.GetGameEventsByDemoID(ctx, d.ID)
	if err != nil {
		t.Fatalf("GetGameEventsByDemoID: %v", err)
	}
	if len(stored) != 1 {
		t.Fatalf("stored events = %d, want 1", len(stored))
	}

	e := stored[0]
	if e.AttackerSteamID.Valid {
		t.Errorf("AttackerSteamID.Valid = true, want false (NULL)")
	}
	if e.VictimSteamID.Valid {
		t.Errorf("VictimSteamID.Valid = true, want false (NULL)")
	}
	if e.Weapon.Valid {
		t.Errorf("Weapon.Valid = true, want false (NULL)")
	}
}

func TestResolveRoundID(t *testing.T) {
	roundMap := map[int]int64{1: 100, 2: 200, 3: 300}

	tests := []struct {
		name        string
		roundNumber int
		roundMap    map[int]int64
		want        int64
	}{
		{
			name:        "found returns ID",
			roundNumber: 2,
			roundMap:    roundMap,
			want:        200,
		},
		{
			name:        "not found returns 0",
			roundNumber: 99,
			roundMap:    roundMap,
			want:        0,
		},
		{
			name:        "nil map returns 0",
			roundNumber: 1,
			roundMap:    nil,
			want:        0,
		},
		{
			name:        "zero round number returns 0",
			roundNumber: 0,
			roundMap:    roundMap,
			want:        0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := demo.ResolveRoundIDForTest(tt.roundNumber, tt.roundMap)
			if got != tt.want {
				t.Errorf("resolveRoundID(%d) = %d, want %d", tt.roundNumber, got, tt.want)
			}
		})
	}
}

func TestMarshalExtraData(t *testing.T) {
	tests := []struct {
		name    string
		extra   map[string]interface{}
		want    string
		wantErr bool
	}{
		{
			name:  "nil map returns empty JSON object",
			extra: nil,
			want:  "{}",
		},
		{
			name:  "empty map returns empty JSON object",
			extra: map[string]interface{}{},
			want:  "{}",
		},
		{
			name:  "non-empty map returns valid JSON",
			extra: map[string]interface{}{"key": "value"},
			want:  `{"key":"value"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := demo.MarshalExtraDataForTest(tt.extra)
			if (err != nil) != tt.wantErr {
				t.Fatalf("marshalExtraData() error = %v, wantErr %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("marshalExtraData() = %q, want %q", got, tt.want)
			}
		})
	}
}

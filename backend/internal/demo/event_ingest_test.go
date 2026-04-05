package demo

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/ok2ju/oversite/backend/internal/store"
)

// mockGameEventCreator records calls to CreateGameEvent for verification.
type mockGameEventCreator struct {
	calls     []store.CreateGameEventParams
	err       error // if non-nil, returned on every CreateGameEvent call
	deleteErr error // if non-nil, returned on DeleteGameEventsByDemoID
	deleted   bool  // tracks whether delete was called
}

func (m *mockGameEventCreator) CreateGameEvent(_ context.Context, arg store.CreateGameEventParams) (store.GameEvent, error) {
	m.calls = append(m.calls, arg)
	if m.err != nil {
		return store.GameEvent{}, m.err
	}
	return store.GameEvent{ID: uuid.New()}, nil
}

func (m *mockGameEventCreator) DeleteGameEventsByDemoID(_ context.Context, _ uuid.UUID) error {
	m.deleted = true
	return m.deleteErr
}

// --- toCreateGameEventParams tests ---

func TestToCreateGameEventParams_KillEvent(t *testing.T) {
	t.Parallel()

	demoID := uuid.New()
	roundID := uuid.New()
	roundMap := map[int]uuid.UUID{3: roundID}

	evt := GameEvent{
		Tick:            1500,
		RoundNumber:     3,
		Type:            "kill",
		AttackerSteamID: "76561198012345678",
		VictimSteamID:   "76561198087654321",
		Weapon:          "ak47",
		X:               -512.5,
		Y:               1024.3,
		Z:               64.0,
		HasPosition:     true,
		ExtraData: map[string]interface{}{
			"headshot":       true,
			"penetrated":     false,
			"flash_assist":   false,
			"through_smoke":  true,
			"no_scope":       false,
			"attacker_blind": false,
			"wallbang":       true,
		},
	}

	params, err := toCreateGameEventParams(demoID, evt, roundMap)
	if err != nil {
		t.Fatalf("toCreateGameEventParams: %v", err)
	}

	if params.DemoID != demoID {
		t.Errorf("DemoID = %v, want %v", params.DemoID, demoID)
	}
	if !params.RoundID.Valid || params.RoundID.UUID != roundID {
		t.Errorf("RoundID = %v, want valid %v", params.RoundID, roundID)
	}
	if params.Tick != 1500 {
		t.Errorf("Tick = %d, want 1500", params.Tick)
	}
	if params.EventType != "kill" {
		t.Errorf("EventType = %q, want %q", params.EventType, "kill")
	}
	if !params.AttackerSteamID.Valid || params.AttackerSteamID.String != "76561198012345678" {
		t.Errorf("AttackerSteamID = %v, want valid '76561198012345678'", params.AttackerSteamID)
	}
	if !params.VictimSteamID.Valid || params.VictimSteamID.String != "76561198087654321" {
		t.Errorf("VictimSteamID = %v, want valid '76561198087654321'", params.VictimSteamID)
	}
	if !params.Weapon.Valid || params.Weapon.String != "ak47" {
		t.Errorf("Weapon = %v, want valid 'ak47'", params.Weapon)
	}
	if !params.X.Valid || params.X.Float64 != -512.5 {
		t.Errorf("X = %v, want valid -512.5", params.X)
	}
	if !params.Y.Valid || params.Y.Float64 != 1024.3 {
		t.Errorf("Y = %v, want valid 1024.3", params.Y)
	}
	if !params.Z.Valid || params.Z.Float64 != 64.0 {
		t.Errorf("Z = %v, want valid 64.0", params.Z)
	}

	// Verify ExtraData contains expected keys
	if !params.ExtraData.Valid {
		t.Fatal("ExtraData should be valid for kill event")
	}
	var extra map[string]interface{}
	if err := json.Unmarshal(params.ExtraData.RawMessage, &extra); err != nil {
		t.Fatalf("unmarshalling ExtraData: %v", err)
	}
	if extra["headshot"] != true {
		t.Errorf("extra[headshot] = %v, want true", extra["headshot"])
	}
	if extra["through_smoke"] != true {
		t.Errorf("extra[through_smoke] = %v, want true", extra["through_smoke"])
	}
	if extra["wallbang"] != true {
		t.Errorf("extra[wallbang] = %v, want true", extra["wallbang"])
	}
}

func TestToCreateGameEventParams_GrenadeEvents(t *testing.T) {
	t.Parallel()

	demoID := uuid.New()
	roundID := uuid.New()
	roundMap := map[int]uuid.UUID{1: roundID}

	tests := []struct {
		name          string
		eventType     string
		weapon        string
		hasVictim     bool
		attackerSteam string
		x, y, z       float64
	}{
		{"grenade_throw", "grenade_throw", "flashbang", false, "76561198012345678", 100.0, 200.0, 50.0},
		{"grenade_detonate", "grenade_detonate", "hegrenade", false, "76561198012345678", 150.0, 250.0, 48.0},
		{"smoke_start", "smoke_start", "smokegrenade", false, "76561198012345678", 300.0, 400.0, 60.0},
		{"smoke_expired", "smoke_expired", "smokegrenade", false, "76561198012345678", 300.0, 400.0, 60.0},
		{"decoy_start", "decoy_start", "decoy", false, "76561198012345678", 500.0, 600.0, 55.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			evt := GameEvent{
				Tick:            2000,
				RoundNumber:     1,
				Type:            tt.eventType,
				AttackerSteamID: tt.attackerSteam,
				Weapon:          tt.weapon,
				X:               tt.x,
				Y:               tt.y,
				Z:               tt.z,
				HasPosition:     true,
			}

			params, err := toCreateGameEventParams(demoID, evt, roundMap)
			if err != nil {
				t.Fatalf("toCreateGameEventParams: %v", err)
			}

			if params.EventType != tt.eventType {
				t.Errorf("EventType = %q, want %q", params.EventType, tt.eventType)
			}
			if !params.AttackerSteamID.Valid || params.AttackerSteamID.String != tt.attackerSteam {
				t.Errorf("AttackerSteamID = %v, want valid %q", params.AttackerSteamID, tt.attackerSteam)
			}
			if params.VictimSteamID.Valid {
				t.Errorf("VictimSteamID should be NULL for %s", tt.eventType)
			}
			if !params.Weapon.Valid || params.Weapon.String != tt.weapon {
				t.Errorf("Weapon = %v, want valid %q", params.Weapon, tt.weapon)
			}
			if !params.X.Valid || params.X.Float64 != tt.x {
				t.Errorf("X = %v, want valid %v", params.X, tt.x)
			}
			if !params.Y.Valid || params.Y.Float64 != tt.y {
				t.Errorf("Y = %v, want valid %v", params.Y, tt.y)
			}
			if !params.Z.Valid || params.Z.Float64 != tt.z {
				t.Errorf("Z = %v, want valid %v", params.Z, tt.z)
			}
			if !params.RoundID.Valid || params.RoundID.UUID != roundID {
				t.Errorf("RoundID = %v, want valid %v", params.RoundID, roundID)
			}
		})
	}
}

func TestToCreateGameEventParams_BombEvents(t *testing.T) {
	t.Parallel()

	demoID := uuid.New()
	roundID := uuid.New()
	roundMap := map[int]uuid.UUID{5: roundID}

	tests := []struct {
		name            string
		eventType       string
		attackerSteamID string
		site            string
	}{
		{"bomb_plant", "bomb_plant", "76561198012345678", "A"},
		{"bomb_defuse", "bomb_defuse", "76561198087654321", "B"},
		{"bomb_explode", "bomb_explode", "", "A"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			evt := GameEvent{
				Tick:            3000,
				RoundNumber:     5,
				Type:            tt.eventType,
				AttackerSteamID: tt.attackerSteamID,
				X:               100.0,
				Y:               200.0,
				Z:               0.0,
				HasPosition:     true,
				ExtraData: map[string]interface{}{
					"site": tt.site,
				},
			}

			params, err := toCreateGameEventParams(demoID, evt, roundMap)
			if err != nil {
				t.Fatalf("toCreateGameEventParams: %v", err)
			}

			if params.EventType != tt.eventType {
				t.Errorf("EventType = %q, want %q", params.EventType, tt.eventType)
			}

			if tt.attackerSteamID == "" {
				if params.AttackerSteamID.Valid {
					t.Errorf("AttackerSteamID should be NULL for %s", tt.eventType)
				}
			} else {
				if !params.AttackerSteamID.Valid || params.AttackerSteamID.String != tt.attackerSteamID {
					t.Errorf("AttackerSteamID = %v, want valid %q", params.AttackerSteamID, tt.attackerSteamID)
				}
			}

			// Verify ExtraData has site
			if !params.ExtraData.Valid {
				t.Fatal("ExtraData should be valid for bomb event")
			}
			var extra map[string]interface{}
			if err := json.Unmarshal(params.ExtraData.RawMessage, &extra); err != nil {
				t.Fatalf("unmarshalling ExtraData: %v", err)
			}
			if extra["site"] != tt.site {
				t.Errorf("extra[site] = %v, want %q", extra["site"], tt.site)
			}
		})
	}
}

func TestToCreateGameEventParams_RoundIDResolution(t *testing.T) {
	t.Parallel()

	demoID := uuid.New()
	roundID := uuid.New()
	roundMap := map[int]uuid.UUID{1: roundID}

	tests := []struct {
		name        string
		roundNumber int
		wantValid   bool
		wantUUID    uuid.UUID
	}{
		{"round in map", 1, true, roundID},
		{"round zero", 0, false, uuid.Nil},
		{"round not in map", 99, false, uuid.Nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			evt := GameEvent{
				Tick:        1000,
				RoundNumber: tt.roundNumber,
				Type:        "kill",
			}

			params, err := toCreateGameEventParams(demoID, evt, roundMap)
			if err != nil {
				t.Fatalf("toCreateGameEventParams: %v", err)
			}

			if params.RoundID.Valid != tt.wantValid {
				t.Errorf("RoundID.Valid = %v, want %v", params.RoundID.Valid, tt.wantValid)
			}
			if tt.wantValid && params.RoundID.UUID != tt.wantUUID {
				t.Errorf("RoundID.UUID = %v, want %v", params.RoundID.UUID, tt.wantUUID)
			}
		})
	}
}

func TestToCreateGameEventParams_EmptyOptionalFields(t *testing.T) {
	t.Parallel()

	demoID := uuid.New()

	evt := GameEvent{
		Tick:            500,
		Type:            "kill",
		AttackerSteamID: "",
		VictimSteamID:   "",
		Weapon:          "",
		X:               10.0,
		Y:               20.0,
		Z:               30.0,
		HasPosition:     true,
	}

	params, err := toCreateGameEventParams(demoID, evt, nil)
	if err != nil {
		t.Fatalf("toCreateGameEventParams: %v", err)
	}

	if params.AttackerSteamID.Valid {
		t.Error("AttackerSteamID should be NULL for empty string")
	}
	if params.VictimSteamID.Valid {
		t.Error("VictimSteamID should be NULL for empty string")
	}
	if params.Weapon.Valid {
		t.Error("Weapon should be NULL for empty string")
	}
}

// --- buildExtraData tests ---

func TestBuildExtraData(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		extra     map[string]interface{}
		wantValid bool
		wantKeys  []string // if valid, check these keys exist
	}{
		{
			name: "kill extra data",
			extra: map[string]interface{}{
				"headshot":      true,
				"penetrated":    false,
				"through_smoke": true,
			},
			wantValid: true,
			wantKeys:  []string{"headshot", "penetrated", "through_smoke"},
		},
		{
			name: "bomb extra data",
			extra: map[string]interface{}{
				"site": "A",
			},
			wantValid: true,
			wantKeys:  []string{"site"},
		},
		{
			name:      "nil map",
			extra:     nil,
			wantValid: false,
		},
		{
			name:      "empty map",
			extra:     map[string]interface{}{},
			wantValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result, err := buildExtraData(tt.extra)
			if err != nil {
				t.Fatalf("buildExtraData: %v", err)
			}

			if result.Valid != tt.wantValid {
				t.Errorf("Valid = %v, want %v", result.Valid, tt.wantValid)
			}

			if tt.wantValid {
				var data map[string]interface{}
				if err := json.Unmarshal(result.RawMessage, &data); err != nil {
					t.Fatalf("unmarshalling result: %v", err)
				}
				for _, key := range tt.wantKeys {
					if _, ok := data[key]; !ok {
						t.Errorf("expected key %q in extra data", key)
					}
				}
			}
		})
	}
}

// --- nullString tests ---

func TestNullString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input     string
		wantValid bool
	}{
		{"ak47", true},
		{"", false},
		{"76561198012345678", true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()
			result := nullString(tt.input)
			if result.Valid != tt.wantValid {
				t.Errorf("nullString(%q).Valid = %v, want %v", tt.input, result.Valid, tt.wantValid)
			}
			if tt.wantValid && result.String != tt.input {
				t.Errorf("nullString(%q).String = %q, want %q", tt.input, result.String, tt.input)
			}
		})
	}
}

// --- IngestGameEvents tests with mock ---

func TestIngestGameEvents_MockCreator(t *testing.T) {
	t.Parallel()

	demoID := uuid.New()
	round1ID := uuid.New()
	round2ID := uuid.New()
	roundMap := map[int]uuid.UUID{1: round1ID, 2: round2ID}

	events := []GameEvent{
		{
			Tick:            1000,
			RoundNumber:     1,
			Type:            "kill",
			AttackerSteamID: "76561198012345678",
			VictimSteamID:   "76561198087654321",
			Weapon:          "ak47",
			X:               100.0, Y: 200.0, Z: 0.0,
			HasPosition: true,
			ExtraData:   map[string]interface{}{"headshot": true},
		},
		{
			Tick:            1500,
			RoundNumber:     1,
			Type:            "grenade_throw",
			AttackerSteamID: "76561198012345678",
			Weapon:          "flashbang",
			X:               150.0, Y: 250.0, Z: 10.0,
			HasPosition: true,
		},
		{
			Tick:            2500,
			RoundNumber:     2,
			Type:            "bomb_plant",
			AttackerSteamID: "76561198087654321",
			X:               300.0, Y: 400.0, Z: 0.0,
			HasPosition: true,
			ExtraData:   map[string]interface{}{"site": "A"},
		},
	}

	mock := &mockGameEventCreator{}
	count, err := IngestGameEvents(context.Background(), mock, demoID, events, roundMap)
	if err != nil {
		t.Fatalf("IngestGameEvents returned error: %v", err)
	}
	if count != 3 {
		t.Errorf("count = %d, want 3", count)
	}
	if len(mock.calls) != 3 {
		t.Fatalf("mock received %d calls, want 3", len(mock.calls))
	}

	// Verify first call (kill event)
	call0 := mock.calls[0]
	if call0.DemoID != demoID {
		t.Errorf("call[0].DemoID = %v, want %v", call0.DemoID, demoID)
	}
	if !call0.RoundID.Valid || call0.RoundID.UUID != round1ID {
		t.Errorf("call[0].RoundID = %v, want valid %v", call0.RoundID, round1ID)
	}
	if call0.EventType != "kill" {
		t.Errorf("call[0].EventType = %q, want %q", call0.EventType, "kill")
	}

	// Verify third call (bomb_plant in round 2)
	call2 := mock.calls[2]
	if !call2.RoundID.Valid || call2.RoundID.UUID != round2ID {
		t.Errorf("call[2].RoundID = %v, want valid %v", call2.RoundID, round2ID)
	}
	if call2.EventType != "bomb_plant" {
		t.Errorf("call[2].EventType = %q, want %q", call2.EventType, "bomb_plant")
	}
}

func TestIngestGameEvents_ErrorPropagation(t *testing.T) {
	t.Parallel()

	demoID := uuid.New()
	dbErr := errors.New("db connection lost")
	mock := &mockGameEventCreator{err: dbErr}

	events := []GameEvent{
		{Tick: 100, Type: "kill", X: 1, Y: 2, Z: 3, HasPosition: true},
		{Tick: 200, Type: "kill", X: 4, Y: 5, Z: 6, HasPosition: true},
	}

	count, err := IngestGameEvents(context.Background(), mock, demoID, events, nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, dbErr) {
		t.Errorf("error = %v, want %v", err, dbErr)
	}
	if count != 0 {
		t.Errorf("count = %d, want 0 on error", count)
	}
	// Should fail fast on first error
	if len(mock.calls) != 1 {
		t.Errorf("mock received %d calls, want 1 (fail fast)", len(mock.calls))
	}
}

func TestIngestGameEvents_EmptyEvents(t *testing.T) {
	t.Parallel()

	mock := &mockGameEventCreator{}
	count, err := IngestGameEvents(context.Background(), mock, uuid.New(), nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 0 {
		t.Errorf("count = %d, want 0", count)
	}
	if len(mock.calls) != 0 {
		t.Errorf("mock received %d calls, want 0", len(mock.calls))
	}
}

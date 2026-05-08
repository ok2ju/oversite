package demo

import "testing"

func TestPairShotsWithImpacts(t *testing.T) {
	type wantHit struct {
		x, y float64
	}

	tests := []struct {
		name   string
		events []GameEvent
		// wantHits maps event index → expected hit_x/hit_y. Indices not in the
		// map must NOT have hit_x/hit_y populated.
		wantHits map[int]wantHit
	}{
		{
			name: "single fire then hurt within window",
			events: []GameEvent{
				{Tick: 100, Type: "weapon_fire", AttackerSteamID: "A", ExtraData: &WeaponFireExtra{Yaw: 0.0}},
				{Tick: 103, Type: "player_hurt", AttackerSteamID: "A", VictimSteamID: "V", X: 10, Y: 20},
			},
			wantHits: map[int]wantHit{0: {x: 10, y: 20}},
		},
		{
			name: "hurt outside window — not paired",
			events: []GameEvent{
				{Tick: 100, Type: "weapon_fire", AttackerSteamID: "A", ExtraData: &WeaponFireExtra{Yaw: 0.0}},
				{Tick: 100 + shotImpactPairWindow + 1, Type: "player_hurt", AttackerSteamID: "A", VictimSteamID: "V", X: 10, Y: 20},
			},
			wantHits: map[int]wantHit{},
		},
		{
			name: "different attackers do not cross-pair",
			events: []GameEvent{
				{Tick: 100, Type: "weapon_fire", AttackerSteamID: "A", ExtraData: &WeaponFireExtra{Yaw: 0.0}},
				{Tick: 102, Type: "player_hurt", AttackerSteamID: "B", VictimSteamID: "V", X: 5, Y: 5},
			},
			wantHits: map[int]wantHit{},
		},
		{
			name: "spray with all hits — each shot pairs with its hurt",
			events: []GameEvent{
				{Tick: 100, Type: "weapon_fire", AttackerSteamID: "A", ExtraData: &WeaponFireExtra{Yaw: 0.0}},
				{Tick: 102, Type: "player_hurt", AttackerSteamID: "A", VictimSteamID: "V", X: 10, Y: 10},
				{Tick: 106, Type: "weapon_fire", AttackerSteamID: "A", ExtraData: &WeaponFireExtra{Yaw: 0.0}},
				{Tick: 108, Type: "player_hurt", AttackerSteamID: "A", VictimSteamID: "V", X: 11, Y: 11},
			},
			wantHits: map[int]wantHit{
				0: {x: 10, y: 10},
				2: {x: 11, y: 11},
			},
		},
		{
			name: "miss between hits — only the second shot pairs",
			events: []GameEvent{
				{Tick: 100, Type: "weapon_fire", AttackerSteamID: "A", ExtraData: &WeaponFireExtra{Yaw: 0.0}}, // miss
				{Tick: 106, Type: "weapon_fire", AttackerSteamID: "A", ExtraData: &WeaponFireExtra{Yaw: 0.0}}, // hits
				{Tick: 108, Type: "player_hurt", AttackerSteamID: "A", VictimSteamID: "V", X: 7, Y: 8},
			},
			// The first shot's lastShot record gets overwritten when WF #2
			// fires; PH at 108 pairs with WF at 106.
			wantHits: map[int]wantHit{1: {x: 7, y: 8}},
		},
		{
			name: "wallbang — only first hurt is recorded on the shot",
			events: []GameEvent{
				{Tick: 100, Type: "weapon_fire", AttackerSteamID: "A", ExtraData: &WeaponFireExtra{Yaw: 0.0}},
				{Tick: 102, Type: "player_hurt", AttackerSteamID: "A", VictimSteamID: "V1", X: 1, Y: 2},
				{Tick: 102, Type: "player_hurt", AttackerSteamID: "A", VictimSteamID: "V2", X: 3, Y: 4},
			},
			wantHits: map[int]wantHit{0: {x: 1, y: 2}},
		},
		{
			name: "weapon_fire with empty attacker id is ignored",
			events: []GameEvent{
				{Tick: 100, Type: "weapon_fire", AttackerSteamID: "", ExtraData: &WeaponFireExtra{Yaw: 0.0}},
				{Tick: 102, Type: "player_hurt", AttackerSteamID: "", VictimSteamID: "V", X: 5, Y: 5},
			},
			wantHits: map[int]wantHit{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pairShotsWithImpacts(tt.events)

			for i, ev := range tt.events {
				if ev.Type != "weapon_fire" {
					continue
				}
				want, expected := tt.wantHits[i]
				wf, _ := ev.ExtraData.(*WeaponFireExtra)
				var gotX, gotY float64
				hasX := wf != nil && wf.HitX != nil
				hasY := wf != nil && wf.HitY != nil
				if hasX {
					gotX = *wf.HitX
				}
				if hasY {
					gotY = *wf.HitY
				}
				if !expected {
					if hasX || hasY {
						t.Errorf("events[%d] should NOT have hit data, got hit_x=%v hit_y=%v", i, gotX, gotY)
					}
					continue
				}
				if !hasX || !hasY {
					t.Errorf("events[%d] missing hit_x/hit_y, want (%v, %v)", i, want.x, want.y)
					continue
				}
				if gotX != want.x || gotY != want.y {
					t.Errorf("events[%d] hit = (%v, %v), want (%v, %v)", i, gotX, gotY, want.x, want.y)
				}
			}
		})
	}
}

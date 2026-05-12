package contacts_test

import (
	"encoding/json"
	"testing"

	"github.com/ok2ju/oversite/internal/demo"
	"github.com/ok2ju/oversite/internal/demo/contacts"
	"github.com/ok2ju/oversite/internal/testutil"
)

// fixtureEvent is the input shape for a single event row in a scenario
// JSON file. The collector consumes typed *demo.XxxExtra pointers, so
// fixtureExtras is decoded once and toGameEvent picks the right struct
// based on the event Type.
type fixtureEvent struct {
	Tick     int           `json:"tick"`
	Type     string        `json:"type"`
	Attacker string        `json:"attacker"`
	Victim   string        `json:"victim"`
	Weapon   string        `json:"weapon"`
	Extras   fixtureExtras `json:"extras"`
}

type fixtureExtras struct {
	HitVictimSteamID string  `json:"hit_victim_steam_id,omitempty"`
	HealthDamage     int     `json:"health_damage,omitempty"`
	ArmorDamage      int     `json:"armor_damage,omitempty"`
	HitGroup         int     `json:"hit_group,omitempty"`
	Penetrated       int     `json:"penetrated,omitempty"`
	AttackerTeam     string  `json:"attacker_team,omitempty"`
	VictimTeam       string  `json:"victim_team,omitempty"`
	Headshot         bool    `json:"headshot,omitempty"`
	Wallbang         bool    `json:"wallbang,omitempty"`
	DurationSecs     float64 `json:"duration_secs,omitempty"`
}

type fixtureVisibilityRow struct {
	Tick    int    `json:"tick"`
	Spotted string `json:"spotted"`
	Spotter string `json:"spotter"`
	State   int8   `json:"state"`
}

type scenarioFixture struct {
	Name         string                 `json:"name"`
	TickRate     float64                `json:"tick_rate"`
	Round        scenarioRound          `json:"round"`
	Subject      string                 `json:"subject"`
	SubjectAlive contacts.AliveRange    `json:"subject_alive"`
	Events       []fixtureEvent         `json:"events"`
	Visibility   []fixtureVisibilityRow `json:"visibility"`
}

type scenarioRound struct {
	Number        int                        `json:"number"`
	StartTick     int                        `json:"start_tick"`
	FreezeEndTick int                        `json:"freeze_end_tick"`
	EndTick       int                        `json:"end_tick"`
	WinnerSide    string                     `json:"winner_side"`
	WinReason     string                     `json:"win_reason"`
	Roster        []scenarioRoundParticipant `json:"roster"`
}

type scenarioRoundParticipant struct {
	SteamID    string `json:"steam_id"`
	PlayerName string `json:"name"`
	TeamSide   string `json:"team_side"`
	Inventory  string `json:"inventory"`
}

func (s scenarioRound) toRoundData() demo.RoundData {
	roster := make([]demo.RoundParticipant, 0, len(s.Roster))
	for _, p := range s.Roster {
		roster = append(roster, demo.RoundParticipant{
			SteamID:    p.SteamID,
			PlayerName: p.PlayerName,
			TeamSide:   p.TeamSide,
			Inventory:  p.Inventory,
		})
	}
	return demo.RoundData{
		Number:        s.Number,
		StartTick:     s.StartTick,
		FreezeEndTick: s.FreezeEndTick,
		EndTick:       s.EndTick,
		WinnerSide:    s.WinnerSide,
		WinReason:     s.WinReason,
		Roster:        roster,
	}
}

// toGameEvent converts a fixture event into the demo.GameEvent shape the
// collector expects, including a typed *demo.XxxExtra pointer.
func (e fixtureEvent) toGameEvent(roundNumber int) demo.GameEvent {
	evt := demo.GameEvent{
		Tick:            e.Tick,
		RoundNumber:     roundNumber,
		Type:            e.Type,
		AttackerSteamID: e.Attacker,
		VictimSteamID:   e.Victim,
		Weapon:          e.Weapon,
	}
	switch e.Type {
	case "weapon_fire":
		evt.ExtraData = &demo.WeaponFireExtra{
			HitVictimSteamID: e.Extras.HitVictimSteamID,
		}
	case "player_hurt":
		evt.ExtraData = &demo.PlayerHurtExtra{
			HealthDamage: e.Extras.HealthDamage,
			ArmorDamage:  e.Extras.ArmorDamage,
			HitGroup:     e.Extras.HitGroup,
			Penetrated:   e.Extras.Penetrated,
			AttackerTeam: e.Extras.AttackerTeam,
			VictimTeam:   e.Extras.VictimTeam,
		}
	case "kill":
		evt.ExtraData = &demo.KillExtra{
			Headshot:     e.Extras.Headshot,
			Penetrated:   e.Extras.Penetrated,
			Wallbang:     e.Extras.Wallbang,
			AttackerTeam: e.Extras.AttackerTeam,
			VictimTeam:   e.Extras.VictimTeam,
		}
	case "player_flashed":
		evt.ExtraData = &demo.PlayerFlashedExtra{
			DurationSecs: e.Extras.DurationSecs,
			AttackerTeam: e.Extras.AttackerTeam,
			VictimTeam:   e.Extras.VictimTeam,
		}
	}
	return evt
}

func loadScenario(t *testing.T, name string) scenarioFixture {
	t.Helper()
	var fix scenarioFixture
	testutil.LoadFixture(t, "contacts/"+name+".json", &fix)
	return fix
}

func (s scenarioFixture) toParseResult() *demo.ParseResult {
	round := s.Round.toRoundData()
	events := make([]demo.GameEvent, 0, len(s.Events))
	for _, e := range s.Events {
		events = append(events, e.toGameEvent(round.Number))
	}
	vis := make([]demo.VisibilityChange, 0, len(s.Visibility))
	for _, v := range s.Visibility {
		vis = append(vis, demo.VisibilityChange{
			RoundNumber:  round.Number,
			Tick:         v.Tick,
			SpottedSteam: v.Spotted,
			SpotterSteam: v.Spotter,
			State:        v.State,
		})
	}
	return &demo.ParseResult{
		Header:     demo.MatchHeader{TickRate: s.TickRate},
		Rounds:     []demo.RoundData{round},
		Events:     events,
		Visibility: vis,
	}
}

func runScenario(t *testing.T, name string) {
	t.Helper()
	fix := loadScenario(t, name)
	result := fix.toParseResult()
	roundMap := map[int]int64{fix.Round.Number: 1}

	got, err := contacts.Run(result, roundMap, contacts.RunOpts{})
	if err != nil {
		t.Fatalf("contacts.Run: %v", err)
	}

	raw, err := json.MarshalIndent(got, "", "  ")
	if err != nil {
		t.Fatalf("marshal got: %v", err)
	}
	raw = append(raw, '\n')

	testutil.CompareGolden(t, "contacts/"+name+".golden.json", raw)
}

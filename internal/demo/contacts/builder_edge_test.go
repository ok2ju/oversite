package contacts_test

import (
	"testing"

	"github.com/ok2ju/oversite/internal/demo"
	"github.com/ok2ju/oversite/internal/demo/contacts"
)

func runResult(t *testing.T, result *demo.ParseResult, roundNumber int) []contacts.ContactMoment {
	t.Helper()
	got, err := contacts.Run(result, map[int]int64{roundNumber: 1}, contacts.RunOpts{})
	if err != nil {
		t.Fatalf("contacts.Run: %v", err)
	}
	return got
}

func basicRound(number int) demo.RoundData {
	return demo.RoundData{
		Number:        number,
		StartTick:     0,
		FreezeEndTick: 100,
		EndTick:       5000,
		Roster: []demo.RoundParticipant{
			{SteamID: "S_P", PlayerName: "P", TeamSide: "T"},
			{SteamID: "S_T2", PlayerName: "T2", TeamSide: "T"},
			{SteamID: "S_E1", PlayerName: "E1", TeamSide: "CT"},
			{SteamID: "S_E2", PlayerName: "E2", TeamSide: "CT"},
		},
	}
}

func TestBots_FilteredAsSubject(t *testing.T) {
	round := basicRound(1)
	round.Roster = append(round.Roster, demo.RoundParticipant{
		SteamID: "0", PlayerName: "BOT Cooper", TeamSide: "CT",
	})
	result := &demo.ParseResult{
		Rounds: []demo.RoundData{round},
		Events: []demo.GameEvent{
			{Tick: 200, RoundNumber: 1, Type: "kill", AttackerSteamID: "0", VictimSteamID: "S_P",
				ExtraData: &demo.KillExtra{AttackerTeam: "CT", VictimTeam: "T"}},
		},
	}
	got := runResult(t, result, 1)
	for _, c := range got {
		if c.SubjectSteam == "0" {
			t.Errorf("bot subject should be filtered: %+v", c)
		}
	}
}

func TestMultiKillSameTick(t *testing.T) {
	round := basicRound(1)
	result := &demo.ParseResult{
		Rounds: []demo.RoundData{round},
		Events: []demo.GameEvent{
			{Tick: 500, RoundNumber: 1, Type: "kill", AttackerSteamID: "S_P", VictimSteamID: "S_E1",
				ExtraData: &demo.KillExtra{AttackerTeam: "T", VictimTeam: "CT"}},
			{Tick: 500, RoundNumber: 1, Type: "kill", AttackerSteamID: "S_P", VictimSteamID: "S_E2",
				ExtraData: &demo.KillExtra{AttackerTeam: "T", VictimTeam: "CT"}},
		},
	}
	got := runResult(t, result, 1)
	var subjectContacts []contacts.ContactMoment
	for _, c := range got {
		if c.SubjectSteam == "S_P" {
			subjectContacts = append(subjectContacts, c)
		}
	}
	if len(subjectContacts) != 1 {
		t.Fatalf("expected 1 contact, got %d", len(subjectContacts))
	}
	c := subjectContacts[0]
	if c.SignalCount != 2 {
		t.Errorf("SignalCount = %d, want 2", c.SignalCount)
	}
	if len(c.Enemies) != 2 {
		t.Errorf("Enemies = %v, want both E1+E2", c.Enemies)
	}
}

func TestWallbangNoVisibility(t *testing.T) {
	round := basicRound(1)
	result := &demo.ParseResult{
		Rounds: []demo.RoundData{round},
		Events: []demo.GameEvent{
			{Tick: 500, RoundNumber: 1, Type: "player_hurt", AttackerSteamID: "S_E1", VictimSteamID: "S_P", Weapon: "ak47",
				ExtraData: &demo.PlayerHurtExtra{HealthDamage: 30, Penetrated: 1, AttackerTeam: "CT", VictimTeam: "T"}},
		},
	}
	got := runResult(t, result, 1)
	var c *contacts.ContactMoment
	for i := range got {
		if got[i].SubjectSteam == "S_P" {
			c = &got[i]
			break
		}
	}
	if c == nil {
		t.Fatal("expected subject contact for S_P")
	}
	if !c.Extras.WallbangTaken {
		t.Errorf("WallbangTaken = false, want true")
	}
	if c.Outcome != contacts.OutcomeDisengaged {
		t.Errorf("outcome = %s, want disengaged", c.Outcome)
	}
}

func TestFlashOnlyContact(t *testing.T) {
	round := basicRound(1)
	result := &demo.ParseResult{
		Rounds: []demo.RoundData{round},
		Events: []demo.GameEvent{
			{Tick: 500, RoundNumber: 1, Type: "player_flashed", AttackerSteamID: "S_E1", VictimSteamID: "S_P",
				ExtraData: &demo.PlayerFlashedExtra{DurationSecs: 1.2, AttackerTeam: "CT", VictimTeam: "T"}},
		},
	}
	got := runResult(t, result, 1)
	var c *contacts.ContactMoment
	for i := range got {
		if got[i].SubjectSteam == "S_P" {
			c = &got[i]
			break
		}
	}
	if c == nil {
		t.Fatal("expected subject contact for S_P")
	}
	if !c.Extras.FlashOnly {
		t.Errorf("FlashOnly = false, want true")
	}
	if c.Outcome != contacts.OutcomeDisengaged {
		t.Errorf("outcome = %s, want disengaged", c.Outcome)
	}
}

func TestRoundEndDuringContact(t *testing.T) {
	round := basicRound(1)
	round.EndTick = 600
	result := &demo.ParseResult{
		Rounds: []demo.RoundData{round},
		Events: []demo.GameEvent{
			{Tick: 599, RoundNumber: 1, Type: "kill", AttackerSteamID: "S_P", VictimSteamID: "S_E1", Weapon: "ak47",
				ExtraData: &demo.KillExtra{AttackerTeam: "T", VictimTeam: "CT"}},
		},
	}
	got := runResult(t, result, 1)
	var c *contacts.ContactMoment
	for i := range got {
		if got[i].SubjectSteam == "S_P" {
			c = &got[i]
			break
		}
	}
	if c == nil {
		t.Fatal("expected subject contact for S_P")
	}
	if !c.Extras.TruncatedRoundEnd {
		t.Errorf("TruncatedRoundEnd = false, want true")
	}
	if c.TPost > 600 {
		t.Errorf("TPost = %d, should be clamped to round end (600)", c.TPost)
	}
}

func TestFriendlyFireFiltered(t *testing.T) {
	round := basicRound(1)
	result := &demo.ParseResult{
		Rounds: []demo.RoundData{round},
		Events: []demo.GameEvent{
			{Tick: 500, RoundNumber: 1, Type: "player_hurt", AttackerSteamID: "S_T2", VictimSteamID: "S_P", Weapon: "ak47",
				ExtraData: &demo.PlayerHurtExtra{HealthDamage: 80, AttackerTeam: "T", VictimTeam: "T"}},
		},
	}
	got := runResult(t, result, 1)
	for _, c := range got {
		if c.SubjectSteam == "S_P" {
			t.Errorf("friendly fire should not produce subject contact, got %+v", c)
		}
	}
}

func TestTeammateFlashDuringContact(t *testing.T) {
	round := basicRound(1)
	result := &demo.ParseResult{
		Rounds: []demo.RoundData{round},
		Events: []demo.GameEvent{
			// Sub-threshold teammate flash (0.5s) covering the engagement tick.
			{Tick: 500, RoundNumber: 1, Type: "player_flashed", AttackerSteamID: "S_T2", VictimSteamID: "S_P",
				ExtraData: &demo.PlayerFlashedExtra{DurationSecs: 0.5}},
			// Real engagement: P kills E1 a few ticks later.
			{Tick: 510, RoundNumber: 1, Type: "kill", AttackerSteamID: "S_P", VictimSteamID: "S_E1", Weapon: "ak47",
				ExtraData: &demo.KillExtra{AttackerTeam: "T", VictimTeam: "CT"}},
		},
	}
	got := runResult(t, result, 1)
	var c *contacts.ContactMoment
	for i := range got {
		if got[i].SubjectSteam == "S_P" {
			c = &got[i]
			break
		}
	}
	if c == nil {
		t.Fatal("expected subject contact for S_P")
	}
	if !c.Extras.TeammateFlashedDuring {
		t.Errorf("TeammateFlashedDuring = false, want true")
	}
}

func TestSimultaneousDoubleKill(t *testing.T) {
	round := basicRound(1)
	result := &demo.ParseResult{
		Rounds: []demo.RoundData{round},
		Events: []demo.GameEvent{
			{Tick: 500, RoundNumber: 1, Type: "kill", AttackerSteamID: "S_P", VictimSteamID: "S_E1", Weapon: "ak47",
				ExtraData: &demo.KillExtra{AttackerTeam: "T", VictimTeam: "CT"}},
			{Tick: 500, RoundNumber: 1, Type: "kill", AttackerSteamID: "S_P", VictimSteamID: "S_E2", Weapon: "ak47",
				ExtraData: &demo.KillExtra{AttackerTeam: "T", VictimTeam: "CT"}},
		},
	}
	got := runResult(t, result, 1)
	var c *contacts.ContactMoment
	for i := range got {
		if got[i].SubjectSteam == "S_P" {
			c = &got[i]
			break
		}
	}
	if c == nil {
		t.Fatal("expected subject contact for S_P")
	}
	if c.Outcome != contacts.OutcomeWonClean {
		t.Errorf("outcome = %s, want won_clean", c.Outcome)
	}
}

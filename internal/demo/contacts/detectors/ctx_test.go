package detectors

import (
	"testing"

	"github.com/ok2ju/oversite/internal/demo"
	"github.com/ok2ju/oversite/internal/demo/contacts"
)

func TestNewRunData(t *testing.T) {
	result := &demo.ParseResult{
		Header: demo.MatchHeader{TickRate: 64},
		Rounds: []demo.RoundData{{
			Number: 1, FreezeEndTick: 100, EndTick: 1000,
			Roster: []demo.RoundParticipant{
				{SteamID: "S_P", TeamSide: "T"},
				{SteamID: "S_E1", TeamSide: "CT"},
			},
		}},
		Events: []demo.GameEvent{
			{Tick: 200, Type: "weapon_fire", AttackerSteamID: "S_P", RoundNumber: 1},
		},
		AnalysisTicks: []demo.AnalysisTick{
			mkTick(200, 0, 0, 0, 0, 0, 0, 0, 0, true, 30),
		},
	}
	rd := NewRunData(result)
	if rd.EventsByRound[1] == nil {
		t.Fatal("missing round 1 events")
	}
	if rd.RoundsByNumber[1] == nil {
		t.Fatal("missing round 1 metadata")
	}
	if rd.PlayersByRound[1]["S_P"].TeamSide != "T" {
		t.Fatal("missing player")
	}
	if rd.TeamsByRound[1]["S_P"] != "T" {
		t.Fatal("team map drift")
	}
	if _, ok := rd.Weapons.Lookup("ak47"); !ok {
		t.Fatal("weapon catalog empty")
	}
	if rd.TickRate != 64.0 {
		t.Errorf("TickRate: got %v, want 64", rd.TickRate)
	}
}

func TestNewRunData_NilResult(t *testing.T) {
	rd := NewRunData(nil)
	if rd == nil {
		t.Fatal("NewRunData(nil) returned nil")
	}
	if rd.EventsByRound == nil || rd.RoundsByNumber == nil || rd.PlayersByRound == nil || rd.TeamsByRound == nil {
		t.Error("NewRunData(nil) should still produce non-nil maps")
	}
}

func TestBuildCtx_PreviousContactEnd(t *testing.T) {
	result := &demo.ParseResult{
		Header: demo.MatchHeader{TickRate: 64},
		Rounds: []demo.RoundData{{
			Number: 1, FreezeEndTick: 100, EndTick: 1000,
			Roster: []demo.RoundParticipant{
				{SteamID: "S_P", TeamSide: "T"},
			},
		}},
	}
	rd := NewRunData(result)
	c := &contacts.Contact{Subject: "S_P", RoundNumber: 1, TFirst: 500}

	first := BuildCtx(c, -1, rd)
	if first.PreviousContactEnd != -1 {
		t.Errorf("first contact: expected -1, got %d", first.PreviousContactEnd)
	}
	if first.SubjectTeam != "T" {
		t.Errorf("subject team: got %q, want T", first.SubjectTeam)
	}

	second := BuildCtx(c, 400, rd)
	if second.PreviousContactEnd != 400 {
		t.Errorf("subsequent: expected 400, got %d", second.PreviousContactEnd)
	}
}

func TestClampPreLookback(t *testing.T) {
	c := &contacts.Contact{TPre: 9440}
	ctx := &DetectorCtx{PreviousContactEnd: -1}
	if got := ClampPreLookback(c, ctx); got != 9440 {
		t.Errorf("no previous: got %d, want 9440", got)
	}

	ctx.PreviousContactEnd = 9500
	if got := ClampPreLookback(c, ctx); got != 9501 {
		t.Errorf("previous after TPre: got %d, want 9501", got)
	}

	ctx.PreviousContactEnd = 9300
	if got := ClampPreLookback(c, ctx); got != 9440 {
		t.Errorf("previous before TPre: got %d, want 9440", got)
	}
}

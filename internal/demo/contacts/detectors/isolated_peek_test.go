package detectors

import (
	"testing"

	"github.com/ok2ju/oversite/internal/demo"
	"github.com/ok2ju/oversite/internal/demo/contacts"
)

func TestIsolatedPeek(t *testing.T) {
	subject := "1"
	teammate := "2"

	t.Run("all_teammates_far_emits", func(t *testing.T) {
		ticks := []demo.AnalysisTick{
			mkTick(9600, 1, 0, 0, 0, 0, 0, 0, 0, true, 30),
			mkTick(9600, 2, 2000, 0, 0, 0, 0, 0, 0, true, 30),
		}
		c := contacts.Contact{Subject: subject, RoundNumber: 1, TFirst: 9600}
		ctx := &DetectorCtx{
			Subject: subject, SubjectTeam: "T", TickRate: 64, Ticks: mkTickIndex(ticks),
			Players: map[string]demo.RoundParticipant{
				subject:  {SteamID: subject, TeamSide: "T"},
				teammate: {SteamID: teammate, TeamSide: "T"},
			},
		}
		got := IsolatedPeek(&c, ctx)
		assertKinds(t, got, []string{"isolated_peek"})
	})

	t.Run("teammate_close_no_emit", func(t *testing.T) {
		ticks := []demo.AnalysisTick{
			mkTick(9600, 1, 0, 0, 0, 0, 0, 0, 0, true, 30),
			mkTick(9600, 2, 500, 0, 0, 0, 0, 0, 0, true, 30),
		}
		c := contacts.Contact{Subject: subject, RoundNumber: 1, TFirst: 9600}
		ctx := &DetectorCtx{
			Subject: subject, SubjectTeam: "T", TickRate: 64, Ticks: mkTickIndex(ticks),
			Players: map[string]demo.RoundParticipant{
				subject:  {SteamID: subject, TeamSide: "T"},
				teammate: {SteamID: teammate, TeamSide: "T"},
			},
		}
		got := IsolatedPeek(&c, ctx)
		assertKinds(t, got, nil)
	})

	t.Run("subject_not_alive_no_emit", func(t *testing.T) {
		ticks := []demo.AnalysisTick{
			mkTick(9600, 1, 0, 0, 0, 0, 0, 0, 0, false, 30),
		}
		c := contacts.Contact{Subject: subject, RoundNumber: 1, TFirst: 9600}
		ctx := &DetectorCtx{
			Subject: subject, SubjectTeam: "T", TickRate: 64, Ticks: mkTickIndex(ticks),
			Players: map[string]demo.RoundParticipant{
				subject: {SteamID: subject, TeamSide: "T"},
			},
		}
		got := IsolatedPeek(&c, ctx)
		assertKinds(t, got, nil)
	})
}

package detectors

import (
	"testing"

	"github.com/ok2ju/oversite/internal/demo"
	"github.com/ok2ju/oversite/internal/demo/contacts"
)

func TestNoRepositionAfterKill(t *testing.T) {
	subject := "1"
	killedEnemy := "2"
	survivingEnemy := "3"

	t.Run("kill_then_stand_still_emits", func(t *testing.T) {
		// Subject killed enemy at 9650, didn't move much by 9700, and a
		// second enemy is alive within 1500u at TLast.
		ticks := []demo.AnalysisTick{
			mkTick(9650, 1, 0, 0, 0, 0, 0, 0, 0, true, 30),
			mkTick(9690, 1, 20, 0, 0, 0, 0, 0, 0, true, 30),
			mkTick(9700, 1, 20, 0, 0, 0, 0, 0, 0, true, 30),
			mkTick(9700, 3, 1000, 0, 0, 0, 0, 0, 0, true, 30),
		}
		c := contacts.Contact{
			Subject: subject, RoundNumber: 1, TFirst: 9600, TLast: 9700, TPost: 9796,
			Enemies: []string{killedEnemy},
			Signals: []contacts.Signal{
				{Tick: 9650, Kind: contacts.SignalKill, Subject: contacts.SubjectAggressor, EnemySteam: killedEnemy},
			},
		}
		ctx := &DetectorCtx{
			Subject: subject, SubjectTeam: "T", TickRate: 64, Ticks: mkTickIndex(ticks),
			Players: map[string]demo.RoundParticipant{
				subject:        {SteamID: subject, TeamSide: "T"},
				survivingEnemy: {SteamID: survivingEnemy, TeamSide: "CT"},
			},
		}
		got := NoRepositionAfterKill(&c, ctx)
		assertKinds(t, got, []string{"no_reposition_after_kill"})
	})

	t.Run("moved_lots_no_emit", func(t *testing.T) {
		ticks := []demo.AnalysisTick{
			mkTick(9650, 1, 0, 0, 0, 0, 0, 0, 0, true, 30),
			mkTick(9700, 1, 300, 0, 0, 0, 0, 0, 0, true, 30),
			mkTick(9700, 3, 1000, 0, 0, 0, 0, 0, 0, true, 30),
		}
		c := contacts.Contact{
			Subject: subject, RoundNumber: 1, TFirst: 9600, TLast: 9700, TPost: 9796,
			Enemies: []string{killedEnemy},
			Signals: []contacts.Signal{
				{Tick: 9650, Kind: contacts.SignalKill, Subject: contacts.SubjectAggressor, EnemySteam: killedEnemy},
			},
		}
		ctx := &DetectorCtx{
			Subject: subject, SubjectTeam: "T", TickRate: 64, Ticks: mkTickIndex(ticks),
			Players: map[string]demo.RoundParticipant{
				subject:        {SteamID: subject, TeamSide: "T"},
				survivingEnemy: {SteamID: survivingEnemy, TeamSide: "CT"},
			},
		}
		got := NoRepositionAfterKill(&c, ctx)
		assertKinds(t, got, nil)
	})

	t.Run("killed_last_enemy_no_emit", func(t *testing.T) {
		// No second enemy alive nearby.
		ticks := []demo.AnalysisTick{
			mkTick(9650, 1, 0, 0, 0, 0, 0, 0, 0, true, 30),
			mkTick(9700, 1, 0, 0, 0, 0, 0, 0, 0, true, 30),
		}
		c := contacts.Contact{
			Subject: subject, RoundNumber: 1, TFirst: 9600, TLast: 9700, TPost: 9796,
			Enemies: []string{killedEnemy},
			Signals: []contacts.Signal{
				{Tick: 9650, Kind: contacts.SignalKill, Subject: contacts.SubjectAggressor, EnemySteam: killedEnemy},
			},
		}
		ctx := &DetectorCtx{
			Subject: subject, SubjectTeam: "T", TickRate: 64, Ticks: mkTickIndex(ticks),
			Players: map[string]demo.RoundParticipant{
				subject: {SteamID: subject, TeamSide: "T"},
			},
		}
		got := NoRepositionAfterKill(&c, ctx)
		assertKinds(t, got, nil)
	})
}

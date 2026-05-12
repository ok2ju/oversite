package detectors

import (
	"testing"

	"github.com/ok2ju/oversite/internal/demo"
	"github.com/ok2ju/oversite/internal/demo/contacts"
)

func TestShotWhileMoving(t *testing.T) {
	subject := "1"

	t.Run("fast_shot_emits", func(t *testing.T) {
		ticks := []demo.AnalysisTick{
			mkTick(9610, 1, 0, 0, 0, 0, 0, 150, 0, true, 30), // 150 u/s
		}
		c := contacts.Contact{Subject: subject, RoundNumber: 1, TFirst: 9600, TLast: 9700}
		ctx := &DetectorCtx{
			Subject: subject, TickRate: 64, Ticks: mkTickIndex(ticks),
			Events: []demo.GameEvent{
				{Tick: 9610, Type: "weapon_fire", AttackerSteamID: subject, Weapon: "ak47"},
			},
		}
		got := ShotWhileMoving(&c, ctx)
		assertKinds(t, got, []string{"shot_while_moving"})
	})

	t.Run("slow_shot_no_emit", func(t *testing.T) {
		ticks := []demo.AnalysisTick{
			mkTick(9610, 1, 0, 0, 0, 0, 0, 50, 0, true, 30), // 50 u/s
		}
		c := contacts.Contact{Subject: subject, RoundNumber: 1, TFirst: 9600, TLast: 9700}
		ctx := &DetectorCtx{
			Subject: subject, TickRate: 64, Ticks: mkTickIndex(ticks),
			Events: []demo.GameEvent{
				{Tick: 9610, Type: "weapon_fire", AttackerSteamID: subject, Weapon: "ak47"},
			},
		}
		got := ShotWhileMoving(&c, ctx)
		assertKinds(t, got, nil)
	})

	t.Run("two_fast_shots_two_emits", func(t *testing.T) {
		ticks := []demo.AnalysisTick{
			mkTick(9610, 1, 0, 0, 0, 0, 0, 200, 0, true, 30),
			mkTick(9620, 1, 0, 0, 0, 0, 0, 200, 0, true, 30),
		}
		c := contacts.Contact{Subject: subject, RoundNumber: 1, TFirst: 9600, TLast: 9700}
		ctx := &DetectorCtx{
			Subject: subject, TickRate: 64, Ticks: mkTickIndex(ticks),
			Events: []demo.GameEvent{
				{Tick: 9610, Type: "weapon_fire", AttackerSteamID: subject, Weapon: "ak47"},
				{Tick: 9620, Type: "weapon_fire", AttackerSteamID: subject, Weapon: "ak47"},
			},
		}
		got := ShotWhileMoving(&c, ctx)
		assertKinds(t, got, []string{"shot_while_moving", "shot_while_moving"})
	})
}

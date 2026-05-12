package detectors

import (
	"testing"

	"github.com/ok2ju/oversite/internal/demo"
	"github.com/ok2ju/oversite/internal/demo/contacts"
)

func TestPeekWhileReloading(t *testing.T) {
	subject := "1"
	weapons := DefaultWeaponCatalog()

	t.Run("low_clip_with_ak_emits", func(t *testing.T) {
		// 8/30 clip (~27%) — under 30% threshold.
		ticks := []demo.AnalysisTick{
			mkTick(9600, 1, 0, 0, 0, 0, 0, 0, 0, true, 8),
		}
		c := contacts.Contact{Subject: subject, RoundNumber: 1, TFirst: 9600, TPre: 9440}
		ctx := &DetectorCtx{
			Subject: subject, TickRate: 64, Ticks: mkTickIndex(ticks), Weapons: weapons,
			PreviousContactEnd: -1,
			Events: []demo.GameEvent{
				{Tick: 9500, Type: "weapon_fire", AttackerSteamID: subject, Weapon: "ak47"},
			},
		}
		got := PeekWhileReloading(&c, ctx)
		assertKinds(t, got, []string{"peek_while_reloading"})
	})

	t.Run("full_clip_no_emit", func(t *testing.T) {
		// 25/30 — over threshold.
		ticks := []demo.AnalysisTick{
			mkTick(9600, 1, 0, 0, 0, 0, 0, 0, 0, true, 25),
		}
		c := contacts.Contact{Subject: subject, RoundNumber: 1, TFirst: 9600, TPre: 9440}
		ctx := &DetectorCtx{
			Subject: subject, TickRate: 64, Ticks: mkTickIndex(ticks), Weapons: weapons,
			PreviousContactEnd: -1,
			Events: []demo.GameEvent{
				{Tick: 9500, Type: "weapon_fire", AttackerSteamID: subject, Weapon: "ak47"},
			},
		}
		got := PeekWhileReloading(&c, ctx)
		assertKinds(t, got, nil)
	})

	t.Run("knife_no_emit", func(t *testing.T) {
		ticks := []demo.AnalysisTick{
			mkTick(9600, 1, 0, 0, 0, 0, 0, 0, 0, true, 0),
		}
		c := contacts.Contact{Subject: subject, RoundNumber: 1, TFirst: 9600, TPre: 9440}
		ctx := &DetectorCtx{
			Subject: subject, TickRate: 64, Ticks: mkTickIndex(ticks), Weapons: weapons,
			PreviousContactEnd: -1,
			Events: []demo.GameEvent{
				{Tick: 9500, Type: "weapon_fire", AttackerSteamID: subject, Weapon: "knife"},
			},
		}
		got := PeekWhileReloading(&c, ctx)
		assertKinds(t, got, nil)
	})
}

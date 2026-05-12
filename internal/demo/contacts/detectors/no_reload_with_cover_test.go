package detectors

import (
	"testing"

	"github.com/ok2ju/oversite/internal/demo"
	"github.com/ok2ju/oversite/internal/demo/contacts"
)

func TestNoReloadWithCover(t *testing.T) {
	subject := "1"
	weapons := DefaultWeaponCatalog()

	t.Run("low_clip_no_reload_no_enemy_emits", func(t *testing.T) {
		// Subject shot at 9700 (last shot), clip stays at 5 through
		// post-window. No enemy fires after.
		ticks := []demo.AnalysisTick{
			mkTick(9700, 1, 0, 0, 0, 0, 0, 0, 0, true, 5),
			mkTick(9796, 1, 0, 0, 0, 0, 0, 0, 0, true, 5),
		}
		c := contacts.Contact{
			Subject: subject, RoundNumber: 1, TFirst: 9600, TLast: 9700, TPost: 9796,
		}
		ctx := &DetectorCtx{
			Subject: subject, TickRate: 64, Ticks: mkTickIndex(ticks), Weapons: weapons,
			Events: []demo.GameEvent{
				{Tick: 9700, Type: "weapon_fire", AttackerSteamID: subject, Weapon: "ak47"},
			},
		}
		got := NoReloadWithCover(&c, ctx)
		assertKinds(t, got, []string{"no_reload_with_cover"})
	})

	t.Run("reload_completes_no_emit", func(t *testing.T) {
		ticks := []demo.AnalysisTick{
			mkTick(9700, 1, 0, 0, 0, 0, 0, 0, 0, true, 5),
			mkTick(9750, 1, 0, 0, 0, 0, 0, 0, 0, true, 30), // reloaded
		}
		c := contacts.Contact{
			Subject: subject, RoundNumber: 1, TFirst: 9600, TLast: 9700, TPost: 9796,
		}
		ctx := &DetectorCtx{
			Subject: subject, TickRate: 64, Ticks: mkTickIndex(ticks), Weapons: weapons,
			Events: []demo.GameEvent{
				{Tick: 9700, Type: "weapon_fire", AttackerSteamID: subject, Weapon: "ak47"},
			},
		}
		got := NoReloadWithCover(&c, ctx)
		assertKinds(t, got, nil)
	})

	t.Run("enemy_fires_no_emit", func(t *testing.T) {
		ticks := []demo.AnalysisTick{
			mkTick(9700, 1, 0, 0, 0, 0, 0, 0, 0, true, 5),
			mkTick(9796, 1, 0, 0, 0, 0, 0, 0, 0, true, 5),
		}
		c := contacts.Contact{
			Subject: subject, RoundNumber: 1, TFirst: 9600, TLast: 9700, TPost: 9796,
		}
		ctx := &DetectorCtx{
			Subject: subject, SubjectTeam: "T", TickRate: 64, Ticks: mkTickIndex(ticks), Weapons: weapons,
			Players: map[string]demo.RoundParticipant{
				"2": {SteamID: "2", TeamSide: "CT"},
			},
			Events: []demo.GameEvent{
				{Tick: 9700, Type: "weapon_fire", AttackerSteamID: subject, Weapon: "ak47"},
				{Tick: 9720, Type: "weapon_fire", AttackerSteamID: "2", Weapon: "ak47"},
			},
		}
		got := NoReloadWithCover(&c, ctx)
		assertKinds(t, got, nil)
	})
}

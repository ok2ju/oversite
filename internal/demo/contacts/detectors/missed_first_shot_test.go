package detectors

import (
	"testing"

	"github.com/ok2ju/oversite/internal/demo"
	"github.com/ok2ju/oversite/internal/demo/contacts"
)

func TestMissedFirstShot(t *testing.T) {
	subjectSteam := "1"
	enemySteam := "2"

	// Subject at (0,0,0), enemy at (1000,0,0) — distance 1000u (<1500).
	ticks := []demo.AnalysisTick{
		mkTick(9600, 1, 0, 0, 0, 0, 0, 0, 0, true, 30),
		mkTick(9600, 2, 1000, 0, 0, 0, 0, 0, 0, true, 30),
	}
	idx := mkTickIndex(ticks)
	weapons := DefaultWeaponCatalog()

	t.Run("miss_at_close_range_emits", func(t *testing.T) {
		c := contacts.Contact{
			Subject: subjectSteam, RoundNumber: 1, TFirst: 9600, TLast: 9700, TPre: 9440, TPost: 9796,
			Enemies: []string{enemySteam},
		}
		ctx := &DetectorCtx{
			Subject: subjectSteam, TickRate: 64, Ticks: idx, Weapons: weapons,
			PreviousContactEnd: -1,
			Events: []demo.GameEvent{
				{Tick: 9600, Type: "weapon_fire", AttackerSteamID: subjectSteam, Weapon: "ak47", ExtraData: &demo.WeaponFireExtra{}},
			},
		}
		got := MissedFirstShot(&c, ctx)
		assertKinds(t, got, []string{"missed_first_shot"})
	})

	t.Run("hit_no_emit", func(t *testing.T) {
		c := contacts.Contact{
			Subject: subjectSteam, RoundNumber: 1, TFirst: 9600, TLast: 9700, TPre: 9440, TPost: 9796,
			Enemies: []string{enemySteam},
		}
		ctx := &DetectorCtx{
			Subject: subjectSteam, TickRate: 64, Ticks: idx, Weapons: weapons,
			PreviousContactEnd: -1,
			Events: []demo.GameEvent{
				{Tick: 9600, Type: "weapon_fire", AttackerSteamID: subjectSteam, Weapon: "ak47",
					ExtraData: &demo.WeaponFireExtra{HitVictimSteamID: enemySteam}},
			},
		}
		got := MissedFirstShot(&c, ctx)
		assertKinds(t, got, nil)
	})

	t.Run("knife_no_emit", func(t *testing.T) {
		c := contacts.Contact{
			Subject: subjectSteam, RoundNumber: 1, TFirst: 9600, TLast: 9700, TPre: 9440, TPost: 9796,
			Enemies: []string{enemySteam},
		}
		ctx := &DetectorCtx{
			Subject: subjectSteam, TickRate: 64, Ticks: idx, Weapons: weapons,
			PreviousContactEnd: -1,
			Events: []demo.GameEvent{
				{Tick: 9600, Type: "weapon_fire", AttackerSteamID: subjectSteam, Weapon: "knife", ExtraData: &demo.WeaponFireExtra{}},
			},
		}
		got := MissedFirstShot(&c, ctx)
		assertKinds(t, got, nil)
	})
}

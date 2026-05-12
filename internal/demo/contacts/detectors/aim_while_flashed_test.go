package detectors

import (
	"testing"

	"github.com/ok2ju/oversite/internal/demo"
	"github.com/ok2ju/oversite/internal/demo/contacts"
)

func TestAimWhileFlashed(t *testing.T) {
	subject := "1"
	enemy := "2"
	teammate := "3"

	t.Run("fire_during_enemy_flash_emits", func(t *testing.T) {
		// 0.9s flash @64Hz ≈ 58 ticks. Fire 30 ticks after.
		c := contacts.Contact{Subject: subject, RoundNumber: 1, TFirst: 9600, TLast: 9700}
		ctx := &DetectorCtx{
			Subject: subject, SubjectTeam: "T", TickRate: 64,
			Players: map[string]demo.RoundParticipant{
				enemy: {SteamID: enemy, TeamSide: "CT"},
			},
			Events: []demo.GameEvent{
				{Tick: 9600, Type: "player_flashed", AttackerSteamID: enemy, VictimSteamID: subject,
					ExtraData: &demo.PlayerFlashedExtra{DurationSecs: 0.9}},
				{Tick: 9630, Type: "weapon_fire", AttackerSteamID: subject, Weapon: "ak47"},
			},
		}
		got := AimWhileFlashed(&c, ctx)
		assertKinds(t, got, []string{"aim_while_flashed"})
	})

	t.Run("teammate_flash_no_emit", func(t *testing.T) {
		c := contacts.Contact{Subject: subject, RoundNumber: 1, TFirst: 9600, TLast: 9700}
		ctx := &DetectorCtx{
			Subject: subject, SubjectTeam: "T", TickRate: 64,
			Players: map[string]demo.RoundParticipant{
				teammate: {SteamID: teammate, TeamSide: "T"},
			},
			Events: []demo.GameEvent{
				{Tick: 9600, Type: "player_flashed", AttackerSteamID: teammate, VictimSteamID: subject,
					ExtraData: &demo.PlayerFlashedExtra{DurationSecs: 0.9}},
				{Tick: 9630, Type: "weapon_fire", AttackerSteamID: subject, Weapon: "ak47"},
			},
		}
		got := AimWhileFlashed(&c, ctx)
		assertKinds(t, got, nil)
	})

	t.Run("sub_threshold_flash_no_emit", func(t *testing.T) {
		c := contacts.Contact{Subject: subject, RoundNumber: 1, TFirst: 9600, TLast: 9700}
		ctx := &DetectorCtx{
			Subject: subject, SubjectTeam: "T", TickRate: 64,
			Players: map[string]demo.RoundParticipant{
				enemy: {SteamID: enemy, TeamSide: "CT"},
			},
			Events: []demo.GameEvent{
				{Tick: 9600, Type: "player_flashed", AttackerSteamID: enemy, VictimSteamID: subject,
					ExtraData: &demo.PlayerFlashedExtra{DurationSecs: 0.3}},
				{Tick: 9630, Type: "weapon_fire", AttackerSteamID: subject, Weapon: "ak47"},
			},
		}
		got := AimWhileFlashed(&c, ctx)
		assertKinds(t, got, nil)
	})
}

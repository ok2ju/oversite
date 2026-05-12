package detectors

import (
	"testing"

	"github.com/ok2ju/oversite/internal/demo"
	"github.com/ok2ju/oversite/internal/demo/contacts"
)

func TestLostHPAdvantage(t *testing.T) {
	subject := "1"
	enemy := "2"

	t.Run("subject_dies_with_advantage_emits", func(t *testing.T) {
		// Pre-contact, enemy was hurt for 50 hp (enemy hp 50). Subject
		// still full hp. Subject dies in contact.
		c := contacts.Contact{
			Subject: subject, RoundNumber: 1, TFirst: 9600, TLast: 9700, Enemies: []string{enemy},
			Signals: []contacts.Signal{
				{Tick: 9650, Kind: contacts.SignalKill, Subject: contacts.SubjectVictim, EnemySteam: enemy},
			},
		}
		ctx := &DetectorCtx{
			Subject: subject, TickRate: 64,
			Events: []demo.GameEvent{
				{Tick: 9000, Type: "player_hurt", VictimSteamID: enemy, ExtraData: &demo.PlayerHurtExtra{HealthDamage: 50}},
			},
		}
		got := LostHPAdvantage(&c, ctx)
		assertKinds(t, got, []string{"lost_hp_advantage"})
	})

	t.Run("subject_survives_no_emit", func(t *testing.T) {
		c := contacts.Contact{
			Subject: subject, RoundNumber: 1, TFirst: 9600, TLast: 9700, Enemies: []string{enemy},
			Signals: []contacts.Signal{
				{Tick: 9650, Kind: contacts.SignalKill, Subject: contacts.SubjectAggressor, EnemySteam: enemy},
			},
		}
		ctx := &DetectorCtx{Subject: subject, TickRate: 64}
		got := LostHPAdvantage(&c, ctx)
		assertKinds(t, got, nil)
	})

	t.Run("small_hp_gap_no_emit", func(t *testing.T) {
		// Enemy 90, subject 100. Gap 10 < 25.
		c := contacts.Contact{
			Subject: subject, RoundNumber: 1, TFirst: 9600, TLast: 9700, Enemies: []string{enemy},
			Signals: []contacts.Signal{
				{Tick: 9650, Kind: contacts.SignalKill, Subject: contacts.SubjectVictim, EnemySteam: enemy},
			},
		}
		ctx := &DetectorCtx{
			Subject: subject, TickRate: 64,
			Events: []demo.GameEvent{
				{Tick: 9000, Type: "player_hurt", VictimSteamID: enemy, ExtraData: &demo.PlayerHurtExtra{HealthDamage: 10}},
			},
		}
		got := LostHPAdvantage(&c, ctx)
		assertKinds(t, got, nil)
	})
}

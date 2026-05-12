package detectors

import (
	"testing"

	"github.com/ok2ju/oversite/internal/demo/contacts"
)

func TestSlowReaction(t *testing.T) {
	cases := []struct {
		name      string
		contact   contacts.Contact
		ctx       *DetectorCtx
		wantKinds []string
	}{
		{
			name: "happy_path_emits",
			contact: contacts.Contact{
				Subject:     "S_P",
				RoundNumber: 1,
				TFirst:      9600,
				TLast:       9700,
				TPre:        9440,
				TPost:       9796,
				Enemies:     []string{"S_E1"},
				Signals: []contacts.Signal{
					{Tick: 9600, EnemySteam: "S_E1", Kind: contacts.SignalVisibility, Subject: contacts.SubjectAggressor},
					{Tick: 9650, EnemySteam: "S_E1", Kind: contacts.SignalWeaponFireHit, Subject: contacts.SubjectAggressor},
				},
			},
			ctx:       &DetectorCtx{Subject: "S_P", SubjectTeam: "T", TickRate: 64, PreviousContactEnd: -1},
			wantKinds: []string{"slow_reaction"},
		},
		{
			name: "below_threshold_no_emit",
			contact: contacts.Contact{
				Subject: "S_P", RoundNumber: 1, TFirst: 9600, TLast: 9700, TPre: 9440, TPost: 9796,
				Enemies: []string{"S_E1"},
				Signals: []contacts.Signal{
					{Tick: 9600, EnemySteam: "S_E1", Kind: contacts.SignalVisibility, Subject: contacts.SubjectAggressor},
					{Tick: 9610, EnemySteam: "S_E1", Kind: contacts.SignalWeaponFireHit, Subject: contacts.SubjectAggressor},
				},
			},
			ctx:       &DetectorCtx{Subject: "S_P", SubjectTeam: "T", TickRate: 64, PreviousContactEnd: -1},
			wantKinds: nil,
		},
		{
			name: "no_visibility_signal_no_emit",
			contact: contacts.Contact{
				Subject: "S_P", RoundNumber: 1, TFirst: 9600, TLast: 9700, TPre: 9440, TPost: 9796,
				Enemies: []string{"S_E1"},
				Signals: []contacts.Signal{
					{Tick: 9650, EnemySteam: "S_E1", Kind: contacts.SignalWeaponFireHit, Subject: contacts.SubjectAggressor},
				},
			},
			ctx:       &DetectorCtx{Subject: "S_P", SubjectTeam: "T", TickRate: 64, PreviousContactEnd: -1},
			wantKinds: nil,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := SlowReaction(&tc.contact, tc.ctx)
			assertKinds(t, got, tc.wantKinds)
		})
	}
}

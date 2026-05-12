package detectors

import (
	"testing"

	"github.com/ok2ju/oversite/internal/demo"
	"github.com/ok2ju/oversite/internal/demo/contacts"
)

func TestBadCrosshairHeight(t *testing.T) {
	subject := "1"
	enemy := "2"

	t.Run("pitch_far_off_emits", func(t *testing.T) {
		// Subject at (0,0,0), enemy at (1000,0,0). Same Z so expected
		// pitch is 0°. Subject's actual pitch is 30°, delta 30°.
		ticks := []demo.AnalysisTick{
			mkTick(9600, 1, 0, 0, 0, 0, 30, 0, 0, true, 30),
			mkTick(9600, 2, 1000, 0, 0, 0, 0, 0, 0, true, 30),
		}
		c := contacts.Contact{Subject: subject, RoundNumber: 1, TFirst: 9600, Enemies: []string{enemy}}
		ctx := &DetectorCtx{
			Subject: subject, SubjectTeam: "T", TickRate: 64, Ticks: mkTickIndex(ticks),
		}
		got := BadCrosshairHeight(&c, ctx)
		assertKinds(t, got, []string{"bad_crosshair_height"})
	})

	t.Run("pitch_close_no_emit", func(t *testing.T) {
		// Subject pitch 2° off expected 0° — under 5° threshold.
		ticks := []demo.AnalysisTick{
			mkTick(9600, 1, 0, 0, 0, 0, 2, 0, 0, true, 30),
			mkTick(9600, 2, 1000, 0, 0, 0, 0, 0, 0, true, 30),
		}
		c := contacts.Contact{Subject: subject, RoundNumber: 1, TFirst: 9600, Enemies: []string{enemy}}
		ctx := &DetectorCtx{Subject: subject, SubjectTeam: "T", TickRate: 64, Ticks: mkTickIndex(ticks)}
		got := BadCrosshairHeight(&c, ctx)
		assertKinds(t, got, nil)
	})

	t.Run("pitch_zero_no_emit", func(t *testing.T) {
		// Pitch == 0 means "not measured" (old demo).
		ticks := []demo.AnalysisTick{
			mkTick(9600, 1, 0, 0, 0, 0, 0, 0, 0, true, 30),
			mkTick(9600, 2, 1000, 0, 0, 0, 0, 0, 0, true, 30),
		}
		c := contacts.Contact{Subject: subject, RoundNumber: 1, TFirst: 9600, Enemies: []string{enemy}}
		ctx := &DetectorCtx{Subject: subject, SubjectTeam: "T", TickRate: 64, Ticks: mkTickIndex(ticks)}
		got := BadCrosshairHeight(&c, ctx)
		assertKinds(t, got, nil)
	})
}

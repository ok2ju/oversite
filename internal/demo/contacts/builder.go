package contacts

import (
	"github.com/ok2ju/oversite/internal/demo"
)

// Contacts are built at a fixed 64-tick assumption. Demos with a
// different tick rate are rejected by the orchestrator (see contacts.Run).
const (
	TickRate64     = 64.0
	SecondsPerTick = 1.0 / TickRate64

	// MergeWindowTicks (T_merge): 2.0 s gap below which signals stay in
	// the same contact.
	MergeWindowTicks = 128

	// PreWindowTicks (T_pre): 2.5 s pre-window for positioning analysis.
	PreWindowTicks = 160

	// PostWindowTicks (T_post): 1.5 s post-window for post-fight reposition
	// mistakes.
	PostWindowTicks = 96

	// IdleCloseTicks is used by the outcome classifier; documented here so
	// the grouper's MergeWindow stays separable from the outcome cutoff.
	IdleCloseTicks = 192

	// TradeWindowTicks is the look-ahead for trade detection (analysis §4.4).
	TradeWindowTicks = 320
)

// Contact is the unfinalized output of the grouper. It carries the signal
// cluster but no outcome yet; the outcome classifier in outcome.go
// transforms it into a fully-populated ContactMoment.
//
// Output order from Build is chronological (the slice is appended as
// signals are walked). Phase 3 detectors rely on this so they can read
// `contacts[i-1].TLast` for the previous-contact-end hint without
// re-sorting.
type Contact struct {
	RoundNumber int    `json:"round_number"`
	RoundID     int64  `json:"round_id"`
	Subject     string `json:"subject"`

	TFirst int32 `json:"t_first"`
	TLast  int32 `json:"t_last"`
	TPre   int32 `json:"t_pre"`
	TPost  int32 `json:"t_post"`

	Enemies    []string       `json:"enemies"`
	EnemyIndex map[string]int `json:"-"`
	Signals    []Signal       `json:"signals"`

	Extras ContactExtras `json:"extras"`
}

// ContactExtras carries flags the grouper and outcome classifier set as
// they walk signals. Marshaled to extras_json on persist.
type ContactExtras struct {
	TruncatedRoundEnd     bool `json:"truncated_round_end,omitempty"`
	FlashOnly             bool `json:"flash_only,omitempty"`
	TeammateFlashedDuring bool `json:"teammate_flashed_during,omitempty"`
	WallbangTaken         bool `json:"wallbang_taken,omitempty"`
}

// BuildInputs is the per-(subject, round) input to Build.
type BuildInputs struct {
	Subject       string
	Round         demo.RoundData
	SubjectAlive  AliveRange
	Signals       []Signal
	TeammateFlash []TeammateFlash
}

// TeammateFlash is the lightweight record needed by the
// teammate_flashed_during flag. Not a contact-opening signal itself.
type TeammateFlash struct {
	Tick         int32
	DurationSecs float64
}

// Build groups signals into contacts using the merge-window algorithm
// from analysis §3.1. Returns one Contact per cluster. Output order is
// chronological (appended as signals are walked).
func Build(in BuildInputs) []Contact {
	if len(in.Signals) == 0 {
		return nil
	}

	contacts := make([]Contact, 0, 4)
	var current *Contact

	for i := range in.Signals {
		s := in.Signals[i]
		if current == nil {
			current = openContact(in, s)
			continue
		}
		if int(s.Tick-current.TLast) <= MergeWindowTicks {
			extendContact(current, s)
			continue
		}
		contacts = append(contacts, finalize(in, *current))
		current = openContact(in, s)
	}
	if current != nil {
		contacts = append(contacts, finalize(in, *current))
	}

	attachTeammateFlashes(contacts, in.TeammateFlash)

	return contacts
}

func openContact(in BuildInputs, s Signal) *Contact {
	return &Contact{
		RoundNumber: in.Round.Number,
		Subject:     in.Subject,
		TFirst:      s.Tick,
		TLast:       s.Tick,
		Enemies:     []string{s.EnemySteam},
		EnemyIndex:  map[string]int{s.EnemySteam: 0},
		Signals:     []Signal{s},
	}
}

func extendContact(c *Contact, s Signal) {
	c.TLast = s.Tick
	c.Signals = append(c.Signals, s)
	if _, ok := c.EnemyIndex[s.EnemySteam]; !ok {
		c.EnemyIndex[s.EnemySteam] = len(c.Enemies)
		c.Enemies = append(c.Enemies, s.EnemySteam)
	}
}

func finalize(in BuildInputs, c Contact) Contact {
	c.TPre = clampTPre(c.TFirst, in)
	c.TPost = clampTPost(c.TLast, in)
	setWallbangFlag(&c)
	setFlashOnlyFlag(&c)
	setTruncatedFlag(&c, in)
	return c
}

// clampTPre returns max(t_first - T_pre, P.alive_tick_in_round, round.freeze_end_tick).
func clampTPre(tFirst int32, in BuildInputs) int32 {
	candidate := tFirst - PreWindowTicks
	if in.SubjectAlive.SpawnTick > candidate {
		candidate = in.SubjectAlive.SpawnTick
	}
	if int32(in.Round.FreezeEndTick) > candidate {
		candidate = int32(in.Round.FreezeEndTick)
	}
	if candidate < 0 {
		return 0
	}
	return candidate
}

// clampTPost returns min(t_last + T_post, P.death_tick_or_round_end).
func clampTPost(tLast int32, in BuildInputs) int32 {
	candidate := tLast + PostWindowTicks
	var cutoff int32
	switch {
	case in.SubjectAlive.DeathTick > 0:
		cutoff = in.SubjectAlive.DeathTick
	case in.Round.EndTick > 0:
		cutoff = int32(in.Round.EndTick)
	default:
		return candidate
	}
	if candidate > cutoff {
		return cutoff
	}
	return candidate
}

// setWallbangFlag fires when the subject took wallbang damage during the
// contact. PlayerHurtExtra.Penetrated > 0 is the player_hurt signal; for
// kills it falls back to KillExtra.Wallbang.
func setWallbangFlag(c *Contact) {
	for i := range c.Signals {
		s := c.Signals[i]
		if s.Subject != SubjectVictim {
			continue
		}
		switch s.Kind {
		case SignalPlayerHurt, SignalUtilityDamage:
			if s.Extras.Penetrated > 0 {
				c.Extras.WallbangTaken = true
				return
			}
		case SignalKill:
			if s.Extras.Wallbang || s.Extras.Penetrated > 0 {
				c.Extras.WallbangTaken = true
				return
			}
		}
	}
}

// setFlashOnlyFlag fires when every signal in the contact is a flash
// (no shots, no damage, no kills, no visibility).
func setFlashOnlyFlag(c *Contact) {
	if len(c.Signals) == 0 {
		return
	}
	for i := range c.Signals {
		if c.Signals[i].Kind != SignalFlash {
			return
		}
	}
	c.Extras.FlashOnly = true
}

// setTruncatedFlag fires when the contact's last signal lands at the
// round-end tick (with a one-tick slop).
func setTruncatedFlag(c *Contact, in BuildInputs) {
	if in.Round.EndTick > 0 && c.TLast >= int32(in.Round.EndTick-1) {
		c.Extras.TruncatedRoundEnd = true
	}
}

// attachTeammateFlashes scans teammate-flash records and sets
// TeammateFlashedDuring on any contact whose [TFirst, TLast] window
// overlaps a teammate-flash effective window. Effective window is
// approximately [Tick, Tick + DurationSecs * 64].
func attachTeammateFlashes(contacts []Contact, flashes []TeammateFlash) {
	if len(contacts) == 0 || len(flashes) == 0 {
		return
	}
	for i := range contacts {
		c := &contacts[i]
		for _, f := range flashes {
			endTick := f.Tick + int32(f.DurationSecs*TickRate64)
			if endTick < c.TFirst || f.Tick > c.TLast {
				continue
			}
			c.Extras.TeammateFlashedDuring = true
			break
		}
	}
}

// clearTransient drops the EnemyIndex dedup map before the persister
// serializes the contact. Builder-internal state never reaches the DB.
func (c *Contact) clearTransient() {
	c.EnemyIndex = nil
}

package detectors

import (
	"github.com/ok2ju/oversite/internal/demo"
	"github.com/ok2ju/oversite/internal/demo/analysis"
	"github.com/ok2ju/oversite/internal/demo/contacts"
)

// DetectorCtx is the per-(contact) read-only context every detector
// consumes. Built once per contact by BuildCtx — see
// 02-context-builder.md.
type DetectorCtx struct {
	// Subject is the SteamID64 (decimal string) of the contact's
	// selected player. Mirrors Contact.Subject — provided here so
	// detectors don't dereference c.Subject for every comparison.
	Subject string

	// SubjectTeam is "CT" or "T" for this round. The opposite side is
	// the enemy team for the duration of the contact.
	SubjectTeam string

	// Ticks is the shared per-player tick index over result.AnalysisTicks
	// for this demo, built once via analysis.BuildTickIndex. Every
	// detector that needs positions, yaw, pitch, velocity, ammo, or
	// alive state reads through this index.
	Ticks analysis.PerPlayerTickIndex

	// Events are the demo's GameEvent rows filtered to the round of
	// this contact. Already sorted ascending by tick. Detectors that
	// need first-shot or first-spotted lookups walk this slice.
	Events []demo.GameEvent

	// Round is a pointer to the parent round metadata (FreezeEndTick,
	// EndTick, Roster). Detectors that need the per-round team map or
	// the freeze-end clamp read here.
	Round *demo.RoundData

	// Players is a SteamID-keyed view of the contact's roster. The
	// value contains the player's TeamSide and PlayerName but NOT a
	// pointer to a per-player struct (cheaper than indirection and
	// avoids retaining the parser-side RoundData slice past Run).
	Players map[string]demo.RoundParticipant

	// Weapons is the static per-weapon catalog (weapons.go). Used by
	// peek_while_reloading and no_reload_with_cover.
	Weapons WeaponCatalog

	// PreviousContactEnd is the TLast of the previous contact for
	// this (demo, subject), or -1 when this is the first contact in
	// the (demo, subject) sequence. Pre-window lookbacks clamp to
	// max(Contact.TPre, PreviousContactEnd+1) to satisfy analysis
	// §9.3.
	PreviousContactEnd int32

	// TickRate is fixed at 64.0 in v1 (the orchestrator gates on
	// [63.9, 64.1] before invoking detectors). Surfaced here so a
	// detector that needs a tick-to-seconds conversion doesn't
	// re-hardcode the constant.
	TickRate float64
}

// RunData groups the per-demo inputs the runner computes once and
// reuses across every contact for that demo. Built at the top of
// detectors.Run (05-runner-persist-wire.md §3.3).
type RunData struct {
	Result         *demo.ParseResult
	Ticks          analysis.PerPlayerTickIndex
	EventsByRound  map[int][]demo.GameEvent
	RoundsByNumber map[int]*demo.RoundData
	PlayersByRound map[int]map[string]demo.RoundParticipant
	TeamsByRound   map[int]map[string]string // SteamID → "CT"/"T"
	Weapons        WeaponCatalog
	TickRate       float64
}

// NewRunData computes RunData from a ParseResult. Allocates four maps
// of size O(rounds) plus the tick index of size O(ticks × players).
// Cost: ~50–150ms for a typical demo.
func NewRunData(result *demo.ParseResult) *RunData {
	if result == nil {
		return &RunData{
			EventsByRound:  map[int][]demo.GameEvent{},
			RoundsByNumber: map[int]*demo.RoundData{},
			PlayersByRound: map[int]map[string]demo.RoundParticipant{},
			TeamsByRound:   map[int]map[string]string{},
			Weapons:        DefaultWeaponCatalog(),
			TickRate:       contacts.TickRate64,
		}
	}
	tickRate := result.Header.TickRate
	if tickRate == 0 {
		tickRate = contacts.TickRate64
	}
	return &RunData{
		Result:         result,
		Ticks:          analysis.BuildTickIndex(result.AnalysisTicks),
		EventsByRound:  contacts.PartitionEventsByRound(result.Events),
		RoundsByNumber: indexRounds(result.Rounds),
		PlayersByRound: buildPlayersByRound(result.Rounds),
		TeamsByRound:   analysis.TeamsByRoundFromRosters(result.Rounds),
		Weapons:        DefaultWeaponCatalog(),
		TickRate:       tickRate,
	}
}

// indexRounds returns a Number → *RoundData map. RoundData is a value
// type in result.Rounds; the map returns a pointer into the slice so
// detectors get O(1) access without copying ~200 bytes per call.
func indexRounds(rounds []demo.RoundData) map[int]*demo.RoundData {
	out := make(map[int]*demo.RoundData, len(rounds))
	for i := range rounds {
		r := &rounds[i]
		if r.Number == 0 {
			continue
		}
		out[r.Number] = r
	}
	return out
}

// buildPlayersByRound returns Round.Number → (SteamID → RoundParticipant).
// One entry per roster row per round. Lets detectors look up the
// enemy team / inventory for an arbitrary SteamID without walking the
// roster slice.
func buildPlayersByRound(rounds []demo.RoundData) map[int]map[string]demo.RoundParticipant {
	out := make(map[int]map[string]demo.RoundParticipant, len(rounds))
	for _, r := range rounds {
		if r.Number == 0 {
			continue
		}
		inner := make(map[string]demo.RoundParticipant, len(r.Roster))
		for _, rp := range r.Roster {
			if rp.SteamID == "" {
				continue
			}
			inner[rp.SteamID] = rp
		}
		out[r.Number] = inner
	}
	return out
}

// BuildCtx assembles the DetectorCtx for one contact. previousContactEnd
// is the TLast of the prior contact for this (demo, subject) — the
// runner threads it through after sorting contacts by t_first ASC. Pass
// -1 for the first contact in the round.
func BuildCtx(c *contacts.Contact, previousContactEnd int32, rd *RunData) *DetectorCtx {
	var round *demo.RoundData
	if rd != nil {
		round = rd.RoundsByNumber[c.RoundNumber]
	}
	var players map[string]demo.RoundParticipant
	var teams map[string]string
	if rd != nil {
		players = rd.PlayersByRound[c.RoundNumber]
		teams = rd.TeamsByRound[c.RoundNumber]
	}
	subjectTeam := ""
	if teams != nil {
		subjectTeam = teams[c.Subject]
	}
	tickRate := contacts.TickRate64
	var idx analysis.PerPlayerTickIndex
	var events []demo.GameEvent
	weapons := DefaultWeaponCatalog()
	if rd != nil {
		tickRate = rd.TickRate
		idx = rd.Ticks
		events = rd.EventsByRound[c.RoundNumber]
		if rd.Weapons != nil {
			weapons = rd.Weapons
		}
	}
	return &DetectorCtx{
		Subject:            c.Subject,
		SubjectTeam:        subjectTeam,
		Ticks:              idx,
		Events:             events,
		Round:              round,
		Players:            players,
		Weapons:            weapons,
		PreviousContactEnd: previousContactEnd,
		TickRate:           tickRate,
	}
}

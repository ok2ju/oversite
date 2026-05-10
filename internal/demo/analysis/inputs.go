package analysis

import (
	"strconv"

	"github.com/ok2ju/oversite/internal/demo"
)

// PerPlayerTickIndex maps a player's decimal-string SteamID to their sampled
// AnalysisTick rows, themselves keyed by tick. The string keying mirrors how
// game events identify players (GameEvent.AttackerSteamID / VictimSteamID),
// so the rule lookups don't have to convert at every fire event.
//
// Slice 8 builds the index once per analyzer pass (BuildTickIndex). Direct
// linear scans per fire event would be O(events × ticks) and demonstrably
// slow on long matches.
type PerPlayerTickIndex map[string]map[int]demo.AnalysisTick

// BuildTickIndex collects ticks into a (steamID, tick) → AnalysisTick lookup.
// SteamIDs come off the wire as uint64 (saves ~20 B per AnalysisTick row vs a
// per-row string); we convert once here so the analyzer rules can match the
// string SteamIDs that travel on GameEvent without a per-event strconv.
func BuildTickIndex(ticks []demo.AnalysisTick) PerPlayerTickIndex {
	if len(ticks) == 0 {
		return nil
	}
	idx := make(PerPlayerTickIndex, 16)
	steamCache := make(map[uint64]string, 16)
	for _, t := range ticks {
		s, ok := steamCache[t.SteamID]
		if !ok {
			s = strconv.FormatUint(t.SteamID, 10)
			steamCache[t.SteamID] = s
		}
		inner, ok := idx[s]
		if !ok {
			inner = make(map[int]demo.AnalysisTick, 64)
			idx[s] = inner
		}
		inner[int(t.Tick)] = t
	}
	return idx
}

// nearestTick returns the most recent AnalysisTick at or before the supplied
// tick for the given steamID. Returns ok=false when the player has no samples
// in the index, or when every recorded sample is strictly after tick (no
// pre-fire state available — the analyzer rules treat this as a skip).
//
// The fire event's tick is rarely an exact match for a sampled tick (samples
// land every tickInterval, fires land on the in-game tick of the shot), so
// the linear walk inside the player's per-tick map is the simplest correct
// thing — for a 30-min match the per-player map holds ~80K entries and the
// walk is bounded by the sample interval being a small constant. If profiling
// flags this, store a sorted []int32 alongside and bsearch.
func nearestTick(idx PerPlayerTickIndex, steamID string, tick int) (demo.AnalysisTick, bool) {
	inner, ok := idx[steamID]
	if !ok || len(inner) == 0 {
		return demo.AnalysisTick{}, false
	}
	if t, ok := inner[tick]; ok {
		return t, true
	}
	bestTick := -1
	var best demo.AnalysisTick
	for t, row := range inner {
		if t > tick {
			continue
		}
		if t > bestTick {
			bestTick = t
			best = row
		}
	}
	if bestTick < 0 {
		return demo.AnalysisTick{}, false
	}
	return best, true
}

// nearestEnemyTick returns the most-recent AnalysisTick at or before the
// supplied tick for any player whose normalized team differs from
// attackerTeam. The aim rule uses this to compare the attacker's pitch
// against the line-of-sight to a plausible target. Returns ok=false when no
// enemy sample is available — the rule treats this as a skip.
//
// teams maps decimal-string SteamID → "CT"/"T". Players missing from the map
// or with an empty side are ignored (we can't decide whether they are an
// enemy).
func nearestEnemyTick(idx PerPlayerTickIndex, teams map[string]string, attackerTeam string, tick int) (demo.AnalysisTick, bool) {
	if attackerTeam == "" {
		return demo.AnalysisTick{}, false
	}
	bestTick := -1
	var best demo.AnalysisTick
	for steamID, inner := range idx {
		side, ok := teams[steamID]
		if !ok || side == "" || side == attackerTeam {
			continue
		}
		for t, row := range inner {
			if t > tick {
				continue
			}
			if t > bestTick {
				bestTick = t
				best = row
			}
		}
	}
	if bestTick < 0 {
		return demo.AnalysisTick{}, false
	}
	return best, true
}

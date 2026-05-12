package analysis

import (
	"sort"
	"strconv"

	"github.com/ok2ju/oversite/internal/demo"
)

// PerPlayerTickIndex is the lookup structure rules use to find a player's
// AnalysisTick row at or before a given tick. It pairs the per-player
// (tick → row) map with a per-player ascending-sorted []int32 of tick
// numbers so callers can binary-search the lookback window in O(log n)
// instead of scanning every sample on every event.
//
// Slice 8 originally exposed the bare map. Slice 9's flick / counter-strafe
// rules made the linear scans the analyzer's hot path on long matches
// (parsing appeared "stuck" at 80% as the analyzer ran), which is why the
// sorted slice was added — every helper here is O(log n) per call.
//
// The struct is intentionally a value type with exported fields rather than
// hidden behind methods. Direct field access keeps the rules' loop bodies
// readable, and callers never need to mutate the index after BuildTickIndex
// returns.
type PerPlayerTickIndex struct {
	// Rows is keyed by decimal-string SteamID (matches GameEvent.* IDs) and
	// inner-keyed by tick. Lookup of a known tick is O(1).
	Rows map[string]map[int]demo.AnalysisTick
	// Sorted holds the same ticks per player in ascending order. Built once
	// at index construction so the bsearch helpers don't re-collect keys
	// per call.
	Sorted map[string][]int32
}

// BuildTickIndex collects ticks into a per-player (tick → AnalysisTick)
// lookup plus an ascending-sorted slice of ticks. SteamIDs come off the wire
// as uint64 (saves ~20 B per AnalysisTick row vs a per-row string); we
// convert once here so the analyzer rules can match the string SteamIDs that
// travel on GameEvent without a per-event strconv.
//
// The parser already emits rows in ascending-tick order per player, so the
// trailing sort.Slice is defensive — ordering is part of the index contract,
// not the parser's promise.
func BuildTickIndex(ticks []demo.AnalysisTick) PerPlayerTickIndex {
	if len(ticks) == 0 {
		return PerPlayerTickIndex{}
	}
	rows := make(map[string]map[int]demo.AnalysisTick, 16)
	sorted := make(map[string][]int32, 16)
	steamCache := make(map[uint64]string, 16)
	for _, t := range ticks {
		s, ok := steamCache[t.SteamID]
		if !ok {
			s = strconv.FormatUint(t.SteamID, 10)
			steamCache[t.SteamID] = s
		}
		inner, ok := rows[s]
		if !ok {
			inner = make(map[int]demo.AnalysisTick, 64)
			rows[s] = inner
		}
		if _, dup := inner[int(t.Tick)]; !dup {
			sorted[s] = append(sorted[s], t.Tick)
		}
		inner[int(t.Tick)] = t
	}
	for s, ts := range sorted {
		sort.Slice(ts, func(i, j int) bool { return ts[i] < ts[j] })
		sorted[s] = ts
	}
	return PerPlayerTickIndex{Rows: rows, Sorted: sorted}
}

// nearestTick returns the most recent AnalysisTick at or before the supplied
// tick for the given steamID. Returns ok=false when the player has no samples
// in the index, or when every recorded sample is strictly after tick (no
// pre-fire state available — the analyzer rules treat this as a skip).
//
// Implementation is sort.Search over the player's sorted-tick slice — O(log n)
// rather than the prior O(n) scan over the entire per-player map.
func nearestTick(idx PerPlayerTickIndex, steamID string, tick int) (demo.AnalysisTick, bool) {
	inner, ok := idx.Rows[steamID]
	if !ok || len(inner) == 0 {
		return demo.AnalysisTick{}, false
	}
	if t, ok := inner[tick]; ok {
		return t, true
	}
	sorted := idx.Sorted[steamID]
	if len(sorted) == 0 {
		return demo.AnalysisTick{}, false
	}
	i := sort.Search(len(sorted), func(i int) bool { return int(sorted[i]) > tick })
	if i == 0 {
		return demo.AnalysisTick{}, false
	}
	return inner[int(sorted[i-1])], true
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
	for steamID, inner := range idx.Rows {
		side, ok := teams[steamID]
		if !ok || side == "" || side == attackerTeam {
			continue
		}
		sorted := idx.Sorted[steamID]
		if len(sorted) == 0 {
			continue
		}
		i := sort.Search(len(sorted), func(i int) bool { return int(sorted[i]) > tick })
		if i == 0 {
			continue
		}
		cand := int(sorted[i-1])
		if cand > bestTick {
			bestTick = cand
			best = inner[cand]
		}
	}
	if bestTick < 0 {
		return demo.AnalysisTick{}, false
	}
	return best, true
}

// previousTickRows returns up to n AnalysisTick rows for the given player
// whose tick is strictly less than fireTick, ordered from most-recent to
// oldest. Returns nil when the player has no samples or no sample lands
// before fireTick.
//
// Used by the lookback rules (counter-strafe, missed-flick, time-to-fire) so
// each one stops walking after collecting its window instead of scanning the
// entire per-player map and bubble-sorting it on every event.
func previousTickRows(idx PerPlayerTickIndex, steamID string, fireTick, n int) []demo.AnalysisTick {
	if n <= 0 {
		return nil
	}
	inner, ok := idx.Rows[steamID]
	if !ok || len(inner) == 0 {
		return nil
	}
	sorted := idx.Sorted[steamID]
	if len(sorted) == 0 {
		return nil
	}
	// First index whose tick >= fireTick — everything to the left is strictly
	// before the fire.
	i := sort.Search(len(sorted), func(i int) bool { return int(sorted[i]) >= fireTick })
	if i == 0 {
		return nil
	}
	out := make([]demo.AnalysisTick, 0, n)
	for j := i - 1; j >= 0 && len(out) < n; j-- {
		out = append(out, inner[int(sorted[j])])
	}
	return out
}

// NearestTick is the exported counterpart to nearestTick — kept as a shim
// so internal/demo/contacts/detectors can reuse the same lookup without
// copying the implementation. New callers prefer the upper-case name.
func NearestTick(idx PerPlayerTickIndex, steamID string, tick int) (demo.AnalysisTick, bool) {
	return nearestTick(idx, steamID, tick)
}

// NearestEnemyTick is the exported counterpart to nearestEnemyTick.
func NearestEnemyTick(idx PerPlayerTickIndex, teams map[string]string, attackerTeam string, tick int) (demo.AnalysisTick, bool) {
	return nearestEnemyTick(idx, teams, attackerTeam, tick)
}

// PreviousTickRows is the exported counterpart to previousTickRows.
func PreviousTickRows(idx PerPlayerTickIndex, steamID string, fireTick, n int) []demo.AnalysisTick {
	return previousTickRows(idx, steamID, fireTick, n)
}

// TickRangeRows is the exported counterpart to tickRangeRows.
func TickRangeRows(idx PerPlayerTickIndex, steamID string, lo, hi int) []demo.AnalysisTick {
	return tickRangeRows(idx, steamID, lo, hi)
}

// TeamsByRoundFromRosters is the exported counterpart to
// teamsByRoundFromRosters — re-exposed for use by
// internal/demo/contacts/detectors.
func TeamsByRoundFromRosters(rounds []demo.RoundData) map[int]map[string]string {
	return teamsByRoundFromRosters(rounds)
}

// tickRangeRows returns AnalysisTick rows for the given player whose tick
// falls in the inclusive [lo, hi] range, ordered ascending. Empty when the
// player has no samples in the window. Used by positioning's
// "any teammate within radius in the last second" predicate so it doesn't
// scan the player's full sample history.
func tickRangeRows(idx PerPlayerTickIndex, steamID string, lo, hi int) []demo.AnalysisTick {
	inner, ok := idx.Rows[steamID]
	if !ok || len(inner) == 0 {
		return nil
	}
	sorted := idx.Sorted[steamID]
	if len(sorted) == 0 {
		return nil
	}
	// First index whose tick >= lo.
	start := sort.Search(len(sorted), func(i int) bool { return int(sorted[i]) >= lo })
	// First index whose tick > hi.
	end := sort.Search(len(sorted), func(i int) bool { return int(sorted[i]) > hi })
	if start >= end {
		return nil
	}
	out := make([]demo.AnalysisTick, 0, end-start)
	for j := start; j < end; j++ {
		out = append(out, inner[int(sorted[j])])
	}
	return out
}

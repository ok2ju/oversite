package analysis

import "github.com/ok2ju/oversite/internal/demo"

// teamsByRoundFromRosters builds a per-round (steamID → "CT"/"T") map from
// the parser's RoundParticipant rosters. Players present in multiple rounds
// (the common case) appear once per round, so a halftime side switch is
// represented correctly without a global side map.
func teamsByRoundFromRosters(rounds []demo.RoundData) map[int]map[string]string {
	out := make(map[int]map[string]string, len(rounds))
	for _, r := range rounds {
		if len(r.Roster) == 0 {
			continue
		}
		inner := make(map[string]string, len(r.Roster))
		for _, rp := range r.Roster {
			if rp.SteamID == "" || rp.TeamSide == "" {
				continue
			}
			inner[rp.SteamID] = rp.TeamSide
		}
		out[r.Number] = inner
	}
	return out
}

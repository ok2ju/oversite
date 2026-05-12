package contacts

import (
	"sort"
	"strings"

	"github.com/ok2ju/oversite/internal/demo"
)

// PartitionEventsByRound is the exported counterpart of
// partitionEventsByRound, exposed for use by
// internal/demo/contacts/detectors.
func PartitionEventsByRound(events []demo.GameEvent) map[int][]demo.GameEvent {
	return partitionEventsByRound(events)
}

// PartitionVisibilityByRound is the exported counterpart of
// partitionVisibilityByRound.
func PartitionVisibilityByRound(vis []demo.VisibilityChange) map[int][]demo.VisibilityChange {
	return partitionVisibilityByRound(vis)
}

// BuildEnemyTeam is the exported counterpart of buildEnemyTeam.
func BuildEnemyTeam(roster []demo.RoundParticipant) map[string]string {
	return buildEnemyTeam(roster)
}

// buildEnemyTeam returns a map from steam_id to team side ("CT"/"T")
// covering everyone in the round roster.
func buildEnemyTeam(roster []demo.RoundParticipant) map[string]string {
	out := make(map[string]string, len(roster))
	for _, p := range roster {
		if p.SteamID == "" {
			continue
		}
		out[p.SteamID] = p.TeamSide
	}
	return out
}

// partitionEventsByRound groups events by round number with a single
// linear pass. Round 0 (warmup) is omitted.
func partitionEventsByRound(events []demo.GameEvent) map[int][]demo.GameEvent {
	out := make(map[int][]demo.GameEvent, 32)
	for i := range events {
		r := events[i].RoundNumber
		if r == 0 {
			continue
		}
		out[r] = append(out[r], events[i])
	}
	return out
}

func partitionVisibilityByRound(vis []demo.VisibilityChange) map[int][]demo.VisibilityChange {
	out := make(map[int][]demo.VisibilityChange, 32)
	for i := range vis {
		r := vis[i].RoundNumber
		if r == 0 {
			continue
		}
		out[r] = append(out[r], vis[i])
	}
	return out
}

// partitionTeammateFlashesByRound finds player_flashed events where the
// attacker and victim are on the same team and pulls out a lightweight
// record per (round, victim) so the orchestrator can later attach the
// extras.teammate_flashed_during flag to contacts.
//
// teammateFlashByVictim is keyed by (round_number, victim_steam_id) so
// the orchestrator can restrict the list to the current subject without
// re-scanning all events.
type teammateFlashByVictim map[int]map[string][]TeammateFlash

func partitionTeammateFlashesByRound(events []demo.GameEvent, rounds []demo.RoundData) teammateFlashByVictim {
	// Build a (round_number, steam_id) -> team_side map so we can detect
	// same-team flashes without leaning on extras team strings.
	teamByRoundSteam := make(map[int]map[string]string, len(rounds))
	for _, r := range rounds {
		m := make(map[string]string, len(r.Roster))
		for _, p := range r.Roster {
			if p.SteamID == "" {
				continue
			}
			m[p.SteamID] = p.TeamSide
		}
		teamByRoundSteam[r.Number] = m
	}

	out := make(teammateFlashByVictim, len(rounds))
	for i := range events {
		evt := &events[i]
		if evt.Type != "player_flashed" {
			continue
		}
		if evt.AttackerSteamID == "" || evt.VictimSteamID == "" {
			continue
		}
		if evt.AttackerSteamID == evt.VictimSteamID {
			continue
		}
		fe, ok := evt.ExtraData.(*demo.PlayerFlashedExtra)
		if !ok || fe == nil {
			continue
		}
		teams := teamByRoundSteam[evt.RoundNumber]
		if teams == nil {
			continue
		}
		attackerTeam := teams[evt.AttackerSteamID]
		victimTeam := teams[evt.VictimSteamID]
		if attackerTeam == "" || victimTeam == "" || attackerTeam != victimTeam {
			continue
		}
		if _, exists := out[evt.RoundNumber]; !exists {
			out[evt.RoundNumber] = make(map[string][]TeammateFlash, 4)
		}
		out[evt.RoundNumber][evt.VictimSteamID] = append(
			out[evt.RoundNumber][evt.VictimSteamID],
			TeammateFlash{Tick: int32(evt.Tick), DurationSecs: fe.DurationSecs},
		)
	}
	return out
}

// filterTeammateFlashesForSubject returns the teammate flashes whose
// victim is the subject. Other roster entries are filtered out at
// partition time, so this is now a direct lookup.
func filterTeammateFlashesForSubject(perRound teammateFlashByVictim, round int, subject string) []TeammateFlash {
	bySubject, ok := perRound[round]
	if !ok {
		return nil
	}
	return bySubject[subject]
}

// buildAlivePerTick returns a closure per round that, given a tick,
// returns the list of steam IDs alive at-or-before that tick. Used by
// the outcome classifier's trade look-ahead (aliveAtTick).
func buildAlivePerTick(result *demo.ParseResult) map[int]func(int32) []string {
	if result == nil {
		return map[int]func(int32) []string{}
	}
	out := make(map[int]func(int32) []string, len(result.Rounds))

	// Build, per round, a sorted slice of (tick, steam, alive) triples.
	type sample struct {
		tick    int32
		steamID string
		alive   bool
	}
	perRound := make(map[int][]sample, len(result.Rounds))
	for _, t := range result.AnalysisTicks {
		// Match against round windows.
		rnd := roundForTick(int(t.Tick), result.Rounds)
		if rnd == 0 {
			continue
		}
		perRound[rnd] = append(perRound[rnd], sample{
			tick:    t.Tick,
			steamID: steamUintToString(t.SteamID),
			alive:   t.IsAlive,
		})
	}

	for rnd, slice := range perRound {
		sort.Slice(slice, func(i, j int) bool {
			if slice[i].tick != slice[j].tick {
				return slice[i].tick < slice[j].tick
			}
			return slice[i].steamID < slice[j].steamID
		})
		captured := slice
		out[rnd] = func(tick int32) []string {
			// Last-known-state-per-steam-at-or-before tick.
			latest := map[string]bool{}
			for _, s := range captured {
				if s.tick > tick {
					break
				}
				latest[s.steamID] = s.alive
			}
			alive := make([]string, 0, len(latest))
			for steam, a := range latest {
				if a {
					alive = append(alive, steam)
				}
			}
			sort.Strings(alive)
			return alive
		}
	}
	return out
}

// roundForTick locates the round a tick falls into. Returns 0 when no
// round contains the tick (e.g. between rounds or warmup).
func roundForTick(tick int, rounds []demo.RoundData) int {
	for _, r := range rounds {
		if r.Number == 0 {
			continue
		}
		end := r.EndTick
		if end == 0 {
			end = tick
		}
		if tick >= r.StartTick && tick <= end {
			return r.Number
		}
	}
	return 0
}

func steamUintToString(id uint64) string {
	if id == 0 {
		return ""
	}
	const digits = "0123456789"
	buf := make([]byte, 0, 20)
	for id > 0 {
		buf = append(buf, digits[id%10])
		id /= 10
	}
	// reverse
	for i, j := 0, len(buf)-1; i < j; i, j = i+1, j-1 {
		buf[i], buf[j] = buf[j], buf[i]
	}
	return string(buf)
}

// deriveAliveRange computes when the subject is alive during the round.
// SpawnTick is the round's FreezeEndTick. DeathTick is the first kill
// where the subject is the victim, or 0 if the subject survives the
// round.
func deriveAliveRange(subject string, round demo.RoundData, events []demo.GameEvent) AliveRange {
	out := AliveRange{SpawnTick: int32(round.FreezeEndTick)}
	for i := range events {
		evt := &events[i]
		if evt.Type != "kill" {
			continue
		}
		if evt.VictimSteamID != subject {
			continue
		}
		out.DeathTick = int32(evt.Tick)
		break
	}
	return out
}

// isHumanSubject returns true if the roster participant is a real
// player. Bots in the project's parser-side roster typically have
// SteamID "0" or "" (see internal/demo/parser.go:shouldSkipPlayer). As
// a fallback, exclude names that look like bots ("BOT" prefix).
func isHumanSubject(p demo.RoundParticipant) bool {
	if p.SteamID == "" || p.SteamID == "0" {
		return false
	}
	if strings.HasPrefix(strings.ToUpper(p.PlayerName), "BOT ") || strings.HasPrefix(strings.ToUpper(p.PlayerName), "BOT_") {
		return false
	}
	return p.TeamSide == "CT" || p.TeamSide == "T"
}

// postWindowKills returns the kills in (tLast, min(tLast +
// TradeWindowTicks, roundEnd)] sorted by tick ascending.
func postWindowKills(events []demo.GameEvent, tLast int32, roundEnd int) []demo.GameEvent {
	upper := tLast + TradeWindowTicks
	if roundEnd > 0 && int32(roundEnd) < upper {
		upper = int32(roundEnd)
	}
	out := make([]demo.GameEvent, 0, 4)
	for i := range events {
		evt := events[i]
		if evt.Type != "kill" {
			continue
		}
		if int32(evt.Tick) <= tLast {
			continue
		}
		if int32(evt.Tick) > upper {
			continue
		}
		out = append(out, evt)
	}
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].Tick < out[j].Tick
	})
	return out
}

// aliveAtTick wraps the per-round closure into a steam_id-keyed set.
func aliveAtTick(perRound func(int32) []string, tick int32) map[string]bool {
	if perRound == nil {
		return map[string]bool{}
	}
	list := perRound(tick)
	out := make(map[string]bool, len(list))
	for _, s := range list {
		out[s] = true
	}
	return out
}

package demo

import (
	"math"
	"sort"
	"strings"
)

// RoundLoadoutEntry is one player's freeze-end inventory for a single round.
// Mirrors the shape stored in round_loadouts (see migration 011) and the
// main-package DTO of the same name. Defined here so the aggregator stays
// self-contained inside this package.
type RoundLoadoutEntry struct {
	SteamID   string
	Inventory string
}

// PlayerTickSample is a single tick-rate sample for one player. The
// aggregator only needs position, view angle, alive state, and the source
// tick — the full TickDatum is intentionally not pulled in to keep this
// package free of database imports.
type PlayerTickSample struct {
	Tick    int
	X, Y    float64
	Yaw     float64 // degrees
	IsAlive bool
}

// BombsiteCentroid is one bombsite's location for the time-on-bombsite stat.
//
// Phase 2 callers populate (X, Y) from BombPlanted event positions and the
// accumulator uses a fixed-radius bounding circle.
//
// Phase 3 callers (when the demo's map has hand-authored polygon data in
// callouts.go) additionally populate the MinX/MaxX/MinY/MaxY fields. When
// MinX < MaxX, the accumulator uses the polygon test instead of the circle —
// strictly more accurate without requiring a separate slice/argument.
type BombsiteCentroid struct {
	Site       string // "A" or "B"
	X, Y       float64
	MinX, MaxX float64
	MinY, MaxY float64
}

// hasPolygon reports whether the bombsite carries hand-authored polygon
// bounds; callers fall back to the bounding-circle proxy otherwise.
func (b BombsiteCentroid) hasPolygon() bool {
	return b.MinX < b.MaxX && b.MinY < b.MaxY
}

// containsPoint reports whether (x, y) is inside the bombsite. Uses the
// hand-authored polygon when available, falling back to the Phase 2
// bounding-circle proxy.
func (b BombsiteCentroid) containsPoint(x, y float64) bool {
	if b.hasPolygon() {
		return x >= b.MinX && x <= b.MaxX && y >= b.MinY && y <= b.MaxY
	}
	return pointWithinRadius(x, y, b.X, b.Y, timeOnSiteRadius)
}

// PlayerMatchStats is the deep-stats payload for a single player in a single
// demo. Computed on demand from already-ingested rounds + events + loadouts —
// no new ingest write path. See app.go::GetPlayerMatchStats and the right-side
// player panel in the viewer.
type PlayerMatchStats struct {
	SteamID          string
	PlayerName       string
	TeamSide         string // CT or T (first observed side; switch handled per-round)
	RoundsPlayed     int
	Kills            int
	Deaths           int
	Assists          int
	Damage           int
	HeadshotKills    int
	ClutchKills      int
	FirstKills       int
	FirstDeaths      int
	OpeningWins      int
	OpeningLosses    int
	TradeKills       int
	HSPercent        float64
	ADR              float64
	DamageByWeapon   []DamageByWeapon
	DamageByOpponent []DamageByOpponent
	Rounds           []PlayerRoundDetail
	Movement         MovementStats
	Timings          TimingStats
	Utility          UtilityStats
	HitGroups        []HitGroupBreakdown
}

// UtilityStats aggregates flash / smoke / he / molotov throws plus flash
// assists and total blind time inflicted for the match. Phase 3 stats —
// require the parser to capture PlayerFlashed events and grenade throws.
type UtilityStats struct {
	FlashesThrown          int
	SmokesThrown           int
	HEsThrown              int
	MolotovsThrown         int
	DecoysThrown           int
	FlashAssists           int     // KillExtra.FlashAssist === true on a kill credited to attacker
	BlindTimeInflictedSecs float64 // sum of PlayerFlashed durations on enemies thrown by this player
	EnemiesFlashed         int     // count of unique-tick enemies flashed by this player
}

// HitGroupBreakdown is one row in the damage-by-hit-group breakdown — the
// "where am I hitting" diagnostic on the Detail tab.
type HitGroupBreakdown struct {
	HitGroup int    // demoinfocs HitGroup byte
	Label    string // human-readable label (Head, Chest, Stomach, …)
	Damage   int
	Hits     int
}

// PlayerRoundDetail is the per-round breakdown for a single player.
type PlayerRoundDetail struct {
	RoundNumber           int
	TeamSide              string
	Kills                 int
	Deaths                int
	Assists               int
	Damage                int
	HeadshotKills         int
	ClutchKills           int
	FirstKill             bool
	FirstDeath            bool
	TradeKill             bool
	LoadoutValue          int      // sum of weapon prices from freeze-end inventory
	DistanceUnits         int      // distance traveled while alive (CS2 world units)
	AliveDurationSecs     float64  // seconds alive in this round
	TimeToFirstContactSec *float64 // nil if no contact in the round
}

// MovementStats captures the match-wide movement profile for a single player.
// Strafe percent is approximate (sample interval = parser default 4 ticks at
// 64 tps = 16 Hz) and is intended for trend-spotting, not as a competitive
// metric. See the panel tooltip.
type MovementStats struct {
	DistanceUnits int     // total distance traveled while alive
	AvgSpeedUps   float64 // distance / alive_seconds
	MaxSpeedUps   float64 // peak instantaneous speed, clamped to 260 u/s
	StrafePercent float64 // % of alive samples where velocity ≠ yaw forward (>60°)

	// Speed-bucket ratios (sum to 1.0 over alive samples). Stationary < 10 u/s,
	// walking < 130 u/s (CS2 walk-key cap), running ≥ 130 u/s.
	StationaryRatio float64
	WalkingRatio    float64
	RunningRatio    float64
}

// TimingStats captures the match-wide timing profile for a single player.
// "First contact" is the first kill or hurt event in a round in which the
// player was attacker or victim, measured from freeze-end-tick.
type TimingStats struct {
	AvgTimeToFirstContactSecs float64 // mean across rounds where contact occurred
	AvgAliveDurationSecs      float64 // mean alive duration per round
	TimeOnSiteASecs           float64 // alive sample seconds inside A-site bounding circle
	TimeOnSiteBSecs           float64 // alive sample seconds inside B-site bounding circle
}

// DamageByWeapon is one row in the damage-by-weapon breakdown.
type DamageByWeapon struct {
	Weapon string
	Damage int
}

// DamageByOpponent is one row in the damage-by-opponent breakdown.
type DamageByOpponent struct {
	SteamID    string
	PlayerName string
	TeamSide   string
	Damage     int
}

// TradeWindowSeconds defines the window in which a kill counts as a trade for
// a teammate's death. Exported so internal/demo/analysis can reuse the same
// threshold without drifting — see analysis.TradeWindowSeconds.
const TradeWindowSeconds = 5.0

// tradeWindowSeconds is retained as a package-internal alias so existing
// consumers in this file keep their tighter spelling. Both names refer to the
// same value; do not change one without the other.
const tradeWindowSeconds = TradeWindowSeconds

// weaponPrices is a hardcoded CS2 weapon → buy-menu price table. Used to
// estimate per-round loadout value from the round_loadouts inventory list.
// Items not in the table contribute zero (free items: knife, grenade defaults
// vary by round; we'd rather under-estimate than mis-attribute).
var weaponPrices = map[string]int{
	// Pistols
	"glock":         200,
	"glock-18":      200,
	"usp-s":         200,
	"hkp2000":       200,
	"p2000":         200,
	"p250":          300,
	"five-seven":    500,
	"fiveseven":     500,
	"tec-9":         500,
	"tec9":          500,
	"cz75-auto":     500,
	"cz75auto":      500,
	"cz75":          500,
	"dual berettas": 300,
	"dualberettas":  300,
	"elite":         300,
	"desert eagle":  700,
	"deagle":        700,
	"r8 revolver":   600,
	"revolver":      600,
	// SMGs
	"mac-10":   1050,
	"mac10":    1050,
	"mp9":      1250,
	"mp7":      1500,
	"mp5-sd":   1500,
	"mp5sd":    1500,
	"ump-45":   1200,
	"ump45":    1200,
	"p90":      2350,
	"pp-bizon": 1400,
	"bizon":    1400,
	// Shotguns
	"nova":      1050,
	"xm1014":    2000,
	"sawed-off": 1100,
	"sawedoff":  1100,
	"mag-7":     1300,
	"mag7":      1300,
	// Rifles
	"famas":    2050,
	"galil ar": 1800,
	"galilar":  1800,
	"galil":    1800,
	"m4a4":     3100,
	"m4a1":     3100,
	"m4a1-s":   2900,
	"m4a1s":    2900,
	"ak-47":    2700,
	"ak47":     2700,
	"sg 553":   3000,
	"sg553":    3000,
	"aug":      3300,
	// Snipers
	"ssg 08":  1700,
	"ssg08":   1700,
	"awp":     4750,
	"scar-20": 5000,
	"scar20":  5000,
	"g3sg1":   5000,
	// LMGs
	"m249":  5200,
	"negev": 1700,
	// Grenades
	"flashbang":    200,
	"hegrenade":    300,
	"smokegrenade": 300,
	"smoke":        300,
	"decoy":        50,
	"molotov":      400,
	"incgrenade":   600,
	"incendiary":   600,
	// Equipment / utility
	"taser":         200,
	"zeus":          200,
	"kevlar":        650,
	"vest":          650,
	"vesthelm":      1000,
	"kevlar+helmet": 1000,
	"defuser":       400,
	"defusekit":     400,
}

// ComputePlayerMatchStats aggregates a single player's deep stats across a
// match. It is a pure function over already-parsed rounds, events, the
// per-round loadouts map keyed by round_number → []RoundLoadoutEntry, and an
// optional per-player tick sample list (sorted by tick) used for the
// movement / timing breakdowns added in Phase 2.
//
// tickRate is the demo's tick-rate (ticks per second); used to convert the
// trade-kill window from seconds to ticks and to compute speeds from the
// sample stream. A non-positive tickRate falls back to 64.
//
// samples may be nil — the movement/timing fields are zero-valued when no
// tick data is available. bombsites holds the per-site centroids derived
// from BombPlanted events at the binding boundary; nil disables the
// time-on-site proxy.
//
// Reuses the cursor-walk pattern from CalculatePlayerRoundStats so the event
// slice is touched exactly twice (once per round contiguous range, once per
// kill within the trade-window pass).
func ComputePlayerMatchStats(
	rounds []RoundData,
	events []GameEvent,
	loadouts map[int][]RoundLoadoutEntry,
	samples []PlayerTickSample,
	bombsites []BombsiteCentroid,
	steamID string,
	tickRate float64,
) PlayerMatchStats {
	if tickRate <= 0 {
		tickRate = 64
	}

	out := PlayerMatchStats{SteamID: steamID}
	if steamID == "" || len(rounds) == 0 {
		return out
	}

	// Movement aggregator state: walked once for the entire match. Per-round
	// totals are stitched onto the corresponding PlayerRoundDetail below.
	movement := newMovementAccumulator(rounds, samples, bombsites, tickRate)

	damageByWeapon := make(map[string]int)
	damageByOpponent := make(map[string]*DamageByOpponent)

	// Phase 3 — utility / hit-group accumulators. Match-level only; the
	// round-tab utility card (Phase 3 frontend) reads these aggregates rather
	// than per-round detail to keep the panel compact.
	var utility UtilityStats
	hitGroupDamage := make(map[int]int)
	hitGroupHits := make(map[int]int)

	cursor := 0
	for _, rd := range rounds {
		// Skip events that precede this round (warmup leftovers).
		for cursor < len(events) && events[cursor].RoundNumber < rd.Number {
			cursor++
		}
		start := cursor
		for cursor < len(events) && events[cursor].RoundNumber == rd.Number {
			cursor++
		}
		roundEvents := events[start:cursor]

		// Reuse the existing CalculatePlayerRoundStats output shape — it
		// already does the cursor walk over killEvents, clutch detection, and
		// first-kill/first-death attribution.
		roundStats := calculateRound(rd.Roster, roundEvents)

		var ps *PlayerRoundStats
		var teamSide string
		for i := range roundStats {
			if roundStats[i].SteamID == steamID {
				ps = &roundStats[i]
				teamSide = roundStats[i].TeamSide
				break
			}
		}
		if teamSide == "" {
			// Player not on the roster this round — also missed by every
			// kill/hurt event. Skip the round entirely.
			continue
		}

		out.RoundsPlayed++
		if out.PlayerName == "" && ps != nil {
			out.PlayerName = ps.PlayerName
		}
		if out.TeamSide == "" {
			out.TeamSide = teamSide
		}

		detail := PlayerRoundDetail{
			RoundNumber: rd.Number,
			TeamSide:    teamSide,
		}
		if ps != nil {
			detail.Kills = ps.Kills
			detail.Deaths = ps.Deaths
			detail.Assists = ps.Assists
			detail.Damage = ps.Damage
			detail.HeadshotKills = ps.HeadshotKills
			detail.ClutchKills = ps.ClutchKills
			detail.FirstKill = ps.FirstKill
			detail.FirstDeath = ps.FirstDeath

			out.Kills += ps.Kills
			out.Deaths += ps.Deaths
			out.Assists += ps.Assists
			out.Damage += ps.Damage
			out.HeadshotKills += ps.HeadshotKills
			out.ClutchKills += ps.ClutchKills
			if ps.FirstKill {
				out.FirstKills++
				out.OpeningWins++
			}
			if ps.FirstDeath {
				out.FirstDeaths++
				out.OpeningLosses++
			}
		}

		// Damage-by-weapon and damage-by-opponent — walk PlayerHurt events
		// where the player is the attacker.
		opponents := make(map[string]string) // steamID → name (this round)
		opponentTeam := make(map[string]string)
		for _, ev := range roundEvents {
			if ev.Type != "player_hurt" {
				continue
			}
			if ev.AttackerSteamID != steamID {
				if h, _ := ev.ExtraData.(*PlayerHurtExtra); h != nil && ev.VictimSteamID == steamID {
					// We are the victim — track who hit us only for nameing
					// fallback (no aggregation here).
					if h.AttackerName != "" {
						opponents[ev.AttackerSteamID] = h.AttackerName
						opponentTeam[ev.AttackerSteamID] = h.AttackerTeam
					}
				}
				continue
			}
			h, _ := ev.ExtraData.(*PlayerHurtExtra)
			if h == nil {
				continue
			}
			weapon := strings.ToLower(strings.TrimSpace(ev.Weapon))
			if weapon == "" {
				weapon = "unknown"
			}
			damageByWeapon[weapon] += h.HealthDamage

			// Hit-group breakdown: only counts damage we dealt to opponents.
			// Friendly fire would skew the "where am I aiming" diagnostic.
			if h.VictimTeam != "" && h.VictimTeam != teamSide {
				hitGroupDamage[h.HitGroup] += h.HealthDamage
				hitGroupHits[h.HitGroup]++
			}

			if ev.VictimSteamID != "" {
				row, ok := damageByOpponent[ev.VictimSteamID]
				if !ok {
					row = &DamageByOpponent{
						SteamID:    ev.VictimSteamID,
						PlayerName: h.VictimName,
						TeamSide:   h.VictimTeam,
					}
					damageByOpponent[ev.VictimSteamID] = row
				}
				if row.PlayerName == "" && h.VictimName != "" {
					row.PlayerName = h.VictimName
				}
				if row.TeamSide == "" && h.VictimTeam != "" {
					row.TeamSide = h.VictimTeam
				}
				row.Damage += h.HealthDamage
			}
		}

		// Phase 3 utility accumulation: throws, flash assists, and blind time.
		// We walk roundEvents once for everything except trade-kills (kept
		// separate below to preserve the early-out semantics).
		for _, ev := range roundEvents {
			switch ev.Type {
			case "grenade_throw":
				if ev.AttackerSteamID != steamID {
					continue
				}
				switch normalizeGrenade(ev.Weapon) {
				case "flashbang":
					utility.FlashesThrown++
				case "smokegrenade":
					utility.SmokesThrown++
				case "hegrenade":
					utility.HEsThrown++
				case "molotov", "incgrenade":
					utility.MolotovsThrown++
				case "decoy":
					utility.DecoysThrown++
				}
			case "player_flashed":
				if ev.AttackerSteamID != steamID {
					continue
				}
				f, _ := ev.ExtraData.(*PlayerFlashedExtra)
				if f == nil {
					continue
				}
				// Only count enemy-flashed (team-mate flashes are not a
				// useful credit). Self-flashes naturally fall under the same
				// rule because the team matches.
				if f.VictimTeam != "" && f.VictimTeam != teamSide {
					utility.BlindTimeInflictedSecs += f.DurationSecs
					utility.EnemiesFlashed++
				}
			case "kill":
				if ev.AttackerSteamID != steamID {
					continue
				}
				k, _ := ev.ExtraData.(*KillExtra)
				if k != nil && k.FlashAssist {
					utility.FlashAssists++
				}
			}
		}

		// Trade-kill detection: did the player kill an enemy within
		// tradeWindowSeconds of a teammate's death? Walk roundEvents twice
		// here — small N (~10–30 kills/round) keeps this O(k²) cheap.
		tradeWindowTicks := int(tradeWindowSeconds * tickRate)
		var tradedThisRound bool
		for _, ev := range roundEvents {
			if ev.Type != "kill" || ev.AttackerSteamID != steamID {
				continue
			}
			isSelfKill := ev.AttackerSteamID != "" && ev.AttackerSteamID == ev.VictimSteamID
			if isSelfKill {
				continue
			}
			// Look back for a teammate death within the trade window.
			for j := len(roundEvents) - 1; j >= 0; j-- {
				prev := roundEvents[j]
				if prev.Tick > ev.Tick {
					continue
				}
				if ev.Tick-prev.Tick > tradeWindowTicks {
					break
				}
				if prev.Type != "kill" {
					continue
				}
				if prev.VictimSteamID == "" || prev.VictimSteamID == steamID {
					continue
				}
				k, _ := prev.ExtraData.(*KillExtra)
				if k == nil {
					continue
				}
				// Same team as the trader? Use this round's team side.
				if k.VictimTeam == teamSide {
					tradedThisRound = true
					out.TradeKills++
					break
				}
			}
			if tradedThisRound {
				break // count at most one trade-kill credit per round
			}
		}
		detail.TradeKill = tradedThisRound

		// Loadout value: sum prices of inventory items at freeze end.
		if entries, ok := loadouts[rd.Number]; ok {
			for _, le := range entries {
				if le.SteamID != steamID {
					continue
				}
				detail.LoadoutValue = sumLoadoutValue(le.Inventory)
				break
			}
		}

		// Time-to-first-contact: ticks between freeze_end_tick and the first
		// kill or hurt event involving this player, converted to seconds. The
		// roundEvents slice is already filtered to this round and ordered by
		// tick — first match wins.
		if rd.FreezeEndTick > 0 {
			for _, ev := range roundEvents {
				if ev.Type != "kill" && ev.Type != "player_hurt" {
					continue
				}
				if ev.AttackerSteamID != steamID && ev.VictimSteamID != steamID {
					continue
				}
				if ev.Tick < rd.FreezeEndTick {
					continue
				}
				secs := float64(ev.Tick-rd.FreezeEndTick) / tickRate
				detail.TimeToFirstContactSec = &secs
				break
			}
		}

		// Movement / alive-duration: pulled from the round-scoped accumulator
		// rather than re-walking the sample slice here.
		if r := movement.roundResult(rd.Number); r != nil {
			detail.DistanceUnits = r.distance
			detail.AliveDurationSecs = r.aliveSeconds
		}

		out.Rounds = append(out.Rounds, detail)
	}

	if out.Kills > 0 {
		out.HSPercent = float64(out.HeadshotKills) / float64(out.Kills) * 100
	}
	if out.RoundsPlayed > 0 {
		out.ADR = float64(out.Damage) / float64(out.RoundsPlayed)
	}

	out.DamageByWeapon = sortDamageByWeapon(damageByWeapon)
	out.DamageByOpponent = sortDamageByOpponent(damageByOpponent)
	out.Movement, out.Timings = movement.finalize(out.Rounds)
	out.Utility = utility
	out.HitGroups = sortHitGroups(hitGroupDamage, hitGroupHits)
	return out
}

// normalizeGrenade lowercases / squashes whitespace from the parser's grenade
// weapon string. demoinfocs may emit "Smoke Grenade" or "smokegrenade"
// depending on the demo and version — collapse both to a stable bucket key.
func normalizeGrenade(weapon string) string {
	w := strings.ToLower(weapon)
	w = strings.ReplaceAll(w, " ", "")
	w = strings.ReplaceAll(w, "-", "")
	switch w {
	case "smokegrenade", "smoke":
		return "smokegrenade"
	case "hegrenade", "he":
		return "hegrenade"
	case "flashbang", "flash":
		return "flashbang"
	case "incendiarygrenade", "incgrenade", "incendiary":
		return "incgrenade"
	case "molotov":
		return "molotov"
	case "decoygrenade", "decoy":
		return "decoy"
	}
	return w
}

// hitGroupLabel maps the demoinfocs HitGroup byte to a human-readable label.
// Out-of-range / unmapped values fall back to "Other" so the panel never
// renders a blank row.
func hitGroupLabel(hg int) string {
	switch hg {
	case 0:
		return "Generic"
	case 1:
		return "Head"
	case 2:
		return "Chest"
	case 3:
		return "Stomach"
	case 4:
		return "Left Arm"
	case 5:
		return "Right Arm"
	case 6:
		return "Left Leg"
	case 7:
		return "Right Leg"
	case 8:
		return "Neck"
	case 10:
		return "Gear"
	}
	return "Other"
}

func sortHitGroups(damage map[int]int, hits map[int]int) []HitGroupBreakdown {
	out := make([]HitGroupBreakdown, 0, len(damage))
	for hg, d := range damage {
		out = append(out, HitGroupBreakdown{
			HitGroup: hg,
			Label:    hitGroupLabel(hg),
			Damage:   d,
			Hits:     hits[hg],
		})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Damage != out[j].Damage {
			return out[i].Damage > out[j].Damage
		}
		return out[i].HitGroup < out[j].HitGroup
	})
	return out
}

// sumLoadoutValue parses a comma-separated inventory list (encodeInventory
// output) and returns the sum of weapon prices from weaponPrices. Items not
// in the price table contribute zero.
func sumLoadoutValue(inventory string) int {
	if inventory == "" {
		return 0
	}
	total := 0
	for _, raw := range strings.Split(inventory, ",") {
		w := strings.ToLower(strings.TrimSpace(raw))
		if w == "" {
			continue
		}
		if price, ok := weaponPrices[w]; ok {
			total += price
		}
	}
	return total
}

func sortDamageByWeapon(m map[string]int) []DamageByWeapon {
	out := make([]DamageByWeapon, 0, len(m))
	for w, d := range m {
		out = append(out, DamageByWeapon{Weapon: w, Damage: d})
	}
	// Sort by damage desc, then weapon asc for stable golden output.
	for i := 1; i < len(out); i++ {
		for j := i; j > 0; j-- {
			if out[j].Damage > out[j-1].Damage ||
				(out[j].Damage == out[j-1].Damage && out[j].Weapon < out[j-1].Weapon) {
				out[j], out[j-1] = out[j-1], out[j]
			} else {
				break
			}
		}
	}
	return out
}

func sortDamageByOpponent(m map[string]*DamageByOpponent) []DamageByOpponent {
	out := make([]DamageByOpponent, 0, len(m))
	for _, row := range m {
		out = append(out, *row)
	}
	for i := 1; i < len(out); i++ {
		for j := i; j > 0; j-- {
			if out[j].Damage > out[j-1].Damage ||
				(out[j].Damage == out[j-1].Damage && out[j].SteamID < out[j-1].SteamID) {
				out[j], out[j-1] = out[j-1], out[j]
			} else {
				break
			}
		}
	}
	return out
}

// Phase 2 — movement / timing aggregator -------------------------------------

// movementMaxSpeed clamps instantaneous samples to a sane upper bound. CS2's
// stock player run-speed cap is ~250 u/s; values past 260 are almost always
// teleports between disconnected positions (round reset, observer slot warp).
const movementMaxSpeed = 260.0

// strafeMinSpeed is the minimum speed (u/s) at which the velocity-vs-yaw
// check is considered. Below this threshold the velocity vector's direction
// is dominated by integer-position quantization noise and would falsely
// register as strafing for stationary players.
const strafeMinSpeed = 60.0

// strafeAngleThresholdDeg flags samples where the velocity vector deviates
// from the player's yaw forward by more than this much. 60° is generous —
// pure W-key forward movement is well below it; A/D-key strafes typically
// land at 80–90°.
const strafeAngleThresholdDeg = 60.0

// speedBucketWalk and speedBucketRun mark the boundaries used by the speed
// histogram. Walking is below CS2's normal-run cap minus a small margin.
const (
	speedBucketStationary = 10.0
	speedBucketWalk       = 130.0
)

// timeOnSiteRadius is the bounding-circle radius (CS2 world units) used by
// the Phase 2 time-on-site proxy. Replaced by per-map polygons in Phase 3
// (see callouts.go).
const timeOnSiteRadius = 800.0

// roundMovement holds the per-round accumulated movement totals returned to
// the main aggregator via roundResult.
type roundMovement struct {
	distance     int
	aliveSeconds float64
}

// movementAccumulator walks the per-player tick sample slice once and
// distributes per-tick deltas to the round it belongs to (using start_tick /
// end_tick boundaries). Match-level totals (max speed, strafe %, speed
// buckets, time-on-site) are computed during the same pass.
type movementAccumulator struct {
	rounds      []RoundData
	bombsiteA   *BombsiteCentroid
	bombsiteB   *BombsiteCentroid
	sampleHz    float64 // tickRate / 4 (parser default sample interval)
	roundTotals map[int]*roundMovement

	// Match-level running totals.
	totalDistance     float64
	totalAliveSamples int
	maxSpeed          float64
	strafeSamples     int
	movingSamples     int
	stationarySamples int
	walkingSamples    int
	runningSamples    int
	siteASamples      int
	siteBSamples      int
}

// newMovementAccumulator preprocesses the sample slice and returns an
// accumulator ready to be queried via roundResult and finalize. samples must
// be sorted by tick — the database query orders by tick already.
func newMovementAccumulator(rounds []RoundData, samples []PlayerTickSample, bombsites []BombsiteCentroid, tickRate float64) *movementAccumulator {
	acc := &movementAccumulator{
		rounds:      rounds,
		sampleHz:    tickRate / 4,
		roundTotals: make(map[int]*roundMovement, len(rounds)),
	}
	if acc.sampleHz <= 0 {
		acc.sampleHz = 16
	}
	for i := range bombsites {
		switch strings.ToUpper(bombsites[i].Site) {
		case "A":
			site := bombsites[i]
			acc.bombsiteA = &site
		case "B":
			site := bombsites[i]
			acc.bombsiteB = &site
		}
	}
	if len(samples) == 0 {
		return acc
	}

	// Walk samples once and bucket each into its round. We use a cursor over
	// the sorted rounds slice so the per-sample lookup is amortized O(1).
	roundOrdered := make([]RoundData, len(rounds))
	copy(roundOrdered, rounds)
	sort.Slice(roundOrdered, func(i, j int) bool {
		return roundOrdered[i].StartTick < roundOrdered[j].StartTick
	})
	cursor := 0

	prevByRound := make(map[int]*PlayerTickSample, len(rounds))

	for i := range samples {
		s := samples[i]

		// Advance cursor to the round whose [StartTick, EndTick] window
		// contains this sample. Samples between rounds (freeze time, end-of-
		// round delays) are skipped — they wouldn't carry meaningful movement
		// signal anyway.
		for cursor < len(roundOrdered) && roundOrdered[cursor].EndTick > 0 && s.Tick > roundOrdered[cursor].EndTick {
			cursor++
		}
		if cursor >= len(roundOrdered) {
			break
		}
		rd := roundOrdered[cursor]
		if s.Tick < rd.StartTick {
			continue
		}

		rt, ok := acc.roundTotals[rd.Number]
		if !ok {
			rt = &roundMovement{}
			acc.roundTotals[rd.Number] = rt
		}

		if s.IsAlive {
			rt.aliveSeconds += 1.0 / acc.sampleHz
			acc.totalAliveSamples++

			// Site-presence check: polygon when the bombsite carries hand-
			// authored bounds (Phase 3), else the bounding-circle proxy.
			if acc.bombsiteA != nil && acc.bombsiteA.containsPoint(s.X, s.Y) {
				acc.siteASamples++
			}
			if acc.bombsiteB != nil && acc.bombsiteB.containsPoint(s.X, s.Y) {
				acc.siteBSamples++
			}
		}

		// Inter-sample delta. We pair against the previous sample for the SAME
		// round only — round transitions and respawns produce position jumps
		// that would otherwise inflate distance and trip the max-speed clamp.
		if prev, ok := prevByRound[rd.Number]; ok && prev.IsAlive && s.IsAlive {
			dx := s.X - prev.X
			dy := s.Y - prev.Y
			dist := math.Sqrt(dx*dx + dy*dy)
			dt := float64(s.Tick-prev.Tick) / (acc.sampleHz * 4) // ticks → seconds; sample interval = 4 ticks
			// dt above intentionally derives from raw ticks rather than the
			// constant 1/sampleHz to handle the off-by-one case where the
			// parser drops a sample (e.g. round-reset jitter).
			if dt > 0 {
				speed := dist / dt
				if speed <= movementMaxSpeed {
					rt.distance += int(math.Round(dist))
					acc.totalDistance += dist
					if speed > acc.maxSpeed {
						acc.maxSpeed = speed
					}
					switch {
					case speed < speedBucketStationary:
						acc.stationarySamples++
					case speed < speedBucketWalk:
						acc.walkingSamples++
					default:
						acc.runningSamples++
					}

					if speed >= strafeMinSpeed {
						acc.movingSamples++
						if isStrafingSample(dx, dy, s.Yaw) {
							acc.strafeSamples++
						}
					}
				}
			}
		}

		// Snapshot the sample for the next iteration of this round. We allocate
		// once per round and overwrite to avoid per-sample allocs.
		if prevByRound[rd.Number] == nil {
			snap := s
			prevByRound[rd.Number] = &snap
		} else {
			*prevByRound[rd.Number] = s
		}
	}
	return acc
}

func (a *movementAccumulator) roundResult(roundNumber int) *roundMovement {
	if a == nil {
		return nil
	}
	return a.roundTotals[roundNumber]
}

// finalize converts the running totals into the public MovementStats /
// TimingStats shapes. roundsDetail is used to compute the timing averages.
func (a *movementAccumulator) finalize(roundsDetail []PlayerRoundDetail) (MovementStats, TimingStats) {
	var movement MovementStats
	var timings TimingStats

	if a == nil {
		return movement, timings
	}

	movement.DistanceUnits = int(math.Round(a.totalDistance))
	movement.MaxSpeedUps = a.maxSpeed
	if a.totalAliveSamples > 0 && a.sampleHz > 0 {
		aliveSeconds := float64(a.totalAliveSamples) / a.sampleHz
		if aliveSeconds > 0 {
			movement.AvgSpeedUps = a.totalDistance / aliveSeconds
		}
		bucketTotal := a.stationarySamples + a.walkingSamples + a.runningSamples
		if bucketTotal > 0 {
			movement.StationaryRatio = float64(a.stationarySamples) / float64(bucketTotal)
			movement.WalkingRatio = float64(a.walkingSamples) / float64(bucketTotal)
			movement.RunningRatio = float64(a.runningSamples) / float64(bucketTotal)
		}
		timings.TimeOnSiteASecs = float64(a.siteASamples) / a.sampleHz
		timings.TimeOnSiteBSecs = float64(a.siteBSamples) / a.sampleHz
	}
	if a.movingSamples > 0 {
		movement.StrafePercent = float64(a.strafeSamples) / float64(a.movingSamples) * 100
	}

	// Timings averages from the per-round detail.
	var contactTotal float64
	var contactCount int
	var aliveTotal float64
	for _, r := range roundsDetail {
		if r.TimeToFirstContactSec != nil {
			contactTotal += *r.TimeToFirstContactSec
			contactCount++
		}
		aliveTotal += r.AliveDurationSecs
	}
	if contactCount > 0 {
		timings.AvgTimeToFirstContactSecs = contactTotal / float64(contactCount)
	}
	if len(roundsDetail) > 0 {
		timings.AvgAliveDurationSecs = aliveTotal / float64(len(roundsDetail))
	}
	return movement, timings
}

// isStrafingSample reports whether the velocity vector (dx, dy) deviates from
// the yaw-forward heading by more than strafeAngleThresholdDeg. yaw is the
// player's view yaw in degrees (CS2 convention: 0° = +X, increasing CCW).
func isStrafingSample(dx, dy, yawDeg float64) bool {
	speed := math.Sqrt(dx*dx + dy*dy)
	if speed < 1e-6 {
		return false
	}
	yawRad := yawDeg * math.Pi / 180
	fx := math.Cos(yawRad)
	fy := math.Sin(yawRad)
	cosTheta := (dx*fx + dy*fy) / speed // unit-vector dot
	if cosTheta > 1 {
		cosTheta = 1
	} else if cosTheta < -1 {
		cosTheta = -1
	}
	angleDeg := math.Acos(cosTheta) * 180 / math.Pi
	return angleDeg > strafeAngleThresholdDeg
}

// pointWithinRadius is the bounding-circle hit-test for the Phase 2 site
// proxy. Replaced by polygon intersection in Phase 3.
func pointWithinRadius(x, y, cx, cy, r float64) bool {
	dx := x - cx
	dy := y - cy
	return dx*dx+dy*dy <= r*r
}

// BombsiteCentroidsFromEvents derives per-site (x, y) centroids from the
// BombPlanted events in the demo. Used by the binding layer to feed the
// Phase 2 time-on-site proxy until per-map polygons land in Phase 3.
//
// Sites with no plants in the match are omitted from the result; the time-
// on-site fields then evaluate to zero, which is the correct fallback (we
// cannot prove the player was on a site we never saw planted).
func BombsiteCentroidsFromEvents(events []GameEvent) []BombsiteCentroid {
	type running struct {
		sumX, sumY float64
		count      int
	}
	bySite := map[string]*running{}
	for _, ev := range events {
		if ev.Type != "bomb_plant" {
			continue
		}
		bp, _ := ev.ExtraData.(*BombPlantExtra)
		var site string
		if bp != nil {
			site = strings.ToUpper(strings.TrimSpace(bp.Site))
		}
		if site != "A" && site != "B" {
			continue
		}
		r, ok := bySite[site]
		if !ok {
			r = &running{}
			bySite[site] = r
		}
		r.sumX += ev.X
		r.sumY += ev.Y
		r.count++
	}
	out := make([]BombsiteCentroid, 0, len(bySite))
	for _, site := range []string{"A", "B"} {
		r, ok := bySite[site]
		if !ok || r.count == 0 {
			continue
		}
		out = append(out, BombsiteCentroid{
			Site: site,
			X:    r.sumX / float64(r.count),
			Y:    r.sumY / float64(r.count),
		})
	}
	return out
}

package detectors

// Entry describes one detector's static metadata. Order in the
// Registered slice defines stable emit order (within a phase) so
// per-contact mistake lists are deterministic regardless of map
// iteration order.
type Entry struct {
	Kind           string   // e.g. "slow_reaction"
	Category       string   // "aim" | "movement" | "positioning" | "utility" | "trade" | "spray"
	Severity       int      // 1=low, 2=medium, 3=high
	Phase          Phase    // PhasePre / PhaseDuring / PhasePost
	Func           Detector // the implementation; nil for v2-deferred rows
	CrossRound     bool     // analysis §9.5 — needs cross-round context
	V2Priority     bool     // explicit "defer to v2" flag distinct from CrossRound
	WriteAggregate bool     // also write an analysis_mistakes row (round-level)
	Note           string   // one-line reason; surfaces in tests
}

// Registered is the list of every detector — v1 and v2 placeholders.
// runner.go iterates this slice in order. Entries with Func == nil are
// skipped (v2 placeholders) but stay listed for traceability.
//
// Populated in init() rather than as a var initializer to break the
// static cycle between detector funcs (which call NewContactMistake →
// lookupCatalogEntry → Registered) and Registered's value.
var Registered []Entry

func init() {
	Registered = []Entry{
		// --- pre-engagement (analysis §4.1) ---
		{Kind: "slow_reaction", Category: "aim", Severity: 2, Phase: PhasePre, Func: SlowReaction, WriteAggregate: true, Note: "first weapon_fire by P after TFirst > 250ms"},
		{Kind: "missed_first_shot", Category: "spray", Severity: 2, Phase: PhasePre, Func: MissedFirstShot, WriteAggregate: true, Note: "first subject-aggressor weapon_fire has no hit_victim_steam_id and distance < 1500u"},
		{Kind: "isolated_peek", Category: "positioning", Severity: 3, Phase: PhasePre, Func: IsolatedPeek, WriteAggregate: true, Note: "no teammate within 800u of P at TFirst"},
		{Kind: "bad_crosshair_height", Category: "aim", Severity: 2, Phase: PhasePre, Func: BadCrosshairHeight, Note: "pitch at TFirst off head-height by >5°"},
		{Kind: "peek_while_reloading", Category: "aim", Severity: 3, Phase: PhasePre, Func: PeekWhileReloading, Note: "ammo_clip at TFirst < 30% of weapon max"},

		{Kind: "wide_peek_when_close", Category: "movement", Severity: 2, Phase: PhasePre, V2Priority: true, Note: "v2: distance<600u + wide swing in last 0.5s"},
		{Kind: "running_into_fight", Category: "movement", Severity: 2, Phase: PhasePre, V2Priority: true, Note: "v2: velocity > walk threshold in last 0.5s pre-shot"},
		{Kind: "no_utility_used", Category: "utility", Severity: 2, Phase: PhasePre, V2Priority: true, Note: "v2: needs grenade-event lookup over 5s pre-window"},
		{Kind: "pre_aimed_wrong_angle", Category: "aim", Severity: 2, Phase: PhasePre, V2Priority: true, Note: "v2: P yaw > 30° off enemy direction 0.3s before TFirst"},
		{Kind: "walked_into_known_angle", Category: "positioning", Severity: 2, Phase: PhasePre, CrossRound: true, V2Priority: true, Note: "v2: cross-round heatmap"},

		// --- during-engagement (analysis §4.2) ---
		{Kind: "shot_while_moving", Category: "movement", Severity: 2, Phase: PhaseDuring, Func: ShotWhileMoving, WriteAggregate: true, Note: "velocity > 110 u/s at any subject-aggressor weapon_fire tick"},
		{Kind: "aim_while_flashed", Category: "aim", Severity: 2, Phase: PhaseDuring, Func: AimWhileFlashed, Note: "P fired while subject-victim flash duration active"},
		{Kind: "lost_hp_advantage", Category: "trade", Severity: 3, Phase: PhaseDuring, Func: LostHPAdvantage, Note: "at TFirst P.hp - E.hp > 25 but P dies in contact"},

		{Kind: "spray_decay", Category: "spray", Severity: 2, Phase: PhaseDuring, V2Priority: true, Note: "v2: hit-rate decay after first hit"},
		{Kind: "crosshair_height_drift", Category: "aim", Severity: 1, Phase: PhaseDuring, V2Priority: true, Note: "v2: pitch RMS dev during fire > 4°"},
		{Kind: "wasted_advantage_no_kill", Category: "aim", Severity: 2, Phase: PhaseDuring, V2Priority: true, Note: "v2: spotted-only lead >=0.3s without conversion"},
		{Kind: "wallbang_taken_no_repos", Category: "positioning", Severity: 2, Phase: PhaseDuring, V2Priority: true, Note: "v2: wallbang flag + <75u movement in 1s"},

		// --- post-engagement (analysis §4.3) ---
		{Kind: "no_reposition_after_kill", Category: "positioning", Severity: 3, Phase: PhasePost, Func: NoRepositionAfterKill, Note: "kill of E_i + <75u movement in 1.5s + another enemy alive within 1500u"},
		{Kind: "no_reload_with_cover", Category: "utility", Severity: 2, Phase: PhasePost, Func: NoReloadWithCover, Note: "ammo_clip<30% + no spotted enemy in next 1.5s + no reload started"},

		{Kind: "swung_for_third", Category: "positioning", Severity: 3, Phase: PhasePost, V2Priority: true, Note: "v2: won 1v2, pushed onto third, died — needs next-contact lookup"},
	}
}

// V1 returns the v1 detectors (Func != nil). runner.go uses this list,
// not Registered, in production. Tests can iterate Registered to assert
// v2 placeholders stay in place.
func V1() []Entry {
	out := make([]Entry, 0, len(Registered))
	for _, e := range Registered {
		if e.Func != nil {
			out = append(out, e)
		}
	}
	return out
}

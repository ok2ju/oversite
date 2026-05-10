package analysis

// AnalysisVersion is the persisted analyzer schema version. Bump when a rule's
// math changes substantively or a new rule lands so the frontend can detect
// stale rows and offer a RecomputeAnalysis. Stored in the
// player_match_analysis.version column. The bump cadence is intentionally
// coarse — adding a metric to extras_json doesn't require a bump; changing the
// crosshair-too-low pitch tolerance does.
const AnalysisVersion = 1

// Category names a coarse rule grouping. The frontend renders one card per
// category; the same category names also drive the side-panel filter chips.
type Category string

const (
	CategoryAim         Category = "aim"
	CategorySpray       Category = "spray"
	CategoryMovement    Category = "movement"
	CategoryUtility     Category = "utility"
	CategoryPositioning Category = "positioning"
	CategoryTrade       Category = "trade"
	CategoryEconomy     Category = "economy"
	CategoryRound       Category = "round"
)

// Severity is the 1–3 weight surfaced to the UI. 1 = informational nudge,
// 2 = standard mistake, 3 = high-impact (e.g. died holding util on the
// last man standing). Persisted into analysis_mistakes.severity.
type Severity int

const (
	SeverityLow  Severity = 1
	SeverityMed  Severity = 2
	SeverityHigh Severity = 3
)

// Template is the static metadata for a mistake kind: which category it
// belongs to, how severe it is, and the human-facing title / coaching
// suggestion that the side panel and detail card render. Templates live on
// the Go side so a future locale switch lands in one place; the bindings
// expose the resolved strings on each MistakeEntry.
type Template struct {
	Category   Category
	Severity   Severity
	Title      string
	Suggestion string
}

// templates maps every persisted MistakeKind to its presentation metadata.
// Adding a new kind requires adding a row here so the panel doesn't fall back
// to the kind string. The fallback (TemplateForKind below) covers Go-only
// rules that haven't been taught to the frontend yet.
var templates = map[string]Template{
	string(MistakeKindNoTradeDeath): {
		Category:   CategoryTrade,
		Severity:   SeverityMed,
		Title:      "Untraded death",
		Suggestion: "Hold the angle your teammate just lost — even one extra trade per half lifts T-side win rate.",
	},
	string(MistakeKindDiedWithUtilUnused): {
		Category:   CategoryUtility,
		Severity:   SeverityHigh,
		Title:      "Died with utility unused",
		Suggestion: "Throw your nades on the way to your hold — dying with util is dropping free damage.",
	},
	string(MistakeKindCrosshairTooLow): {
		Category:   CategoryAim,
		Severity:   SeverityLow,
		Title:      "Crosshair too low",
		Suggestion: "Pre-aim head level on every common angle — flicking down loses more time than checking high.",
	},
	string(MistakeKindShotWhileMoving): {
		Category:   CategoryMovement,
		Severity:   SeverityMed,
		Title:      "Shot while moving",
		Suggestion: "Counter-strafe before firing — even small drift past 30 u/s drops your first-bullet accuracy.",
	},
	string(MistakeKindSlowReaction): {
		Category:   CategoryAim,
		Severity:   SeverityMed,
		Title:      "Slow reaction",
		Suggestion: "Pre-aim the angle and pre-fire on first sound — when you start the engagement reactive you're already losing.",
	},
	string(MistakeKindMissedFlick): {
		Category:   CategoryAim,
		Severity:   SeverityLow,
		Title:      "Missed flick",
		Suggestion: "Stop swinging through the angle — flicks past 30° land far less often than a held pre-aim.",
	},
	string(MistakeKindMissedFirstShot): {
		Category:   CategorySpray,
		Severity:   SeverityMed,
		Title:      "Missed first shot",
		Suggestion: "Tap, don't spray, on the opener — your first bullet sits in the most accurate cone.",
	},
	string(MistakeKindSprayDecay): {
		Category:   CategorySpray,
		Severity:   SeverityMed,
		Title:      "Spray decay",
		Suggestion: "Burst-fire past shot 5 — recoil control is harder than just stopping and re-tapping.",
	},
	string(MistakeKindNoCounterStrafe): {
		Category:   CategoryMovement,
		Severity:   SeverityMed,
		Title:      "No counter-strafe",
		Suggestion: "Tap the opposite key for one tick before firing — without a stop, even a rifle shoots like an SMG.",
	},
	string(MistakeKindUnusedSmoke): {
		Category:   CategoryUtility,
		Severity:   SeverityLow,
		Title:      "Unused smoke",
		Suggestion: "Pair smokes with a teammate's push — solo lineups without follow-up rarely win the round.",
	},
	string(MistakeKindSurvivedWithUtil): {
		Category:   CategoryUtility,
		Severity:   SeverityMed,
		Title:      "Survived with utility unused",
		Suggestion: "Drop your last flash for the 2nd bombsite hit — saving util for the next round only helps if you live.",
	},
	string(MistakeKindIsolatedPeek): {
		Category:   CategoryPositioning,
		Severity:   SeverityHigh,
		Title:      "Isolated peek",
		Suggestion: "Wait for a teammate within 600 u — peeking alone trades your life for almost nothing.",
	},
	string(MistakeKindRepeatedDeathZone): {
		Category:   CategoryPositioning,
		Severity:   SeverityMed,
		Title:      "Repeated death zone",
		Suggestion: "You died in this spot 3+ times — switch the position or add util support before peeking again.",
	},
	string(MistakeKindWalkedIntoMolotov): {
		Category:   CategoryUtility,
		Severity:   SeverityLow,
		Title:      "Walked into molotov",
		Suggestion: "Listen for the inferno tick before pushing — running through fire is 50–80 free damage every time.",
	},
	string(MistakeKindEcoMisbuy): {
		Category:   CategoryEconomy,
		Severity:   SeverityLow,
		Title:      "Eco misbuy",
		Suggestion: "Force-buy when the enemy is also broke — both sides eco'ing is a free round you're refusing to take.",
	},
	string(MistakeKindCaughtReloading): {
		Category:   CategoryAim,
		Severity:   SeverityHigh,
		Title:      "Caught reloading",
		Suggestion: "Reload behind cover, never on the angle — finish the engagement, break LoS, then top up your clip.",
	},
	string(MistakeKindFlashAssist): {
		Category:   CategoryUtility,
		Severity:   SeverityLow,
		Title:      "Flash assist",
		Suggestion: "Keep blinding the angle your teammate is about to peek — your flashes are setting up their kills.",
	},
	string(MistakeKindHeDamage): {
		Category:   CategoryUtility,
		Severity:   SeverityLow,
		Title:      "HE damage",
		Suggestion: "Stack HEs on stacked enemies — one well-placed grenade can soften 3 players for the next push.",
	},
}

// TemplateForKind returns the presentation metadata for a kind. Unknown kinds
// (Go-only rules that haven't been taught to the frontend yet) get a neutral
// fallback so the panel still renders the row.
func TemplateForKind(kind string) Template {
	if t, ok := templates[kind]; ok {
		return t
	}
	return Template{
		Category:   CategoryRound,
		Severity:   SeverityLow,
		Title:      kind,
		Suggestion: "",
	}
}

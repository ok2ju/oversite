package analysis

// AnalysisVersion is the persisted analyzer schema version. Bump when a rule's
// math changes substantively or a new rule lands so the frontend can detect
// stale rows and offer a RecomputeAnalysis. Stored in the
// player_match_analysis.version column. The bump cadence is intentionally
// coarse — adding a metric to extras_json doesn't require a bump; changing the
// crosshair-too-low pitch tolerance does.
const AnalysisVersion = 2

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
	// WhyItHurts is a one-sentence, plain-English explanation of the cost
	// of this mistake — used by the mistake-detail subtitle (P2-3) and the
	// coaching errors-strip (P5-4). Distinct from Suggestion: WhyItHurts
	// names the damage; Suggestion names the fix.
	WhyItHurts string
}

// templates maps every persisted MistakeKind to its presentation metadata.
// Adding a new kind requires adding a row here so the panel doesn't fall back
// to the kind string. The fallback (TemplateForKind below) covers Go-only
// rules that haven't been taught to the frontend yet.
var templates = map[string]Template{
	string(MistakeKindShotWhileMoving): {
		Category:   CategoryMovement,
		Severity:   SeverityMed,
		Title:      "Shot while moving",
		Suggestion: "Counter-strafe before firing — even small drift past 30 u/s drops your first-bullet accuracy.",
		WhyItHurts: "First-bullet accuracy collapses past ~25 u/s of drift, so the duel is decided before the spray.",
	},
	string(MistakeKindSlowReaction): {
		Category:   CategoryAim,
		Severity:   SeverityMed,
		Title:      "Slow reaction",
		Suggestion: "Pre-aim the angle and pre-fire on first sound — when you start the engagement reactive you're already losing.",
		WhyItHurts: "If you fire 100 ms after the enemy, you've already eaten the bullet that decides the duel.",
	},
	string(MistakeKindMissedFirstShot): {
		Category:   CategorySpray,
		Severity:   SeverityMed,
		Title:      "Missed first shot",
		Suggestion: "Tap, don't spray, on the opener — your first bullet sits in the most accurate cone.",
		WhyItHurts: "The first bullet is your most accurate one — miss it and you're spraying into recoil to recover.",
	},
	string(MistakeKindSprayDecay): {
		Category:   CategorySpray,
		Severity:   SeverityMed,
		Title:      "Spray decay",
		Suggestion: "Burst-fire past shot 5 — recoil control is harder than just stopping and re-tapping.",
		WhyItHurts: "Past shot 5 the cone is so wide most bullets miss — you're just feeding ammo into a wall.",
	},
	string(MistakeKindNoCounterStrafe): {
		Category:   CategoryMovement,
		Severity:   SeverityMed,
		Title:      "No counter-strafe",
		Suggestion: "Tap the opposite key for one tick before firing — without a stop, even a rifle shoots like an SMG.",
		WhyItHurts: "Without a counter-strafe your rifle's first-bullet cone is closer to a deagle's than a tap kill.",
	},
	string(MistakeKindIsolatedPeek): {
		Category:   CategoryPositioning,
		Severity:   SeverityHigh,
		Title:      "Isolated peek",
		Suggestion: "Wait for a teammate within 600 u — peeking alone trades your life for almost nothing.",
		WhyItHurts: "Without a trade nearby, your death is a free pick — the enemy gets the kill and the position.",
	},
	string(MistakeKindRepeatedDeathZone): {
		Category:   CategoryPositioning,
		Severity:   SeverityMed,
		Title:      "Repeated death zone",
		Suggestion: "You died in this spot 3+ times — switch the position or add util support before peeking again.",
		WhyItHurts: "The enemy has read this position — every repeat peek is a duel you're starting at a disadvantage.",
	},
	string(MistakeKindEcoMisbuy): {
		Category:   CategoryEconomy,
		Severity:   SeverityLow,
		Title:      "Eco misbuy",
		Suggestion: "Force-buy when the enemy is also broke — both sides eco'ing is a free round you're refusing to take.",
		WhyItHurts: "Saving when the enemy is also poor concedes a round you could have stolen with pistols.",
	},
	string(MistakeKindCaughtReloading): {
		Category:   CategoryAim,
		Severity:   SeverityHigh,
		Title:      "Caught reloading",
		Suggestion: "Reload behind cover, never on the angle — finish the engagement, break LoS, then top up your clip.",
		WhyItHurts: "You can't shoot back. Whoever swung the angle gets a free kill.",
	},
	string(MistakeKindFlashAssist): {
		Category:   CategoryUtility,
		Severity:   SeverityLow,
		Title:      "Flash assist",
		Suggestion: "Keep blinding the angle your teammate is about to peek — your flashes are setting up their kills.",
		WhyItHurts: "A good flash hands your teammate a free duel — losing the habit costs your team easy openers.",
	},
	string(MistakeKindHeDamage): {
		Category:   CategoryUtility,
		Severity:   SeverityLow,
		Title:      "HE damage",
		Suggestion: "Stack HEs on stacked enemies — one well-placed grenade can soften 3 players for the next push.",
		WhyItHurts: "Skipped HE damage is HP your team has to take from rifles instead — every chip shot matters.",
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

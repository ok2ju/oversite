package analysis

// HabitKey is a stable identifier for a coachable habit/metric. Persisted into
// HabitReport rows as a string so the frontend can switch on the value without
// duplicating the catalog. Adding a habit means adding a constant here, an
// entry in the Norms() table, and (if used in HabitReport) a builder mapping
// in habits.go.
type HabitKey string

// Habit identifiers. Strings match §6.1 of docs/.../analysis-overhaul.md.
const (
	HabitCounterStrafe      HabitKey = "counter_strafe"
	HabitReaction           HabitKey = "reaction"
	HabitFirstShotAcc       HabitKey = "first_shot_acc"
	HabitShootingInMotion   HabitKey = "shooting_in_motion"
	HabitCrouchBeforeShot   HabitKey = "crouch_before_shot"
	HabitFlickBalance       HabitKey = "flick_balance"
	HabitTradeTiming        HabitKey = "trade_timing"
	HabitUtilityUsed        HabitKey = "utility_used"
	HabitIsolatedPeekDeaths HabitKey = "isolated_peek_deaths"
	HabitRepeatedDeathZone  HabitKey = "repeated_death_zone"
)

// Direction indicates how a metric's value relates to "better".
type Direction string

const (
	// LowerIsBetter — smaller values are better (e.g. reaction time ms).
	// GoodThreshold/WarnThreshold are upper bounds: value <= GoodThreshold ⇒ good.
	LowerIsBetter Direction = "lower"
	// HigherIsBetter — larger values are better (e.g. first-shot accuracy %).
	// GoodThreshold/WarnThreshold are lower bounds: value >= GoodThreshold ⇒ good.
	HigherIsBetter Direction = "higher"
	// Balanced — value should sit inside a target band centred on an ideal.
	// Uses GoodMin/GoodMax (good band) and WarnMin/WarnMax (warn band) instead
	// of GoodThreshold/WarnThreshold.
	Balanced Direction = "balanced"
)

// Status is the tri-state classification surfaced to the UI as a coloured pill.
type Status string

const (
	StatusGood Status = "good"
	StatusWarn Status = "warn"
	StatusBad  Status = "bad"
)

// Norm describes how to classify a single habit metric. The catalog returned
// by Norms() is the single source of truth: the analyzer ships the Status with
// each HabitRow, so the frontend never re-implements the thresholds.
type Norm struct {
	Key         HabitKey
	Label       string
	Description string
	Unit        string
	Direction   Direction

	// Thresholds used when Direction is LowerIsBetter or HigherIsBetter.
	GoodThreshold float64
	WarnThreshold float64

	// Bands used when Direction is Balanced. Values inside [GoodMin, GoodMax]
	// classify as good; values inside [WarnMin, WarnMax] (and outside the good
	// band) classify as warn; everything else is bad. Both bands must be
	// non-empty when Direction == Balanced.
	GoodMin float64
	GoodMax float64
	WarnMin float64
	WarnMax float64
}

// ClassifyHabit returns the Status for a value against the given norm.
// Behaviour by direction:
//
//   - LowerIsBetter:  value <= GoodThreshold ⇒ good; value <= WarnThreshold ⇒ warn; else bad.
//   - HigherIsBetter: value >= GoodThreshold ⇒ good; value >= WarnThreshold ⇒ warn; else bad.
//   - Balanced:       in [GoodMin,GoodMax] ⇒ good; in [WarnMin,WarnMax] ⇒ warn; else bad.
//
// An unknown direction conservatively returns StatusBad — the caller will
// surface the value as "no norm classifier" and the row stays clickable.
func ClassifyHabit(value float64, norm Norm) Status {
	switch norm.Direction {
	case LowerIsBetter:
		if value <= norm.GoodThreshold {
			return StatusGood
		}
		if value <= norm.WarnThreshold {
			return StatusWarn
		}
		return StatusBad
	case HigherIsBetter:
		if value >= norm.GoodThreshold {
			return StatusGood
		}
		if value >= norm.WarnThreshold {
			return StatusWarn
		}
		return StatusBad
	case Balanced:
		if value >= norm.GoodMin && value <= norm.GoodMax {
			return StatusGood
		}
		if value >= norm.WarnMin && value <= norm.WarnMax {
			return StatusWarn
		}
		return StatusBad
	}
	return StatusBad
}

// normCatalog is the per-habit metadata table. Source of truth for labels,
// units, descriptions, and thresholds. See docs/.../analysis-overhaul.md §6.1.
var normCatalog = map[HabitKey]Norm{
	HabitCounterStrafe: {
		Key:           HabitCounterStrafe,
		Label:         "Counter-strafe",
		Description:   "Stop before firing so the first bullet lands.",
		Unit:          "ms",
		Direction:     LowerIsBetter,
		GoodThreshold: 100,
		WarnThreshold: 200,
	},
	HabitReaction: {
		Key:           HabitReaction,
		Label:         "Reaction",
		Description:   "Time from seeing the enemy to firing.",
		Unit:          "ms",
		Direction:     LowerIsBetter,
		GoodThreshold: 200,
		WarnThreshold: 280,
	},
	HabitFirstShotAcc: {
		Key:           HabitFirstShotAcc,
		Label:         "First-shot accuracy",
		Description:   "Hits on the bullet that decides duels.",
		Unit:          "%",
		Direction:     HigherIsBetter,
		GoodThreshold: 50,
		WarnThreshold: 35,
	},
	HabitShootingInMotion: {
		Key:           HabitShootingInMotion,
		Label:         "Shooting in motion",
		Description:   "Share of shots fired while moving — wastes accuracy.",
		Unit:          "%",
		Direction:     LowerIsBetter,
		GoodThreshold: 12,
		WarnThreshold: 20,
	},
	HabitCrouchBeforeShot: {
		Key:           HabitCrouchBeforeShot,
		Label:         "Crouch before shot",
		Description:   "Crouching before firing freezes you on the angle.",
		Unit:          "%",
		Direction:     LowerIsBetter,
		GoodThreshold: 5,
		WarnThreshold: 10,
	},
	HabitFlickBalance: {
		Key:         HabitFlickBalance,
		Label:       "Flick balance",
		Description: "Over- vs under-flicks — balanced means your sensitivity matches your aim.",
		Unit:        "%",
		Direction:   Balanced,
		GoodMin:     45,
		GoodMax:     55,
		WarnMin:     40,
		WarnMax:     60,
	},
	HabitTradeTiming: {
		Key:           HabitTradeTiming,
		Label:         "Trade timing",
		Description:   "Share of teammates' deaths you traded.",
		Unit:          "%",
		Direction:     HigherIsBetter,
		GoodThreshold: 70,
		WarnThreshold: 50,
	},
	HabitUtilityUsed: {
		Key:           HabitUtilityUsed,
		Label:         "Utility used",
		Description:   "Share of your nades thrown before the round ended.",
		Unit:          "%",
		Direction:     HigherIsBetter,
		GoodThreshold: 75,
		WarnThreshold: 50,
	},
	HabitIsolatedPeekDeaths: {
		Key:           HabitIsolatedPeekDeaths,
		Label:         "Isolated peek deaths",
		Description:   "Deaths with no teammate within trade range.",
		Unit:          "",
		Direction:     LowerIsBetter,
		GoodThreshold: 0,
		WarnThreshold: 2,
	},
	HabitRepeatedDeathZone: {
		Key:           HabitRepeatedDeathZone,
		Label:         "Repeated death zones",
		Description:   "Areas where you keep dying — re-think the angle.",
		Unit:          "",
		Direction:     LowerIsBetter,
		GoodThreshold: 0,
		WarnThreshold: 1,
	},
}

// LookupNorm returns the Norm for a habit and ok=true, or a zero Norm and
// ok=false if the key is unknown.
func LookupNorm(key HabitKey) (Norm, bool) {
	n, ok := normCatalog[key]
	return n, ok
}

// AllHabitKeys returns every habit in the catalog in a stable, deliberate
// order — the order the in-app habit checklist renders. The first six are the
// "micro" habits (also surfaced on the coaching landing page); the remaining
// five are match-shape habits that only appear on the in-app checklist.
func AllHabitKeys() []HabitKey {
	return []HabitKey{
		HabitCounterStrafe,
		HabitReaction,
		HabitFirstShotAcc,
		HabitShootingInMotion,
		HabitCrouchBeforeShot,
		HabitFlickBalance,
		HabitTradeTiming,
		HabitUtilityUsed,
		HabitIsolatedPeekDeaths,
		HabitRepeatedDeathZone,
	}
}

package analysis

// Drill prescriptions for the "Next drill" card on the analysis page (P1-3).
// The catalog is intentionally small and deterministic — one habit → one
// drill → one duration. The card picks a single drill so insight converts
// into practice; we do not list five drills and let the player pick.
//
// See docs/.../analysis-overhaul.md §6.2 for the source table. Adding a
// drill means appending a Drill to drillCatalog (the slice order doubles as
// the impact rank — higher in the slice = higher priority on ties).

// Drill describes a single training prescription. Why is filled at pick
// time from the matching Norm.Description so the catalog stays in sync
// with the habit copy without duplicating it.
type Drill struct {
	Key      HabitKey
	Title    string
	Why      string
	Duration string
	Chips    []string
}

// drillCatalog is the priority-ordered list of habit-targeted drills. The
// first habit in this slice wins on ties — counter-strafe is the most
// damaging miss to fix, so it leads. A habit not in this slice has no
// drill and will not be picked even when classified bad/warn; the
// maintenance drill takes over only when no listed habit is bad/warn.
var drillCatalog = []Drill{
	{
		Key:      HabitCounterStrafe,
		Title:    `Aim Lab — "Strafe Aim" routine`,
		Duration: "10 min",
		Chips:    []string{"counter-strafe", "1 habit"},
	},
	{
		Key:      HabitFirstShotAcc,
		Title:    "DM aim_botz — first-bullet, no spray",
		Duration: "10–15 min",
		Chips:    []string{"no spray", "1 habit"},
	},
	{
		Key:      HabitShootingInMotion,
		Title:    "DM aim_botz — strafe + stop, AK only",
		Duration: "10 min",
		Chips:    []string{"strafe + stop", "1 habit"},
	},
	{
		Key:      HabitReaction,
		Title:    `Aim Lab — "Reflex Tracking"`,
		Duration: "8 min",
		Chips:    []string{"reaction", "1 habit"},
	},
	{
		Key:      HabitCrouchBeforeShot,
		Title:    "Awareness drill — rebind crouch off mouse",
		Duration: "5 min",
		Chips:    []string{"awareness", "1 habit"},
	},
	{
		Key:      HabitFlickBalance,
		Title:    "Aim Lab — Microflex + sens calibration",
		Duration: "15 min",
		Chips:    []string{"sensitivity", "1 habit"},
	},
	{
		Key:      HabitTradeTiming,
		Title:    "Watch your last 5 untraded deaths in the viewer",
		Duration: "10 min",
		Chips:    []string{"trade timing", "1 habit"},
	},
}

// MaintenanceDrill is the fallback drill returned when nothing in the
// catalog is classified bad/warn. The Key is empty by design — callers
// rendering "you're hitting your norms" copy should branch on Key == "".
var MaintenanceDrill = Drill{
	Title:    "Light warmup — keep your routine",
	Why:      "You're hitting your norms. Keep the muscle memory warm.",
	Duration: "5 min",
	Chips:    []string{"warmup", "maintenance"},
}

// PickNextDrill returns the drill prescribed for the worst-status habit in
// rows. Priority: bad first, then warn; ties broken by impact rank (the
// position in drillCatalog, lower index wins). Habits not in drillCatalog
// are ignored — they're surfaced in the habit checklist but don't drive
// the drill card. Returns MaintenanceDrill when no listed habit is
// bad/warn (or when rows is empty).
//
// Why text is pulled from the habit's Norm.Description at pick time so the
// drill catalog never duplicates the habit copy. A habit whose Norm has
// gone missing falls back to an empty Why; the frontend hides the row.
func PickNextDrill(rows []HabitRow) Drill {
	byKey := make(map[HabitKey]HabitRow, len(rows))
	for _, r := range rows {
		byKey[r.Key] = r
	}
	for _, target := range []Status{StatusBad, StatusWarn} {
		for _, drill := range drillCatalog {
			row, ok := byKey[drill.Key]
			if !ok || row.Status != target {
				continue
			}
			out := drill
			if n, ok := LookupNorm(drill.Key); ok {
				out.Why = n.Description
			}
			return out
		}
	}
	return MaintenanceDrill
}

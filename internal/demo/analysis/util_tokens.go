package analysis

import "strings"

// parseUtilFromInventory returns the deduplicated set of normalized utility
// tokens present in a freeze-end inventory string. Inventory tokens come from
// encodeInventory (Equipment.String() joined by ","); we normalize via
// lowercase + stripping whitespace and hyphens before comparing against the
// same normalization applied to grenade_throw.Weapon. Non-utility entries
// (rifles, kits, kevlar, …) are dropped.
func parseUtilFromInventory(inv string) []string {
	if inv == "" {
		return nil
	}
	seen := make(map[string]struct{}, 4)
	out := make([]string, 0, 4)
	for _, raw := range strings.Split(inv, ",") {
		tok := normalizeUtilToken(raw)
		if !isUtilToken(tok) {
			continue
		}
		if _, dup := seen[tok]; dup {
			continue
		}
		seen[tok] = struct{}{}
		out = append(out, tok)
	}
	return out
}

// normalizeUtilToken collapses Equipment.String()-style names to a stable key
// shared between inventory entries and grenade_throw.Weapon. Both surfaces use
// the same demoinfocs Equipment.String() output, so the only divergence we
// need to absorb is whitespace and hyphenation drift across versions.
func normalizeUtilToken(raw string) string {
	s := strings.ToLower(strings.TrimSpace(raw))
	s = strings.ReplaceAll(s, " ", "")
	s = strings.ReplaceAll(s, "-", "")
	return s
}

// utilTokenSet enumerates the normalized grenade types we consider for the
// rule. Aliases ("incgrenade" / "incendiarygrenade", "decoy" / "decoygrenade")
// cover Equipment.String() drift across demoinfocs versions.
var utilTokenSet = map[string]struct{}{
	"hegrenade":         {},
	"flashbang":         {},
	"smokegrenade":      {},
	"molotov":           {},
	"incgrenade":        {},
	"incendiarygrenade": {},
	"decoy":             {},
	"decoygrenade":      {},
}

func isUtilToken(tok string) bool {
	if tok == "" {
		return false
	}
	_, ok := utilTokenSet[tok]
	return ok
}

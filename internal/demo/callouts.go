package demo

import "fmt"

// calloutRegion defines a named rectangular area on a CS2 map in world-space coordinates.
// MinZ/MaxZ are optional: when MinZ < MaxZ, the Z coordinate is checked too.
// This disambiguates vertically stacked regions (e.g. Nuke A/B sites).
type calloutRegion struct {
	Name       string
	MinX, MaxX float64
	MinY, MaxY float64
	MinZ, MaxZ float64
}

// mapCallouts maps CS2 map names to their callout regions.
var mapCallouts = map[string][]calloutRegion{
	"de_dust2": {
		{Name: "T Spawn", MinX: -700, MaxX: 200, MinY: -1100, MaxY: -200},
		{Name: "CT Spawn", MinX: -600, MaxX: 200, MinY: 2400, MaxY: 3200},
		{Name: "A Long", MinX: 800, MaxX: 1600, MinY: 200, MaxY: 1500},
		{Name: "A Site", MinX: 800, MaxX: 1500, MinY: 1800, MaxY: 2800},
		{Name: "A Short", MinX: -100, MaxX: 600, MinY: 1400, MaxY: 2200},
		{Name: "A Ramp", MinX: 700, MaxX: 1300, MinY: 1400, MaxY: 1800},
		{Name: "Mid", MinX: -400, MaxX: 400, MinY: 200, MaxY: 1400},
		{Name: "Mid Doors", MinX: -600, MaxX: -200, MinY: 500, MaxY: 1200},
		{Name: "B Tunnels", MinX: -1800, MaxX: -900, MinY: -400, MaxY: 600},
		{Name: "B Site", MinX: -1800, MaxX: -800, MinY: 1200, MaxY: 2400},
		{Name: "B Window", MinX: -600, MaxX: -200, MinY: 1400, MaxY: 2000},
		{Name: "Pit", MinX: 1400, MaxX: 1900, MinY: 2400, MaxY: 3000},
		{Name: "Long Doors", MinX: 800, MaxX: 1600, MinY: -400, MaxY: 200},
		{Name: "Upper Tunnels", MinX: -900, MaxX: -200, MinY: -400, MaxY: 200},
		{Name: "Outside Long", MinX: 200, MaxX: 800, MinY: -600, MaxY: 200},
	},
	"de_mirage": {
		{Name: "T Spawn", MinX: 800, MaxX: 1600, MinY: -3100, MaxY: -2200},
		{Name: "CT Spawn", MinX: -400, MaxX: 400, MinY: 100, MaxY: 800},
		{Name: "A Site", MinX: -900, MaxX: 0, MinY: -1200, MaxY: -200},
		{Name: "A Ramp", MinX: -400, MaxX: 200, MinY: -2000, MaxY: -1200},
		{Name: "A Palace", MinX: -1600, MaxX: -900, MinY: -1600, MaxY: -800},
		{Name: "Mid", MinX: -200, MaxX: 600, MinY: -2200, MaxY: -1200},
		{Name: "Top Mid", MinX: -200, MaxX: 400, MinY: -1400, MaxY: -600},
		{Name: "B Site", MinX: -2200, MaxX: -1300, MinY: -200, MaxY: 600},
		{Name: "B Apartments", MinX: -600, MaxX: 200, MinY: -3000, MaxY: -2200},
		{Name: "B Short", MinX: -1300, MaxX: -600, MinY: -800, MaxY: -200},
		{Name: "Connector", MinX: -900, MaxX: -200, MinY: -800, MaxY: -200},
		{Name: "Jungle", MinX: -1200, MaxX: -600, MinY: -1400, MaxY: -600},
		{Name: "Window", MinX: -500, MaxX: 0, MinY: -800, MaxY: -200},
		{Name: "Underpass", MinX: -200, MaxX: 600, MinY: -1000, MaxY: -400},
		{Name: "Catwalk", MinX: -1600, MaxX: -900, MinY: -600, MaxY: 0},
	},
	"de_inferno": {
		{Name: "T Spawn", MinX: 900, MaxX: 1700, MinY: -200, MaxY: 600},
		{Name: "CT Spawn", MinX: -1200, MaxX: -400, MinY: 2200, MaxY: 3000},
		{Name: "A Site", MinX: -200, MaxX: 600, MinY: 2000, MaxY: 2800},
		{Name: "A Long", MinX: 600, MaxX: 1400, MinY: 1200, MaxY: 2200},
		{Name: "A Short", MinX: -200, MaxX: 400, MinY: 1200, MaxY: 2000},
		{Name: "Apartments", MinX: -200, MaxX: 800, MinY: 400, MaxY: 1200},
		{Name: "Mid", MinX: 200, MaxX: 1000, MinY: 600, MaxY: 1400},
		{Name: "Banana", MinX: -600, MaxX: 200, MinY: 200, MaxY: 1200},
		{Name: "B Site", MinX: -1000, MaxX: -200, MinY: 600, MaxY: 1600},
		{Name: "Pit", MinX: 600, MaxX: 1200, MinY: 2200, MaxY: 2800},
		{Name: "Library", MinX: -600, MaxX: 0, MinY: 2400, MaxY: 3000},
		{Name: "Arch", MinX: -400, MaxX: 200, MinY: 1600, MaxY: 2200},
		{Name: "Second Mid", MinX: 600, MaxX: 1200, MinY: 800, MaxY: 1400},
		{Name: "Top Banana", MinX: -400, MaxX: 200, MinY: 1000, MaxY: 1400},
		{Name: "Construction", MinX: -1200, MaxX: -600, MinY: 800, MaxY: 1600},
	},
	"de_nuke": {
		{Name: "T Spawn", MinX: -800, MaxX: 0, MinY: -1400, MaxY: -600},
		{Name: "CT Spawn", MinX: -200, MaxX: 600, MinY: 600, MaxY: 1200},
		{Name: "A Site", MinX: -600, MaxX: 400, MinY: -200, MaxY: 600, MinZ: -550, MaxZ: 0},
		{Name: "B Site", MinX: -600, MaxX: 400, MinY: -200, MaxY: 600, MinZ: -900, MaxZ: -550},
		{Name: "Outside", MinX: 400, MaxX: 1600, MinY: -1200, MaxY: 200},
		{Name: "Ramp", MinX: -800, MaxX: 0, MinY: -600, MaxY: 200},
		{Name: "Lobby", MinX: -400, MaxX: 400, MinY: -800, MaxY: -200},
		{Name: "Hut", MinX: -400, MaxX: 0, MinY: 0, MaxY: 400},
		{Name: "Heaven", MinX: -200, MaxX: 400, MinY: 200, MaxY: 600},
		{Name: "Vent", MinX: 0, MaxX: 400, MinY: -400, MaxY: 200},
		{Name: "Secret", MinX: 400, MaxX: 1000, MinY: 0, MaxY: 600},
		{Name: "Squeaky", MinX: -600, MaxX: -200, MinY: -400, MaxY: 0},
		{Name: "Yard", MinX: 200, MaxX: 1000, MinY: -800, MaxY: -200},
		{Name: "Garage", MinX: 400, MaxX: 1200, MinY: 200, MaxY: 800},
		{Name: "Silo", MinX: 600, MaxX: 1200, MinY: -400, MaxY: 200},
	},
	"de_overpass": {
		{Name: "T Spawn", MinX: -2200, MaxX: -1400, MinY: -800, MaxY: 200},
		{Name: "CT Spawn", MinX: -400, MaxX: 400, MinY: 200, MaxY: 1000},
		{Name: "A Site", MinX: -800, MaxX: 0, MinY: -800, MaxY: 0},
		{Name: "A Long", MinX: -1600, MaxX: -800, MinY: -800, MaxY: 0},
		{Name: "B Site", MinX: -600, MaxX: 200, MinY: 600, MaxY: 1400},
		{Name: "B Short", MinX: -600, MaxX: 200, MinY: 200, MaxY: 600},
		{Name: "B Long", MinX: -1800, MaxX: -800, MinY: 400, MaxY: 1200},
		{Name: "Connector", MinX: -800, MaxX: -200, MinY: -200, MaxY: 400},
		{Name: "Restrooms", MinX: -400, MaxX: 200, MinY: -400, MaxY: 200},
		{Name: "Mid", MinX: -1200, MaxX: -400, MinY: -400, MaxY: 400},
		{Name: "Monster", MinX: -1400, MaxX: -600, MinY: 600, MaxY: 1200},
		{Name: "Water", MinX: -200, MaxX: 400, MinY: 1000, MaxY: 1600},
		{Name: "Playground", MinX: -1200, MaxX: -600, MinY: -200, MaxY: 400},
		{Name: "Bank", MinX: -400, MaxX: 200, MinY: -200, MaxY: 200},
		{Name: "Fountain", MinX: -600, MaxX: 0, MinY: 200, MaxY: 600},
	},
	"de_anubis": {
		{Name: "T Spawn", MinX: -200, MaxX: 600, MinY: -2600, MaxY: -1800},
		{Name: "CT Spawn", MinX: -200, MaxX: 600, MinY: 400, MaxY: 1200},
		{Name: "A Site", MinX: -1200, MaxX: -400, MinY: -400, MaxY: 400},
		{Name: "A Main", MinX: -1200, MaxX: -400, MinY: -1400, MaxY: -400},
		{Name: "A Connector", MinX: -600, MaxX: 0, MinY: -800, MaxY: 0},
		{Name: "Mid", MinX: -200, MaxX: 400, MinY: -1600, MaxY: -600},
		{Name: "B Site", MinX: 400, MaxX: 1200, MinY: -400, MaxY: 400},
		{Name: "B Main", MinX: 600, MaxX: 1400, MinY: -1600, MaxY: -600},
		{Name: "B Connector", MinX: 200, MaxX: 600, MinY: -800, MaxY: 0},
		{Name: "Canal", MinX: -200, MaxX: 400, MinY: -600, MaxY: 200},
		{Name: "Bridge", MinX: -200, MaxX: 400, MinY: -200, MaxY: 400},
		{Name: "Palace", MinX: -1400, MaxX: -600, MinY: -200, MaxY: 600},
		{Name: "Ruins", MinX: 600, MaxX: 1400, MinY: -200, MaxY: 600},
		{Name: "Street", MinX: 200, MaxX: 800, MinY: -2000, MaxY: -1200},
		{Name: "Alley", MinX: -800, MaxX: -200, MinY: -2000, MaxY: -1200},
	},
	"de_ancient": {
		{Name: "T Spawn", MinX: -200, MaxX: 600, MinY: -2200, MaxY: -1400},
		{Name: "CT Spawn", MinX: -200, MaxX: 600, MinY: 600, MaxY: 1400},
		{Name: "A Site", MinX: -800, MaxX: 0, MinY: -200, MaxY: 600},
		{Name: "A Main", MinX: -800, MaxX: 0, MinY: -1200, MaxY: -200},
		{Name: "Mid", MinX: -200, MaxX: 400, MinY: -1400, MaxY: -400},
		{Name: "B Site", MinX: 200, MaxX: 1000, MinY: -200, MaxY: 600},
		{Name: "B Ramp", MinX: 400, MaxX: 1000, MinY: -1000, MaxY: -200},
		{Name: "Cave", MinX: -600, MaxX: 0, MinY: -600, MaxY: 0},
		{Name: "Elbow", MinX: -1000, MaxX: -400, MinY: -600, MaxY: 200},
		{Name: "Donut", MinX: -200, MaxX: 400, MinY: -400, MaxY: 200},
		{Name: "Side Path", MinX: 400, MaxX: 1000, MinY: -600, MaxY: 0},
		{Name: "Temple", MinX: -400, MaxX: 200, MinY: 0, MaxY: 600},
		{Name: "Jaguar", MinX: -800, MaxX: -200, MinY: 200, MaxY: 800},
		{Name: "House", MinX: 200, MaxX: 800, MinY: 200, MaxY: 800},
		{Name: "Water", MinX: 0, MaxX: 600, MinY: -1800, MaxY: -1200},
	},
}

// resolveCallout returns the named callout for a position on a given map.
// If no matching region is found, returns a coordinate string "(x, y)".
func resolveCallout(mapName string, x, y, z float64) string {
	regions, ok := mapCallouts[mapName]
	if !ok {
		return fmt.Sprintf("(%.0f, %.0f)", x, y)
	}
	for _, r := range regions {
		if x >= r.MinX && x <= r.MaxX && y >= r.MinY && y <= r.MaxY {
			if r.MinZ < r.MaxZ && (z < r.MinZ || z > r.MaxZ) {
				continue
			}
			return r.Name
		}
	}
	return fmt.Sprintf("(%.0f, %.0f)", x, y)
}

// bombsitePolygon is a single A or B site polygon for the time-on-bombsite
// stat. Phase 3 — replaces the centroid bounding-circle proxy used in
// Phase 2. Polygons are rectangular (axis-aligned) here to keep the
// hand-authored data tractable; if a future map needs an irregular shape we
// can switch to a convex-hull check without touching the call site.
type bombsitePolygon struct {
	Site       string // "A" or "B"
	MinX, MaxX float64
	MinY, MaxY float64
}

// mapBombsites lists per-map A / B site rectangles in CS2 world coordinates.
// Sourced from radar overlays and verified against bomb-plant positions in
// the project's reference demos. Add new maps here as the supported list
// grows.
var mapBombsites = map[string][]bombsitePolygon{
	"de_dust2": {
		{Site: "A", MinX: 800, MaxX: 1500, MinY: 1800, MaxY: 2800},
		{Site: "B", MinX: -1800, MaxX: -800, MinY: 1200, MaxY: 2400},
	},
	"de_mirage": {
		{Site: "A", MinX: -900, MaxX: 0, MinY: -1200, MaxY: -200},
		{Site: "B", MinX: -2200, MaxX: -1300, MinY: -200, MaxY: 600},
	},
	"de_inferno": {
		{Site: "A", MinX: -200, MaxX: 600, MinY: 2000, MaxY: 2800},
		{Site: "B", MinX: -1000, MaxX: -200, MinY: 600, MaxY: 1600},
	},
	"de_nuke": {
		{Site: "A", MinX: -600, MaxX: 400, MinY: -200, MaxY: 600},
		{Site: "B", MinX: -600, MaxX: 400, MinY: -200, MaxY: 600},
	},
	"de_overpass": {
		{Site: "A", MinX: -800, MaxX: 0, MinY: -800, MaxY: 0},
		{Site: "B", MinX: -600, MaxX: 200, MinY: 600, MaxY: 1400},
	},
	"de_anubis": {
		{Site: "A", MinX: -1200, MaxX: -400, MinY: -400, MaxY: 400},
		{Site: "B", MinX: 400, MaxX: 1200, MinY: -400, MaxY: 400},
	},
	"de_ancient": {
		{Site: "A", MinX: -800, MaxX: 0, MinY: -200, MaxY: 600},
		{Site: "B", MinX: 200, MaxX: 1000, MinY: -200, MaxY: 600},
	},
}

// BombsitePolygonsForMap returns the per-site polygon list for a given CS2
// map, or nil if the map is unknown. Used by the Phase 3 time-on-site
// computation; callers fall back to the Phase 2 centroid proxy when nil.
func BombsitePolygonsForMap(mapName string) []SitePolygon {
	regions, ok := mapBombsites[mapName]
	if !ok {
		return nil
	}
	out := make([]SitePolygon, len(regions))
	for i, r := range regions {
		out[i] = SitePolygon(r)
	}
	return out
}

// SitePolygon is the exported axis-aligned bombsite rectangle used by the
// player-stats aggregator. We expose this rather than the package-internal
// bombsitePolygon to keep the polygon data in callouts.go alongside the
// existing per-map regions while letting player_stats.go reference the
// shape without an import cycle.
type SitePolygon struct {
	Site       string
	MinX, MaxX float64
	MinY, MaxY float64
}

// grenadeDisplayName maps demoinfocs weapon strings to short display names.
func grenadeDisplayName(weapon string) string {
	switch weapon {
	case "Smoke Grenade":
		return "Smoke"
	case "Flashbang":
		return "Flash"
	case "HE Grenade":
		return "HE"
	case "Decoy Grenade", "Decoy":
		return "Decoy"
	case "Incendiary Grenade", "Molotov":
		return "Molotov"
	default:
		return weapon
	}
}

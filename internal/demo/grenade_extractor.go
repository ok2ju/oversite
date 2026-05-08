package demo

import "fmt"

// GrenadeLineup represents a correlated throw→detonate grenade pair.
type GrenadeLineup struct {
	Tick        int     `json:"tick"`
	RoundNumber int     `json:"round_number"`
	SteamID     string  `json:"steam_id"`
	MapName     string  `json:"map_name"`
	GrenadeType string  `json:"grenade_type"`
	ThrowX      float64 `json:"throw_x"`
	ThrowY      float64 `json:"throw_y"`
	ThrowZ      float64 `json:"throw_z"`
	ThrowYaw    float64 `json:"throw_yaw"`
	ThrowPitch  float64 `json:"throw_pitch"`
	LandX       float64 `json:"land_x"`
	LandY       float64 `json:"land_y"`
	LandZ       float64 `json:"land_z"`
	Title       string  `json:"title"`
}

// throwKey uniquely identifies a pending throw for correlation.
type throwKey struct {
	steamID  string
	entityID int
}

// pendingThrow stores data from a grenade_throw event awaiting its detonation.
type pendingThrow struct {
	tick        int
	roundNumber int
	weapon      string
	x, y, z     float64
	yaw, pitch  float64
}

// detonationTypes lists the event types that represent grenade detonation.
// "fire_start" was added to correlate incendiary/molotov throws (spike finding).
var detonationTypes = map[string]bool{
	"grenade_detonate": true,
	"smoke_start":      true,
	"decoy_start":      true,
	"fire_start":       true,
}

// ExtractGrenadeLineups correlates grenade throw and detonate events into lineup entries.
func ExtractGrenadeLineups(mapName string, events []GameEvent) []GrenadeLineup {
	// Map of throwKey → FIFO queue of pending throws.
	pending := make(map[throwKey][]pendingThrow)

	// First pass: collect all throws.
	for i := range events {
		ev := &events[i]
		if ev.Type != "grenade_throw" {
			continue
		}
		eid := extractEntityID(ev.ExtraData)
		if eid == 0 {
			continue
		}
		key := throwKey{steamID: ev.AttackerSteamID, entityID: eid}

		var yaw, pitch float64
		if ge, ok := ev.ExtraData.(*GrenadeThrowExtra); ok && ge != nil {
			yaw = ge.ThrowYaw
			pitch = ge.ThrowPitch
		}

		pending[key] = append(pending[key], pendingThrow{
			tick:        ev.Tick,
			roundNumber: ev.RoundNumber,
			weapon:      ev.Weapon,
			x:           ev.X,
			y:           ev.Y,
			z:           ev.Z,
			yaw:         yaw,
			pitch:       pitch,
		})
	}

	// Second pass: match detonations to throws.
	var lineups []GrenadeLineup
	for i := range events {
		ev := &events[i]
		if !detonationTypes[ev.Type] {
			continue
		}
		eid := extractEntityID(ev.ExtraData)
		if eid == 0 {
			continue
		}
		key := throwKey{steamID: ev.AttackerSteamID, entityID: eid}

		queue := pending[key]
		if len(queue) == 0 {
			continue
		}

		// Dequeue oldest throw (FIFO).
		thr := queue[0]
		pending[key] = queue[1:]

		displayName := grenadeDisplayName(thr.weapon)
		title := generateTitle(mapName, displayName, thr.x, thr.y, thr.z, ev.X, ev.Y, ev.Z)

		lineups = append(lineups, GrenadeLineup{
			Tick:        thr.tick,
			RoundNumber: thr.roundNumber,
			SteamID:     ev.AttackerSteamID,
			MapName:     mapName,
			GrenadeType: displayName,
			ThrowX:      thr.x,
			ThrowY:      thr.y,
			ThrowZ:      thr.z,
			ThrowYaw:    thr.yaw,
			ThrowPitch:  thr.pitch,
			LandX:       ev.X,
			LandY:       ev.Y,
			LandZ:       ev.Z,
			Title:       title,
		})
	}

	return lineups
}

// generateTitle creates a human-readable title like "Smoke T Spawn → A Site".
func generateTitle(mapName, grenadeDisplay string, throwX, throwY, throwZ, landX, landY, landZ float64) string {
	from := resolveCallout(mapName, throwX, throwY, throwZ)
	to := resolveCallout(mapName, landX, landY, landZ)
	return fmt.Sprintf("%s %s → %s", grenadeDisplay, from, to)
}

// extractEntityID retrieves the entity_id from ExtraData. Returns 0 when the
// extras either don't carry an entity_id (e.g. kill, bomb_*) or weren't set
// at all. Knows about the four typed-extra shapes that can carry one.
func extractEntityID(extra any) int {
	switch e := extra.(type) {
	case *GrenadeThrowExtra:
		return e.EntityID
	case *GrenadeBounceExtra:
		return e.EntityID
	case *GrenadeDetonateExtra:
		return e.EntityID
	default:
		return 0
	}
}

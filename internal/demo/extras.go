package demo

// EventExtra is the marker interface implemented by every typed extra-data
// struct below. It exists so consumers (parser, shot_impacts, grenade
// extractor) can store any extra in GameEvent.ExtraData without losing type
// information at the type-assertion site.
//
// Replaces the previous `map[string]interface{}` per event. A typed struct
// allocates once when boxed into the interface; the map allocated 1 hmap +
// buckets + one boxed value per primitive (~10-15 small allocs per kill
// event). With ~50K events per match this was the dominant GC source during
// parse (see performance-review-todo.md).
//
// JSON serialization rules:
//
//   - Hot fields used by the kill-log/scoreboard/heatmap (headshot,
//     attacker_name, victim_name, attacker_team, victim_team,
//     assister_steam_id, health_damage) live as REAL COLUMNS on game_events
//     (migration 010). They are tagged `json:"-"` so they never duplicate
//     into the extra_data blob, and read by `events.go` toEventParams to
//     populate CreateGameEventParams. The frontend reads them as top-level
//     fields on `GameEvent`.
//
//   - Cold fields (penetrated, flash_assist, no_scope, …) stay in
//     extra_data because they are rendered in <1% of UI paths and not worth
//     the schema churn.
type EventExtra interface {
	isEventExtra()
}

// KillExtra is the kill-event payload.
//
// Promoted-to-column fields (Headshot, AssisterSteamID, AttackerName,
// AttackerTeam, VictimName, VictimTeam) carry `json:"-"` so they are not
// re-serialized into extra_data. They remain in the in-memory struct because
// stats.go reads them during the parse-time round stats pass — extracting
// them into a parallel parser-only struct would just duplicate the layout.
//
// Optional position fields use *float64 because 0 is a legitimate world
// coordinate near map origin and the frontend's `typeof === "number"` check
// distinguishes "absent" from "0".
type KillExtra struct {
	Headshot        bool     `json:"-"`
	Penetrated      int      `json:"penetrated"`
	FlashAssist     bool     `json:"flash_assist"`
	ThroughSmoke    bool     `json:"through_smoke"`
	NoScope         bool     `json:"no_scope"`
	AttackerBlind   bool     `json:"attacker_blind"`
	Wallbang        bool     `json:"wallbang"`
	AssisterSteamID string   `json:"-"`
	AssisterName    string   `json:"assister_name,omitempty"`
	AssisterTeam    string   `json:"assister_team,omitempty"`
	AttackerName    string   `json:"-"`
	AttackerTeam    string   `json:"-"`
	AttackerX       *float64 `json:"attacker_x,omitempty"`
	AttackerY       *float64 `json:"attacker_y,omitempty"`
	AttackerZ       *float64 `json:"attacker_z,omitempty"`
	VictimName      string   `json:"-"`
	VictimTeam      string   `json:"-"`
}

func (*KillExtra) isEventExtra() {}

// WeaponFireExtra is the weapon-fire payload. HitX/HitY/HitVictimSteamID are
// filled in by pairShotsWithImpacts after the parse pass; their absence
// indicates an unmatched shot (miss / wall hit).
type WeaponFireExtra struct {
	Yaw              float64  `json:"yaw"`
	Pitch            float64  `json:"pitch"`
	HitX             *float64 `json:"hit_x,omitempty"`
	HitY             *float64 `json:"hit_y,omitempty"`
	HitVictimSteamID string   `json:"hit_victim_steam_id,omitempty"`
}

func (*WeaponFireExtra) isEventExtra() {}

// PlayerHurtExtra is the player-hurt payload. HealthDamage / AttackerName /
// AttackerTeam / VictimName / VictimTeam are promoted to columns and carry
// `json:"-"`; ArmorDamage stays in JSON because nothing outside the parser
// reads it.
//
// HitGroup is a stable byte from demoinfocs (0=generic, 1=head, 2=chest, 3=
// stomach, 4=left-arm, 5=right-arm, 6=left-leg, 7=right-leg, 8=neck,
// 10=gear). Stored in the JSON blob — readers convert to a label lazily.
type PlayerHurtExtra struct {
	HealthDamage int `json:"-"`
	ArmorDamage  int `json:"armor_damage"`
	HitGroup     int `json:"hit_group"`
	// Penetrated is the wallbang count from the demoinfocs player_hurt
	// event. The parser currently leaves this zero; the contact-moment
	// builder reads it for the wallbang_taken flag. Populated by the
	// contact-builder test fixture loader; a follow-up may wire it through
	// the live parser once demoinfocs exposes it.
	Penetrated   int    `json:"penetrated,omitempty"`
	AttackerName string `json:"-"`
	AttackerTeam string `json:"-"`
	VictimName   string `json:"-"`
	VictimTeam   string `json:"-"`
}

func (*PlayerHurtExtra) isEventExtra() {}

// PlayerFlashedExtra is the player_flashed payload — emitted once per player
// blinded by a flashbang. DurationSecs is the on-target blind time taken
// from the demoinfocs FlashDuration() helper. The grenade thrower (attacker)
// lives on the parent GameEvent.AttackerSteamID so the aggregator can credit
// flash assists without poking through the JSON blob.
type PlayerFlashedExtra struct {
	DurationSecs float64 `json:"duration_secs"`
	AttackerName string  `json:"-"`
	AttackerTeam string  `json:"-"`
	VictimName   string  `json:"-"`
	VictimTeam   string  `json:"-"`
}

func (*PlayerFlashedExtra) isEventExtra() {}

// GrenadeThrowExtra is the grenade-throw payload. EntityID is the projectile
// entity index used to correlate throw → bounces → detonation.
type GrenadeThrowExtra struct {
	EntityID   int     `json:"entity_id,omitempty"`
	ThrowYaw   float64 `json:"throw_yaw,omitempty"`
	ThrowPitch float64 `json:"throw_pitch,omitempty"`
}

func (*GrenadeThrowExtra) isEventExtra() {}

// GrenadeBounceExtra is the grenade-bounce payload.
type GrenadeBounceExtra struct {
	BounceNr int `json:"bounce_nr"`
	EntityID int `json:"entity_id,omitempty"`
}

func (*GrenadeBounceExtra) isEventExtra() {}

// GrenadeDetonateExtra covers HE / flash / smoke_start / smoke_expired /
// decoy_start / fire_start. All use the same JSON shape (entity_id only).
type GrenadeDetonateExtra struct {
	EntityID int `json:"entity_id"`
}

func (*GrenadeDetonateExtra) isEventExtra() {}

// BombPlantExtra is the bomb-plant payload. Site is "A", "B", or "" (unknown).
type BombPlantExtra struct {
	Site string `json:"site"`
}

func (*BombPlantExtra) isEventExtra() {}

// BombDefuseExtra is the bomb-defuse payload.
type BombDefuseExtra struct {
	Site   string `json:"site"`
	HasKit bool   `json:"has_kit"`
}

func (*BombDefuseExtra) isEventExtra() {}

// BombExplodeExtra is the bomb-explode payload.
type BombExplodeExtra struct {
	Site string `json:"site"`
}

func (*BombExplodeExtra) isEventExtra() {}

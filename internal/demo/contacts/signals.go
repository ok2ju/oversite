package contacts

import (
	"sort"

	"github.com/ok2ju/oversite/internal/demo"
)

// Signal is one tick-precise ground-truth interaction between the
// subject and a specific enemy. The grouper sorts a slice of these by
// (Tick ASC, Kind ASC, EnemySteam ASC) and walks linearly.
type Signal struct {
	Tick       int32        `json:"tick"`
	EnemySteam string       `json:"enemy_steam"`
	Kind       SignalKind   `json:"kind"`
	Subject    SignalRole   `json:"subject"`
	Extras     SignalExtras `json:"extras"`
}

// SignalKind enumerates the six ground-truth kinds defined in analysis §2.
type SignalKind string

const (
	SignalVisibility    SignalKind = "visibility"
	SignalWeaponFireHit SignalKind = "weapon_fire_hit"
	SignalPlayerHurt    SignalKind = "player_hurt"
	SignalKill          SignalKind = "kill"
	SignalFlash         SignalKind = "flash"
	SignalUtilityDamage SignalKind = "utility_damage"
)

// SignalRole records which side the subject played in the interaction.
// Lets the outcome classifier distinguish "P killed E" from "E killed P"
// without re-deriving from event extras.
type SignalRole int8

const (
	SubjectAggressor SignalRole = 1
	SubjectVictim    SignalRole = 0
)

// SignalExtras carries kind-specific context the grouper and outcome
// classifier need.
type SignalExtras struct {
	HealthDamage int     `json:"health_damage,omitempty"`
	Weapon       string  `json:"weapon,omitempty"`
	FlashSeconds float64 `json:"flash_seconds,omitempty"`
	Headshot     bool    `json:"headshot,omitempty"`
	Penetrated   int     `json:"penetrated,omitempty"`
	// Wallbang mirrors KillExtra.Wallbang for kill signals where the
	// subject is the victim. Combined with Penetrated > 0 it drives
	// the contact's wallbang_taken flag.
	Wallbang bool `json:"wallbang,omitempty"`
}

// CollectInputs is everything the collector needs for one (subject, round)
// pass.
type CollectInputs struct {
	Subject      string
	SubjectTeam  string
	Round        demo.RoundData
	Events       []demo.GameEvent
	Visibility   []demo.VisibilityChange
	EnemyTeam    map[string]string
	AlivePerTick func(int32) []string
	SubjectAlive AliveRange
}

// AliveRange is the half-open tick interval [SpawnTick, DeathTick) when
// P is alive in this round. DeathTick == 0 means alive until round end.
type AliveRange struct {
	SpawnTick int32 `json:"spawn_tick"`
	DeathTick int32 `json:"death_tick"`
}

// utilityWeapons is the set of weapons treated as utility damage rather
// than direct player_hurt for outcome classification.
var utilityWeapons = map[string]bool{
	"hegrenade": true,
	"inferno":   true,
	"flashbang": true,
	"molotov":   true,
}

// bombDamageWeapons are filtered out at signal collection.
var bombDamageWeapons = map[string]bool{
	"c4":    true,
	"world": true,
}

// CollectSignals walks the round's events + visibility and emits one
// Signal per ground-truth interaction with the subject. Output is
// sorted ascending by (Tick, Kind, EnemySteam).
func CollectSignals(in CollectInputs) []Signal {
	signals := make([]Signal, 0, 32)

	freezeEnd := int32(in.Round.FreezeEndTick)

	// Visibility -> SignalVisibility (only state == 1 / spotted_on).
	for _, row := range in.Visibility {
		if row.State != 1 {
			continue
		}
		if freezeEnd > 0 && int32(row.Tick) < freezeEnd {
			continue
		}
		var enemy string
		var role SignalRole
		switch {
		case row.SpottedSteam == in.Subject:
			enemy = row.SpotterSteam
			role = SubjectVictim
		case row.SpotterSteam == in.Subject:
			enemy = row.SpottedSteam
			role = SubjectAggressor
		default:
			continue
		}
		if enemy == "" {
			continue
		}
		if team, ok := in.EnemyTeam[enemy]; ok && team == in.SubjectTeam {
			continue
		}
		signals = append(signals, Signal{
			Tick:       int32(row.Tick),
			EnemySteam: enemy,
			Kind:       SignalVisibility,
			Subject:    role,
		})
	}

	// Walk events for the other five kinds.
	for i := range in.Events {
		evt := &in.Events[i]
		if freezeEnd > 0 && int32(evt.Tick) < freezeEnd {
			continue
		}

		switch evt.Type {
		case "weapon_fire":
			fe, ok := evt.ExtraData.(*demo.WeaponFireExtra)
			if !ok || fe == nil || fe.HitVictimSteamID == "" {
				continue
			}
			var enemy string
			var role SignalRole
			switch {
			case evt.AttackerSteamID == in.Subject:
				enemy = fe.HitVictimSteamID
				role = SubjectAggressor
			case fe.HitVictimSteamID == in.Subject:
				enemy = evt.AttackerSteamID
				role = SubjectVictim
			default:
				continue
			}
			if enemy == "" || enemy == in.Subject {
				continue
			}
			if team, ok := in.EnemyTeam[enemy]; ok && team == in.SubjectTeam {
				continue
			}
			signals = append(signals, Signal{
				Tick:       int32(evt.Tick),
				EnemySteam: enemy,
				Kind:       SignalWeaponFireHit,
				Subject:    role,
				Extras:     SignalExtras{Weapon: evt.Weapon},
			})

		case "player_hurt":
			he, ok := evt.ExtraData.(*demo.PlayerHurtExtra)
			if !ok || he == nil {
				continue
			}
			if bombDamageWeapons[evt.Weapon] {
				continue
			}
			if he.AttackerTeam != "" && he.VictimTeam != "" && he.AttackerTeam == he.VictimTeam {
				continue
			}
			var enemy string
			var role SignalRole
			switch {
			case evt.AttackerSteamID == in.Subject && evt.VictimSteamID != "":
				enemy = evt.VictimSteamID
				role = SubjectAggressor
			case evt.VictimSteamID == in.Subject && evt.AttackerSteamID != "":
				enemy = evt.AttackerSteamID
				role = SubjectVictim
			default:
				continue
			}
			if enemy == in.Subject {
				continue
			}
			if team, ok := in.EnemyTeam[enemy]; ok && team == in.SubjectTeam {
				continue
			}
			kind := SignalPlayerHurt
			if utilityWeapons[evt.Weapon] {
				kind = SignalUtilityDamage
			}
			signals = append(signals, Signal{
				Tick:       int32(evt.Tick),
				EnemySteam: enemy,
				Kind:       kind,
				Subject:    role,
				Extras: SignalExtras{
					HealthDamage: he.HealthDamage,
					Weapon:       evt.Weapon,
					Penetrated:   he.Penetrated,
				},
			})

		case "kill":
			ke, ok := evt.ExtraData.(*demo.KillExtra)
			if !ok {
				continue
			}
			if evt.AttackerSteamID == evt.VictimSteamID {
				continue
			}
			if ke != nil && ke.AttackerTeam != "" && ke.VictimTeam != "" && ke.AttackerTeam == ke.VictimTeam {
				continue
			}
			var enemy string
			var role SignalRole
			switch {
			case evt.AttackerSteamID == in.Subject && evt.VictimSteamID != "":
				enemy = evt.VictimSteamID
				role = SubjectAggressor
			case evt.VictimSteamID == in.Subject && evt.AttackerSteamID != "":
				enemy = evt.AttackerSteamID
				role = SubjectVictim
			default:
				continue
			}
			if enemy == in.Subject {
				continue
			}
			if team, ok := in.EnemyTeam[enemy]; ok && team == in.SubjectTeam {
				continue
			}
			extras := SignalExtras{Weapon: evt.Weapon}
			if ke != nil {
				extras.Headshot = ke.Headshot
				extras.Penetrated = ke.Penetrated
				extras.Wallbang = ke.Wallbang
			}
			signals = append(signals, Signal{
				Tick:       int32(evt.Tick),
				EnemySteam: enemy,
				Kind:       SignalKill,
				Subject:    role,
				Extras:     extras,
			})

		case "player_flashed":
			fe, ok := evt.ExtraData.(*demo.PlayerFlashedExtra)
			if !ok || fe == nil {
				continue
			}
			if fe.DurationSecs < 0.7 {
				continue
			}
			var enemy string
			var role SignalRole
			switch {
			case evt.VictimSteamID == in.Subject && evt.AttackerSteamID != "":
				enemy = evt.AttackerSteamID
				role = SubjectVictim
			case evt.AttackerSteamID == in.Subject && evt.VictimSteamID != "":
				enemy = evt.VictimSteamID
				role = SubjectAggressor
			default:
				continue
			}
			if enemy == in.Subject {
				continue
			}
			if team, ok := in.EnemyTeam[enemy]; ok && team == in.SubjectTeam {
				continue
			}
			signals = append(signals, Signal{
				Tick:       int32(evt.Tick),
				EnemySteam: enemy,
				Kind:       SignalFlash,
				Subject:    role,
				Extras:     SignalExtras{FlashSeconds: fe.DurationSecs},
			})
		}
	}

	sort.SliceStable(signals, func(i, j int) bool {
		a, b := signals[i], signals[j]
		if a.Tick != b.Tick {
			return a.Tick < b.Tick
		}
		if a.Kind != b.Kind {
			return a.Kind < b.Kind
		}
		return a.EnemySteam < b.EnemySteam
	})

	return signals
}

import type { GameEventType } from "@/types/demo"
import { getWeaponIconPath } from "@/lib/viewer/weapon-icons"

// Wire-level weapon strings for grenade_throw events. The parser stores the
// equipment .String() value; demoinfocs uses these display names.
const GRENADE_WEAPON_TO_FILE: Record<string, string> = {
  Flashbang: "flashbang",
  "Smoke Grenade": "smokegrenade",
  "HE Grenade": "hegrenade",
  Molotov: "molotov",
  "Incendiary Grenade": "incgrenade",
  "Decoy Grenade": "decoy",
}

// Static icon paths for event types where the icon isn't keyed on a weapon
// name (bomb actions, planted_c4, defuser).
const STATIC_ICONS: Partial<Record<GameEventType, string>> = {
  bomb_plant: "/equipment/c4.svg",
  bomb_defuse: "/equipment/defuser.svg",
  // bomb_explode is folded into the spine bomb bar, not rendered as a lane
  // icon — left out intentionally.
}

// Resolve the sprite path for a timeline event. Returns null for events the
// lane renderer draws as a non-sprite glyph (player_hurt dot, etc.).
export function getEventIconPath(
  eventType: GameEventType,
  weapon: string | null | undefined,
): string | null {
  if (eventType === "kill") {
    return getWeaponIconPath(weapon)
  }
  if (eventType === "grenade_throw") {
    if (!weapon) return null
    const file = GRENADE_WEAPON_TO_FILE[weapon]
    if (file) return `/equipment/${file}.svg`
    // Fall back to the generic weapon resolver for atypical names.
    return getWeaponIconPath(weapon)
  }
  if (eventType === "player_flashed") {
    return "/equipment/flashbang.svg"
  }
  return STATIC_ICONS[eventType] ?? null
}

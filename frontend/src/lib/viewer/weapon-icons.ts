// Maps CS2 weapon display names (e.g. "AK-47", "Desert Eagle") to SVG icon
// paths under /equipment/ — the bundle of CS2 sprites in frontend/public.
//
// Names match `e.Weapon.String()` from demoinfocs-go, which is also what the
// PRIMARY/SECONDARY weapon sets use elsewhere in the viewer.
//
// Returns null for unmapped names so callers can render a fallback.

const WEAPON_TO_FILE: Record<string, string> = {
  // Rifles
  "AK-47": "ak47",
  M4A4: "m4a1",
  M4A1: "m4a1_silencer",
  AUG: "aug",
  "SG 553": "sg556",
  FAMAS: "famas",
  "Galil AR": "galilar",

  // Snipers
  AWP: "awp",
  G3SG1: "g3sg1",
  "SCAR-20": "scar20",
  "SSG 08": "ssg08",

  // SMGs
  MP7: "mp7",
  MP9: "mp9",
  "MP5-SD": "mp5sd",
  P90: "p90",
  "MAC-10": "mac10",
  "UMP-45": "ump45",
  "PP-Bizon": "bizon",

  // Heavy
  Nova: "nova",
  XM1014: "xm1014",
  "Sawed-Off": "sawedoff",
  "MAG-7": "mag7",
  M249: "m249",
  Negev: "negev",

  // Pistols
  "Glock-18": "glock",
  "USP-S": "usp_silencer",
  P2000: "p2000",
  P250: "p250",
  "Tec-9": "tec9",
  "Five-SeveN": "fiveseven",
  "CZ75 Auto": "cz75a",
  "Desert Eagle": "deagle",
  "R8 Revolver": "revolver",
  "Dual Berettas": "elite",

  // Equipment
  Knife: "knife",
  C4: "c4",
  "Zeus x27": "taser",
  "Kevlar Vest": "kevlar",
  "Kevlar + Helmet": "assaultsuit",

  // Grenades
  "Smoke Grenade": "smokegrenade",
  Flashbang: "flashbang",
  "HE Grenade": "hegrenade",
  Molotov: "molotov",
  "Incendiary Grenade": "incgrenade",
  "Decoy Grenade": "decoy",
}

export function getWeaponIconPath(
  name: string | null | undefined,
): string | null {
  if (!name) return null
  const file = WEAPON_TO_FILE[name]
  return file ? `/equipment/${file}.svg` : null
}

export const HEADSHOT_ICON_PATH = "/deathnotice/icon_headshot.svg"
export const NOSCOPE_ICON_PATH = "/deathnotice/noscope.svg"
export const PENETRATE_ICON_PATH = "/deathnotice/penetrate.svg"
export const SMOKE_KILL_ICON_PATH = "/deathnotice/smoke_kill.svg"
export const BLIND_KILL_ICON_PATH = "/deathnotice/blind_kill.svg"
export const INAIR_KILL_ICON_PATH = "/deathnotice/inairkill.svg"

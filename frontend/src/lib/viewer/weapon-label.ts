// Format the active weapon + ammo into a single line for the viewer overlay
// (canvas subtitle and the team bar rows). Returns null when there's nothing
// useful to display so callers can hide the label.
//
// Display rules:
//   - no weapon                        → null
//   - weapon with ammo (clip > 0 OR
//     reserve > 0)                     → "WEAPON  clip / reserve"
//   - weapon without ammo (knife,
//     bomb, single-use grenades)       → "WEAPON"
export function formatWeaponLabel(
  weapon: string | null | undefined,
  ammoClip: number,
  ammoReserve: number,
): string | null {
  if (!weapon) return null
  if (ammoClip > 0 || ammoReserve > 0) {
    return `${weapon}  ${ammoClip} / ${ammoReserve}`
  }
  return weapon
}

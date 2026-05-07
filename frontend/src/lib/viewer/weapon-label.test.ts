import { describe, expect, it } from "vitest"
import { formatWeaponLabel } from "./weapon-label"

describe("formatWeaponLabel", () => {
  it("returns null when there is no weapon", () => {
    expect(formatWeaponLabel(null, 30, 90)).toBeNull()
    expect(formatWeaponLabel("", 30, 90)).toBeNull()
    expect(formatWeaponLabel(undefined, 30, 90)).toBeNull()
  })

  it("includes ammo when clip or reserve is non-zero", () => {
    expect(formatWeaponLabel("AK-47", 30, 90)).toBe("AK-47  30 / 90")
    expect(formatWeaponLabel("Desert Eagle", 0, 35)).toBe(
      "Desert Eagle  0 / 35",
    )
    expect(formatWeaponLabel("USP-S", 12, 0)).toBe("USP-S  12 / 0")
  })

  it("omits ammo when both counts are zero", () => {
    expect(formatWeaponLabel("Knife", 0, 0)).toBe("Knife")
    expect(formatWeaponLabel("C4", 0, 0)).toBe("C4")
    expect(formatWeaponLabel("Smoke Grenade", 0, 0)).toBe("Smoke Grenade")
  })
})

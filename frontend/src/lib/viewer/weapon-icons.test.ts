import { describe, expect, it } from "vitest"
import { getWeaponIconPath, HEADSHOT_ICON_PATH } from "./weapon-icons"

describe("getWeaponIconPath", () => {
  it("returns null for falsy names", () => {
    expect(getWeaponIconPath(null)).toBeNull()
    expect(getWeaponIconPath(undefined)).toBeNull()
    expect(getWeaponIconPath("")).toBeNull()
  })

  it("maps display names to /equipment paths", () => {
    expect(getWeaponIconPath("AK-47")).toBe("/equipment/ak47.svg")
    expect(getWeaponIconPath("AWP")).toBe("/equipment/awp.svg")
    expect(getWeaponIconPath("Desert Eagle")).toBe("/equipment/deagle.svg")
    expect(getWeaponIconPath("USP-S")).toBe("/equipment/usp_silencer.svg")
    expect(getWeaponIconPath("Glock-18")).toBe("/equipment/glock.svg")
    expect(getWeaponIconPath("Knife")).toBe("/equipment/knife.svg")
    expect(getWeaponIconPath("Smoke Grenade")).toBe(
      "/equipment/smokegrenade.svg",
    )
    expect(getWeaponIconPath("Zeus x27")).toBe("/equipment/taser.svg")
  })

  it("returns null for unmapped weapons", () => {
    expect(getWeaponIconPath("Goldfish Cannon")).toBeNull()
    expect(getWeaponIconPath("ak47")).toBeNull() // case-sensitive — must match display name
  })

  it("exposes deathnotice icon paths", () => {
    expect(HEADSHOT_ICON_PATH).toBe("/deathnotice/icon_headshot.svg")
  })
})

import { describe, it, expect, beforeEach } from "vitest"
import { useFaceitStore } from "./faceit"

describe("faceitStore", () => {
  beforeEach(() => {
    useFaceitStore.getState().reset()
  })

  it("has correct initial state", () => {
    const state = useFaceitStore.getState()
    expect(state.profile).toBeNull()
    expect(state.isLoading).toBe(false)
  })

  it("setProfile updates profile", () => {
    const profile = {
      nickname: "TestPlayer",
      avatar_url: "https://example.com/avatar.png",
      elo: 2100,
      level: 9,
      country: "US",
      matches_played: 142,
      current_streak: { type: "win" as const, count: 3 },
    }
    useFaceitStore.getState().setProfile(profile)
    expect(useFaceitStore.getState().profile).toEqual(profile)
  })

  it("setLoading updates isLoading", () => {
    useFaceitStore.getState().setLoading(true)
    expect(useFaceitStore.getState().isLoading).toBe(true)
  })

  it("clearProfile sets profile to null", () => {
    useFaceitStore.getState().setProfile({
      nickname: "Test",
      avatar_url: null,
      elo: 1000,
      level: 5,
      country: "US",
      matches_played: 50,
      current_streak: { type: "none" as const, count: 0 },
    })
    useFaceitStore.getState().clearProfile()
    expect(useFaceitStore.getState().profile).toBeNull()
  })
})

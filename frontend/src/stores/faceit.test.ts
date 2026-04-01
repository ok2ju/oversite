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
      id: "user-1",
      nickname: "TestPlayer",
      avatarUrl: "https://example.com/avatar.png",
      elo: 2100,
      level: 9,
      country: "US",
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
      id: "user-1",
      nickname: "Test",
      avatarUrl: null,
      elo: 1000,
      level: 5,
      country: "US",
    })
    useFaceitStore.getState().clearProfile()
    expect(useFaceitStore.getState().profile).toBeNull()
  })
})

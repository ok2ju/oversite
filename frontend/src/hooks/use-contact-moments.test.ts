import { vi, describe, it, expect, beforeEach } from "vitest"
import { waitFor } from "@testing-library/react"
import { renderHookWithProviders } from "@/test/render"
import {
  mockAppBindings,
  mockRuntime,
  resetAppBindings,
} from "@/test/mocks/bindings"

vi.mock("@wailsjs/go/main/App", () => mockAppBindings)
vi.mock("@wailsjs/runtime/runtime", () => mockRuntime)

import { useContactMoments } from "./use-contact-moments"

beforeEach(() => {
  resetAppBindings()
})

describe("useContactMoments", () => {
  it("is disabled when demoId is null", () => {
    const { result } = renderHookWithProviders(() =>
      useContactMoments(null, 1, "player-1"),
    )
    expect(result.current.isLoading).toBe(false)
    expect(result.current.isFetched).toBe(false)
    expect(mockAppBindings.GetContactMoments).not.toHaveBeenCalled()
  })

  it("is disabled when roundNumber is null", () => {
    const { result } = renderHookWithProviders(() =>
      useContactMoments("42", null, "player-1"),
    )
    expect(result.current.isFetched).toBe(false)
    expect(mockAppBindings.GetContactMoments).not.toHaveBeenCalled()
  })

  it("is disabled when steamId is null", () => {
    const { result } = renderHookWithProviders(() =>
      useContactMoments("42", 1, null),
    )
    expect(result.current.isFetched).toBe(false)
    expect(mockAppBindings.GetContactMoments).not.toHaveBeenCalled()
  })

  it("resolves with the binding payload when all three are provided", async () => {
    mockAppBindings.GetContactMoments.mockResolvedValueOnce([
      {
        id: 1,
        demo_id: 42,
        round_id: 100,
        round_number: 1,
        subject_steam: "player-1",
        t_first: 1500,
        t_last: 1600,
        t_pre: 1450,
        t_post: 1700,
        enemies: ["enemy-1"],
        outcome: "won_clean",
        signal_count: 3,
        extras: {},
        mistakes: [],
      },
    ])
    const { result } = renderHookWithProviders(() =>
      useContactMoments("42", 1, "player-1"),
    )
    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(result.current.data).toHaveLength(1)
    expect(mockAppBindings.GetContactMoments).toHaveBeenCalledWith(
      "42",
      1,
      "player-1",
    )
  })

  it("re-fetches when roundNumber changes", async () => {
    mockAppBindings.GetContactMoments.mockResolvedValue([])
    const { rerender, result } = renderHookWithProviders(
      ({ round }: { round: number }) =>
        useContactMoments("42", round, "player-1"),
      { initialProps: { round: 1 } },
    )
    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(mockAppBindings.GetContactMoments).toHaveBeenCalledWith(
      "42",
      1,
      "player-1",
    )

    rerender({ round: 2 })
    await waitFor(() =>
      expect(mockAppBindings.GetContactMoments).toHaveBeenCalledWith(
        "42",
        2,
        "player-1",
      ),
    )
  })
})

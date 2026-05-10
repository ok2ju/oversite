import { describe, it, expect, beforeEach, vi } from "vitest"
import { renderHook, waitFor } from "@testing-library/react"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { mockAppBindings, resetAppBindings } from "@/test/mocks/bindings"
import { usePlayerStats } from "./use-player-stats"

vi.mock("@wailsjs/go/main/App", () => mockAppBindings)

// We deliberately wrap with a fresh QueryClient per test rather than reusing
// renderWithProviders here — the project convention forbids raw providers in
// component tests, but a hook test needs full control over query state to
// assert "no fetch" cases without the cache replaying a prior result.
function makeWrapper() {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false, gcTime: 0 } },
  })
  function Wrapper({ children }: { children: React.ReactNode }) {
    return <QueryClientProvider client={client}>{children}</QueryClientProvider>
  }
  return Wrapper
}

describe("usePlayerStats", () => {
  beforeEach(() => {
    resetAppBindings()
  })

  it("does not fetch when demoId is null", () => {
    renderHook(() => usePlayerStats(null, "STEAM_A"), {
      wrapper: makeWrapper(),
    })
    expect(mockAppBindings.GetPlayerMatchStats).not.toHaveBeenCalled()
  })

  it("does not fetch when steamId is null", () => {
    renderHook(() => usePlayerStats("1", null), { wrapper: makeWrapper() })
    expect(mockAppBindings.GetPlayerMatchStats).not.toHaveBeenCalled()
  })

  it("fetches and returns player stats", async () => {
    const { result } = renderHook(() => usePlayerStats("1", "STEAM_A"), {
      wrapper: makeWrapper(),
    })

    await waitFor(() => {
      expect(result.current.isSuccess).toBe(true)
    })

    expect(mockAppBindings.GetPlayerMatchStats).toHaveBeenCalledWith(
      "1",
      "STEAM_A",
    )
    expect(result.current.data?.steam_id).toBe("STEAM_A")
    expect(result.current.data?.kills).toBe(3)
    expect(result.current.data?.rounds).toHaveLength(2)
    expect(result.current.data?.movement.distance_units).toBe(6000)
    expect(result.current.data?.timings.avg_alive_duration_secs).toBe(51)
  })
})

import { describe, it, expect, vi, beforeEach } from "vitest"
import { renderHook, waitFor } from "@testing-library/react"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { createElement } from "react"
import { mockAppBindings, resetAppBindings } from "@/test/mocks/bindings"
import { useFaceitSync } from "./use-faceit-sync"

vi.mock("@wailsjs/go/main/App", () => mockAppBindings)

function createWrapper() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false, gcTime: 0 } },
  })
  return ({ children }: { children: React.ReactNode }) =>
    createElement(QueryClientProvider, { client: queryClient }, children)
}

describe("useFaceitSync", () => {
  beforeEach(() => {
    resetAppBindings()
  })

  it("calls SyncFaceitMatches and returns inserted count", async () => {
    mockAppBindings.SyncFaceitMatches.mockResolvedValueOnce(3)

    const { result } = renderHook(() => useFaceitSync(), {
      wrapper: createWrapper(),
    })

    result.current.mutate()

    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(result.current.data).toBe(3)
    expect(mockAppBindings.SyncFaceitMatches).toHaveBeenCalledOnce()
  })

  it("invalidates faceit queries on success", async () => {
    mockAppBindings.SyncFaceitMatches.mockResolvedValueOnce(1)

    const queryClient = new QueryClient({
      defaultOptions: { queries: { retry: false, gcTime: 0 } },
    })
    const invalidateSpy = vi.spyOn(queryClient, "invalidateQueries")
    const wrapper = ({ children }: { children: React.ReactNode }) =>
      createElement(QueryClientProvider, { client: queryClient }, children)

    const { result } = renderHook(() => useFaceitSync(), { wrapper })

    result.current.mutate()

    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: ["faceit"] })
    expect(invalidateSpy).toHaveBeenCalledWith({
      queryKey: ["faceit-matches"],
    })
  })

  it("exposes error on failure", async () => {
    mockAppBindings.SyncFaceitMatches.mockRejectedValueOnce(
      new Error("sync failed"),
    )

    const { result } = renderHook(() => useFaceitSync(), {
      wrapper: createWrapper(),
    })

    result.current.mutate()

    await waitFor(() => expect(result.current.isError).toBe(true))
    expect(result.current.error).toBeInstanceOf(Error)
  })
})

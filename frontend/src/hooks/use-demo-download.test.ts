import { describe, it, expect, vi, beforeEach } from "vitest"
import { renderHook, waitFor } from "@testing-library/react"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { createElement } from "react"
import { mockAppBindings, resetAppBindings } from "@/test/mocks/bindings"
import { useDemoDownload } from "./use-demo-download"

vi.mock("@wailsjs/go/main/App", () => mockAppBindings)

function createWrapper() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false, gcTime: 0 } },
  })
  return ({ children }: { children: React.ReactNode }) =>
    createElement(QueryClientProvider, { client: queryClient }, children)
}

describe("useDemoDownload", () => {
  beforeEach(() => {
    resetAppBindings()
  })

  it("calls ImportMatchDemo with the faceit match ID", async () => {
    mockAppBindings.ImportMatchDemo.mockResolvedValueOnce(undefined)

    const { result } = renderHook(() => useDemoDownload(), {
      wrapper: createWrapper(),
    })

    result.current.mutate("match-abc-123")

    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(mockAppBindings.ImportMatchDemo).toHaveBeenCalledWith(
      "match-abc-123",
    )
  })

  it("invalidates demos and faceit-matches queries on success", async () => {
    mockAppBindings.ImportMatchDemo.mockResolvedValueOnce(undefined)

    const queryClient = new QueryClient({
      defaultOptions: { queries: { retry: false, gcTime: 0 } },
    })
    const invalidateSpy = vi.spyOn(queryClient, "invalidateQueries")
    const wrapper = ({ children }: { children: React.ReactNode }) =>
      createElement(QueryClientProvider, { client: queryClient }, children)

    const { result } = renderHook(() => useDemoDownload(), { wrapper })

    result.current.mutate("match-xyz")

    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: ["demos"] })
    expect(invalidateSpy).toHaveBeenCalledWith({
      queryKey: ["faceit-matches"],
    })
  })

  it("exposes error on failure", async () => {
    mockAppBindings.ImportMatchDemo.mockRejectedValueOnce(
      new Error("download failed"),
    )

    const { result } = renderHook(() => useDemoDownload(), {
      wrapper: createWrapper(),
    })

    result.current.mutate("match-fail")

    await waitFor(() => expect(result.current.isError).toBe(true))
    expect(result.current.error).toBeInstanceOf(Error)
  })
})

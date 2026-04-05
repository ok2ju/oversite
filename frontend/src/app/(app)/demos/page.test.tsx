import { vi, describe, it, expect } from "vitest"
import { screen, waitFor } from "@testing-library/react"
import { renderWithProviders, userEvent } from "@/test/render"
import DemosPage from "@/app/(app)/demos/page"
import { server } from "@/test/msw/server"
import { http, HttpResponse } from "msw"

vi.mock("next/navigation", () => ({
  useRouter: vi.fn(() => ({ push: vi.fn() })),
}))

describe("DemosPage", () => {
  it("renders title and upload button", async () => {
    renderWithProviders(<DemosPage />)

    expect(screen.getByText("Demos")).toBeInTheDocument()
    expect(
      screen.getByRole("button", { name: /upload demo/i }),
    ).toBeInTheDocument()
  })

  it("shows empty state when no demos exist", async () => {
    server.use(
      http.get("/api/v1/demos", () => {
        return HttpResponse.json({
          data: [],
          meta: { total: 0, page: 1, per_page: 20 },
        })
      }),
    )

    renderWithProviders(<DemosPage />)

    await waitFor(() => {
      expect(screen.getByText(/no demos yet/i)).toBeInTheDocument()
    })
  })

  it("renders demo list when demos exist", async () => {
    renderWithProviders(<DemosPage />)

    await waitFor(() => {
      expect(screen.getByText("de_dust2")).toBeInTheDocument()
    })
  })

  it("opens upload dialog when upload button is clicked", async () => {
    const user = userEvent.setup()
    renderWithProviders(<DemosPage />)

    await user.click(screen.getByRole("button", { name: /upload demo/i }))
    await waitFor(() => {
      expect(screen.getByRole("dialog")).toBeInTheDocument()
    })
  })
})

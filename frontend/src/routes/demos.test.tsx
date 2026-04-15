import { vi, describe, it, expect } from "vitest"
import { screen, waitFor } from "@testing-library/react"
import { renderWithProviders, userEvent } from "@/test/render"
import { mockAppBindings, mockRuntime } from "@/test/mocks/bindings"
import DemosPage from "@/routes/demos"

vi.mock("@wailsjs/go/main/App", () => mockAppBindings)
vi.mock("@wailsjs/runtime/runtime", () => mockRuntime)

const mockNavigate = vi.fn()
vi.mock("react-router-dom", async () => {
  const actual = await vi.importActual("react-router-dom")
  return { ...actual, useNavigate: () => mockNavigate }
})

describe("DemosPage", () => {
  it("renders title and import button", async () => {
    renderWithProviders(<DemosPage />)

    expect(screen.getByText("Demos")).toBeInTheDocument()
    expect(
      screen.getByRole("button", { name: /import demo/i }),
    ).toBeInTheDocument()
  })

  it("renders folder import button", async () => {
    renderWithProviders(<DemosPage />)

    expect(
      screen.getByRole("button", { name: /import folder/i }),
    ).toBeInTheDocument()
  })

  it("calls ImportDemoFolder when folder button is clicked", async () => {
    const user = userEvent.setup()
    renderWithProviders(<DemosPage />)

    await user.click(screen.getByRole("button", { name: /import folder/i }))

    await waitFor(() => {
      expect(mockAppBindings.ImportDemoFolder).toHaveBeenCalled()
    })
  })

  it("shows empty state when no demos exist", async () => {
    mockAppBindings.ListDemos.mockResolvedValueOnce({
      data: [],
      meta: { total: 0, page: 1, per_page: 20 },
    })

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

  it("opens import dialog when import button is clicked", async () => {
    const user = userEvent.setup()
    renderWithProviders(<DemosPage />)

    await user.click(screen.getByRole("button", { name: /import demo/i }))
    await waitFor(() => {
      expect(screen.getByRole("dialog")).toBeInTheDocument()
    })
  })
})

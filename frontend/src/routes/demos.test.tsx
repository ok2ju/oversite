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
  it("renders the watch banner and toolbar chips", async () => {
    renderWithProviders(<DemosPage />)

    expect(screen.getByText(/Watching folder/)).toBeInTheDocument()
    for (const chip of ["All", "Wins", "Losses", "Parsing"]) {
      expect(screen.getByRole("button", { name: chip })).toBeInTheDocument()
    }
  })

  it("calls ImportDemoFolder when Re-scan is clicked", async () => {
    const user = userEvent.setup()
    renderWithProviders(<DemosPage />)

    await user.click(screen.getByRole("button", { name: /re-scan/i }))

    await waitFor(() => {
      expect(mockAppBindings.ImportDemoFolder).toHaveBeenCalled()
    })
  })

  it("calls ImportDemoFile when Import demos is clicked", async () => {
    const user = userEvent.setup()
    renderWithProviders(<DemosPage />)

    await user.click(screen.getByRole("button", { name: /import demos/i }))

    await waitFor(() => {
      expect(mockAppBindings.ImportDemoFile).toHaveBeenCalled()
    })
  })

  it("shows an empty-state row when there are no demos", async () => {
    mockAppBindings.ListDemos.mockResolvedValueOnce({
      data: [],
      meta: { total: 0, page: 1, per_page: 20 },
    })

    renderWithProviders(<DemosPage />)

    await waitFor(() => {
      expect(screen.getByText(/No demos match/i)).toBeInTheDocument()
    })
  })

  it("renders library rows when demos exist", async () => {
    renderWithProviders(<DemosPage />)

    await waitFor(() => {
      expect(screen.getByText("Dust II")).toBeInTheDocument()
      expect(screen.getByText("Mirage")).toBeInTheDocument()
    })
  })

  it("narrows the table when the Parsing chip is active", async () => {
    const user = userEvent.setup()
    renderWithProviders(<DemosPage />)

    await waitFor(() => {
      expect(screen.getByText("Dust II")).toBeInTheDocument()
    })

    await user.click(screen.getByRole("button", { name: "Parsing" }))

    expect(screen.queryByText("Dust II")).not.toBeInTheDocument()
    expect(screen.getByText("Mirage")).toBeInTheDocument()
  })
})

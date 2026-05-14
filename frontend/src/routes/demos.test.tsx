import { vi, describe, it, expect } from "vitest"
import { screen, waitFor } from "@testing-library/react"
import { renderWithProviders, userEvent } from "@/test/render"
import { mockAppBindings, mockRuntime } from "@/test/mocks/bindings"
import DemosPage from "@/routes/demos"
import { DemosHeaderActions } from "@/components/demos/demos-header-actions"

function renderPageWithHeader() {
  return renderWithProviders(
    <>
      <DemosHeaderActions />
      <DemosPage />
    </>,
  )
}

vi.mock("@wailsjs/go/main/App", () => mockAppBindings)
vi.mock("@wailsjs/runtime/runtime", () => mockRuntime)

const mockNavigate = vi.fn()
vi.mock("react-router-dom", async () => {
  const actual = await vi.importActual("react-router-dom")
  return { ...actual, useNavigate: () => mockNavigate }
})

describe("DemosPage", () => {
  it("renders the toolbar chips", async () => {
    renderWithProviders(<DemosPage />)

    await waitFor(() => {
      expect(screen.getByRole("button", { name: /^All/ })).toBeInTheDocument()
    })
    for (const chip of ["Ready", "Parsing", "Failed"]) {
      expect(screen.getByRole("button", { name: chip })).toBeInTheDocument()
    }
  })

  it("calls ImportDemoFile when Import demo is clicked", async () => {
    const user = userEvent.setup()
    renderPageWithHeader()

    await user.click(screen.getByRole("button", { name: /import demo/i }))

    await waitFor(() => {
      expect(mockAppBindings.ImportDemoFile).toHaveBeenCalled()
    })
  })

  it("shows the empty hero when there are no demos", async () => {
    mockAppBindings.ListDemos.mockResolvedValueOnce({
      data: [],
      meta: { total: 0, page: 1, per_page: 20 },
    })

    renderWithProviders(<DemosPage />)

    await waitFor(() => {
      expect(screen.getByText(/Add your first demo/i)).toBeInTheDocument()
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

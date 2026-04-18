import { describe, it, expect, vi } from "vitest"
import { screen, waitFor } from "@testing-library/react"
import { renderWithProviders } from "@/test/render"
import { mockAppBindings, mockRuntime } from "@/test/mocks/bindings"
import DashboardPage from "@/routes/dashboard"

vi.mock("@wailsjs/go/main/App", () => mockAppBindings)
vi.mock("@wailsjs/runtime/runtime", () => mockRuntime)

const mockNavigate = vi.fn()
vi.mock("react-router-dom", async () => {
  const actual = await vi.importActual("react-router-dom")
  return { ...actual, useNavigate: () => mockNavigate }
})

describe("DashboardPage", () => {
  it("renders page heading", () => {
    renderWithProviders(<DashboardPage />)

    expect(screen.getByText("Dashboard")).toBeInTheDocument()
    expect(screen.getByText("Your recent Faceit matches")).toBeInTheDocument()
  })

  it("shows loading states initially then renders profile data", async () => {
    renderWithProviders(<DashboardPage />)

    expect(screen.getByTestId("profile-card-skeleton")).toBeInTheDocument()

    await waitFor(() => {
      expect(screen.getByText("TestPlayer")).toBeInTheDocument()
    })

    expect(screen.getByText("1850")).toBeInTheDocument()
    expect(screen.getByText("Level 8")).toBeInTheDocument()
  })
})

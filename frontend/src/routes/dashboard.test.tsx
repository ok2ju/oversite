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
  it("shows the profile hero skeleton while loading", () => {
    renderWithProviders(<DashboardPage />)
    expect(screen.getByTestId("profile-hero-skeleton")).toBeInTheDocument()
  })

  it("renders only the profile hero and recent matches after load", async () => {
    renderWithProviders(<DashboardPage />)

    await waitFor(() => {
      expect(screen.getByText("TestPlayer")).toBeInTheDocument()
    })

    expect(screen.getByText("1,850")).toBeInTheDocument()
    expect(screen.getByText("Recent Matches")).toBeInTheDocument()
    expect(screen.queryByText("Performance")).not.toBeInTheDocument()
    expect(screen.queryByText("Recent form")).not.toBeInTheDocument()
    expect(screen.queryByText("Map performance")).not.toBeInTheDocument()
    expect(screen.queryByText("Weapons")).not.toBeInTheDocument()
  })
})

import { describe, it, expect } from "vitest"
import { screen, waitFor } from "@testing-library/react"
import { renderWithProviders } from "@/test/render"
import DashboardPage from "@/app/(app)/dashboard/page"

// Recharts uses ResizeObserver internally which jsdom doesn't have
class ResizeObserverStub {
  observe() {}
  unobserve() {}
  disconnect() {}
}
globalThis.ResizeObserver = ResizeObserverStub as unknown as typeof ResizeObserver

describe("DashboardPage", () => {
  it("renders page heading", () => {
    renderWithProviders(<DashboardPage />)

    expect(screen.getByText("Dashboard")).toBeInTheDocument()
    expect(
      screen.getByText("Your Faceit stats and ELO history"),
    ).toBeInTheDocument()
  })

  it("shows loading states initially then renders profile data", async () => {
    renderWithProviders(<DashboardPage />)

    // Loading skeletons appear initially
    expect(screen.getByTestId("profile-card-skeleton")).toBeInTheDocument()

    // After data loads, profile card shows
    await waitFor(() => {
      expect(screen.getByText("TestPlayer")).toBeInTheDocument()
    })

    expect(screen.getByText("1850")).toBeInTheDocument()
    expect(screen.getByText("Level 8")).toBeInTheDocument()
  })

  it("renders ELO chart section with time range buttons", async () => {
    renderWithProviders(<DashboardPage />)

    expect(screen.getByText("ELO History")).toBeInTheDocument()
    expect(screen.getByText("30d")).toBeInTheDocument()
    expect(screen.getByText("90d")).toBeInTheDocument()

    // Wait for chart data to load
    await waitFor(() => {
      expect(
        screen.queryByTestId("elo-chart-skeleton"),
      ).not.toBeInTheDocument()
    })
  })
})

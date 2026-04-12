import { vi, describe, it, expect } from "vitest"
import { screen } from "@testing-library/react"
import { renderWithProviders, userEvent } from "@/test/render"
import { EloChart } from "@/components/dashboard/elo-chart"
import type { EloHistoryPoint } from "@/types/faceit"

const mockData: EloHistoryPoint[] = [
  { elo: 1800, map_name: "de_dust2", played_at: "2026-03-01T12:00:00Z" },
  { elo: 1820, map_name: "de_mirage", played_at: "2026-03-05T14:00:00Z" },
  { elo: 1850, map_name: "de_dust2", played_at: "2026-03-15T18:00:00Z" },
]

describe("EloChart", () => {
  it("renders time range buttons", () => {
    renderWithProviders(
      <EloChart
        data={mockData}
        isLoading={false}
        days={30}
        onDaysChange={vi.fn()}
      />,
    )

    expect(screen.getByText("30d")).toBeInTheDocument()
    expect(screen.getByText("90d")).toBeInTheDocument()
    expect(screen.getByText("180d")).toBeInTheDocument()
    expect(screen.getByText("All")).toBeInTheDocument()
  })

  it("calls onDaysChange when clicking a time range button", async () => {
    const onDaysChange = vi.fn()
    const user = userEvent.setup()

    renderWithProviders(
      <EloChart
        data={mockData}
        isLoading={false}
        days={30}
        onDaysChange={onDaysChange}
      />,
    )

    await user.click(screen.getByText("90d"))
    expect(onDaysChange).toHaveBeenCalledWith(90)
  })

  it("shows loading skeleton when isLoading is true", () => {
    renderWithProviders(
      <EloChart
        data={undefined}
        isLoading={true}
        days={30}
        onDaysChange={vi.fn()}
      />,
    )

    expect(screen.getByTestId("elo-chart-skeleton")).toBeInTheDocument()
  })

  it("shows empty state when no data", () => {
    renderWithProviders(
      <EloChart data={[]} isLoading={false} days={30} onDaysChange={vi.fn()} />,
    )

    expect(screen.getByText("No ELO history available")).toBeInTheDocument()
  })

  it("shows empty state when data is undefined", () => {
    renderWithProviders(
      <EloChart
        data={undefined}
        isLoading={false}
        days={30}
        onDaysChange={vi.fn()}
      />,
    )

    expect(screen.getByText("No ELO history available")).toBeInTheDocument()
  })

  it("renders chart container when data is provided", () => {
    const { container } = renderWithProviders(
      <EloChart
        data={mockData}
        isLoading={false}
        days={30}
        onDaysChange={vi.fn()}
      />,
    )

    expect(
      container.querySelector(".recharts-responsive-container"),
    ).toBeInTheDocument()
  })

  it("renders the ELO History title", () => {
    renderWithProviders(
      <EloChart
        data={mockData}
        isLoading={false}
        days={30}
        onDaysChange={vi.fn()}
      />,
    )

    expect(screen.getByText("ELO History")).toBeInTheDocument()
  })
})

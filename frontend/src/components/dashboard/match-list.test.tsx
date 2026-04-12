import { vi, describe, it, expect } from "vitest"
import { screen, waitFor, within } from "@testing-library/react"
import { renderWithProviders, userEvent } from "@/test/render"
import { MatchList } from "@/components/dashboard/match-list"

const mockNavigate = vi.fn()
vi.mock("react-router-dom", async () => {
  const actual = await vi.importActual("react-router-dom")
  return { ...actual, useNavigate: () => mockNavigate }
})

async function waitForMatches() {
  await waitFor(() => {
    expect(screen.getByTestId("match-row-fm-1")).toBeInTheDocument()
  })
}

describe("MatchList", () => {
  it("renders match rows with map, score, result, and date", async () => {
    renderWithProviders(<MatchList />)
    await waitForMatches()

    const row1 = screen.getByTestId("match-row-fm-1")
    expect(within(row1).getByText("de_dust2")).toBeInTheDocument()
    expect(within(row1).getByText("16 - 10")).toBeInTheDocument()

    const row2 = screen.getByTestId("match-row-fm-2")
    expect(within(row2).getByText("de_mirage")).toBeInTheDocument()
    expect(within(row2).getByText("12 - 16")).toBeInTheDocument()

    expect(screen.getByTestId("match-row-fm-3")).toBeInTheDocument()
  })

  it("renders positive ELO change in green", async () => {
    renderWithProviders(<MatchList />)
    await waitForMatches()

    expect(screen.getByText("+25")).toBeInTheDocument()
    expect(screen.getByText("+25").className).toMatch(/text-green/)
  })

  it("renders negative ELO change in red", async () => {
    renderWithProviders(<MatchList />)
    await waitForMatches()

    const elements = screen.getAllByText("-20")
    expect(elements[0].className).toMatch(/text-red/)
  })

  it("shows '--' for null ELO", async () => {
    renderWithProviders(<MatchList />)
    await waitForMatches()

    const row3 = screen.getByTestId("match-row-fm-3")
    expect(within(row3).getByText("--")).toBeInTheDocument()
  })

  it("filters by map", async () => {
    const user = userEvent.setup()
    renderWithProviders(<MatchList />)
    await waitForMatches()

    await user.selectOptions(screen.getByLabelText("Map"), "de_dust2")

    await waitFor(() => {
      expect(screen.queryByTestId("match-row-fm-2")).not.toBeInTheDocument()
    })

    expect(screen.getByTestId("match-row-fm-1")).toBeInTheDocument()
    expect(screen.getByTestId("match-row-fm-4")).toBeInTheDocument()
  })

  it("filters by result", async () => {
    const user = userEvent.setup()
    renderWithProviders(<MatchList />)
    await waitForMatches()

    await user.selectOptions(screen.getByLabelText("Result"), "W")

    await waitFor(() => {
      expect(screen.queryByTestId("match-row-fm-2")).not.toBeInTheDocument()
    })

    expect(screen.getByTestId("match-row-fm-1")).toBeInTheDocument()
    expect(screen.getByTestId("match-row-fm-3")).toBeInTheDocument()
  })

  it("shows pagination controls with Previous disabled on page 1", async () => {
    renderWithProviders(<MatchList />)
    await waitForMatches()

    expect(screen.getByText(/Page 1/)).toBeInTheDocument()
    expect(screen.getByRole("button", { name: /previous/i })).toBeDisabled()
  })

  it("navigates to demo viewer on click when match has demo", async () => {
    mockNavigate.mockClear()
    const user = userEvent.setup()

    renderWithProviders(<MatchList />)
    await waitForMatches()

    await user.click(screen.getByTestId("match-row-fm-1"))
    expect(mockNavigate).toHaveBeenCalledWith("/demos/demo-1")
  })

  it("shows 'Import Demo' button for matches without demo", async () => {
    renderWithProviders(<MatchList />)
    await waitForMatches()

    const row2 = screen.getByTestId("match-row-fm-2")
    expect(within(row2).getByText("Import Demo")).toBeInTheDocument()

    const row3 = screen.getByTestId("match-row-fm-3")
    expect(within(row3).getByText("Import Demo")).toBeInTheDocument()
  })

  it("does not show 'Import Demo' for matches with demo", async () => {
    renderWithProviders(<MatchList />)
    await waitForMatches()

    const row1 = screen.getByTestId("match-row-fm-1")
    expect(
      row1.querySelector('[data-testid="import-demo-btn"]'),
    ).not.toBeInTheDocument()
  })

  it("shows loading skeletons", () => {
    const { container } = renderWithProviders(<MatchList />)

    const skeletons = container.querySelectorAll(
      '[data-testid="match-skeleton"]',
    )
    expect(skeletons.length).toBeGreaterThan(0)
  })
})

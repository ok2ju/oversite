import { describe, it, expect, vi, beforeEach } from "vitest"
import { screen } from "@testing-library/react"
import { renderWithProviders, userEvent } from "@/test/render"
import {
  mockAppBindings,
  mockRuntime,
  resetAllWailsMocks,
} from "@/test/mocks/bindings"
import { MatchRow } from "@/components/dashboard/match-row"
import { useDemoStore } from "@/stores/demo"
import type { FaceitMatch } from "@/types/faceit"

vi.mock("@wailsjs/go/main/App", () => mockAppBindings)
vi.mock("@wailsjs/runtime/runtime", () => mockRuntime)

function makeMatch(overrides: Partial<FaceitMatch> = {}): FaceitMatch {
  return {
    id: "m-1",
    faceit_match_id: "1-abc",
    map_name: "de_mirage",
    score_team: 16,
    score_opponent: 12,
    result: "W",
    kills: 20,
    deaths: 15,
    assists: 5,
    adr: 82.5,
    demo_url: null,
    demo_id: null,
    has_demo: false,
    played_at: new Date(Date.now() - 60 * 60 * 1000).toISOString(),
    ...overrides,
  }
}

describe("MatchRow", () => {
  beforeEach(() => {
    resetAllWailsMocks()
    useDemoStore.getState().reset()
  })

  it("uses win colour when result is W", () => {
    renderWithProviders(<MatchRow match={makeMatch({ result: "W" })} />)
    const dot = screen.getByLabelText("Win")
    expect(dot).toHaveStyle({ background: "var(--win)" })
  })

  it("uses loss colour when result is L", () => {
    renderWithProviders(<MatchRow match={makeMatch({ result: "L" })} />)
    const dot = screen.getByLabelText("Loss")
    expect(dot).toHaveStyle({ background: "var(--loss)" })
  })

  it("shows Imported pill when has_demo is true", () => {
    renderWithProviders(
      <MatchRow match={makeMatch({ has_demo: true, demo_id: "42" })} />,
    )
    expect(screen.getByText("Imported")).toBeInTheDocument()
  })

  it("shows No demo pill when neither demo_url nor has_demo", () => {
    renderWithProviders(<MatchRow match={makeMatch()} />)
    expect(screen.getByText("No demo")).toBeInTheDocument()
  })

  it("does not show Import button or Demo available pill when only demo_url is present — shows Import button instead", () => {
    renderWithProviders(
      <MatchRow match={makeMatch({ demo_url: "https://x" })} />,
    )
    expect(
      screen.getByRole("button", { name: /import demo/i }),
    ).toBeInTheDocument()
    expect(screen.queryByText("Demo available")).not.toBeInTheDocument()
  })

  it("hides Import button when has_demo is true", () => {
    renderWithProviders(
      <MatchRow
        match={makeMatch({
          has_demo: true,
          demo_id: "42",
          demo_url: "https://x",
        })}
      />,
    )
    expect(
      screen.queryByRole("button", { name: /import demo/i }),
    ).not.toBeInTheDocument()
  })

  it("hides Import button when demo_url is missing", () => {
    renderWithProviders(<MatchRow match={makeMatch()} />)
    expect(
      screen.queryByRole("button", { name: /import demo/i }),
    ).not.toBeInTheDocument()
  })

  it("calls ImportMatchDemo and does not fire row onClick when Import is clicked", async () => {
    const user = userEvent.setup()
    const onClick = vi.fn()
    const match = makeMatch({ demo_url: "https://x" })
    renderWithProviders(<MatchRow match={match} onClick={onClick} />)

    await user.click(screen.getByRole("button", { name: /import demo/i }))

    expect(mockAppBindings.ImportMatchDemo).toHaveBeenCalledWith("1-abc")
    expect(onClick).not.toHaveBeenCalled()
  })

  it("fires onClick with the match when the row is clicked and demo is ready", async () => {
    const user = userEvent.setup()
    const onClick = vi.fn()
    const match = makeMatch({ has_demo: true, demo_id: "42" })
    renderWithProviders(<MatchRow match={match} onClick={onClick} />)

    await user.click(screen.getByTestId(`match-row-${match.id}`))

    expect(onClick).toHaveBeenCalledWith(match)
  })

  it("does not fire onClick when row is clicked but demo is not ready", async () => {
    const user = userEvent.setup()
    const onClick = vi.fn()
    const match = makeMatch({ demo_url: "https://x" })
    renderWithProviders(<MatchRow match={match} onClick={onClick} />)

    await user.click(screen.getByTestId(`match-row-${match.id}`))

    expect(onClick).not.toHaveBeenCalled()
  })

  it("shows Parsing… indicator and delays navigation when demo is parsing", async () => {
    const user = userEvent.setup()
    const onClick = vi.fn()
    const match = makeMatch({ has_demo: true, demo_id: "42" })

    useDemoStore.getState().updateImportProgress({
      demoId: 42,
      fileName: "match.dem",
      percent: 40,
      stage: "parsing",
    })

    renderWithProviders(<MatchRow match={match} onClick={onClick} />)

    expect(screen.getByText(/parsing/i)).toBeInTheDocument()

    await user.click(screen.getByTestId(`match-row-${match.id}`))
    expect(onClick).not.toHaveBeenCalled()

    useDemoStore.getState().updateImportProgress({
      demoId: 42,
      fileName: "match.dem",
      percent: 100,
      stage: "complete",
    })

    await vi.waitFor(() => expect(onClick).toHaveBeenCalledWith(match))
  })
})

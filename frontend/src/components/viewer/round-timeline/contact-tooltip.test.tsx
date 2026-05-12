import { describe, it, expect } from "vitest"
import { fireEvent } from "@testing-library/react"
import { renderWithProviders } from "@/test/render"
import { ContactTooltip } from "./contact-tooltip"
import type { ContactMarker } from "@/lib/timeline/types"
import type { main } from "@wailsjs/go/models"

function marker(overrides: Partial<ContactMarker> = {}): ContactMarker {
  return {
    id: 1,
    subjectSteam: "player-1",
    tFirst: 1500,
    tPre: 1450,
    tLast: 1600,
    tPost: 1700,
    outcome: "untraded_death" as main.ContactOutcome,
    enemies: ["enemy-1", "enemy-2"],
    mistakes: [],
    worstSeverity: 0,
    ...overrides,
  }
}

function m(overrides: Partial<main.ContactMistake> = {}): main.ContactMistake {
  return {
    kind: "slow_reaction",
    category: "aim",
    severity: 2,
    phase: "pre",
    tick: 1480,
    extras: {},
    ...overrides,
  } as unknown as main.ContactMistake
}

describe("ContactTooltip", () => {
  it("renders the outcome label and the elapsed-time header", () => {
    const { getByText, getByTestId } = renderWithProviders(
      <ContactTooltip
        contact={marker({ outcome: "untraded_death" as main.ContactOutcome })}
        tickRate={64}
        roundStartTick={1000}
      />,
    )
    expect(getByTestId("contact-tooltip")).toBeInTheDocument()
    expect(getByText(/Untraded death/i)).toBeInTheDocument()
  })

  it("renders the 'No mistakes' empty state for a clean contact", () => {
    const { getByText, queryByTestId } = renderWithProviders(
      <ContactTooltip
        contact={marker({
          mistakes: [],
          outcome: "won_clean" as main.ContactOutcome,
        })}
        tickRate={64}
        roundStartTick={1000}
      />,
    )
    expect(getByText(/No mistakes/i)).toBeInTheDocument()
    expect(queryByTestId("contact-tooltip-expand")).not.toBeInTheDocument()
  })

  it("renders all mistakes when there are 3 or fewer (no expand)", () => {
    const { queryByTestId, getAllByRole } = renderWithProviders(
      <ContactTooltip
        contact={marker({
          mistakes: [
            m({ kind: "slow_reaction", severity: 2, phase: "pre", tick: 1480 }),
            m({
              kind: "isolated_peek",
              category: "positioning",
              severity: 3,
              phase: "pre",
              tick: 1490,
            }),
          ],
        })}
        tickRate={64}
        roundStartTick={1000}
      />,
    )
    expect(queryByTestId("contact-tooltip-phase-pre")).toBeInTheDocument()
    expect(
      queryByTestId("contact-tooltip-phase-during"),
    ).not.toBeInTheDocument()
    expect(queryByTestId("contact-tooltip-expand")).not.toBeInTheDocument()
    expect(getAllByRole("listitem")).toHaveLength(2)
  })

  it("renders top-3 mistakes by severity DESC and a '+N more' button when there are more", () => {
    const mistakes = [
      m({ kind: "slow_reaction", severity: 2, phase: "pre", tick: 1480 }),
      m({
        kind: "isolated_peek",
        category: "positioning",
        severity: 3,
        phase: "pre",
        tick: 1490,
      }),
      m({
        kind: "shot_while_moving",
        category: "movement",
        severity: 2,
        phase: "during",
        tick: 1520,
      }),
      m({
        kind: "lost_hp_advantage",
        category: "trade",
        severity: 3,
        phase: "during",
        tick: 1540,
      }),
      m({
        kind: "no_reload_with_cover",
        category: "utility",
        severity: 2,
        phase: "post",
        tick: 1650,
      }),
    ]
    const { getByTestId, getByText } = renderWithProviders(
      <ContactTooltip
        contact={marker({ mistakes })}
        tickRate={64}
        roundStartTick={1000}
      />,
    )
    expect(getByTestId("contact-tooltip-expand")).toHaveTextContent("+2 more")
    expect(getByText(/Isolated peek/i)).toBeInTheDocument()
    expect(getByText(/Lost HP advantage/i)).toBeInTheDocument()
  })

  it("expands the rest on click", () => {
    const mistakes = [
      m({ kind: "slow_reaction", severity: 2, phase: "pre", tick: 1480 }),
      m({
        kind: "isolated_peek",
        category: "positioning",
        severity: 3,
        phase: "pre",
        tick: 1490,
      }),
      m({
        kind: "shot_while_moving",
        category: "movement",
        severity: 2,
        phase: "during",
        tick: 1520,
      }),
      m({
        kind: "lost_hp_advantage",
        category: "trade",
        severity: 3,
        phase: "during",
        tick: 1540,
      }),
      m({
        kind: "no_reload_with_cover",
        category: "utility",
        severity: 2,
        phase: "post",
        tick: 1650,
      }),
    ]
    const { getByTestId, queryByTestId, getAllByRole } = renderWithProviders(
      <ContactTooltip
        contact={marker({ mistakes })}
        tickRate={64}
        roundStartTick={1000}
      />,
    )
    fireEvent.click(getByTestId("contact-tooltip-expand"))
    expect(queryByTestId("contact-tooltip-expand")).not.toBeInTheDocument()
    expect(getAllByRole("listitem")).toHaveLength(5)
    expect(getByTestId("contact-tooltip-phase-pre")).toBeInTheDocument()
    expect(getByTestId("contact-tooltip-phase-during")).toBeInTheDocument()
    expect(getByTestId("contact-tooltip-phase-post")).toBeInTheDocument()
  })

  it("renders the enemy count", () => {
    const { getByText } = renderWithProviders(
      <ContactTooltip
        contact={marker({ enemies: ["A", "B", "C"] })}
        tickRate={64}
        roundStartTick={1000}
      />,
    )
    expect(getByText(/3 enemies/i)).toBeInTheDocument()
  })
})

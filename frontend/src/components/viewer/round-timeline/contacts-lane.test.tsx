import { describe, it, expect, beforeEach } from "vitest"
import { fireEvent } from "@testing-library/react"
import { renderWithProviders } from "@/test/render"
import { TooltipProvider } from "@/components/ui/tooltip"
import { ContactsLane } from "./contacts-lane"
import { useViewerStore } from "@/stores/viewer"
import type { ContactMarker } from "@/lib/timeline/types"
import type { main } from "@wailsjs/go/models"

function marker(overrides: Partial<ContactMarker> = {}): ContactMarker {
  return {
    id: 7,
    subjectSteam: "player-1",
    tFirst: 1500,
    tPre: 1450,
    tLast: 1600,
    tPost: 1700,
    outcome: "won_clean" as main.ContactOutcome,
    enemies: ["enemy-1"],
    mistakes: [],
    worstSeverity: 0,
    ...overrides,
  }
}

beforeEach(() => {
  useViewerStore.getState().reset()
})

describe("ContactsLane", () => {
  it("renders the no-player placeholder when hasPlayer is false", () => {
    const { getByTestId, queryByTestId } = renderWithProviders(
      <ContactsLane
        contacts={[]}
        roundStartTick={1000}
        roundEndTick={5000}
        hasPlayer={false}
        activeContactId={null}
      />,
    )
    expect(
      getByTestId("round-timeline-contacts-placeholder"),
    ).toBeInTheDocument()
    expect(queryByTestId("round-timeline-contacts")).not.toBeInTheDocument()
  })

  it("renders the empty state when player is selected but no contacts", () => {
    const { getByTestId } = renderWithProviders(
      <ContactsLane
        contacts={[]}
        roundStartTick={1000}
        roundEndTick={5000}
        hasPlayer={true}
        activeContactId={null}
      />,
    )
    expect(getByTestId("round-timeline-contacts-empty")).toBeInTheDocument()
  })

  it("renders one marker per contact", () => {
    const { queryByTestId } = renderWithProviders(
      <TooltipProvider>
        <ContactsLane
          contacts={[marker({ id: 1 }), marker({ id: 2 })]}
          roundStartTick={1000}
          roundEndTick={5000}
          hasPlayer={true}
          activeContactId={null}
        />
      </TooltipProvider>,
    )
    expect(queryByTestId("contact-marker-1")).toBeInTheDocument()
    expect(queryByTestId("contact-marker-2")).toBeInTheDocument()
  })

  it("seeks to t_pre on click (not t_first)", () => {
    const { getByTestId } = renderWithProviders(
      <TooltipProvider>
        <ContactsLane
          contacts={[marker({ id: 1, tFirst: 1500, tPre: 1450 })]}
          roundStartTick={1000}
          roundEndTick={5000}
          hasPlayer={true}
          activeContactId={null}
        />
      </TooltipProvider>,
    )
    fireEvent.click(getByTestId("contact-marker-1"))
    expect(useViewerStore.getState().currentTick).toBe(1450)
  })

  it("pauses playback on click", () => {
    useViewerStore.setState({ isPlaying: true })
    const { getByTestId } = renderWithProviders(
      <TooltipProvider>
        <ContactsLane
          contacts={[marker({ id: 1 })]}
          roundStartTick={1000}
          roundEndTick={5000}
          hasPlayer={true}
          activeContactId={null}
        />
      </TooltipProvider>,
    )
    fireEvent.click(getByTestId("contact-marker-1"))
    expect(useViewerStore.getState().isPlaying).toBe(false)
  })

  it("positions the marker proportionally to tFirst", () => {
    const { getByTestId } = renderWithProviders(
      <TooltipProvider>
        <ContactsLane
          contacts={[marker({ id: 1, tFirst: 3000 })]} // halfway through [1000, 5000]
          roundStartTick={1000}
          roundEndTick={5000}
          hasPlayer={true}
          activeContactId={null}
        />
      </TooltipProvider>,
    )
    const btn = getByTestId("contact-marker-1")
    expect(btn.getAttribute("style")).toMatch(/left:\s*50%/)
  })

  it("renders the highlight on the marker matching activeContactId", () => {
    const { getByTestId } = renderWithProviders(
      <TooltipProvider>
        <ContactsLane
          contacts={[marker({ id: 1 }), marker({ id: 2 })]}
          roundStartTick={1000}
          roundEndTick={5000}
          hasPlayer={true}
          activeContactId={2}
        />
      </TooltipProvider>,
    )
    const a = getByTestId("contact-marker-1")
    const b = getByTestId("contact-marker-2")
    expect(a).not.toHaveAttribute("data-active")
    expect(b).toHaveAttribute("data-active", "true")
    expect(b).toHaveClass("ring-2")
  })

  it("renders no highlight when activeContactId is null", () => {
    const { getByTestId } = renderWithProviders(
      <TooltipProvider>
        <ContactsLane
          contacts={[marker({ id: 1 }), marker({ id: 2 })]}
          roundStartTick={1000}
          roundEndTick={5000}
          hasPlayer={true}
          activeContactId={null}
        />
      </TooltipProvider>,
    )
    expect(getByTestId("contact-marker-1")).not.toHaveAttribute("data-active")
    expect(getByTestId("contact-marker-1")).not.toHaveClass("ring-2")
  })
})

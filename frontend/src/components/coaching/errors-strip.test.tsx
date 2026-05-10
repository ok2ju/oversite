import { describe, it, expect } from "vitest"
import { screen } from "@testing-library/react"
import { renderWithProviders } from "@/test/render"
import { ErrorsStrip } from "@/components/coaching/errors-strip"

describe("ErrorsStrip", () => {
  it("renders only fire-related kinds with count > 0, ordered by their canonical sequence", () => {
    renderWithProviders(
      <ErrorsStrip
        latestDemoId="42"
        errors={[
          { kind: "no_trade_death", total: 8 }, // not fire-related — hidden
          { kind: "missed_first_shot", total: 5 },
          { kind: "shot_while_moving", total: 2 },
        ]}
      />,
    )

    expect(
      screen.getByTestId("errors-strip-card-missed_first_shot"),
    ).toBeInTheDocument()
    expect(
      screen.getByTestId("errors-strip-card-shot_while_moving"),
    ).toBeInTheDocument()
    expect(
      screen.queryByTestId("errors-strip-card-no_trade_death"),
    ).not.toBeInTheDocument()
    expect(
      screen.queryByTestId("errors-strip-card-slow_reaction"),
    ).not.toBeInTheDocument()

    expect(
      screen.getByTestId("errors-strip-count-missed_first_shot"),
    ).toHaveTextContent("5")
  })

  it("renders the 'see duel-by-duel' CTA pointing to the latest demo", () => {
    renderWithProviders(
      <ErrorsStrip
        latestDemoId="42"
        errors={[{ kind: "missed_first_shot", total: 1 }]}
      />,
    )
    const link = screen.getByTestId("errors-strip-cta")
    expect(link).toHaveAttribute("href", "/demos/42/analysis")
  })

  it("hides the CTA when there is no latest demo id", () => {
    renderWithProviders(
      <ErrorsStrip
        latestDemoId=""
        errors={[{ kind: "missed_first_shot", total: 1 }]}
      />,
    )
    expect(screen.queryByTestId("errors-strip-cta")).not.toBeInTheDocument()
  })

  it("renders the empty state when no fire-related kinds have a count", () => {
    renderWithProviders(<ErrorsStrip latestDemoId="42" errors={[]} />)
    expect(screen.getByTestId("errors-strip-empty")).toBeInTheDocument()
  })

  it("filters out non-fire kinds even when they have counts", () => {
    renderWithProviders(
      <ErrorsStrip
        latestDemoId="42"
        errors={[
          { kind: "no_trade_death", total: 4 },
          { kind: "isolated_peek", total: 3 },
        ]}
      />,
    )
    expect(screen.getByTestId("errors-strip-empty")).toBeInTheDocument()
  })
})

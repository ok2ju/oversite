import { describe, it, expect, vi } from "vitest"
import { screen } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { renderWithProviders } from "@/test/render"
import { MicroCard } from "@/components/coaching/micro-card"
import type { CoachingHabitRow } from "@/types/analysis"

function makeRow(overrides: Partial<CoachingHabitRow> = {}): CoachingHabitRow {
  return {
    key: "counter_strafe",
    label: "Counter-strafe",
    description: "Stop before firing so the first bullet lands.",
    unit: "ms",
    direction: "lower",
    value: 120,
    status: "warn",
    good_threshold: 100,
    warn_threshold: 200,
    good_min: 0,
    good_max: 0,
    warn_min: 0,
    warn_max: 0,
    previous_value: null,
    delta: null,
    trend: [
      { demo_id: "3", match_date: "2026-05-01", value: 120 },
      { demo_id: "2", match_date: "2026-04-30", value: 150 },
      { demo_id: "1", match_date: "2026-04-29", value: 180 },
    ],
    ...overrides,
  }
}

describe("MicroCard", () => {
  it("renders ms-scale value as seconds with norm and status", () => {
    renderWithProviders(<MicroCard row={makeRow()} />)

    expect(screen.getByTestId("micro-card-counter_strafe")).toHaveAttribute(
      "data-status",
      "warn",
    )
    expect(
      screen.getByTestId("micro-card-value-counter_strafe"),
    ).toHaveTextContent("0.12")
    expect(
      screen.getByTestId("micro-card-norm-counter_strafe"),
    ).toHaveTextContent("norm ≤ 0.10 s")
    expect(
      screen.getByTestId("micro-card-status-counter_strafe"),
    ).toHaveTextContent("warn")
  })

  it("renders sparkline when trend has 2+ points", () => {
    renderWithProviders(<MicroCard row={makeRow()} />)
    expect(screen.getByTestId("sparkline")).toBeInTheDocument()
  })

  it("hides sparkline and shows 'first demo' when trend has < 2 points", () => {
    renderWithProviders(
      <MicroCard
        row={makeRow({
          trend: [{ demo_id: "1", match_date: "2026-04-29", value: 180 }],
        })}
      />,
    )
    expect(
      screen.getByTestId("micro-card-trend-empty-counter_strafe"),
    ).toBeInTheDocument()
    expect(screen.queryByTestId("sparkline")).not.toBeInTheDocument()
  })

  it("renders a higher-is-better percentage with good direction copy", () => {
    renderWithProviders(
      <MicroCard
        row={makeRow({
          key: "first_shot_acc",
          label: "First-shot accuracy",
          description: "Hits on the bullet that decides duels.",
          unit: "%",
          direction: "higher",
          value: 54,
          status: "good",
          good_threshold: 50,
          warn_threshold: 35,
        })}
      />,
    )
    expect(
      screen.getByTestId("micro-card-value-first_shot_acc"),
    ).toHaveTextContent("54")
    expect(
      screen.getByTestId("micro-card-norm-first_shot_acc"),
    ).toHaveTextContent("norm ≥ 50 %")
  })

  it("calls onClick when used as a button", async () => {
    const onClick = vi.fn()
    const user = userEvent.setup()
    renderWithProviders(<MicroCard row={makeRow()} onClick={onClick} />)
    await user.click(screen.getByTestId("micro-card-counter_strafe"))
    expect(onClick).toHaveBeenCalledTimes(1)
  })

  it("renders a balanced direction with a min–max norm range", () => {
    renderWithProviders(
      <MicroCard
        row={makeRow({
          key: "flick_balance",
          label: "Flick balance",
          description: "Over- vs under-flicks balance.",
          unit: "%",
          direction: "balanced",
          value: 55,
          status: "good",
          good_threshold: 0,
          warn_threshold: 0,
          good_min: 45,
          good_max: 55,
          warn_min: 40,
          warn_max: 60,
        })}
      />,
    )
    expect(
      screen.getByTestId("micro-card-norm-flick_balance"),
    ).toHaveTextContent("norm 45–55 %")
  })
})

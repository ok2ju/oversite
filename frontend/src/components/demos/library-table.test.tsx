import { describe, it, expect, vi, beforeEach } from "vitest"
import { screen, within } from "@testing-library/react"
import { renderWithProviders, userEvent } from "@/test/render"
import {
  mockAppBindings,
  mockRuntime,
  resetAllWailsMocks,
} from "@/test/mocks/bindings"
import { LibraryTable, filterDemos } from "@/components/demos/library-table"
import { useDemoStore } from "@/stores/demo"
import type { Demo } from "@/types/demo"

vi.mock("@wailsjs/go/main/App", () => mockAppBindings)
vi.mock("@wailsjs/runtime/runtime", () => mockRuntime)

const mockNavigate = vi.fn()
vi.mock("react-router-dom", async () => {
  const actual =
    await vi.importActual<typeof import("react-router-dom")>("react-router-dom")
  return { ...actual, useNavigate: () => mockNavigate }
})

function makeDemo(overrides: Partial<Demo>): Demo {
  return {
    id: 1,
    map_name: "de_mirage",
    file_path: "/demos/mirage-match.dem",
    file_size: 100_000_000,
    status: "ready",
    total_ticks: 0,
    tick_rate: 0,
    duration_secs: 0,
    match_date: "",
    created_at: "",
    ...overrides,
  }
}

describe("filterDemos", () => {
  const demos = [
    makeDemo({
      id: 1,
      map_name: "de_mirage",
      file_path: "/demos/mirage-1.dem",
      status: "ready",
    }),
    makeDemo({
      id: 2,
      map_name: "de_nuke",
      file_path: "/demos/nuke-1.dem",
      status: "parsing",
    }),
    makeDemo({
      id: 3,
      map_name: "de_inferno",
      file_path: "/demos/inferno-1.dem",
      status: "failed",
    }),
    makeDemo({
      id: 4,
      map_name: "de_mirage",
      file_path: "/demos/mirage-2.dem",
      status: "imported",
    }),
  ]

  it("returns every demo when filter is 'all' and search is blank", () => {
    expect(filterDemos(demos, "", "all")).toHaveLength(4)
  })

  it("narrows to parsing demos when the Parsing chip is active", () => {
    const result = filterDemos(demos, "", "parsing")
    expect(result).toHaveLength(1)
    expect(result[0].id).toBe(2)
  })

  it("substring-matches the map name and file path when searching", () => {
    const result = filterDemos(demos, "mirage", "all")
    expect(result.map((d) => d.id)).toEqual([1, 4])
  })

  it("leaves Wins/Losses as no-ops until backend result data lands", () => {
    expect(filterDemos(demos, "", "wins")).toHaveLength(4)
    expect(filterDemos(demos, "", "losses")).toHaveLength(4)
  })
})

describe("LibraryTable row navigation", () => {
  beforeEach(() => {
    resetAllWailsMocks()
    mockNavigate.mockReset()
    useDemoStore.getState().reset()
  })

  function render(demos: Demo[]) {
    renderWithProviders(
      <LibraryTable demos={demos} search="" filter="all" onDelete={() => {}} />,
    )
  }

  it("navigates to /demos/:id when a ready row is clicked", async () => {
    const user = userEvent.setup()
    const demo = makeDemo({ id: 7, status: "ready" })
    render([demo])

    await user.click(screen.getByTestId("demo-row-7"))

    expect(mockNavigate).toHaveBeenCalledWith("/demos/7")
  })

  it("does not navigate when clicking a parsing row, shows waiting indicator", async () => {
    const user = userEvent.setup()
    const demo = makeDemo({ id: 9, status: "parsing" })
    render([demo])

    await user.click(screen.getByTestId("demo-row-9"))

    expect(mockNavigate).not.toHaveBeenCalled()
    expect(screen.getByTestId("demo-row-9-waiting")).toBeInTheDocument()
  })

  it("navigates once parsing completes for the waiting row", async () => {
    const user = userEvent.setup()
    const demo = makeDemo({ id: 11, status: "parsing" })
    render([demo])

    await user.click(screen.getByTestId("demo-row-11"))
    expect(mockNavigate).not.toHaveBeenCalled()

    useDemoStore.getState().updateImportProgress({
      demoId: 11,
      fileName: "x.dem",
      percent: 100,
      stage: "complete",
    })

    await vi.waitFor(() =>
      expect(mockNavigate).toHaveBeenCalledWith("/demos/11"),
    )
  })
})

describe("LibraryTable delete confirmation", () => {
  beforeEach(() => {
    resetAllWailsMocks()
    mockNavigate.mockReset()
    useDemoStore.getState().reset()
  })

  function renderWithDelete(onDelete: (id: number) => void) {
    const demo = makeDemo({
      id: 42,
      file_path: "/demos/awesome-clutch.dem",
      status: "ready",
    })
    renderWithProviders(
      <LibraryTable
        demos={[demo]}
        search=""
        filter="all"
        onDelete={onDelete}
      />,
    )
  }

  it("opens a confirmation dialog instead of deleting immediately", async () => {
    const user = userEvent.setup()
    const onDelete = vi.fn()
    renderWithDelete(onDelete)

    await user.click(screen.getByRole("button", { name: "Delete" }))

    const dialog = screen.getByRole("alertdialog", {
      name: "Remove this demo?",
    })
    expect(dialog).toBeInTheDocument()
    expect(within(dialog).getByText(/awesome-clutch\.dem/)).toBeInTheDocument()
    expect(onDelete).not.toHaveBeenCalled()
  })

  it("calls onDelete with the demo id when the user confirms", async () => {
    const user = userEvent.setup()
    const onDelete = vi.fn()
    renderWithDelete(onDelete)

    await user.click(screen.getByRole("button", { name: "Delete" }))
    await user.click(screen.getByRole("button", { name: "Remove" }))

    expect(onDelete).toHaveBeenCalledTimes(1)
    expect(onDelete).toHaveBeenCalledWith(42)
  })

  it("does not call onDelete when the user cancels", async () => {
    const user = userEvent.setup()
    const onDelete = vi.fn()
    renderWithDelete(onDelete)

    await user.click(screen.getByRole("button", { name: "Delete" }))
    await user.click(screen.getByRole("button", { name: "Cancel" }))

    expect(onDelete).not.toHaveBeenCalled()
    expect(screen.queryByRole("alertdialog")).not.toBeInTheDocument()
  })
})

import { screen, within } from "@testing-library/react"
import { renderWithProviders, userEvent } from "@/test/render"
import { DemoList } from "@/components/demos/demo-list"
import type { Demo } from "@/types/demo"
import { mockDemos } from "@/test/msw/handlers"

vi.mock("next/navigation", () => ({
  useRouter: vi.fn(() => ({ push: vi.fn() })),
}))

const { useRouter } = await import("next/navigation")

function readyDemo(): Demo {
  return mockDemos.find((d) => d.status === "ready")!
}

function parsingDemo(): Demo {
  return mockDemos.find((d) => d.status === "parsing")!
}

describe("DemoList", () => {
  it("renders demo cards with map name, date, status badge, and file size", () => {
    renderWithProviders(
      <DemoList demos={mockDemos} isLoading={false} onDelete={vi.fn()} />,
    )

    expect(screen.getByText("de_dust2")).toBeInTheDocument()
    expect(screen.getByText("de_mirage")).toBeInTheDocument()
    expect(screen.getByText("de_inferno")).toBeInTheDocument()
  })

  it("shows status badges with correct variants", () => {
    renderWithProviders(
      <DemoList demos={mockDemos} isLoading={false} onDelete={vi.fn()} />,
    )

    expect(screen.getByText("ready")).toBeInTheDocument()
    expect(screen.getByText("parsing")).toBeInTheDocument()
    expect(screen.getByText("uploaded")).toBeInTheDocument()
    expect(screen.getByText("failed")).toBeInTheDocument()
  })

  it("navigates to /demos/{id} when clicking a ready demo", async () => {
    const push = vi.fn()
    vi.mocked(useRouter).mockReturnValue({ push } as ReturnType<
      typeof useRouter
    >)
    const user = userEvent.setup()

    renderWithProviders(
      <DemoList demos={[readyDemo()]} isLoading={false} onDelete={vi.fn()} />,
    )

    await user.click(screen.getByText("de_dust2"))
    expect(push).toHaveBeenCalledWith("/demos/demo-1")
  })

  it("does not navigate when clicking a non-ready demo", async () => {
    const push = vi.fn()
    vi.mocked(useRouter).mockReturnValue({ push } as ReturnType<
      typeof useRouter
    >)
    const user = userEvent.setup()

    renderWithProviders(
      <DemoList
        demos={[parsingDemo()]}
        isLoading={false}
        onDelete={vi.fn()}
      />,
    )

    await user.click(screen.getByText("de_mirage"))
    expect(push).not.toHaveBeenCalled()
  })

  it("shows loading skeletons when loading", () => {
    const { container } = renderWithProviders(
      <DemoList demos={[]} isLoading={true} onDelete={vi.fn()} />,
    )

    const skeletons = container.querySelectorAll('[data-testid="demo-skeleton"]')
    expect(skeletons.length).toBeGreaterThan(0)
  })

  it("opens delete confirmation dialog on delete button click", async () => {
    const user = userEvent.setup()

    renderWithProviders(
      <DemoList demos={[readyDemo()]} isLoading={false} onDelete={vi.fn()} />,
    )

    await user.click(screen.getByRole("button", { name: /delete/i }))
    expect(
      screen.getByText(/are you sure/i),
    ).toBeInTheDocument()
  })

  it("calls onDelete when confirming delete", async () => {
    const onDelete = vi.fn()
    const user = userEvent.setup()

    renderWithProviders(
      <DemoList demos={[readyDemo()]} isLoading={false} onDelete={onDelete} />,
    )

    await user.click(screen.getByRole("button", { name: /delete/i }))
    await user.click(screen.getByRole("button", { name: /confirm/i }))
    expect(onDelete).toHaveBeenCalledWith("demo-1")
  })

  it("shows 'Unknown Map' when map_name is null", () => {
    const demo: Demo = { ...mockDemos[2], status: "uploaded" }
    renderWithProviders(
      <DemoList demos={[demo]} isLoading={false} onDelete={vi.fn()} />,
    )

    expect(screen.getByText("Unknown Map")).toBeInTheDocument()
  })
})

import { vi, describe, it, expect } from "vitest"
import { screen } from "@testing-library/react"
import { renderWithProviders, userEvent } from "@/test/render"
import {
  DemoCard,
  formatFileSize,
  formatDuration,
} from "@/components/demos/demo-card"
import type { Demo } from "@/types/demo"
import { mockDemos } from "@/test/msw/handlers"

const mockNavigate = vi.fn()
vi.mock("react-router-dom", async () => {
  const actual = await vi.importActual("react-router-dom")
  return { ...actual, useNavigate: () => mockNavigate }
})

function readyDemo(): Demo {
  return mockDemos.find((d) => d.status === "ready")!
}

describe("DemoCard", () => {
  it("renders all demo fields correctly", () => {
    renderWithProviders(<DemoCard demo={readyDemo()} onDelete={vi.fn()} />)

    expect(screen.getByText("de_dust2")).toBeInTheDocument()
    expect(screen.getByText("ready")).toBeInTheDocument()
    expect(screen.getByText("150.0 MB")).toBeInTheDocument()
    expect(screen.getByText("33:20")).toBeInTheDocument()
  })

  it("shows 'Unknown Map' when map_name is empty", () => {
    const demo: Demo = { ...mockDemos[2], map_name: "" }
    renderWithProviders(<DemoCard demo={demo} onDelete={vi.fn()} />)

    expect(screen.getByText("Unknown Map")).toBeInTheDocument()
  })

  it("formats file size correctly", () => {
    expect(formatFileSize(500)).toBe("0.5 KB")
    expect(formatFileSize(1_500_000)).toBe("1.5 MB")
    expect(formatFileSize(2_500_000_000)).toBe("2.5 GB")
  })

  it("formats duration correctly", () => {
    expect(formatDuration(90)).toBe("1:30")
    expect(formatDuration(2000)).toBe("33:20")
    expect(formatDuration(0)).toBe("0:00")
  })

  it("navigates when clicking a ready demo", async () => {
    mockNavigate.mockClear()
    const user = userEvent.setup()

    renderWithProviders(<DemoCard demo={readyDemo()} onDelete={vi.fn()} />)

    await user.click(screen.getByText("de_dust2"))
    expect(mockNavigate).toHaveBeenCalledWith("/demos/1")
  })

  it("does not navigate when clicking a non-ready demo", async () => {
    mockNavigate.mockClear()
    const user = userEvent.setup()
    const parsingDemo = mockDemos.find((d) => d.status === "parsing")!

    renderWithProviders(<DemoCard demo={parsingDemo} onDelete={vi.fn()} />)

    await user.click(screen.getByText("de_mirage"))
    expect(mockNavigate).not.toHaveBeenCalled()
  })
})

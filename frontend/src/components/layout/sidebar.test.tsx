import { describe, it, expect, vi } from "vitest"
import { screen } from "@testing-library/react"
import { renderWithProviders } from "@/test/render"
import { mockAppBindings, mockRuntime } from "@/test/mocks/bindings"
import { Sidebar, navItems } from "@/components/layout/sidebar"

vi.mock("@wailsjs/go/main/App", () => mockAppBindings)
vi.mock("@wailsjs/runtime/runtime", () => mockRuntime)

describe("Sidebar", () => {
  it("renders enabled nav items as links and disabled items without an href", () => {
    renderWithProviders(<Sidebar />)

    for (const item of navItems) {
      const node = screen.getByText(item.label)
      const link = node.closest("a")
      if (item.disabled) {
        expect(link).toBeNull()
      } else {
        expect(link).toHaveAttribute("href", item.href)
      }
    }

    expect(navItems).toHaveLength(5)
  })

  it("renders the Oversite brand", () => {
    renderWithProviders(<Sidebar />)
    expect(screen.getByText("Oversite")).toBeInTheDocument()
  })

  it("marks the Demos nav item active when on /demos", () => {
    renderWithProviders(<Sidebar />, { initialRoute: "/demos" })

    const demosLink = screen.getByText("Demos").closest("a")
    expect(demosLink).toHaveClass("active")
  })
})

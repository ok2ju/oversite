import { describe, it, expect, vi } from "vitest"
import { screen } from "@testing-library/react"
import { renderWithProviders } from "@/test/render"
import { Sidebar, navItems } from "@/components/layout/sidebar"

vi.mock("next/navigation", () => ({
  usePathname: vi.fn(() => "/dashboard"),
}))

describe("Sidebar", () => {
  it("renders all 6 navigation links with correct labels", () => {
    renderWithProviders(<Sidebar />)

    for (const item of navItems) {
      expect(screen.getByText(item.label)).toBeInTheDocument()
    }

    expect(navItems).toHaveLength(6)
  })

  it('renders the "Oversite" brand link', () => {
    renderWithProviders(<Sidebar />)

    const brandLink = screen.getByText("Oversite")
    expect(brandLink).toBeInTheDocument()
    expect(brandLink.closest("a")).toHaveAttribute("href", "/dashboard")
  })

  it("each link has the correct href", () => {
    renderWithProviders(<Sidebar />)

    for (const item of navItems) {
      const link = screen.getByText(item.label).closest("a")
      expect(link).toHaveAttribute("href", item.href)
    }
  })
})

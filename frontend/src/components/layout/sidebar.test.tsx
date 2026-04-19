import { describe, it, expect, vi } from "vitest"
import { screen } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { renderWithProviders } from "@/test/render"
import { Sidebar, navItems } from "@/components/layout/sidebar"

const logoutMock = vi.fn()

vi.mock("@/components/providers/auth-provider", () => ({
  useAuth: () => ({
    user: { user_id: "u1", faceit_id: "f1", nickname: "TestPlayer" },
    isLoading: false,
    isAuthenticated: true,
    logout: logoutMock,
  }),
}))

describe("Sidebar", () => {
  it("renders every nav item with correct href", () => {
    renderWithProviders(<Sidebar />)

    for (const item of navItems) {
      const link = screen.getByText(item.label).closest("a")
      expect(link).toHaveAttribute("href", item.href)
    }

    expect(navItems).toHaveLength(7)
  })

  it("renders the Oversite brand", () => {
    renderWithProviders(<Sidebar />)
    expect(screen.getByText("Oversite")).toBeInTheDocument()
  })

  it("marks the Demos nav item active when on /demos", () => {
    renderWithProviders(<Sidebar />, { initialRoute: "/demos" })

    const demosLink = screen.getByText("Demos").closest("a")
    expect(demosLink).toHaveClass("active")

    const dashboardLink = screen.getByText("Dashboard").closest("a")
    expect(dashboardLink).not.toHaveClass("active")
  })

  it("renders a logout button that calls logout when clicked", async () => {
    const user = userEvent.setup()
    renderWithProviders(<Sidebar />)

    const logoutBtn = screen.getByRole("button", { name: /log out/i })
    expect(logoutBtn).toBeEnabled()
    await user.click(logoutBtn)
    expect(logoutMock).toHaveBeenCalledTimes(1)
  })
})

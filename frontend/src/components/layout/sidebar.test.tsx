import { describe, it, expect, vi } from "vitest"
import { screen, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { renderWithProviders } from "@/test/render"
import { mockAppBindings, mockRuntime } from "@/test/mocks/bindings"
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

    expect(navItems).toHaveLength(6)
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

  it("opens the Faceit profile in the system browser", async () => {
    const user = userEvent.setup()
    renderWithProviders(<Sidebar />)

    const btn = await screen.findByRole("button", { name: /faceit account/i })
    await waitFor(() => expect(btn).toBeEnabled())
    await user.click(btn)

    expect(mockRuntime.BrowserOpenURL).toHaveBeenCalledWith(
      expect.stringContaining("faceit.com/en/players/"),
    )
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

import { describe, it, expect, vi } from "vitest"
import { screen } from "@testing-library/react"
import * as themeProvider from "@/components/providers/theme-provider"
import * as authProvider from "@/components/providers/auth-provider"
import { renderWithProviders, userEvent } from "@/test/render"
import { Header } from "@/components/layout/header"

const mockSetTheme = vi.fn()
const mockLogout = vi.fn<() => Promise<void>>().mockResolvedValue(undefined)

vi.spyOn(themeProvider, "useTheme").mockReturnValue({
  theme: "dark",
  setTheme: mockSetTheme,
})

vi.spyOn(authProvider, "useAuth").mockReturnValue({
  user: { user_id: "1", faceit_id: "abc", nickname: "TestPlayer" },
  isLoading: false,
  isAuthenticated: true,
  logout: mockLogout,
})

describe("Header", () => {
  it("renders the theme toggle button", () => {
    renderWithProviders(<Header />)

    expect(
      screen.getByRole("button", { name: "Toggle theme" }),
    ).toBeInTheDocument()
  })

  it('calls setTheme with "light" when current theme is "dark"', async () => {
    const user = userEvent.setup()
    renderWithProviders(<Header />)

    await user.click(screen.getByRole("button", { name: "Toggle theme" }))

    expect(mockSetTheme).toHaveBeenCalledWith("light")
  })

  it('calls setTheme with "dark" when current theme is "light"', async () => {
    vi.spyOn(themeProvider, "useTheme").mockReturnValue({
      theme: "light",
      setTheme: mockSetTheme,
    })

    const user = userEvent.setup()
    renderWithProviders(<Header />)

    await user.click(screen.getByRole("button", { name: "Toggle theme" }))

    expect(mockSetTheme).toHaveBeenCalledWith("dark")
  })

  it("displays the user nickname when authenticated", () => {
    renderWithProviders(<Header />)

    expect(screen.getByText("TestPlayer")).toBeInTheDocument()
  })

  it("renders logout button when authenticated", () => {
    renderWithProviders(<Header />)

    expect(screen.getByRole("button", { name: "Log out" })).toBeInTheDocument()
  })

  it("calls logout when logout button is clicked", async () => {
    const user = userEvent.setup()
    renderWithProviders(<Header />)

    await user.click(screen.getByRole("button", { name: "Log out" }))

    expect(mockLogout).toHaveBeenCalled()
  })

  it("hides logout button when not authenticated", () => {
    vi.spyOn(authProvider, "useAuth").mockReturnValue({
      user: null,
      isLoading: false,
      isAuthenticated: false,
      logout: mockLogout,
    })

    renderWithProviders(<Header />)

    expect(
      screen.queryByRole("button", { name: "Log out" }),
    ).not.toBeInTheDocument()
  })
})

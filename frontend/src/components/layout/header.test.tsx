import { describe, it, expect, vi } from "vitest"
import { screen } from "@testing-library/react"
import { useTheme } from "next-themes"
import { renderWithProviders, userEvent } from "@/test/render"
import { Header } from "@/components/layout/header"

const mockSetTheme = vi.fn()

vi.mock("next-themes", () => ({
  useTheme: vi.fn(() => ({
    theme: "dark",
    setTheme: mockSetTheme,
    resolvedTheme: "dark",
    themes: ["light", "dark"],
    systemTheme: undefined,
    forcedTheme: undefined,
  })),
  ThemeProvider: ({ children }: { children: React.ReactNode }) => children,
}))

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
    vi.mocked(useTheme).mockReturnValue({
      theme: "light",
      setTheme: mockSetTheme,
      resolvedTheme: "light",
      themes: ["light", "dark"],
      systemTheme: undefined,
      forcedTheme: undefined,
    })

    const user = userEvent.setup()
    renderWithProviders(<Header />)

    await user.click(screen.getByRole("button", { name: "Toggle theme" }))

    expect(mockSetTheme).toHaveBeenCalledWith("dark")
  })
})

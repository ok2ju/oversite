import { describe, it, expect, vi } from "vitest"
import { screen } from "@testing-library/react"
import * as themeProvider from "@/components/providers/theme-provider"
import { renderWithProviders, userEvent } from "@/test/render"
import { Header } from "@/components/layout/header"

const mockSetTheme = vi.fn()

vi.spyOn(themeProvider, "useTheme").mockReturnValue({
  theme: "dark",
  setTheme: mockSetTheme,
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
})

import { screen } from "@testing-library/react"
import { describe, expect, it } from "vitest"
import { renderWithProviders } from "@/test/render"
import LoginPage from "@/routes/login"

describe("LoginPage", () => {
  it("renders app title", () => {
    renderWithProviders(<LoginPage />)
    expect(screen.getByText("Oversite")).toBeInTheDocument()
  })

  it("renders sign in with Faceit button", () => {
    renderWithProviders(<LoginPage />)
    expect(
      screen.getByRole("link", { name: /sign in with faceit/i }),
    ).toBeInTheDocument()
  })

  it("button links to the backend OAuth endpoint", () => {
    renderWithProviders(<LoginPage />)
    const link = screen.getByRole("link", { name: /sign in with faceit/i })
    expect(link).toHaveAttribute("href", "/api/v1/auth/faceit")
  })
})

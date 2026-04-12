import { screen } from "@testing-library/react"
import { describe, expect, it, vi } from "vitest"
import { renderWithProviders, userEvent } from "@/test/render"
import { mockAppBindings } from "@/test/mocks/bindings"
import LoginPage from "@/routes/login"

vi.mock("@wailsjs/go/main/App", () => mockAppBindings)

describe("LoginPage", () => {
  it("renders app title", () => {
    renderWithProviders(<LoginPage />)
    expect(screen.getByText("Oversite")).toBeInTheDocument()
  })

  it("renders sign in with Faceit button", () => {
    renderWithProviders(<LoginPage />)
    expect(
      screen.getByRole("button", { name: /sign in with faceit/i }),
    ).toBeInTheDocument()
  })

  it("calls LoginWithFaceit binding on click", async () => {
    const user = userEvent.setup()
    renderWithProviders(<LoginPage />)

    await user.click(
      screen.getByRole("button", { name: /sign in with faceit/i }),
    )
    expect(mockAppBindings.LoginWithFaceit).toHaveBeenCalled()
  })
})

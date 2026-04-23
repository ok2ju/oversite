import { screen } from "@testing-library/react"
import { describe, expect, it, vi } from "vitest"
import { renderWithProviders, userEvent } from "@/test/render"
import { mockAppBindings } from "@/test/mocks/bindings"
import LoginPage from "@/routes/login"

vi.mock("@wailsjs/go/main/App", () => mockAppBindings)

describe("LoginPage (first-run hero)", () => {
  it("renders the Empty/first-run hero title", () => {
    renderWithProviders(<LoginPage />)
    expect(
      screen.getByText("Connect Faceit to get started"),
    ).toBeInTheDocument()
  })

  it("renders the Connect Faceit primary CTA", () => {
    renderWithProviders(<LoginPage />)
    expect(
      screen.getByRole("button", { name: /connect faceit account/i }),
    ).toBeInTheDocument()
  })

  it("renders the 3-step onboarding grid", () => {
    renderWithProviders(<LoginPage />)
    expect(screen.getByText("Sign in")).toBeInTheDocument()
    expect(screen.getByText("Pick a folder")).toBeInTheDocument()
    expect(screen.getByText("Review & watch")).toBeInTheDocument()
  })

  it("calls LoginWithFaceit when the primary CTA is clicked", async () => {
    const user = userEvent.setup()
    renderWithProviders(<LoginPage />)

    await user.click(
      screen.getByRole("button", { name: /connect faceit account/i }),
    )
    expect(mockAppBindings.LoginWithFaceit).toHaveBeenCalled()
  })
})

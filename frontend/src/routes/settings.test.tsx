import { screen } from "@testing-library/react"
import { describe, expect, it, vi } from "vitest"
import { renderWithProviders } from "@/test/render"
import { mockAppBindings } from "@/test/mocks/bindings"
import SettingsPage from "@/routes/settings"

vi.mock("@wailsjs/go/main/App", () => mockAppBindings)

describe("SettingsPage", () => {
  it("renders heading", () => {
    renderWithProviders(<SettingsPage />, { initialRoute: "/settings" })
    expect(
      screen.getByRole("heading", { name: "Settings" }),
    ).toBeInTheDocument()
  })

  it("renders description text", () => {
    renderWithProviders(<SettingsPage />)
    expect(
      screen.getByText("Manage your account and preferences"),
    ).toBeInTheDocument()
  })
})

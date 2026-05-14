import { describe, it, expect } from "vitest"
import { screen } from "@testing-library/react"
import { renderWithProviders } from "@/test/render"
import { Header } from "@/components/layout/header"

describe("Header", () => {
  it("renders a derived page title for the current route", () => {
    renderWithProviders(<Header />, { initialRoute: "/demos" })
    expect(
      screen
        .getAllByText("Demos library")
        .some((el) => el.classList.contains("page-title")),
    ).toBe(true)
  })

  it("prefers an explicit title prop over the derived title", () => {
    renderWithProviders(<Header title="Custom" subtitle="Sub text" />)
    expect(
      screen
        .getAllByText("Custom")
        .some((el) => el.classList.contains("page-title")),
    ).toBe(true)
    expect(screen.getByText("Sub text")).toBeInTheDocument()
  })

  it("renders action slot content", () => {
    renderWithProviders(
      <Header actions={<button type="button">Custom action</button>} />,
    )
    expect(
      screen.getByRole("button", { name: "Custom action" }),
    ).toBeInTheDocument()
  })

  it("renders demo viewer title for /demos/:id routes", () => {
    renderWithProviders(<Header />, { initialRoute: "/demos/123" })
    expect(
      screen
        .getAllByText("Demo viewer")
        .some((el) => el.classList.contains("page-title")),
    ).toBe(true)
  })

  it("hides the breadcrumb on the home (/demos) route", () => {
    renderWithProviders(<Header />, { initialRoute: "/demos" })
    expect(screen.queryByRole("link", { name: "Home" })).toBeNull()
  })

  it("renders a breadcrumb on non-home routes", () => {
    renderWithProviders(<Header />, { initialRoute: "/demos/123/overview" })
    const home = screen.getByRole("link", { name: "Home" })
    expect(home).toHaveAttribute("href", "/demos")
  })
})

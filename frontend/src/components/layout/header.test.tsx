import { describe, it, expect, vi } from "vitest"
import { screen } from "@testing-library/react"
import { renderWithProviders } from "@/test/render"
import { Header } from "@/components/layout/header"

vi.mock("@/components/providers/auth-provider", () => ({
  useAuth: () => ({
    user: { user_id: "u1", faceit_id: "f1", nickname: "TestPlayer" },
    isLoading: false,
    isAuthenticated: true,
    logout: vi.fn(),
  }),
}))

vi.mock("@/hooks/use-faceit-sync", () => ({
  useFaceitSync: () => ({
    mutate: vi.fn(),
    isPending: false,
  }),
}))

describe("Header", () => {
  it("renders a derived page title for the current route", () => {
    renderWithProviders(<Header />, { initialRoute: "/dashboard" })
    expect(
      screen
        .getAllByText("Dashboard")
        .some((el) => el.classList.contains("page-title")),
    ).toBe(true)
  })

  it("prefers an explicit title prop over the derived title", () => {
    renderWithProviders(<Header title="Custom" subtitle="Sub text" />)
    expect(screen.getByText("Custom")).toBeInTheDocument()
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

  it("renders match detail title for /matches/:id routes", () => {
    renderWithProviders(<Header />, { initialRoute: "/matches/123" })
    expect(
      screen
        .getAllByText("Match detail")
        .some((el) => el.classList.contains("page-title")),
    ).toBe(true)
  })

  it("renders a breadcrumb with Home link", () => {
    renderWithProviders(<Header />, { initialRoute: "/demos" })
    const home = screen.getByRole("link", { name: "Home" })
    expect(home).toHaveAttribute("href", "/dashboard")
  })

  it("renders a sync button by default", () => {
    renderWithProviders(<Header />, { initialRoute: "/dashboard" })
    expect(
      screen.getByRole("button", { name: /sync faceit data/i }),
    ).toBeInTheDocument()
  })
})

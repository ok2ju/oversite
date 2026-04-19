import { describe, it, expect, vi, beforeEach } from "vitest"
import { screen } from "@testing-library/react"
import { renderWithProviders, userEvent } from "@/test/render"
import { MatchToolbar } from "@/components/match/toolbar"

const mockNavigate = vi.fn()
vi.mock("react-router-dom", async () => {
  const actual =
    await vi.importActual<typeof import("react-router-dom")>("react-router-dom")
  return { ...actual, useNavigate: () => mockNavigate }
})

describe("MatchToolbar", () => {
  beforeEach(() => {
    mockNavigate.mockReset()
  })

  it("renders a Play demo button enabled when status is ready", () => {
    renderWithProviders(
      <MatchToolbar matchId="m-1" demoId={42} demoStatus="ready" />,
    )
    const btn = screen.getByRole("button", { name: /play demo/i })
    expect(btn).toBeEnabled()
  })

  it("disables Play demo when demoId is missing", () => {
    renderWithProviders(<MatchToolbar matchId="m-1" demoId={null} />)
    expect(screen.getByRole("button", { name: /play demo/i })).toBeDisabled()
  })

  it("disables Play demo when status is not ready", () => {
    renderWithProviders(
      <MatchToolbar matchId="m-1" demoId={42} demoStatus="parsing" />,
    )
    expect(screen.getByRole("button", { name: /play demo/i })).toBeDisabled()
  })

  it("navigates to /demos/:id when Play demo is clicked", async () => {
    const user = userEvent.setup()
    renderWithProviders(
      <MatchToolbar matchId="m-1" demoId={42} demoStatus="ready" />,
    )
    await user.click(screen.getByRole("button", { name: /play demo/i }))
    expect(mockNavigate).toHaveBeenCalledWith("/demos/42")
  })
})

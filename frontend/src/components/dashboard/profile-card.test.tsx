import { describe, it, expect } from "vitest"
import { screen } from "@testing-library/react"
import { renderWithProviders } from "@/test/render"
import { ProfileCard } from "@/components/dashboard/profile-card"
import type { FaceitProfile } from "@/types/faceit"

function mockProfile(overrides?: Partial<FaceitProfile>): FaceitProfile {
  return {
    nickname: "TestPlayer",
    avatar_url: "https://example.com/avatar.png",
    elo: 1850,
    level: 8,
    country: "US",
    matches_played: 142,
    current_streak: { type: "win", count: 3 },
    ...overrides,
  }
}

describe("ProfileCard", () => {
  it("renders nickname, ELO, level, and matches played", () => {
    renderWithProviders(
      <ProfileCard profile={mockProfile()} isLoading={false} />,
    )

    expect(screen.getByText("TestPlayer")).toBeInTheDocument()
    expect(screen.getByText("1850")).toBeInTheDocument()
    expect(screen.getByText("Level 8")).toBeInTheDocument()
    expect(screen.getByText("142")).toBeInTheDocument()
  })

  it("renders country flag", () => {
    renderWithProviders(
      <ProfileCard profile={mockProfile()} isLoading={false} />,
    )

    expect(screen.getByLabelText("US")).toBeInTheDocument()
  })

  it("renders win streak in green", () => {
    renderWithProviders(
      <ProfileCard profile={mockProfile()} isLoading={false} />,
    )

    const streak = screen.getByText("W3")
    expect(streak).toBeInTheDocument()
    expect(streak.className).toContain("text-green-500")
  })

  it("renders loss streak in red", () => {
    renderWithProviders(
      <ProfileCard
        profile={mockProfile({ current_streak: { type: "loss", count: 2 } })}
        isLoading={false}
      />,
    )

    const streak = screen.getByText("L2")
    expect(streak).toBeInTheDocument()
    expect(streak.className).toContain("text-red-500")
  })

  it("shows loading skeletons when isLoading is true", () => {
    renderWithProviders(
      <ProfileCard profile={undefined} isLoading={true} />,
    )

    expect(screen.getByTestId("profile-card-skeleton")).toBeInTheDocument()
  })

  it("handles null avatar gracefully with fallback icon", () => {
    renderWithProviders(
      <ProfileCard
        profile={mockProfile({ avatar_url: null })}
        isLoading={false}
      />,
    )

    expect(screen.getByText("TestPlayer")).toBeInTheDocument()
    expect(screen.queryByRole("img")).not.toBeInTheDocument()
  })

  it("handles null country gracefully", () => {
    renderWithProviders(
      <ProfileCard
        profile={mockProfile({ country: null })}
        isLoading={false}
      />,
    )

    expect(screen.getByText("TestPlayer")).toBeInTheDocument()
    expect(screen.queryByLabelText("US")).not.toBeInTheDocument()
  })

  it("shows dash for no streak", () => {
    renderWithProviders(
      <ProfileCard
        profile={mockProfile({ current_streak: { type: "none", count: 0 } })}
        isLoading={false}
      />,
    )

    expect(screen.getByText("-")).toBeInTheDocument()
  })

  it("returns null when no profile and not loading", () => {
    renderWithProviders(
      <ProfileCard profile={undefined} isLoading={false} />,
    )

    expect(screen.queryByText("ELO")).not.toBeInTheDocument()
    expect(screen.queryByTestId("profile-card-skeleton")).not.toBeInTheDocument()
  })
})

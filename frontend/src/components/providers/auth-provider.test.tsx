import { screen, waitFor } from "@testing-library/react"
import { http, HttpResponse } from "msw"
import { describe, expect, it, vi, beforeEach } from "vitest"
import { renderWithProviders } from "@/test/render"
import { server } from "@/test/msw/server"
import { AuthProvider, useAuth } from "./auth-provider"

// Mock next/navigation
const mockPush = vi.fn()
const mockPathname = vi.fn(() => "/dashboard")
vi.mock("next/navigation", () => ({
  useRouter: () => ({ push: mockPush }),
  usePathname: () => mockPathname(),
}))

function AuthConsumer() {
  const { user, isLoading, isAuthenticated } = useAuth()
  return (
    <div>
      <span data-testid="loading">{String(isLoading)}</span>
      <span data-testid="authenticated">{String(isAuthenticated)}</span>
      {user && <span data-testid="nickname">{user.nickname}</span>}
    </div>
  )
}

describe("AuthProvider", () => {
  beforeEach(() => {
    mockPush.mockClear()
    mockPathname.mockReturnValue("/dashboard")
  })

  it("shows loading state while checking session", () => {
    // Use a handler that never resolves to keep loading state
    server.use(
      http.get("/api/v1/auth/me", () => {
        return new Promise(() => {})
      }),
    )

    renderWithProviders(
      <AuthProvider>
        <AuthConsumer />
      </AuthProvider>,
    )

    expect(screen.getByTestId("loading")).toHaveTextContent("true")
    expect(screen.getByTestId("authenticated")).toHaveTextContent("false")
  })

  it("renders children when authenticated", async () => {
    // Default MSW handler returns 200 with user data
    renderWithProviders(
      <AuthProvider>
        <AuthConsumer />
      </AuthProvider>,
    )

    await waitFor(() => {
      expect(screen.getByTestId("loading")).toHaveTextContent("false")
    })

    expect(screen.getByTestId("authenticated")).toHaveTextContent("true")
    expect(screen.getByTestId("nickname")).toHaveTextContent("TestPlayer")
  })

  it("redirects to /login when unauthenticated", async () => {
    server.use(
      http.get("/api/v1/auth/me", () => {
        return HttpResponse.json(
          { error: "unauthorized" },
          { status: 401 },
        )
      }),
    )

    renderWithProviders(
      <AuthProvider>
        <AuthConsumer />
      </AuthProvider>,
    )

    await waitFor(() => {
      expect(mockPush).toHaveBeenCalledWith("/login")
    })
  })

  it("does not redirect when on /login page", async () => {
    mockPathname.mockReturnValue("/login")

    server.use(
      http.get("/api/v1/auth/me", () => {
        return HttpResponse.json(
          { error: "unauthorized" },
          { status: 401 },
        )
      }),
    )

    renderWithProviders(
      <AuthProvider>
        <AuthConsumer />
      </AuthProvider>,
    )

    await waitFor(() => {
      expect(screen.getByTestId("loading")).toHaveTextContent("false")
    })

    expect(mockPush).not.toHaveBeenCalled()
  })

  it("does not redirect when on /callback page", async () => {
    mockPathname.mockReturnValue("/callback")

    server.use(
      http.get("/api/v1/auth/me", () => {
        return HttpResponse.json(
          { error: "unauthorized" },
          { status: 401 },
        )
      }),
    )

    renderWithProviders(
      <AuthProvider>
        <AuthConsumer />
      </AuthProvider>,
    )

    await waitFor(() => {
      expect(screen.getByTestId("loading")).toHaveTextContent("false")
    })

    expect(mockPush).not.toHaveBeenCalled()
  })

  it("exposes correct user data after successful auth check", async () => {
    server.use(
      http.get("/api/v1/auth/me", () => {
        return HttpResponse.json({
          user_id: "custom-id",
          faceit_id: "faceit-123",
          nickname: "ProPlayer",
        })
      }),
    )

    renderWithProviders(
      <AuthProvider>
        <AuthConsumer />
      </AuthProvider>,
    )

    await waitFor(() => {
      expect(screen.getByTestId("authenticated")).toHaveTextContent("true")
    })

    expect(screen.getByTestId("nickname")).toHaveTextContent("ProPlayer")
  })
})

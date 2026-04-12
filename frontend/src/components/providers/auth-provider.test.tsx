import { screen, waitFor } from "@testing-library/react"
import { http, HttpResponse } from "msw"
import { describe, expect, it, vi, beforeEach } from "vitest"
import { renderWithProviders } from "@/test/render"
import { server } from "@/test/msw/server"
import { AuthProvider, useAuth } from "./auth-provider"

const mockNavigate = vi.fn()
vi.mock("react-router-dom", async () => {
  const actual = await vi.importActual("react-router-dom")
  return { ...actual, useNavigate: () => mockNavigate }
})

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
    mockNavigate.mockClear()
  })

  it("shows loading state while checking session", () => {
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
        return HttpResponse.json({ error: "unauthorized" }, { status: 401 })
      }),
    )

    renderWithProviders(
      <AuthProvider>
        <AuthConsumer />
      </AuthProvider>,
      { initialRoute: "/dashboard" },
    )

    await waitFor(() => {
      expect(mockNavigate).toHaveBeenCalledWith("/login")
    })
  })

  it("does not redirect when on /login page", async () => {
    server.use(
      http.get("/api/v1/auth/me", () => {
        return HttpResponse.json({ error: "unauthorized" }, { status: 401 })
      }),
    )

    renderWithProviders(
      <AuthProvider>
        <AuthConsumer />
      </AuthProvider>,
      { initialRoute: "/login" },
    )

    await waitFor(() => {
      expect(screen.getByTestId("loading")).toHaveTextContent("false")
    })

    expect(mockNavigate).not.toHaveBeenCalled()
  })

  it("does not redirect when on /callback page", async () => {
    server.use(
      http.get("/api/v1/auth/me", () => {
        return HttpResponse.json({ error: "unauthorized" }, { status: 401 })
      }),
    )

    renderWithProviders(
      <AuthProvider>
        <AuthConsumer />
      </AuthProvider>,
      { initialRoute: "/callback" },
    )

    await waitFor(() => {
      expect(screen.getByTestId("loading")).toHaveTextContent("false")
    })

    expect(mockNavigate).not.toHaveBeenCalled()
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

import { createContext, useContext, useCallback, useEffect } from "react"
import { useQuery, useQueryClient } from "@tanstack/react-query"
import { useNavigate, useLocation } from "react-router-dom"
import { GetCurrentUser, Logout } from "@wailsjs/go/main/App"

export interface User {
  user_id: string
  faceit_id: string
  nickname: string
}

export interface AuthContextValue {
  user: User | null
  isLoading: boolean
  isAuthenticated: boolean
  logout: () => Promise<void>
}

const AuthContext = createContext<AuthContextValue | null>(null)

const PUBLIC_PATHS = ["/login", "/callback"]

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const navigate = useNavigate()
  const { pathname } = useLocation()
  const queryClient = useQueryClient()

  const {
    data: user,
    isLoading,
    isError,
  } = useQuery<User>({
    queryKey: ["auth", "me"],
    queryFn: () => GetCurrentUser() as Promise<User>,
    retry: false,
  })

  const isAuthenticated = !!user && !isError
  const isPublicPath = PUBLIC_PATHS.some((path) => pathname.startsWith(path))

  useEffect(() => {
    if (!isLoading && !isAuthenticated && !isPublicPath) {
      navigate("/login")
    }
  }, [isLoading, isAuthenticated, isPublicPath, navigate])

  const logout = useCallback(async () => {
    await Logout()
    queryClient.invalidateQueries({ queryKey: ["auth", "me"] })
    navigate("/login")
  }, [queryClient, navigate])

  return (
    <AuthContext.Provider
      value={{ user: user ?? null, isLoading, isAuthenticated, logout }}
    >
      {children}
    </AuthContext.Provider>
  )
}

export function useAuth(): AuthContextValue {
  const context = useContext(AuthContext)
  if (!context) {
    throw new Error("useAuth must be used within an AuthProvider")
  }
  return context
}

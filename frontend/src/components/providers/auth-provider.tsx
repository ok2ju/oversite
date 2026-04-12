import { createContext, useContext, useEffect } from "react"
import { useQuery } from "@tanstack/react-query"
import { useNavigate, useLocation } from "react-router-dom"

export interface User {
  user_id: string
  faceit_id: string
  nickname: string
}

export interface AuthContextValue {
  user: User | null
  isLoading: boolean
  isAuthenticated: boolean
}

const AuthContext = createContext<AuthContextValue | null>(null)

const PUBLIC_PATHS = ["/login", "/callback"]

async function fetchCurrentUser(): Promise<User> {
  const res = await fetch("/api/v1/auth/me")
  if (!res.ok) {
    throw new Error("Unauthorized")
  }
  const json = await res.json()
  return json
}

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const navigate = useNavigate()
  const { pathname } = useLocation()

  const {
    data: user,
    isLoading,
    isError,
  } = useQuery<User>({
    queryKey: ["auth", "me"],
    queryFn: fetchCurrentUser,
    retry: false,
  })

  const isAuthenticated = !!user && !isError
  const isPublicPath = PUBLIC_PATHS.some((path) => pathname.startsWith(path))

  useEffect(() => {
    if (!isLoading && !isAuthenticated && !isPublicPath) {
      navigate("/login")
    }
  }, [isLoading, isAuthenticated, isPublicPath, navigate])

  return (
    <AuthContext.Provider
      value={{ user: user ?? null, isLoading, isAuthenticated }}
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

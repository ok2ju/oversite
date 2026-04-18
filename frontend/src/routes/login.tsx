import { useState } from "react"
import { Button } from "@/components/ui/button"
import { LoginWithFaceit } from "@wailsjs/go/main/App"
import { useQueryClient } from "@tanstack/react-query"
import { useNavigate } from "react-router-dom"

export default function LoginPage() {
  const [isLoading, setIsLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const queryClient = useQueryClient()
  const navigate = useNavigate()

  async function handleLogin() {
    setIsLoading(true)
    setError(null)
    try {
      await LoginWithFaceit()
      await queryClient.invalidateQueries({ queryKey: ["auth", "me"] })
      await queryClient.invalidateQueries({ queryKey: ["faceit"] })
      navigate("/dashboard")
    } catch (err) {
      setError(err instanceof Error ? err.message : "Login failed")
    } finally {
      setIsLoading(false)
    }
  }

  return (
    <div className="flex min-h-screen flex-col items-center justify-center gap-8">
      <div className="flex flex-col items-center gap-2">
        <h1 className="text-4xl font-bold tracking-tight">Oversite</h1>
        <p className="text-muted-foreground">
          CS2 demo viewer and analytics platform
        </p>
      </div>
      {error && (
        <p className="max-w-sm text-center text-sm text-destructive">{error}</p>
      )}
      <Button size="lg" onClick={handleLogin} disabled={isLoading}>
        {isLoading ? "Signing in..." : "Sign in with Faceit"}
      </Button>
      {isLoading && (
        <p className="text-sm text-muted-foreground">
          Complete sign-in in your browser...
        </p>
      )}
    </div>
  )
}

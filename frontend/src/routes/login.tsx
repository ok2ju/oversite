import { useState } from "react"
import { useQueryClient } from "@tanstack/react-query"
import { useNavigate } from "react-router-dom"
import { LoginWithFaceit } from "@wailsjs/go/main/App"
import { EmptyHero } from "@/components/auth/empty-hero"

export default function LoginPage() {
  const [isLoading, setIsLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const queryClient = useQueryClient()
  const navigate = useNavigate()

  async function handleConnect() {
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
    <EmptyHero
      onConnect={handleConnect}
      onSkip={() => navigate("/dashboard")}
      isLoading={isLoading}
      error={error}
    />
  )
}

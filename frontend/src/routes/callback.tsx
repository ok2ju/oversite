import { useEffect } from "react"
import { useNavigate } from "react-router-dom"

export default function CallbackPage() {
  const navigate = useNavigate()

  useEffect(() => {
    navigate("/dashboard")
  }, [navigate])

  return (
    <div className="flex min-h-screen items-center justify-center">
      <div className="flex flex-col items-center gap-4">
        <div className="h-8 w-8 animate-spin rounded-full border-4 border-primary border-t-transparent" />
        <p className="text-sm text-muted-foreground">Completing sign in...</p>
      </div>
    </div>
  )
}

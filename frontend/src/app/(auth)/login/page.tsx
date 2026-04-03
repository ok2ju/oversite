import { Button } from "@/components/ui/button"

export default function LoginPage() {
  return (
    <div className="flex min-h-screen flex-col items-center justify-center gap-8">
      <div className="flex flex-col items-center gap-2">
        <h1 className="text-4xl font-bold tracking-tight">Oversite</h1>
        <p className="text-muted-foreground">CS2 demo viewer and analytics platform</p>
      </div>
      <Button asChild size="lg">
        <a href="/api/v1/auth/faceit">Sign in with Faceit</a>
      </Button>
    </div>
  )
}

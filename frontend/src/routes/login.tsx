import { Button } from "@/components/ui/button"
import { LoginWithFaceit } from "@wailsjs/go/main/App"

export default function LoginPage() {
  return (
    <div className="flex min-h-screen flex-col items-center justify-center gap-8">
      <div className="flex flex-col items-center gap-2">
        <h1 className="text-4xl font-bold tracking-tight">Oversite</h1>
        <p className="text-muted-foreground">
          CS2 demo viewer and analytics platform
        </p>
      </div>
      <Button size="lg" onClick={() => LoginWithFaceit()}>
        Sign in with Faceit
      </Button>
    </div>
  )
}

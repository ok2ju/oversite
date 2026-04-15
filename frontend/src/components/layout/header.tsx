import { Moon, Sun, LogOut } from "lucide-react"
import { useTheme } from "@/components/providers/theme-provider"
import { useAuth } from "@/components/providers/auth-provider"
import { Button } from "@/components/ui/button"

export function Header() {
  const { theme, setTheme } = useTheme()
  const { user, isAuthenticated, logout } = useAuth()

  return (
    <header className="flex h-14 items-center justify-between border-b bg-card px-6">
      <div>
        {isAuthenticated && user && (
          <span className="text-sm text-muted-foreground">{user.nickname}</span>
        )}
      </div>
      <div className="flex items-center gap-2">
        <Button
          variant="ghost"
          size="icon"
          onClick={() => setTheme(theme === "dark" ? "light" : "dark")}
          aria-label="Toggle theme"
        >
          <Sun className="h-4 w-4 rotate-0 scale-100 transition-all dark:-rotate-90 dark:scale-0" />
          <Moon className="absolute h-4 w-4 rotate-90 scale-0 transition-all dark:rotate-0 dark:scale-100" />
        </Button>
        {isAuthenticated && (
          <Button
            variant="ghost"
            size="icon"
            onClick={logout}
            aria-label="Log out"
          >
            <LogOut className="h-4 w-4" />
          </Button>
        )}
      </div>
    </header>
  )
}

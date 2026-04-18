import { useFaceitProfile } from "@/hooks/use-faceit"
import { useFaceitSync } from "@/hooks/use-faceit-sync"
import { useFaceitSyncProgress } from "@/hooks/use-faceit-sync-progress"
import { ProfileCard } from "@/components/dashboard/profile-card"
import { MatchList } from "@/components/dashboard/match-list"
import { Button } from "@/components/ui/button"

export default function DashboardPage() {
  const { data: profile, isLoading: profileLoading } = useFaceitProfile()
  const sync = useFaceitSync()
  const { progress, reset: resetProgress } = useFaceitSyncProgress()

  function handleSync() {
    resetProgress()
    sync.mutate(undefined, {
      onSettled: () => {
        setTimeout(resetProgress, 2000)
      },
    })
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">Dashboard</h1>
          <p className="mt-1 text-muted-foreground">
            Your recent Faceit matches
          </p>
        </div>
        <div className="flex items-center gap-3">
          {sync.isPending && progress && (
            <span className="text-sm text-muted-foreground">
              Syncing... {progress.current}/{progress.total}
            </span>
          )}
          <Button
            onClick={handleSync}
            disabled={sync.isPending}
            data-testid="sync-button"
          >
            {sync.isPending ? "Syncing..." : "Sync Matches"}
          </Button>
        </div>
      </div>

      <ProfileCard profile={profile} isLoading={profileLoading} />
      <MatchList />
    </div>
  )
}

import { useEffect } from "react"
import { useOutletContext } from "react-router-dom"
import { useFaceitProfile } from "@/hooks/use-faceit"
import { useParseProgress } from "@/hooks/use-parse-progress"
import { ProfileHero } from "@/components/dashboard/profile-hero"
import { RecentMatches } from "@/components/dashboard/recent-matches"
import { DashboardHeaderActions } from "@/components/dashboard/dashboard-header-actions"
import type { HeaderActionsContext } from "@/routes/root"

export default function DashboardPage() {
  const { data: profile, isLoading: profileLoading } = useFaceitProfile()
  const ctx = useOutletContext<HeaderActionsContext | undefined>()
  useParseProgress()

  useEffect(() => {
    ctx?.setHeaderActions(<DashboardHeaderActions />)
    return () => ctx?.setHeaderActions(null)
  }, [ctx])

  return (
    <div className="flex flex-col gap-[18px]">
      <ProfileHero profile={profile} isLoading={profileLoading} />
      <RecentMatches />
    </div>
  )
}

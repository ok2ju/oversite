import { useFaceitProfile } from "@/hooks/use-faceit"
import { useParseProgress } from "@/hooks/use-parse-progress"
import { ProfileHero } from "@/components/dashboard/profile-hero"
import { RecentMatches } from "@/components/dashboard/recent-matches"

export default function DashboardPage() {
  const { data: profile, isLoading: profileLoading } = useFaceitProfile()
  useParseProgress()

  return (
    <div className="flex flex-col gap-[18px]">
      <ProfileHero profile={profile} isLoading={profileLoading} />
      <RecentMatches />
    </div>
  )
}

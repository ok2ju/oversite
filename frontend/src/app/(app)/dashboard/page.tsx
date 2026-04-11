"use client"

import { useState } from "react"
import { useFaceitProfile, useEloHistory } from "@/hooks/use-faceit"
import { ProfileCard } from "@/components/dashboard/profile-card"
import { EloChart } from "@/components/dashboard/elo-chart"

export default function DashboardPage() {
  const { data: profile, isLoading: profileLoading } = useFaceitProfile()
  const [days, setDays] = useState(30)
  const { data: eloHistory, isLoading: eloLoading } = useEloHistory(days)

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold">Dashboard</h1>
        <p className="mt-1 text-muted-foreground">
          Your Faceit stats and ELO history
        </p>
      </div>

      <ProfileCard profile={profile} isLoading={profileLoading} />
      <EloChart
        data={eloHistory}
        isLoading={eloLoading}
        days={days}
        onDaysChange={setDays}
      />
    </div>
  )
}

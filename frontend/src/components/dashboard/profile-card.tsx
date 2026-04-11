import Image from "next/image"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Badge } from "@/components/ui/badge"
import { Skeleton } from "@/components/ui/skeleton"
import { User } from "lucide-react"
import type { FaceitProfile } from "@/types/faceit"

interface ProfileCardProps {
  profile: FaceitProfile | undefined
  isLoading: boolean
}

const levelColors: Record<number, string> = {
  1: "bg-zinc-500",
  2: "bg-zinc-400",
  3: "bg-yellow-600",
  4: "bg-yellow-500",
  5: "bg-orange-500",
  6: "bg-orange-400",
  7: "bg-red-500",
  8: "bg-red-400",
  9: "bg-purple-500",
  10: "bg-purple-400",
}

function countryFlag(code: string): string {
  return String.fromCodePoint(
    ...code
      .toUpperCase()
      .split("")
      .map((c) => 0x1f1e6 + c.charCodeAt(0) - 65),
  )
}

export function ProfileCard({ profile, isLoading }: ProfileCardProps) {
  if (isLoading) {
    return (
      <Card data-testid="profile-card-skeleton">
        <CardHeader>
          <div className="flex items-center gap-4">
            <Skeleton className="h-16 w-16 rounded-full" />
            <div className="space-y-2">
              <Skeleton className="h-5 w-32" />
              <Skeleton className="h-4 w-20" />
            </div>
          </div>
        </CardHeader>
        <CardContent>
          <div className="flex gap-6">
            <Skeleton className="h-8 w-16" />
            <Skeleton className="h-8 w-16" />
            <Skeleton className="h-8 w-16" />
          </div>
        </CardContent>
      </Card>
    )
  }

  if (!profile) return null

  const streakColor =
    profile.current_streak.type === "win"
      ? "text-green-500"
      : profile.current_streak.type === "loss"
        ? "text-red-500"
        : "text-muted-foreground"

  const streakLabel =
    profile.current_streak.type === "none"
      ? "-"
      : `${profile.current_streak.type === "win" ? "W" : profile.current_streak.type === "loss" ? "L" : "D"}${profile.current_streak.count}`

  return (
    <Card>
      <CardHeader>
        <div className="flex items-center gap-4">
          {profile.avatar_url ? (
            <Image
              src={profile.avatar_url}
              alt={profile.nickname}
              width={64}
              height={64}
              className="h-16 w-16 rounded-full"
            />
          ) : (
            <div className="flex h-16 w-16 items-center justify-center rounded-full bg-muted">
              <User className="h-8 w-8 text-muted-foreground" />
            </div>
          )}
          <div>
            <CardTitle className="flex items-center gap-2">
              {profile.nickname}
              {profile.country && (
                <span aria-label={profile.country}>
                  {countryFlag(profile.country)}
                </span>
              )}
            </CardTitle>
            {profile.level != null && (
              <Badge
                className={`mt-1 ${levelColors[profile.level] ?? "bg-zinc-500"} text-white`}
              >
                Level {profile.level}
              </Badge>
            )}
          </div>
        </div>
      </CardHeader>
      <CardContent>
        <div className="flex gap-6">
          <div>
            <p className="text-sm text-muted-foreground">ELO</p>
            <p className="text-xl font-bold">{profile.elo ?? "-"}</p>
          </div>
          <div>
            <p className="text-sm text-muted-foreground">Matches</p>
            <p className="text-xl font-bold">{profile.matches_played}</p>
          </div>
          <div>
            <p className="text-sm text-muted-foreground">Streak</p>
            <p className={`text-xl font-bold ${streakColor}`}>{streakLabel}</p>
          </div>
        </div>
      </CardContent>
    </Card>
  )
}

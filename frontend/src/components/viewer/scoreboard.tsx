import { useViewerStore } from "@/stores/viewer"
import { useScoreboard } from "@/hooks/use-scoreboard"
import type { ScoreboardEntry } from "@/types/scoreboard"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"

interface ScoreboardProps {
  visible: boolean
}

function TeamTable({
  entries,
  side,
  selectedSteamId,
  onSelectPlayer,
}: {
  entries: ScoreboardEntry[]
  side: "CT" | "T"
  selectedSteamId: string | null
  onSelectPlayer: (steamId: string | null) => void
}) {
  const sideLabel = side === "CT" ? "Counter-Terrorists" : "Terrorists"
  const sideColor = side === "CT" ? "text-sky-400" : "text-amber-400"
  const headerBorder =
    side === "CT" ? "border-sky-400/40" : "border-amber-400/40"

  return (
    <div data-testid={`scoreboard-team-${side.toLowerCase()}`}>
      <h3 className={`mb-2 text-sm font-semibold ${sideColor}`}>{sideLabel}</h3>
      <Table>
        <TableHeader>
          <TableRow className={`border-b ${headerBorder} hover:bg-transparent`}>
            <TableHead className="h-8 px-2 text-xs text-white/70">
              Player
            </TableHead>
            <TableHead className="h-8 w-12 px-2 text-center text-xs text-white/70">
              K
            </TableHead>
            <TableHead className="h-8 w-12 px-2 text-center text-xs text-white/70">
              D
            </TableHead>
            <TableHead className="h-8 w-12 px-2 text-center text-xs text-white/70">
              A
            </TableHead>
            <TableHead className="h-8 w-14 px-2 text-center text-xs text-white/70">
              ADR
            </TableHead>
            <TableHead className="h-8 w-14 px-2 text-center text-xs text-white/70">
              HS%
            </TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {entries.map((entry) => {
            const isSelected = selectedSteamId === entry.steam_id
            return (
              <TableRow
                key={entry.steam_id}
                data-testid={`scoreboard-player-${entry.steam_id}`}
                className={`cursor-pointer border-white/10 ${
                  isSelected
                    ? "bg-white/20 hover:bg-white/25"
                    : "hover:bg-white/10"
                }`}
                onClick={() =>
                  onSelectPlayer(isSelected ? null : entry.steam_id)
                }
              >
                <TableCell className="px-2 py-1.5 text-sm text-white">
                  {entry.player_name}
                </TableCell>
                <TableCell className="px-2 py-1.5 text-center text-sm text-white">
                  {entry.kills}
                </TableCell>
                <TableCell className="px-2 py-1.5 text-center text-sm text-white">
                  {entry.deaths}
                </TableCell>
                <TableCell className="px-2 py-1.5 text-center text-sm text-white">
                  {entry.assists}
                </TableCell>
                <TableCell className="px-2 py-1.5 text-center text-sm text-white">
                  {Math.round(entry.adr)}
                </TableCell>
                <TableCell className="px-2 py-1.5 text-center text-sm text-white">
                  {Math.round(entry.hs_percent)}%
                </TableCell>
              </TableRow>
            )
          })}
        </TableBody>
      </Table>
    </div>
  )
}

export function Scoreboard({ visible }: ScoreboardProps) {
  const demoId = useViewerStore((s) => s.demoId)
  const selectedSteamId = useViewerStore((s) => s.selectedPlayerSteamId)
  const setSelectedPlayer = useViewerStore((s) => s.setSelectedPlayer)

  const { data: entries } = useScoreboard(demoId)

  if (!visible || !entries) return null

  const ctPlayers = entries.filter((e) => e.team_side === "CT")
  const tPlayers = entries.filter((e) => e.team_side === "T")

  return (
    <div
      data-testid="scoreboard-overlay"
      className="absolute inset-0 z-20 flex items-center justify-center bg-black/80 backdrop-blur-sm"
    >
      <div className="w-full max-w-2xl rounded-lg border border-white/20 bg-black/90 p-4">
        <h2 className="mb-4 text-center text-lg font-bold text-white">
          Scoreboard
        </h2>
        <div className="space-y-4">
          <TeamTable
            entries={ctPlayers}
            side="CT"
            selectedSteamId={selectedSteamId}
            onSelectPlayer={setSelectedPlayer}
          />
          <TeamTable
            entries={tPlayers}
            side="T"
            selectedSteamId={selectedSteamId}
            onSelectPlayer={setSelectedPlayer}
          />
        </div>
      </div>
    </div>
  )
}

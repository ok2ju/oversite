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

  const accentDot =
    side === "CT"
      ? "bg-sky-400 shadow-[0_0_8px_2px_rgba(56,189,248,0.5)]"
      : "bg-orange-400 shadow-[0_0_8px_2px_rgba(251,146,60,0.5)]"
  return (
    <div data-testid={`scoreboard-team-${side.toLowerCase()}`}>
      <h3
        className={`mb-2 flex items-center gap-2 text-[11px] font-semibold ${sideColor}`}
      >
        <span
          aria-hidden="true"
          className={`h-1.5 w-1.5 rotate-45 ${accentDot}`}
        />
        <span className="hud-callsign">{sideLabel}</span>
      </h3>
      <Table>
        <TableHeader>
          <TableRow className={`border-b ${headerBorder} hover:bg-transparent`}>
            <TableHead className="hud-callsign h-7 px-2 text-[10px] text-white/55">
              Player
            </TableHead>
            <TableHead className="hud-callsign h-7 w-12 px-2 text-center text-[10px] text-white/55">
              K
            </TableHead>
            <TableHead className="hud-callsign h-7 w-12 px-2 text-center text-[10px] text-white/55">
              D
            </TableHead>
            <TableHead className="hud-callsign h-7 w-12 px-2 text-center text-[10px] text-white/55">
              A
            </TableHead>
            <TableHead className="hud-callsign h-7 w-14 px-2 text-center text-[10px] text-white/55">
              ADR
            </TableHead>
            <TableHead className="hud-callsign h-7 w-14 px-2 text-center text-[10px] text-white/55">
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
                className={`cursor-pointer border-white/[0.06] transition-colors ${
                  isSelected
                    ? "bg-white/20 hover:bg-white/25"
                    : "hover:bg-white/[0.07]"
                }`}
                onClick={() =>
                  onSelectPlayer(isSelected ? null : entry.steam_id)
                }
              >
                <TableCell className="px-2 py-1.5 text-[13px] font-medium text-white">
                  {entry.player_name}
                </TableCell>
                <TableCell className="hud-display px-2 py-1.5 text-center text-[14px] text-white">
                  {entry.kills}
                </TableCell>
                <TableCell className="hud-display px-2 py-1.5 text-center text-[14px] text-white/70">
                  {entry.deaths}
                </TableCell>
                <TableCell className="hud-display px-2 py-1.5 text-center text-[14px] text-white/85">
                  {entry.assists}
                </TableCell>
                <TableCell className="px-2 py-1.5 text-center font-mono text-[12px] tabular-nums text-white/85">
                  {Math.round(entry.adr)}
                </TableCell>
                <TableCell className="px-2 py-1.5 text-center font-mono text-[12px] tabular-nums text-white/85">
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
      className="absolute inset-0 z-20 flex items-center justify-center bg-black/85 backdrop-blur-md"
    >
      <div className="hud-panel hud-scan w-full max-w-2xl overflow-hidden rounded-xl p-5">
        <div className="mb-4 flex items-center justify-center gap-3">
          <span className="h-px flex-1 bg-gradient-to-r from-transparent via-white/20 to-transparent" />
          <h2 className="hud-callsign text-[12px] font-semibold text-white/85">
            Scoreboard
          </h2>
          <span className="h-px flex-1 bg-gradient-to-r from-transparent via-white/20 to-transparent" />
        </div>
        <div className="space-y-5">
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

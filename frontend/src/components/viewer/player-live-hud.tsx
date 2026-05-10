import { usePlayerLiveHud } from "@/hooks/use-player-live-hud"
import { Progress } from "@/components/ui/progress"

interface PlayerLiveHudProps {
  steamId: string
}

const MAX_HEALTH = 100
const MAX_ARMOR = 100
const MAX_SPEED_UPS = 260

function formatSpeed(speedUps: number | null): string {
  if (speedUps === null) return "—"
  return `${Math.round(speedUps)} u/s`
}

function formatMoney(money: number): string {
  return `$${money.toLocaleString()}`
}

// PlayerLiveHud renders the current-tick HUD card for the selected player.
// Drives off the shared TickBuffer via usePlayerLiveHud (4 Hz poll, no extra
// network round-trip).
export function PlayerLiveHud({ steamId }: PlayerLiveHudProps) {
  const frame = usePlayerLiveHud(steamId)

  if (!frame) {
    return (
      <div
        data-testid="player-live-hud-empty"
        className="rounded-md border border-white/10 bg-white/5 p-3 text-sm text-white/60"
      >
        Waiting for live data…
      </div>
    )
  }

  const { data, speedUps } = frame
  const dead = !data.is_alive

  return (
    <div
      data-testid="player-live-hud"
      className="space-y-3 rounded-md border border-white/10 bg-white/5 p-3 text-white"
    >
      <div className="flex items-baseline justify-between">
        <span className="text-xs uppercase tracking-wide text-white/60">
          Live HUD
        </span>
        {dead && (
          <span className="text-xs font-semibold text-red-400">DEAD</span>
        )}
      </div>

      <div>
        <div className="flex justify-between text-xs">
          <span className="text-white/60">HP</span>
          <span
            data-testid="player-live-hud-health"
            className="tabular-nums text-white"
          >
            {data.health}
          </span>
        </div>
        <Progress
          value={(Math.max(0, data.health) / MAX_HEALTH) * 100}
          className="mt-1 h-1.5"
        />
      </div>

      <div>
        <div className="flex justify-between text-xs">
          <span className="text-white/60">Armor</span>
          <span
            data-testid="player-live-hud-armor"
            className="tabular-nums text-white"
          >
            {data.armor}
            {data.has_helmet ? " + helmet" : ""}
          </span>
        </div>
        <Progress
          value={(Math.max(0, data.armor) / MAX_ARMOR) * 100}
          className="mt-1 h-1.5"
        />
      </div>

      <div className="grid grid-cols-2 gap-2 text-xs">
        <div>
          <div className="text-white/60">Money</div>
          <div
            data-testid="player-live-hud-money"
            className="tabular-nums text-white"
          >
            {formatMoney(data.money)}
          </div>
        </div>
        <div>
          <div className="text-white/60">Weapon</div>
          <div
            data-testid="player-live-hud-weapon"
            className="truncate text-white"
          >
            {data.weapon ?? "—"}
          </div>
        </div>
        <div>
          <div className="text-white/60">Speed</div>
          <div
            data-testid="player-live-hud-speed"
            className="tabular-nums text-white"
          >
            {formatSpeed(speedUps)}
          </div>
        </div>
        <div>
          <div className="text-white/60">Defuser</div>
          <div className="text-white">{data.has_defuser ? "yes" : "no"}</div>
        </div>
      </div>

      {speedUps !== null && (
        <Progress
          value={Math.min(100, (speedUps / MAX_SPEED_UPS) * 100)}
          className="h-1"
        />
      )}
    </div>
  )
}

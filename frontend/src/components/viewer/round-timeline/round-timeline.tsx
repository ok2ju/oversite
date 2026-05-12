"use client"

import { useCallback, useEffect, useMemo, useRef, useState } from "react"
import { useViewerStore } from "@/stores/viewer"
import { useRoundTimelineModel } from "@/hooks/use-round-timeline-model"
import { TooltipProvider } from "@/components/ui/tooltip"
import { clientXToPercent } from "@/lib/viewer/timeline-utils"
import { findActiveContact } from "@/lib/timeline/contacts"
import { Lane } from "./lane"
import { BombSpine } from "./bomb-spine"
import { ContactsLane } from "./contacts-lane"
import { DuelsLane } from "./duels-lane"
import { Playhead } from "./playhead"
import { useDuelTimeline } from "@/hooks/use-duel-timeline"
import { useMistakeTimeline } from "@/hooks/use-mistake-timeline"
import type { Round } from "@/types/round"

interface RoundTimelineProps {
  round: Round
}

// Top-level orchestrator. Builds the model, measures its own width so the
// clustering pass can decide what overlaps, paints the lanes + spine + mistakes
// + playhead, and handles scrub-drag input across the full track.
export function RoundTimeline({ round }: RoundTimelineProps) {
  const trackRef = useRef<HTMLDivElement>(null)
  const draggingRef = useRef(false)
  const seekRef = useRef<(clientX: number) => void>(null)
  const activeMouseListenersRef = useRef<{
    move: (ev: MouseEvent) => void
    up: () => void
  } | null>(null)
  const activeTouchListenersRef = useRef<{
    move: (ev: TouchEvent) => void
    end: () => void
  } | null>(null)
  const [width, setWidth] = useState(800)

  useEffect(() => {
    const node = trackRef.current
    if (!node) return
    const update = () => setWidth(node.clientWidth)
    update()
    const ro = new ResizeObserver(update)
    ro.observe(node)
    return () => ro.disconnect()
  }, [])

  const { model } = useRoundTimelineModel(round, width)
  const demoId = useViewerStore((s) => s.demoId)
  const selectedPlayerSteamId = useViewerStore((s) => s.selectedPlayerSteamId)
  const currentTick = useViewerStore((s) => s.currentTick)
  const { data: duels } = useDuelTimeline(demoId, selectedPlayerSteamId)
  const { data: mistakes } = useMistakeTimeline(demoId, selectedPlayerSteamId)

  // Derive the active-contact id from the playhead. `currentTick` is the
  // single source of truth — every seek (click / scrub / round switch)
  // updates this without store coupling. useMemo short-circuits the
  // marker re-render when the playhead stays inside the same window.
  const contacts = model?.contacts
  const activeContactId = useMemo(
    () =>
      contacts ? (findActiveContact(contacts, currentTick)?.id ?? null) : null,
    [contacts, currentTick],
  )

  // Detach any in-flight document listeners on unmount so a drag that's still
  // active when the component goes away doesn't leak handlers.
  useEffect(() => {
    return () => {
      const mouse = activeMouseListenersRef.current
      if (mouse) {
        document.removeEventListener("mousemove", mouse.move)
        document.removeEventListener("mouseup", mouse.up)
        activeMouseListenersRef.current = null
      }
      const touch = activeTouchListenersRef.current
      if (touch) {
        document.removeEventListener("touchmove", touch.move)
        document.removeEventListener("touchend", touch.end)
        activeTouchListenersRef.current = null
      }
      draggingRef.current = false
    }
  }, [])

  // Keep the seek closure fresh whenever the round window changes — the
  // document listeners hold a ref to it, so swapping rounds mid-drag stays
  // accurate. The track spans the live phase only, so scrubbing maps to
  // [freeze_end_tick, end_tick].
  useEffect(() => {
    const start =
      round.freeze_end_tick > 0 ? round.freeze_end_tick : round.start_tick
    const end = round.end_tick
    const span = Math.max(1, end - start)
    seekRef.current = (clientX: number) => {
      const node = trackRef.current
      if (!node) return
      const rect = node.getBoundingClientRect()
      const pct = clientXToPercent(clientX, rect)
      const tick = start + Math.round((pct / 100) * span)
      const state = useViewerStore.getState()
      state.pause()
      state.setTick(tick)
    }
  }, [round.start_tick, round.freeze_end_tick, round.end_tick])

  const handleMouseDown = useCallback((e: React.MouseEvent) => {
    // Ignore clicks on interactive children (event icons, cluster popouts);
    // they handle their own seek.
    if ((e.target as HTMLElement).closest("button, [role='menu']")) return
    draggingRef.current = true
    seekRef.current?.(e.clientX)
    const handleMouseMove = (ev: MouseEvent) => {
      if (draggingRef.current) seekRef.current?.(ev.clientX)
    }
    const handleMouseUp = () => {
      draggingRef.current = false
      document.removeEventListener("mousemove", handleMouseMove)
      document.removeEventListener("mouseup", handleMouseUp)
      activeMouseListenersRef.current = null
    }
    document.addEventListener("mousemove", handleMouseMove)
    document.addEventListener("mouseup", handleMouseUp)
    activeMouseListenersRef.current = {
      move: handleMouseMove,
      up: handleMouseUp,
    }
  }, [])

  const handleTouchStart = useCallback((e: React.TouchEvent) => {
    if ((e.target as HTMLElement).closest("button, [role='menu']")) return
    draggingRef.current = true
    seekRef.current?.(e.touches[0].clientX)
    const handleTouchMove = (ev: TouchEvent) => {
      ev.preventDefault()
      if (draggingRef.current) seekRef.current?.(ev.touches[0].clientX)
    }
    const handleTouchEnd = () => {
      draggingRef.current = false
      document.removeEventListener("touchmove", handleTouchMove)
      document.removeEventListener("touchend", handleTouchEnd)
      activeTouchListenersRef.current = null
    }
    document.addEventListener("touchmove", handleTouchMove, { passive: false })
    document.addEventListener("touchend", handleTouchEnd)
    activeTouchListenersRef.current = {
      move: handleTouchMove,
      end: handleTouchEnd,
    }
  }, [])

  if (!model) {
    return (
      <div
        data-testid="round-timeline"
        className="flex h-[88px] items-center justify-center text-[10px] text-white/40"
      >
        Loading round timeline…
      </div>
    )
  }

  const isPlayerMode = !!model.selectedPlayerSteamId
  const side: "team" | "player" = isPlayerMode ? "player" : "team"

  return (
    <TooltipProvider delayDuration={150}>
      <div
        ref={trackRef}
        data-testid="round-timeline"
        role="slider"
        aria-valuemin={model.roundStartTick}
        aria-valuemax={model.roundEndTick}
        aria-valuenow={useViewerStore.getState().currentTick}
        aria-label="Round timeline"
        tabIndex={0}
        onMouseDown={handleMouseDown}
        onTouchStart={handleTouchStart}
        className="relative flex cursor-pointer flex-col gap-1 select-none"
      >
        <Lane
          clusters={model.topLane}
          roundStartTick={model.roundStartTick}
          roundEndTick={model.roundEndTick}
          variant="top"
          side={side}
        />
        <BombSpine
          spine={model.spine}
          roundStartTick={model.roundStartTick}
          roundEndTick={model.roundEndTick}
        />
        <Lane
          clusters={model.bottomLane}
          roundStartTick={model.roundStartTick}
          roundEndTick={model.roundEndTick}
          variant="bottom"
          side={side}
        />
        <ContactsLane
          contacts={model.contacts}
          roundStartTick={model.roundStartTick}
          roundEndTick={model.roundEndTick}
          hasPlayer={isPlayerMode}
          activeContactId={activeContactId}
        />
        <DuelsLane
          duels={duels ?? []}
          mistakes={mistakes ?? []}
          roundStartTick={model.roundStartTick}
          roundEndTick={model.roundEndTick}
          roundNumber={round.round_number}
          selectedPlayerSteamId={selectedPlayerSteamId}
          hasPlayer={isPlayerMode}
        />
        <Playhead
          roundStartTick={model.roundStartTick}
          roundEndTick={model.roundEndTick}
        />
      </div>
    </TooltipProvider>
  )
}

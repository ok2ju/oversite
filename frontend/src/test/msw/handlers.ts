import { http, HttpResponse } from "msw"
import type { Demo, GameEvent, TickData } from "@/types/demo"
import type { Round } from "@/types/round"

export const mockDemos: Demo[] = [
  {
    id: "demo-1",
    map_name: "de_dust2",
    file_size: 150_000_000,
    status: "ready",
    total_ticks: 128000,
    tick_rate: 64,
    duration_secs: 2000,
    match_date: "2026-03-01T18:00:00Z",
    created_at: "2026-03-01T19:00:00Z",
  },
  {
    id: "demo-2",
    map_name: "de_mirage",
    file_size: 120_000_000,
    status: "parsing",
    total_ticks: null,
    tick_rate: null,
    duration_secs: null,
    match_date: null,
    created_at: "2026-03-02T10:00:00Z",
  },
  {
    id: "demo-3",
    map_name: null,
    file_size: 80_000_000,
    status: "uploaded",
    total_ticks: null,
    tick_rate: null,
    duration_secs: null,
    match_date: null,
    created_at: "2026-03-03T12:00:00Z",
  },
  {
    id: "demo-4",
    map_name: "de_inferno",
    file_size: 140_000_000,
    status: "failed",
    total_ticks: null,
    tick_rate: null,
    duration_secs: null,
    match_date: null,
    created_at: "2026-03-04T08:00:00Z",
  },
]

export const handlers = [
  http.get("/api/v1/auth/me", () => {
    return HttpResponse.json({
      user_id: "test-user-id",
      faceit_id: "test-faceit-id",
      nickname: "TestPlayer",
    })
  }),

  http.get("/api/v1/demos", ({ request }) => {
    const url = new URL(request.url)
    const page = Number(url.searchParams.get("page") ?? "1")
    const perPage = Number(url.searchParams.get("per_page") ?? "20")
    const start = (page - 1) * perPage
    const sliced = mockDemos.slice(start, start + perPage)
    return HttpResponse.json({
      data: sliced,
      meta: { total: mockDemos.length, page, per_page: perPage },
    })
  }),

  http.get("/api/v1/demos/:id", ({ params }) => {
    const demo = mockDemos.find((d) => d.id === params.id)
    if (!demo) {
      return HttpResponse.json({ error: "demo not found" }, { status: 404 })
    }
    return HttpResponse.json({ data: demo })
  }),

  http.post("/api/v1/demos", () => {
    return HttpResponse.json(
      {
        data: {
          id: "demo-new",
          status: "uploaded",
          file_size: 100_000_000,
          created_at: new Date().toISOString(),
        },
      },
      { status: 202 },
    )
  }),

  http.delete("/api/v1/demos/:id", ({ params }) => {
    const demo = mockDemos.find((d) => d.id === params.id)
    if (!demo) {
      return HttpResponse.json({ error: "demo not found" }, { status: 404 })
    }
    return new HttpResponse(null, { status: 204 })
  }),

  http.get("/api/v1/demos/:id/events", ({ params }) => {
    const demo = mockDemos.find((d) => d.id === params.id)
    if (!demo) {
      return HttpResponse.json({ error: "demo not found" }, { status: 404 })
    }
    const events: GameEvent[] = [
      {
        id: "evt-kill-1",
        demo_id: String(params.id),
        round_id: null,
        tick: 1024,
        event_type: "kill",
        attacker_steam_id: "76561198000000001",
        victim_steam_id: "76561198000000002",
        weapon: "AK-47",
        x: -500,
        y: 1000,
        z: 100,
        extra_data: { attacker_x: -600, attacker_y: 800, headshot: true },
      },
      {
        id: "evt-smoke-start-1",
        demo_id: String(params.id),
        round_id: null,
        tick: 2048,
        event_type: "smoke_start",
        attacker_steam_id: "76561198000000001",
        victim_steam_id: null,
        weapon: "Smoke Grenade",
        x: 200,
        y: 300,
        z: 0,
        extra_data: { entity_id: "smoke-entity-1" },
      },
      {
        id: "evt-smoke-expired-1",
        demo_id: String(params.id),
        round_id: null,
        tick: 3200,
        event_type: "smoke_expired",
        attacker_steam_id: null,
        victim_steam_id: null,
        weapon: null,
        x: 200,
        y: 300,
        z: 0,
        extra_data: { entity_id: "smoke-entity-1" },
      },
      {
        id: "evt-he-1",
        demo_id: String(params.id),
        round_id: null,
        tick: 4096,
        event_type: "grenade_detonate",
        attacker_steam_id: "76561198000000001",
        victim_steam_id: null,
        weapon: "HE Grenade",
        x: 0,
        y: 500,
        z: 0,
        extra_data: null,
      },
      {
        id: "evt-flash-1",
        demo_id: String(params.id),
        round_id: null,
        tick: 5000,
        event_type: "grenade_detonate",
        attacker_steam_id: "76561198000000001",
        victim_steam_id: null,
        weapon: "Flashbang",
        x: -300,
        y: 200,
        z: 0,
        extra_data: null,
      },
      {
        id: "evt-bomb-plant-1",
        demo_id: String(params.id),
        round_id: null,
        tick: 6000,
        event_type: "bomb_plant",
        attacker_steam_id: "76561198000000003",
        victim_steam_id: null,
        weapon: null,
        x: 100,
        y: -200,
        z: 0,
        extra_data: null,
      },
      {
        id: "evt-bomb-defuse-1",
        demo_id: String(params.id),
        round_id: null,
        tick: 6500,
        event_type: "bomb_defuse",
        attacker_steam_id: "76561198000000004",
        victim_steam_id: null,
        weapon: null,
        x: 100,
        y: -200,
        z: 0,
        extra_data: { has_kit: true },
      },
    ]
    return HttpResponse.json({ data: events })
  }),

  http.get("/api/v1/demos/:id/rounds", ({ params }) => {
    const demo = mockDemos.find((d) => d.id === params.id)
    if (!demo) {
      return HttpResponse.json({ error: "demo not found" }, { status: 404 })
    }
    const rounds: Round[] = [
      {
        id: "round-1",
        round_number: 1,
        start_tick: 0,
        end_tick: 3200,
        winner_side: "CT",
        ct_score: 1,
        t_score: 0,
      },
      {
        id: "round-2",
        round_number: 2,
        start_tick: 3200,
        end_tick: 6400,
        winner_side: "T",
        ct_score: 1,
        t_score: 1,
      },
      {
        id: "round-3",
        round_number: 3,
        start_tick: 6400,
        end_tick: 9600,
        winner_side: "CT",
        ct_score: 2,
        t_score: 1,
      },
    ]
    return HttpResponse.json({ data: rounds })
  }),

  http.get("/api/v1/demos/:id/ticks", ({ request, params }) => {
    const url = new URL(request.url)
    const startTick = Number(url.searchParams.get("start_tick") ?? "0")
    const endTick = Number(url.searchParams.get("end_tick") ?? "0")
    const demo = mockDemos.find((d) => d.id === params.id)
    if (!demo) {
      return HttpResponse.json({ error: "demo not found" }, { status: 404 })
    }
    const data: TickData[] = []
    const allSteamIds = ["76561198000000001", "76561198000000002"]
    const steamIdsParam = url.searchParams.get("steam_ids")
    const steamIds = steamIdsParam
      ? steamIdsParam.split(",").map((s) => s.trim())
      : allSteamIds
    for (let t = startTick; t <= Math.min(endTick, startTick + 9); t++) {
      for (const sid of steamIds) {
        data.push({
          tick: t,
          steam_id: sid,
          x: t * 1.0,
          y: t * 2.0,
          z: 0,
          yaw: 90,
          health: 100,
          armor: 100,
          is_alive: true,
          weapon: "ak47",
        })
      }
    }
    return HttpResponse.json({ data })
  }),
]

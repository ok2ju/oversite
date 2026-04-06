import { http, HttpResponse } from "msw"
import type { Demo, TickData } from "@/types/demo"

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

  http.get("/api/v1/demos/:id/ticks", ({ request, params }) => {
    const url = new URL(request.url)
    const startTick = Number(url.searchParams.get("start_tick") ?? "0")
    const endTick = Number(url.searchParams.get("end_tick") ?? "0")
    const demo = mockDemos.find((d) => d.id === params.id)
    if (!demo) {
      return HttpResponse.json({ error: "demo not found" }, { status: 404 })
    }
    const data: TickData[] = []
    const steamIds = ["76561198000000001", "76561198000000002"]
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

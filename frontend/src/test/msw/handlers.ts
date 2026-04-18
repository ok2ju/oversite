import { http, HttpResponse } from "msw"
import {
  mockDemos,
  createMockEvents,
  mockRounds,
  createMockTickData,
  mockFaceitMatches,
  mockFaceitProfile,
  mockUser,
} from "@/test/fixtures"

// Re-export fixtures for backward compatibility with existing tests
export { mockDemos } from "@/test/fixtures"
export { mockFaceitMatches } from "@/test/fixtures"

export const handlers = [
  http.get("/api/v1/auth/me", () => {
    return HttpResponse.json(mockUser)
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
    const demo = mockDemos.find((d) => d.id === Number(params.id))
    if (!demo) {
      return HttpResponse.json({ error: "demo not found" }, { status: 404 })
    }
    return HttpResponse.json({ data: demo })
  }),

  http.post("/api/v1/demos", () => {
    return HttpResponse.json(
      {
        data: {
          id: 99,
          status: "imported",
          file_size: 100_000_000,
          created_at: new Date().toISOString(),
        },
      },
      { status: 202 },
    )
  }),

  http.delete("/api/v1/demos/:id", ({ params }) => {
    const demo = mockDemos.find((d) => d.id === Number(params.id))
    if (!demo) {
      return HttpResponse.json({ error: "demo not found" }, { status: 404 })
    }
    return new HttpResponse(null, { status: 204 })
  }),

  http.get("/api/v1/demos/:id/events", ({ params }) => {
    const demo = mockDemos.find((d) => d.id === Number(params.id))
    if (!demo) {
      return HttpResponse.json({ error: "demo not found" }, { status: 404 })
    }
    return HttpResponse.json({ data: createMockEvents(String(params.id)) })
  }),

  http.get("/api/v1/demos/:id/rounds", ({ params }) => {
    const demo = mockDemos.find((d) => d.id === Number(params.id))
    if (!demo) {
      return HttpResponse.json({ error: "demo not found" }, { status: 404 })
    }
    return HttpResponse.json({ data: mockRounds })
  }),

  http.get("/api/v1/demos/:id/ticks", ({ request, params }) => {
    const url = new URL(request.url)
    const startTick = Number(url.searchParams.get("start_tick") ?? "0")
    const endTick = Number(url.searchParams.get("end_tick") ?? "0")
    const demo = mockDemos.find((d) => d.id === Number(params.id))
    if (!demo) {
      return HttpResponse.json({ error: "demo not found" }, { status: 404 })
    }
    const allSteamIds = ["76561198000000001", "76561198000000002"]
    const steamIdsParam = url.searchParams.get("steam_ids")
    const steamIds = steamIdsParam
      ? steamIdsParam.split(",").map((s) => s.trim())
      : allSteamIds
    return HttpResponse.json({
      data: createMockTickData(startTick, endTick, steamIds),
    })
  }),

  http.get("/api/v1/faceit/profile", () => {
    return HttpResponse.json({ data: mockFaceitProfile })
  }),

  http.get("/api/v1/faceit/matches", ({ request }) => {
    const url = new URL(request.url)
    const page = Number(url.searchParams.get("page") ?? "1")
    const perPage = Number(url.searchParams.get("per_page") ?? "20")
    const mapFilter = url.searchParams.get("map_name")
    const resultFilter = url.searchParams.get("result")

    let filtered = [...mockFaceitMatches]
    if (mapFilter) {
      filtered = filtered.filter((m) => m.map_name === mapFilter)
    }
    if (resultFilter) {
      filtered = filtered.filter((m) => m.result === resultFilter)
    }

    const start = (page - 1) * perPage
    const sliced = filtered.slice(start, start + perPage)
    return HttpResponse.json({
      data: sliced,
      meta: { total: filtered.length, page, per_page: perPage },
    })
  }),
]

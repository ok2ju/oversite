import { vi } from "vitest"
import {
  mockDemos,
  createMockEvents,
  mockRounds,
  mockScoreboardEntries,
} from "@/test/fixtures"
import type { Demo, DemoSummary } from "@/types/demo"

function toSummary(d: Demo): DemoSummary {
  const { file_path: _path, ...rest } = d
  return {
    ...rest,
    file_name: _path.split("/").pop() ?? "",
  }
}

// ---------------------------------------------------------------------------
// Wails App binding mocks (wailsjs/go/main/App)
// ---------------------------------------------------------------------------

/**
 * Default mock implementations for all App binding methods.
 * Extend this object as new bindings are added to app.go.
 */
export const mockAppBindings = {
  Greet: vi
    .fn<(name: string) => Promise<string>>()
    .mockResolvedValue("Hello TestPlayer, welcome to Oversite!"),

  ListDemos: vi
    .fn<
      (
        page: number,
        perPage: number,
      ) => Promise<{
        data: DemoSummary[]
        meta: { total: number; page: number; per_page: number }
      }>
    >()
    .mockImplementation((page = 1, perPage = 20) => {
      const start = (page - 1) * perPage
      const sliced = mockDemos.slice(start, start + perPage).map(toSummary)
      return Promise.resolve({
        data: sliced,
        meta: { total: mockDemos.length, page, per_page: perPage },
      })
    }),

  ImportDemoByPath: vi
    .fn<(path: string) => Promise<void>>()
    .mockResolvedValue(undefined),

  ImportDemoFile: vi.fn<() => Promise<void>>().mockResolvedValue(undefined),

  DeleteDemo: vi
    .fn<(id: number) => Promise<void>>()
    .mockResolvedValue(undefined),

  GetDemoByID: vi
    .fn<(id: string) => Promise<(typeof mockDemos)[0]>>()
    .mockImplementation((id: string) => {
      const demo = mockDemos.find((d) => String(d.id) === id)
      if (!demo) return Promise.reject(new Error("demo not found"))
      return Promise.resolve(demo)
    }),

  GetDemoRounds: vi
    .fn<(demoId: string) => Promise<typeof mockRounds>>()
    .mockResolvedValue(mockRounds),

  GetDemoEvents: vi
    .fn<(demoId: string) => Promise<ReturnType<typeof createMockEvents>>>()
    .mockImplementation((demoId: string) =>
      Promise.resolve(createMockEvents(demoId)),
    ),

  GetEventsByTypes: vi
    .fn<
      (
        demoId: string,
        eventTypes: string[],
      ) => Promise<ReturnType<typeof createMockEvents>>
    >()
    .mockImplementation((demoId: string, eventTypes: string[]) => {
      const all = createMockEvents(demoId)
      if (!eventTypes?.length) return Promise.resolve([])
      const set = new Set(eventTypes)
      return Promise.resolve(all.filter((e) => set.has(e.event_type)))
    }),

  GetDemoTicks: vi
    .fn<
      (demoId: string, startTick: number, endTick: number) => Promise<never[]>
    >()
    .mockResolvedValue([]),

  GetRoundRoster: vi
    .fn<(demoId: string, roundNumber: number) => Promise<never[]>>()
    .mockResolvedValue([]),

  GetScoreboard: vi
    .fn<(demoId: string) => Promise<typeof mockScoreboardEntries>>()
    .mockResolvedValue(mockScoreboardEntries),

  GetPlayerMatchStats: vi
    .fn<
      (
        demoId: string,
        steamId: string,
      ) => Promise<{
        steam_id: string
        player_name: string
        team_side: string
        rounds_played: number
        kills: number
        deaths: number
        assists: number
        damage: number
        hs_kills: number
        clutch_kills: number
        first_kills: number
        first_deaths: number
        opening_wins: number
        opening_losses: number
        trade_kills: number
        hs_percent: number
        adr: number
        damage_by_weapon: Array<{ weapon: string; damage: number }>
        damage_by_opponent: Array<{
          steam_id: string
          player_name: string
          team_side: string
          damage: number
        }>
        rounds: Array<{
          round_number: number
          team_side: string
          kills: number
          deaths: number
          assists: number
          damage: number
          hs_kills: number
          clutch_kills: number
          first_kill: boolean
          first_death: boolean
          trade_kill: boolean
          loadout_value: number
          distance_units: number
          alive_duration_secs: number
          time_to_first_contact_sec: number | null
        }>
        movement: {
          distance_units: number
          avg_speed_ups: number
          max_speed_ups: number
          strafe_percent: number
          stationary_ratio: number
          walking_ratio: number
          running_ratio: number
        }
        timings: {
          avg_time_to_first_contact_secs: number
          avg_alive_duration_secs: number
          time_on_site_a_secs: number
          time_on_site_b_secs: number
        }
        utility: {
          flashes_thrown: number
          smokes_thrown: number
          hes_thrown: number
          molotovs_thrown: number
          decoys_thrown: number
          flash_assists: number
          blind_time_inflicted_secs: number
          enemies_flashed: number
        }
        hit_groups: Array<{
          hit_group: number
          label: string
          damage: number
          hits: number
        }>
      }>
    >()
    .mockImplementation((_demoId, steamId) =>
      Promise.resolve({
        steam_id: steamId,
        player_name: "MockPlayer",
        team_side: "CT",
        rounds_played: 2,
        kills: 3,
        deaths: 1,
        assists: 1,
        damage: 250,
        hs_kills: 1,
        clutch_kills: 0,
        first_kills: 1,
        first_deaths: 0,
        opening_wins: 1,
        opening_losses: 0,
        trade_kills: 0,
        hs_percent: 33.33,
        adr: 125,
        damage_by_weapon: [
          { weapon: "ak-47", damage: 175 },
          { weapon: "deagle", damage: 75 },
        ],
        damage_by_opponent: [
          {
            steam_id: "STEAM_X",
            player_name: "Enemy1",
            team_side: "T",
            damage: 175,
          },
          {
            steam_id: "STEAM_Y",
            player_name: "Enemy2",
            team_side: "T",
            damage: 75,
          },
        ],
        rounds: [
          {
            round_number: 1,
            team_side: "CT",
            kills: 2,
            deaths: 0,
            assists: 0,
            damage: 175,
            hs_kills: 1,
            clutch_kills: 0,
            first_kill: true,
            first_death: false,
            trade_kill: false,
            loadout_value: 4750,
            distance_units: 4200,
            alive_duration_secs: 80,
            time_to_first_contact_sec: 12.5,
          },
          {
            round_number: 2,
            team_side: "CT",
            kills: 1,
            deaths: 1,
            assists: 1,
            damage: 75,
            hs_kills: 0,
            clutch_kills: 0,
            first_kill: false,
            first_death: false,
            trade_kill: false,
            loadout_value: 3700,
            distance_units: 1800,
            alive_duration_secs: 22,
            time_to_first_contact_sec: 4.0,
          },
        ],
        movement: {
          distance_units: 6000,
          avg_speed_ups: 110,
          max_speed_ups: 248,
          strafe_percent: 35,
          stationary_ratio: 0.2,
          walking_ratio: 0.5,
          running_ratio: 0.3,
        },
        timings: {
          avg_time_to_first_contact_secs: 8.25,
          avg_alive_duration_secs: 51,
          time_on_site_a_secs: 18,
          time_on_site_b_secs: 4,
        },
        utility: {
          flashes_thrown: 6,
          smokes_thrown: 4,
          hes_thrown: 2,
          molotovs_thrown: 1,
          decoys_thrown: 0,
          flash_assists: 1,
          blind_time_inflicted_secs: 14.5,
          enemies_flashed: 7,
        },
        hit_groups: [
          { hit_group: 2, label: "Chest", damage: 130, hits: 4 },
          { hit_group: 1, label: "Head", damage: 95, hits: 1 },
          { hit_group: 6, label: "Left Leg", damage: 25, hits: 2 },
        ],
      }),
    ),

  GetHeatmapData: vi
    .fn<
      (
        demoIDs: number[],
        weapons: string[],
        playerSteamID: string,
        side: string,
      ) => Promise<Array<{ x: number; y: number; kill_count: number }>>
    >()
    .mockResolvedValue([
      { x: 100.5, y: 200.5, kill_count: 3 },
      { x: 300.0, y: 400.0, kill_count: 1 },
    ]),

  GetMistakeTimeline: vi
    .fn<
      (
        demoId: string,
        steamId: string,
      ) => Promise<
        Array<{
          kind: string
          round_number: number
          tick: number
          steam_id: string
          extras: Record<string, unknown> | null
        }>
      >
    >()
    .mockResolvedValue([]),

  GetPlayerAnalysis: vi
    .fn<
      (
        demoId: string,
        steamId: string,
      ) => Promise<{
        steam_id: string
        overall_score: number
        trade_pct: number
        avg_trade_ticks: number
        extras: Record<string, unknown> | null
      }>
    >()
    .mockResolvedValue({
      steam_id: "",
      overall_score: 0,
      trade_pct: 0,
      avg_trade_ticks: 0,
      extras: null,
    }),

  GetAnalysisStatus: vi
    .fn<(demoId: string) => Promise<{ demo_id: string; status: string }>>()
    .mockResolvedValue({ demo_id: "", status: "ready" }),

  GetHabitReport: vi
    .fn<
      (
        demoId: string,
        steamId: string,
      ) => Promise<{
        demo_id: string
        steam_id: string
        as_of: string
        habits: Array<{
          key: string
          label: string
          description: string
          unit: string
          direction: string
          value: number
          status: string
          good_threshold: number
          warn_threshold: number
          good_min: number
          good_max: number
          warn_min: number
          warn_max: number
          previous_value: number | null
          delta: number | null
        }>
      }>
    >()
    .mockResolvedValue({
      demo_id: "",
      steam_id: "",
      as_of: "",
      habits: [],
    }),

  GetHabitHistory: vi
    .fn<
      (
        steamId: string,
        habitKey: string,
        limit: number,
      ) => Promise<
        Array<{ demo_id: string; match_date: string; value: number }>
      >
    >()
    .mockResolvedValue([]),

  GetPlayerRoundAnalysis: vi
    .fn<
      (
        demoId: string,
        steamId: string,
      ) => Promise<
        Array<{
          steam_id: string
          round_number: number
          trade_pct: number
          extras: Record<string, unknown> | null
        }>
      >
    >()
    .mockResolvedValue([]),

  RecomputeAnalysis: vi
    .fn<(demoId: string) => Promise<void>>()
    .mockResolvedValue(undefined),

  GetMinEngagementsForAimCritique: vi
    .fn<() => Promise<number>>()
    .mockResolvedValue(8),
  SetMinEngagementsForAimCritique: vi
    .fn<(n: number) => Promise<void>>()
    .mockResolvedValue(undefined),

  GetUniqueWeapons: vi
    .fn<(demoIDs: number[]) => Promise<string[]>>()
    .mockResolvedValue(["AK-47", "M4A1", "AWP"]),

  GetUniquePlayers: vi
    .fn<
      (
        demoIDs: number[],
      ) => Promise<Array<{ steam_id: string; player_name: string }>>
    >()
    .mockResolvedValue([
      { steam_id: "STEAM_A", player_name: "Player1" },
      { steam_id: "STEAM_B", player_name: "Player2" },
    ]),

  GetWeaponStats: vi
    .fn<
      (
        demoID: string,
      ) => Promise<
        Array<{ weapon: string; kill_count: number; hs_count: number }>
      >
    >()
    .mockResolvedValue([
      { weapon: "AK-47", kill_count: 10, hs_count: 5 },
      { weapon: "M4A1", kill_count: 7, hs_count: 3 },
      { weapon: "AWP", kill_count: 4, hs_count: 0 },
    ]),

  LogsDir: vi
    .fn<() => Promise<string>>()
    .mockResolvedValue("/tmp/oversite/logs"),

  OpenLogsFolder: vi.fn<() => Promise<void>>().mockResolvedValue(undefined),
}

/**
 * Call in a test file or setup to activate the App binding mock:
 *
 *   vi.mock("@wailsjs/go/main/App", () => mockAppBindings)
 *
 * Or for per-test control:
 *
 *   import { mockAppBindings } from "@/test/mocks/bindings"
 *   mockAppBindings.GetDemoByID.mockResolvedValueOnce(customDemo)
 */
export function resetAppBindings() {
  Object.values(mockAppBindings).forEach((fn) => fn.mockClear())
}

// ---------------------------------------------------------------------------
// Wails runtime mocks (wailsjs/runtime)
// ---------------------------------------------------------------------------

export const mockRuntime = {
  // Events
  EventsEmit: vi.fn<(eventName: string, ...data: unknown[]) => void>(),
  EventsOn: vi
    .fn<
      (eventName: string, callback: (...data: unknown[]) => void) => () => void
    >()
    .mockReturnValue(vi.fn()),
  EventsOnMultiple: vi
    .fn<
      (
        eventName: string,
        callback: (...data: unknown[]) => void,
        maxCallbacks: number,
      ) => () => void
    >()
    .mockReturnValue(vi.fn()),
  EventsOnce: vi
    .fn<
      (eventName: string, callback: (...data: unknown[]) => void) => () => void
    >()
    .mockReturnValue(vi.fn()),
  EventsOff:
    vi.fn<(eventName: string, ...additionalEventNames: string[]) => void>(),
  EventsOffAll: vi.fn(),

  // Logging
  LogPrint: vi.fn(),
  LogTrace: vi.fn(),
  LogDebug: vi.fn(),
  LogInfo: vi.fn(),
  LogWarning: vi.fn(),
  LogError: vi.fn(),
  LogFatal: vi.fn(),

  // Window
  WindowReload: vi.fn(),
  WindowReloadApp: vi.fn(),
  WindowSetAlwaysOnTop: vi.fn(),
  WindowCenter: vi.fn(),
  WindowSetTitle: vi.fn(),
  WindowFullscreen: vi.fn(),
  WindowUnfullscreen: vi.fn(),
  WindowIsFullscreen: vi.fn<() => Promise<boolean>>().mockResolvedValue(false),
  WindowSetSize: vi.fn(),
  WindowGetSize: vi
    .fn<() => Promise<{ w: number; h: number }>>()
    .mockResolvedValue({ w: 1280, h: 800 }),
  WindowSetMaxSize: vi.fn(),
  WindowSetMinSize: vi.fn(),
  WindowSetPosition: vi.fn(),
  WindowGetPosition: vi
    .fn<() => Promise<{ x: number; y: number }>>()
    .mockResolvedValue({ x: 0, y: 0 }),
  WindowHide: vi.fn(),
  WindowShow: vi.fn(),
  WindowMaximise: vi.fn(),
  WindowToggleMaximise: vi.fn(),
  WindowUnmaximise: vi.fn(),
  WindowIsMaximised: vi.fn<() => Promise<boolean>>().mockResolvedValue(false),
  WindowMinimise: vi.fn(),
  WindowUnminimise: vi.fn(),
  WindowIsMinimised: vi.fn<() => Promise<boolean>>().mockResolvedValue(false),
  WindowIsNormal: vi.fn<() => Promise<boolean>>().mockResolvedValue(true),
  WindowSetBackgroundColour: vi.fn(),
  WindowSetSystemDefaultTheme: vi.fn(),
  WindowSetLightTheme: vi.fn(),
  WindowSetDarkTheme: vi.fn(),

  // Screen
  ScreenGetAll: vi
    .fn<
      () => Promise<
        Array<{
          isCurrent: boolean
          isPrimary: boolean
          width: number
          height: number
        }>
      >
    >()
    .mockResolvedValue([
      { isCurrent: true, isPrimary: true, width: 1920, height: 1080 },
    ]),

  // Browser
  BrowserOpenURL: vi.fn(),

  // Environment
  Environment: vi
    .fn<() => Promise<{ buildType: string; platform: string; arch: string }>>()
    .mockResolvedValue({
      buildType: "dev",
      platform: "darwin",
      arch: "arm64",
    }),

  // App lifecycle
  Quit: vi.fn(),
  Hide: vi.fn(),
  Show: vi.fn(),

  // Clipboard
  ClipboardGetText: vi.fn<() => Promise<string>>().mockResolvedValue(""),
  ClipboardSetText: vi
    .fn<(text: string) => Promise<boolean>>()
    .mockResolvedValue(true),

  // Drag and drop
  OnFileDrop: vi.fn(),
  OnFileDropOff: vi.fn(),
}

export function resetRuntimeMocks() {
  Object.values(mockRuntime).forEach((fn) => fn.mockClear())
}

/**
 * Reset all Wails mocks (bindings + runtime). Call in beforeEach/afterEach.
 */
export function resetAllWailsMocks() {
  resetAppBindings()
  resetRuntimeMocks()
}

import { vi } from "vitest"
import {
  mockDemos,
  createMockEvents,
  mockRounds,
  mockScoreboardEntries,
  mockFaceitMatches,
  mockFaceitProfile,
  mockUser,
} from "@/test/fixtures"

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

  GetCurrentUser: vi
    .fn<() => Promise<typeof mockUser>>()
    .mockResolvedValue(mockUser),

  LoginWithFaceit: vi.fn<() => Promise<void>>().mockResolvedValue(undefined),

  Logout: vi.fn<() => Promise<void>>().mockResolvedValue(undefined),

  ListDemos: vi
    .fn<
      (
        page: number,
        perPage: number,
      ) => Promise<{
        data: typeof mockDemos
        meta: { total: number; page: number; per_page: number }
      }>
    >()
    .mockImplementation((page = 1, perPage = 20) => {
      const start = (page - 1) * perPage
      const sliced = mockDemos.slice(start, start + perPage)
      return Promise.resolve({
        data: sliced,
        meta: { total: mockDemos.length, page, per_page: perPage },
      })
    }),

  ImportDemoByPath: vi
    .fn<(path: string) => Promise<void>>()
    .mockResolvedValue(undefined),

  ImportDemoFile: vi.fn<() => Promise<void>>().mockResolvedValue(undefined),

  ImportDemoFolder: vi
    .fn<() => Promise<{ imported: typeof mockDemos; errors: string[] }>>()
    .mockResolvedValue({ imported: [], errors: [] }),

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

  GetFaceitProfile: vi
    .fn<() => Promise<typeof mockFaceitProfile>>()
    .mockResolvedValue(mockFaceitProfile),

  SyncFaceitMatches: vi.fn<() => Promise<number>>().mockResolvedValue(5),

  ImportMatchDemo: vi
    .fn<(faceitMatchID: string) => Promise<void>>()
    .mockResolvedValue(undefined),

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

  GetFaceitMatches: vi
    .fn<
      (
        page: number,
        perPage: number,
        mapName: string,
        result: string,
      ) => Promise<{
        data: typeof mockFaceitMatches
        meta: { total: number; page: number; per_page: number }
      }>
    >()
    .mockImplementation((page = 1, perPage = 20, mapName = "", result = "") => {
      let filtered = [...mockFaceitMatches]
      if (mapName) {
        filtered = filtered.filter((m) => m.map_name === mapName)
      }
      if (result) {
        filtered = filtered.filter((m) => m.result === result)
      }
      const start = (page - 1) * perPage
      const sliced = filtered.slice(start, start + perPage)
      return Promise.resolve({
        data: sliced,
        meta: { total: filtered.length, page, per_page: perPage },
      })
    }),
}

/**
 * Call in a test file or setup to activate the App binding mock:
 *
 *   vi.mock("@wailsjs/go/main/App", () => mockAppBindings)
 *
 * Or for per-test control:
 *
 *   import { mockAppBindings } from "@/test/mocks/bindings"
 *   mockAppBindings.GetCurrentUser.mockResolvedValueOnce(customUser)
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

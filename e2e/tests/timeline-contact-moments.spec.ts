import { test, expect, type Page } from "@playwright/test"

/**
 * Phase 5 — contact-moment timeline e2e
 * (.claude/plans/timeline-contact-moments/phase-5/05-e2e-test.md).
 *
 * Covers parent README §5 §4 end-to-end:
 *  - round mode lane: grenade + bomb events only (no kills/hurts).
 *  - player mode lane: contacts; hover tooltip; click seek.
 *  - active-marker highlight tracks playhead through [tPre, tPost].
 *
 * The spec assumes the reference demo testdata/demos/1.dem has been
 * imported. The `beforeEach` hook imports it via the Wails binding if
 * a ready demo is not already present; if the binding isn't reachable
 * (no `wails dev`), the suite skips.
 *
 * State-based assertions only — no screenshot snapshots (see plan §3).
 */

const DEMO_FIXTURE_PATH = "testdata/demos/1.dem"

const ALLOWED_ROUND_MODE_KINDS = new Set([
  "grenade",
  "bomb_plant",
  "bomb_defuse",
  "bomb_explode",
])

test.describe("timeline contact moments", () => {
  test.beforeEach(async ({ page }) => {
    await page.goto("/")
    const ok = await ensureBindings(page)
    if (!ok) test.skip(true, "Wails bindings unavailable — run under `wails dev`")
    const demoId = await ensureDemoImported(page)
    if (!demoId) test.skip(true, "Reference demo not importable in this environment")
    await openDemoViewer(page, demoId!)
  })

  test("round mode lane shows only grenade + bomb event markers", async ({
    page,
  }) => {
    const timeline = page.getByTestId("round-timeline").first()
    await expect(timeline).toBeVisible()

    // The contacts lane should render the no-player placeholder.
    await expect(
      page.getByTestId("round-timeline-contacts-placeholder").first(),
    ).toBeVisible()

    // Every event marker on the lane should carry an allowed `kind`
    // attribute (round-mode lanes only carry grenade + bomb events).
    const markers = page.locator("[data-testid^='event-marker-']")
    const count = await markers.count()
    if (count === 0) {
      test.skip(true, "Reference demo has no round-mode events on first round")
    }
    for (let i = 0; i < count; i++) {
      const kind = await markers.nth(i).getAttribute("data-kind")
      expect(kind, `marker ${i} kind`).not.toBeNull()
      expect(ALLOWED_ROUND_MODE_KINDS, `marker ${i} kind ${kind}`).toContain(
        kind,
      )
    }
  })

  test("selecting a player swaps the contacts lane to markers", async ({
    page,
  }) => {
    const steam = await firstPlayerSteamWithContacts(page)
    if (!steam) test.skip(true, "Reference demo has no contacts in any round")

    await setSelectedPlayer(page, steam!)

    await expect(
      page.getByTestId("round-timeline-contacts-placeholder"),
    ).toHaveCount(0)

    await expect(page.locator("[data-testid^='contact-marker-']").first())
      .toBeVisible({ timeout: 5_000 })
  })

  test("hovering a contact marker reveals the tooltip with outcome", async ({
    page,
  }) => {
    const steam = await firstPlayerSteamWithContacts(page)
    if (!steam) test.skip(true, "Reference demo has no contacts in any round")
    await setSelectedPlayer(page, steam!)

    const marker = page.locator("[data-testid^='contact-marker-']").first()
    await marker.hover()

    const tooltip = page.getByTestId("contact-tooltip").first()
    await expect(tooltip).toBeVisible({ timeout: 3_000 })
    await expect(tooltip).toContainText(/won|traded|untraded|disengaged|damage|partial/i)
  })

  test("clicking a marker pauses playback and seeks to tPre", async ({
    page,
  }) => {
    const steam = await firstPlayerSteamWithContacts(page)
    if (!steam) test.skip(true, "Reference demo has no contacts in any round")
    await setSelectedPlayer(page, steam!)

    const target = await firstCachedContact(page)
    expect(target, "expected a cached contact moment").not.toBeNull()
    expect(target!.tPre).toBeGreaterThan(0)

    const marker = page.getByTestId(`contact-marker-${target!.id}`)
    await marker.click()

    const state = await readViewerState(page)
    expect(state).not.toBeNull()
    expect(state!.isPlaying).toBe(false)
    expect(state!.currentTick).toBe(target!.tPre)

    await expect(marker).toHaveAttribute("data-active", "true")
  })

  test("scrubbing past tPost drops the active marker highlight", async ({
    page,
  }) => {
    const steam = await firstPlayerSteamWithContacts(page)
    if (!steam) test.skip(true, "Reference demo has no contacts in any round")
    await setSelectedPlayer(page, steam!)

    const target = await firstCachedContact(page)
    expect(target).not.toBeNull()

    const marker = page.getByTestId(`contact-marker-${target!.id}`)
    await marker.click()
    await expect(marker).toHaveAttribute("data-active", "true")

    await page.evaluate((tick: number) => {
      const w = window as unknown as {
        __useViewerStore?: { getState: () => { setTick: (t: number) => void } }
      }
      w.__useViewerStore?.getState().setTick(tick)
    }, target!.tPost + 100)

    await expect(marker).not.toHaveAttribute("data-active", "true")
  })
})

// --- helpers --------------------------------------------------------------

interface CachedContact {
  id: number
  tPre: number
  tPost: number
}

async function ensureBindings(page: Page): Promise<boolean> {
  return await page.evaluate(() => {
    const w = window as unknown as {
      go?: { main?: { App?: { ListDemos?: unknown } } }
    }
    return typeof w.go?.main?.App?.ListDemos === "function"
  })
}

async function ensureDemoImported(page: Page): Promise<string | null> {
  return await page.evaluate(async (path: string) => {
    const w = window as unknown as {
      go: {
        main: {
          App: {
            ListDemos: (
              page: number,
              perPage: number,
            ) => Promise<{ demos?: Array<{ id: number; status: string }> }>
            ImportDemoByPath?: (p: string) => Promise<void>
          }
        }
      }
    }
    const App = w.go.main.App
    const existing = await App.ListDemos(1, 50)
    const ready = existing.demos?.find((d) => d.status === "ready")
    if (ready) return String(ready.id)
    if (!App.ImportDemoByPath) return null
    try {
      await App.ImportDemoByPath(path)
    } catch (e) {
      console.error("ImportDemoByPath failed:", e)
      return null
    }
    // Poll for the import to reach ready (parse runs synchronously but
    // the binding returns after enqueue).
    for (let i = 0; i < 30; i++) {
      await new Promise((r) => setTimeout(r, 1_000))
      const list = await App.ListDemos(1, 50)
      const r = list.demos?.find((d) => d.status === "ready")
      if (r) return String(r.id)
    }
    return null
  }, DEMO_FIXTURE_PATH)
}

async function openDemoViewer(page: Page, demoId: string): Promise<void> {
  await page.goto(`/demos/${demoId}`)
  await expect(page.getByTestId("round-timeline").first()).toBeVisible({
    timeout: 15_000,
  })
}

async function setSelectedPlayer(page: Page, steamId: string): Promise<void> {
  await page.evaluate((id: string) => {
    const w = window as unknown as {
      __useViewerStore?: {
        getState: () => { setSelectedPlayer: (s: string) => void }
      }
    }
    w.__useViewerStore?.getState().setSelectedPlayer(id)
  }, steamId)
}

async function firstPlayerSteamWithContacts(
  page: Page,
): Promise<string | null> {
  return await page.evaluate(async () => {
    const w = window as unknown as {
      go: {
        main: {
          App: {
            GetDemoByID: (id: string) => Promise<{ id: number }>
            GetDemoRounds: (id: string) => Promise<Array<{ round_number: number }>>
            GetRoundRoster: (
              id: string,
              roundNum: number,
            ) => Promise<Array<{ steam_id: string }>>
            GetContactMoments: (
              id: string,
              roundNum: number,
              subjectSteam: string,
            ) => Promise<unknown[]>
          }
        }
      }
      __useViewerStore?: {
        getState: () => { demoId: string | null }
      }
    }
    const demoId = w.__useViewerStore?.getState().demoId
    if (!demoId) return null
    const rounds = await w.go.main.App.GetDemoRounds(demoId)
    for (const r of rounds.slice(0, 8)) {
      const roster = await w.go.main.App.GetRoundRoster(demoId, r.round_number)
      for (const p of roster) {
        if (!p.steam_id || p.steam_id === "0") continue
        const contacts = await w.go.main.App.GetContactMoments(
          demoId,
          r.round_number,
          p.steam_id,
        )
        if (contacts.length > 0) {
          return p.steam_id
        }
      }
    }
    return null
  })
}

async function firstCachedContact(page: Page): Promise<CachedContact | null> {
  return await page.evaluate(() => {
    const w = window as unknown as {
      __queryClient?: {
        getQueriesData: (filters: { queryKey: unknown[] }) => Array<
          [
            unknown,
            Array<{ id: number; t_pre: number; t_post: number }> | undefined,
          ]
        >
      }
    }
    const qc = w.__queryClient
    if (!qc) return null
    const matches = qc.getQueriesData({ queryKey: ["contact-moments"] })
    for (const [, data] of matches) {
      if (Array.isArray(data) && data.length > 0) {
        const c = data[0]
        return { id: c.id, tPre: c.t_pre, tPost: c.t_post }
      }
    }
    return null
  })
}

async function readViewerState(
  page: Page,
): Promise<{ isPlaying: boolean; currentTick: number } | null> {
  return await page.evaluate(() => {
    const w = window as unknown as {
      __useViewerStore?: {
        getState: () => { isPlaying: boolean; currentTick: number }
      }
    }
    const s = w.__useViewerStore?.getState()
    return s ? { isPlaying: s.isPlaying, currentTick: s.currentTick } : null
  })
}

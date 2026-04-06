import { test, expect } from "@playwright/test"

// Unskip once the full playback pipeline is wired (P3-T06)
test.skip("event layer renders concurrent effects on radar", async ({ page }) => {
  await page.goto("/demos/demo-1")

  const container = page.getByTestId("viewer-canvas-container")
  await expect(container).toBeVisible()

  // Seek to a tick where multiple effects are active simultaneously
  // (kill + smoke + HE all visible at the same time)
  await expect(container).toHaveScreenshot("event-layer-concurrent-effects.png")
})

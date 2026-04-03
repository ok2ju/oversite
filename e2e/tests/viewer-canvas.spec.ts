import { test, expect } from "@playwright/test"

// Unskip once the viewer route (e.g. /demos/:id) mounts ViewerCanvas
test.skip("viewer canvas renders PixiJS canvas element", async ({ page }) => {
  await page.goto("/demos/test-demo-id")

  const container = page.getByTestId("viewer-canvas-container")
  await expect(container).toBeVisible()

  const canvas = container.locator("canvas")
  await expect(canvas).toBeVisible()

  await expect(container).toHaveScreenshot("viewer-canvas-initial.png")
})

import { test, expect } from "@playwright/test"

// Unskip once the full stack (API + worker + DB) is available via docker-compose
test.skip("upload demo → parse → appears in library", async ({ page }) => {
  // 1. Navigate to demo library
  await page.goto("/demos")

  // 2. Verify empty state
  await expect(page.getByText(/no demos yet/i)).toBeVisible()

  // 3. Click upload and select a .dem file
  await page.getByRole("button", { name: /upload demo/i }).click()
  await expect(page.getByRole("dialog")).toBeVisible()

  const fileInput = page.getByTestId("file-input")
  await fileInput.setInputFiles("e2e/fixtures/test.dem")

  await page.getByRole("button", { name: /^upload$/i }).click()

  // 4. Wait for "uploaded" status to appear
  await expect(page.getByText("uploaded")).toBeVisible({ timeout: 10000 })

  // 5. Wait for "ready" status (worker parses the demo)
  await expect(page.getByText("ready")).toBeVisible({ timeout: 60000 })

  // 6. Verify demo appears with map name
  await expect(page.getByText(/de_/)).toBeVisible()
})

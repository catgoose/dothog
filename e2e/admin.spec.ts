import { test, expect } from "@playwright/test";
import { navigateTo, waitForHtmx } from "./helpers";

test.describe("Admin Page", () => {
  test("renders page with title", async ({ page }) => {
    await navigateTo(page, "/admin");
    await expect(page.locator("h1")).toContainText("Admin");
  });

  test("database status is visible", async ({ page }) => {
    await navigateTo(page, "/admin");
    await expect(page.locator("#db-status")).toBeVisible();
  });

  test("database table info is shown", async ({ page }) => {
    await navigateTo(page, "/admin");
    // Should show table headers for DB info
    for (const col of ["Table", "Columns", "Rows"]) {
      await expect(page.locator(`th:has-text("${col}")`)).toBeVisible();
    }
  });

  test("re-init database button works", async ({ page }) => {
    await navigateTo(page, "/admin");
    const reinitBtn = page.locator('button:has-text("Re-init")').first();
    await expect(reinitBtn).toBeVisible();
    await reinitBtn.click();
    await waitForHtmx(page);
    // DB status should refresh without errors
    await expect(page.locator("#db-status")).toBeVisible();
  });
});

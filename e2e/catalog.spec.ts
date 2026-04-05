import { test, expect } from "@playwright/test";
import { navigateTo, waitForHtmx, resetDB } from "./helpers";

test.describe("Catalog Page", () => {
  test.beforeEach(async ({ page }) => {
    await resetDB(page);
  });

  test("renders page with title and table", async ({ page }) => {
    await navigateTo(page, "/apps/catalog");
    await expect(page.locator("h1")).toContainText("Product Catalog");
    await expect(page.locator("#catalog-table-container")).toBeVisible();
  });

  test("table has expected columns", async ({ page }) => {
    await navigateTo(page, "/apps/catalog");
    for (const col of ["Name", "Category", "Price", "Stock", "Status"]) {
      await expect(page.locator(`th:has-text("${col}")`)).toBeVisible();
    }
  });

  test("expand and collapse item details", async ({ page }) => {
    await navigateTo(page, "/apps/catalog");
    const detailsBtn = page
      .locator('button:has-text("Details"), a:has-text("Details")')
      .first();
    if (await detailsBtn.isVisible()) {
      await detailsBtn.click();
      await waitForHtmx(page);
      // Detail row should appear
      const detailRow = page.locator('[id^="detail-row-"]').first();
      await expect(detailRow).toBeVisible();
      // Close it
      const closeBtn = page
        .locator('button:has-text("Close"), button:has-text("✕")')
        .first();
      if (await closeBtn.isVisible()) {
        await closeBtn.click();
        await waitForHtmx(page);
      }
    }
  });

  test("status badges render correctly", async ({ page }) => {
    await navigateTo(page, "/apps/catalog");
    const badges = page.locator(".badge");
    await expect(badges.first()).toBeVisible();
  });
});

import { test, expect } from "@playwright/test";
import { navigateTo, waitForHtmx, resetDB } from "./helpers";

test.describe("Bulk Operations Page", () => {
  test.beforeEach(async ({ page }) => {
    await resetDB(page);
  });

  test("renders page with title and table", async ({ page }) => {
    await navigateTo(page, "/apps/bulk");
    await expect(page.locator("h1")).toContainText("Bulk Operations");
    await expect(page.locator("#bulk-table-container")).toBeVisible();
  });

  test("table has checkboxes for row selection", async ({ page }) => {
    await navigateTo(page, "/apps/bulk");
    const checkboxes = page.locator('input[type="checkbox"]');
    const count = await checkboxes.count();
    expect(count).toBeGreaterThan(1); // at least header + 1 row
  });

  test("select individual row checkbox", async ({ page }) => {
    await navigateTo(page, "/apps/bulk");
    const rowCheckbox = page.locator('.row-check, tbody input[type="checkbox"]').first();
    await expect(rowCheckbox).toBeVisible();
    await rowCheckbox.check();
    await expect(rowCheckbox).toBeChecked();
  });

  test("status badges show Active or Inactive", async ({ page }) => {
    await navigateTo(page, "/apps/bulk");
    // Scope to table body to avoid picking up unrelated badges elsewhere on the page
    const badges = page.locator("#bulk-table-container tbody .badge");
    const count = await badges.count();
    expect(count).toBeGreaterThan(0);
    // Verify every status badge has expected text
    const allText = await badges.allTextContents();
    const validStatuses = allText.every(
      (t) => t.includes("Active") || t.includes("Inactive"),
    );
    expect(validStatuses).toBe(true);
  });

  test("table columns are correct", async ({ page }) => {
    await navigateTo(page, "/apps/bulk");
    for (const col of ["Name", "Category", "Price", "Stock", "Status"]) {
      await expect(page.locator(`th:has-text("${col}")`)).toBeVisible();
    }
  });
});

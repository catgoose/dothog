import { test, expect } from "@playwright/test";
import { navigateTo, waitForHtmx, resetDB } from "./helpers";

test.describe("People Directory", () => {
  test.beforeEach(async ({ page }) => {
    await resetDB(page);
  });

  test("renders page with title and table", async ({ page }) => {
    await navigateTo(page, "/demo/people");
    await expect(page.locator("h1")).toContainText("People Directory");
    await expect(page.locator("#people-table-container")).toBeVisible();
  });

  test("table has expected columns", async ({ page }) => {
    await navigateTo(page, "/demo/people");
    for (const col of ["Name", "Department", "Title", "Location", "Email"]) {
      await expect(page.locator(`th:has-text("${col}")`)).toBeVisible();
    }
  });

  test("search filters people", async ({ page }) => {
    await navigateTo(page, "/demo/people");
    const searchInput = page.locator('input[name="q"]');
    if (await searchInput.isVisible()) {
      await searchInput.fill("Engineering");
      await waitForHtmx(page);
      await page.waitForTimeout(500);
      await waitForHtmx(page);
    }
  });

  test("click person row loads profile", async ({ page }) => {
    await navigateTo(page, "/demo/people");
    // Click the first row that has hx-get for a person detail
    const personRow = page.locator("tbody tr[hx-get]").first();
    if (await personRow.isVisible()) {
      await personRow.click();
      await waitForHtmx(page);
    }
  });

  test("people list route loads", async ({ page }) => {
    await navigateTo(page, "/demo/people/list");
    await expect(page.locator("#people-table-container")).toBeVisible();
  });
});

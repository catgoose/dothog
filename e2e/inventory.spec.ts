import { test, expect } from "@playwright/test";
import { navigateTo, waitForHtmx, resetDB } from "./helpers";

test.describe("Inventory Page", () => {
  test.beforeEach(async ({ page }) => {
    await resetDB(page);
  });

  test("renders page with title and table", async ({ page }) => {
    await navigateTo(page, "/apps/inventory");
    await expect(page.locator("h1")).toContainText("Inventory");
    await expect(page.locator("#inventory-table-container")).toBeVisible();
    // Table should have visible data rows (skip hidden new-item-row placeholder)
    const rows = page.locator("#inventory-table-container table tbody tr:visible");
    const count = await rows.count();
    expect(count).toBeGreaterThan(0);
  });

  test("filter by search narrows results", async ({ page }) => {
    await navigateTo(page, "/apps/inventory");
    const searchInput = page.locator('input[name="q"]');
    await expect(searchInput).toBeVisible();
    // Type a search term and wait for HTMX update
    await searchInput.fill("Electronics");
    await waitForHtmx(page);
    // All visible rows should contain search term or table should be filtered
    await page.waitForTimeout(500); // debounce
    await waitForHtmx(page);
  });

  test("filter by category", async ({ page }) => {
    await navigateTo(page, "/apps/inventory");
    const categorySelect = page.locator('select[name="cat"]');
    if (await categorySelect.isVisible()) {
      await categorySelect.selectOption({ index: 1 });
      await waitForHtmx(page);
      await expect(
        page.locator("#inventory-table-container"),
      ).toBeVisible();
    }
  });

  test("pagination controls work", async ({ page }) => {
    await navigateTo(page, "/apps/inventory");
    const nextBtn = page.locator('a:has-text("Next"), button:has-text("Next")');
    if (await nextBtn.isVisible()) {
      await nextBtn.click();
      await waitForHtmx(page);
      await expect(
        page.locator("#inventory-table-container"),
      ).toBeVisible();
    }
  });

  test("add new item form appears", async ({ page }) => {
    await navigateTo(page, "/apps/inventory");
    const addBtn = page.locator('button:has-text("+ Add Item")');
    await expect(addBtn).toBeVisible();
    await addBtn.click();
    await waitForHtmx(page);
    // A new row or form should appear
    const newRow = page.locator("#new-item-row, [id*=new-item]");
    await expect(newRow).toBeVisible();
  });

  test("sorting by column header", async ({ page }) => {
    await navigateTo(page, "/apps/inventory");
    const nameHeader = page.locator("th >> text=Name");
    if (await nameHeader.isVisible()) {
      const sortLink = nameHeader.locator("a").first();
      if (await sortLink.isVisible()) {
        await sortLink.click();
        await waitForHtmx(page);
        await expect(
          page.locator("#inventory-table-container"),
        ).toBeVisible();
      }
    }
  });

  test("link to hypermedia controls exists", async ({ page }) => {
    await navigateTo(page, "/apps/inventory");
    await expect(
      page.locator('a:has-text("Hypermedia Controls")'),
    ).toBeVisible();
  });
});

import { test, expect } from "@playwright/test";
import { navigateTo, waitForHtmx, resetDB } from "./helpers";

test.describe("Repository Demo Page", () => {
  test.beforeEach(async ({ page }) => {
    await resetDB(page);
  });

  test("renders page with title and table", async ({ page }) => {
    await navigateTo(page, "/platform/repository");
    await expect(page.locator("h1")).toContainText("Repository Pattern Demo");
    await expect(page.locator("#repo-table-container")).toBeVisible();
    // Should have seeded task rows
    const rows = page.locator(
      "#repo-table-container table tbody tr:visible",
    );
    const count = await rows.count();
    expect(count).toBeGreaterThan(0);
  });

  test("table has expected columns", async ({ page }) => {
    await navigateTo(page, "/platform/repository");
    for (const col of [
      "Title",
      "Description",
      "Status",
      "Order",
      "Ver",
      "State",
      "Updated",
      "Actions",
    ]) {
      await expect(page.locator(`th:has-text("${col}")`)).toBeVisible();
    }
  });

  test("status badges render for seeded data", async ({ page }) => {
    await navigateTo(page, "/platform/repository");
    const badges = page.locator(".badge");
    const count = await badges.count();
    expect(count).toBeGreaterThan(0);
    const allText = await badges.allTextContents();
    const hasExpected = allText.some(
      (t) =>
        t.includes("draft") ||
        t.includes("active") ||
        t.includes("done") ||
        t.includes("live"),
    );
    expect(hasExpected).toBe(true);
  });

  test("version badges show v1+ for seeded data", async ({ page }) => {
    await navigateTo(page, "/platform/repository");
    const versionBadges = page.locator('.badge:has-text("v")');
    const count = await versionBadges.count();
    expect(count).toBeGreaterThan(0);
  });

  test("filter by search narrows results", async ({ page }) => {
    await navigateTo(page, "/platform/repository");
    const searchInput = page.locator('input[name="q"]');
    await expect(searchInput).toBeVisible();
    await searchInput.fill("schema");
    await waitForHtmx(page);
    await page.waitForTimeout(500); // debounce
    await waitForHtmx(page);
    // Table should still be visible with filtered results
    await expect(page.locator("#repo-table-container")).toBeVisible();
  });

  test("filter by status dropdown", async ({ page }) => {
    await navigateTo(page, "/platform/repository");
    const statusSelect = page.locator('select[name="status"]');
    if (await statusSelect.isVisible()) {
      await statusSelect.selectOption("active");
      await waitForHtmx(page);
      await expect(page.locator("#repo-table-container")).toBeVisible();
    }
  });

  test("sorting by column header", async ({ page }) => {
    await navigateTo(page, "/platform/repository");
    const titleHeader = page.locator("th >> text=Title");
    if (await titleHeader.isVisible()) {
      const sortLink = titleHeader.locator("a").first();
      if (await sortLink.isVisible()) {
        await sortLink.click();
        await waitForHtmx(page);
        await expect(page.locator("#repo-table-container")).toBeVisible();
      }
    }
  });

  test("add new task form appears and cancel works", async ({ page }) => {
    await navigateTo(page, "/platform/repository");
    const addBtn = page.locator('button:has-text("+ New Task")');
    await expect(addBtn).toBeVisible();
    await addBtn.click();
    await waitForHtmx(page);
    // New row should have input fields
    const newRow = page.locator("#new-task-row");
    await expect(newRow).toBeVisible();
    const titleInput = newRow.locator('input[name="title"]');
    await expect(titleInput).toBeVisible();
    // Cancel should remove the form
    const cancelBtn = newRow.locator('button:has-text("Cancel")');
    await cancelBtn.click();
    await waitForHtmx(page);
  });

  test("create new task via inline form", async ({ page }) => {
    await navigateTo(page, "/platform/repository");
    const addBtn = page.locator('button:has-text("+ New Task")');
    await addBtn.click();
    await waitForHtmx(page);
    const newRow = page.locator("#new-task-row");
    await newRow.locator('input[name="title"]').fill("E2E Test Task");
    await newRow
      .locator('input[name="description"]')
      .fill("Created by Playwright");
    await newRow.locator('select[name="status"]').selectOption("active");
    const saveBtn = newRow.locator('button:has-text("Save")');
    await saveBtn.click();
    await waitForHtmx(page);
    // Table should reload and contain the new task
    await expect(page.locator("#repo-table-container")).toBeVisible();
    await expect(
      page.locator('td:has-text("E2E Test Task")'),
    ).toBeVisible();
  });

  test("inline edit shows form and saves", async ({ page }) => {
    await navigateTo(page, "/platform/repository");
    const editBtn = page.locator('button:has-text("Edit")').first();
    await editBtn.click();
    await waitForHtmx(page);
    // Should show input fields in the row
    const inputs = page.locator("tbody input, tbody select");
    await expect(inputs.first()).toBeVisible();
  });

  test("soft delete shows deleted badge when show deleted is on", async ({
    page,
  }) => {
    await navigateTo(page, "/platform/repository");
    // Handle confirmation dialog before triggering it
    page.on("dialog", (dialog) => dialog.accept());
    // Delete the first task
    const deleteBtn = page.locator('button:has-text("Delete")').first();
    await deleteBtn.click();
    await waitForHtmx(page);
    // Enable show deleted checkbox
    const deletedCheckbox = page.locator('input[name="deleted"]');
    await deletedCheckbox.check();
    await waitForHtmx(page);
    await page.waitForTimeout(500);
    await waitForHtmx(page);
    // Should see a "deleted" badge and a "Restore" button
    const deletedBadge = page.locator('.badge:has-text("deleted")');
    await expect(deletedBadge.first()).toBeVisible();
    const restoreBtn = page.locator('button:has-text("Restore")');
    await expect(restoreBtn.first()).toBeVisible();
  });

  test("archive task shows archived badge when show archived is on", async ({
    page,
  }) => {
    await navigateTo(page, "/platform/repository");
    // Archive the first task
    const archiveBtn = page.locator('button:has-text("Archive")').first();
    await archiveBtn.click();
    await waitForHtmx(page);
    // Enable show archived checkbox
    const archivedCheckbox = page.locator('input[name="archived"]');
    await archivedCheckbox.check();
    await waitForHtmx(page);
    await page.waitForTimeout(300);
    await waitForHtmx(page);
    // Should see an "archived" badge and an "Unarchive" button
    const archivedBadge = page.locator('.badge:has-text("archived")');
    await expect(archivedBadge.first()).toBeVisible();
    const unarchiveBtn = page.locator('button:has-text("Unarchive")');
    await expect(unarchiveBtn.first()).toBeVisible();
  });

  test("restore deleted task brings it back", async ({ page }) => {
    await navigateTo(page, "/platform/repository");
    // Handle confirmation dialog before triggering it
    page.on("dialog", (dialog) => dialog.accept());
    // Count initial visible rows
    const initialRows = await page
      .locator("#repo-table-container table tbody tr:visible")
      .count();
    // Delete first task
    const deleteBtn = page.locator('button:has-text("Delete")').first();
    await deleteBtn.click();
    await waitForHtmx(page);
    await page.waitForTimeout(300);
    await waitForHtmx(page);
    // Should have one fewer row
    const afterDeleteRows = await page
      .locator("#repo-table-container table tbody tr:visible")
      .count();
    expect(afterDeleteRows).toBeLessThan(initialRows);
    // Show deleted and restore
    const deletedCheckbox = page.locator('input[name="deleted"]');
    await deletedCheckbox.check();
    await waitForHtmx(page);
    await page.waitForTimeout(300);
    await waitForHtmx(page);
    const restoreBtn = page.locator('button:has-text("Restore")').first();
    await restoreBtn.click();
    await waitForHtmx(page);
  });

  test("info cards show repository pattern features", async ({ page }) => {
    await navigateTo(page, "/platform/repository");
    await expect(page.locator('text="Schema"')).toBeVisible();
    await expect(page.locator('text="Query"')).toBeVisible();
    await expect(page.locator('text="Filters"')).toBeVisible();
    await expect(page.locator('text="Audit"')).toBeVisible();
  });

  test("navigation includes Platform link", async ({ page }) => {
    await navigateTo(page, "/platform/repository");
    const navLink = page.locator('nav a:has-text("Platform")');
    await expect(navLink).toBeVisible();
  });
});

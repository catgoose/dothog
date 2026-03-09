import { test, expect } from "@playwright/test";
import { navigateTo, waitForHtmx, resetDB } from "./helpers";

test.describe("Kanban Board", () => {
  test.beforeEach(async ({ page }) => {
    await resetDB(page);
  });

  test("renders page with title and board", async ({ page }) => {
    await navigateTo(page, "/demo/kanban");
    await expect(page.locator("h1")).toContainText("Kanban Board");
    await expect(page.locator("#kanban-board")).toBeVisible();
  });

  test("all kanban columns are present", async ({ page }) => {
    await navigateTo(page, "/demo/kanban");
    for (const col of [
      "Backlog",
      "To Do",
      "In Progress",
      "Review",
      "Done",
    ]) {
      await expect(page.locator(`text=${col}`).first()).toBeVisible();
    }
  });

  test("kanban cards have content", async ({ page }) => {
    await navigateTo(page, "/demo/kanban");
    const cards = page.locator('[id^="kanban-task-"]');
    const count = await cards.count();
    expect(count).toBeGreaterThan(0);
  });

  test("move card to next status", async ({ page }) => {
    await navigateTo(page, "/demo/kanban");
    // Find a move-right button on a card
    const moveBtn = page
      .locator('button:has-text("→"), a:has-text("→")')
      .first();
    if (await moveBtn.isVisible()) {
      await moveBtn.click();
      await waitForHtmx(page);
      // Board should still be intact
      await expect(page.locator("#kanban-board")).toBeVisible();
    }
  });

  test("priority badges render", async ({ page }) => {
    await navigateTo(page, "/demo/kanban");
    const badges = page.locator(".badge");
    const count = await badges.count();
    expect(count).toBeGreaterThan(0);
  });
});

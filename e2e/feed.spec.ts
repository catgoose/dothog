import { test, expect } from "@playwright/test";
import { navigateTo, resetDB } from "./helpers";

test.describe("Activity Feed", () => {
  test.beforeEach(async ({ page }) => {
    await resetDB(page);
  });

  test("renders page with title", async ({ page }) => {
    await navigateTo(page, "/realtime/feed");
    await expect(page.locator("h1")).toContainText("Activity Feed");
  });

  test("feed container is present", async ({ page }) => {
    await navigateTo(page, "/realtime/feed");
    await expect(page.locator("#feed-container")).toBeVisible();
  });

  test("feed items display", async ({ page }) => {
    await navigateTo(page, "/realtime/feed");
    await expect(page.locator("#feed-items")).toBeVisible();
  });
});

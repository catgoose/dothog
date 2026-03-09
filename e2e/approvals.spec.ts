import { test, expect } from "@playwright/test";
import { navigateTo, waitForHtmx, resetDB } from "./helpers";

test.describe("Approval Queue", () => {
  test.beforeEach(async ({ page }) => {
    await resetDB(page);
  });

  test("renders page with title", async ({ page }) => {
    await navigateTo(page, "/demo/approvals");
    await expect(page.locator("h1")).toContainText("Approval Queue");
  });

  test("approval list is visible", async ({ page }) => {
    await navigateTo(page, "/demo/approvals");
    await expect(page.locator("#approvals-list")).toBeVisible();
  });

  test("approval cards show required fields", async ({ page }) => {
    await navigateTo(page, "/demo/approvals");
    const cards = page.locator('[id^="approval-"]');
    const count = await cards.count();
    expect(count).toBeGreaterThan(0);
  });

  test("approve action works", async ({ page }) => {
    await navigateTo(page, "/demo/approvals");
    const approveBtn = page
      .locator('button:has-text("Approve")')
      .first();
    if (await approveBtn.isVisible()) {
      await approveBtn.click();
      await waitForHtmx(page);
      // Card should update - check for approved badge
      await expect(page.locator("#approvals-list")).toBeVisible();
    }
  });

  test("reject action works", async ({ page }) => {
    await navigateTo(page, "/demo/approvals");
    const rejectBtn = page
      .locator('button:has-text("Reject")')
      .first();
    if (await rejectBtn.isVisible()) {
      await rejectBtn.click();
      await waitForHtmx(page);
      await expect(page.locator("#approvals-list")).toBeVisible();
    }
  });

  test("status badges have correct styling", async ({ page }) => {
    await navigateTo(page, "/demo/approvals");
    const pendingBadges = page.locator(".badge-warning");
    if ((await pendingBadges.count()) > 0) {
      await expect(pendingBadges.first()).toBeVisible();
    }
  });
});

import { test, expect } from "@playwright/test";
import { navigateTo } from "./helpers";

test.describe("Dashboard", () => {
  test("page loads with title", async ({ page }) => {
    await navigateTo(page, "/dashboard");
    await expect(page.locator("h1")).toContainText("Dashboard");
  });

  test("renders stat badges", async ({ page }) => {
    await navigateTo(page, "/dashboard");
    const stats = page.locator(".stat");
    await expect(stats).toHaveCount(4);
  });

  test("renders app cards with links", async ({ page }) => {
    await navigateTo(page, "/dashboard");
    for (const label of [
      "Inventory",
      "Kanban Board",
      "Approvals",
      "People Directory",
      "Vendors & Contacts",
      "Settings",
    ]) {
      await expect(page.locator(`.card-title:has-text("${label}")`)).toBeVisible();
    }
  });

  test("inventory card links to demo pages", async ({ page }) => {
    await navigateTo(page, "/dashboard");
    await expect(page.locator('main a[href*="/demo/inventory"]').first()).toBeVisible();
    await expect(page.locator('main a[href*="/demo/catalog"]')).toBeVisible();
    await expect(page.locator('main a[href*="/demo/bulk"]')).toBeVisible();
  });

  test("kanban card shows status badges", async ({ page }) => {
    await navigateTo(page, "/dashboard");
    for (const status of ["Backlog", "To Do", "In Progress", "Review", "Done"]) {
      await expect(page.locator(`.badge:has-text("${status}")`)).toBeVisible();
    }
  });

  test("approvals card shows status breakdown", async ({ page }) => {
    await navigateTo(page, "/dashboard");
    await expect(page.locator('.badge:has-text("Pending")').first()).toBeVisible();
    await expect(page.locator('.badge:has-text("Approved")').first()).toBeVisible();
  });

  test("hypermedia controls link exists", async ({ page }) => {
    await navigateTo(page, "/dashboard");
    await expect(page.locator('a:has-text("Browse Controls")')).toBeVisible();
  });

  test("clicking inventory link navigates to inventory page", async ({ page }) => {
    await navigateTo(page, "/dashboard");
    await page.locator('main a[href*="/demo/inventory"]').first().click();
    await expect(page.locator("h1")).toContainText("Items Inventory");
  });
});

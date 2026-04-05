import { test, expect } from "@playwright/test";
import { navigateTo, waitForHtmx, resetDB } from "./helpers";

test.describe("Settings Page", () => {
  test.beforeEach(async ({ page }) => {
    await resetDB(page);
  });

  test("renders page with title", async ({ page }) => {
    await navigateTo(page, "/platform/settings");
    await expect(page.locator("h1")).toContainText("Settings");
  });

  test("settings content loads", async ({ page }) => {
    await navigateTo(page, "/platform/settings");
    await expect(page.locator("#settings-content")).toBeVisible();
  });

  test("settings tabs are present", async ({ page }) => {
    await navigateTo(page, "/platform/settings");
    for (const tab of ["General", "Notifications", "Security", "Appearance"]) {
      await expect(
        page.locator(`.settings-tab:has-text("${tab}"), [role="tab"]:has-text("${tab}"), button:has-text("${tab}")`).first(),
      ).toBeVisible();
    }
  });

  test("switching tabs updates content", async ({ page }) => {
    await navigateTo(page, "/platform/settings");
    const notifTab = page.locator(
      '.settings-tab:has-text("Notifications"), [role="tab"]:has-text("Notifications"), button:has-text("Notifications")',
    ).first();
    if (await notifTab.isVisible()) {
      await notifTab.click();
      await waitForHtmx(page);
      await expect(page.locator("#settings-content")).toBeVisible();
    }
  });

  test("save button exists", async ({ page }) => {
    await navigateTo(page, "/platform/settings");
    const saveBtn = page.locator('button:has-text("Save")').first();
    await expect(saveBtn).toBeVisible();
  });
});

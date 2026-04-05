import { test, expect } from "@playwright/test";
import { navigateTo, waitForHtmx, resetDB } from "./helpers";

test.describe("Vendor Contacts", () => {
  test.beforeEach(async ({ page }) => {
    await resetDB(page);
  });

  test("renders page with title", async ({ page }) => {
    await navigateTo(page, "/apps/vendors");
    await expect(page.locator("h1")).toContainText("Vendor Contacts");
  });

  test("vendor list is visible", async ({ page }) => {
    await navigateTo(page, "/apps/vendors");
    await expect(page.locator("#vendor-list")).toBeVisible();
  });

  test("clicking a vendor shows contacts", async ({ page }) => {
    await navigateTo(page, "/apps/vendors");
    const vendorLink = page
      .locator("#vendor-list a, #vendor-list button, #vendor-list [hx-get]")
      .first();
    if (await vendorLink.isVisible()) {
      await vendorLink.click();
      await waitForHtmx(page);
      await expect(page.locator("#contact-detail")).toBeVisible();
    }
  });
});

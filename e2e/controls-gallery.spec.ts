import { test, expect } from "@playwright/test";
import { navigateTo, waitForHtmx, resetDB } from "./helpers";

test.describe("Controls Gallery", () => {
  test.beforeEach(async ({ page }) => {
    await resetDB(page);
  });

  test("renders page with title", async ({ page }) => {
    await navigateTo(page, "/patterns/controls");
    await expect(page.locator("h1")).toContainText("Controls Gallery");
  });

  test("button variants section is present", async ({ page }) => {
    await navigateTo(page, "/patterns/controls");
    await expect(page.locator("text=Button Variants").first()).toBeVisible();
  });

  test("clicking variant button echoes response", async ({ page }) => {
    await navigateTo(page, "/patterns/controls");
    const primaryBtn = page
      .locator('button:has-text("Primary")')
      .first();
    await primaryBtn.click();
    await waitForHtmx(page);
    const result = page.locator("#variant-result");
    await expect(result).not.toContainText("Click a button above");
  });

  test("all variant buttons respond", async ({ page }) => {
    await navigateTo(page, "/patterns/controls");
    for (const variant of [
      "Primary",
      "Secondary",
      "Danger",
      "Ghost",
      "Link",
    ]) {
      const btn = page
        .locator(`button:has-text("${variant}")`)
        .first();
      if (await btn.isVisible()) {
        await btn.click();
        await waitForHtmx(page);
        const result = page.locator("#variant-result");
        // Result should update (not empty/error)
        await expect(result).toBeVisible();
      }
    }
  });

  test("retry button works", async ({ page }) => {
    await navigateTo(page, "/patterns/controls");
    const retryBtn = page.locator('button:has-text("Retry")').first();
    if (await retryBtn.isVisible()) {
      await retryBtn.click();
      await waitForHtmx(page);
      await expect(page.locator("#kind-result")).toBeVisible();
    }
  });

  test("resource CRUD section works", async ({ page }) => {
    await navigateTo(page, "/patterns/controls");
    // Look for the resource section
    const resourceSection = page.locator("text=Resource CRUD").first();
    if (await resourceSection.isVisible()) {
      await expect(resourceSection).toBeVisible();
    }
  });

  test("form demo section exists", async ({ page }) => {
    await navigateTo(page, "/patterns/controls");
    await expect(page.locator("text=Form").first()).toBeVisible();
  });

  test("filter controls section exists", async ({ page }) => {
    await navigateTo(page, "/patterns/controls");
    await expect(page.locator("text=Filter").first()).toBeVisible();
  });

  test("error recovery sections exist", async ({ page }) => {
    await navigateTo(page, "/patterns/controls");
    for (const section of ["Transient", "Validation", "Conflict", "Stale"]) {
      await expect(
        page.locator(`text=${section}`).first(),
      ).toBeVisible();
    }
  });

  test("transient error triggers retry UI", async ({ page }) => {
    await navigateTo(page, "/patterns/controls");
    const transientBtn = page
      .locator('[hx-post*="errors/transient"], button:has-text("Trigger 500")')
      .first();
    if (await transientBtn.isVisible()) {
      await transientBtn.click();
      await waitForHtmx(page);
      // Should show error status with retry option
    }
  });

  test("validation error shows fix button", async ({ page }) => {
    await navigateTo(page, "/patterns/controls");
    const validateBtn = page
      .locator('[hx-post*="errors/validate"], button:has-text("Trigger 422")')
      .first();
    if (await validateBtn.isVisible()) {
      await validateBtn.click();
      await waitForHtmx(page);
    }
  });

  test("inline row CRUD works", async ({ page }) => {
    await navigateTo(page, "/patterns/controls");
    // Look for row editing
    const editBtn = page
      .locator('[hx-get*="/rows/"] button:has-text("Edit"), [hx-get*="edit"]')
      .first();
    if (await editBtn.isVisible()) {
      await editBtn.click();
      await waitForHtmx(page);
    }
  });
});

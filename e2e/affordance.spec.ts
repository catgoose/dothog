import { test, expect } from "@playwright/test";
import { navigateTo, waitForHtmx, resetDB } from "./helpers";

// Affordance contract: buttons do actions, links navigate, every control
// carries a programmatic accessible name, icon-only controls name the action
// they perform, and accordions use native elements.

test.describe("Affordance: actions are buttons, not links or fake buttons", () => {
  test.beforeEach(async ({ page }) => {
    await resetDB(page);
  });

  test("notifications identity control is a button menu issuing a POST", async ({
    page,
  }) => {
    await navigateTo(page, "/realtime/notifications");
    await expect(page.locator('div[role="button"]')).toHaveCount(0);
    const choices = page.locator(
      'button[hx-post="/realtime/notifications/identity"]',
    );
    expect(await choices.count()).toBeGreaterThan(0);
    // The identity switch mutates, so it must not hang off an anchor.
    await expect(
      page.locator('a[hx-post*="/notifications/identity"]'),
    ).toHaveCount(0);
  });
});

test.describe("Affordance: icon-only controls expose accessible names", () => {
  test.beforeEach(async ({ page }) => {
    await resetDB(page);
  });

  test("canvas palette buttons name the color", async ({ page }) => {
    await navigateTo(page, "/realtime/canvas");
    await expect(
      page.getByRole("button", { name: /^Select color / }).first(),
    ).toBeVisible();
  });

  test("hotzone nudge buttons name the action", async ({ page }) => {
    await navigateTo(page, "/realtime/tavern/hotzones");
    await expect(
      page.getByRole("button", { name: "Decrease update interval" }).first(),
    ).toBeVisible();
    await expect(
      page.getByRole("button", { name: "Increase update interval" }).first(),
    ).toBeVisible();
  });

  test("realtime interval slider is reachable by accessible name", async ({
    page,
  }) => {
    await navigateTo(page, "/realtime/dashboard");
    await expect(page.getByLabel("Update interval").first()).toBeVisible();
  });

  test("controls gallery inline-edit inputs are named", async ({ page }) => {
    await navigateTo(page, "/patterns/controls");
    await expect(page.getByLabel("Item name").first()).toBeVisible();
    await expect(page.getByLabel(/^Category for /).first()).toBeVisible();
    await expect(page.getByLabel(/^Price for /).first()).toBeVisible();
    await expect(page.getByLabel(/^Active for /).first()).toBeVisible();
  });

  test("repository inline-edit controls are named", async ({ page }) => {
    await navigateTo(page, "/platform/repository");
    await page.locator('button:has-text("Edit")').first().click();
    await waitForHtmx(page);
    await expect(page.getByLabel(/^Status for /).first()).toBeVisible();
    await expect(page.getByLabel(/^Sort order for /).first()).toBeVisible();
  });
});

test.describe("Affordance: form fields are label-associated", () => {
  test.beforeEach(async ({ page }) => {
    await resetDB(page);
  });

  test("settings fields are reachable by their visible label", async ({
    page,
  }) => {
    await navigateTo(page, "/platform/settings");
    await waitForHtmx(page);
    await expect(page.getByLabel("App Name")).toBeVisible();
  });

  test("links editor fields are reachable by their labels", async ({ page }) => {
    await navigateTo(page, "/api/links");
    for (const name of ["Source", "Rel", "Target", "Title", "Group"]) {
      await expect(page.getByLabel(name)).toBeVisible();
    }
  });

  test("list filters are reachable by accessible name", async ({ page }) => {
    await navigateTo(page, "/patterns/lists");
    await expect(page.getByLabel("Category")).toBeVisible();
  });
});

test.describe("Affordance: accordions are native exclusive details", () => {
  test("component accordions use <details name> for exclusive open", async ({
    page,
  }) => {
    await navigateTo(page, "/components/cards");
    const panels = page.locator('details[name="accordion-panels"]');
    expect(await panels.count()).toBeGreaterThan(1);
    // Opening a second panel closes the first (native exclusive accordion).
    const first = panels.nth(0);
    const second = panels.nth(1);
    await first.locator("summary").click();
    await expect(first).toHaveJSProperty("open", true);
    await second.locator("summary").click();
    await expect(second).toHaveJSProperty("open", true);
    await expect(first).toHaveJSProperty("open", false);
  });
});

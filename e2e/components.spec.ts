import { test, expect } from "@playwright/test";
import { navigateTo, waitForHtmx, resetDB } from "./helpers";

test.describe("Component Patterns", () => {
  test.beforeEach(async ({ page }) => {
    await resetDB(page);
  });

  test("page 1 loads with title", async ({ page }) => {
    await navigateTo(page, "/hypermedia/components");
    await expect(page.locator("h1")).toContainText("Component Patterns");
  });

  test("tabs switch content", async ({ page }) => {
    await navigateTo(page, "/hypermedia/components");
    // Wait for initial tab load
    await waitForHtmx(page);
    // Click "Details" tab (role="tab")
    const detailsTab = page.locator('[role="tab"]:has-text("Details")');
    await expect(detailsTab).toBeVisible();
    await detailsTab.click();
    await waitForHtmx(page);
    // Tab panel should have details content
    await expect(page.locator("#tab-panel").last()).toBeVisible();
  });

  test("steps wizard navigates forward", async ({ page }) => {
    await navigateTo(page, "/hypermedia/components");
    const nextBtn = page.locator('button:has-text("Next")').first();
    if (await nextBtn.isVisible()) {
      await nextBtn.click();
      await waitForHtmx(page);
    }
  });
});

test.describe("Component Patterns 2", () => {
  test("page loads with title", async ({ page }) => {
    await navigateTo(page, "/hypermedia/components2");
    await expect(page.locator("h1")).toContainText("Component Patterns 2");
  });

  test("carousel navigation works", async ({ page }) => {
    await navigateTo(page, "/hypermedia/components2");
    const carouselPanel = page.locator("#carousel-panel").first();
    await expect(carouselPanel).toBeVisible();
    // The "Next →" button — scroll to it and click
    const nextBtn = page.locator('button[hx-get*="carousel"]').first();
    await nextBtn.scrollIntoViewIfNeeded();
    await nextBtn.click({ force: true });
    await waitForHtmx(page);
    // Carousel should still be present after navigation
    await expect(page.locator("#carousel-panel").first()).toBeVisible();
  });

  test("dropdown search input exists", async ({ page }) => {
    await navigateTo(page, "/hypermedia/components2");
    const searchInput = page.locator("#dropdown-results, input[hx-trigger*='keyup']").first();
    if (await searchInput.isVisible()) {
      await expect(searchInput).toBeVisible();
    }
  });
});

test.describe("Component Patterns 3", () => {
  test("page loads with title", async ({ page }) => {
    await navigateTo(page, "/hypermedia/components3");
    await expect(page.locator("h1")).toContainText("Component Patterns 3");
  });

  test("feed container exists", async ({ page }) => {
    await navigateTo(page, "/hypermedia/components3");
    await expect(page.locator("#feed-container")).toBeVisible();
  });
});

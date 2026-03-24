import { test, expect } from "@playwright/test";
import { navigateTo } from "./helpers";

test.describe("Home Page", () => {
  test("renders hero section with title and CTAs", async ({ page }) => {
    await navigateTo(page, "/");
    await expect(page.locator("h1")).toContainText("HATEOAS & REST");
    await expect(
      page.locator('#base-content a:has-text("Dashboard")'),
    ).toHaveAttribute("href", "/dashboard");
    await expect(
      page.locator('a:has-text("Controls Gallery")'),
    ).toHaveAttribute("href", "/hypermedia/controls");
  });

  test("renders reach-up pyramid sections", async ({ page }) => {
    await navigateTo(page, "/");
    await expect(page.locator("text=The Reach-Up Model")).toBeVisible();
    await expect(page.locator("h3:has-text('Behavior')")).toBeVisible();
    await expect(page.locator("h3:has-text('Presentation')")).toBeVisible();
  });

  test("renders domain map cards", async ({ page }) => {
    await navigateTo(page, "/");
    await expect(page.locator(".card-title:has-text('Go + SQL')")).toBeVisible();
    await expect(page.locator(".card-title:has-text('HTTP + HTMX')")).toBeVisible();
    await expect(page.locator(".card-title:has-text('templ + DaisyUI')")).toBeVisible();
    await expect(page.locator(".card-title:has-text('Alpine.js')")).toBeVisible();
  });

  test("renders see it in action section", async ({ page }) => {
    await navigateTo(page, "/");
    await expect(page.locator("text=See It In Action")).toBeVisible();
  });

  test("navbar is present", async ({ page }) => {
    await navigateTo(page, "/");
    await expect(page.locator("nav.navbar")).toBeVisible();
  });

  test("health endpoint returns OK", async ({ request }) => {
    const resp = await request.get("/health");
    expect(resp.ok()).toBe(true);
    const body = await resp.json();
    expect(body.status).toBe("healthy");
    expect(body.name).toBe("dothog");
  });
});

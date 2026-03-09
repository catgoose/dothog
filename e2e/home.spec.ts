import { test, expect } from "@playwright/test";
import { navigateTo } from "./helpers";

test.describe("Home Page", () => {
  test("renders hero section with title and CTAs", async ({ page }) => {
    await navigateTo(page, "/");
    await expect(page.locator("h1")).toContainText("Go + HTMX + Templ");
    await expect(page.locator('.hero a:has-text("Dashboard")')).toHaveAttribute(
      "href",
      "/dashboard",
    );
    await expect(
      page.locator('a:has-text("Controls Gallery")'),
    ).toHaveAttribute("href", "/hypermedia/controls");
  });

  test("renders stack cards", async ({ page }) => {
    await navigateTo(page, "/");
    await expect(page.locator(".card-title:has-text('Go + Echo')")).toBeVisible();
    await expect(page.locator(".card-title:has-text('HTMX')")).toBeVisible();
    await expect(page.locator(".card-title:has-text('Templ')")).toBeVisible();
  });

  test("renders feature cards", async ({ page }) => {
    await navigateTo(page, "/");
    await expect(page.locator(".card-title:has-text('Composable SQL Fragments')")).toBeVisible();
    await expect(page.locator(".card-title:has-text('Hypermedia Controls')")).toBeVisible();
    await expect(page.locator(".card-title:has-text('DaisyUI + Tailwind')")).toBeVisible();
    await expect(page.locator(".card-title:has-text('Feature Flags at Build Time')")).toBeVisible();
  });

  test("renders quick start section", async ({ page }) => {
    await navigateTo(page, "/");
    await expect(page.locator("text=Quick Start")).toBeVisible();
  });

  test("navbar is present", async ({ page }) => {
    await navigateTo(page, "/");
    await expect(page.locator("nav.navbar")).toBeVisible();
  });

  test("health endpoint returns OK", async ({ request }) => {
    const resp = await request.get("/health");
    expect(resp.ok()).toBe(true);
    const body = await resp.json();
    expect(body.status).toBe("ok");
    expect(body.time).toBeTruthy();
  });
});

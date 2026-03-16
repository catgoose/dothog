#!/usr/bin/env node

// Captures screenshots of the running demo app.
// Usage: node scripts/screenshots.mjs [--base-url http://localhost:3000]
//
// Requires: npx playwright install chromium

import { chromium } from "playwright";
import { mkdirSync } from "fs";
import { resolve, dirname } from "path";
import { fileURLToPath } from "url";

const __dirname = dirname(fileURLToPath(import.meta.url));
const outDir = resolve(__dirname, "..", "docs", "screenshots");
const baseURL =
  process.argv.includes("--base-url")
    ? process.argv[process.argv.indexOf("--base-url") + 1]
    : "http://localhost:3000";

mkdirSync(outDir, { recursive: true });

// Pages to screenshot (path, filename, optional setup)
const pages = [
  { path: "/", name: "home", title: "Home" },
  { path: "/dashboard", name: "dashboard", title: "Dashboard" },
  { path: "/hypermedia/controls", name: "controls", title: "Controls Gallery" },
  { path: "/demo/inventory", name: "inventory", title: "Inventory Table" },
  { path: "/demo/catalog", name: "catalog", title: "Catalog" },
  { path: "/demo/bulk", name: "bulk", title: "Bulk Operations" },
  { path: "/demo/people", name: "people", title: "People Directory" },
  { path: "/demo/kanban", name: "kanban", title: "Kanban Board" },
  { path: "/demo/approvals", name: "approvals", title: "Approvals" },
  { path: "/demo/feed", name: "feed", title: "Activity Feed" },
  { path: "/demo/settings", name: "settings", title: "Settings" },
  { path: "/demo/vendors", name: "vendors", title: "Vendors & Contacts" },
  { path: "/hypermedia/crud", name: "crud", title: "CRUD" },
  { path: "/hypermedia/interactions", name: "interactions", title: "Interactions" },
  { path: "/hypermedia/state", name: "state", title: "State Patterns" },
  { path: "/hypermedia/components3", name: "components3", title: "Components 3" },
  { path: "/hypermedia/realtime", name: "realtime", title: "Realtime Dashboard" },
  { path: "/demo/logging", name: "logging", title: "Retrospective Logging" },
  { path: "/admin/error-traces", name: "error-traces", title: "Error Traces" },
];

async function waitForApp(url, maxRetries = 30) {
  for (let i = 0; i < maxRetries; i++) {
    try {
      const resp = await fetch(`${url}/health`);
      if (resp.ok) return;
    } catch {
      // not ready yet
    }
    await new Promise((r) => setTimeout(r, 1000));
  }
  throw new Error(`App not reachable at ${url} after ${maxRetries}s`);
}

async function main() {
  console.log(`Waiting for app at ${baseURL}...`);
  await waitForApp(baseURL);
  console.log("App is ready.");

  const browser = await chromium.launch();

  // --- Theme helpers (random DaisyUI theme per screenshot) ---
  const daisyThemes = [
    "light", "dark", "cupcake", "emerald", "corporate", "synthwave",
    "retro", "cyberpunk", "valentine", "garden", "forest", "lofi",
    "pastel", "fantasy", "wireframe", "luxury", "dracula", "cmyk",
    "autumn", "business", "acid", "lemonade", "night", "coffee",
    "winter", "dim", "nord", "sunset", "caramellatte", "abyss", "silk",
  ];

  let lastTheme = null;
  function randomTheme() {
    const choices = lastTheme ? daisyThemes.filter((t) => t !== lastTheme) : daisyThemes;
    const theme = choices[Math.floor(Math.random() * choices.length)];
    lastTheme = theme;
    return theme;
  }

  async function setTheme(page, theme) {
    await page.evaluate((t) => {
      document.documentElement.dataset.theme = t;
    }, theme);
    await page.waitForTimeout(200);
  }

  // --- Static screenshots ---
  const ctx = await browser.newContext({
    viewport: { width: 1280, height: 800 },
    deviceScaleFactor: 2,
  });

  for (const { path, name, title } of pages) {
    const page = await ctx.newPage();
    try {
      const theme = randomTheme();
      console.log(`Capturing ${title} (${path}) [theme: ${theme}]...`);
      await page.goto(`${baseURL}${path}`, { waitUntil: "networkidle" });
      await setTheme(page, theme);

      // Full-page screenshot
      await page.screenshot({
        path: resolve(outDir, `${name}.png`),
        fullPage: true,
      });

      // Viewport-only screenshot (for README cards)
      await page.screenshot({
        path: resolve(outDir, `${name}-viewport.png`),
        fullPage: false,
      });

      console.log(`  -> ${name}.png, ${name}-viewport.png`);
    } catch (err) {
      console.warn(`  WARN: Failed to capture ${title}: ${err.message}`);
    } finally {
      await page.close();
    }
  }

  await ctx.close();

  // --- Error control screenshots (require interactions) ---
  console.log("\nCapturing error control screenshots...");
  const errCtx = await browser.newContext({
    viewport: { width: 1280, height: 800 },
    deviceScaleFactor: 2,
  });

  // Reset DB before error screenshots
  const setupPage = await errCtx.newPage();
  try {
    await setupPage.goto(`${baseURL}/admin/db/reinit`, { waitUntil: "networkidle" });
  } catch { /* POST may not be navigable, use fetch below */ }
  await setupPage.evaluate((url) => fetch(`${url}/admin/db/reinit`, { method: "POST" }), baseURL);
  await setupPage.close();

  // Helper: wait for HTMX requests to finish
  async function waitForHtmx(page, timeout = 5000) {
    await page.waitForFunction(
      () => !document.querySelector(".htmx-request"),
      { timeout },
    ).catch(() => {});
    await page.waitForTimeout(300);
  }

  // Default error controls
  const defaultErrors = [
    {
      name: "error-400-bad-request",
      title: "400 Bad Request",
      action: async (page) => {
        await page.goto(`${baseURL}/demo/repository/tasks/abc`, { waitUntil: "networkidle" });
        await page.waitForTimeout(500);
      },
      fullPage: true,
    },
    {
      name: "error-404-not-found",
      title: "404 Not Found",
      action: async (page) => {
        await page.goto(`${baseURL}/demo/repository/tasks/99999`, { waitUntil: "networkidle" });
        await page.waitForTimeout(500);
      },
      fullPage: true,
    },
  ];

  // Custom error recovery controls (all on the controls page)
  const recoveryErrors = [
    {
      name: "error-recovery-transient",
      title: "Transient Error (Retry)",
      selector: 'button:has-text("Save Record")',
      resultId: "#transient-result",
    },
    {
      name: "error-recovery-validation",
      title: "Validation Error (Fix & Resubmit)",
      selector: 'form[hx-post*="errors/validate"] button[type="submit"]',
      resultId: "#validate-result",
    },
    {
      name: "error-recovery-conflict",
      title: "Conflict Error (Update or Copy)",
      selector: 'button[hx-post*="errors/conflict"]',
      resultId: "#conflict-result",
    },
    {
      name: "error-recovery-stale",
      title: "Stale Data Error (Refresh or Force)",
      selector: 'form[hx-post*="errors/stale"] button[type="submit"]',
      resultId: "#stale-result",
    },
    {
      name: "error-recovery-cascade",
      title: "Cascade Error (Reassign or Force Delete)",
      selector: 'button[hx-delete*="errors/cascade"]',
      resultId: "#cascade-result",
    },
  ];

  // Capture default error screenshots
  for (const { name, title, action, fullPage } of defaultErrors) {
    const page = await errCtx.newPage();
    try {
      const theme = randomTheme();
      console.log(`Capturing ${title} [theme: ${theme}]...`);
      await action(page);
      await setTheme(page, theme);
      await page.screenshot({
        path: resolve(outDir, `${name}.png`),
        fullPage: fullPage ?? false,
      });
      console.log(`  -> ${name}.png`);
    } catch (err) {
      console.warn(`  WARN: Failed to capture ${title}: ${err.message}`);
    } finally {
      await page.close();
    }
  }

  // Capture recovery error screenshots (each needs a fresh controls page)
  for (const { name, title, selector, resultId } of recoveryErrors) {
    const page = await errCtx.newPage();
    try {
      const theme = randomTheme();
      console.log(`Capturing ${title} [theme: ${theme}]...`);
      await page.goto(`${baseURL}/hypermedia/controls`, { waitUntil: "networkidle" });
      await setTheme(page, theme);
      await waitForHtmx(page);
      await page.locator(selector).click();
      await waitForHtmx(page);
      await page.locator(resultId).screenshot({
        path: resolve(outDir, `${name}.png`),
      });
      console.log(`  -> ${name}.png`);
    } catch (err) {
      console.warn(`  WARN: Failed to capture ${title}: ${err.message}`);
    } finally {
      await page.close();
    }
  }

  await errCtx.close();

  // --- Errors page screenshots ---
  console.log("\nCapturing errors page screenshots...");

  const errPageCtx = await browser.newContext({
    viewport: { width: 1280, height: 800 },
    deviceScaleFactor: 2,
  });

  // Reset DB
  const resetPage = await errPageCtx.newPage();
  await resetPage.goto(`${baseURL}/hypermedia/errors`, { waitUntil: "networkidle" });
  await resetPage.evaluate((url) => fetch(`${url}/admin/db/reinit`, { method: "POST" }), baseURL);
  await resetPage.waitForTimeout(500);
  await resetPage.close();

  const errorsPageShots = [
    {
      name: "errors-banner-404",
      title: "Banner: 404 Not Found",
      action: async (page) => {
        await page.locator('button:has-text("Trigger 404")').click();
        await waitForHtmx(page);
      },
    },
    {
      name: "errors-banner-400",
      title: "Banner: 400 Bad Request",
      action: async (page) => {
        await page.locator('button:has-text("Trigger 400")').click();
        await waitForHtmx(page);
      },
    },
    {
      name: "errors-banner-500",
      title: "Banner: 500 Internal Server Error",
      action: async (page) => {
        await page.locator('button:has-text("Trigger 500")').click();
        await waitForHtmx(page);
      },
    },
    {
      name: "errors-banner-403",
      title: "Banner: 403 Forbidden",
      action: async (page) => {
        await page.locator('button:has-text("Trigger 403")').click();
        await waitForHtmx(page);
      },
    },
    {
      name: "errors-inline-validation",
      title: "Inline Form Validation (422)",
      action: async (page) => {
        const form = page.locator('form[hx-post*="errors/form"]');
        await form.locator('input[name="name"]').fill("");
        await form.locator('input[name="email"]').fill("bad-email");
        await form.locator('button[type="submit"]').click();
        await waitForHtmx(page);
      },
    },
    {
      name: "errors-oob-warning",
      title: "OOB Warning + Success",
      action: async (page) => {
        await page.locator('button:has-text("Load with Warning")').click();
        await waitForHtmx(page);
      },
    },
    {
      name: "errors-flaky-retry",
      title: "Flaky Endpoint (Retry)",
      action: async (page) => {
        await page.locator('button:has-text("Call Flaky Endpoint")').click();
        await waitForHtmx(page);
      },
    },
    {
      name: "errors-report-modal",
      title: "Report Issue Modal",
      action: async (page) => {
        await page.locator('button:has-text("Trigger 500")').click();
        await waitForHtmx(page);
        await page.locator('#error-status button:has-text("Report Issue")').click();
        await waitForHtmx(page);
        await page.waitForTimeout(300);
      },
    },
  ];

  for (const { name, title, action } of errorsPageShots) {
    const page = await errPageCtx.newPage();
    try {
      const theme = randomTheme();
      console.log(`Capturing ${title} [theme: ${theme}]...`);
      await page.goto(`${baseURL}/hypermedia/errors`, { waitUntil: "networkidle" });
      await setTheme(page, theme);
      await action(page);
      await page.screenshot({
        path: resolve(outDir, `${name}.png`),
        fullPage: true,
      });
      console.log(`  -> ${name}.png`);
    } catch (err) {
      console.warn(`  WARN: Failed to capture ${title}: ${err.message}`);
    } finally {
      await page.close();
    }
  }

  await errPageCtx.close();

  await browser.close();
  console.log("\nDone. Screenshots saved to docs/screenshots/");
}

main().catch((err) => {
  console.error(err);
  process.exit(1);
});

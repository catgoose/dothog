#!/usr/bin/env node

// Captures screenshots of the running demo app.
// Usage: node scripts/screenshots.mjs [--base-url http://localhost:8080]
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
    : "http://localhost:8080";

mkdirSync(outDir, { recursive: true });

// Pages to screenshot (path, filename, optional setup)
const pages = [
  { path: "/", name: "home", title: "Home" },
  { path: "/hypermedia/controls", name: "controls", title: "Controls Gallery" },
  { path: "/tables/inventory", name: "inventory", title: "Inventory Table" },
  { path: "/tables/catalog", name: "catalog", title: "Catalog" },
  { path: "/hypermedia/realtime", name: "realtime", title: "Realtime Dashboard" },
  { path: "/hypermedia/crud", name: "crud", title: "CRUD" },
  { path: "/hypermedia/interactions", name: "interactions", title: "Interactions" },
  { path: "/hypermedia/state", name: "state", title: "State Patterns" },
  { path: "/tables/bulk", name: "bulk", title: "Bulk Operations" },
  { path: "/hypermedia/components3", name: "components3", title: "Components 3" },
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

  // --- Static screenshots ---
  const ctx = await browser.newContext({
    viewport: { width: 1280, height: 800 },
    deviceScaleFactor: 2,
  });

  for (const { path, name, title } of pages) {
    const page = await ctx.newPage();
    try {
      console.log(`Capturing ${title} (${path})...`);
      await page.goto(`${baseURL}${path}`, { waitUntil: "networkidle" });
      await page.waitForTimeout(500);

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

  await browser.close();
  console.log("\nDone. Screenshots saved to docs/screenshots/");
}

main().catch((err) => {
  console.error(err);
  process.exit(1);
});

#!/usr/bin/env node

// Captures screenshots and interaction recordings of the running demo app.
// Usage: node scripts/screenshots.mjs [--base-url http://localhost:8080]
//
// Requires: npx playwright install chromium

import { chromium } from "playwright";
import { mkdirSync, existsSync } from "fs";
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
];

// Interactions to record as video (converted to GIF in CI)
const recordings = [
  {
    name: "table-filter",
    title: "Table Filtering",
    path: "/tables/inventory",
    async interact(page) {
      // Wait for table to load
      await page.waitForSelector("table tbody tr", { timeout: 5000 });
      await page.waitForTimeout(500);

      // Type in search to trigger filtering
      const search = page.locator('input[name="search"]').first();
      if (await search.isVisible()) {
        await search.click();
        for (const char of "wire") {
          await search.press(char);
          await page.waitForTimeout(200);
        }
        // Wait for HTMX response
        await page.waitForTimeout(1000);

        // Clear and try another search
        await search.fill("");
        await page.waitForTimeout(800);
      }
    },
  },
  {
    name: "controls-demo",
    title: "Controls Interaction",
    path: "/hypermedia/controls",
    async interact(page) {
      await page.waitForTimeout(500);

      // Click a few interactive elements
      const btn = page.locator("button").filter({ hasText: /click|count/i }).first();
      if (await btn.isVisible()) {
        await btn.click();
        await page.waitForTimeout(400);
        await btn.click();
        await page.waitForTimeout(400);
        await btn.click();
        await page.waitForTimeout(600);
      }
    },
  },
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

  // --- Interaction recordings (video -> GIF via ffmpeg) ---
  for (const rec of recordings) {
    const videoDir = resolve(outDir, "videos");
    mkdirSync(videoDir, { recursive: true });

    const recCtx = await browser.newContext({
      viewport: { width: 1280, height: 800 },
      deviceScaleFactor: 2,
      recordVideo: {
        dir: videoDir,
        size: { width: 1280, height: 800 },
      },
    });

    const page = await recCtx.newPage();
    try {
      console.log(`Recording ${rec.title} (${rec.path})...`);
      await page.goto(`${baseURL}${rec.path}`, { waitUntil: "networkidle" });
      await page.waitForTimeout(500);

      await rec.interact(page);

      await page.waitForTimeout(500);
    } catch (err) {
      console.warn(`  WARN: Failed to record ${rec.title}: ${err.message}`);
    } finally {
      await page.close();
    }

    // Playwright saves video after context close
    await recCtx.close();

    // Rename the video file (Playwright uses a random name)
    const { readdirSync, renameSync } = await import("fs");
    const videos = readdirSync(videoDir).filter((f) => f.endsWith(".webm"));
    if (videos.length > 0) {
      const latest = videos[videos.length - 1];
      renameSync(
        resolve(videoDir, latest),
        resolve(videoDir, `${rec.name}.webm`),
      );
      console.log(`  -> videos/${rec.name}.webm`);
    }
  }

  await browser.close();
  console.log("\nDone. Screenshots saved to docs/screenshots/");
  console.log("Run ffmpeg to convert videos to GIFs (see workflow).");
}

main().catch((err) => {
  console.error(err);
  process.exit(1);
});

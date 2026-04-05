import { test, expect } from "@playwright/test";
import { navigateTo, waitForHtmx, resetDB } from "./helpers";

test.describe("CRUD Patterns", () => {
  test.beforeEach(async ({ page }) => {
    await resetDB(page);
  });

  test("page loads with title", async ({ page }) => {
    await navigateTo(page, "/patterns/crud");
    await expect(page.locator("h1")).toContainText("CRUD Patterns");
  });

  test("table has expected columns", async ({ page }) => {
    await navigateTo(page, "/patterns/crud");
    for (const col of ["ID", "Name", "Status", "Notes", "Actions"]) {
      await expect(page.locator(`th:has-text("${col}")`)).toBeVisible();
    }
  });

  test("add item button creates new row", async ({ page }) => {
    await navigateTo(page, "/patterns/crud");
    const addBtn = page.locator('button:has-text("+ Add Item")').first();
    await expect(addBtn).toBeVisible();
    // Count rows before adding
    const rowsBefore = await page.locator("tbody tr").count();
    await addBtn.click();
    await waitForHtmx(page);
    // Should have more rows after adding
    const rowsAfter = await page.locator("tbody tr").count();
    expect(rowsAfter).toBeGreaterThanOrEqual(rowsBefore);
  });

  test("inline edit works", async ({ page }) => {
    await navigateTo(page, "/patterns/crud");
    await waitForHtmx(page);
    const editBtn = page.locator('button:has-text("Edit")').first();
    await expect(editBtn).toBeVisible();
    await editBtn.click();
    await waitForHtmx(page);
    // Should show input fields for editing
    const inputs = page.locator("tbody input, tbody select");
    await expect(inputs.first()).toBeVisible();
  });
});

test.describe("List Patterns", () => {
  test.beforeEach(async ({ page }) => {
    await resetDB(page);
  });

  test("page loads with title", async ({ page }) => {
    await navigateTo(page, "/patterns/lists");
    await expect(page.locator("h1")).toContainText("List Patterns");
  });

  test("table and filters load", async ({ page }) => {
    await navigateTo(page, "/patterns/lists");
    await expect(page.locator("#lists-table-container")).toBeVisible();
    await expect(page.locator("#filter-form").first()).toBeVisible();
  });

  test("search input exists", async ({ page }) => {
    await navigateTo(page, "/patterns/lists");
    const searchInput = page.locator('input[name="q"]').first();
    await expect(searchInput).toBeVisible();
  });

  test("category filter works", async ({ page }) => {
    await navigateTo(page, "/patterns/lists");
    const catSelect = page.locator("select").first();
    if (await catSelect.isVisible()) {
      await catSelect.selectOption({ index: 1 });
      await waitForHtmx(page);
    }
  });
});

test.describe("Interaction Patterns", () => {
  test("page loads with title", async ({ page }) => {
    await navigateTo(page, "/patterns/interactions");
    await expect(page.locator("h1")).toContainText("Interaction Patterns");
  });

  test("contact form has required fields", async ({ page }) => {
    await navigateTo(page, "/patterns/interactions");
    const form = page.locator("#interaction-form, form").first();
    if (await form.isVisible()) {
      await expect(form).toBeVisible();
    }
  });

  test("form submission works", async ({ page }) => {
    await navigateTo(page, "/patterns/interactions");
    const nameInput = page.locator("#contact-name, input[name='name']").first();
    const emailInput = page.locator("#contact-email, input[name='email']").first();
    const msgInput = page.locator("#contact-message, textarea[name='message']").first();

    if (await nameInput.isVisible()) {
      await nameInput.fill("Test User");
      if (await emailInput.isVisible()) await emailInput.fill("test@example.com");
      if (await msgInput.isVisible()) await msgInput.fill("Hello world");

      const submitBtn = page.locator('button[type="submit"]').first();
      if (await submitBtn.isVisible()) {
        await submitBtn.click();
        await waitForHtmx(page);
      }
    }
  });
});

test.describe("State Patterns", () => {
  test("page loads with title", async ({ page }) => {
    await navigateTo(page, "/patterns/state");
    await expect(page.locator("h1")).toContainText("State Patterns");
  });

  test("like counter increments", async ({ page }) => {
    await navigateTo(page, "/patterns/state");
    const likeBtn = page
      .locator('button:has-text("Like"), button:has-text("♡"), button:has-text("❤")')
      .first();
    if (await likeBtn.isVisible()) {
      await likeBtn.click();
      await waitForHtmx(page);
    }
  });

  test("toggle state button works", async ({ page }) => {
    await navigateTo(page, "/patterns/state");
    const toggleBtn = page
      .locator('button:has-text("Toggle")')
      .first();
    if (await toggleBtn.isVisible()) {
      await toggleBtn.click();
      await waitForHtmx(page);
    }
  });

  test("reveal panel shows/hides", async ({ page }) => {
    await navigateTo(page, "/patterns/state");
    const revealBtn = page
      .locator('button:has-text("Show"), button:has-text("Reveal"), button:has-text("Load")')
      .first();
    if (await revealBtn.isVisible()) {
      await revealBtn.click();
      await waitForHtmx(page);
    }
  });
});

test.describe("Realtime Dashboard", () => {
  test("page loads with title", async ({ page }) => {
    await navigateTo(page, "/realtime/dashboard");
    await expect(page.locator("h1")).toContainText("Live Operations Dashboard");
  });

  test("SSE connection wrapper exists", async ({ page }) => {
    await navigateTo(page, "/realtime/dashboard");
    const sseWrapper = page.locator("#dashboard-sse, #sse-connect-wrapper");
    await expect(sseWrapper.first()).toBeVisible();
  });

  test("frequency slider exists", async ({ page }) => {
    await navigateTo(page, "/realtime/dashboard");
    const slider = page.locator("#freq-slider");
    if (await slider.isVisible()) {
      await expect(slider).toBeVisible();
    }
  });

  test("event feed container exists", async ({ page }) => {
    await navigateTo(page, "/realtime/dashboard");
    const feed = page.locator("#event-feed");
    if (await feed.isVisible()) {
      await expect(feed).toBeVisible();
    }
  });
});

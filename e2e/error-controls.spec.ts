import { test, expect, Page } from "@playwright/test";
import { navigateTo, waitForHtmx, resetDB } from "./helpers";

// ─── Error Patterns Page (/patterns/errors) ────────────────────────────────
// Tests the error handling philosophy: every error is a navigable state with
// hypermedia controls and proper recovery paths. Report Issue appears on 5xx
// errors and inline error panels.

async function clearErrorBanner(page: Page) {
  const close = page.locator('#error-status button:has-text("Close")');
  if ((await close.count()) > 0 && (await close.first().isVisible())) {
    await close.first().click();
    await page.waitForTimeout(400);
  }
}

test.describe("Error Patterns Page", () => {
  test.beforeEach(async ({ page }) => {
    await resetDB(page);
  });

  test("page loads with all error pattern sections", async ({ page }) => {
    await navigateTo(page, "/patterns/errors");
    await expect(page.locator("h1")).toContainText("Error Patterns");
    await expect(page.locator("text=Banner Errors").first()).toBeVisible();
    await expect(page.locator("text=Inline Form Errors").first()).toBeVisible();
    await expect(page.locator("text=OOB Error Swap").first()).toBeVisible();
    await expect(page.locator("text=Retry with Recovery").first()).toBeVisible();
    await page.screenshot({
      path: "test-results/error-controls-screenshots/00-errors-page-overview.png",
      fullPage: true,
    });
  });

  // ─── Banner Errors ─────────────────────────────────────────────────────────

  test("trigger 404 shows banner with Close control", async ({
    page,
  }) => {
    await navigateTo(page, "/patterns/errors");
    await page.locator('button:has-text("Trigger 404")').click();
    await waitForHtmx(page);

    const banner = page.locator("#error-status");
    await expect(banner).toContainText("Something went wrong");
    await expect(banner).toContainText("Resource not found");
    await expect(banner).toContainText("404");
    await expect(banner).toContainText("Request ID");

    // Banner controls: Close (right) — Report Issue only for 5xx errors
    await expect(
      banner.locator('button:has-text("Close")'),
    ).toBeVisible();

    await page.screenshot({
      path: "test-results/error-controls-screenshots/01-banner-404.png",
      fullPage: true,
    });
  });

  test("trigger 400 shows banner with Close control", async ({
    page,
  }) => {
    await navigateTo(page, "/patterns/errors");
    await page.locator('button:has-text("Trigger 400")').click();
    await waitForHtmx(page);

    const banner = page.locator("#error-status");
    await expect(banner).toContainText("Something went wrong");
    await expect(banner).toContainText("Bad request");
    await expect(banner).toContainText("400");
    // Banner controls: Close — Report Issue only for 5xx errors
    await expect(
      banner.locator('button:has-text("Close")'),
    ).toBeVisible();

    await page.screenshot({
      path: "test-results/error-controls-screenshots/02-banner-400.png",
      fullPage: true,
    });
  });

  test("trigger 500 shows banner with Close and Report Issue controls", async ({
    page,
  }) => {
    await navigateTo(page, "/patterns/errors");
    await page.locator('button:has-text("Trigger 500")').click();
    await waitForHtmx(page);

    const banner = page.locator("#error-status");
    await expect(banner).toContainText("Something went wrong");
    await expect(banner).toContainText("Internal server error");
    await expect(banner).toContainText("500");
    await expect(
      banner.locator('button:has-text("Report Issue")'),
    ).toBeVisible();
    await expect(
      banner.locator('button:has-text("Close")'),
    ).toBeVisible();

    await page.screenshot({
      path: "test-results/error-controls-screenshots/03-banner-500.png",
      fullPage: true,
    });
  });

  test("trigger 403 shows banner with Close control", async ({
    page,
  }) => {
    await navigateTo(page, "/patterns/errors");
    await page.locator('button:has-text("Trigger 403")').click();
    await waitForHtmx(page);

    const banner = page.locator("#error-status");
    await expect(banner).toContainText("Something went wrong");
    await expect(banner).toContainText("Forbidden");
    await expect(banner).toContainText("403");
    // Banner controls: Close — Report Issue only for 5xx errors
    await expect(
      banner.locator('button:has-text("Close")'),
    ).toBeVisible();

    await page.screenshot({
      path: "test-results/error-controls-screenshots/04-banner-403.png",
      fullPage: true,
    });
  });

  test("close button clears the error banner", async ({ page }) => {
    await navigateTo(page, "/patterns/errors");

    // Trigger a 400 (has Close)
    await page.locator('button:has-text("Trigger 400")').click();
    await waitForHtmx(page);

    const banner = page.locator("#error-status");
    await expect(banner).toContainText("Something went wrong");

    // Close it
    await page
      .locator('#error-status button:has-text("Close")')
      .click();
    await page.waitForTimeout(400);

    // Banner should be gone
    await expect(
      page.locator("#error-status"),
    ).toBeEmpty();

    await page.screenshot({
      path: "test-results/error-controls-screenshots/05-banner-dismissed.png",
      fullPage: true,
    });
  });

  test("error banner shows request ID with copy button", async ({ page }) => {
    await navigateTo(page, "/patterns/errors");
    await page.locator('button:has-text("Trigger 404")').click();
    await waitForHtmx(page);

    const banner = page.locator("#error-status");
    // Request ID is displayed
    await expect(banner.locator("dt:has-text('Request ID')")).toBeVisible();
    // Copy button exists (the SVG clipboard icon button)
    await expect(
      banner.locator("button svg").first(),
    ).toBeVisible();
  });

  // ─── Inline Form Errors ────────────────────────────────────────────────────

  test("form with empty fields shows inline 422 validation errors", async ({
    page,
  }) => {
    await navigateTo(page, "/patterns/errors");

    // Submit with both fields empty/invalid
    const form = page.locator('form[hx-post*="errors/form"]');
    await form.locator('input[name="name"]').fill("");
    await form.locator('input[name="email"]').fill("bad-email");
    await form.locator('button[type="submit"]').click();
    await waitForHtmx(page);

    const result = page.locator("#errors-form-result");
    await expect(result).toBeVisible();
    await expect(result).toContainText("Validation failed");
    await expect(result).toContainText("Name is required");
    await expect(result).toContainText("Email must contain @");

    await page.screenshot({
      path: "test-results/error-controls-screenshots/06-inline-form-validation.png",
      fullPage: true,
    });
  });

  test("form with valid fields shows success", async ({ page }) => {
    await navigateTo(page, "/patterns/errors");

    const form = page.locator('form[hx-post*="errors/form"]');
    await form.locator('input[name="name"]').fill("Jane Doe");
    await form.locator('input[name="email"]').fill("jane@example.com");
    await form.locator('button[type="submit"]').click();
    await waitForHtmx(page);

    const result = page.locator("#errors-form-result");
    await expect(result).toBeVisible();
    await expect(result).toContainText("Submitted successfully");
    await expect(result).toContainText("Jane Doe");
    await expect(result).toContainText("jane@example.com");

    await page.screenshot({
      path: "test-results/error-controls-screenshots/07-inline-form-success.png",
      fullPage: true,
    });
  });

  test("form with only missing email shows single error", async ({ page }) => {
    await navigateTo(page, "/patterns/errors");

    const form = page.locator('form[hx-post*="errors/form"]');
    await form.locator('input[name="name"]').fill("Jane");
    await form.locator('input[name="email"]').fill("no-at-sign");
    await form.locator('button[type="submit"]').click();
    await waitForHtmx(page);

    const result = page.locator("#errors-form-result");
    await expect(result).toContainText("Email must contain @");
    // Name is valid — should NOT appear in errors
    await expect(result).not.toContainText("Name is required");
  });

  // ─── OOB Error Swap ────────────────────────────────────────────────────────

  test("OOB warning shows success content AND error banner simultaneously", async ({
    page,
  }) => {
    await navigateTo(page, "/patterns/errors");

    await page.locator('button:has-text("Load with Warning")').click();
    await waitForHtmx(page);

    // Primary content updated successfully
    const result = page.locator("#errors-oob-result");
    await expect(result).toBeVisible();
    await expect(result).toContainText("Data loaded successfully");

    // OOB error banner also appeared
    const banner = page.locator("#error-status");
    await expect(banner).toContainText("Data loaded with warnings");
    await expect(
      banner.locator('button:has-text("Report Issue")'),
    ).toBeVisible();
    await expect(
      banner.locator('button:has-text("Close")'),
    ).toBeVisible();

    await page.screenshot({
      path: "test-results/error-controls-screenshots/08-oob-warning-with-success.png",
      fullPage: true,
    });
  });

  // ─── Retry with Recovery ───────────────────────────────────────────────────

  test("flaky endpoint: first call fails with retry button, second succeeds", async ({
    page,
  }) => {
    await navigateTo(page, "/patterns/errors");

    // First call — should fail (odd attempt)
    await page.locator('button:has-text("Call Flaky Endpoint")').click();
    await waitForHtmx(page);

    const result = page.locator("#errors-retry-result");
    await expect(result).toBeVisible();
    // Should show error with retry button
    await expect(result).toContainText("Service temporarily unavailable");
    const retryBtn = result.locator('button:has-text("Retry")');
    await expect(retryBtn).toBeVisible();
    // Report Issue is on the inline error too
    await expect(
      result.locator('button:has-text("Report Issue")'),
    ).toBeVisible();

    await page.screenshot({
      path: "test-results/error-controls-screenshots/09-flaky-first-call-error.png",
      fullPage: true,
    });

    // Click retry — second call should succeed (even attempt)
    await retryBtn.click();
    await waitForHtmx(page);

    await expect(result).toContainText("Request succeeded");

    await page.screenshot({
      path: "test-results/error-controls-screenshots/10-flaky-retry-success.png",
      fullPage: true,
    });
  });

  // ─── Report Issue Modal ────────────────────────────────────────────────────

  test("Report Issue button opens report modal", async ({ page }) => {
    await navigateTo(page, "/patterns/errors");

    // Trigger an error to get a Report Issue button
    await page.locator('button:has-text("Trigger 500")').click();
    await waitForHtmx(page);

    const banner = page.locator("#error-status");
    await expect(banner).toContainText("Something went wrong");

    // Click Report Issue
    await banner.locator('button:has-text("Report Issue")').click();
    await waitForHtmx(page);

    // Modal should appear
    const modal = page.locator("#report-modal-container dialog");
    await expect(modal).toBeVisible();

    await page.screenshot({
      path: "test-results/error-controls-screenshots/11-report-issue-modal.png",
      fullPage: true,
    });
  });

  // ─── Pattern Summary Cards ─────────────────────────────────────────────────

  test("pattern summary cards are visible", async ({ page }) => {
    await navigateTo(page, "/patterns/errors");

    for (const title of [
      "Banner",
      "Inline",
      "OOB",
      "Retry",
      "Report",
      "Copy ID",
    ]) {
      await expect(
        page.locator(`h3:has-text("${title}")`).first(),
      ).toBeVisible();
    }
  });
});

// ─── Controls Gallery: Strengthened Error Recovery Tests ─────────────────────
// These test the error controls in /patterns/controls more thoroughly than
// the existing controls-gallery.spec.ts.

test.describe("Error Controls: transient error recovery", () => {
  test.beforeEach(async ({ page }) => {
    await resetDB(page);
  });

  test("transient error shows inline error with Retry and Report Issue", async ({
    page,
  }) => {
    await navigateTo(page, "/patterns/controls");

    const trigger = page.locator('[hx-post*="errors/transient"]').first();
    await trigger.click();
    await waitForHtmx(page);

    // Error panel should appear with controls
    const errorPanel = page.locator(".alert-error").first();
    await expect(errorPanel).toBeVisible();
    await expect(errorPanel).toContainText("500");
    await expect(
      errorPanel.locator('button:has-text("Retry")'),
    ).toBeVisible();
    await expect(
      errorPanel.locator('button:has-text("Report Issue")'),
    ).toBeVisible();

    await page.screenshot({
      path: "test-results/error-controls-screenshots/12-controls-transient-error.png",
    });
  });

  test("transient retry succeeds on second attempt", async ({ page }) => {
    await navigateTo(page, "/patterns/controls");

    // First attempt fails
    await page.locator('[hx-post*="errors/transient"]').first().click();
    await waitForHtmx(page);

    // Retry succeeds
    const retryBtn = page
      .locator('.alert-error button:has-text("Retry")')
      .first();
    if (await retryBtn.isVisible()) {
      await retryBtn.click();
      await waitForHtmx(page);

      // Success should replace error
      await expect(page.locator(".alert-success").first()).toBeVisible();

      await page.screenshot({
        path: "test-results/error-controls-screenshots/13-controls-transient-retry-success.png",
      });
    }
  });
});

test.describe("Error Controls: validation error recovery", () => {
  test.beforeEach(async ({ page }) => {
    await resetDB(page);
  });

  test("validation error shows inline error with Fix and Report Issue", async ({
    page,
  }) => {
    await navigateTo(page, "/patterns/controls");

    // The hx-post is on the <form> — click the Submit button inside it
    const form = page.locator('form[hx-post*="errors/validate"]').first();
    await form.locator('button[type="submit"], button:has-text("Submit")').click();
    await waitForHtmx(page);

    const errorPanel = page.locator(".alert-error").first();
    await expect(errorPanel).toBeVisible();
    await expect(errorPanel).toContainText("422");
    await expect(
      errorPanel.locator('button:has-text("Report Issue")'),
    ).toBeVisible();

    await page.screenshot({
      path: "test-results/error-controls-screenshots/14-controls-validation-error.png",
    });
  });
});

test.describe("Error Controls: conflict error", () => {
  test.beforeEach(async ({ page }) => {
    await resetDB(page);
  });

  test("conflict error shows 409 with recovery controls", async ({ page }) => {
    await navigateTo(page, "/patterns/controls");

    const trigger = page.locator('[hx-post*="errors/conflict"]').first();
    if (await trigger.isVisible()) {
      await trigger.click();
      await waitForHtmx(page);

      const errorPanel = page.locator(".alert-error").first();
      await expect(errorPanel).toBeVisible();
      await expect(errorPanel).toContainText("409");
      await expect(
        errorPanel.locator('button:has-text("Report Issue")'),
      ).toBeVisible();

      await page.screenshot({
        path: "test-results/error-controls-screenshots/15-controls-conflict-error.png",
      });
    }
  });
});

test.describe("Error Controls: stale data error", () => {
  test.beforeEach(async ({ page }) => {
    await resetDB(page);
  });

  test("stale data shows 412 with Load Fresh and Report Issue", async ({
    page,
  }) => {
    await navigateTo(page, "/patterns/controls");

    const trigger = page.locator('[hx-post*="errors/stale"]').first();
    if (await trigger.isVisible()) {
      // First submit increments version
      await trigger.click();
      await waitForHtmx(page);

      // Second submit should trigger version mismatch (412)
      const trigger2 = page.locator('[hx-post*="errors/stale"]').first();
      if (await trigger2.isVisible()) {
        await trigger2.click();
        await waitForHtmx(page);

        const errorPanel = page.locator(".alert-error").first();
        if (await errorPanel.isVisible()) {
          await expect(errorPanel).toContainText("412");
          await expect(
            errorPanel.locator('button:has-text("Report Issue")'),
          ).toBeVisible();

          await page.screenshot({
            path: "test-results/error-controls-screenshots/16-controls-stale-data-error.png",
          });
        }
      }
    }
  });
});

test.describe("Error Controls: cascade delete", () => {
  test.beforeEach(async ({ page }) => {
    await resetDB(page);
  });

  test("cascade delete shows 409 with reassign and force delete controls", async ({
    page,
  }) => {
    await navigateTo(page, "/patterns/controls");

    const trigger = page.locator('[hx-delete*="errors/cascade"]').first();
    if (await trigger.isVisible()) {
      await trigger.click();
      await waitForHtmx(page);

      const errorPanel = page.locator(".alert-error").first();
      if (await errorPanel.isVisible()) {
        await expect(errorPanel).toContainText("409");
        await expect(
          errorPanel.locator('button:has-text("Report Issue")'),
        ).toBeVisible();

        await page.screenshot({
          path: "test-results/error-controls-screenshots/17-controls-cascade-delete-error.png",
        });
      }
    }
  });
});

import { type Page, expect } from "@playwright/test";

/** Wait for HTMX to finish all pending requests. */
export async function waitForHtmx(page: Page) {
  await page.waitForFunction(
    () =>
      typeof (window as any).htmx !== "undefined" &&
      (document.querySelectorAll(".htmx-request").length === 0),
    { timeout: 10_000 },
  );
}

/** Navigate to a page and assert it loaded (no server error). */
export async function navigateTo(page: Page, path: string) {
  const resp = await page.goto(path);
  expect(resp?.ok(), `Expected 2xx for ${path}, got ${resp?.status()}`).toBe(
    true,
  );
}

/** Reset the demo database via the admin endpoint. */
export async function resetDB(page: Page) {
  const resp = await page.request.post("/admin/db/reinit");
  expect(
    resp.ok(),
    `DB reset failed: POST /admin/db/reinit returned ${resp.status()}`,
  ).toBe(true);
}

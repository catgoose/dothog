import { test, expect } from "@playwright/test";
import { resetDB } from "./helpers";

// GET stays safe: state changes require an unsafe method. The demo error and
// logging triggers, resource restore, and identity switch are POST-only, so a
// GET to those paths must not perform the action.

test.describe("Safety: unsafe actions reject GET", () => {
  test("error trigger is POST-only", async ({ request }) => {
    const get = await request.get("/patterns/errors/trigger/500");
    expect(
      get.status(),
      "GET must not trigger the server error",
    ).toBeGreaterThanOrEqual(400);
    const post = await request.post("/patterns/errors/trigger/500");
    expect(post.status()).toBe(500);
  });

  test("logging trigger is POST-only", async ({ request }) => {
    const get = await request.get("/platform/logging/trigger/500");
    expect(get.status()).toBeGreaterThanOrEqual(400);
  });

  test("resource restore is POST-only", async ({ request }) => {
    const get = await request.get("/patterns/controls/resource/restore");
    expect(get.status()).toBeGreaterThanOrEqual(400);
  });

  test("notification identity switch is POST-only", async ({ request }) => {
    const get = await request.get("/realtime/notifications/identity");
    expect(get.status()).toBeGreaterThanOrEqual(400);
  });
});

test.describe("Safety: GET resource view does not mutate", () => {
  test.beforeEach(async ({ page }) => {
    await resetDB(page);
  });

  test("deleting then re-viewing the resource keeps it deleted", async ({
    request,
  }) => {
    await request.delete("/patterns/controls/resource");
    // A safe GET must not auto-restore the deleted resource.
    const view = await request.get("/patterns/controls/resource");
    expect(view.ok()).toBe(true);
    const deleted = await view.text();
    expect(deleted).toContain("deleted");
    expect(deleted).not.toContain("server-driven UI component");
    // Restore is the explicit unsafe action.
    const restore = await request.post("/patterns/controls/resource/restore");
    expect(restore.ok()).toBe(true);
    expect(await restore.text()).toContain("server-driven UI component");
  });
});

import { defineConfig } from "@playwright/test";

// One source of truth for the e2e port: baseURL, the health URL, and the
// server's SERVER_LISTEN_PORT all derive from it (no hardcoded duplicate).
const PORT = Number(process.env.E2E_PORT) || 3000;
const baseURL = `http://localhost:${PORT}`;

export default defineConfig({
  testDir: ".",
  timeout: 30_000,
  // Retry only on CI; locally a retry-masked pass hides flake.
  retries: process.env.CI ? 2 : 0,
  workers: 1,
  use: {
    baseURL,
    headless: true,
    trace: "on-first-retry",
  },
  projects: [
    {
      name: "chromium",
      use: { browserName: "chromium" },
    },
  ],
  webServer: {
    command: process.env.CI
      ? `./${process.env.APP_BINARY || "dothog"} --env=test`
      : "go run main.go --env=test",
    cwd: "..",
    url: `${baseURL}/health`,
    reuseExistingServer: !process.env.CI,
    timeout: 60_000,
    // --env=test has no tracked .env.test (the *.test gitignore pattern blocks
    // it), so the server falls back to these OS env values instead of a
    // checked-in secrets file. Test-only values; never production secrets.
    env: {
      SERVER_LISTEN_PORT: String(PORT),
      APP_NAME: "dothog",
      SESSION_SETTINGS_COOKIE_NAME: "dothog_e2e_session_id",
      SESSION_SECRET: "e2e-test-only-not-a-real-secret",
      LOG_LEVEL: "warn",
    },
  },
});

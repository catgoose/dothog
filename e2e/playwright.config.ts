import { defineConfig } from "@playwright/test";

export default defineConfig({
  testDir: ".",
  timeout: 30_000,
  retries: 2,
  workers: 1,
  use: {
    baseURL: "http://localhost:3000",
    headless: true,
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
    url: "http://localhost:3000/health",
    reuseExistingServer: !process.env.CI,
    timeout: 60_000,
  },
});

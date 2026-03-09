import { defineConfig } from "@playwright/test";

export default defineConfig({
  testDir: "./e2e",
  timeout: 30_000,
  retries: 2,
  workers: 1,
  use: {
    baseURL: "http://localhost:8080",
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
      ? "./go-htmx-demo --env=test"
      : "go run main.go --env=test",
    url: "http://localhost:8080/health",
    reuseExistingServer: !process.env.CI,
    timeout: 60_000,
  },
});

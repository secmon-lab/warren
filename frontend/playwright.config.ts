import { defineConfig } from "@playwright/test";

if (!process.env.BASE_URL) {
  throw new Error(
    "BASE_URL is not set. Run e2e tests via ./frontend/scripts/e2e.sh, " +
      "or export BASE_URL=http://127.0.0.1:<port> pointing at a running warren server. " +
      "The previous fallback to http://localhost:8080 was removed to avoid " +
      "accidentally hitting unrelated services on a commonly used port."
  );
}

export default defineConfig({
  testDir: "./e2e/tests",
  fullyParallel: true,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 2 : 0,
  workers: process.env.CI ? 1 : undefined,
  reporter: process.env.CI
    ? [["list"], ["html"], ["github"]]
    : [["list"], ["html"]],
  use: {
    baseURL: process.env.BASE_URL,
    actionTimeout: 10_000,
    trace: "on-first-retry",
    screenshot: "only-on-failure",
    video: "retain-on-failure",
  },
  timeout: 30_000,
  globalTimeout: process.env.CI ? 600_000 : 120_000,
  expect: {
    timeout: 5_000,
  },
  projects: [
    {
      name: "chromium",
      use: {
        browserName: "chromium",
      },
    },
  ],
});

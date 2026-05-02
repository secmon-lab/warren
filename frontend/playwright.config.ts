import { defineConfig } from "@playwright/test";

// Note: we intentionally do NOT throw at module load when BASE_URL is unset.
// The Playwright CLI loads this config for non-execution operations too
// (e.g. `playwright test --list`, the VS Code extension's test discovery),
// and a top-level throw would break those flows. Instead we leave `baseURL`
// undefined when BASE_URL is missing — any `page.goto("/")` then fails with
// a clear relative-URL error, preserving the fail-loud behavior at the
// point where it actually matters (test execution) without silently
// falling back to http://localhost:8080.

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

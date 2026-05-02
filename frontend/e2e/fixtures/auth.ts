import { test as base, expect, type Page } from "@playwright/test";

export const test = base.extend<{ authenticatedPage: Page }>({
  authenticatedPage: async ({ page }, use) => {
    // With --no-authn the server returns an anonymous user for /api/auth/me,
    // and AuthGuard then renders the app shell (which includes <header>).
    // Wait for that header to appear instead of using networkidle, which is
    // forbidden by the project's frontend rules and unreliable when WebSocket
    // or polling traffic is active.
    await page.goto("/");
    await expect(page.getByRole("banner")).toBeVisible();
    await use(page);
  },
});

export { expect } from "@playwright/test";

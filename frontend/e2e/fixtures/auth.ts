import { test as base, type Page } from "@playwright/test";

export const test = base.extend<{ authenticatedPage: Page }>({
  authenticatedPage: async ({ page }, use) => {
    // With --no-authn, the server returns an anonymous user for /api/auth/me.
    // Simply navigate to root so the AuthProvider loads and marks us as authenticated.
    await page.goto("/");
    await page.waitForLoadState("networkidle");
    await use(page);
  },
});

export { expect } from "@playwright/test";

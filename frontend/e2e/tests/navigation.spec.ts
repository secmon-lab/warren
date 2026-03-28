import { test, expect } from "../fixtures";
import { DashboardPage } from "../pages/DashboardPage";

test.describe("Navigation", () => {
  test("should display dashboard on initial load", async ({
    authenticatedPage: page,
  }) => {
    const dashboard = new DashboardPage(page);
    await expect(dashboard.heading).toBeVisible();
    await expect(dashboard.openTicketsCard).toBeVisible();
    await expect(dashboard.newAlertsCard).toBeVisible();
  });

  test("should navigate to all sidebar pages", async ({
    authenticatedPage: page,
  }) => {
    const dashboard = new DashboardPage(page);

    // Navigate through each sidebar page
    const pages = [
      { target: "tickets" as const, url: /\/tickets$/, heading: "Tickets" },
      { target: "alerts" as const, url: /\/alerts$/, heading: "Alerts" },
      {
        target: "queue" as const,
        url: /\/queue$/,
        heading: "Queued Alerts",
      },
      {
        target: "knowledge" as const,
        url: /\/knowledge$/,
        heading: "Knowledge",
      },
      { target: "settings" as const, url: /\/settings$/, heading: "Settings" },
    ];

    for (const p of pages) {
      await dashboard.navigateTo(p.target);
      await expect(page).toHaveURL(p.url);
      await expect(
        page.getByRole("heading", { name: p.heading })
      ).toBeVisible();
    }
  });

  test("should show 404 for unknown routes", async ({
    authenticatedPage: page,
  }) => {
    await page.goto("/this-page-does-not-exist");
    await page.waitForLoadState("networkidle");
    await expect(page.getByText("Page not found")).toBeVisible();
  });
});

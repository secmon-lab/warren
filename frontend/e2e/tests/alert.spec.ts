import { test, expect } from "../fixtures";
import { AlertListPage } from "../pages/AlertListPage";

test.describe("Alerts", () => {
  test("should display alerts page with empty state", async ({
    authenticatedPage: page,
  }) => {
    const alertList = new AlertListPage(page);
    await alertList.goto();
    await expect(alertList.heading).toBeVisible();
    await expect(alertList.noAlertsMessage).toBeVisible();
  });

  test("should show status filter tabs", async ({
    authenticatedPage: page,
  }) => {
    const alertList = new AlertListPage(page);
    await alertList.goto();

    await expect(alertList.newTab).toBeVisible();
    await expect(alertList.declinedTab).toBeVisible();
  });

  test("should switch between new and declined tabs", async ({
    authenticatedPage: page,
  }) => {
    const alertList = new AlertListPage(page);
    await alertList.goto();

    // Click Declined tab
    await alertList.declinedTab.click();
    await page.waitForLoadState("networkidle");
    await expect(alertList.noAlertsMessage).toBeVisible();

    // Click New tab
    await alertList.newTab.click();
    await page.waitForLoadState("networkidle");
    await expect(alertList.noAlertsMessage).toBeVisible();
  });
});

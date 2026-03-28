import { test, expect } from "../fixtures";
import { QueuedAlertListPage } from "../pages/QueuedAlertListPage";

test.describe("Queued Alerts", () => {
  test("should display queued alerts page with empty state", async ({
    authenticatedPage: page,
  }) => {
    const queuePage = new QueuedAlertListPage(page);
    await queuePage.goto();
    await expect(queuePage.heading).toBeVisible();
    await expect(queuePage.noQueuedAlertsMessage).toBeVisible();
  });

  test("should have search functionality", async ({
    authenticatedPage: page,
  }) => {
    const queuePage = new QueuedAlertListPage(page);
    await queuePage.goto();

    await expect(queuePage.searchInput).toBeVisible();
    await expect(queuePage.searchButton).toBeVisible();

    // Search with empty result
    await queuePage.searchInput.fill("nonexistent");
    await queuePage.searchButton.click();
    await expect(queuePage.noQueuedAlertsMessage).toBeVisible();
  });
});

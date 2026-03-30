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

  test("should not show bulk action buttons when no alerts exist", async ({
    authenticatedPage: page,
  }) => {
    const queuePage = new QueuedAlertListPage(page);
    await queuePage.goto();

    await expect(queuePage.discardAllButton).not.toBeVisible();
    await expect(queuePage.reprocessAllButton).not.toBeVisible();
  });

  test("should not have checkboxes (old UI removed)", async ({
    authenticatedPage: page,
  }) => {
    const queuePage = new QueuedAlertListPage(page);
    await queuePage.goto();

    // Verify no checkboxes exist on the page
    const checkboxes = page.getByRole("checkbox");
    await expect(checkboxes).toHaveCount(0);

    // Verify "Select all on this page" text is gone
    await expect(page.getByText("Select all on this page")).not.toBeVisible();
  });
});

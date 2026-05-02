import { test, expect } from "../fixtures";
import { TicketListPage } from "../pages/TicketListPage";
import {
  createTicketViaAPI,
  resolveTicketViaAPI,
} from "../helpers/api";

test.describe("Tickets", () => {
  test("should display empty ticket list", async ({
    authenticatedPage: page,
  }) => {
    const ticketList = new TicketListPage(page);
    await ticketList.goto();
    await expect(ticketList.heading).toBeVisible();
    await expect(ticketList.noTicketsMessage).toBeVisible();
  });

  test("should show new ticket button", async ({
    authenticatedPage: page,
  }) => {
    const ticketList = new TicketListPage(page);
    await ticketList.goto();
    await expect(ticketList.newTicketButton).toBeVisible();
  });

  test("should open create ticket modal", async ({
    authenticatedPage: page,
  }) => {
    const ticketList = new TicketListPage(page);
    await ticketList.goto();

    await ticketList.newTicketButton.click();
    await expect(page.getByText("Create New Ticket")).toBeVisible();
    await expect(page.getByLabel("Title")).toBeVisible();
    await expect(page.getByLabel("Description")).toBeVisible();
    await expect(
      page.getByRole("button", { name: "Create" })
    ).toBeVisible();
    await expect(
      page.getByRole("button", { name: "Cancel" })
    ).toBeVisible();

    // Close modal
    await page.getByRole("button", { name: "Cancel" }).click();
    await expect(page.getByText("Create New Ticket")).not.toBeVisible();
  });

  test("should display active and archived tabs", async ({
    authenticatedPage: page,
  }) => {
    const ticketList = new TicketListPage(page);
    await ticketList.goto();

    await expect(ticketList.activeTab).toBeVisible();
    await expect(ticketList.archivedTab).toBeVisible();
  });

  test("should switch between active and archived tabs", async ({
    authenticatedPage: page,
  }) => {
    const ticketList = new TicketListPage(page);
    await ticketList.goto();

    // Switch to archived tab
    await ticketList.archivedTab.click();
    await page.waitForLoadState("networkidle");
    await expect(ticketList.noTicketsMessage).toBeVisible();

    // Switch back to active tab
    await ticketList.activeTab.click();
    await page.waitForLoadState("networkidle");
    await expect(ticketList.noTicketsMessage).toBeVisible();
  });

  test("should have search functionality", async ({
    authenticatedPage: page,
  }) => {
    const ticketList = new TicketListPage(page);
    await ticketList.goto();

    await expect(ticketList.searchInput).toBeVisible();
    await expect(ticketList.searchButton).toBeVisible();

    // Search with empty result
    await ticketList.searchInput.fill("nonexistent");
    await ticketList.searchButton.click();
    await page.waitForLoadState("networkidle");
    await expect(ticketList.noTicketsMessage).toBeVisible();
  });

  test("should show archive all resolved button and confirmation dialog", async ({
    authenticatedPage: page,
  }) => {
    const suffix = Date.now().toString();

    // Create resolved tickets via API
    const ticket1 = await createTicketViaAPI(
      page,
      `ArchDialogTest1-${suffix}`,
      "Test ticket for archive"
    );
    await resolveTicketViaAPI(page, ticket1.id);

    const ticket2 = await createTicketViaAPI(
      page,
      `ArchDialogTest2-${suffix}`,
      "Test ticket for archive"
    );
    await resolveTicketViaAPI(page, ticket2.id);

    const ticketList = new TicketListPage(page);
    await ticketList.goto();

    // Archive All Resolved button should be visible when there are resolved tickets
    await expect(ticketList.archiveAllResolvedButton).toBeVisible();

    // Click the button and verify confirmation dialog appears with count
    await ticketList.archiveAllResolvedButton.click();
    await expect(
      page.getByText(/Archive \d+ resolved tickets?\? This cannot be undone easily\./)
    ).toBeVisible();

    // Cancel
    await page.getByRole("button", { name: "Cancel" }).click();
    await expect(
      page.getByText(/Archive \d+ resolved tickets?\?/)
    ).not.toBeVisible();
  });

  test("should archive all resolved tickets via confirmation dialog", async ({
    authenticatedPage: page,
  }) => {
    const suffix = Date.now().toString();

    // Create a mix of tickets with unique names
    const resolved1 = await createTicketViaAPI(
      page,
      `ToArchive-${suffix}`,
      "Will be archived"
    );
    await resolveTicketViaAPI(page, resolved1.id);

    const openTicket = await createTicketViaAPI(
      page,
      `StayOpen-${suffix}`,
      "Should remain open"
    );

    const ticketList = new TicketListPage(page);
    await ticketList.goto();

    await expect(ticketList.archiveAllResolvedButton).toBeVisible();

    // Click archive button — dialog opens
    await ticketList.archiveAllResolvedButton.click();

    // Wait for dialog and click the confirm button inside it.
    // The trigger ("Archive All Resolved") on the page also matches the
    // substring "Archive", so we scope the lookup to the alertdialog to
    // disambiguate and avoid the overlay-intercepts-click race.
    const confirmButton = page
      .getByRole("alertdialog")
      .getByRole("button", { name: "Archive", exact: true });
    await expect(confirmButton).toBeVisible();
    await confirmButton.click();

    // Wait for refetch to complete
    await page.waitForTimeout(1000);
    await ticketList.goto();

    // After archiving, the resolved ticket should no longer be in the active list
    await expect(page.getByText(`ToArchive-${suffix}`)).not.toBeVisible();
    // Open ticket should still be visible
    await expect(page.getByText(`StayOpen-${suffix}`).first()).toBeVisible();
  });
});

import { test, expect } from "../fixtures";
import { TicketListPage } from "../pages/TicketListPage";

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
});

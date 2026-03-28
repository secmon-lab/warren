import { type Page } from "@playwright/test";
import { BasePage } from "./BasePage";

export class TicketListPage extends BasePage {
  constructor(page: Page) {
    super(page);
  }

  async goto() {
    await super.goto("/tickets");
  }

  get heading() {
    return this.page.getByRole("heading", { name: "Tickets" });
  }

  get newTicketButton() {
    return this.page.getByRole("button", { name: "New Ticket" });
  }

  get activeTab() {
    return this.page.getByRole("tab", { name: "Active" });
  }

  get archivedTab() {
    return this.page.getByRole("tab", { name: "Archived" });
  }

  get noTicketsMessage() {
    return this.page.getByText("No tickets found");
  }

  get searchInput() {
    return this.page.getByPlaceholder("Search keyword...");
  }

  get searchButton() {
    return this.page.getByRole("button", { name: "Search" });
  }

  ticketItems() {
    return this.page.locator("[data-testid^='ticket-item-']");
  }

  ticketItemByTitle(title: string) {
    return this.page.getByText(title);
  }
}

import { type Page } from "@playwright/test";
import { BasePage } from "./BasePage";

export class DashboardPage extends BasePage {
  constructor(page: Page) {
    super(page);
  }

  async goto() {
    await super.goto("/");
  }

  get heading() {
    return this.page.getByRole("heading", { name: "Dashboard" });
  }

  get openTicketsCard() {
    return this.page.getByText("Open Tickets", { exact: true });
  }

  get newAlertsCard() {
    return this.page.getByText("New Alerts", { exact: true });
  }

  get activityFeed() {
    return this.page.getByText("Activity Feed");
  }

  get createTicketButton() {
    return this.page.getByRole("button", { name: "Create Ticket" });
  }

  get viewAllTicketsLink() {
    return this.page.getByRole("link", { name: "View All" }).first();
  }
}

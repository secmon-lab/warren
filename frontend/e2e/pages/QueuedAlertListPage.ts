import { type Page } from "@playwright/test";
import { BasePage } from "./BasePage";

export class QueuedAlertListPage extends BasePage {
  constructor(page: Page) {
    super(page);
  }

  async goto() {
    await super.goto("/queue");
  }

  get heading() {
    return this.page.getByRole("heading", { name: "Queued Alerts" });
  }

  get searchInput() {
    return this.page.getByPlaceholder("Search by keyword...");
  }

  get searchButton() {
    return this.page.getByRole("button", { name: "Search" });
  }

  get noQueuedAlertsMessage() {
    return this.page.getByText("No queued alerts");
  }
}

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

  get alertCount() {
    return this.page.locator("text=/\\d+ queued alerts?/");
  }

  get discardAllButton() {
    return this.page.getByRole("button", { name: /Discard All/ });
  }

  get reprocessAllButton() {
    return this.page.getByRole("button", { name: /Reprocess All/ });
  }

  get confirmDialogTitle() {
    return this.page.getByRole("heading", { level: 2 });
  }

  get confirmDialogDescription() {
    return this.page.locator("[role=alertdialog] p, [role=dialog] p");
  }

  get confirmButton() {
    return this.page.getByRole("button", { name: /Discard All|Reprocess All/ });
  }

  get cancelButton() {
    return this.page.getByRole("button", { name: "Cancel" });
  }
}

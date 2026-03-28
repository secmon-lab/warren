import { type Page } from "@playwright/test";
import { BasePage } from "./BasePage";

export class AlertDetailPage extends BasePage {
  constructor(page: Page) {
    super(page);
  }

  async goto(alertId: string) {
    await super.goto(`/alerts/${alertId}`);
  }

  get backButton() {
    return this.page.getByText("Back to Alerts");
  }

  get alertTitle() {
    return this.page.locator('[data-slot="card-title"]');
  }

  get attributesCard() {
    return this.page.getByText("Attributes");
  }

  get rawDataCard() {
    return this.page.getByText("Raw Data");
  }

  get createTicketButton() {
    return this.page.getByRole("button", { name: "Create Ticket" });
  }
}

import { type Page } from "@playwright/test";
import { BasePage } from "./BasePage";

export class AlertListPage extends BasePage {
  constructor(page: Page) {
    super(page);
  }

  async goto() {
    await super.goto("/alerts");
  }

  get heading() {
    return this.page.getByRole("heading", { name: "Alerts" });
  }

  get newTab() {
    return this.page.getByRole("button", { name: "New" });
  }

  get declinedTab() {
    return this.page.getByRole("button", { name: "Declined" });
  }

  get noAlertsMessage() {
    return this.page.getByText("No alerts found");
  }

  alertCards() {
    return this.page.locator("[data-testid^='alert-card-']");
  }
}

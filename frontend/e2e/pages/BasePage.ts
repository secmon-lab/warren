import { type Page, type Locator, type TestInfo } from "@playwright/test";

export class BasePage {
  readonly page: Page;
  readonly testInfo?: TestInfo;

  constructor(page: Page, testInfo?: TestInfo) {
    this.page = page;
    this.testInfo = testInfo;
  }

  async goto(path: string) {
    await this.page.goto(path);
    await this.page.waitForLoadState("networkidle");
  }

  async waitForNavigation() {
    await this.page.waitForLoadState("networkidle");
  }

  async screenshot(name: string) {
    const path = this.testInfo
      ? this.testInfo.outputPath(`${name}.png`)
      : `test-results/${name}.png`;
    await this.page.screenshot({ path });
  }

  async waitForVisible(locator: Locator) {
    await locator.waitFor({ state: "visible" });
  }

  async waitForHidden(locator: Locator) {
    await locator.waitFor({ state: "hidden" });
  }

  // Sidebar navigation
  get sidebar() {
    return {
      dashboard: this.page.getByRole("link", { name: "Dashboard" }),
      tickets: this.page.getByRole("link", { name: "Tickets" }),
      alerts: this.page.getByRole("link", { name: "Alerts" }),
      queue: this.page.getByRole("link", { name: "Queue" }),
      knowledge: this.page.getByRole("link", { name: "Knowledge" }),
      memory: this.page.getByRole("link", { name: "Memory" }),
      diagnosis: this.page.getByRole("link", { name: "Diagnosis" }),
      settings: this.page.getByRole("link", { name: "Settings" }),
    };
  }

  async navigateTo(
    target:
      | "dashboard"
      | "tickets"
      | "alerts"
      | "queue"
      | "knowledge"
      | "memory"
      | "diagnosis"
      | "settings"
  ) {
    await this.sidebar[target].click();
    await this.waitForNavigation();
  }
}

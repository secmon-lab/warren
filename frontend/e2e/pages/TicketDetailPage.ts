import { type Page } from "@playwright/test";
import { BasePage } from "./BasePage";

export class TicketDetailPage extends BasePage {
  constructor(page: Page) {
    super(page);
  }

  async goto(ticketId: string) {
    await super.goto(`/tickets/${ticketId}`);
  }

  get backButton() {
    return this.page.getByText("Back to Tickets");
  }

  get ticketTitle() {
    return this.page.getByRole("heading", { level: 1 });
  }

  get editButton() {
    return this.page.getByRole("button", { name: "Edit" });
  }

  get resolveButton() {
    return this.page.getByRole("button", { name: "Resolve" });
  }

  get archiveButton() {
    return this.page.getByRole("button", { name: "Archive" });
  }

  // chat-session-redesign Phase 6: Conversation card selectors.
  get conversationHeading() {
    return this.page.getByRole("heading", { name: /^Conversation/ });
  }

  get conversationSlackToggle() {
    return this.page.getByRole("button", { name: "Slack", exact: true });
  }

  get conversationWebToggle() {
    return this.page.getByRole("button", { name: "Web", exact: true });
  }

  get conversationCliToggle() {
    return this.page.getByRole("button", { name: "CLI", exact: true });
  }

  get conversationAllSessionsLink() {
    return this.page.getByRole("button", { name: "All sessions" });
  }
}

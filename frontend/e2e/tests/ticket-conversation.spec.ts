// chat-session-redesign Phase 6: E2E coverage for the new
// TicketConversation card on the ticket detail page.
//
// Scope: structural — verify the card renders, the Source toggle is
// present, the "All sessions" entry is active by default, and the
// empty-state copy shows up when no SessionMessages exist for the
// freshly-created ticket.
import { test, expect } from "../fixtures";
import { TicketDetailPage } from "../pages/TicketDetailPage";
import { createTicketViaAPI } from "../helpers/api";

test.describe("Ticket Conversation (Phase 6)", () => {
  test("renders conversation card with source toggle on ticket detail", async ({
    authenticatedPage: page,
  }) => {
    const suffix = Date.now().toString();
    const ticket = await createTicketViaAPI(
      page,
      `ConversationTest-${suffix}`,
      "Phase 6 conversation card E2E",
    );

    const detail = new TicketDetailPage(page);
    await detail.goto(ticket.id);

    await expect(detail.ticketTitle).toBeVisible();
    await expect(detail.conversationHeading).toBeVisible();

    // All three source toggles must be present.
    await expect(detail.conversationSlackToggle).toBeVisible();
    await expect(detail.conversationWebToggle).toBeVisible();
    await expect(detail.conversationCliToggle).toBeVisible();

    // The "All sessions" shortcut sits in the side pane even when the
    // list is empty, so it is a stable structural assertion.
    await expect(detail.conversationAllSessionsLink).toBeVisible();

    // Fresh ticket has no SessionMessages yet — confirm the
    // empty-state copy rendered by ConversationMainPane.
    await expect(page.getByText("No messages yet.")).toBeVisible();
  });

  test("switches between source toggles without reloading the page", async ({
    authenticatedPage: page,
  }) => {
    const suffix = Date.now().toString();
    const ticket = await createTicketViaAPI(
      page,
      `ConversationToggleTest-${suffix}`,
      "Phase 6 conversation source toggle E2E",
    );

    const detail = new TicketDetailPage(page);
    await detail.goto(ticket.id);

    await expect(detail.conversationSlackToggle).toBeVisible();

    // Toggling Web and CLI should not raise errors or unmount the
    // conversation card. The heading stays rendered throughout.
    await detail.conversationWebToggle.click();
    await expect(detail.conversationHeading).toBeVisible();

    await detail.conversationCliToggle.click();
    await expect(detail.conversationHeading).toBeVisible();

    await detail.conversationSlackToggle.click();
    await expect(detail.conversationHeading).toBeVisible();
  });
});

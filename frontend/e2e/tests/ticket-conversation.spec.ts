// chat-session-redesign Phase 6 (revised): Conversation card on the
// ticket detail page is the unified Slack/Web/CLI surface. Each ticket
// detail page renders one card with the sessions sidebar + message
// pane. Tests exercise structural invariants only — actual WebSocket
// streaming and Turn creation are covered by Go integration tests.
import { test, expect } from "../fixtures";
import { TicketDetailPage } from "../pages/TicketDetailPage";
import {
  archiveTicketViaAPI,
  createTicketViaAPI,
  resolveTicketViaAPI,
} from "../helpers/api";

test.describe("Ticket Conversation (Phase 6)", () => {
  test("renders unified conversation card with sidebar + New Chat", async ({
    authenticatedPage: page,
  }) => {
    const suffix = Date.now().toString();
    const ticket = await createTicketViaAPI(
      page,
      `ConversationTest-${suffix}`,
      "Phase 6 unified conversation E2E",
    );

    try {
      const detail = new TicketDetailPage(page);
      await detail.goto(ticket.id);

      await expect(detail.ticketTitle).toBeVisible();
      await expect(detail.conversationHeading).toBeVisible();

      // The "New Chat" sidebar action is always present.
      await expect(detail.conversationNewChatButton).toBeVisible();

      // A freshly created ticket has no Slack/Web/CLI sessions yet —
      // the sidebar renders the empty-state message.
      await expect(page.getByText("No sessions yet.")).toBeVisible();
    } finally {
      // Always remove the test ticket from the active list so
      // subsequent spec files (e.g. ticket.spec) that assume an
      // empty active-ticket list still pass. E2E tests share the
      // same Firestore emulator across files, so cleanup has to be
      // explicit. archiveTicket rejects open tickets, so resolve
      // first — both calls are non-fatal on cleanup.
      try {
        await resolveTicketViaAPI(page, ticket.id);
        await archiveTicketViaAPI(page, ticket.id);
      } catch {
        // Cleanup failures should not mask a real test failure; the
        // next run will see stale tickets and flag them via the
        // empty-list assertions in ticket.spec.
      }
    }
  });
});

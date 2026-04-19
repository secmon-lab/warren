// chat-session-redesign Phase 6 (revised): Conversation card on the
// ticket detail page is the unified Slack/Web/CLI surface. Each ticket
// detail page renders one card with the sessions sidebar + message
// pane. Tests exercise structural invariants only — actual WebSocket
// streaming and Turn creation are covered by Go integration tests.
//
// File name is prefixed `zzz_` deliberately: Playwright's CI config
// uses `workers: 1` and walks the test dir alphabetically, so this
// suite must sort AFTER ticket.spec.ts to avoid leaving a stray
// ticket behind that breaks ticket.spec's empty-list assertions.
// Warren has no hard-delete mutation — archive-then-switch-tabs
// would pollute the archived tab instead — so "run last" is the
// least bad option until a dedicated test-data API exists.
import { test, expect } from "../fixtures";
import { TicketDetailPage } from "../pages/TicketDetailPage";
import { createTicketViaAPI } from "../helpers/api";

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

    const detail = new TicketDetailPage(page);
    await detail.goto(ticket.id);

    await expect(detail.ticketTitle).toBeVisible();
    await expect(detail.conversationHeading).toBeVisible();

    // The "New Chat" sidebar action is always present.
    await expect(detail.conversationNewChatButton).toBeVisible();

    // A freshly created ticket has no Slack/Web/CLI sessions yet —
    // the sidebar renders the empty-state message.
    await expect(page.getByText("No sessions yet.")).toBeVisible();
  });
});

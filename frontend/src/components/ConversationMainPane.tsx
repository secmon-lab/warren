import { SessionMessage } from "@/lib/types";
import { TurnCard } from "./TurnCard";

interface ConversationMainPaneProps {
  messages: SessionMessage[];
  loading?: boolean;
}

// chat-session-redesign Phase 6: main-pane timeline. Messages from the
// backend arrive sorted by CreatedAt ASC (see
// pkg/repository/firestore/session_v2.go:GetTicketSessionMessages), so
// adjacent-grouping by TurnID keeps a single req/res cycle cohesive.
// Messages without a TurnID are rendered as individual groups rather
// than one lumped-together block — two consecutive nil-TurnID posts
// from different sessions are logically distinct and collapsing them
// would hide that.
export function ConversationMainPane({ messages, loading }: ConversationMainPaneProps) {
  if (loading && messages.length === 0) {
    return (
      <div className="text-sm text-muted-foreground text-center py-6">
        Loading messages…
      </div>
    );
  }
  if (messages.length === 0) {
    return (
      <div className="text-sm text-muted-foreground text-center py-6">
        No messages yet.
      </div>
    );
  }

  const groups = groupByTurn(messages);
  return (
    <div className="space-y-4">
      {groups.map((g) => (
        <TurnCard key={g.key} turnID={g.turnID} messages={g.messages} />
      ))}
    </div>
  );
}

type TurnGroup = {
  // key is guaranteed unique per rendered group (turn-{id} for
  // TurnID-bound groups, untagged-{sessionID}-{anchorMessageID} for
  // TurnID-less groups) so React's reconciliation stays stable even
  // when multiple sessions contribute orphan messages.
  key: string;
  turnID: string | null;
  messages: SessionMessage[];
};

function groupByTurn(messages: SessionMessage[]): TurnGroup[] {
  const groups: TurnGroup[] = [];
  let current: TurnGroup | null = null;
  for (const m of messages) {
    const tid = m.turnID ?? null;
    const sameGroup =
      current !== null &&
      current.turnID === tid &&
      // TurnID-less messages only stay in the same group when they
      // originate from the same Session.
      (tid !== null || current.messages[0].sessionID === m.sessionID);
    if (!sameGroup) {
      const groupKey: string =
        tid !== null ? `turn-${tid}` : `untagged-${m.sessionID}-${m.id}`;
      current = { key: groupKey, turnID: tid, messages: [] };
      groups.push(current);
    }
    current!.messages.push(m);
  }
  return groups;
}

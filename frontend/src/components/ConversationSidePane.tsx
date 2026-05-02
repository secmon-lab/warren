import { SessionMessage } from "@/lib/types";
import { formatRelativeTime } from "@/lib/utils-extended";

interface ConversationSidePaneProps {
  messages: SessionMessage[];
  selectedSessionID: string | null;
  onSelectSession: (sessionID: string | null) => void;
}

// chat-session-redesign Phase 6: side list of Sessions scoped to the
// current Source toggle. Sessions are derived from the same Message
// query on the main pane rather than loaded separately — per spec the
// backend exposes them via one call.
export function ConversationSidePane({
  messages,
  selectedSessionID,
  onSelectSession,
}: ConversationSidePaneProps) {
  const sessions = summarizeSessions(messages);

  return (
    <div className="space-y-1">
      <button
        type="button"
        onClick={() => onSelectSession(null)}
        className={`w-full text-left rounded px-2 py-1 text-sm ${
          selectedSessionID === null
            ? "bg-primary/10 font-medium"
            : "hover:bg-muted"
        }`}>
        All sessions
      </button>
      {sessions.map((s) => (
        <button
          key={s.sessionID}
          type="button"
          onClick={() => onSelectSession(s.sessionID)}
          className={`w-full text-left rounded px-2 py-1 text-sm ${
            selectedSessionID === s.sessionID
              ? "bg-primary/10 font-medium"
              : "hover:bg-muted"
          }`}>
          <div className="truncate">{s.sessionID.slice(0, 12)}…</div>
          <div className="text-xs text-muted-foreground">
            {s.messageCount} msgs · {formatRelativeTime(s.mostRecentAt)}
          </div>
        </button>
      ))}
    </div>
  );
}

type SessionSummary = {
  sessionID: string;
  messageCount: number;
  mostRecentAt: string;
};

function summarizeSessions(messages: SessionMessage[]): SessionSummary[] {
  const bySession = new Map<string, SessionSummary>();
  for (const m of messages) {
    const prev = bySession.get(m.sessionID);
    if (!prev) {
      bySession.set(m.sessionID, {
        sessionID: m.sessionID,
        messageCount: 1,
        mostRecentAt: m.createdAt,
      });
    } else {
      prev.messageCount += 1;
      if (m.createdAt > prev.mostRecentAt) prev.mostRecentAt = m.createdAt;
    }
  }
  return Array.from(bySession.values()).sort((a, b) =>
    b.mostRecentAt.localeCompare(a.mostRecentAt),
  );
}

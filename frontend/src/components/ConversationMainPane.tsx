import { useState } from "react";
import { SessionMessage } from "@/lib/types";
import { MessageBubble } from "./MessageBubble";
import { ChevronDown, ChevronRight, Wrench } from "lucide-react";
import { formatRelativeTime } from "@/lib/utils-extended";

interface ConversationMainPaneProps {
  messages: SessionMessage[];
  loading?: boolean;
}

// chat-session-redesign Phase 6 (revised): timeline groups messages
// by "exchange" rather than by TurnID. An exchange starts at each
// user message and extends until the next user message; AI outputs
// inside the exchange (response / plan / trace / warning) render as
// one block so trace chatter collapses into a single progress panel
// rather than a stream of individually-timestamped bubbles. TurnID
// is intentionally NOT used for grouping — the field was flagged as
// unreliable in review, and showing an incorrect boundary is worse
// than no boundary at all.
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

  const exchanges = groupByExchange(messages);
  return (
    <div className="space-y-6">
      {exchanges.map((ex) => (
        <ExchangeView key={ex.key} exchange={ex} />
      ))}
    </div>
  );
}

type Exchange = {
  key: string;
  user?: SessionMessage;
  traces: SessionMessage[];
  primary: SessionMessage[]; // response / plan / warning
};

function groupByExchange(messages: SessionMessage[]): Exchange[] {
  const out: Exchange[] = [];
  let current: Exchange | null = null;
  const flush = () => {
    if (current && (current.user || current.traces.length || current.primary.length)) {
      out.push(current);
    }
  };
  for (const m of messages) {
    if (m.type === "user") {
      flush();
      current = { key: `ex-${m.id}`, user: m, traces: [], primary: [] };
      continue;
    }
    if (!current) {
      // AI output with no preceding user message (e.g. Slack session
      // pre-migration). Open a standalone exchange.
      current = { key: `ex-orphan-${m.id}`, traces: [], primary: [] };
    }
    if (m.type === "trace") {
      current.traces.push(m);
    } else {
      current.primary.push(m);
    }
  }
  flush();
  return out;
}

function ExchangeView({ exchange }: { exchange: Exchange }) {
  return (
    <div className="space-y-2">
      {exchange.user && <MessageBubble message={exchange.user} />}
      {exchange.traces.length > 0 && <ProgressPanel traces={exchange.traces} />}
      {exchange.primary.map((m) => (
        <MessageBubble key={m.id} message={m} />
      ))}
    </div>
  );
}

// ProgressPanel collapses N trace messages into one collapsible view.
// Mirrors the Slack "updatable context block" UX without rendering
// the per-message timestamps that cluttered the previous design.
function ProgressPanel({ traces }: { traces: SessionMessage[] }) {
  const [open, setOpen] = useState(false);
  const latest = traces[traces.length - 1];
  const latestLabel = latest ? formatRelativeTime(latest.createdAt) : "";
  return (
    <div className="rounded-md border border-slate-200 bg-slate-50">
      <button
        type="button"
        onClick={() => setOpen((v) => !v)}
        className="w-full flex items-center gap-2 px-3 py-2 text-xs text-slate-700 hover:bg-slate-100">
        {open ? (
          <ChevronDown className="h-3.5 w-3.5" />
        ) : (
          <ChevronRight className="h-3.5 w-3.5" />
        )}
        <Wrench className="h-3.5 w-3.5" />
        <span className="font-medium">
          Progress ({traces.length} step{traces.length === 1 ? "" : "s"})
        </span>
        {latestLabel && (
          <span className="ml-auto text-[10px] text-slate-500">
            {latestLabel}
          </span>
        )}
      </button>
      {open && (
        <ul className="px-3 pb-3 space-y-1 text-xs font-mono text-slate-600">
          {traces.map((t) => (
            <li key={t.id} className="whitespace-pre-wrap break-words">
              {t.content}
            </li>
          ))}
        </ul>
      )}
      {!open && latest && (
        <div className="px-9 pb-2 text-[11px] italic text-slate-500 truncate">
          {firstLine(latest.content)}
        </div>
      )}
    </div>
  );
}

function firstLine(s: string): string {
  const i = s.indexOf("\n");
  return i === -1 ? s : s.slice(0, i);
}

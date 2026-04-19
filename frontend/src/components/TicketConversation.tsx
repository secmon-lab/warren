import React, { useEffect, useMemo, useRef, useState } from "react";
import { useQuery } from "@apollo/client";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Textarea } from "@/components/ui/textarea";
import { Badge } from "@/components/ui/badge";
import { MessageSquare, Plus, Send, Wifi, WifiOff, Loader2, AlertCircle } from "lucide-react";
import {
  GET_TICKET_SESSION_MESSAGES,
  GET_TICKET_SESSIONS,
} from "@/lib/graphql/queries";
import { Session, SessionMessage } from "@/lib/types";
import { useWebSocket } from "@/hooks/useWebSocket";
import { formatRelativeTime } from "@/lib/utils-extended";
import { ConversationMainPane } from "./ConversationMainPane";

interface TicketConversationProps {
  ticketId: string;
}

type SelectedSession =
  | { mode: "existing"; sessionID: string; source: string }
  | { mode: "new" }
  | null;

// chat-session-redesign Phase 6 (revised): single unified Conversation
// surface for Slack / Web / CLI. The left sidebar lists every Session
// attached to the ticket with a source badge; the right pane shows
// that Session's message timeline. Web and CLI sessions get a send
// box so users can continue the conversation; Slack sessions are
// read-only (the origin is the Slack thread itself).
//
// No new WebSocket connection is opened on page mount — the backend
// only creates a new Web Session once the user clicks "New Chat" or
// resumes an existing one, so simply visiting a ticket does not
// manufacture empty Sessions.
export function TicketConversation({ ticketId }: TicketConversationProps) {
  const { data: sessionsData, refetch: refetchSessions } = useQuery<{
    ticketSessions: Session[];
  }>(GET_TICKET_SESSIONS, {
    variables: { ticketId },
    fetchPolicy: "cache-and-network",
  });
  const {
    data: messagesData,
    loading: messagesLoading,
    error: messagesError,
    refetch: refetchMessages,
  } = useQuery<{
    ticketSessionMessages: SessionMessage[];
  }>(GET_TICKET_SESSION_MESSAGES, {
    variables: { ticketId, limit: 500, offset: 0 },
    fetchPolicy: "cache-and-network",
  });

  const sessions = useMemo(
    () => sortSessions(sessionsData?.ticketSessions ?? []),
    [sessionsData],
  );
  const allMessages = messagesData?.ticketSessionMessages ?? [];

  const [selected, setSelected] = useState<SelectedSession>(null);

  // Default-select the most recent Slack session once sessions arrive.
  useEffect(() => {
    if (selected !== null) return;
    if (sessions.length === 0) return;
    const slack = sessions.find((s) => s.source === "slack");
    const target = slack ?? sessions[0];
    setSelected({
      mode: "existing",
      sessionID: target.id,
      source: target.source || "",
    });
  }, [sessions, selected]);

  const displayedSession: Session | null = useMemo(() => {
    if (!selected || selected.mode !== "existing") return null;
    return sessions.find((s) => s.id === selected.sessionID) ?? null;
  }, [selected, sessions]);

  const persistedForSelected = useMemo(() => {
    if (!selected || selected.mode === "new") return [];
    return allMessages.filter((m) => m.sessionID === selected.sessionID);
  }, [allMessages, selected]);

  // Open a WebSocket iff the selected Session is Web/CLI OR the user
  // just clicked "New Chat". Slack sessions never get a WS (read-only).
  const wsSessionID = useMemo(() => {
    if (!selected) return undefined;
    if (selected.mode === "new") return undefined;
    if (selected.source === "slack") return undefined;
    return selected.sessionID;
  }, [selected]);
  const wsEnabled =
    selected !== null &&
    (selected.mode === "new" ||
      (selected.mode === "existing" && selected.source !== "slack"));

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <MessageSquare className="h-5 w-5" />
          Conversation ({allMessages.length})
        </CardTitle>
      </CardHeader>
      <CardContent>
        {messagesError && (
          <div className="text-sm text-red-600 py-2">
            Error loading conversation: {messagesError.message}
          </div>
        )}
        <div className="grid grid-cols-1 md:grid-cols-[240px_1fr] gap-4">
          <div className="border-r md:pr-4">
            <SessionSidebar
              sessions={sessions}
              selected={selected}
              onSelect={(s) => setSelected(s)}
            />
          </div>
          <div className="min-w-0">
            {wsEnabled ? (
              <WebChatPane
                ticketId={ticketId}
                sessionIdForResume={wsSessionID}
                persistedMessages={persistedForSelected}
                messagesLoading={messagesLoading}
                onTurnCompleted={() => {
                  refetchSessions();
                  refetchMessages();
                }}
                onSessionCreated={() => {
                  refetchSessions();
                }}
              />
            ) : (
              <ReadOnlyPane
                messages={persistedForSelected}
                loading={messagesLoading}
                session={displayedSession}
              />
            )}
          </div>
        </div>
      </CardContent>
    </Card>
  );
}

// --- sidebar ----------------------------------------------------------

interface SessionSidebarProps {
  sessions: Session[];
  selected: SelectedSession;
  onSelect: (s: SelectedSession) => void;
}

function SessionSidebar({ sessions, selected, onSelect }: SessionSidebarProps) {
  return (
    <div className="space-y-1">
      <button
        type="button"
        onClick={() => onSelect({ mode: "new" })}
        className={`w-full flex items-center gap-2 rounded px-2 py-2 text-sm font-medium ${
          selected?.mode === "new" ? "bg-primary/10" : "hover:bg-muted"
        }`}>
        <Plus className="h-4 w-4" />
        New Chat
      </button>
      <div className="pt-2 pb-1 text-[10px] uppercase tracking-wider text-muted-foreground">
        Sessions
      </div>
      {sessions.length === 0 && (
        <div className="text-xs text-muted-foreground px-2 py-1">
          No sessions yet.
        </div>
      )}
      {sessions.map((s) => (
        <button
          key={s.id}
          type="button"
          onClick={() =>
            onSelect({
              mode: "existing",
              sessionID: s.id,
              source: s.source || "",
            })
          }
          className={`w-full text-left rounded px-2 py-1.5 text-sm ${
            selected?.mode === "existing" && selected.sessionID === s.id
              ? "bg-primary/10 font-medium"
              : "hover:bg-muted"
          }`}>
          <div className="flex items-center gap-2">
            <SourceBadge source={s.source} />
            <span className="truncate flex-1">{s.id.slice(0, 12)}…</span>
          </div>
          <div className="text-xs text-muted-foreground pl-1 mt-0.5">
            {formatRelativeTime(s.updatedAt || s.createdAt)}
          </div>
        </button>
      ))}
    </div>
  );
}

function SourceBadge({ source }: { source: string }) {
  const label = sourceLabel(source);
  const cls = sourceColorClass(source);
  return (
    <span
      className={`inline-flex items-center rounded px-1.5 py-0.5 text-[10px] font-medium uppercase tracking-wide ${cls}`}>
      {label}
    </span>
  );
}

function sourceLabel(source: string): string {
  switch (source) {
    case "slack":
      return "Slack";
    case "web":
      return "Web";
    case "cli":
      return "CLI";
    default:
      return "—";
  }
}

function sourceColorClass(source: string): string {
  switch (source) {
    case "slack":
      return "bg-purple-100 text-purple-800";
    case "web":
      return "bg-blue-100 text-blue-800";
    case "cli":
      return "bg-slate-100 text-slate-800";
    default:
      return "bg-gray-100 text-gray-600";
  }
}

// --- read-only pane (Slack or no selection) ---------------------------

interface ReadOnlyPaneProps {
  messages: SessionMessage[];
  loading: boolean;
  session: Session | null;
}

function ReadOnlyPane({ messages, loading, session }: ReadOnlyPaneProps) {
  return (
    <div className="space-y-3">
      {session?.source === "slack" && session.slackURL && (
        <div className="text-xs text-muted-foreground">
          Slack thread — continue the conversation in{" "}
          <a
            href={session.slackURL}
            target="_blank"
            rel="noreferrer"
            className="underline">
            Slack
          </a>
          .
        </div>
      )}
      <ConversationMainPane messages={messages} loading={loading} />
    </div>
  );
}

// --- interactive Web/CLI pane ----------------------------------------

interface WebChatPaneProps {
  ticketId: string;
  sessionIdForResume?: string;
  persistedMessages: SessionMessage[];
  messagesLoading: boolean;
  onTurnCompleted: () => void;
  onSessionCreated: () => void;
}

function WebChatPane({
  ticketId,
  sessionIdForResume,
  persistedMessages,
  messagesLoading,
  onTurnCompleted,
  onSessionCreated,
}: WebChatPaneProps) {
  const [message, setMessage] = useState("");
  const textareaRef = useRef<HTMLTextAreaElement>(null);
  const { status, messages: wsMessages, sendMessage } = useWebSocket(
    ticketId,
    {
      onMessage: (m) => {
        if (m.type === "status" && /^Turn /.test(m.content)) {
          onTurnCompleted();
        }
      },
    },
    sessionIdForResume,
  );

  useEffect(() => {
    if (wsMessages.length === 0) return;
    const last = wsMessages[wsMessages.length - 1];
    if (last.type === "status" && /^Session /.test(last.content)) {
      onSessionCreated();
    }
  }, [wsMessages, onSessionCreated]);

  const handleSubmit = (e?: React.FormEvent) => {
    e?.preventDefault();
    const trimmed = message.trim();
    if (!trimmed) return;
    if (status !== "connected") return;
    if (sendMessage(trimmed)) {
      setMessage("");
      setTimeout(() => textareaRef.current?.focus(), 0);
    }
  };

  const handleKeyDown = (e: React.KeyboardEvent<HTMLTextAreaElement>) => {
    if (e.key === "Enter" && !e.shiftKey) {
      e.preventDefault();
      handleSubmit();
    }
  };

  const liveMessages = useMemo(
    () =>
      wsMessages
        .filter((m) => m.type === "message" || m.type === "trace")
        .map((m, idx) => ({
          id: m.message_id || `ws-${idx}`,
          type: m.type as "message" | "trace",
          content: m.content,
        })),
    [wsMessages],
  );

  return (
    <div className="flex flex-col min-h-[320px]">
      <div className="flex items-center justify-end mb-2">
        <ConnectionBadge status={status} />
      </div>
      <div className="flex-1 space-y-3">
        <ConversationMainPane
          messages={persistedMessages}
          loading={messagesLoading}
        />
        {liveMessages.length > 0 && (
          <div className="space-y-1 border-t pt-2 mt-2">
            <div className="text-[10px] uppercase tracking-wider text-muted-foreground">
              Live
            </div>
            {liveMessages.map((m) => (
              <div key={m.id} className="text-sm">
                {m.type === "trace" ? (
                  <span className="text-muted-foreground italic text-xs">
                    {m.content}
                  </span>
                ) : (
                  <span className="whitespace-pre-wrap">{m.content}</span>
                )}
              </div>
            ))}
          </div>
        )}
      </div>
      <form
        onSubmit={handleSubmit}
        className="flex items-end gap-2 mt-3 border-t pt-3">
        <Textarea
          ref={textareaRef}
          value={message}
          onChange={(e) => setMessage(e.target.value)}
          onKeyDown={handleKeyDown}
          placeholder={
            status === "connected"
              ? "Type a message… (Shift+Enter for new line)"
              : "Connecting…"
          }
          disabled={status !== "connected"}
          className="flex-1 min-h-[40px] resize-none"
          rows={1}
        />
        <Button
          type="submit"
          size="icon"
          disabled={!message.trim() || status !== "connected"}>
          <Send className="h-4 w-4" />
        </Button>
      </form>
    </div>
  );
}

function ConnectionBadge({
  status,
}: {
  status: "connecting" | "connected" | "disconnected" | "error";
}) {
  switch (status) {
    case "connected":
      return (
        <Badge variant="default" className="gap-1">
          <Wifi className="h-3 w-3" /> Connected
        </Badge>
      );
    case "connecting":
      return (
        <Badge variant="secondary" className="gap-1">
          <Loader2 className="h-3 w-3 animate-spin" /> Connecting
        </Badge>
      );
    case "error":
      return (
        <Badge variant="destructive" className="gap-1">
          <AlertCircle className="h-3 w-3" /> Error
        </Badge>
      );
    default:
      return (
        <Badge variant="outline" className="gap-1">
          <WifiOff className="h-3 w-3" /> Disconnected
        </Badge>
      );
  }
}

// --- helpers ----------------------------------------------------------

function sortSessions(sessions: Session[]): Session[] {
  const order = (s: string) => {
    switch (s) {
      case "slack":
        return 0;
      case "web":
        return 1;
      case "cli":
        return 2;
      default:
        return 3;
    }
  };
  return [...sessions].sort((a, b) => {
    const d = order(a.source || "") - order(b.source || "");
    if (d !== 0) return d;
    const at = a.updatedAt || a.createdAt || "";
    const bt = b.updatedAt || b.createdAt || "";
    return bt.localeCompare(at);
  });
}

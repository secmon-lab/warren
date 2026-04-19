import { useMemo, useState } from "react";
import { useQuery } from "@apollo/client";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { MessageSquare } from "lucide-react";
import { GET_TICKET_SESSION_MESSAGES } from "@/lib/graphql/queries";
import { SessionMessage } from "@/lib/types";
import { ConversationSidePane } from "./ConversationSidePane";
import { ConversationMainPane } from "./ConversationMainPane";

type Source = "slack" | "web" | "cli";

interface TicketConversationProps {
  ticketId: string;
}

// chat-session-redesign Phase 6: unified Slack + Web conversation
// viewer replacing the legacy Comments card. Source toggle filters the
// `ticketSessionMessages` query server-side; session sidebar lets the
// user drill into one Session at a time.
export function TicketConversation({ ticketId }: TicketConversationProps) {
  const [source, setSource] = useState<Source>("slack");
  const [selectedSessionID, setSelectedSessionID] = useState<string | null>(null);

  const { data, loading, error } = useQuery<{
    ticketSessionMessages: SessionMessage[];
  }>(GET_TICKET_SESSION_MESSAGES, {
    variables: { ticketId, source, limit: 500, offset: 0 },
    fetchPolicy: "cache-and-network",
  });

  const all = data?.ticketSessionMessages ?? [];
  const filtered = useMemo(
    () =>
      selectedSessionID == null
        ? all
        : all.filter((m) => m.sessionID === selectedSessionID),
    [all, selectedSessionID],
  );

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center justify-between">
          <div className="flex items-center gap-2">
            <MessageSquare className="h-5 w-5" />
            Conversation ({all.length})
          </div>
          <SourceToggle source={source} onChange={(s) => {
            setSource(s);
            setSelectedSessionID(null);
          }} />
        </CardTitle>
      </CardHeader>
      <CardContent>
        {error && (
          <div className="text-sm text-red-600 py-2">
            Error loading conversation: {error.message}
          </div>
        )}
        <div className="grid grid-cols-1 md:grid-cols-[220px_1fr] gap-4">
          <div className="border-r md:pr-4">
            <ConversationSidePane
              messages={all}
              selectedSessionID={selectedSessionID}
              onSelectSession={setSelectedSessionID}
            />
          </div>
          <div className="min-w-0">
            <ConversationMainPane messages={filtered} loading={loading} />
          </div>
        </div>
      </CardContent>
    </Card>
  );
}

interface SourceToggleProps {
  source: Source;
  onChange: (s: Source) => void;
}

function SourceToggle({ source, onChange }: SourceToggleProps) {
  const options: { value: Source; label: string }[] = [
    { value: "slack", label: "Slack" },
    { value: "web", label: "Web" },
    { value: "cli", label: "CLI" },
  ];
  return (
    <div className="inline-flex rounded-md border border-border">
      {options.map((o) => (
        <button
          key={o.value}
          type="button"
          onClick={() => onChange(o.value)}
          className={`px-3 py-1 text-xs ${
            source === o.value
              ? "bg-primary text-primary-foreground"
              : "bg-background hover:bg-muted"
          }`}>
          {o.label}
        </button>
      ))}
    </div>
  );
}

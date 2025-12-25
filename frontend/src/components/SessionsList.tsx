import { useQuery } from "@apollo/client";
import { useNavigate } from "react-router-dom";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { GET_TICKET_SESSIONS } from "@/lib/graphql/queries";
import { Session } from "@/lib/types";
import { formatRelativeTime } from "@/lib/utils-extended";
import { MessageSquare, Loader2 } from "lucide-react";

interface SessionsListProps {
  ticketId: string;
}

const SESSION_STATUS_COLORS = {
  running: "bg-blue-100 text-blue-800",
  completed: "bg-green-100 text-green-800",
  aborted: "bg-gray-100 text-gray-800",
} as const;

const SESSION_STATUS_LABELS = {
  running: "🔄 Running",
  completed: "✅ Completed",
  aborted: "🛑 Aborted",
} as const;

export function SessionsList({ ticketId }: SessionsListProps) {
  const navigate = useNavigate();

  const { data, loading, error } = useQuery(GET_TICKET_SESSIONS, {
    variables: { ticketId },
    skip: !ticketId,
  });

  const sessions: Session[] = data?.ticketSessions || [];

  const handleSessionClick = (sessionId: string) => {
    navigate(`/sessions/${sessionId}`);
  };

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <MessageSquare className="h-5 w-5" />
          Chat Sessions
        </CardTitle>
      </CardHeader>
      <CardContent>
        {loading && (
          <div className="flex items-center justify-center py-4">
            <Loader2 className="h-5 w-5 animate-spin text-muted-foreground" />
          </div>
        )}

        {error && (
          <div className="text-sm text-red-600">
            Error loading sessions: {error.message}
          </div>
        )}

        {!loading && !error && sessions.length === 0 && (
          <div className="text-sm text-muted-foreground">
            No chat sessions found
          </div>
        )}

        {!loading && !error && sessions.length > 0 && (
          <div className="space-y-2">
            {sessions.map((session) => (
              <div
                key={session.id}
                onClick={() => handleSessionClick(session.id)}
                className="p-3 rounded-md border cursor-pointer hover:bg-muted/50 transition-colors"
              >
                <div className="flex items-start justify-between gap-2">
                  <div className="flex-1 min-w-0">
                    <div className="flex items-center gap-2 mb-1">
                      <span className="text-sm font-mono text-muted-foreground">
                        #{session.id.slice(0, 8)}
                      </span>
                      <Badge
                        className={
                          SESSION_STATUS_COLORS[
                            session.status as keyof typeof SESSION_STATUS_COLORS
                          ] || "bg-gray-100 text-gray-800"
                        }
                        variant="secondary"
                      >
                        {SESSION_STATUS_LABELS[
                          session.status as keyof typeof SESSION_STATUS_LABELS
                        ] || session.status}
                      </Badge>
                    </div>
                    <div className="text-xs text-muted-foreground">
                      Created {formatRelativeTime(session.createdAt)}
                    </div>
                  </div>
                </div>
              </div>
            ))}
          </div>
        )}
      </CardContent>
    </Card>
  );
}

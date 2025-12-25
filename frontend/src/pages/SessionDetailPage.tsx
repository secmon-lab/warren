import { useQuery } from "@apollo/client";
import { useParams, useNavigate } from "react-router-dom";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar";
import { GET_SESSION, GET_SESSION_MESSAGES } from "@/lib/graphql/queries";
import { Session, SessionMessage } from "@/lib/types";
import { formatRelativeTime } from "@/lib/utils-extended";
import { ChevronLeft, MessageSquare, Clock, Calendar, User as UserIcon, ExternalLink } from "lucide-react";
import ReactMarkdown from "react-markdown";
import { format } from "date-fns";

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

export default function SessionDetailPage() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();

  const {
    data: sessionData,
    loading: sessionLoading,
    error: sessionError,
  } = useQuery(GET_SESSION, {
    variables: { id },
    skip: !id,
  });

  const {
    data: messagesData,
    loading: messagesLoading,
    error: messagesError,
  } = useQuery(GET_SESSION_MESSAGES, {
    variables: { sessionId: id },
    skip: !id,
  });

  const session: Session | null = sessionData?.session;
  const messages: SessionMessage[] = messagesData?.sessionMessages || [];

  const handleBackToTicket = () => {
    if (session?.ticketID) {
      navigate(`/tickets/${session.ticketID}`);
    } else {
      navigate(-1);
    }
  };

  const formatAbsoluteTime = (dateString: string) => {
    return format(new Date(dateString), "yyyy-MM-dd HH:mm:ss XXX");
  };

  const formatCompactTime = (dateString: string) => {
    return format(new Date(dateString), "HH:mm");
  };

  // Message type configurations
  const messageTypeConfig = {
    trace: {
      wrapperClass: "hover:bg-gray-100/60",
      timeClass: "opacity-0 group-hover:opacity-100 transition-opacity",
      separatorClass: "border-gray-300/70",
      separatorPadding: "",
    },
    plan: {
      wrapperClass: "hover:bg-blue-50/50",
      timeClass: "",
      separatorClass: "border-blue-300/60",
      separatorPadding: "pb-2",
    },
    response: {
      wrapperClass: "hover:bg-gray-100/60",
      timeClass: "",
      separatorClass: "border-gray-300/70",
      separatorPadding: "pb-2",
    },
  } as const;

  const renderMessageContent = (message: SessionMessage) => {
    if (message.type === "trace") {
      return (
        <div className="flex-1">
          <span className="text-xs text-muted-foreground italic whitespace-pre-line">
            {message.content}
          </span>
        </div>
      );
    }

    if (message.type === "plan") {
      return (
        <div className="flex-1 min-w-0">
          <div className="px-4 py-3 rounded-lg bg-gradient-to-br from-blue-50 to-blue-50/50 shadow-sm">
            <div className="text-sm prose prose-sm max-w-none prose-p:my-2 prose-ul:my-2 prose-li:my-0.5 whitespace-pre-line">
              <ReactMarkdown>{message.content}</ReactMarkdown>
            </div>
          </div>
        </div>
      );
    }

    if (message.type === "response") {
      return (
        <div className="flex-1 min-w-0">
          <div className="px-4 py-3 rounded-lg bg-white shadow-sm ring-1 ring-gray-200/50">
            <div className="text-sm prose prose-sm max-w-none prose-p:my-2 prose-ul:my-2 prose-li:my-0.5">
              <ReactMarkdown>{message.content}</ReactMarkdown>
            </div>
          </div>
        </div>
      );
    }

    // Default
    return (
      <div className="flex-1 min-w-0">
        <div className="px-4 py-3 rounded-lg bg-gray-50/80 shadow-sm">
          <div className="text-sm prose prose-sm max-w-none prose-p:my-2 prose-ul:my-2 prose-li:my-0.5">
            <ReactMarkdown>{message.content}</ReactMarkdown>
          </div>
        </div>
      </div>
    );
  };

  const renderMessage = (message: SessionMessage) => {
    const timeLabel = formatCompactTime(message.createdAt);
    const config =
      messageTypeConfig[message.type as keyof typeof messageTypeConfig] ||
      messageTypeConfig.response;

    return (
      <div key={message.id}>
        {/* Message wrapper */}
        <div
          className={`flex gap-3 py-2 px-4 ${config.wrapperClass} transition-colors group`}
        >
          {/* Time label */}
          <div
            className={`text-xs text-gray-400 font-mono w-12 flex-shrink-0 text-right pt-0.5 ${config.timeClass}`}
          >
            {timeLabel}
          </div>
          {/* Message content */}
          {renderMessageContent(message)}
        </div>
        {/* Separator */}
        <div className={`flex gap-3 px-4 ${config.separatorPadding}`}>
          <div className="w-12 flex-shrink-0" />
          <div className="flex-1 pr-12">
            <div className={`border-b ${config.separatorClass}`} />
          </div>
        </div>
      </div>
    );
  };

  if (!id) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="text-lg text-red-600">No session ID provided</div>
      </div>
    );
  }

  if (sessionLoading || messagesLoading) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="text-lg">Loading session...</div>
      </div>
    );
  }

  if (sessionError || messagesError) {
    const errorMessage =
      (sessionError as Error)?.message ||
      (messagesError as Error)?.message ||
      "Unknown error";
    return (
      <div className="flex items-center justify-center h-64">
        <div className="text-lg text-red-600">
          Error loading session: {errorMessage}
        </div>
      </div>
    );
  }

  if (!session) {
    return (
      <div className="space-y-4">
        <div className="flex items-center justify-center h-64">
          <div className="text-lg text-red-600">Session not found</div>
        </div>
        <div className="flex justify-center">
          <Button variant="outline" onClick={() => navigate(-1)}>
            <ChevronLeft className="h-4 w-4 mr-2" />
            Go back
          </Button>
        </div>
      </div>
    );
  }

  return (
    <div className="flex flex-col h-screen">
      {/* Compact Header */}
      <div className="border-b bg-white px-4 py-3 flex-shrink-0">
        <div className="flex items-center gap-3">
          <Button variant="ghost" size="sm" onClick={handleBackToTicket}>
            <ChevronLeft className="h-4 w-4 mr-1" />
            Back to Ticket
          </Button>
          <div className="h-4 w-px bg-gray-300" />
          <h1 className="text-lg font-semibold">
            Chat Session #{session.id.slice(0, 8)}
          </h1>
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
      </div>

      {/* Main Content - Single Column like Slack */}
      <div className="flex-1 overflow-hidden flex">
        {/* Messages Area (Main) */}
        <div className="flex-1 flex flex-col bg-white">
          <ScrollArea className="flex-1">
            <div className="py-2">
              {messages.length === 0 ? (
                <div className="text-center text-muted-foreground py-8">
                  No messages in this session
                </div>
              ) : (
                messages.map((message) => renderMessage(message))
              )}
            </div>
          </ScrollArea>
        </div>

        {/* Sidebar (Session Info) */}
        <div className="w-80 border-l bg-gray-50 overflow-y-auto flex-shrink-0">
          <div className="p-4 space-y-4">
            <div>
              <h2 className="text-sm font-semibold mb-3 flex items-center gap-2">
                <MessageSquare className="h-4 w-4" />
                Session Information
              </h2>
              <div className="space-y-3 text-sm">
                <div>
                  <span className="text-gray-500 text-xs">Session ID</span>
                  <p className="font-mono text-xs mt-1 break-all">
                    {session.id}
                  </p>
                </div>
                <div>
                  <span className="text-gray-500 text-xs">Ticket ID</span>
                  <p className="font-mono text-xs mt-1">
                    {session.ticketID.slice(0, 8)}
                  </p>
                </div>
                {(session.user || session.userID) && (
                  <div>
                    <span className="text-gray-500 text-xs flex items-center gap-1">
                      <UserIcon className="h-3 w-3" />
                      Started By
                    </span>
                    <div className="mt-1 flex items-center gap-2">
                      <Avatar className="h-5 w-5">
                        <AvatarImage
                          src={`/api/user/${session.user?.id || session.userID}/icon`}
                          alt={session.user?.name || session.userID!}
                        />
                        <AvatarFallback className="text-xs leading-none">
                          {(session.user?.name || session.userID!).charAt(0).toUpperCase()}
                        </AvatarFallback>
                      </Avatar>
                      <span className="text-sm">
                        {session.user?.name || session.userID}
                      </span>
                    </div>
                  </div>
                )}
                {session.query && (
                  <div>
                    <span className="text-gray-500 text-xs">Original Query</span>
                    <p className="text-xs mt-1 whitespace-pre-wrap">
                      {session.query}
                    </p>
                  </div>
                )}
                {session.intent && (
                  <div>
                    <span className="text-gray-500 text-xs">Intent</span>
                    <p className="text-xs mt-1">
                      {session.intent}
                    </p>
                  </div>
                )}
                {session.slackURL && (
                  <div>
                    <span className="text-gray-500 text-xs">Slack Message</span>
                    <p className="text-xs mt-1">
                      <a
                        href={session.slackURL}
                        target="_blank"
                        rel="noopener noreferrer"
                        className="text-blue-600 hover:underline flex items-center gap-1"
                      >
                        Open in Slack
                        <ExternalLink className="h-3 w-3" />
                      </a>
                    </p>
                  </div>
                )}
                <div>
                  <span className="text-gray-500 text-xs flex items-center gap-1">
                    <Calendar className="h-3 w-3" />
                    Created
                  </span>
                  <p className="text-xs mt-1">
                    {formatRelativeTime(session.createdAt)}
                  </p>
                  <p className="font-mono text-xs text-gray-500 mt-0.5">
                    {formatAbsoluteTime(session.createdAt)}
                  </p>
                </div>
                <div>
                  <span className="text-gray-500 text-xs flex items-center gap-1">
                    <Clock className="h-3 w-3" />
                    Updated
                  </span>
                  <p className="text-xs mt-1">
                    {formatRelativeTime(session.updatedAt)}
                  </p>
                  <p className="font-mono text-xs text-gray-500 mt-0.5">
                    {formatAbsoluteTime(session.updatedAt)}
                  </p>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}

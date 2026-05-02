import ReactMarkdown from "react-markdown";
import { SessionMessage } from "@/lib/types";
import { formatRelativeTime } from "@/lib/utils-extended";
import { Bot, User, AlertTriangle, Wrench, ListTodo } from "lucide-react";

interface MessageBubbleProps {
  message: SessionMessage;
}

// chat-session-redesign Phase 6: single SessionMessage render.
// Variant per message type so the timeline is scannable at a glance.
export function MessageBubble({ message }: MessageBubbleProps) {
  const isUser = message.type === "user";
  const label = messageTypeLabel(message.type);
  const Icon = messageTypeIcon(message.type);
  // Prefer the author's Slack avatar when the message is a user post
  // with a SlackUserID — matches Slack's own rendering so Conversation
  // and Slack thread feel consistent. Web/CLI authors fall through to
  // the generic silhouette icon.
  const avatarURL =
    isUser && message.author?.slackUserID
      ? `/api/user/${message.author.slackUserID}/icon`
      : null;

  return (
    <div className={`flex gap-3 ${isUser ? "flex-row-reverse" : ""}`}>
      {/* Fixed 32px circle for every role. The avatar-less case uses
          padding + a small icon; the avatar case fills the circle
          with the image. Both render at the same outer diameter so
          the layout stays aligned whether a user has a Slack avatar
          or not. */}
      <div
        className={`flex-shrink-0 h-8 w-8 rounded-full overflow-hidden flex items-center justify-center ${iconBgClass(
          message.type,
        )}`}>
        {avatarURL ? (
          <img
            src={avatarURL}
            alt={message.author?.displayName || label}
            className="h-full w-full object-cover"
            onError={(e) => {
              // Fall back to the generic icon if the avatar URL
              // returns an error (unauthenticated session, deleted
              // user, Slack API failure, ...).
              e.currentTarget.style.display = "none";
            }}
          />
        ) : (
          <Icon className="h-4 w-4" />
        )}
      </div>
      <div className={`flex-1 min-w-0 ${isUser ? "text-right" : ""}`}>
        <div className="text-xs text-muted-foreground mb-1">
          <span className="font-medium">
            {message.author?.displayName || label}
          </span>
          {/* Timestamp on user messages only: AI-produced bubbles
              inherit the exchange's timestamp from the preceding
              user message (and the Progress panel), so repeating
              per-message times was just noise. */}
          {isUser && formatRelativeTime(message.createdAt) && (
            <>
              <span className="mx-1">·</span>
              <span>{formatRelativeTime(message.createdAt)}</span>
            </>
          )}
        </div>
        <div
          className={`inline-block max-w-full rounded-lg px-3 py-2 text-sm break-words ${bubbleClass(
            message.type,
          )}`}>
          {renderContent(message)}
        </div>
      </div>
    </div>
  );
}

// renderContent dispatches on message type: AI-produced messages
// (response / plan / warning) frequently contain markdown, so render
// them via ReactMarkdown. User messages and raw traces are rendered
// as pre-wrapped plain text to preserve spacing and avoid
// misinterpreting user-typed asterisks.
function renderContent(m: SessionMessage) {
  if (m.type === "response" || m.type === "plan" || m.type === "warning") {
    return (
      <div className="prose prose-sm max-w-none prose-p:my-2 prose-ul:my-2 prose-li:my-0.5">
        <ReactMarkdown>{m.content}</ReactMarkdown>
      </div>
    );
  }
  return <span className="whitespace-pre-wrap">{m.content}</span>;
}

function messageTypeLabel(t: SessionMessage["type"]): string {
  switch (t) {
    case "user":
      return "User";
    case "response":
      return "Response";
    case "plan":
      return "Plan";
    case "trace":
      return "Trace";
    case "warning":
      return "Warning";
    default:
      return t;
  }
}

function messageTypeIcon(t: SessionMessage["type"]) {
  switch (t) {
    case "user":
      return User;
    case "plan":
      return ListTodo;
    case "trace":
      return Wrench;
    case "warning":
      return AlertTriangle;
    default:
      return Bot;
  }
}

function iconBgClass(t: SessionMessage["type"]): string {
  switch (t) {
    case "user":
      return "bg-blue-100 text-blue-700";
    case "warning":
      return "bg-amber-100 text-amber-700";
    case "trace":
      return "bg-slate-100 text-slate-600";
    case "plan":
      return "bg-purple-100 text-purple-700";
    default:
      return "bg-emerald-100 text-emerald-700";
  }
}

function bubbleClass(t: SessionMessage["type"]): string {
  switch (t) {
    case "user":
      return "bg-blue-50 border border-blue-100";
    case "warning":
      return "bg-amber-50 border border-amber-100";
    case "trace":
      return "bg-slate-50 border border-slate-100 font-mono text-xs";
    case "plan":
      return "bg-purple-50 border border-purple-100";
    default:
      return "bg-emerald-50 border border-emerald-100";
  }
}

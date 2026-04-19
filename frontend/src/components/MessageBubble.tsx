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

  return (
    <div className={`flex gap-3 ${isUser ? "flex-row-reverse" : ""}`}>
      <div className={`flex-shrink-0 rounded-full p-2 ${iconBgClass(message.type)}`}>
        <Icon className="h-4 w-4" />
      </div>
      <div className={`flex-1 min-w-0 ${isUser ? "text-right" : ""}`}>
        <div className="text-xs text-muted-foreground mb-1">
          <span className="font-medium">
            {message.author?.displayName || label}
          </span>
          {formatRelativeTime(message.createdAt) && (
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

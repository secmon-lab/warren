import { SessionMessage } from "@/lib/types";
import { MessageBubble } from "./MessageBubble";

interface TurnCardProps {
  turnID: string | null | undefined;
  messages: SessionMessage[];
}

// chat-session-redesign Phase 6: groups messages that share a TurnID
// (one req/res cycle) into a single bounded card so the timeline stays
// visually hierarchical.
export function TurnCard({ turnID, messages }: TurnCardProps) {
  if (messages.length === 0) return null;

  return (
    <div className="rounded-md border border-border bg-background p-3 space-y-3">
      {turnID && (
        <div className="text-[10px] uppercase tracking-wider text-muted-foreground">
          Turn {turnID.slice(0, 8)}
        </div>
      )}
      {messages.map((m) => (
        <MessageBubble key={m.id} message={m} />
      ))}
    </div>
  );
}

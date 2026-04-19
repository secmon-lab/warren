import { useMutation } from "@apollo/client";
import { useState } from "react";
import { AlertTriangle, HelpCircle, Check, X } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Textarea } from "@/components/ui/textarea";
import { RESOLVE_HITL_REQUEST } from "@/lib/graphql/queries";
import { HITLView } from "@/lib/websocket-types";

interface HITLPanelProps {
  // The current pending HITL request for this session, or null when
  // no approval/question is in flight. Cleared by the parent when a
  // hitl_request_resolved envelope arrives or the mutation completes
  // client-side.
  request: HITLView | null;
  onResolved?: () => void;
}

// HITLPanel renders an in-timeline approval/question prompt for a Web
// Session. The backend's webProgressHandle.PresentHITL publishes a
// hitl_request_pending envelope that populates `request`; the
// ConversationMainPane hosts this component just above the live
// message feed so the user cannot miss the prompt.
//
// Rendering branches on request.type: tool_approval shows the tool
// name/args and Allow / Deny buttons; question shows radio options
// plus a Submit button. Both forms include an optional comment field
// that is forwarded via the resolveHITLRequest mutation — Slack's
// presenter has the same shape, so WebUI behavior stays parallel.
export function HITLPanel({ request, onResolved }: HITLPanelProps) {
  const [comment, setComment] = useState("");
  const [selectedAnswer, setSelectedAnswer] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);
  const [resolve] = useMutation(RESOLVE_HITL_REQUEST);

  if (!request || request.status !== "pending") {
    return null;
  }

  const submit = async (approved: boolean) => {
    if (submitting) return;
    setSubmitting(true);
    try {
      await resolve({
        variables: {
          id: request.id,
          approved,
          answer: selectedAnswer,
          comment: comment || null,
        },
      });
      setComment("");
      setSelectedAnswer(null);
      onResolved?.();
    } finally {
      setSubmitting(false);
    }
  };

  if (request.type === "tool_approval") {
    const toolName = String(request.payload?.tool_name ?? "(unknown tool)");
    const toolArgs = request.payload?.tool_args as Record<string, unknown> | undefined;
    return (
      <div className="rounded-md border border-amber-200 bg-amber-50 p-3 space-y-2">
        <div className="flex items-center gap-2 text-amber-800">
          <AlertTriangle className="h-4 w-4" />
          <span className="font-medium text-sm">Approval Required</span>
        </div>
        <div className="text-sm">
          <code className="px-1.5 py-0.5 rounded bg-amber-100 font-mono">
            {toolName}
          </code>
          {toolArgs && Object.keys(toolArgs).length > 0 && (
            <ul className="mt-1.5 text-xs space-y-0.5 text-amber-900">
              {Object.entries(toolArgs).map(([k, v]) => (
                <li key={k}>
                  <span className="font-semibold">{k}:</span> {String(v)}
                </li>
              ))}
            </ul>
          )}
        </div>
        <Textarea
          value={comment}
          onChange={(e) => setComment(e.target.value)}
          placeholder="Comment (optional)"
          className="min-h-[60px] text-xs"
          disabled={submitting}
        />
        <div className="flex gap-2">
          <Button
            size="sm"
            onClick={() => submit(true)}
            disabled={submitting}
            className="bg-emerald-600 hover:bg-emerald-700">
            <Check className="h-3.5 w-3.5 mr-1" /> Allow
          </Button>
          <Button
            size="sm"
            variant="destructive"
            onClick={() => submit(false)}
            disabled={submitting}>
            <X className="h-3.5 w-3.5 mr-1" /> Deny
          </Button>
        </div>
      </div>
    );
  }

  if (request.type === "question") {
    const question = String(request.payload?.question ?? "");
    const options = Array.isArray(request.payload?.options)
      ? (request.payload?.options as unknown[]).map((o) => String(o))
      : [];
    return (
      <div className="rounded-md border border-blue-200 bg-blue-50 p-3 space-y-2">
        <div className="flex items-center gap-2 text-blue-800">
          <HelpCircle className="h-4 w-4" />
          <span className="font-medium text-sm">Question</span>
        </div>
        <div className="text-sm text-blue-900">{question}</div>
        <div className="space-y-1">
          {options.map((opt) => (
            <label key={opt} className="flex items-center gap-2 text-sm cursor-pointer">
              <input
                type="radio"
                name={`hitl-${request.id}`}
                value={opt}
                checked={selectedAnswer === opt}
                onChange={() => setSelectedAnswer(opt)}
                disabled={submitting}
              />
              <span>{opt}</span>
            </label>
          ))}
        </div>
        <Textarea
          value={comment}
          onChange={(e) => setComment(e.target.value)}
          placeholder="Additional comment (optional)"
          className="min-h-[60px] text-xs"
          disabled={submitting}
        />
        <Button
          size="sm"
          onClick={() => submit(true)}
          disabled={submitting || !selectedAnswer}>
          Submit
        </Button>
      </div>
    );
  }

  return null;
}

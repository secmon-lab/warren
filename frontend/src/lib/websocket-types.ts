// WebSocket message types for chat functionality

// Client -> Server messages
export interface ChatMessage {
  type: 'message' | 'ping';
  content: string;
  timestamp: number;
}

// Server -> Client messages
export interface ChatResponse {
  type: 'message' | 'history' | 'status' | 'error' | 'pong' | 'trace';
  content: string;
  user?: User;
  timestamp: number;
  message_id?: string;
}

export interface User {
  id: string;
  name: string;
}

// Helper type guards
export function isChatMessage(data: unknown): data is ChatMessage {
  if (typeof data !== 'object' || data === null) return false;
  const msg = data as Record<string, unknown>;
  return (
    (msg.type === 'message' || msg.type === 'ping') &&
    typeof msg.content === 'string' &&
    typeof msg.timestamp === 'number'
  );
}

export function isChatResponse(data: unknown): data is ChatResponse {
  if (typeof data !== 'object' || data === null) return false;
  const msg = data as Record<string, unknown>;
  const validTypes = ['message', 'history', 'status', 'error', 'pong', 'trace'];
  return (
    validTypes.includes(msg.type as string) &&
    typeof msg.content === 'string' &&
    typeof msg.timestamp === 'number'
  );
}

// Message factory functions
export function createChatMessage(content: string): ChatMessage {
  return {
    type: 'message',
    content,
    timestamp: Date.now(),
  };
}

export function createPingMessage(): ChatMessage {
  return {
    type: 'ping',
    content: '',
    timestamp: Date.now(),
  };
}

// Response type discriminators
export function isMessageResponse(response: ChatResponse): response is ChatResponse & { type: 'message' } {
  return response.type === 'message';
}

export function isHistoryResponse(response: ChatResponse): response is ChatResponse & { type: 'history' } {
  return response.type === 'history';
}

export function isStatusResponse(response: ChatResponse): response is ChatResponse & { type: 'status' } {
  return response.type === 'status';
}

export function isErrorResponse(response: ChatResponse): response is ChatResponse & { type: 'error' } {
  return response.type === 'error';
}

export function isPongResponse(response: ChatResponse): response is ChatResponse & { type: 'pong' } {
  return response.type === 'pong';
}

export function isTraceResponse(response: ChatResponse): response is ChatResponse & { type: 'trace' } {
  return response.type === 'trace';
}

// chat-session-redesign Phase 6: new discriminated envelope emitted by
// the backend (pkg/controller/websocket/events.go). The legacy
// ChatResponse shape above is still sent by the Phase 7 compat shim;
// new callers should prefer EventEnvelope.
export type EventKind =
  | 'session_created'
  | 'session_message_added'
  | 'turn_started'
  | 'turn_ended';

export interface MessageView {
  id: string;
  session_id: string;
  turn_id?: string | null;
  ticket_id?: string | null;
  type: 'trace' | 'plan' | 'response' | 'user' | 'warning';
  content: string;
  author?: {
    user_id: string;
    display_name: string;
    slack_user_id?: string | null;
    email?: string | null;
  } | null;
  created_at: string;
}

export interface SessionView {
  id: string;
  source: 'slack' | 'web' | 'cli';
  ticket_id?: string | null;
  user_id?: string;
  created_at: string;
}

export interface TurnView {
  id: string;
  session_id: string;
  status: 'running' | 'completed' | 'aborted';
  started_at: string;
  ended_at?: string | null;
}

export interface EventEnvelope {
  event: EventKind;
  session_id: string;
  turn_id?: string | null;
  timestamp: string;
  message?: MessageView | null;
  session?: SessionView | null;
  turn?: TurnView | null;
  status?: 'running' | 'completed' | 'aborted' | null;
}

export function isEventEnvelope(data: unknown): data is EventEnvelope {
  if (typeof data !== 'object' || data === null) return false;
  const e = data as Record<string, unknown>;
  const validKinds: EventKind[] = [
    'session_created',
    'session_message_added',
    'turn_started',
    'turn_ended',
  ];
  return (
    typeof e.event === 'string' &&
    (validKinds as string[]).includes(e.event) &&
    typeof e.session_id === 'string'
  );
}
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
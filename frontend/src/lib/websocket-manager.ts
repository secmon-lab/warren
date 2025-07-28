// WebSocket connection manager that maintains persistent connections
// This ensures connections stay alive when switching between tabs/components

import type { ChatResponse } from './websocket-types';

interface ConnectionInfo {
  ws: WebSocket;
  ticketId: string;
  tabId: string;
  lastActivity: number;
  messages: ChatResponse[]; // Cache messages for each connection
}

class WebSocketManager {
  private connections: Map<string, ConnectionInfo> = new Map();
  private cleanupInterval: ReturnType<typeof setInterval> | null = null;
  
  constructor() {
    // Start cleanup timer to remove stale connections
    this.cleanupInterval = setInterval(() => {
      this.cleanupStaleConnections();
    }, 60000); // Check every minute
  }

  getConnectionKey(ticketId: string, tabId: string): string {
    return `${ticketId}:${tabId}`;
  }

  getConnection(ticketId: string, tabId: string): WebSocket | null {
    const key = this.getConnectionKey(ticketId, tabId);
    const info = this.connections.get(key);
    
    if (!info) return null;
    
    // Check if connection is still valid
    if (info.ws.readyState === WebSocket.CLOSED || 
        info.ws.readyState === WebSocket.CLOSING) {
      this.connections.delete(key);
      return null;
    }
    
    // Update last activity
    info.lastActivity = Date.now();
    return info.ws;
  }

  addConnection(ticketId: string, tabId: string, ws: WebSocket, existingMessages?: ChatResponse[]): void {
    const key = this.getConnectionKey(ticketId, tabId);
    
    // Close existing connection if any
    const existing = this.connections.get(key);
    if (existing && existing.ws.readyState !== WebSocket.CLOSED) {
      existing.ws.close(1000, "Replacing with new connection");
    }
    
    this.connections.set(key, {
      ws,
      ticketId,
      tabId,
      lastActivity: Date.now(),
      messages: existingMessages || existing?.messages || []
    });
    
    console.log(`[WebSocketManager] Added connection for ticket ${ticketId}, tab ${tabId}`);
  }

  removeConnection(ticketId: string, tabId: string): void {
    const key = this.getConnectionKey(ticketId, tabId);
    const info = this.connections.get(key);
    
    if (info) {
      if (info.ws.readyState !== WebSocket.CLOSED) {
        info.ws.close(1000, "Connection removed");
      }
      this.connections.delete(key);
      console.log(`[WebSocketManager] Removed connection for ticket ${ticketId}, tab ${tabId}`);
    }
  }

  private cleanupStaleConnections(): void {
    const now = Date.now();
    const staleTimeout = 5 * 60 * 1000; // 5 minutes
    
    for (const [key, info] of this.connections.entries()) {
      // Remove closed connections
      if (info.ws.readyState === WebSocket.CLOSED || 
          info.ws.readyState === WebSocket.CLOSING) {
        this.connections.delete(key);
        console.log(`[WebSocketManager] Cleaned up closed connection for ${key}`);
        continue;
      }
      
      // Remove inactive connections
      if (now - info.lastActivity > staleTimeout) {
        info.ws.close(1000, "Connection timeout due to inactivity");
        this.connections.delete(key);
        console.log(`[WebSocketManager] Cleaned up inactive connection for ${key}`);
      }
    }
  }

  getMessages(ticketId: string, tabId: string): ChatResponse[] {
    const key = this.getConnectionKey(ticketId, tabId);
    const info = this.connections.get(key);
    console.log(`[WebSocketManager] Getting messages for ${key}`, {
      found: !!info,
      messageCount: info?.messages?.length || 0,
      allKeys: Array.from(this.connections.keys())
    });
    return info?.messages || [];
  }

  addMessage(ticketId: string, tabId: string, message: ChatResponse): void {
    const key = this.getConnectionKey(ticketId, tabId);
    const info = this.connections.get(key);
    
    console.log(`[WebSocketManager] Adding message to ${key}`, {
      found: !!info,
      messageType: message.type,
      currentCount: info?.messages?.length || 0
    });
    
    if (info && message.type !== 'pong') { // Don't cache pong messages
      info.messages.push(message);
      info.lastActivity = Date.now();
      
      // Limit message cache size to prevent memory issues
      const MAX_MESSAGES = 500;
      if (info.messages.length > MAX_MESSAGES) {
        info.messages = info.messages.slice(-MAX_MESSAGES);
      }
      
      console.log(`[WebSocketManager] Message added, new count: ${info.messages.length}`);
    }
  }

  clearMessages(ticketId: string, tabId: string): void {
    const key = this.getConnectionKey(ticketId, tabId);
    const info = this.connections.get(key);
    
    if (info) {
      info.messages = [];
    }
  }

  destroy(): void {
    // Clean up all connections
    for (const info of this.connections.values()) {
      if (info.ws.readyState !== WebSocket.CLOSED) {
        info.ws.close(1000, "Manager shutdown");
      }
    }
    this.connections.clear();
    
    // Clear cleanup interval
    if (this.cleanupInterval) {
      clearInterval(this.cleanupInterval);
      this.cleanupInterval = null;
    }
  }
}

// Export singleton instance
export const wsManager = new WebSocketManager();

// Clean up on page unload
if (typeof window !== 'undefined') {
  window.addEventListener('beforeunload', () => {
    wsManager.destroy();
  });
}
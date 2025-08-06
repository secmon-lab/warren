import { useCallback, useEffect, useRef, useState } from 'react';
import { useAuth } from '@/contexts/auth-context';
import {
  ChatResponse,
  createChatMessage,
  createPingMessage,
  isChatResponse,
} from '@/lib/websocket-types';
import { wsManager } from '@/lib/websocket-manager';

export type WebSocketStatus = 'connecting' | 'connected' | 'disconnected' | 'error';

interface UseWebSocketOptions {
  onMessage?: (message: ChatResponse) => void;
  onStatusChange?: (status: WebSocketStatus) => void;
  reconnectInterval?: number;
  maxReconnectAttempts?: number;
}

// Store tab IDs per ticket to ensure persistence across component remounts
const tabIdStore = new Map<string, string>();

// Get or create a persistent tab ID for a ticket
function getOrCreateTabId(ticketId: string): string {
  const existing = tabIdStore.get(ticketId);
  if (existing) {
    return existing;
  }
  
  const timestamp = Date.now();
  const random = Math.random().toString(36).substring(2, 9);
  const tabId = `tab_${timestamp}_${random}`;
  tabIdStore.set(ticketId, tabId);
  return tabId;
}

export function useWebSocket(
  ticketId: string,
  options: UseWebSocketOptions = {}
) {
  const {
    onMessage,
    onStatusChange,
    reconnectInterval = 8000, // Increased to 8 seconds to reduce frequency
    maxReconnectAttempts = 3, // Limited to 3 attempts
  } = options;

  const { user } = useAuth();
  const [status, setStatus] = useState<WebSocketStatus>('disconnected');
  const [messages, setMessages] = useState<ChatResponse[]>([]);
  
  const wsRef = useRef<WebSocket | null>(null);
  const reconnectCountRef = useRef(0);
  const reconnectTimeoutRef = useRef<NodeJS.Timeout | null>(null);
  const pingIntervalRef = useRef<NodeJS.Timeout | null>(null);
  // Use persistent tab ID for this ticket
  const tabIdRef = useRef<string>(getOrCreateTabId(ticketId));

  const onStatusChangeRef = useRef(onStatusChange);
  const onMessageRef = useRef(onMessage);
  
  // Initialize messages from cache on mount
  useEffect(() => {
    if (ticketId) {
      const cachedMessages = wsManager.getMessages(ticketId, tabIdRef.current);
      console.log(`[useWebSocket] Checking cache for ticket ${ticketId}, tab ${tabIdRef.current}`);
      console.log(`[useWebSocket] Found ${cachedMessages.length} cached messages`);
      if (cachedMessages.length > 0) {
        console.log(`[useWebSocket] Restoring messages:`, cachedMessages);
        setMessages(cachedMessages);
      }
    }
  }, [ticketId]);
  
  // Update refs when props change
  useEffect(() => {
    onStatusChangeRef.current = onStatusChange;
    onMessageRef.current = onMessage;
  }, [onStatusChange, onMessage]);

  const updateStatus = useCallback((newStatus: WebSocketStatus) => {
    setStatus(newStatus);
    onStatusChangeRef.current?.(newStatus);
  }, []);

  const connect = useCallback(() => {
    if (!user?.sub || !ticketId) {
      console.log('Skipping connection - missing user or ticketId', { user: user?.sub, ticketId });
      return;
    }

    // Check if we already have a connection in the manager
    const existingWs = wsManager.getConnection(ticketId, tabIdRef.current);
    if (existingWs) {
      console.log('Reusing existing WebSocket connection from manager', {
        ticketId,
        tabId: tabIdRef.current,
        readyState: existingWs.readyState
      });
      wsRef.current = existingWs;
      
      // If already open, update status
      if (existingWs.readyState === WebSocket.OPEN) {
        updateStatus('connected');
      }
      return;
    }

    // Prevent multiple simultaneous connections
    if (wsRef.current) {
      const state = wsRef.current.readyState;
      if (state === WebSocket.CONNECTING || state === WebSocket.OPEN) {
        console.log('WebSocket already connected or connecting, skipping...', {
          readyState: state,
          states: { CONNECTING: WebSocket.CONNECTING, OPEN: WebSocket.OPEN }
        });
        return;
      }
    }

    updateStatus('connecting');

    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const host = window.location.host;
    // Include tab ID in the URL as a query parameter
    const url = `${protocol}//${host}/ws/chat/ticket/${ticketId}?tab=${tabIdRef.current}`;

    console.log('Creating new WebSocket connection to:', url);
    console.log('User:', user);
    console.log('Tab ID:', tabIdRef.current);
    console.log('Connection timestamp:', new Date().toISOString());
    
    const ws = new WebSocket(url);
    wsRef.current = ws;
    
    // Add to manager for persistence
    wsManager.addConnection(ticketId, tabIdRef.current, ws);

    ws.onopen = () => {
      console.log('WebSocket connected to ticket:', ticketId, 'with tab ID:', tabIdRef.current);
      updateStatus('connected');
      reconnectCountRef.current = 0; // Reset retry counter on successful connection

      // Start ping interval to keep connection alive (only if not already running)
      if (!pingIntervalRef.current) {
        pingIntervalRef.current = setInterval(() => {
          if (ws.readyState === WebSocket.OPEN) {
            try {
              const pingMessage = createPingMessage();
              ws.send(JSON.stringify(pingMessage));
              console.debug('Sent ping to keep connection alive');
            } catch (error) {
              console.warn('Failed to send ping:', error);
            }
          } else {
            console.debug('Skipping ping - WebSocket not open:', ws.readyState);
          }
        }, 25000); // Send ping every 25 seconds (more frequent than server timeout)
      }
    };

    ws.onmessage = (event) => {
      console.log('Raw WebSocket message received:', JSON.stringify(event.data));
      try {
        const data = JSON.parse(event.data);
        console.log('Parsed WebSocket message:', data);
        
        if (isChatResponse(data)) {
          // Add to messages array and cache (except pong messages)
          if (data.type !== 'pong') {
            setMessages(prev => [...prev, data]);
            wsManager.addMessage(ticketId, tabIdRef.current, data);
          }
          
          // Call the message handler
          onMessageRef.current?.(data);
        } else {
          console.warn('Received invalid message format:', data);
        }
      } catch (error) {
        console.error('Failed to parse WebSocket message:', error);
        console.error('Raw message data (quoted):', JSON.stringify(event.data));
        console.error('Raw message length:', event.data.length);
        console.error('First 100 chars:', event.data.substring(0, 100));
      }
    };

    ws.onerror = (error) => {
      console.error('WebSocket error:', error);
      console.error('WebSocket readyState:', ws.readyState);
      console.error('WebSocket URL:', url);
      console.error('Error timestamp:', new Date().toISOString());
      updateStatus('error');
    };

    ws.onclose = (event) => {
      console.log('WebSocket closed:', event.code, event.reason);
      updateStatus('disconnected');
      
      // Clear ping interval
      if (pingIntervalRef.current) {
        clearInterval(pingIntervalRef.current);
        pingIntervalRef.current = null;
      }

      // Don't reconnect for normal closure (1000) or going away (1001)
      if (event.code === 1000 || event.code === 1001) {
        console.log('WebSocket closed normally, not reconnecting');
        return;
      }

      // Don't reconnect if we've reached max attempts
      if (reconnectCountRef.current >= maxReconnectAttempts) {
        console.warn('Max reconnection attempts reached. Stopping reconnection.');
        return;
      }

      // Calculate exponential backoff delay
      let delay = reconnectInterval * Math.pow(2, reconnectCountRef.current);
      
      // Special handling for abnormal closure (1006) or no status code (1005) - wait longer
      if (event.code === 1006 || event.code === 1005) {
        delay = Math.max(delay, 12000); // At least 12 seconds for abnormal close
        console.warn('WebSocket closed abnormally, waiting longer before retry.', { 
          code: event.code, 
          reason: event.reason,
          wasClean: event.wasClean
        });
      }

      // Minimum 8 second delay for all reconnections
      delay = Math.max(delay, 8000);

      console.log(`Reconnecting in ${delay}ms... (attempt ${reconnectCountRef.current + 1}/${maxReconnectAttempts})`);
      
      reconnectTimeoutRef.current = setTimeout(() => {
        reconnectCountRef.current++;
        connect();
      }, delay);
    };

    // Add authorization header via subprotocol (if needed)
    // Note: WebSocket doesn't support custom headers, so auth is handled by cookies
  }, [user, ticketId, updateStatus, maxReconnectAttempts, reconnectInterval]);

  const disconnect = useCallback(() => {
    // Clear reconnect timeout
    if (reconnectTimeoutRef.current) {
      clearTimeout(reconnectTimeoutRef.current);
      reconnectTimeoutRef.current = null;
    }

    // Clear ping interval
    if (pingIntervalRef.current) {
      clearInterval(pingIntervalRef.current);
      pingIntervalRef.current = null;
    }

    // Reset reconnection counter to prevent immediate reconnection
    reconnectCountRef.current = 0;

    // Remove from manager and close connection
    if (ticketId) {
      wsManager.removeConnection(ticketId, tabIdRef.current);
    }
    
    wsRef.current = null;
    updateStatus('disconnected');
  }, [updateStatus, ticketId]);

  const sendMessage = useCallback((content: string) => {
    if (wsRef.current?.readyState !== WebSocket.OPEN) {
      console.error('WebSocket is not connected');
      return false;
    }

    const message = createChatMessage(content);

    try {
      wsRef.current.send(JSON.stringify(message));
      return true;
    } catch (error) {
      console.error('Failed to send message:', error);
      return false;
    }
  }, []);

  // Connect on mount but DON'T disconnect on unmount to maintain persistent connections
  useEffect(() => {
    // Skip if no ticketId or user
    if (!ticketId || !user?.sub) {
      return;
    }

    // Add a small delay to prevent race conditions with multiple mounts
    const connectTimer = setTimeout(() => {
      // Connect if not already connected/connecting
      if (!wsRef.current || 
          (wsRef.current.readyState !== WebSocket.OPEN && 
           wsRef.current.readyState !== WebSocket.CONNECTING)) {
        connect();
      }
    }, 100);

    return () => {
      clearTimeout(connectTimer);
      
      // Clear any pending reconnect timeouts
      if (reconnectTimeoutRef.current) {
        clearTimeout(reconnectTimeoutRef.current);
        reconnectTimeoutRef.current = null;
      }
      
      // DON'T disconnect when component unmounts - maintain persistent connection
      // This allows the connection to stay alive when switching tabs
      // The WebSocketManager will handle connection lifecycle
    };
  }, [ticketId, user?.sub, connect]); // Only reconnect when ticketId or user changes

  // Check for existing connection on mount
  useEffect(() => {
    if (!ticketId || !user?.sub) return;
    
    const existingWs = wsManager.getConnection(ticketId, tabIdRef.current);
    if (existingWs && existingWs.readyState === WebSocket.OPEN) {
      console.log('Found existing open connection, reusing it');
      wsRef.current = existingWs;
      updateStatus('connected');
      
      // Reattach event handlers
      existingWs.onmessage = (event) => {
        console.log('Raw WebSocket message received:', JSON.stringify(event.data));
        try {
          const data = JSON.parse(event.data);
          console.log('Parsed WebSocket message:', data);
          
          if (isChatResponse(data)) {
            // Add to messages array and cache (except pong messages)
            if (data.type !== 'pong') {
              setMessages(prev => [...prev, data]);
              wsManager.addMessage(ticketId, tabIdRef.current, data);
            }
            
            // Call the message handler
            onMessageRef.current?.(data);
          } else {
            console.warn('Received invalid message format:', data);
          }
        } catch (error) {
          console.error('Failed to parse WebSocket message:', error);
        }
      };
    }
  }, [ticketId, user?.sub, updateStatus]);

  const stopReconnecting = useCallback(() => {
    if (reconnectTimeoutRef.current) {
      clearTimeout(reconnectTimeoutRef.current);
      reconnectTimeoutRef.current = null;
    }
    reconnectCountRef.current = maxReconnectAttempts; // Prevent further attempts
    console.log('Reconnection stopped manually');
  }, [maxReconnectAttempts]);

  // Handle network reconnection but NOT visibility changes
  useEffect(() => {
    // We intentionally DON'T handle visibility changes to keep connections alive
    // when switching tabs
    
    const handleOnline = () => {
      if (status === 'disconnected' && ticketId && user?.sub) {
        // Check if we're not already trying to reconnect
        if (!reconnectTimeoutRef.current && reconnectCountRef.current < maxReconnectAttempts) {
          console.log('Network came back online, reconnecting');
          connect();
        }
      }
    };

    window.addEventListener('online', handleOnline);

    return () => {
      window.removeEventListener('online', handleOnline);
    };
  }, [status, ticketId, user?.sub, connect, maxReconnectAttempts]);

  // Add explicit cleanup function for when user navigates away from ticket
  const cleanup = useCallback(() => {
    console.log('Cleaning up WebSocket connection for ticket:', ticketId);
    disconnect();
  }, [ticketId, disconnect]);
  
  // Add function to clear message history
  const clearMessages = useCallback(() => {
    console.log('Clearing message history for ticket:', ticketId);
    setMessages([]);
    if (ticketId) {
      wsManager.clearMessages(ticketId, tabIdRef.current);
    }
  }, [ticketId]);

  return {
    status,
    messages,
    sendMessage,
    connect,
    disconnect,
    stopReconnecting,
    cleanup, // For explicit cleanup when needed
    clearMessages, // For clearing message history
  };
}
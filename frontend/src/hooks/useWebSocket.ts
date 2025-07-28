import { useCallback, useEffect, useRef, useState } from 'react';
import { useAuth } from '@/contexts/auth-context';
import {
  ChatResponse,
  createChatMessage,
  createPingMessage,
  isChatResponse,
} from '@/lib/websocket-types';

export type WebSocketStatus = 'connecting' | 'connected' | 'disconnected' | 'error';

interface UseWebSocketOptions {
  onMessage?: (message: ChatResponse) => void;
  onStatusChange?: (status: WebSocketStatus) => void;
  reconnectInterval?: number;
  maxReconnectAttempts?: number;
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

  const onStatusChangeRef = useRef(onStatusChange);
  const onMessageRef = useRef(onMessage);
  
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

    // Prevent multiple simultaneous connections
    if (wsRef.current?.readyState === WebSocket.CONNECTING) {
      console.log('Already connecting, skipping...');
      return;
    }

    // Clean up existing connection
    if (wsRef.current?.readyState === WebSocket.OPEN) {
      console.log('Closing existing connection for reconnect');
      wsRef.current.close();
    }

    updateStatus('connecting');

    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const host = window.location.host;
    const url = `${protocol}//${host}/ws/chat/ticket/${ticketId}`;

    console.log('Attempting WebSocket connection to:', url);
    console.log('User:', user);
    console.log('Connection timestamp:', new Date().toISOString());
    console.log('Current WebSocket state:', wsRef.current?.readyState);
    
    const ws = new WebSocket(url);
    wsRef.current = ws;

    ws.onopen = () => {
      console.log('WebSocket connected to ticket:', ticketId);
      updateStatus('connected');
      reconnectCountRef.current = 0; // Reset retry counter on successful connection

      // Start ping interval to keep connection alive
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
    };

    ws.onmessage = (event) => {
      console.log('Raw WebSocket message received:', JSON.stringify(event.data));
      try {
        const data = JSON.parse(event.data);
        console.log('Parsed WebSocket message:', data);
        
        if (isChatResponse(data)) {
          // Add to messages array (except pong messages)
          if (data.type !== 'pong') {
            setMessages(prev => [...prev, data]);
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
        console.warn('WebSocket closed abnormally, waiting longer before retry.', { code: event.code, reason: event.reason });
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
  }, [user?.sub, ticketId, updateStatus]);

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

    // Close WebSocket connection with normal closure code
    if (wsRef.current) {
      wsRef.current.close(1000, "Client closing connection");
      wsRef.current = null;
    }

    updateStatus('disconnected');
  }, [updateStatus]);

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

  // Connect on mount and disconnect on unmount
  useEffect(() => {
    // Skip if no ticketId or user
    if (!ticketId || !user?.sub) {
      return;
    }

    // Connect if not already connected/connecting
    if (wsRef.current?.readyState !== WebSocket.OPEN && 
        wsRef.current?.readyState !== WebSocket.CONNECTING) {
      connect();
    }

    return () => {
      // Clear any pending reconnect timeouts
      if (reconnectTimeoutRef.current) {
        clearTimeout(reconnectTimeoutRef.current);
        reconnectTimeoutRef.current = null;
      }
      
      // Only disconnect if we're actually connected
      if (wsRef.current && wsRef.current.readyState !== WebSocket.CLOSED) {
        disconnect();
      }
    };
  }, [ticketId, user?.sub]); // Only reconnect when ticketId or user changes

  const stopReconnecting = useCallback(() => {
    if (reconnectTimeoutRef.current) {
      clearTimeout(reconnectTimeoutRef.current);
      reconnectTimeoutRef.current = null;
    }
    reconnectCountRef.current = maxReconnectAttempts; // Prevent further attempts
    console.log('Reconnection stopped manually');
  }, [maxReconnectAttempts]);

  // Prevent disconnection on page visibility change or network change
  useEffect(() => {
    const handleVisibilityChange = () => {
      // Don't disconnect when page becomes hidden
      if (document.visibilityState === 'visible' && status === 'disconnected' && ticketId && user?.sub) {
        console.log('Page became visible, reconnecting if needed');
        connect();
      }
    };

    const handleOnline = () => {
      if (status === 'disconnected' && ticketId && user?.sub) {
        console.log('Network came back online, reconnecting');
        connect();
      }
    };

    document.addEventListener('visibilitychange', handleVisibilityChange);
    window.addEventListener('online', handleOnline);

    return () => {
      document.removeEventListener('visibilitychange', handleVisibilityChange);
      window.removeEventListener('online', handleOnline);
    };
  }, [status, ticketId, user?.sub, connect]);

  return {
    status,
    messages,
    sendMessage,
    connect,
    disconnect,
    stopReconnecting,
  };
}
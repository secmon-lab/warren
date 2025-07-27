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
    reconnectInterval = 5000, // Increased from 1000ms to 5000ms
    maxReconnectAttempts = 3, // Reduced from 5 to 3
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
    if (!user?.sub || !ticketId) return;

    // Clean up existing connection
    if (wsRef.current?.readyState === WebSocket.OPEN) {
      wsRef.current.close();
    }

    updateStatus('connecting');

    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const host = window.location.host;
    const url = `${protocol}//${host}/ws/chat/ticket/${ticketId}`;

    console.log('Attempting WebSocket connection to:', url);
    console.log('User:', user);
    
    const ws = new WebSocket(url);
    wsRef.current = ws;

    ws.onopen = () => {
      console.log('WebSocket connected to ticket:', ticketId);
      updateStatus('connected');
      reconnectCountRef.current = 0; // Reset retry counter on successful connection

      // Start ping interval to keep connection alive
      pingIntervalRef.current = setInterval(() => {
        if (ws.readyState === WebSocket.OPEN) {
          const pingMessage = createPingMessage();
          ws.send(JSON.stringify(pingMessage));
        }
      }, 30000); // Send ping every 30 seconds
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

      // Don't reconnect if we've reached max attempts
      if (reconnectCountRef.current >= maxReconnectAttempts) {
        console.warn('Max reconnection attempts reached. Stopping reconnection.');
        return;
      }

      // Special handling for abnormal closure (1006)
      let delay = reconnectInterval * Math.pow(2, reconnectCountRef.current);
      if (event.code === 1006) {
        // For abnormal closures, wait longer
        delay = Math.max(delay, 10000); // At least 10 seconds
        console.warn('Abnormal WebSocket closure detected. Waiting longer before retry.');
      }

      console.log(`Reconnecting in ${delay}ms... (attempt ${reconnectCountRef.current + 1}/${maxReconnectAttempts})`);
      
      reconnectTimeoutRef.current = setTimeout(() => {
        reconnectCountRef.current++;
        connect();
      }, delay);
    };

    // Add authorization header via subprotocol (if needed)
    // Note: WebSocket doesn't support custom headers, so auth is handled by cookies
  }, [user?.sub, ticketId, reconnectInterval, maxReconnectAttempts]);

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

    // Close WebSocket connection
    if (wsRef.current) {
      wsRef.current.close();
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
    connect();

    return () => {
      disconnect();
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

  return {
    status,
    messages,
    sendMessage,
    connect,
    disconnect,
    stopReconnecting,
  };
}
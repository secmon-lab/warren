import React, { useState, useRef, useEffect } from 'react';
import { Send, Loader2, Wifi, WifiOff, AlertCircle } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { ScrollArea } from '@/components/ui/scroll-area';
import { Textarea } from '@/components/ui/textarea';
import { Badge } from '@/components/ui/badge';
// Alert component removed - using simple div instead
import { useWebSocket } from '@/hooks/useWebSocket';
import { useAuth } from '@/contexts/auth-context';
import { 
  ChatResponse, 
  isMessageResponse, 
  isTraceResponse, 
  isErrorResponse,
  isStatusResponse 
} from '@/lib/websocket-types';
import { UserName } from '@/components/ui/user-name';
import { cn } from '@/lib/utils';

interface TicketChatProps {
  ticketId: string;
}

export function TicketChat({ ticketId }: TicketChatProps) {
  const { user } = useAuth();
  const [message, setMessage] = useState('');
  const [isSubmitting, setIsSubmitting] = useState(false);
  const scrollAreaRef = useRef<HTMLDivElement>(null);
  const textareaRef = useRef<HTMLTextAreaElement>(null);

  const { status, messages, sendMessage } = useWebSocket(ticketId);

  // Auto-scroll to bottom when messages change
  useEffect(() => {
    if (scrollAreaRef.current) {
      const scrollContainer = scrollAreaRef.current.querySelector('[data-radix-scroll-area-viewport]');
      if (scrollContainer) {
        scrollContainer.scrollTop = scrollContainer.scrollHeight;
      }
    }
  }, [messages]);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    
    const trimmedMessage = message.trim();
    if (!trimmedMessage || isSubmitting || status !== 'connected') return;

    setIsSubmitting(true);
    const success = sendMessage(trimmedMessage);
    
    if (success) {
      setMessage('');
      // Refocus textarea after clearing
      setTimeout(() => textareaRef.current?.focus(), 0);
    }
    
    setIsSubmitting(false);
  };

  const handleKeyDown = (e: React.KeyboardEvent<HTMLTextAreaElement>) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      handleSubmit(e);
    }
  };

  const renderMessage = (msg: ChatResponse, index: number) => {
    const isCurrentUser = msg.user?.id === user?.sub;
    
    if (isTraceResponse(msg)) {
      return (
        <div key={index} className="flex justify-center py-1">
          <span className="text-xs text-muted-foreground italic">{msg.content}</span>
        </div>
      );
    }

    if (isStatusResponse(msg)) {
      return (
        <div key={index} className="flex justify-center py-2">
          <Badge variant="secondary" className="text-xs">
            {msg.content}
          </Badge>
        </div>
      );
    }

    if (isErrorResponse(msg)) {
      return (
        <div key={index} className="mx-4 my-2">
          <div className="flex items-center space-x-2 p-3 bg-red-50 border border-red-200 rounded-lg text-red-800">
            <AlertCircle className="h-4 w-4" />
            <span className="text-sm">{msg.content}</span>
          </div>
        </div>
      );
    }

    if (isMessageResponse(msg)) {
      return (
        <div
          key={index}
          className={cn(
            "flex w-full mb-4",
            isCurrentUser ? "justify-end" : "justify-start"
          )}
        >
          <div className={cn(
            "flex max-w-[70%] gap-3",
            isCurrentUser ? "flex-row-reverse" : "flex-row"
          )}>
            {msg.user && (
              <div className="flex-shrink-0">
                <UserName userID={msg.user.id} className="text-sm font-medium" />
              </div>
            )}
            <div
              className={cn(
                "px-4 py-2 rounded-lg",
                isCurrentUser 
                  ? "bg-primary text-primary-foreground" 
                  : "bg-muted"
              )}
            >
              <p className="text-sm whitespace-pre-wrap break-words">{msg.content}</p>
              <p className="text-xs opacity-70 mt-1">
                {new Date(msg.timestamp).toLocaleTimeString()}
              </p>
            </div>
          </div>
        </div>
      );
    }

    return null;
  };

  const getStatusBadge = () => {
    switch (status) {
      case 'connected':
        return (
          <Badge variant="default" className="gap-1">
            <Wifi className="h-3 w-3" />
            Connected
          </Badge>
        );
      case 'connecting':
        return (
          <Badge variant="secondary" className="gap-1">
            <Loader2 className="h-3 w-3 animate-spin" />
            Connecting...
          </Badge>
        );
      case 'disconnected':
        return (
          <Badge variant="outline" className="gap-1">
            <WifiOff className="h-3 w-3" />
            Disconnected
          </Badge>
        );
      case 'error':
        return (
          <Badge variant="destructive" className="gap-1">
            <AlertCircle className="h-3 w-3" />
            Error
          </Badge>
        );
    }
  };

  return (
    <Card className="flex flex-col h-full">
      <CardHeader className="pb-3">
        <div className="flex items-center justify-between">
          <CardTitle>Chat</CardTitle>
          {getStatusBadge()}
        </div>
      </CardHeader>
      <CardContent className="flex-1 flex flex-col min-h-0 p-0">
        <ScrollArea ref={scrollAreaRef} className="flex-1 px-4">
          <div className="py-4">
            {messages.length === 0 ? (
              <div className="text-center text-muted-foreground text-sm py-8">
                No messages yet. Start a conversation!
              </div>
            ) : (
              messages.map((msg, index) => renderMessage(msg, index))
            )}
          </div>
        </ScrollArea>
        
        <form onSubmit={handleSubmit} className="p-4 border-t">
          <div className="flex gap-2">
            <Textarea
              ref={textareaRef}
              value={message}
              onChange={(e) => setMessage(e.target.value)}
              onKeyDown={handleKeyDown}
              placeholder={
                status === 'connected' 
                  ? "Type a message... (Shift+Enter for new line)" 
                  : "Connecting to chat..."
              }
              disabled={status !== 'connected' || isSubmitting}
              className="min-h-[60px] resize-none"
              rows={2}
            />
            <Button
              type="submit"
              size="icon"
              disabled={!message.trim() || status !== 'connected' || isSubmitting}
              className="self-end"
            >
              {isSubmitting ? (
                <Loader2 className="h-4 w-4 animate-spin" />
              ) : (
                <Send className="h-4 w-4" />
              )}
            </Button>
          </div>
        </form>
      </CardContent>
    </Card>
  );
}
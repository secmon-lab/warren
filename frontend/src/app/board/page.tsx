'use client';

import { useQuery } from '@apollo/client';
import Link from 'next/link';
import { MainLayout } from '@/components/layout/main-layout';
import { Card, CardContent, CardHeader } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { GET_TICKETS } from '@/lib/graphql/queries';
import { Ticket, TicketStatus, TICKET_STATUS_LABELS, TICKET_STATUS_COLORS } from '@/lib/types';
import { formatRelativeTime } from '@/lib/utils-extended';
import { AlertCircle, MessageSquare, User } from 'lucide-react';

const BOARD_STATUSES: TicketStatus[] = ['open', 'pending', 'resolved'];

export default function BoardPage() {
  const { data, loading, error } = useQuery(GET_TICKETS, {
    variables: {
      statuses: BOARD_STATUSES,
    },
  });

  const tickets: Ticket[] = data?.tickets || [];

  const ticketsByStatus = BOARD_STATUSES.reduce((acc, status) => {
    acc[status] = tickets.filter(ticket => ticket.status === status);
    return acc;
  }, {} as Record<TicketStatus, Ticket[]>);

  if (loading) {
    return (
      <MainLayout>
        <div className="flex items-center justify-center h-64">
          <div className="text-lg">Loading board...</div>
        </div>
      </MainLayout>
    );
  }

  if (error) {
    return (
      <MainLayout>
        <div className="flex items-center justify-center h-64">
          <div className="text-lg text-red-600">Error loading board: {error.message}</div>
        </div>
      </MainLayout>
    );
  }

  return (
    <MainLayout>
      <div className="space-y-6">
        <div>
          <h1 className="text-3xl font-bold tracking-tight">Board</h1>
          <p className="text-muted-foreground">
            Kanban board view of security tickets
          </p>
        </div>

        <div className="grid gap-6 lg:grid-cols-3">
          {BOARD_STATUSES.map((status) => (
            <div key={status} className="space-y-4">
              <div className="flex items-center justify-between">
                <h2 className="text-lg font-semibold flex items-center gap-2">
                  <Badge className={TICKET_STATUS_COLORS[status]} variant="secondary">
                    {TICKET_STATUS_LABELS[status]}
                  </Badge>
                  <span className="text-sm text-muted-foreground">
                    ({ticketsByStatus[status].length})
                  </span>
                </h2>
              </div>

              <div className="space-y-3 min-h-[400px]">
                {ticketsByStatus[status].map((ticket) => (
                  <Link key={ticket.id} href={`/tickets/${ticket.id}`}>
                    <Card className="hover:shadow-md transition-shadow cursor-pointer">
                      <CardHeader className="pb-2">
                        <div className="flex items-start justify-between">
                          <div className="flex items-center gap-2">
                            <AlertCircle className="h-4 w-4 text-muted-foreground" />
                            <span className="text-sm text-muted-foreground font-mono">
                              #{ticket.id.slice(0, 8)}
                            </span>
                          </div>
                          <Badge 
                            className={TICKET_STATUS_COLORS[ticket.status as TicketStatus]}
                            variant="secondary"
                          >
                            {TICKET_STATUS_LABELS[ticket.status as TicketStatus]}
                          </Badge>
                        </div>
                      </CardHeader>
                      <CardContent className="pt-0">
                        <h3 className="font-medium text-sm mb-3 line-clamp-2">
                          Ticket {ticket.id.slice(0, 8)}
                        </h3>
                        
                        <div className="space-y-2">
                          <div className="flex items-center justify-between text-xs text-muted-foreground">
                            <div className="flex items-center gap-1">
                              <User className="h-3 w-3" />
                              <span>{ticket.assignee ? ticket.assignee.name : 'Unassigned'}</span>
                            </div>
                            <span>{formatRelativeTime(ticket.createdAt)}</span>
                          </div>
                          
                          <div className="flex items-center gap-3 text-xs text-muted-foreground">
                            <div className="flex items-center gap-1">
                              <MessageSquare className="h-3 w-3" />
                              <span>{ticket.comments.length} comments</span>
                            </div>
                            <div className="flex items-center gap-1">
                              <AlertCircle className="h-3 w-3" />
                              <span>{ticket.alerts.length} alerts</span>
                            </div>
                          </div>
                        </div>
                      </CardContent>
                    </Card>
                  </Link>
                ))}
                
                {ticketsByStatus[status].length === 0 && (
                  <div className="flex items-center justify-center h-32 border-2 border-dashed border-muted rounded-lg">
                    <p className="text-sm text-muted-foreground">No {status} tickets</p>
                  </div>
                )}
              </div>
            </div>
          ))}
        </div>

        <div className="mt-8 p-4 bg-muted/50 rounded-lg">
          <div className="flex items-center justify-between text-sm">
            <span className="text-muted-foreground">
              Total tickets on board: {tickets.length}
            </span>
            <div className="flex gap-4">
              {BOARD_STATUSES.map((status) => (
                <span key={status} className="text-muted-foreground">
                  {TICKET_STATUS_LABELS[status]}: {ticketsByStatus[status].length}
                </span>
              ))}
            </div>
          </div>
        </div>
      </div>
    </MainLayout>
  );
} 
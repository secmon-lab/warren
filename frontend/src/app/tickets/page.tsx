'use client';

import { useQuery } from '@apollo/client';
import { useState } from 'react';
import Link from 'next/link';
import { MainLayout } from '@/components/layout/main-layout';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { Pagination, PaginationContent, PaginationItem, PaginationLink, PaginationNext, PaginationPrevious } from '@/components/ui/pagination';
import { GET_TICKETS } from '@/lib/graphql/queries';
import { Ticket, TicketStatus, TICKET_STATUS_LABELS, TICKET_STATUS_COLORS } from '@/lib/types';
import { formatRelativeTime } from '@/lib/utils-extended';
import { AlertCircle, MessageSquare, User } from 'lucide-react';

const ITEMS_PER_PAGE = 10;
const ALL_STATUSES: TicketStatus[] = ['open', 'pending', 'resolved', 'archived'];

export default function TicketsPage() {
  const [currentPage, setCurrentPage] = useState(1);
  const [selectedStatuses, setSelectedStatuses] = useState<TicketStatus[]>([]);
  const [activeTab, setActiveTab] = useState<'all' | TicketStatus>('all');

  const { data, loading, error } = useQuery(GET_TICKETS, {
    variables: {
      statuses: selectedStatuses.length > 0 ? selectedStatuses : undefined,
      offset: (currentPage - 1) * ITEMS_PER_PAGE,
      limit: ITEMS_PER_PAGE,
    },
  });

  // Sort tickets by createdAt in descending order (newest first)
  const tickets: Ticket[] = [...(data?.tickets || [])].sort((a: Ticket, b: Ticket) => 
    new Date(b.createdAt).getTime() - new Date(a.createdAt).getTime()
  );

  const handleStatusFilter = (status: TicketStatus | 'all') => {
    if (status === 'all') {
      setSelectedStatuses([]);
      setActiveTab('all');
    } else {
      setSelectedStatuses([status]);
      setActiveTab(status);
    }
    setCurrentPage(1);
  };

  if (loading) {
    return (
      <MainLayout>
        <div className="flex items-center justify-center h-64">
          <div className="text-lg">Loading tickets...</div>
        </div>
      </MainLayout>
    );
  }

  if (error) {
    return (
      <MainLayout>
        <div className="flex items-center justify-center h-64">
          <div className="text-lg text-red-600">Error loading tickets: {error.message}</div>
        </div>
      </MainLayout>
    );
  }

  return (
    <MainLayout>
      <div className="space-y-6">
        <div>
          <h1 className="text-3xl font-bold tracking-tight">Tickets</h1>
          <p className="text-muted-foreground">
            Manage and track security tickets
          </p>
        </div>

        <Tabs value={activeTab} onValueChange={(value) => handleStatusFilter(value as TicketStatus | 'all')} className="space-y-6">
          <TabsList>
            <TabsTrigger value="all">All</TabsTrigger>
            {ALL_STATUSES.map((status) => (
              <TabsTrigger key={status} value={status}>
                {TICKET_STATUS_LABELS[status]}
              </TabsTrigger>
            ))}
          </TabsList>

          <TabsContent value={activeTab} className="space-y-4">
            <Card>
              <CardHeader>
                <CardTitle>
                  {tickets.length} {activeTab === 'all' ? '' : activeTab} tickets
                </CardTitle>
              </CardHeader>
              <CardContent className="p-0">
                <div className="divide-y">
                  {tickets.map((ticket) => (
                    <Link key={ticket.id} href={`/tickets/${ticket.id}`}>
                      <div className="p-4 hover:bg-muted/50 transition-colors">
                        <div className="flex items-start gap-3">
                          <AlertCircle className="h-5 w-5 text-muted-foreground mt-0.5" />
                          <div className="flex-1 min-w-0">
                            <div className="flex items-center gap-2 mb-1">
                              <Badge 
                                className={TICKET_STATUS_COLORS[ticket.status as TicketStatus]}
                                variant="secondary"
                              >
                                {TICKET_STATUS_LABELS[ticket.status as TicketStatus]}
                              </Badge>
                              <span className="text-sm text-muted-foreground">
                                #{ticket.id.slice(0, 8)}
                              </span>
                            </div>
                            <h3 className="font-medium text-foreground hover:text-primary">
                              {ticket.title || `Ticket ${ticket.id.slice(0, 8)}`}
                            </h3>
                            {ticket.description && (
                              <p className="text-sm text-muted-foreground mt-1 line-clamp-2">
                                {ticket.description}
                              </p>
                            )}
                            <div className="flex items-center gap-4 mt-2 text-sm text-muted-foreground">
                              <span>opened {formatRelativeTime(ticket.createdAt)}</span>
                              <div className="flex items-center gap-1">
                                <User className="h-4 w-4" />
                                <span>{ticket.assignee ? ticket.assignee.name : 'Unassigned'}</span>
                              </div>
                              <div className="flex items-center gap-1">
                                <MessageSquare className="h-4 w-4" />
                                <span>{ticket.comments.length}</span>
                              </div>
                              <div className="flex items-center gap-1">
                                <AlertCircle className="h-4 w-4" />
                                <span>{ticket.alerts.length} alerts</span>
                              </div>
                            </div>
                          </div>
                        </div>
                      </div>
                    </Link>
                  ))}
                </div>
              </CardContent>
            </Card>
          </TabsContent>
        </Tabs>

        <Pagination>
          <PaginationContent>
            <PaginationItem>
              <PaginationPrevious 
                href="#" 
                onClick={(e) => {
                  e.preventDefault();
                  if (currentPage > 1) setCurrentPage(currentPage - 1);
                }}
              />
            </PaginationItem>
            <PaginationItem>
              <PaginationLink href="#" isActive>
                {currentPage}
              </PaginationLink>
            </PaginationItem>
            <PaginationItem>
              <PaginationNext 
                href="#"
                onClick={(e) => {
                  e.preventDefault();
                  setCurrentPage(currentPage + 1);
                }}
              />
            </PaginationItem>
          </PaginationContent>
        </Pagination>
      </div>
    </MainLayout>
  );
} 
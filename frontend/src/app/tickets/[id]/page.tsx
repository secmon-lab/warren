'use client';

import { useQuery, useMutation } from '@apollo/client';
import { useParams } from 'next/navigation';
import { useState } from 'react';
import { MainLayout } from '@/components/layout/main-layout';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { GET_TICKET, UPDATE_TICKET_STATUS } from '@/lib/graphql/queries';
import { Ticket, TicketStatus, TICKET_STATUS_LABELS, TICKET_STATUS_COLORS } from '@/lib/types';
import { formatRelativeTime } from '@/lib/utils-extended';
import { AlertCircle, MessageSquare, Calendar, User, Clock, FileText, Eye } from 'lucide-react';

export default function TicketDetailPage() {
  const params = useParams();
  const ticketId = params.id as string;
  const [isUpdating, setIsUpdating] = useState(false);

  const { data, loading, error, refetch } = useQuery(GET_TICKET, {
    variables: { id: ticketId },
  });

  const [updateTicketStatus] = useMutation(UPDATE_TICKET_STATUS, {
    onCompleted: () => {
      console.log('Ticket status updated successfully');
      refetch();
    },
    onError: (error) => {
      console.error('Failed to update ticket status:', error.message);
    },
  });

  const ticket: Ticket = data?.ticket;

  const handleStatusUpdate = async (newStatus: TicketStatus) => {
    setIsUpdating(true);
    try {
      await updateTicketStatus({
        variables: {
          id: ticketId,
          status: newStatus,
        },
      });
    } finally {
      setIsUpdating(false);
    }
  };

  if (loading) {
    return (
      <MainLayout>
        <div className="flex items-center justify-center h-64">
          <div className="text-lg">Loading ticket...</div>
        </div>
      </MainLayout>
    );
  }

  if (error || !ticket) {
    return (
      <MainLayout>
        <div className="flex items-center justify-center h-64">
          <div className="text-lg text-red-600">
            {error ? `Error loading ticket: ${error.message}` : 'Ticket not found'}
          </div>
        </div>
      </MainLayout>
    );
  }

  return (
    <MainLayout>
      <div className="space-y-6">
        {/* Header */}
        <div className="flex items-start justify-between">
          <div className="space-y-1">
            <div className="flex items-center gap-2">
              <h1 className="text-3xl font-bold tracking-tight">
                {ticket.title || `Ticket ${ticket.id.slice(0, 8)}`}
              </h1>
              <Badge 
                className={TICKET_STATUS_COLORS[ticket.status as TicketStatus]}
                variant="secondary"
              >
                {TICKET_STATUS_LABELS[ticket.status as TicketStatus]}
              </Badge>
            </div>
            <p className="text-muted-foreground">
              #{ticket.id} • Created {formatRelativeTime(ticket.createdAt)} • 
              Updated {formatRelativeTime(ticket.updatedAt)}
            </p>
          </div>
          <div className="flex items-center gap-2">
            <div className="flex gap-1">
              <Button
                size="sm"
                variant={ticket.status === 'open' ? 'default' : 'outline'}
                onClick={() => handleStatusUpdate('open')}
                disabled={isUpdating || ticket.status === 'open'}
              >
                🔍 Open
              </Button>
              <Button
                size="sm"
                variant={ticket.status === 'resolved' ? 'default' : 'outline'}
                onClick={() => handleStatusUpdate('resolved')}
                disabled={isUpdating || ticket.status === 'resolved'}
              >
                ✅ Resolve
              </Button>
            </div>
          </div>
        </div>

        <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
          {/* Main Content */}
          <div className="lg:col-span-2 space-y-6">
            {/* Summary Section */}
            {ticket.summary && (
              <Card>
                <CardHeader>
                  <CardTitle className="flex items-center gap-2">
                    <Eye className="h-5 w-5" />
                    Summary
                  </CardTitle>
                </CardHeader>
                <CardContent>
                  <p className="text-sm leading-relaxed">
                    {ticket.summary}
                  </p>
                </CardContent>
              </Card>
            )}

            {/* Description Section */}
            {ticket.description && (
              <Card>
                <CardHeader>
                  <CardTitle className="flex items-center gap-2">
                    <FileText className="h-5 w-5" />
                    Description
                  </CardTitle>
                </CardHeader>
                <CardContent>
                  <p className="text-sm leading-relaxed whitespace-pre-wrap">
                    {ticket.description}
                  </p>
                </CardContent>
              </Card>
            )}

            {/* Alerts Section */}
            <Card>
              <CardHeader>
                <CardTitle className="flex items-center gap-2">
                  <AlertCircle className="h-5 w-5" />
                  Related Alerts ({ticket.alerts.length})
                </CardTitle>
              </CardHeader>
              <CardContent className="p-0">
                <div className="divide-y">
                  {ticket.alerts.map((alert) => (
                    <div key={alert.id} className="p-4">
                      <div className="flex items-start gap-3">
                        <AlertCircle className="h-5 w-5 text-orange-500 mt-0.5" />
                        <div className="flex-1 min-w-0">
                          <h4 className="font-medium text-foreground">
                            {alert.title}
                          </h4>
                          {alert.description && (
                            <p className="text-sm text-muted-foreground mt-1">
                              {alert.description}
                            </p>
                          )}
                          <div className="flex items-center gap-4 mt-2 text-sm text-muted-foreground">
                            <span>#{alert.id.slice(0, 8)}</span>
                            <span>created {formatRelativeTime(alert.createdAt)}</span>
                          </div>
                        </div>
                      </div>
                    </div>
                  ))}
                </div>
              </CardContent>
            </Card>

            {/* Comments Section */}
            <Card>
              <CardHeader>
                <CardTitle className="flex items-center gap-2">
                  <MessageSquare className="h-5 w-5" />
                  Comments ({ticket.comments.length})
                </CardTitle>
              </CardHeader>
              <CardContent className="p-0">
                <div className="divide-y">
                  {ticket.comments.length === 0 ? (
                    <div className="p-4 text-center text-muted-foreground">
                      No comments yet
                    </div>
                  ) : (
                    ticket.comments.map((comment) => (
                      <div key={comment.id} className="p-4">
                        <div className="flex items-start gap-3">
                          <div className="w-8 h-8 bg-primary/10 rounded-full flex items-center justify-center">
                            <User className="h-4 w-4" />
                          </div>
                          <div className="flex-1 min-w-0">
                            <div className="flex items-center gap-2 mb-1">
                              <span className="font-medium">System</span>
                              <span className="text-sm text-muted-foreground">
                                {formatRelativeTime(comment.createdAt)}
                              </span>
                            </div>
                            <p className="text-sm leading-relaxed whitespace-pre-wrap">
                              {comment.content}
                            </p>
                          </div>
                        </div>
                      </div>
                    ))
                  )}
                </div>
              </CardContent>
            </Card>
          </div>

          {/* Sidebar */}
          <div className="space-y-6">
            {/* Metadata */}
            <Card>
              <CardHeader>
                <CardTitle>Details</CardTitle>
              </CardHeader>
              <CardContent className="space-y-4">
                <div className="flex items-center gap-2 text-sm">
                  <User className="h-4 w-4 text-muted-foreground" />
                  <span className="text-muted-foreground">Assignee:</span>
                  <span>Unassigned</span>
                </div>
                <div className="flex items-center gap-2 text-sm">
                  <Calendar className="h-4 w-4 text-muted-foreground" />
                  <span className="text-muted-foreground">Created:</span>
                  <span>{formatRelativeTime(ticket.createdAt)}</span>
                </div>
                <div className="flex items-center gap-2 text-sm">
                  <Clock className="h-4 w-4 text-muted-foreground" />
                  <span className="text-muted-foreground">Updated:</span>
                  <span>{formatRelativeTime(ticket.updatedAt)}</span>
                </div>
                <div className="flex items-center gap-2 text-sm">
                  <AlertCircle className="h-4 w-4 text-muted-foreground" />
                  <span className="text-muted-foreground">Alerts:</span>
                  <span>{ticket.alerts.length}</span>
                </div>
                <div className="flex items-center gap-2 text-sm">
                  <MessageSquare className="h-4 w-4 text-muted-foreground" />
                  <span className="text-muted-foreground">Comments:</span>
                  <span>{ticket.comments.length}</span>
                </div>
              </CardContent>
            </Card>

            {/* Actions */}
            <Card>
              <CardHeader>
                <CardTitle>Actions</CardTitle>
              </CardHeader>
              <CardContent className="space-y-2">
                <Button className="w-full" variant="outline">
                  Add Comment
                </Button>
                <Button className="w-full" variant="outline">
                  Edit Ticket
                </Button>
                <Button className="w-full" variant="destructive">
                  Delete Ticket
                </Button>
              </CardContent>
            </Card>
          </div>
        </div>
      </div>
    </MainLayout>
  );
} 
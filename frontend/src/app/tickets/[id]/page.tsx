'use client';

import { useQuery } from '@apollo/client';
import { useState, use } from 'react';
import { MainLayout } from '@/components/layout/main-layout';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Separator } from '@/components/ui/separator';
import { GET_TICKET } from '@/lib/graphql/queries';
import { Ticket, Alert, TICKET_STATUS_LABELS, TICKET_STATUS_COLORS } from '@/lib/types';
import { formatDateTime, formatRelativeTime, generateSlackLink, generateAlertSlackLink } from '@/lib/utils-extended';
import { AlertCircle, MessageSquare, User, ExternalLink } from 'lucide-react';

interface TicketDetailPageProps {
  params: Promise<{ id: string }>;
}

export default function TicketDetailPage({ params }: TicketDetailPageProps) {
  const resolvedParams = use(params);
  const [selectedAlert, setSelectedAlert] = useState<Alert | null>(null);
  
  const { data, loading, error } = useQuery(GET_TICKET, {
    variables: { id: resolvedParams.id },
  });

  const ticket: Ticket = data?.ticket;

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

  const slackLink = generateSlackLink(ticket.id);

  return (
    <MainLayout>
      <div className="space-y-6">
        <div className="flex items-center justify-between">
          <div>
            <div className="flex items-center gap-2 mb-2">
              <h1 className="text-3xl font-bold tracking-tight">
                Ticket #{ticket.id.slice(0, 8)}
              </h1>
              <Badge className={TICKET_STATUS_COLORS[ticket.status as keyof typeof TICKET_STATUS_COLORS]}>
                {TICKET_STATUS_LABELS[ticket.status as keyof typeof TICKET_STATUS_LABELS]}
              </Badge>
            </div>
            <p className="text-muted-foreground">
              Created {formatRelativeTime(ticket.createdAt)}
            </p>
          </div>
          <div className="flex gap-2">
            <Button variant="outline" asChild>
              <a href={slackLink} target="_blank" rel="noopener noreferrer">
                <ExternalLink className="mr-2 h-4 w-4" />
                View in Slack
              </a>
            </Button>
            <Button>Edit Ticket</Button>
          </div>
        </div>

        <div className="grid gap-6 lg:grid-cols-3">
          <div className="lg:col-span-2 space-y-6">
            {/* Ticket Details */}
            <Card>
              <CardHeader>
                <CardTitle className="flex items-center gap-2">
                  <AlertCircle className="h-5 w-5" />
                  Ticket Information
                </CardTitle>
              </CardHeader>
              <CardContent className="space-y-4">
                <div className="grid grid-cols-2 gap-4">
                  <div>
                    <label className="text-sm font-medium">ID</label>
                    <p className="text-sm text-muted-foreground font-mono">{ticket.id}</p>
                  </div>
                  <div>
                    <label className="text-sm font-medium">Status</label>
                    <p className="text-sm">
                      <Badge className={TICKET_STATUS_COLORS[ticket.status as keyof typeof TICKET_STATUS_COLORS]}>
                        {TICKET_STATUS_LABELS[ticket.status as keyof typeof TICKET_STATUS_LABELS]}
                      </Badge>
                    </p>
                  </div>
                  <div>
                    <label className="text-sm font-medium">Created</label>
                    <p className="text-sm text-muted-foreground">{formatDateTime(ticket.createdAt)}</p>
                  </div>
                  <div>
                    <label className="text-sm font-medium">Last Updated</label>
                    <p className="text-sm text-muted-foreground">{formatDateTime(ticket.updatedAt)}</p>
                  </div>
                </div>
                <div>
                  <label className="text-sm font-medium">Assignee</label>
                  <div className="flex items-center gap-2 mt-1">
                    <User className="h-4 w-4 text-muted-foreground" />
                    <span className="text-sm text-muted-foreground">Unassigned</span>
                  </div>
                </div>
              </CardContent>
            </Card>

            {/* Comments */}
            <Card>
              <CardHeader>
                <CardTitle className="flex items-center gap-2">
                  <MessageSquare className="h-5 w-5" />
                  Comments ({ticket.comments.length})
                </CardTitle>
              </CardHeader>
              <CardContent>
                {ticket.comments.length === 0 ? (
                  <p className="text-sm text-muted-foreground">No comments yet.</p>
                ) : (
                  <div className="space-y-4">
                    {ticket.comments.map((comment) => (
                      <div key={comment.id} className="border rounded-lg p-4">
                        <div className="flex items-center justify-between mb-2">
                          <div className="flex items-center gap-2">
                            <User className="h-4 w-4" />
                            <span className="font-medium">System</span>
                          </div>
                          <span className="text-sm text-muted-foreground">
                            {formatRelativeTime(comment.createdAt)}
                          </span>
                        </div>
                        <p className="text-sm">{comment.content}</p>
                      </div>
                    ))}
                  </div>
                )}
              </CardContent>
            </Card>
          </div>

          <div className="space-y-6">
            {/* Related Alerts */}
            <Card>
              <CardHeader>
                <CardTitle className="flex items-center gap-2">
                  <AlertCircle className="h-5 w-5" />
                  Related Alerts ({ticket.alerts.length})
                </CardTitle>
              </CardHeader>
              <CardContent>
                {ticket.alerts.length === 0 ? (
                  <p className="text-sm text-muted-foreground">No related alerts.</p>
                ) : (
                  <div className="space-y-2">
                    {ticket.alerts.slice(0, 5).map((alert) => (
                      <div
                        key={alert.id}
                        className="p-3 border rounded-lg cursor-pointer hover:bg-muted/50 transition-colors"
                        onClick={() => setSelectedAlert(alert)}
                      >
                        <div className="flex items-start justify-between">
                          <div className="flex-1 min-w-0">
                            <h4 className="font-medium text-sm truncate">{alert.title}</h4>
                            <p className="text-xs text-muted-foreground">
                              {formatRelativeTime(alert.createdAt)}
                            </p>
                          </div>
                          <ExternalLink className="h-3 w-3 text-muted-foreground ml-2" />
                        </div>
                      </div>
                    ))}
                    {ticket.alerts.length > 5 && (
                      <Button variant="ghost" size="sm" className="w-full">
                        View all {ticket.alerts.length} alerts
                      </Button>
                    )}
                  </div>
                )}
              </CardContent>
            </Card>

            {/* Alert Detail Modal/Panel */}
            {selectedAlert && (
              <Card>
                <CardHeader className="flex flex-row items-center justify-between">
                  <CardTitle className="text-lg">Alert Details</CardTitle>
                  <Button 
                    variant="ghost" 
                    size="sm"
                    onClick={() => setSelectedAlert(null)}
                  >
                    ✕
                  </Button>
                </CardHeader>
                <CardContent className="space-y-4">
                  <div>
                    <label className="text-sm font-medium">ID</label>
                    <p className="text-sm text-muted-foreground font-mono">{selectedAlert.id}</p>
                  </div>
                  <div>
                    <label className="text-sm font-medium">Title</label>
                    <p className="text-sm">{selectedAlert.title}</p>
                  </div>
                  {selectedAlert.description && (
                    <div>
                      <label className="text-sm font-medium">Description</label>
                      <p className="text-sm text-muted-foreground">{selectedAlert.description}</p>
                    </div>
                  )}
                  <div>
                    <label className="text-sm font-medium">Created</label>
                    <p className="text-sm text-muted-foreground">
                      {formatDateTime(selectedAlert.createdAt)}
                    </p>
                  </div>
                  <Separator />
                  <Button variant="outline" size="sm" className="w-full" asChild>
                    <a 
                      href={generateAlertSlackLink(selectedAlert.id)} 
                      target="_blank" 
                      rel="noopener noreferrer"
                    >
                      <ExternalLink className="mr-2 h-4 w-4" />
                      View Alert in Slack
                    </a>
                  </Button>
                </CardContent>
              </Card>
            )}
          </div>
        </div>
      </div>
    </MainLayout>
  );
} 
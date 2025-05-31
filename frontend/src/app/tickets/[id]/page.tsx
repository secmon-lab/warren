'use client';

import { useQuery, useMutation } from '@apollo/client';
import { useParams } from 'next/navigation';
import { useState } from 'react';
import { MainLayout } from '@/components/layout/main-layout';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Dialog, DialogContent, DialogHeader, DialogTitle } from '@/components/ui/dialog';
import { Separator } from '@/components/ui/separator';
import { GET_TICKET, UPDATE_TICKET_STATUS } from '@/lib/graphql/queries';
import { Ticket, TicketStatus, TICKET_STATUS_LABELS, TICKET_STATUS_COLORS, Alert } from '@/lib/types';
import { formatRelativeTime } from '@/lib/utils-extended';
import { AlertCircle, MessageSquare, Calendar, User, Clock, FileText, Eye, Code, Database, Hash, ExternalLink } from 'lucide-react';

export default function TicketDetailPage() {
  const params = useParams();
  const ticketId = params.id as string;
  const [isUpdating, setIsUpdating] = useState(false);
  const [selectedAlert, setSelectedAlert] = useState<Alert | null>(null);

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

  const handleAlertClick = (alert: Alert) => {
    setSelectedAlert(alert);
  };

  const formatJsonData = (jsonString: string) => {
    try {
      const parsed = JSON.parse(jsonString);
      return JSON.stringify(parsed, null, 2);
    } catch {
      return jsonString;
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
                    <div 
                      key={alert.id} 
                      className="p-4 cursor-pointer hover:bg-muted/50 transition-colors"
                      onClick={() => handleAlertClick(alert)}
                    >
                      <div className="flex items-start gap-3">
                        <AlertCircle className="h-5 w-5 text-orange-500 mt-0.5" />
                        <div className="flex-1 min-w-0">
                          <h4 className="font-medium text-foreground hover:text-primary">
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
                            <Badge variant="outline" className="text-xs">
                              {alert.schema}
                            </Badge>
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

        {/* Alert Detail Dialog */}
        <Dialog open={!!selectedAlert} onOpenChange={() => setSelectedAlert(null)}>
          <DialogContent className="max-w-4xl max-h-[80vh] overflow-y-auto">
            {selectedAlert && (
              <>
                <DialogHeader>
                  <DialogTitle className="flex items-center gap-2">
                    <AlertCircle className="h-5 w-5 text-orange-500" />
                    Alert Details
                  </DialogTitle>
                </DialogHeader>
                
                <div className="space-y-6">
                  {/* Basic Information */}
                  <div className="grid grid-cols-2 gap-4">
                    <div className="space-y-2">
                      <label className="text-sm font-medium flex items-center gap-2">
                        <Hash className="h-4 w-4" />
                        ID
                      </label>
                      <p className="text-sm text-muted-foreground font-mono">{selectedAlert.id}</p>
                    </div>
                    <div className="space-y-2">
                      <label className="text-sm font-medium flex items-center gap-2">
                        <Clock className="h-4 w-4" />
                        Created At
                      </label>
                      <p className="text-sm text-muted-foreground">
                        {formatRelativeTime(selectedAlert.createdAt)}
                      </p>
                    </div>
                  </div>

                  <div className="space-y-2">
                    <label className="text-sm font-medium">Title</label>
                    <p className="text-sm">{selectedAlert.title}</p>
                  </div>

                  {selectedAlert.description && (
                    <div className="space-y-2">
                      <label className="text-sm font-medium">Description</label>
                      <p className="text-sm text-muted-foreground">{selectedAlert.description}</p>
                    </div>
                  )}

                  <Separator />

                  {/* Schema Information */}
                  <div className="space-y-2">
                    <label className="text-sm font-medium flex items-center gap-2">
                      <Code className="h-4 w-4" />
                      Schema
                    </label>
                    <Badge variant="outline" className="font-mono">
                      {selectedAlert.schema}
                    </Badge>
                  </div>

                  {/* Attributes */}
                  {selectedAlert.attributes && selectedAlert.attributes.length > 0 && (
                    <div className="space-y-2">
                      <label className="text-sm font-medium">Attributes</label>
                      <div className="grid grid-cols-1 md:grid-cols-2 gap-2">
                        {selectedAlert.attributes.map((attr, index) => (
                          <div key={index} className="p-3 border rounded-md">
                            <div className="flex items-center justify-between mb-1">
                              <span className="text-sm font-medium">{attr.key}</span>
                              {attr.auto && (
                                <Badge variant="secondary" className="text-xs">Auto</Badge>
                              )}
                            </div>
                            {attr.link ? (
                              <a 
                                href={attr.link} 
                                target="_blank" 
                                rel="noopener noreferrer"
                                className="text-sm text-blue-600 hover:text-blue-800 flex items-center gap-1"
                              >
                                {attr.value}
                                <ExternalLink className="h-3 w-3" />
                              </a>
                            ) : (
                              <span className="text-sm text-muted-foreground font-mono">
                                {attr.value}
                              </span>
                            )}
                          </div>
                        ))}
                      </div>
                    </div>
                  )}

                  <Separator />

                  {/* Raw Data */}
                  <div className="space-y-2">
                    <label className="text-sm font-medium flex items-center gap-2">
                      <Database className="h-4 w-4" />
                      Raw Data
                    </label>
                    <div className="bg-muted p-4 rounded-md">
                      <pre className="text-xs overflow-x-auto whitespace-pre-wrap">
                        {formatJsonData(selectedAlert.data)}
                      </pre>
                    </div>
                  </div>
                </div>
              </>
            )}
          </DialogContent>
        </Dialog>
      </div>
    </MainLayout>
  );
} 
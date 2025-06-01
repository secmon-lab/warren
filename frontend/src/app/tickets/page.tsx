'use client';

import { useQuery, useMutation } from '@apollo/client';
import { useState, Suspense } from 'react';
import { useSearchParams, useRouter } from 'next/navigation';
import { MainLayout } from '@/components/layout/main-layout';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { Button } from '@/components/ui/button';
import { Dialog, DialogContent, DialogHeader, DialogTitle } from '@/components/ui/dialog';
import { Separator } from '@/components/ui/separator';
import { Pagination, PaginationContent, PaginationItem, PaginationLink, PaginationNext, PaginationPrevious } from '@/components/ui/pagination';
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from '@/components/ui/collapsible';
import { UserWithAvatar } from '@/components/ui/user-name';
import { ResolveInfo } from '@/components/ui/resolve-info';
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger } from '@/components/ui/dropdown-menu';
import { GET_TICKETS, GET_TICKET, UPDATE_TICKET_STATUS } from '@/lib/graphql/queries';
import { Ticket, TicketStatus, TICKET_STATUS_LABELS, TICKET_STATUS_COLORS, Alert } from '@/lib/types';
import { formatRelativeTime } from '@/lib/utils-extended';
import { AlertCircle, MessageSquare, User, Ticket as TicketIcon, Calendar, Clock, FileText, Eye, Code, Database, Hash, ExternalLink, ChevronDown, ChevronUp, ArrowLeft, Archive, ArchiveRestore } from 'lucide-react';

const ITEMS_PER_PAGE = 10;
const ALERTS_PER_PAGE = 5;
const ALL_STATUSES: TicketStatus[] = ['open', 'pending', 'resolved', 'archived'];

function TicketsPageContent() {
  const [currentPage, setCurrentPage] = useState(1);
  const [selectedStatuses, setSelectedStatuses] = useState<TicketStatus[]>([]);
  const [activeTab, setActiveTab] = useState<'all' | TicketStatus>('all');
  const [selectedAlert, setSelectedAlert] = useState<Alert | null>(null);
  const [isSummaryOpen, setIsSummaryOpen] = useState(false);
  const [alertsCurrentPage, setAlertsCurrentPage] = useState(1);
  const [isUpdatingStatus, setIsUpdatingStatus] = useState(false);
  
  const searchParams = useSearchParams();
  const router = useRouter();
  const ticketId = searchParams.get('id');

  const [updateTicketStatus] = useMutation(UPDATE_TICKET_STATUS, {
    refetchQueries: [
      { query: GET_TICKET, variables: { id: ticketId } },
      { query: GET_TICKETS }
    ],
  });

  const { data: ticketsData, loading: ticketsLoading, error: ticketsError } = useQuery(GET_TICKETS, {
    variables: {
      statuses: selectedStatuses.length > 0 ? selectedStatuses : undefined,
      offset: (currentPage - 1) * ITEMS_PER_PAGE,
      limit: ITEMS_PER_PAGE,
    },
    skip: !!ticketId,
  });

  const { data: ticketData, loading: ticketLoading, error: ticketError } = useQuery(GET_TICKET, {
    variables: { id: ticketId },
    skip: !ticketId,
  });

  const ticket: Ticket = ticketData?.ticket;

  // Sort tickets by createdAt in descending order (newest first)
  const tickets: Ticket[] = [...(ticketsData?.tickets || [])].sort((a: Ticket, b: Ticket) => 
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

  const handleTicketClick = (ticketId: string) => {
    const params = new URLSearchParams();
    params.set('id', ticketId);
    router.push(`/tickets?${params.toString()}`);
  };

  const handleBackToList = () => {
    router.push('/tickets');
  };

  const handleAlertClick = (alert: Alert) => {
    setSelectedAlert(alert);
  };

  const handleStatusChange = async (newStatus: TicketStatus) => {
    if (!ticket || isUpdatingStatus) return;
    
    setIsUpdatingStatus(true);
    try {
      await updateTicketStatus({
        variables: {
          id: ticket.id,
          status: newStatus,
        },
      });
    } catch (error) {
      console.error('Failed to update ticket status:', error);
      alert('Failed to update ticket status');
    } finally {
      setIsUpdatingStatus(false);
    }
  };

  const handleArchive = async () => {
    if (!ticket || isUpdatingStatus) return;
    
    if (!confirm('Are you sure you want to archive this ticket? Archived tickets will be excluded from inspection and adaptation.')) {
      return;
    }
    
    setIsUpdatingStatus(true);
    try {
      await updateTicketStatus({
        variables: {
          id: ticket.id,
          status: 'archived',
        },
      });
    } catch (error) {
      console.error('Failed to archive ticket:', error);
      alert('Failed to archive ticket');
    } finally {
      setIsUpdatingStatus(false);
    }
  };

  const handleUnarchive = async () => {
    if (!ticket || isUpdatingStatus) return;
    
    if (!confirm('Are you sure you want to unarchive this ticket? It will be set to Open status.')) {
      return;
    }
    
    setIsUpdatingStatus(true);
    try {
      await updateTicketStatus({
        variables: {
          id: ticket.id,
          status: 'open',
        },
      });
    } catch (error) {
      console.error('Failed to unarchive ticket:', error);
      alert('Failed to unarchive ticket');
    } finally {
      setIsUpdatingStatus(false);
    }
  };

  const formatJsonData = (jsonString: string) => {
    try {
      const parsed = JSON.parse(jsonString);
      return JSON.stringify(parsed, null, 2);
    } catch {
      return jsonString;
    }
  };

  // If viewing a specific ticket
  if (ticketId) {
    if (ticketLoading) {
      return (
        <MainLayout>
          <div className="flex items-center justify-center h-64">
            <div className="text-lg">Loading ticket...</div>
          </div>
        </MainLayout>
      );
    }

    if (ticketError || !ticket) {
      return (
        <MainLayout>
          <div className="flex items-center justify-center h-64">
            <div className="text-lg text-red-600">
              {ticketError ? `Error loading ticket: ${ticketError.message}` : 'Ticket not found'}
            </div>
          </div>
        </MainLayout>
      );
    }

    // Paginate alerts
    const paginatedAlerts = ticket?.alerts ? ticket.alerts.slice(
      (alertsCurrentPage - 1) * ALERTS_PER_PAGE,
      alertsCurrentPage * ALERTS_PER_PAGE
    ) : [];

    const totalAlertsPages = ticket?.alerts ? Math.ceil(ticket.alerts.length / ALERTS_PER_PAGE) : 0;

    return (
      <MainLayout>
        <div className="space-y-6">
          {/* Header */}
          <div className="flex items-start justify-between">
            <div className="space-y-3 flex-1">
              <div className="flex items-center gap-2">
                <Button variant="ghost" size="sm" onClick={handleBackToList}>
                  <ArrowLeft className="h-4 w-4 mr-1" />
                  Back to tickets
                </Button>
                <Badge 
                  className={TICKET_STATUS_COLORS[ticket.status as TicketStatus]}
                  variant="secondary"
                >
                  {TICKET_STATUS_LABELS[ticket.status as TicketStatus]}
                </Badge>
              </div>
              <h1 className="text-3xl font-bold tracking-tight break-words" title={ticket.title || `Ticket ${ticket.id.slice(0, 8)}`}>
                {ticket.title || `Ticket ${ticket.id.slice(0, 8)}`}
              </h1>
              <p className="text-muted-foreground">
                #{ticket.id} • Created {formatRelativeTime(ticket.createdAt)} • 
                Updated {formatRelativeTime(ticket.updatedAt)}
              </p>
            </div>
          </div>

          <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
            {/* Main Content */}
            <div className="lg:col-span-2 space-y-6">
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

              {/* Summary Section - Collapsible */}
              {ticket.summary && (
                <Card>
                  <Collapsible open={isSummaryOpen} onOpenChange={setIsSummaryOpen}>
                    <CollapsibleTrigger asChild>
                      <CardHeader className="cursor-pointer hover:bg-muted/50 transition-colors">
                        <CardTitle className="flex items-center justify-between">
                          <div className="flex items-center gap-2">
                            <Eye className="h-5 w-5" />
                            Summary
                          </div>
                          {isSummaryOpen ? (
                            <ChevronUp className="h-4 w-4" />
                          ) : (
                            <ChevronDown className="h-4 w-4" />
                          )}
                        </CardTitle>
                      </CardHeader>
                    </CollapsibleTrigger>
                    <CollapsibleContent>
                      <CardContent>
                        <p className="text-sm leading-relaxed">
                          {ticket.summary}
                        </p>
                      </CardContent>
                    </CollapsibleContent>
                  </Collapsible>
                </Card>
              )}

              {/* Resolve Information Section */}
              <ResolveInfo ticket={ticket} />

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

              {/* Alerts Section with Pagination */}
              <Card>
                <CardHeader>
                  <CardTitle className="flex items-center gap-2">
                    <AlertCircle className="h-5 w-5" />
                    Related Alerts ({ticket.alerts.length})
                  </CardTitle>
                </CardHeader>
                <CardContent className="p-0">
                  <div className="divide-y">
                    {paginatedAlerts.map((alert) => (
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
                  
                  {/* Alerts Pagination */}
                  {totalAlertsPages > 1 && (
                    <div className="p-4 border-t">
                      <Pagination>
                        <PaginationContent>
                          <PaginationItem>
                            <PaginationPrevious 
                              href="#" 
                              onClick={(e) => {
                                e.preventDefault();
                                if (alertsCurrentPage > 1) setAlertsCurrentPage(alertsCurrentPage - 1);
                              }}
                            />
                          </PaginationItem>
                          
                          {/* Show page numbers */}
                          {Array.from({ length: totalAlertsPages }, (_, i) => i + 1).map((page) => (
                            <PaginationItem key={page}>
                              <PaginationLink 
                                href="#" 
                                isActive={page === alertsCurrentPage}
                                onClick={(e) => {
                                  e.preventDefault();
                                  setAlertsCurrentPage(page);
                                }}
                              >
                                {page}
                              </PaginationLink>
                            </PaginationItem>
                          ))}
                          
                          <PaginationItem>
                            <PaginationNext 
                              href="#"
                              onClick={(e) => {
                                e.preventDefault();
                                if (alertsCurrentPage < totalAlertsPages) setAlertsCurrentPage(alertsCurrentPage + 1);
                              }}
                            />
                          </PaginationItem>
                        </PaginationContent>
                      </Pagination>
                    </div>
                  )}
                </CardContent>
              </Card>
            </div>

            {/* Sidebar */}
            <div className="space-y-6">
              {/* Details & Status Management */}
              <Card>
                <CardHeader>
                  <CardTitle>Details</CardTitle>
                </CardHeader>
                <CardContent className="space-y-4">
                  <div className="flex items-center gap-2 text-sm">
                    <User className="h-4 w-4 text-muted-foreground" />
                    <span className="text-muted-foreground">Assignee:</span>
                    {ticket.assignee ? (
                      <UserWithAvatar 
                        userID={ticket.assignee.id} 
                        fallback={ticket.assignee.name}
                        avatarSize="sm"
                      />
                    ) : (
                      <span>Unassigned</span>
                    )}
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

                  <Separator />

                  {ticket.status === 'archived' ? (
                    <div className="space-y-2">
                      <label className="text-sm font-medium">Status</label>
                      <div className="w-full p-2">
                        <Badge 
                          className={TICKET_STATUS_COLORS[ticket.status as TicketStatus]}
                          variant="secondary"
                        >
                          {TICKET_STATUS_LABELS[ticket.status as TicketStatus]}
                        </Badge>
                      </div>
                      <Button 
                        variant="secondary" 
                        size="sm"
                        className="w-full text-muted-foreground hover:text-foreground hover:bg-secondary/80 transition-colors cursor-pointer"
                        onClick={handleUnarchive}
                        disabled={isUpdatingStatus}
                      >
                        <ArchiveRestore className="h-4 w-4 mr-2" />
                        Unarchive
                      </Button>
                    </div>
                  ) : (
                    <>
                      <div className="space-y-2">
                        <label className="text-sm font-medium">Status</label>
                        <DropdownMenu>
                          <DropdownMenuTrigger asChild>
                            <Button 
                              variant="outline" 
                              className="w-full justify-between"
                              disabled={isUpdatingStatus}
                            >
                              <Badge 
                                className={TICKET_STATUS_COLORS[ticket.status as TicketStatus]}
                                variant="secondary"
                              >
                                {TICKET_STATUS_LABELS[ticket.status as TicketStatus]}
                              </Badge>
                              <ChevronDown className="h-4 w-4" />
                            </Button>
                          </DropdownMenuTrigger>
                          <DropdownMenuContent className="w-full">
                            {(['open', 'pending', 'resolved'] as TicketStatus[]).map((status) => (
                              <DropdownMenuItem
                                key={status}
                                onClick={() => handleStatusChange(status)}
                                disabled={ticket.status === status || isUpdatingStatus}
                              >
                                <span className="flex items-center gap-2">
                                  <Badge 
                                    className={TICKET_STATUS_COLORS[status]}
                                    variant="secondary"
                                  >
                                    {TICKET_STATUS_LABELS[status]}
                                  </Badge>
                                  {ticket.status === status && <span className="text-xs text-muted-foreground">(current)</span>}
                                </span>
                              </DropdownMenuItem>
                            ))}
                          </DropdownMenuContent>
                        </DropdownMenu>
                      </div>

                      <div className="space-y-2">
                        <Button 
                          variant="secondary" 
                          size="sm"
                          className="w-full text-muted-foreground hover:text-foreground hover:bg-secondary/80 transition-colors cursor-pointer"
                          onClick={handleArchive}
                          disabled={isUpdatingStatus}
                        >
                          <Archive className="h-4 w-4 mr-2" />
                          Mark as Archived
                        </Button>
                      </div>
                    </>
                  )}
                </CardContent>
              </Card>
            </div>
          </div>

          {/* Alert Detail Dialog */}
          <Dialog open={!!selectedAlert} onOpenChange={() => setSelectedAlert(null)}>
            <DialogContent className="w-[95vw] max-w-[95vw] sm:max-w-[95vw] md:max-w-[90vw] lg:max-w-[1200px] max-h-[80vh] overflow-y-auto">
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

  // Normal tickets list view
  if (ticketsLoading) {
    return (
      <MainLayout>
        <div className="flex items-center justify-center h-64">
          <div className="text-lg">Loading tickets...</div>
        </div>
      </MainLayout>
    );
  }

  if (ticketsError) {
    return (
      <MainLayout>
        <div className="flex items-center justify-center h-64">
          <div className="text-lg text-red-600">Error loading tickets: {ticketsError.message}</div>
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
                    <div 
                      key={ticket.id} 
                      className="p-3 hover:bg-muted/50 transition-colors cursor-pointer"
                      onClick={() => handleTicketClick(ticket.id)}
                    >
                      <div className="flex items-start gap-3">
                        <TicketIcon className="h-4 w-4 text-muted-foreground mt-0.5" />
                        <div className="flex-1 min-w-0">
                          <div className="flex items-center gap-2 mb-1">
                            <Badge 
                              className={TICKET_STATUS_COLORS[ticket.status as TicketStatus]}
                              variant="secondary"
                            >
                              {TICKET_STATUS_LABELS[ticket.status as TicketStatus]}
                            </Badge>
                          </div>
                          <h3 className="font-medium text-foreground hover:text-primary mb-1 truncate" title={ticket.title || `Ticket ${ticket.id.slice(0, 8)}`}>
                            {ticket.title || `Ticket ${ticket.id.slice(0, 8)}`}
                          </h3>
                          <div className="flex items-center gap-4 text-sm text-muted-foreground">
                            <span>opened {formatRelativeTime(ticket.createdAt)}</span>
                            <div className="flex items-center gap-1">
                              <User className="h-3 w-3" />
                              {ticket.assignee ? (
                                <UserWithAvatar 
                                  userID={ticket.assignee.id} 
                                  fallback={ticket.assignee.name}
                                  avatarSize="sm"
                                />
                              ) : (
                                <span>Unassigned</span>
                              )}
                            </div>
                            <div className="flex items-center gap-1">
                              <MessageSquare className="h-3 w-3" />
                              <span>{ticket.comments.length}</span>
                            </div>
                            <div className="flex items-center gap-1">
                              <AlertCircle className="h-3 w-3" />
                              <span>{ticket.alerts.length}</span>
                            </div>
                          </div>
                        </div>
                      </div>
                    </div>
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

export default function TicketsPage() {
  return (
    <Suspense fallback={
      <MainLayout>
        <div className="flex items-center justify-center h-64">
          <div className="text-lg">Loading...</div>
        </div>
      </MainLayout>
    }>
      <TicketsPageContent />
    </Suspense>
  );
} 
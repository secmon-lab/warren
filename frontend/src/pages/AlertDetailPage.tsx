import { useQuery, useMutation } from "@apollo/client";
import { useParams, useNavigate } from "react-router-dom";
import { useState } from "react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Separator } from "@/components/ui/separator";
import { Label } from "@/components/ui/label";
import { Slider } from "@/components/ui/slider";
import {
  Pagination,
  PaginationContent,
  PaginationItem,
  PaginationLink,
  PaginationNext,
  PaginationPrevious,
} from "@/components/ui/pagination";
import { useToast } from "@/hooks/use-toast";

import { GET_ALERT, CREATE_TICKET_FROM_ALERTS, GET_SIMILAR_TICKETS_FOR_ALERT, BIND_ALERTS_TO_TICKET } from "@/lib/graphql/queries";
import { Alert, Ticket, TICKET_STATUS_COLORS, TICKET_STATUS_LABELS, TicketStatus } from "@/lib/types";
import { formatRelativeTime } from "@/lib/utils-extended";
import {
  ChevronLeft,
  AlertTriangle,
  ExternalLink,
  Eye,
  Copy,
  Database,
  Link2,
  Tag,
  Plus,
  Hash,
  Users,
} from "lucide-react";

export default function AlertDetailPage() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const { toast } = useToast();
  const [isCreatingTicket, setIsCreatingTicket] = useState(false);
  const [isBindingTicket, setIsBindingTicket] = useState(false);
  const [similarTicketsThreshold, setSimilarTicketsThreshold] = useState([0.95]);
  const [similarTicketsCommittedThreshold, setSimilarTicketsCommittedThreshold] = useState([0.95]);
  const [similarTicketsCurrentPage, setSimilarTicketsCurrentPage] = useState(1);

  const ITEMS_PER_PAGE = 5;
  const similarTicketsOffset = (similarTicketsCurrentPage - 1) * ITEMS_PER_PAGE;

  const {
    data: alertData,
    loading: alertLoading,
    error: alertError,
  } = useQuery(GET_ALERT, {
    variables: { id },
    skip: !id,
  });

  const alert: Alert = alertData?.alert;

  const {
    data: similarTicketsData,
    loading: similarTicketsLoading,
    error: similarTicketsError,
    refetch: refetchSimilarTickets,
  } = useQuery(GET_SIMILAR_TICKETS_FOR_ALERT, {
    variables: {
      alertId: alert?.id,
      threshold: similarTicketsCommittedThreshold[0],
      offset: similarTicketsOffset,
      limit: ITEMS_PER_PAGE,
    },
    skip: !alert?.id,
    fetchPolicy: "cache-and-network",
  });

  const [createTicketFromAlerts] = useMutation(CREATE_TICKET_FROM_ALERTS, {
    onCompleted: (data) => {
      toast({
        title: "Ticket Created",
        description: `Ticket "${data.createTicketFromAlerts.title}" has been created successfully.`,
      });
      // Navigate to the new ticket
      navigate(`/tickets/${data.createTicketFromAlerts.id}`);
    },
    onError: (error) => {
      toast({
        title: "Error",
        description: `Failed to create ticket: ${error.message}`,
        variant: "destructive",
      });
      setIsCreatingTicket(false);
    },
  });

  const [bindAlertsToTicket] = useMutation(BIND_ALERTS_TO_TICKET, {
    onCompleted: (data) => {
      toast({
        title: "Alert Bound",
        description: `Alert has been bound to ticket "${data.bindAlertsToTicket.title}".`,
      });
      // Navigate to the ticket
      navigate(`/tickets/${data.bindAlertsToTicket.id}`);
    },
    onError: (error) => {
      toast({
        title: "Error",
        description: `Failed to bind alert: ${error.message}`,
        variant: "destructive",
      });
      setIsBindingTicket(false);
    },
  });

  const handleBackToList = () => {
    navigate("/alerts");
  };

  const handleCreateTicket = async () => {
    if (!alert?.id) return;

    setIsCreatingTicket(true);
    try {
      await createTicketFromAlerts({
        variables: {
          alertIds: [alert.id],
        },
      });
    } catch (error) {
      // Error handling is done in the onError callback
      console.error("Error creating ticket:", error);
    }
  };

  const handleBindToTicket = async (ticketId: string) => {
    if (!alert?.id) return;

    setIsBindingTicket(true);
    try {
      await bindAlertsToTicket({
        variables: {
          ticketId,
          alertIds: [alert.id],
        },
      });
    } catch (error) {
      console.error("Error binding alert to ticket:", error);
    }
  };

  const handleCopyToClipboard = async (text: string) => {
    try {
      await navigator.clipboard.writeText(text);
    } catch (error) {
      console.error("Failed to copy to clipboard:", error);
    }
  };

  const handleSimilarTicketsThresholdChange = (value: number[]) => {
    setSimilarTicketsCommittedThreshold(value);
    setSimilarTicketsCurrentPage(1); // Reset to first page when threshold changes
    // Trigger refetch with new threshold
    if (alert?.id) {
      refetchSimilarTickets({
        alertId: alert.id,
        threshold: value[0],
        offset: 0,
        limit: ITEMS_PER_PAGE,
      });
    }
  };

  const handleSimilarTicketClick = (ticket: Ticket) => {
    navigate(`/tickets/${ticket.id}`);
  };

  const handleSimilarTicketsPageChange = (page: number) => {
    setSimilarTicketsCurrentPage(page);
  };

  const formatAbsoluteTime = (dateString: string) => {
    const date = new Date(dateString);
    const year = date.getFullYear();
    const month = String(date.getMonth() + 1).padStart(2, "0");
    const day = String(date.getDate()).padStart(2, "0");
    const hours = String(date.getHours()).padStart(2, "0");
    const minutes = String(date.getMinutes()).padStart(2, "0");
    const seconds = String(date.getSeconds()).padStart(2, "0");

    // Get timezone offset and format it as +09:00 style
    const timezoneOffset = -date.getTimezoneOffset();
    const offsetHours = Math.floor(Math.abs(timezoneOffset) / 60);
    const offsetMinutes = Math.round(Math.abs(timezoneOffset) % 60);
    const offsetSign = timezoneOffset >= 0 ? "+" : "-";
    const timezone = `${offsetSign}${String(offsetHours).padStart(
      2,
      "0"
    )}:${String(offsetMinutes).padStart(2, "0")}`;

    return `${year}-${month}-${day} ${hours}:${minutes}:${seconds} ${timezone}`;
  };

  const parseAlertData = (dataString: string) => {
    try {
      return JSON.parse(dataString);
    } catch {
      return dataString;
    }
  };

  if (alertLoading) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="text-lg">Loading alert...</div>
      </div>
    );
  }

  if (alertError) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="text-lg text-red-600">
          Error loading alert: {alertError.message}
        </div>
      </div>
    );
  }

  if (!alert) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="text-lg">Alert not found</div>
      </div>
    );
  }

  const parsedData = parseAlertData(alert.data);

  // Similar tickets data
  const similarTickets: Ticket[] = similarTicketsData?.similarTicketsForAlert?.tickets || [];
  const similarTicketsTotalCount = similarTicketsData?.similarTicketsForAlert?.totalCount || 0;
  const similarTicketsTotalPages = Math.ceil(similarTicketsTotalCount / ITEMS_PER_PAGE);

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div className="flex items-center space-x-4">
          <Button
            variant="ghost"
            size="sm"
            onClick={handleBackToList}
            className="flex items-center space-x-2"
          >
            <ChevronLeft className="h-4 w-4" />
            <span>Back to Alerts</span>
          </Button>
        </div>
      </div>

      {/* Alert Header Card */}
      <Card>
        <CardHeader>
          <div className="flex items-center gap-2 mb-2">
            <AlertTriangle className="h-5 w-5 text-orange-500" />
            <Badge variant="outline">
              {alert.schema}
            </Badge>
            {alert.ticket ? (
              <Badge variant="secondary">
                Assigned to Ticket
              </Badge>
            ) : (
              <Badge variant="outline">
                Unassigned
              </Badge>
            )}
          </div>
          <CardTitle className="text-2xl mb-2">
            {alert.title}
          </CardTitle>
          {alert.description && (
            <p className="text-muted-foreground mb-4">
              {alert.description}
            </p>
          )}
        </CardHeader>
      </Card>

      {/* Alert Details */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        {/* Main Content */}
        <div className="lg:col-span-2 space-y-6">
          {/* Alert Attributes */}
          {alert.attributes && alert.attributes.length > 0 && (
            <Card>
              <CardHeader>
                <CardTitle className="flex items-center gap-2">
                  <Tag className="h-4 w-4" />
                  Attributes
                </CardTitle>
              </CardHeader>
              <CardContent>
                <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
                  {alert.attributes.map((attr, index) => (
                    <div 
                      key={index}
                      className="p-3 bg-muted rounded-lg"
                    >
                      <div className="flex items-start justify-between mb-1">
                        <span className="font-medium text-sm">
                          {attr.key}
                        </span>
                        {attr.auto && (
                          <Badge variant="secondary" className="text-xs">
                            Auto
                          </Badge>
                        )}
                      </div>
                      <div className="flex items-center gap-2">
                        {attr.link ? (
                          <a
                            href={attr.link}
                            target="_blank"
                            rel="noopener noreferrer"
                            className="text-blue-600 hover:text-blue-800 text-sm break-all flex items-center gap-1"
                          >
                            {attr.value}
                            <ExternalLink className="h-3 w-3 flex-shrink-0" />
                          </a>
                        ) : (
                          <span className="text-sm text-muted-foreground break-all">
                            {attr.value}
                          </span>
                        )}
                        <Button
                          variant="ghost"
                          size="sm"
                          onClick={() => handleCopyToClipboard(attr.value)}
                          className="h-6 w-6 p-0 flex-shrink-0"
                        >
                          <Copy className="h-3 w-3" />
                        </Button>
                      </div>
                    </div>
                  ))}
                </div>
              </CardContent>
            </Card>
          )}

          {/* Alert Data */}
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <Database className="h-4 w-4" />
                Raw Data
              </CardTitle>
            </CardHeader>
            <CardContent>
              <div className="relative">
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={() => handleCopyToClipboard(alert.data)}
                  className="absolute top-2 right-2 h-8 w-8 p-0"
                >
                  <Copy className="h-3 w-3" />
                </Button>
                <pre className="bg-muted p-4 rounded-lg text-xs overflow-auto max-h-96">
                  {typeof parsedData === 'object' 
                    ? JSON.stringify(parsedData, null, 2)
                    : alert.data
                  }
                </pre>
              </div>
            </CardContent>
          </Card>
        </div>

        {/* Sidebar */}
        <div className="space-y-6">
          {/* Ticket Information */}
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <Link2 className="h-4 w-4" />
                {alert.ticket ? "Associated Ticket" : "Ticket"}
              </CardTitle>
            </CardHeader>
            <CardContent>
              {alert.ticket ? (
                <div className="space-y-3">
                  <div>
                    <span className="text-sm font-medium">ID</span>
                    <p className="text-sm text-muted-foreground font-mono">
                      {alert.ticket.id}
                    </p>
                  </div>
                  <div>
                    <span className="text-sm font-medium">Title</span>
                    <p className="text-sm text-muted-foreground">
                      {alert.ticket.title}
                    </p>
                  </div>
                  <div>
                    <span className="text-sm font-medium">Status</span>
                    <div className="mt-1">
                      <Badge variant="outline">
                        {alert.ticket.status}
                      </Badge>
                    </div>
                  </div>
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => navigate(`/tickets/${alert.ticket!.id}`)}
                    className="w-full"
                  >
                    View Ticket
                  </Button>
                </div>
              ) : (
                <div className="space-y-3">
                  <p className="text-sm text-muted-foreground">
                    This alert is not associated with any ticket.
                  </p>
                  <Button
                    onClick={handleCreateTicket}
                    disabled={isCreatingTicket}
                    className="w-full flex items-center gap-2"
                  >
                    <Plus className="h-4 w-4" />
                    {isCreatingTicket ? "Creating..." : "Create Ticket"}
                  </Button>
                </div>
              )}
            </CardContent>
          </Card>

          {/* Alert Metadata */}
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <Eye className="h-4 w-4" />
                Metadata
              </CardTitle>
            </CardHeader>
            <CardContent className="space-y-3">
              <div>
                <span className="text-sm font-medium">Alert ID</span>
                <div className="flex items-start gap-2 mt-1">
                  <p className="text-sm text-muted-foreground font-mono break-all flex-1">
                    {alert.id}
                  </p>
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={() => handleCopyToClipboard(alert.id)}
                    className="h-6 w-6 p-0 flex-shrink-0"
                  >
                    <Copy className="h-3 w-3" />
                  </Button>
                </div>
              </div>
              <Separator />
              <div>
                <span className="text-sm font-medium">Schema</span>
                <div className="flex items-center gap-2 mt-1">
                  <Badge variant="outline" className="font-mono">
                    {alert.schema}
                  </Badge>
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={() => handleCopyToClipboard(alert.schema)}
                    className="h-6 w-6 p-0"
                  >
                    <Copy className="h-3 w-3" />
                  </Button>
                </div>
              </div>
              <Separator />
              <div>
                <span className="text-sm font-medium">Created</span>
                <p className="text-sm text-muted-foreground">
                  {formatAbsoluteTime(alert.createdAt)}
                </p>
                <p className="text-xs text-muted-foreground">
                  {formatRelativeTime(alert.createdAt)}
                </p>
              </div>
            </CardContent>
          </Card>

          {/* Similar Tickets */}
          {!alert.ticket && (
            <Card>
              <CardHeader>
                <CardTitle className="flex items-center gap-2">
                  <Hash className="h-5 w-5" />
                  Similar Tickets
                </CardTitle>
              </CardHeader>
              <CardContent className="space-y-4">
                {/* Threshold Slider */}
                <div className="space-y-3">
                  <div className="flex items-center justify-between">
                    <Label htmlFor="threshold" className="text-sm font-medium">
                      Similarity Threshold
                    </Label>
                    <span className="text-sm text-muted-foreground">
                      {similarTicketsThreshold[0].toFixed(2)}
                    </span>
                  </div>
                  <Slider
                    id="threshold"
                    min={0}
                    max={1}
                    step={0.01}
                    value={similarTicketsThreshold}
                    onValueChange={setSimilarTicketsThreshold}
                    onValueCommit={handleSimilarTicketsThresholdChange}
                    className="w-full"
                  />
                  <div className="flex justify-between text-xs text-muted-foreground">
                    <span>0.0 (Less similar)</span>
                    <span>1.0 (More similar)</span>
                  </div>
                </div>

                {/* Loading State */}
                {similarTicketsLoading && (
                  <div className="flex items-center justify-center py-8">
                    <div className="text-sm text-muted-foreground">Loading similar tickets...</div>
                  </div>
                )}

                {/* Error State */}
                {similarTicketsError && (
                  <div className="flex items-center justify-center py-8">
                    <div className="text-sm text-red-600">
                      Error loading similar tickets: {similarTicketsError.message}
                    </div>
                  </div>
                )}

                {/* Empty State */}
                {!similarTicketsLoading && !similarTicketsError && similarTickets.length === 0 && (
                  <div className="flex items-center justify-center py-8">
                    <div className="text-center">
                      <Users className="h-8 w-8 mx-auto mb-2 text-muted-foreground" />
                      <div className="text-sm text-muted-foreground">
                        No similar tickets found with threshold {similarTicketsCommittedThreshold[0].toFixed(2)}
                      </div>
                    </div>
                  </div>
                )}

                {/* Tickets List */}
                {!similarTicketsLoading && !similarTicketsError && similarTickets.length > 0 && (
                  <div className="space-y-3">
                    {similarTickets.map((ticket) => (
                      <div
                        key={ticket.id}
                        className="p-3 border rounded-lg cursor-pointer hover:bg-muted/50 transition-colors"
                        onClick={() => handleSimilarTicketClick(ticket)}
                      >
                        <div className="flex items-start justify-between mb-2">
                          <div className="flex-1 min-w-0">
                            <h4 className="font-medium text-sm truncate">
                              {ticket.isTest && "🧪 [TEST] "}
                              {ticket.title}
                            </h4>
                            <p className="text-xs text-muted-foreground mt-1">
                              #{ticket.id.slice(0, 8)} • Created {formatRelativeTime(ticket.createdAt)}
                            </p>
                          </div>
                          <div className="flex items-center gap-2">
                            <Badge
                              className={TICKET_STATUS_COLORS[ticket.status as TicketStatus]}
                              variant="secondary"
                            >
                              {TICKET_STATUS_LABELS[ticket.status as TicketStatus]}
                            </Badge>
                            <Button
                              variant="outline"
                              size="sm"
                              onClick={(e) => {
                                e.stopPropagation();
                                handleBindToTicket(ticket.id);
                              }}
                              disabled={isBindingTicket}
                              className="flex-shrink-0"
                            >
                              Bind
                            </Button>
                          </div>
                        </div>
                        
                        {ticket.description && (
                          <p className="text-xs text-muted-foreground line-clamp-2">
                            {ticket.description}
                          </p>
                        )}
                        
                        {ticket.assignee && (
                          <div className="flex items-center gap-1 mt-2">
                            <Users className="h-3 w-3 text-muted-foreground" />
                            <span className="text-xs text-muted-foreground">
                              {ticket.assignee.name}
                            </span>
                          </div>
                        )}
                      </div>
                    ))}
                  </div>
                )}

                {/* Pagination */}
                {!similarTicketsLoading && !similarTicketsError && similarTicketsTotalPages > 1 && (
                  <div className="pt-4">
                    <Pagination>
                      <PaginationContent>
                        <PaginationItem>
                          <PaginationPrevious
                            onClick={() => {
                              if (similarTicketsCurrentPage > 1) handleSimilarTicketsPageChange(similarTicketsCurrentPage - 1);
                            }}
                            className={
                              similarTicketsCurrentPage === 1
                                ? "pointer-events-none opacity-50"
                                : "cursor-pointer"
                            }
                          />
                        </PaginationItem>
                        
                        {/* Show page numbers with truncation */}
                        {(() => {
                          const maxVisiblePages = 10;
                          const pageNumbers = [];
                          
                          if (similarTicketsTotalPages <= maxVisiblePages) {
                            // Show all pages if total is 10 or less
                            for (let i = 1; i <= similarTicketsTotalPages; i++) {
                              pageNumbers.push(i);
                            }
                          } else {
                            // Show truncated pagination for more than 10 pages
                            const startPage = Math.max(1, similarTicketsCurrentPage - 4);
                            const endPage = Math.min(similarTicketsTotalPages, similarTicketsCurrentPage + 4);
                            
                            // Always show first page
                            if (startPage > 1) {
                              pageNumbers.push(1);
                              if (startPage > 2) {
                                pageNumbers.push('...');
                              }
                            }
                            
                            // Show pages around current page
                            for (let i = startPage; i <= endPage; i++) {
                              pageNumbers.push(i);
                            }
                            
                            // Always show last page
                            if (endPage < similarTicketsTotalPages) {
                              if (endPage < similarTicketsTotalPages - 1) {
                                pageNumbers.push('...');
                              }
                              pageNumbers.push(similarTicketsTotalPages);
                            }
                          }
                          
                          return pageNumbers.map((page, index) => (
                            <PaginationItem key={index}>
                              {page === '...' ? (
                                <span className="px-3 py-2 text-sm text-muted-foreground">...</span>
                              ) : (
                                <PaginationLink
                                  isActive={page === similarTicketsCurrentPage}
                                  onClick={() => handleSimilarTicketsPageChange(page as number)}
                                  className="cursor-pointer"
                                >
                                  {page}
                                </PaginationLink>
                              )}
                            </PaginationItem>
                          ));
                        })()}

                        <PaginationItem>
                          <PaginationNext
                            onClick={() => {
                              if (similarTicketsCurrentPage < similarTicketsTotalPages) handleSimilarTicketsPageChange(similarTicketsCurrentPage + 1);
                            }}
                            className={
                              similarTicketsCurrentPage === similarTicketsTotalPages
                                ? "pointer-events-none opacity-50"
                                : "cursor-pointer"
                            }
                          />
                        </PaginationItem>
                      </PaginationContent>
                    </Pagination>

                    <div className="text-center mt-2">
                      <span className="text-xs text-muted-foreground">
                        Showing {Math.min((similarTicketsCurrentPage - 1) * ITEMS_PER_PAGE + 1, similarTicketsTotalCount)}-
                        {Math.min(similarTicketsCurrentPage * ITEMS_PER_PAGE, similarTicketsTotalCount)} of {similarTicketsTotalCount} tickets
                      </span>
                    </div>
                  </div>
                )}
              </CardContent>
            </Card>
          )}
        </div>
      </div>
    </div>
  );
} 
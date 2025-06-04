import { useQuery, useMutation } from "@apollo/client";
import { useState, useEffect } from "react";
import { useParams, useNavigate } from "react-router-dom";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Separator } from "@/components/ui/separator";
import {
  Pagination,
  PaginationContent,
  PaginationItem,
  PaginationLink,
  PaginationNext,
  PaginationPrevious,
} from "@/components/ui/pagination";
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from "@/components/ui/collapsible";
import { UserWithAvatar } from "@/components/ui/user-name";
import { ResolveInfo } from "@/components/ui/resolve-info";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { GET_TICKET, UPDATE_TICKET_STATUS } from "@/lib/graphql/queries";
import {
  Ticket,
  TicketStatus,
  TICKET_STATUS_LABELS,
  TICKET_STATUS_COLORS,
  Alert,
} from "@/lib/types";
import { formatRelativeTime } from "@/lib/utils-extended";
import {
  AlertCircle,
  MessageSquare,
  User,
  Calendar,
  Clock,
  FileText,
  Eye,
  Hash,
  ChevronDown,
  ChevronUp,
  ArrowLeft,
  Archive,
  ArchiveRestore,
} from "lucide-react";

const ALERTS_PER_PAGE = 5;

export default function TicketDetailPage() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const [selectedAlert, setSelectedAlert] = useState<Alert | null>(null);
  const [isSummaryOpen, setIsSummaryOpen] = useState(false);
  const [alertsCurrentPage, setAlertsCurrentPage] = useState(1);
  const [isUpdatingStatus, setIsUpdatingStatus] = useState(false);

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

  // Reset other state when ticket changes
  useEffect(() => {
    if (id) {
      setAlertsCurrentPage(1);
      setSelectedAlert(null);
      setIsSummaryOpen(false);
    }
  }, [id]);

  const [updateTicketStatus] = useMutation(UPDATE_TICKET_STATUS, {
    refetchQueries: [{ query: GET_TICKET, variables: { id } }],
  });

  const {
    data: ticketData,
    loading: ticketLoading,
    error: ticketError,
  } = useQuery(GET_TICKET, {
    variables: { id },
    skip: !id,
  });

  const ticket: Ticket = ticketData?.ticket;

  const handleBackToList = () => {
    navigate("/tickets");
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
      console.error("Failed to update ticket status:", error);
      alert("Failed to update ticket status");
    } finally {
      setIsUpdatingStatus(false);
    }
  };

  const handleArchive = async () => {
    if (!ticket || isUpdatingStatus) return;

    if (!confirm("Are you sure you want to archive this ticket?")) {
      return;
    }

    setIsUpdatingStatus(true);
    try {
      await updateTicketStatus({
        variables: {
          id: ticket.id,
          status: "archived",
        },
      });
    } catch (error) {
      console.error("Failed to archive ticket:", error);
      alert("Failed to archive ticket");
    } finally {
      setIsUpdatingStatus(false);
    }
  };

  const handleUnarchive = async () => {
    if (!ticket || isUpdatingStatus) return;

    if (!confirm("Are you sure you want to unarchive this ticket?")) {
      return;
    }

    setIsUpdatingStatus(true);
    try {
      await updateTicketStatus({
        variables: {
          id: ticket.id,
          status: "open",
        },
      });
    } catch (error) {
      console.error("Failed to unarchive ticket:", error);
      alert("Failed to unarchive ticket");
    } finally {
      setIsUpdatingStatus(false);
    }
  };

  if (!id) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="text-lg text-red-600">No ticket ID provided</div>
      </div>
    );
  }

  if (ticketLoading) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="text-lg">Loading ticket...</div>
      </div>
    );
  }

  if (ticketError || !ticket) {
    return (
      <div className="space-y-4">
        <div className="flex items-center justify-center h-64">
          <div className="text-lg text-red-600">
            Error loading ticket: {ticketError?.message || "Ticket not found"}
          </div>
        </div>
        <div className="flex justify-center">
          <Button variant="outline" onClick={handleBackToList}>
            <ArrowLeft className="h-4 w-4 mr-2" />
            Back to tickets
          </Button>
        </div>
      </div>
    );
  }

  // Paginate alerts
  const paginatedAlerts = ticket?.alerts
    ? ticket.alerts.slice(
        (alertsCurrentPage - 1) * ALERTS_PER_PAGE,
        alertsCurrentPage * ALERTS_PER_PAGE
      )
    : [];

  const totalAlertsPages = ticket?.alerts
    ? Math.ceil(ticket.alerts.length / ALERTS_PER_PAGE)
    : 0;

  return (
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
              variant="secondary">
              {TICKET_STATUS_LABELS[ticket.status as TicketStatus]}
            </Badge>
          </div>
          <h1
            className="text-3xl font-bold tracking-tight break-words"
            title={ticket.title || `Ticket ${ticket.id.slice(0, 8)}`}>
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
                    <p className="text-sm leading-relaxed">{ticket.summary}</p>
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
                    onClick={() => handleAlertClick(alert)}>
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
                          <span>
                            created {formatRelativeTime(alert.createdAt)}
                          </span>
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
                          onClick={() => {
                            if (alertsCurrentPage > 1)
                              setAlertsCurrentPage(alertsCurrentPage - 1);
                          }}
                          className={
                            alertsCurrentPage === 1
                              ? "pointer-events-none opacity-50"
                              : "cursor-pointer"
                          }
                        />
                      </PaginationItem>

                      {/* Show page numbers */}
                      {Array.from(
                        { length: totalAlertsPages },
                        (_, i) => i + 1
                      ).map((page) => (
                        <PaginationItem key={page}>
                          <PaginationLink
                            isActive={page === alertsCurrentPage}
                            onClick={() => setAlertsCurrentPage(page)}
                            className="cursor-pointer">
                            {page}
                          </PaginationLink>
                        </PaginationItem>
                      ))}

                      <PaginationItem>
                        <PaginationNext
                          onClick={() => {
                            if (alertsCurrentPage < totalAlertsPages)
                              setAlertsCurrentPage(alertsCurrentPage + 1);
                          }}
                          className={
                            alertsCurrentPage === totalAlertsPages
                              ? "pointer-events-none opacity-50"
                              : "cursor-pointer"
                          }
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
                <span>{formatAbsoluteTime(ticket.createdAt)}</span>
              </div>

              <div className="flex items-center gap-2 text-sm">
                <Clock className="h-4 w-4 text-muted-foreground" />
                <span className="text-muted-foreground">Updated:</span>
                <span>{formatAbsoluteTime(ticket.updatedAt)}</span>
              </div>

              <Separator />

              {/* Status Management */}
              <div className="space-y-2">
                <span className="text-sm font-medium">Status</span>
                <DropdownMenu>
                  <DropdownMenuTrigger asChild>
                    <Button
                      variant="outline"
                      className="w-full justify-start"
                      disabled={isUpdatingStatus}>
                      <Badge
                        className={
                          TICKET_STATUS_COLORS[ticket.status as TicketStatus]
                        }
                        variant="secondary">
                        {TICKET_STATUS_LABELS[ticket.status as TicketStatus]}
                      </Badge>
                    </Button>
                  </DropdownMenuTrigger>
                  <DropdownMenuContent className="w-full">
                    {(["open", "pending", "resolved"] as TicketStatus[]).map(
                      (status) => (
                        <DropdownMenuItem
                          key={status}
                          onClick={() => handleStatusChange(status)}
                          disabled={ticket.status === status}>
                          <Badge
                            className={TICKET_STATUS_COLORS[status]}
                            variant="secondary">
                            {TICKET_STATUS_LABELS[status]}
                          </Badge>
                        </DropdownMenuItem>
                      )
                    )}
                  </DropdownMenuContent>
                </DropdownMenu>
              </div>

              <Separator />

              {/* Archive/Unarchive */}
              <div className="space-y-2">
                {ticket.status === "archived" ? (
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={handleUnarchive}
                    disabled={isUpdatingStatus}
                    className="w-full">
                    <ArchiveRestore className="h-4 w-4 mr-2" />
                    Unarchive
                  </Button>
                ) : (
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={handleArchive}
                    disabled={isUpdatingStatus}
                    className="w-full">
                    <Archive className="h-4 w-4 mr-2" />
                    Archive
                  </Button>
                )}
              </div>
            </CardContent>
          </Card>

          {/* Statistics */}
          <Card>
            <CardHeader>
              <CardTitle>Statistics</CardTitle>
            </CardHeader>
            <CardContent className="space-y-3">
              <div className="flex items-center justify-between text-sm">
                <span className="text-muted-foreground">Comments</span>
                <span className="font-medium">{ticket.comments.length}</span>
              </div>
              <div className="flex items-center justify-between text-sm">
                <span className="text-muted-foreground">Alerts</span>
                <span className="font-medium">{ticket.alerts.length}</span>
              </div>
            </CardContent>
          </Card>
        </div>
      </div>

      {/* Alert Detail Dialog */}
      <Dialog
        open={!!selectedAlert}
        onOpenChange={() => setSelectedAlert(null)}>
        <DialogContent className="max-w-4xl max-h-[80vh] overflow-y-auto">
          <DialogHeader>
            <DialogTitle className="flex items-center gap-2">
              <AlertCircle className="h-5 w-5" />
              Alert Details
            </DialogTitle>
          </DialogHeader>
          {selectedAlert && (
            <div className="space-y-4">
              <div className="grid grid-cols-2 gap-4 text-sm">
                <div>
                  <span className="font-medium">ID:</span>
                  <p className="text-muted-foreground font-mono">
                    {selectedAlert.id}
                  </p>
                </div>
                <div>
                  <span className="font-medium">Schema:</span>
                  <p className="text-muted-foreground">
                    {selectedAlert.schema}
                  </p>
                </div>
                <div>
                  <span className="font-medium">Created:</span>
                  <p className="text-muted-foreground">
                    {formatAbsoluteTime(selectedAlert.createdAt)}
                  </p>
                </div>
                <div>
                  <span className="font-medium">Updated:</span>
                  <p className="text-muted-foreground">
                    {formatAbsoluteTime(selectedAlert.createdAt)}
                  </p>
                </div>
              </div>

              {selectedAlert.title && (
                <div>
                  <span className="font-medium">Title:</span>
                  <p className="text-muted-foreground mt-1">
                    {selectedAlert.title}
                  </p>
                </div>
              )}

              {selectedAlert.description && (
                <div>
                  <span className="font-medium">Description:</span>
                  <p className="text-muted-foreground mt-1 whitespace-pre-wrap">
                    {selectedAlert.description}
                  </p>
                </div>
              )}

              {selectedAlert.data &&
                Object.keys(JSON.parse(selectedAlert.data || "{}")).length >
                  0 && (
                  <div>
                    <div className="flex items-center gap-2 mb-2">
                      <Hash className="h-4 w-4" />
                      <span className="font-medium">Attributes:</span>
                    </div>
                    <div className="bg-muted p-3 rounded-md">
                      <div className="grid gap-2">
                        {Object.entries(
                          JSON.parse(selectedAlert.data || "{}")
                        ).map(([key, value]) => (
                          <div
                            key={key}
                            className="flex items-start gap-2 text-sm">
                            <span className="font-mono text-muted-foreground min-w-0 flex-shrink-0">
                              {key}:
                            </span>
                            <span className="font-mono break-all">
                              {typeof value === "string"
                                ? value
                                : JSON.stringify(value)}
                            </span>
                          </div>
                        ))}
                      </div>
                    </div>
                  </div>
                )}
            </div>
          )}
        </DialogContent>
      </Dialog>
    </div>
  );
}

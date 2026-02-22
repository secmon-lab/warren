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
  DialogFooter,
} from "@/components/ui/dialog";
import { Separator } from "@/components/ui/separator";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
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
import { useErrorToast, useSuccessToast } from "@/hooks/use-toast";
import { TagSelector } from "@/components/ui/tag-selector";
import {
  GET_TICKET,
  GET_TICKET_ALERTS,
  RESOLVE_TICKET,
  REOPEN_TICKET,
  ARCHIVE_TICKET,
  UNARCHIVE_TICKET,
  UPDATE_TICKET_TAGS,
  GET_TAGS,
} from "@/lib/graphql/queries";
import {
  Ticket,
  TicketStatus,
  TICKET_STATUS_LABELS,
  TICKET_STATUS_COLORS,
  AlertConclusion,
  ALERT_CONCLUSION_LABELS,
  ALERT_CONCLUSION_DESCRIPTIONS,
  Alert,
  TagMetadata,
} from "@/lib/types";
import { formatRelativeTime } from "@/lib/utils-extended";
import {
  ChevronLeft,
  Archive,
  User,
  AlertCircle,
  MessageSquare,
  Calendar,
  Clock,
  Check,
  ExternalLink,
  FileText,
  Eye,
  Hash,
  ChevronDown,
  ChevronUp,
  ArchiveRestore,
  Copy,
  Pencil,
  Tag,
  RotateCcw,
} from "lucide-react";
import { EditConclusionModal } from "@/components/ui/edit-conclusion-modal";
import { EditTicketModal } from "@/components/EditTicketModal";
import { SimilarTickets } from "@/components/SimilarTickets";
import { TicketComments } from "@/components/TicketComments";
import { SalvageModal } from "@/components/SalvageModal";
import { TicketChat } from "@/components/TicketChat";
import { SessionsList } from "@/components/SessionsList";

const ALERTS_PER_PAGE = 5;

export default function TicketDetailPage() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const [selectedAlert, setSelectedAlert] = useState<Alert | null>(null);
  const [isSummaryOpen, setIsSummaryOpen] = useState(false);
  const [alertsCurrentPage, setAlertsCurrentPage] = useState(1);
  const [isUpdatingStatus, setIsUpdatingStatus] = useState(false);
  const [isEditTicketModalOpen, setIsEditTicketModalOpen] = useState(false);
  const [isEditConclusionModalOpen, setIsEditConclusionModalOpen] =
    useState(false);
  const [isSalvageModalOpen, setIsSalvageModalOpen] = useState(false);
  const [isResolveModalOpen, setIsResolveModalOpen] = useState(false);
  const [resolveConclusion, setResolveConclusion] = useState<AlertConclusion | "">("");
  const [resolveReason, setResolveReason] = useState("");
  const [tags, setTags] = useState<string[]>([]);
  
  // Initialize chat open state from localStorage
  const [isChatOpen, setIsChatOpen] = useState(() => {
    if (!id) return false;
    const saved = localStorage.getItem(`ticket-chat-open-${id}`);
    return saved === 'true';
  });

  const errorToast = useErrorToast();
  const successToast = useSuccessToast();

  // Handle opening chat and persisting state
  const handleStartChat = () => {
    setIsChatOpen(true);
    if (id) {
      localStorage.setItem(`ticket-chat-open-${id}`, 'true');
    }
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

  // Reset other state when ticket changes
  useEffect(() => {
    if (id) {
      setAlertsCurrentPage(1);
      setSelectedAlert(null);
      setIsSummaryOpen(false);
    }
  }, [id]);

  const [resolveTicket] = useMutation(RESOLVE_TICKET, {
    refetchQueries: [{ query: GET_TICKET, variables: { id } }],
  });
  const [reopenTicket] = useMutation(REOPEN_TICKET, {
    refetchQueries: [{ query: GET_TICKET, variables: { id } }],
  });
  const [archiveTicket] = useMutation(ARCHIVE_TICKET, {
    refetchQueries: [{ query: GET_TICKET, variables: { id } }],
  });
  const [unarchiveTicket] = useMutation(UNARCHIVE_TICKET, {
    refetchQueries: [{ query: GET_TICKET, variables: { id } }],
  });

  const [updateTicketTags] = useMutation(UPDATE_TICKET_TAGS, {
    onCompleted: () => {
      successToast("Tags updated successfully");
    },
    onError: (error) => {
      errorToast(`Failed to update tags: ${error.message}`);
    },
  });

  const {
    data: ticketData,
    loading: ticketLoading,
    error: ticketError,
  } = useQuery(GET_TICKET, {
    variables: { id },
    skip: !id,
  });

  const { data: tagsData } = useQuery(GET_TAGS);
  const allTags: TagMetadata[] = tagsData?.tags || [];

  const { data: alertsData } = useQuery(GET_TICKET_ALERTS, {
    variables: {
      id,
      offset: (alertsCurrentPage - 1) * ALERTS_PER_PAGE,
      limit: ALERTS_PER_PAGE,
    },
    skip: !id,
  });

  const ticket: Ticket = ticketData?.ticket;
  const alertsResponse = alertsData?.ticket?.alertsPaginated;

  // Set initial tags when ticket data loads
  useEffect(() => {
    if (ticket?.tags) {
      setTags(ticket.tags);
    }
  }, [ticket?.tags]);

  const handleTagsChange = async (newTags: string[]) => {
    if (!ticket?.id) return;
    
    setTags(newTags);
    
    // Convert tag names to tag IDs
    const tagIds: string[] = [];
    for (const tagName of newTags) {
      const tag = allTags.find(t => t.name === tagName);
      if (tag) {
        tagIds.push(tag.id);
      }
    }
    
    try {
      await updateTicketTags({
        variables: {
          ticketId: ticket.id,
          tagIds: tagIds,
        },
      });
    } catch (error) {
      console.error("Error updating tags:", error);
    }
  };

  const handleBackToList = () => {
    navigate("/tickets");
  };

  const handleAlertClick = (alert: Alert) => {
    setSelectedAlert(alert);
  };

  const handleResolveOpen = () => {
    setResolveConclusion("");
    setResolveReason("");
    setIsResolveModalOpen(true);
  };

  const handleResolveSubmit = async () => {
    if (!ticket || !resolveConclusion) return;

    setIsUpdatingStatus(true);
    try {
      await resolveTicket({
        variables: {
          id: ticket.id,
          conclusion: resolveConclusion,
          reason: resolveReason,
        },
      });
      successToast("Ticket resolved successfully");
      setIsResolveModalOpen(false);
    } catch (error) {
      console.error("Failed to resolve ticket:", error);
      errorToast("Failed to resolve ticket");
    } finally {
      setIsUpdatingStatus(false);
    }
  };

  const handleReopen = async () => {
    if (!ticket || isUpdatingStatus) return;

    setIsUpdatingStatus(true);
    try {
      await reopenTicket({ variables: { id: ticket.id } });
      successToast("Ticket reopened successfully");
    } catch (error) {
      console.error("Failed to reopen ticket:", error);
      errorToast("Failed to reopen ticket");
    } finally {
      setIsUpdatingStatus(false);
    }
  };

  const handleArchive = async () => {
    if (!ticket || isUpdatingStatus) return;

    setIsUpdatingStatus(true);
    try {
      await archiveTicket({ variables: { id: ticket.id } });
      successToast("Ticket archived successfully");
    } catch (error) {
      console.error("Failed to archive ticket:", error);
      errorToast("Failed to archive ticket");
    } finally {
      setIsUpdatingStatus(false);
    }
  };

  // cSpell:ignore unarchive
  const handleUnarchive = async () => {
    if (!ticket || isUpdatingStatus) return;

    setIsUpdatingStatus(true);
    try {
      await unarchiveTicket({ variables: { id: ticket.id } });
      successToast("Ticket unarchived successfully");
    } catch (error) {
      console.error("Failed to unarchive ticket:", error);
      errorToast("Failed to unarchive ticket");
    } finally {
      setIsUpdatingStatus(false);
    }
  };

  const handleEditConclusion = () => {
    setIsEditConclusionModalOpen(true);
  };

  const handleEditTicket = () => {
    setIsEditTicketModalOpen(true);
  };

  const handleDownloadAlerts = () => {
    if (!ticket?.id || totalAlerts === 0) {
      errorToast("No alerts to download");
      return;
    }

    // Create download URL
    const downloadUrl = `/api/tickets/${ticket.id}/alerts/download`;

    // Create a temporary link element and trigger download
    const link = document.createElement("a");
    link.href = downloadUrl;
    link.download = `ticket-${ticket.id}-alerts-${new Date()
      .toISOString()
      .slice(0, 19)
      .replace(/:/g, "-")}.jsonl`;
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);

    successToast(`Downloading ${totalAlerts} alert(s)`);
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

  if (ticketError) {
    const errorMessage = (ticketError as Error)?.message || String(ticketError);
    
    // Check if this is an authentication/authorization error
    if (
      errorMessage.includes("Authentication required") ||
      errorMessage.includes("Invalid authentication token") ||
      errorMessage.includes("JSON.parse") ||
      errorMessage.includes("unexpected character")
    ) {
      return (
        <div className="flex items-center justify-center h-64">
          <div className="text-center">
            <div className="text-lg text-red-600 mb-4">
              Authentication required
            </div>
            <div className="text-sm text-muted-foreground mb-4">
              Please log in to access this ticket
            </div>
            <Button
              onClick={() => (window.location.href = "/api/auth/login")}
              className="flex items-center gap-2">
              Sign In with Slack
            </Button>
          </div>
        </div>
      );
    }

    return (
      <div className="flex items-center justify-center h-64">
        <div className="text-lg text-red-600">
          Error loading ticket: {errorMessage}
        </div>
      </div>
    );
  }

  if (!ticket) {
    return (
      <div className="space-y-4">
        <div className="flex items-center justify-center h-64">
          <div className="text-lg text-red-600">Ticket not found</div>
        </div>
        <div className="flex justify-center">
          <Button variant="outline" onClick={handleBackToList}>
            <ChevronLeft className="h-4 w-4 mr-2" />
            Back to tickets
          </Button>
        </div>
      </div>
    );
  }

  // Use paginated alerts from server
  const paginatedAlerts = alertsResponse?.alerts || [];
  const totalAlerts = alertsResponse?.totalCount || ticket?.alertsCount || 0;
  const totalAlertsPages = Math.ceil(totalAlerts / ALERTS_PER_PAGE);

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-start justify-between">
        <div className="space-y-3 flex-1">
          <div className="flex items-center gap-2">
            <Button variant="ghost" size="sm" onClick={handleBackToList}>
              <ChevronLeft className="h-4 w-4 mr-1" />
              Back to tickets
            </Button>
            <Badge
              className={TICKET_STATUS_COLORS[ticket.status as TicketStatus]}
              variant="secondary">
              {TICKET_STATUS_LABELS[ticket.status as TicketStatus]}
            </Badge>
            {ticket.isTest && (
              <Badge
                variant="outline"
                className="bg-orange-50 text-orange-700 border-orange-200">
                🧪 TEST TICKET
              </Badge>
            )}
            {ticket.slackLink && (
              <Button variant="outline" size="sm" asChild>
                <a
                  href={ticket.slackLink}
                  target="_blank"
                  rel="noopener noreferrer"
                  className="inline-flex items-center gap-2">
                  <MessageSquare className="h-4 w-4" />
                  Open in Slack
                  <ExternalLink className="h-3 w-3" />
                </a>
              </Button>
            )}
          </div>
          <div className="flex items-start gap-3">
            <h1
              className="text-3xl font-bold tracking-tight break-words flex-1"
              title={ticket.title || `Ticket ${ticket.id.slice(0, 8)}`}>
              {ticket.isTest && "🧪 [TEST] "}
              {ticket.title || `Ticket ${ticket.id.slice(0, 8)}`}
            </h1>
            <Button
              variant="outline"
              size="sm"
              onClick={handleEditTicket}
              className="flex items-center gap-2 flex-shrink-0">
              <Pencil className="h-4 w-4" />
              Edit
            </Button>
          </div>
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
          <ResolveInfo
            ticket={ticket}
            onEditConclusion={handleEditConclusion}
          />

          {/* Comments Section */}
          <TicketComments ticketId={ticket.id} />

          {/* Chat Section */}
          {!isChatOpen ? (
            <Card>
              <CardHeader>
                <CardTitle className="flex items-center justify-between">
                  <div className="flex items-center gap-2">
                    <MessageSquare className="h-5 w-5" />
                    AI Chat Assistant
                  </div>
                  <Button
                    onClick={handleStartChat}
                    className="flex items-center gap-2">
                    <MessageSquare className="h-4 w-4" />
                    Start Chat
                  </Button>
                </CardTitle>
              </CardHeader>
              <CardContent>
                <p className="text-sm text-muted-foreground">
                  Click "Start Chat" to begin a conversation with the AI assistant about this ticket.
                </p>
              </CardContent>
            </Card>
          ) : (
            <TicketChat ticketId={ticket.id} />
          )}

          {/* Alerts Section with Pagination */}
          <Card>
            <CardHeader>
              <div className="flex items-center justify-between">
                <CardTitle className="flex items-center gap-2">
                  <AlertCircle className="h-5 w-5" />
                  Related Alerts ({totalAlerts})
                </CardTitle>
                <div className="flex items-center gap-2">
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => setIsSalvageModalOpen(true)}
                    className="flex items-center gap-2">
                    <AlertCircle className="h-4 w-4" />
                    Salvage
                  </Button>
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={handleDownloadAlerts}
                    className="flex items-center gap-2"
                    disabled={totalAlerts === 0}>
                    <FileText className="h-4 w-4" />
                    Download
                  </Button>
                </div>
              </div>
            </CardHeader>
            <CardContent className="p-0">
              <div className="divide-y">
                {paginatedAlerts.map((alert: Alert) => (
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

                      {/* Show page numbers with ellipsis for large page counts */}
                      {(() => {
                        const maxVisiblePages = 5;
                        const pages = [];

                        if (totalAlertsPages <= maxVisiblePages) {
                          // Show all pages if total is small
                          for (let i = 1; i <= totalAlertsPages; i++) {
                            pages.push(
                              <PaginationItem key={i}>
                                <PaginationLink
                                  isActive={i === alertsCurrentPage}
                                  onClick={() => setAlertsCurrentPage(i)}
                                  className="cursor-pointer">
                                  {i}
                                </PaginationLink>
                              </PaginationItem>
                            );
                          }
                        } else {
                          // Show first page
                          pages.push(
                            <PaginationItem key={1}>
                              <PaginationLink
                                isActive={1 === alertsCurrentPage}
                                onClick={() => setAlertsCurrentPage(1)}
                                className="cursor-pointer">
                                1
                              </PaginationLink>
                            </PaginationItem>
                          );

                          // Show ellipsis if needed
                          if (alertsCurrentPage > 3) {
                            pages.push(
                              <PaginationItem key="ellipsis1">
                                <span className="px-3 py-2">...</span>
                              </PaginationItem>
                            );
                          }

                          // Show current page and neighbors
                          const start = Math.max(2, alertsCurrentPage - 1);
                          const end = Math.min(
                            totalAlertsPages - 1,
                            alertsCurrentPage + 1
                          );

                          for (let i = start; i <= end; i++) {
                            if (i !== 1 && i !== totalAlertsPages) {
                              pages.push(
                                <PaginationItem key={i}>
                                  <PaginationLink
                                    isActive={i === alertsCurrentPage}
                                    onClick={() => setAlertsCurrentPage(i)}
                                    className="cursor-pointer">
                                    {i}
                                  </PaginationLink>
                                </PaginationItem>
                              );
                            }
                          }

                          // Show ellipsis if needed
                          if (alertsCurrentPage < totalAlertsPages - 2) {
                            pages.push(
                              <PaginationItem key="ellipsis2">
                                <span className="px-3 py-2">...</span>
                              </PaginationItem>
                            );
                          }

                          // Show last page
                          if (totalAlertsPages > 1) {
                            pages.push(
                              <PaginationItem key={totalAlertsPages}>
                                <PaginationLink
                                  isActive={
                                    totalAlertsPages === alertsCurrentPage
                                  }
                                  onClick={() =>
                                    setAlertsCurrentPage(totalAlertsPages)
                                  }
                                  className="cursor-pointer">
                                  {totalAlertsPages}
                                </PaginationLink>
                              </PaginationItem>
                            );
                          }
                        }

                        return pages;
                      })()}

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
              <div className="space-y-1">
                <div className="flex items-center gap-2">
                  <User className="h-3 w-3 text-muted-foreground" />
                  <span className="text-xs font-medium text-muted-foreground tracking-wide uppercase">
                    Assignee
                  </span>
                </div>
                <div className="ml-5">
                  {ticket.assignee ? (
                    <div className="text-xs">
                      <UserWithAvatar
                        userID={ticket.assignee.id}
                        fallback={ticket.assignee.name}
                        avatarSize="sm"
                      />
                    </div>
                  ) : (
                    <span className="text-xs text-muted-foreground">
                      Unassigned
                    </span>
                  )}
                </div>
              </div>

              <div className="space-y-1">
                <div className="flex items-center gap-2">
                  <Calendar className="h-3 w-3 text-muted-foreground" />
                  <span className="text-xs font-medium text-muted-foreground tracking-wide uppercase">
                    Created
                  </span>
                </div>
                <div className="ml-5">
                  <span className="text-xs font-mono">
                    {formatAbsoluteTime(ticket.createdAt)}
                  </span>
                </div>
              </div>

              <div className="space-y-1">
                <div className="flex items-center gap-2">
                  <Clock className="h-3 w-3 text-muted-foreground" />
                  <span className="text-xs font-medium text-muted-foreground tracking-wide uppercase">
                    Updated
                  </span>
                </div>
                <div className="ml-5">
                  <span className="text-xs font-mono">
                    {formatAbsoluteTime(ticket.updatedAt)}
                  </span>
                </div>
              </div>

              {ticket.slackLink && (
                <div className="space-y-1">
                  <div className="flex items-center gap-2">
                    <MessageSquare className="h-3 w-3 text-muted-foreground" />
                    <span className="text-xs font-medium text-muted-foreground tracking-wide uppercase">
                      Slack Discussion
                    </span>
                  </div>
                  <div className="ml-5">
                    <a
                      href={ticket.slackLink}
                      target="_blank"
                      rel="noopener noreferrer"
                      className="flex items-center justify-center gap-2 w-full px-3 py-1 text-sm bg-purple-50 hover:bg-purple-100 text-purple-700 hover:text-purple-800 rounded-md border border-purple-200 transition-colors">
                      <MessageSquare className="h-4 w-4" />
                      Open in Slack
                      <ExternalLink className="h-3 w-3" />
                    </a>
                  </div>
                </div>
              )}

              <Separator />

              {/* Status Management */}
              <div className="space-y-2">
                <div className="flex items-center gap-2">
                  <Badge
                    className={TICKET_STATUS_COLORS[ticket.status as TicketStatus]}
                    variant="secondary">
                    {TICKET_STATUS_LABELS[ticket.status as TicketStatus]}
                  </Badge>
                </div>

                {ticket.status === "open" && (
                  <Button
                    size="sm"
                    onClick={handleResolveOpen}
                    disabled={isUpdatingStatus}
                    className="w-full">
                    <Check className="h-4 w-4 mr-2" />
                    Resolve
                  </Button>
                )}

                {ticket.status === "resolved" && (
                  <div className="flex gap-2">
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={handleReopen}
                      disabled={isUpdatingStatus}
                      className="flex-1">
                      <RotateCcw className="h-4 w-4 mr-1" />
                      Reopen
                    </Button>
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={handleArchive}
                      disabled={isUpdatingStatus}
                      className="flex-1">
                      <Archive className="h-4 w-4 mr-1" />
                      Archive
                    </Button>
                  </div>
                )}

                {ticket.status === "archived" && (
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={handleUnarchive}
                    disabled={isUpdatingStatus}
                    className="w-full">
                    <ArchiveRestore className="h-4 w-4 mr-2" />
                    Unarchive
                  </Button>
                )}
              </div>
            </CardContent>
          </Card>

          {/* Tags */}
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <Tag className="h-4 w-4" />
                Tags
              </CardTitle>
            </CardHeader>
            <CardContent>
              <TagSelector
                selectedTags={tags}
                onTagsChange={handleTagsChange}
                disabled={false}
              />
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
                <span className="font-medium">Paginated</span>
              </div>
              <div className="flex items-center justify-between text-sm">
                <span className="text-muted-foreground">Alerts</span>
                <span className="font-medium">{ticket.alertsCount || 0}</span>
              </div>
            </CardContent>
          </Card>

          {/* Sessions */}
          <SessionsList ticketId={ticket.id} />

          {/* Similar Tickets */}
          <SimilarTickets ticketId={ticket.id} />
        </div>
      </div>

      {/* Alert Detail Dialog */}
      <Dialog
        open={!!selectedAlert}
        onOpenChange={() => setSelectedAlert(null)}>
        <DialogContent
          className="!max-w-[90vw] w-full max-h-[85vh] overflow-y-auto p-6"
          style={{ maxWidth: "90vw" }}>
          <DialogHeader>
            <DialogTitle className="flex items-center gap-2">
              <AlertCircle className="h-5 w-5" />
              Alert Details
            </DialogTitle>
          </DialogHeader>
          {selectedAlert && (
            <div className="space-y-4 w-full min-w-0">
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
                <div className="w-full">
                  <span className="font-medium">Title:</span>
                  <p className="text-muted-foreground mt-1">
                    {selectedAlert.title}
                  </p>
                </div>
              )}

              {selectedAlert.description && (
                <div className="w-full">
                  <span className="font-medium">Description:</span>
                  <p className="text-muted-foreground mt-1 whitespace-pre-wrap">
                    {selectedAlert.description}
                  </p>
                </div>
              )}

              {selectedAlert.attributes &&
                selectedAlert.attributes.length > 0 && (
                  <div className="w-full">
                    <div className="flex items-center gap-2 mb-2">
                      <ExternalLink className="h-4 w-4" />
                      <span className="font-medium">Attributes:</span>
                    </div>
                    <div className="space-y-2">
                      {selectedAlert.attributes.map((attr, index) => (
                        <div
                          key={index}
                          className="flex items-start gap-2 p-2 bg-muted rounded-md w-full">
                          <div className="flex-1 min-w-0">
                            <div className="flex items-center gap-2 mb-1">
                              <span className="font-medium text-sm">
                                {attr.key}:
                              </span>
                              {attr.auto && (
                                <Badge variant="outline" className="text-xs">
                                  auto
                                </Badge>
                              )}
                            </div>
                            {attr.link ? (
                              <a
                                href={attr.link}
                                target="_blank"
                                rel="noopener noreferrer"
                                className="text-blue-600 hover:text-blue-800 underline text-sm break-all">
                                {attr.value}
                              </a>
                            ) : (
                              <span className="text-sm font-mono break-all text-muted-foreground">
                                {attr.value}
                              </span>
                            )}
                          </div>
                        </div>
                      ))}
                    </div>
                  </div>
                )}

              {selectedAlert.data &&
                Object.keys(JSON.parse(selectedAlert.data || "{}")).length >
                  0 && (
                  <div className="w-full">
                    <div className="flex items-center justify-between mb-2">
                      <div className="flex items-center gap-2">
                        <Hash className="h-4 w-4" />
                        <span className="font-medium">Data:</span>
                      </div>
                      <CopyButton data={selectedAlert.data} />
                    </div>
                    <div className="bg-muted p-4 rounded-md w-full min-w-0 overflow-auto">
                      <pre className="text-sm font-mono whitespace-pre-wrap text-foreground min-w-0 w-full break-words overflow-wrap-anywhere">
                        {JSON.stringify(
                          JSON.parse(selectedAlert.data || "{}"),
                          null,
                          2
                        )}
                      </pre>
                    </div>
                  </div>
                )}
            </div>
          )}
        </DialogContent>
      </Dialog>

      {/* Resolve Modal */}
      <Dialog open={isResolveModalOpen} onOpenChange={(open) => { if (!isUpdatingStatus) setIsResolveModalOpen(open); }}>
        <DialogContent className="sm:max-w-lg">
          <DialogHeader>
            <DialogTitle>Resolve Ticket</DialogTitle>
          </DialogHeader>
          <div className="space-y-6">
            <div className="space-y-3">
              <Label htmlFor="resolve-conclusion" className="text-sm font-medium">
                Conclusion <span className="text-red-500">*</span>
              </Label>
              <Select
                value={resolveConclusion}
                onValueChange={(value) => setResolveConclusion(value as AlertConclusion)}
                disabled={isUpdatingStatus}
              >
                <SelectTrigger className="w-full">
                  <SelectValue placeholder="Select a conclusion..." />
                </SelectTrigger>
                <SelectContent>
                  {Object.entries(ALERT_CONCLUSION_LABELS).map(([value, label]) => (
                    <SelectItem key={value} value={value}>
                      {label}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
              {resolveConclusion && (
                <div className="bg-muted/50 rounded-lg p-3 border">
                  <p className="text-sm text-muted-foreground">
                    {ALERT_CONCLUSION_DESCRIPTIONS[resolveConclusion as AlertConclusion]}
                  </p>
                </div>
              )}
            </div>
            <div className="space-y-3">
              <Label htmlFor="resolve-reason" className="text-sm font-medium">
                Reason
              </Label>
              <Textarea
                id="resolve-reason"
                placeholder="Add detailed reasoning, context, or additional information..."
                value={resolveReason}
                onChange={(e: React.ChangeEvent<HTMLTextAreaElement>) => setResolveReason(e.target.value)}
                disabled={isUpdatingStatus}
                rows={4}
                className="resize-none"
              />
              <p className="text-xs text-muted-foreground">
                Provide context for this conclusion to help future analysis.
              </p>
            </div>
          </div>
          <DialogFooter className="gap-2">
            <Button
              variant="outline"
              onClick={() => setIsResolveModalOpen(false)}
              disabled={isUpdatingStatus}
            >
              Cancel
            </Button>
            <Button
              onClick={handleResolveSubmit}
              disabled={isUpdatingStatus || !resolveConclusion}
              className="min-w-[80px]"
            >
              {isUpdatingStatus ? "Resolving..." : "Resolve"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Edit Conclusion Modal */}
      <EditConclusionModal
        isOpen={isEditConclusionModalOpen}
        onClose={() => setIsEditConclusionModalOpen(false)}
        ticketId={ticket.id}
        currentConclusion={ticket.conclusion}
        currentReason={ticket.reason}
      />

      <EditTicketModal
        isOpen={isEditTicketModalOpen}
        onClose={() => setIsEditTicketModalOpen(false)}
        ticket={ticket}
      />

      {/* Salvage Modal */}
      <SalvageModal
        isOpen={isSalvageModalOpen}
        onClose={() => setIsSalvageModalOpen(false)}
        ticketId={ticket.id}
      />
    </div>
  );
}

// Copy Button Component
function CopyButton({ data }: { data: string }) {
  const [copied, setCopied] = useState(false);

  const handleCopy = async () => {
    try {
      const formatted = JSON.stringify(JSON.parse(data), null, 2);
      await navigator.clipboard.writeText(formatted);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch (error) {
      console.error("Failed to copy data:", error);
    }
  };

  return (
    <Button
      variant="outline"
      size="sm"
      onClick={handleCopy}
      className="h-8 px-2">
      {copied ? (
        <Check className="h-3 w-3 text-green-600" />
      ) : (
        <Copy className="h-3 w-3" />
      )}
      <span className="ml-1 text-xs">{copied ? "Copied!" : "Copy"}</span>
    </Button>
  );
}

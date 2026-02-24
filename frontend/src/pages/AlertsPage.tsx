import { useQuery, useMutation } from "@apollo/client";
import { useState, useMemo, useCallback } from "react";
import { useNavigate } from "react-router-dom";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import {
  Pagination,
  PaginationContent,
  PaginationItem,
  PaginationLink,
  PaginationNext,
  PaginationPrevious,
} from "@/components/ui/pagination";
import {
  GET_ALERTS,
  GET_TAGS,
  DECLINE_ALERTS,
  CREATE_TICKET_FROM_ALERTS,
} from "@/lib/graphql/queries";
import { Alert, AlertStatus, TagMetadata } from "@/lib/types";
import { AlertTriangle, Tag, Ban, Ticket } from "lucide-react";
import { generateTagColor } from "@/lib/tag-colors";
import { useConfirm } from "@/hooks/use-confirm";
import { useToast } from "@/hooks/use-toast";

interface AlertsData {
  alerts: {
    alerts: Alert[];
    totalCount: number;
  };
}

export default function AlertsPage() {
  const navigate = useNavigate();
  const confirm = useConfirm();
  const { toast } = useToast();
  const [currentPage, setCurrentPage] = useState(1);
  const [statusFilter, setStatusFilter] = useState<AlertStatus>("ACTIVE");
  const [selectedIds, setSelectedIds] = useState<Set<string>>(new Set());
  const ITEMS_PER_PAGE = 10;

  const handleStatusChange = (newStatus: AlertStatus) => {
    setStatusFilter(newStatus);
    setCurrentPage(1);
    setSelectedIds(new Set());
  };

  const handlePageChange = (page: number) => {
    setCurrentPage(page);
    setSelectedIds(new Set());
  };

  const {
    data: alertsData,
    loading: alertsLoading,
    error: alertsError,
  } = useQuery<AlertsData>(GET_ALERTS, {
    variables: {
      offset: (currentPage - 1) * ITEMS_PER_PAGE,
      limit: ITEMS_PER_PAGE,
      status: statusFilter,
    },
  });

  const { data: tagsData } = useQuery(GET_TAGS);
  const tagsByName = new Map(
    (tagsData?.tags || []).map((t: TagMetadata) => [t.name, t])
  );

  const [declineAlerts, { loading: decliningAlerts }] = useMutation(
    DECLINE_ALERTS,
    {
      refetchQueries: [{ query: GET_ALERTS, variables: { offset: (currentPage - 1) * ITEMS_PER_PAGE, limit: ITEMS_PER_PAGE, status: statusFilter } }],
      onCompleted: (data) => {
        const count = data.declineAlerts.length;
        toast({
          title: "Alerts declined",
          description: `${count} alert(s) have been declined.`,
        });
        setSelectedIds(new Set());
      },
      onError: (error) => {
        toast({
          title: "Failed to decline alerts",
          description: error.message,
          variant: "destructive",
        });
      },
    }
  );

  const [createTicketFromAlerts, { loading: creatingTicket }] = useMutation(
    CREATE_TICKET_FROM_ALERTS,
    {
      refetchQueries: [{ query: GET_ALERTS, variables: { offset: (currentPage - 1) * ITEMS_PER_PAGE, limit: ITEMS_PER_PAGE, status: statusFilter } }],
      onCompleted: (data) => {
        toast({
          title: "Ticket created",
          description: `Ticket has been created from selected alerts.`,
        });
        setSelectedIds(new Set());
        navigate(`/tickets/${data.createTicketFromAlerts.id}`);
      },
      onError: (error) => {
        toast({
          title: "Failed to create ticket",
          description: error.message,
          variant: "destructive",
        });
      },
    }
  );

  // Sort alerts by createdAt in descending order (newest first)
  const sortedAlerts: Alert[] = useMemo(() => {
    if (!alertsData?.alerts?.alerts) return [];

    return [...alertsData.alerts.alerts].sort(
      (a, b) =>
        new Date(b.createdAt).getTime() - new Date(a.createdAt).getTime()
    );
  }, [alertsData?.alerts?.alerts]);

  // Calculate pagination values
  const totalCount = alertsData?.alerts?.totalCount || 0;
  const totalPages = Math.ceil(totalCount / ITEMS_PER_PAGE);

  const handleAlertClick = (alertId: string) => {
    navigate(`/alerts/${alertId}`);
  };

  const toggleSelect = useCallback(
    (alertId: string) => {
      setSelectedIds((prev) => {
        const next = new Set(prev);
        if (next.has(alertId)) {
          next.delete(alertId);
        } else {
          next.add(alertId);
        }
        return next;
      });
    },
    []
  );

  const toggleSelectAll = useCallback(() => {
    if (selectedIds.size === sortedAlerts.length) {
      setSelectedIds(new Set());
    } else {
      setSelectedIds(new Set(sortedAlerts.map((a) => a.id)));
    }
  }, [selectedIds.size, sortedAlerts]);

  const handleDecline = useCallback(async () => {
    const count = selectedIds.size;
    const confirmed = await confirm({
      title: "Decline Alerts",
      description: `Are you sure you want to decline ${count} alert(s)? This will mark them as declined.`,
      confirmText: "Decline",
      cancelText: "Cancel",
      variant: "destructive",
    });
    if (!confirmed) return;

    declineAlerts({ variables: { ids: Array.from(selectedIds) } });
  }, [selectedIds, confirm, declineAlerts]);

  const handleCreateTicket = useCallback(() => {
    createTicketFromAlerts({
      variables: { alertIds: Array.from(selectedIds) },
    });
  }, [selectedIds, createTicketFromAlerts]);

  const formatDate = (dateString: string) => {
    const date = new Date(dateString);
    const datePart = date.toISOString().split("T")[0].replace(/-/g, "/");
    const timePart = date.toISOString().split("T")[1].split(".")[0];
    return `${datePart} ${timePart}`;
  };

  const isActionLoading = decliningAlerts || creatingTicket;

  if (alertsLoading) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="text-lg">Loading alerts...</div>
      </div>
    );
  }

  if (alertsError) {
    // Check if this is an authentication/authorization error
    if (
      alertsError.message?.includes("Authentication required") ||
      alertsError.message?.includes("Invalid authentication token") ||
      alertsError.message?.includes("JSON.parse") ||
      alertsError.message?.includes("unexpected character")
    ) {
      return (
        <div className="flex items-center justify-center h-64">
          <div className="text-center">
            <div className="text-lg text-red-600 mb-4">
              Authentication required
            </div>
            <div className="text-sm text-muted-foreground mb-4">
              Please log in to access alerts
            </div>
            <Button
              onClick={() => (window.location.href = "/api/auth/login")}
              className="flex items-center gap-2"
            >
              Sign In with Slack
            </Button>
          </div>
        </div>
      );
    }

    return (
      <div className="flex items-center justify-center h-64">
        <div className="text-lg text-red-600">
          Error loading alerts: {alertsError.message}
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold tracking-tight">Alerts</h1>
          <p className="text-muted-foreground">
            Monitor and manage security alerts
          </p>
        </div>
      </div>

      {/* Status filter tabs */}
      <div className="flex gap-2">
        {[
          {
            status: "ACTIVE" as AlertStatus,
            label: "New",
            icon: AlertTriangle,
          },
          { status: "DECLINED" as AlertStatus, label: "Declined", icon: Ban },
        ].map(({ status, label, icon: Icon }) => (
          <Button
            key={status}
            variant={statusFilter === status ? "default" : "outline"}
            size="sm"
            onClick={() => handleStatusChange(status)}
          >
            <Icon className="h-4 w-4 mr-1" />
            {label}
          </Button>
        ))}
      </div>

      {/* Action bar - shown when alerts are selected */}
      {selectedIds.size > 0 && (
        <div className="flex items-center gap-3 px-4 py-2 bg-muted/50 border rounded-lg">
          <span className="text-sm font-medium">
            {selectedIds.size} selected
          </span>
          <div className="flex gap-2 ml-auto">
            <Button
              variant="outline"
              size="sm"
              onClick={handleCreateTicket}
              disabled={isActionLoading}
            >
              <Ticket className="h-4 w-4 mr-1" />
              {creatingTicket ? "Creating..." : "Create Ticket"}
            </Button>
            {statusFilter === "ACTIVE" && (
              <Button
                variant="destructive"
                size="sm"
                onClick={handleDecline}
                disabled={isActionLoading}
              >
                <Ban className="h-4 w-4 mr-1" />
                {decliningAlerts ? "Declining..." : "Decline"}
              </Button>
            )}
          </div>
        </div>
      )}

      {sortedAlerts.length === 0 ? (
        <div className="bg-card text-card-foreground rounded-xl border shadow-sm">
          <div className="flex items-center justify-center h-32 px-6">
            <p className="text-muted-foreground">No alerts found</p>
          </div>
        </div>
      ) : (
        <div className="space-y-2">
          {/* Select all header */}
          <div className="flex items-center gap-3 px-4 py-2">
            <Checkbox
              checked={
                sortedAlerts.length > 0 &&
                selectedIds.size === sortedAlerts.length
              }
              onCheckedChange={toggleSelectAll}
            />
            <span className="text-sm text-muted-foreground">
              Select all on this page
            </span>
          </div>

          {sortedAlerts.map((alert: Alert) => (
            <div
              key={alert.id}
              className="bg-card text-card-foreground rounded-xl border shadow-sm cursor-pointer hover:shadow-md transition-shadow overflow-hidden"
              onClick={() => handleAlertClick(alert.id)}
            >
              <div className="px-4 py-6">
                <div className="flex items-start gap-3">
                  {/* Checkbox */}
                  <div
                    className="pt-0.5"
                    onClick={(e) => e.stopPropagation()}
                  >
                    <Checkbox
                      checked={selectedIds.has(alert.id)}
                      onCheckedChange={() => toggleSelect(alert.id)}
                    />
                  </div>

                  {/* Alert content */}
                  <div className="flex-1 min-w-0">
                    <div className="flex items-center gap-2 mb-1">
                      <AlertTriangle className="h-4 w-4 text-orange-500" />
                      <Badge variant="outline" className="text-xs">
                        {alert.schema}
                      </Badge>
                      {alert.status === "DECLINED" && (
                        <Badge variant="destructive" className="text-xs">
                          Declined
                        </Badge>
                      )}
                      {alert.ticket && (
                        <Badge variant="secondary" className="text-xs">
                          Assigned
                        </Badge>
                      )}
                      <span className="text-xs text-muted-foreground ml-auto">
                        {formatDate(alert.createdAt)}
                      </span>
                    </div>

                    <div className="flex items-start justify-between">
                      <div className="flex-1 min-w-0 pr-2">
                        <h3 className="text-base font-semibold mb-1 line-clamp-1">
                          {alert.title}
                        </h3>
                        {alert.description && (
                          <p className="text-sm text-muted-foreground line-clamp-2 mb-1">
                            {alert.description}
                          </p>
                        )}

                        {/* Alert Tags and Attributes */}
                        <div className="flex flex-wrap gap-1 max-w-full mb-1">
                          {alert.tags &&
                            alert.tags.length > 0 &&
                            alert.tags.map((tag, index) => {
                              const tagData = tagsByName.get(tag) as
                                | TagMetadata
                                | undefined;
                              const colorClass =
                                tagData?.color || generateTagColor(tag);
                              return (
                                <Badge
                                  key={`tag-${index}`}
                                  variant="secondary"
                                  className={`text-xs ${colorClass}`}
                                >
                                  <Tag className="h-3 w-3 mr-1" />
                                  {tag}
                                </Badge>
                              );
                            })}
                        </div>

                        {/* Alert Attributes */}
                        <div className="flex flex-wrap gap-1 max-w-full">
                          {alert.attributes &&
                            alert.attributes.length > 0 && (
                              <>
                                {alert.attributes
                                  .slice(0, 3)
                                  .map((attr, index: number) => {
                                    const displayText = `${attr.key}: ${attr.value}`;
                                    const isLong = displayText.length > 40;
                                    const truncatedText = isLong
                                      ? `${displayText.substring(0, 37)}...`
                                      : displayText;

                                    return (
                                      <Badge
                                        key={index}
                                        variant={
                                          attr.auto ? "outline" : "default"
                                        }
                                        className="text-xs max-w-[280px] truncate inline-block"
                                        title={displayText}
                                      >
                                        {truncatedText}
                                      </Badge>
                                    );
                                  })}
                                {alert.attributes.length > 3 && (
                                  <Badge variant="outline" className="text-xs">
                                    +{alert.attributes.length - 3} more
                                  </Badge>
                                )}
                              </>
                            )}
                          {alert.ticket && (
                            <Badge variant="outline" className="text-xs">
                              Ticket #{alert.ticket.id.slice(-8)}
                            </Badge>
                          )}
                        </div>
                      </div>
                    </div>
                  </div>
                </div>
              </div>
            </div>
          ))}
        </div>
      )}

      {/* Pagination */}
      {totalPages > 1 && (
        <div className="flex justify-center mt-6">
          <Pagination>
            <PaginationContent>
              <PaginationItem>
                <PaginationPrevious
                  onClick={() => handlePageChange(Math.max(1, currentPage - 1))}
                  className={
                    currentPage === 1
                      ? "pointer-events-none opacity-50"
                      : "cursor-pointer"
                  }
                />
              </PaginationItem>
              {/* Show page numbers with truncation */}
              {(() => {
                const maxVisiblePages = 10;
                const pageNumbers: (number | string)[] = [];

                if (totalPages <= maxVisiblePages) {
                  for (let i = 1; i <= totalPages; i++) {
                    pageNumbers.push(i);
                  }
                } else {
                  const startPage = Math.max(1, currentPage - 4);
                  const endPage = Math.min(totalPages, currentPage + 4);

                  if (startPage > 1) {
                    pageNumbers.push(1);
                    if (startPage > 2) {
                      pageNumbers.push("...");
                    }
                  }

                  for (let i = startPage; i <= endPage; i++) {
                    pageNumbers.push(i);
                  }

                  if (endPage < totalPages) {
                    if (endPage < totalPages - 1) {
                      pageNumbers.push("...");
                    }
                    pageNumbers.push(totalPages);
                  }
                }

                return pageNumbers.map((page, index) => (
                  <PaginationItem key={index}>
                    {page === "..." ? (
                      <span className="px-3 py-2 text-sm text-muted-foreground">
                        ...
                      </span>
                    ) : (
                      <PaginationLink
                        isActive={page === currentPage}
                        onClick={() => handlePageChange(page as number)}
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
                  onClick={() =>
                    handlePageChange(Math.min(totalPages, currentPage + 1))
                  }
                  className={
                    currentPage === totalPages
                      ? "pointer-events-none opacity-50"
                      : "cursor-pointer"
                  }
                />
              </PaginationItem>
            </PaginationContent>
          </Pagination>
        </div>
      )}
    </div>
  );
}

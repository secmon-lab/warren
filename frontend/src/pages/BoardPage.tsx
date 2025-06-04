import { useQuery, useMutation } from "@apollo/client";
import { useNavigate } from "react-router-dom";
import { useState, useMemo } from "react";
import { Card, CardContent } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { UserWithAvatar } from "@/components/ui/user-name";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import {
  GET_TICKETS,
  UPDATE_TICKET_STATUS,
  UPDATE_MULTIPLE_TICKETS_STATUS,
} from "@/lib/graphql/queries";
import {
  Ticket,
  TicketStatus,
  TICKET_STATUS_LABELS,
  TICKET_STATUS_COLORS,
} from "@/lib/types";
import { formatRelativeTime } from "@/lib/utils-extended";
import {
  AlertCircle,
  MessageSquare,
  User,
  MoreHorizontal,
  Archive,
} from "lucide-react";
import { useErrorToast, useSuccessToast } from "@/hooks/use-toast";
import { useConfirm } from "@/hooks/use-confirm";

const BOARD_STATUSES: TicketStatus[] = ["open", "pending", "resolved"];

export default function BoardPage() {
  const navigate = useNavigate();
  const [draggedTicket, setDraggedTicket] = useState<Ticket | null>(null);
  const [isUpdatingStatus, setIsUpdatingStatus] = useState(false);

  const errorToast = useErrorToast();
  const successToast = useSuccessToast();
  const confirm = useConfirm();

  const [updateTicketStatus] = useMutation(UPDATE_TICKET_STATUS, {
    refetchQueries: [
      { query: GET_TICKETS, variables: { statuses: BOARD_STATUSES } },
    ],
  });

  const [updateMultipleTicketsStatus] = useMutation(
    UPDATE_MULTIPLE_TICKETS_STATUS,
    {
      refetchQueries: [
        { query: GET_TICKETS, variables: { statuses: BOARD_STATUSES } },
      ],
    }
  );

  const { data, loading, error } = useQuery(GET_TICKETS, {
    variables: {
      statuses: BOARD_STATUSES,
    },
  });

  const tickets: Ticket[] = useMemo(() => data?.tickets || [], [data?.tickets]);

  const ticketsByStatus = useMemo(() => {
    return BOARD_STATUSES.reduce((acc, status) => {
      acc[status] = tickets.filter((ticket) => ticket.status === status);
      return acc;
    }, {} as Record<TicketStatus, Ticket[]>);
  }, [tickets]);

  const handleTicketClick = (ticketId: string) => {
    navigate(`/tickets/${ticketId}`);
  };

  const handleDragStart = (e: React.DragEvent, ticket: Ticket) => {
    setDraggedTicket(ticket);
    e.dataTransfer.effectAllowed = "move";
  };

  const handleDragOver = (e: React.DragEvent) => {
    e.preventDefault();
    e.dataTransfer.dropEffect = "move";
  };

  const handleDrop = async (e: React.DragEvent, targetStatus: TicketStatus) => {
    e.preventDefault();

    if (
      !draggedTicket ||
      draggedTicket.status === targetStatus ||
      isUpdatingStatus
    ) {
      setDraggedTicket(null);
      return;
    }

    setIsUpdatingStatus(true);
    try {
      await updateTicketStatus({
        variables: {
          id: draggedTicket.id,
          status: targetStatus,
        },
      });
      successToast(`Ticket moved to ${targetStatus}`);
    } catch (error) {
      console.error("Failed to update ticket status:", error);
      errorToast("Failed to update ticket status");
    } finally {
      setIsUpdatingStatus(false);
      setDraggedTicket(null);
    }
  };

  const handleArchiveResolved = async () => {
    const resolvedTickets = ticketsByStatus["resolved"];
    if (resolvedTickets.length === 0) return;

    const confirmed = await confirm({
      title: "Archive Resolved Tickets",
      description: `Are you sure you want to archive ${resolvedTickets.length} resolved tickets?`,
      confirmText: "Archive",
      variant: "destructive",
    });

    if (!confirmed) return;

    setIsUpdatingStatus(true);
    try {
      await updateMultipleTicketsStatus({
        variables: {
          ids: resolvedTickets.map((ticket) => ticket.id),
          status: "archived",
        },
      });
      successToast(`Successfully archived ${resolvedTickets.length} tickets`);
    } catch (error) {
      console.error("Failed to archive tickets:", error);
      errorToast("Failed to archive tickets");
    } finally {
      setIsUpdatingStatus(false);
    }
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="text-lg">Loading board...</div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="text-lg text-red-600">
          Error loading board: {error.message}
        </div>
      </div>
    );
  }

  return (
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
                <Badge
                  className={TICKET_STATUS_COLORS[status]}
                  variant="secondary">
                  {TICKET_STATUS_LABELS[status]}
                </Badge>
                <span className="text-sm text-muted-foreground">
                  ({ticketsByStatus[status].length})
                </span>
              </h2>

              {status === "resolved" && ticketsByStatus[status].length > 0 && (
                <DropdownMenu>
                  <DropdownMenuTrigger asChild>
                    <Button
                      variant="ghost"
                      size="sm"
                      disabled={isUpdatingStatus}>
                      <MoreHorizontal className="h-4 w-4" />
                    </Button>
                  </DropdownMenuTrigger>
                  <DropdownMenuContent>
                    <DropdownMenuItem onClick={handleArchiveResolved}>
                      <Archive className="h-4 w-4 mr-2" />
                      Archive All Resolved
                    </DropdownMenuItem>
                  </DropdownMenuContent>
                </DropdownMenu>
              )}
            </div>

            <div
              className="space-y-2 min-h-[400px] p-2 border-2 border-dashed border-transparent rounded-lg transition-colors"
              onDragOver={handleDragOver}
              onDrop={(e) => handleDrop(e, status)}
              style={{
                borderColor:
                  draggedTicket && draggedTicket.status !== status
                    ? "#e2e8f0"
                    : "transparent",
              }}>
              {ticketsByStatus[status].map((ticket) => (
                <Card
                  key={ticket.id}
                  className="hover:shadow-md transition-shadow cursor-move"
                  draggable
                  onDragStart={(e) => handleDragStart(e, ticket)}
                  onClick={() => handleTicketClick(ticket.id)}>
                  <CardContent className="p-3">
                    <h3
                      className="font-medium text-sm mb-2 line-clamp-2"
                      title={ticket.title || `Ticket ${ticket.id.slice(0, 8)}`}>
                      {ticket.title || `Ticket ${ticket.id.slice(0, 8)}`}
                    </h3>

                    {/* Show conclusion for resolved tickets */}
                    {ticket.status === "resolved" && ticket.conclusion && (
                      <p className="text-xs text-muted-foreground mb-2 line-clamp-1">
                        <span className="font-medium">Resolution:</span>{" "}
                        {ticket.conclusion}
                      </p>
                    )}

                    <div className="space-y-1.5">
                      <div className="flex items-center justify-between text-xs text-muted-foreground">
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
                        <span>{formatRelativeTime(ticket.createdAt)}</span>
                      </div>

                      <div className="flex items-center gap-3 text-xs text-muted-foreground">
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
                  </CardContent>
                </Card>
              ))}

              {ticketsByStatus[status].length === 0 && (
                <div className="flex items-center justify-center h-32 border-2 border-dashed border-muted rounded-lg">
                  <p className="text-sm text-muted-foreground">
                    No {status} tickets
                  </p>
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
  );
}

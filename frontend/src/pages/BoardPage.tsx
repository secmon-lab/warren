import { useQuery, useMutation } from "@apollo/client";
import { useNavigate } from "react-router-dom";
import { useState, useMemo } from "react";
import { Card, CardContent } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { CreateTicketModal } from "@/components/CreateTicketModal";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { GET_TICKETS, UPDATE_TICKET_STATUS, UPDATE_MULTIPLE_TICKETS_STATUS } from "@/lib/graphql/queries";
import { Ticket, TicketStatus, TICKET_STATUS_LABELS } from "@/lib/types";
import { MoreHorizontal, Archive, Plus } from "lucide-react";
import { useErrorToast, useSuccessToast } from "@/hooks/use-toast";
import { useConfirm } from "@/hooks/use-confirm";

const BOARD_STATUSES: TicketStatus[] = ["open", "pending", "resolved"];

export default function BoardPage() {
  const navigate = useNavigate();
  const [draggedTicket, setDraggedTicket] = useState<Ticket | null>(null);
  const [isUpdatingStatus, setIsUpdatingStatus] = useState(false);
  const [createModalOpen, setCreateModalOpen] = useState(false);

  const errorToast = useErrorToast();
  const successToast = useSuccessToast();
  const confirm = useConfirm();

  const [updateTicketStatus] = useMutation(UPDATE_TICKET_STATUS, {
    refetchQueries: [
      { query: GET_TICKETS, variables: { statuses: BOARD_STATUSES } },
    ],
  });

  const [updateMultipleTicketsStatus] = useMutation(UPDATE_MULTIPLE_TICKETS_STATUS, {
    refetchQueries: [
      { query: GET_TICKETS, variables: { statuses: BOARD_STATUSES } },
    ],
  });

  const { data, loading, error } = useQuery(GET_TICKETS, {
    variables: {
      statuses: BOARD_STATUSES,
    },
  });

  const tickets: Ticket[] = useMemo(
    () => data?.tickets?.tickets || [],
    [data?.tickets?.tickets]
  );

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

  const handleArchiveTicket = async (ticketId: string) => {
    if (isUpdatingStatus) return;

    const confirmed = await confirm({
      title: "Archive Ticket",
      description: "Are you sure you want to archive this ticket?",
      confirmText: "Archive",
      variant: "destructive",
    });

    if (!confirmed) return;

    setIsUpdatingStatus(true);
    try {
      await updateTicketStatus({
        variables: {
          id: ticketId,
          status: "archived",
        },
      });
      successToast("Ticket archived successfully");
    } catch (error) {
      console.error("Failed to archive ticket:", error);
      errorToast("Failed to archive ticket");
    } finally {
      setIsUpdatingStatus(false);
    }
  };

  const handleBulkArchiveResolved = async () => {
    if (isUpdatingStatus) return;

    const resolvedTickets = ticketsByStatus["resolved"];
    if (resolvedTickets.length === 0) {
      errorToast("No resolved tickets to archive");
      return;
    }

    const confirmed = await confirm({
      title: "Archive All Resolved Tickets",
      description: `Are you sure you want to archive all ${resolvedTickets.length} resolved tickets?`,
      confirmText: "Archive All",
      variant: "destructive",
    });

    if (!confirmed) return;

    setIsUpdatingStatus(true);
    try {
      // Archive all resolved tickets using bulk mutation
      await updateMultipleTicketsStatus({
        variables: {
          ids: resolvedTickets.map((ticket) => ticket.id),
          status: "archived",
        },
      });
      successToast(`${resolvedTickets.length} tickets archived successfully`);
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
    const errorMessage = (error as Error)?.message || String(error);
    
    // Check if this is an authentication/authorization error
    if (errorMessage.includes('Authentication required') ||
        errorMessage.includes('Invalid authentication token') ||
        errorMessage.includes('JSON.parse') ||
        errorMessage.includes('unexpected character')) {
      return (
        <div className="flex items-center justify-center h-64">
          <div className="text-center">
            <div className="text-lg text-red-600 mb-4">
              Authentication required
            </div>
            <div className="text-sm text-muted-foreground mb-4">
              Please log in to access the board
            </div>
            <Button 
              onClick={() => window.location.href = '/api/auth/login'}
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
          Error loading board: {errorMessage}
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold tracking-tight">Board</h1>
          <p className="text-muted-foreground">
            Kanban board view of security tickets
          </p>
        </div>
        <Button
          onClick={() => setCreateModalOpen(true)}
          className="flex items-center gap-2">
          <Plus className="h-4 w-4" />
          Create Ticket
        </Button>
      </div>

      <CreateTicketModal
        isOpen={createModalOpen}
        onClose={() => setCreateModalOpen(false)}
      />

      <div className="grid gap-6 lg:grid-cols-3">
        {BOARD_STATUSES.map((status) => (
          <div
            key={status}
            className="flex-1"
            onDragOver={handleDragOver}
            onDrop={(e) => handleDrop(e, status)}>
            <div className="mb-4">
              <h2 className="font-semibold text-lg flex items-center gap-2 justify-between">
                <div className="flex items-center gap-2">
                  <Badge variant="secondary" className="px-2">
                    {TICKET_STATUS_LABELS[status]}
                  </Badge>
                  <span className="text-sm text-muted-foreground">
                    ({ticketsByStatus[status].length})
                  </span>
                </div>
                {status === "resolved" && ticketsByStatus[status].length > 0 && (
                  <DropdownMenu>
                    <DropdownMenuTrigger asChild>
                      <Button
                        variant="ghost"
                        size="sm"
                        className="h-8 w-8 p-0"
                        disabled={isUpdatingStatus}>
                        <MoreHorizontal className="h-4 w-4" />
                      </Button>
                    </DropdownMenuTrigger>
                    <DropdownMenuContent align="end">
                      <DropdownMenuItem
                        onClick={handleBulkArchiveResolved}
                        className="text-destructive">
                        <Archive className="h-4 w-4 mr-2" />
                        Archive All Resolved ({ticketsByStatus[status].length})
                      </DropdownMenuItem>
                    </DropdownMenuContent>
                  </DropdownMenu>
                )}
              </h2>
            </div>

            <div
              className="space-y-2 min-h-[400px] p-2 border-2 border-dashed border-transparent rounded-lg transition-colors"
              style={{
                borderColor:
                  draggedTicket && draggedTicket.status !== status
                    ? "#e2e8f0"
                    : "transparent",
              }}>
              {ticketsByStatus[status].map((ticket) => (
                <Card
                  key={ticket.id}
                  className="mb-3 cursor-pointer hover:shadow-md transition-shadow"
                  draggable
                  onDragStart={(e) => handleDragStart(e, ticket)}
                  onClick={() => handleTicketClick(ticket.id)}>
                  <CardContent className="p-4">
                    <div className="space-y-2">
                      <div className="flex items-center justify-between">
                        <div className="flex items-center gap-2">
                          <Badge
                            variant={
                              status === "resolved" ? "outline" : "secondary"
                            }>
                            {TICKET_STATUS_LABELS[status]}
                          </Badge>
                          {ticket.isTest && (
                            <Badge
                              variant="outline"
                              className="bg-orange-50 text-orange-700 border-orange-200">
                              🧪 TEST
                            </Badge>
                          )}
                        </div>
                        <DropdownMenu>
                          <DropdownMenuTrigger asChild>
                            <Button
                              variant="ghost"
                              size="sm"
                              className="h-8 w-8 p-0"
                              onClick={(e) => e.stopPropagation()}>
                              <MoreHorizontal className="h-4 w-4" />
                            </Button>
                          </DropdownMenuTrigger>
                          <DropdownMenuContent align="end">
                            {status !== "archived" && (
                              <DropdownMenuItem
                                onClick={(e) => {
                                  e.stopPropagation();
                                  handleArchiveTicket(ticket.id);
                                }}>
                                <Archive className="h-4 w-4 mr-2" />
                                Archive
                              </DropdownMenuItem>
                            )}
                          </DropdownMenuContent>
                        </DropdownMenu>
                      </div>
                      <div>
                        <h3 className="font-medium text-sm leading-tight">
                          {ticket.isTest && "🧪 [TEST] "}
                          {ticket.title}
                        </h3>
                        {ticket.description && (
                          <p className="text-xs text-muted-foreground mt-1 line-clamp-2">
                            {ticket.description}
                          </p>
                        )}
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

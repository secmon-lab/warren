import { useQuery } from "@apollo/client";
import { useState, useMemo } from "react";
import { useNavigate } from "react-router-dom";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import {
  Pagination,
  PaginationContent,
  PaginationItem,
  PaginationLink,
  PaginationNext,
  PaginationPrevious,
} from "@/components/ui/pagination";
import { UserWithAvatar } from "@/components/ui/user-name";
import { CreateTicketModal } from "@/components/CreateTicketModal";
import { GET_TICKETS } from "@/lib/graphql/queries";
import { Ticket, TicketStatus, TICKET_STATUS_LABELS } from "@/lib/types";
import { AlertCircle, MessageSquare, User, Plus } from "lucide-react";

const ITEMS_PER_PAGE = 10;
const ALL_STATUSES: TicketStatus[] = [
  "open",
  "pending",
  "resolved",
  "archived",
];

// Badge variants for ticket statuses
const getStatusBadgeVariant = (status: TicketStatus) => {
  switch (status) {
    case "open":
      return "default";
    case "pending":
      return "secondary";
    case "resolved":
      return "outline";
    case "archived":
      return "secondary";
    default:
      return "secondary";
  }
};

export default function TicketsPage() {
  const [currentPage, setCurrentPage] = useState(1);
  const [selectedStatuses, setSelectedStatuses] = useState<TicketStatus[]>([]);
  const [activeTab, setActiveTab] = useState<"all" | TicketStatus>("all");
  const [createModalOpen, setCreateModalOpen] = useState(false);

  const navigate = useNavigate();

  const {
    data: ticketsData,
    loading: ticketsLoading,
    error: ticketsError,
  } = useQuery(GET_TICKETS, {
    variables: {
      statuses: selectedStatuses.length > 0 ? selectedStatuses : undefined,
      offset: (currentPage - 1) * ITEMS_PER_PAGE,
      limit: ITEMS_PER_PAGE,
    },
  });

  // Sort tickets by createdAt in descending order (newest first)
  const tickets: Ticket[] = useMemo(
    () =>
      [...(ticketsData?.tickets?.tickets || [])].sort(
        (a: Ticket, b: Ticket) =>
          new Date(b.createdAt).getTime() - new Date(a.createdAt).getTime()
      ),
    [ticketsData?.tickets?.tickets]
  );

  const handleStatusFilter = (status: TicketStatus | "all") => {
    if (status === "all") {
      setSelectedStatuses([]);
      setActiveTab("all");
    } else {
      setSelectedStatuses([status]);
      setActiveTab(status);
    }
    setCurrentPage(1);
  };

  const handleTicketClick = (ticketId: string) => {
    navigate(`/tickets/${ticketId}`);
  };

  if (ticketsLoading) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="text-lg">Loading tickets...</div>
      </div>
    );
  }

  if (ticketsError) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="text-lg text-red-600">
          Error loading tickets: {ticketsError.message}
        </div>
      </div>
    );
  }

  const totalPages = Math.ceil(
    (ticketsData?.tickets?.totalCount || 0) / ITEMS_PER_PAGE
  );

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold tracking-tight">Tickets</h1>
          <p className="text-muted-foreground">
            Manage and track security incidents
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

      <Tabs
        value={activeTab}
        onValueChange={(value) =>
          handleStatusFilter(value as TicketStatus | "all")
        }>
        <TabsList>
          <TabsTrigger value="all">All</TabsTrigger>
          {ALL_STATUSES.map((status) => (
            <TabsTrigger key={status} value={status}>
              {TICKET_STATUS_LABELS[status]}
            </TabsTrigger>
          ))}
        </TabsList>

        <TabsContent value={activeTab} className="space-y-4">
          {tickets.length === 0 ? (
            <Card>
              <CardContent className="flex items-center justify-center h-32">
                <p className="text-muted-foreground">No tickets found</p>
              </CardContent>
            </Card>
          ) : (
            <>
              <div className="space-y-3">
                {tickets.map((ticket) => (
                  <Card
                    key={ticket.id}
                    className="cursor-pointer hover:shadow-md transition-shadow"
                    onClick={() => handleTicketClick(ticket.id)}>
                    <CardHeader className="pb-3">
                      <div className="flex items-start justify-between">
                        <div className="flex-1 min-w-0">
                          <div className="flex items-center gap-2 mb-2">
                            <Badge
                              variant={getStatusBadgeVariant(
                                ticket.status as TicketStatus
                              )}>
                              {
                                TICKET_STATUS_LABELS[
                                  ticket.status as TicketStatus
                                ]
                              }
                            </Badge>
                            {ticket.isTest && (
                              <Badge
                                variant="outline"
                                className="bg-orange-50 text-orange-700 border-orange-200">
                                🧪 TEST
                              </Badge>
                            )}
                          </div>
                          <CardTitle className="text-lg leading-6 break-words">
                            {ticket.isTest && "🧪 [TEST] "}
                            {ticket.title}
                          </CardTitle>
                        </div>
                        <div className="flex items-center gap-2 ml-4 flex-shrink-0">
                          {ticket.alerts.length > 0 && (
                            <div className="flex items-center text-sm text-muted-foreground">
                              <AlertCircle className="h-4 w-4 mr-1" />
                              {ticket.alerts.length}
                            </div>
                          )}
                          {ticket.comments.length > 0 && (
                            <div className="flex items-center text-sm text-muted-foreground">
                              <MessageSquare className="h-4 w-4 mr-1" />
                              {ticket.comments.length}
                            </div>
                          )}
                        </div>
                      </div>
                    </CardHeader>

                    <CardContent className="pt-0 -mt-4">
                      <div className="flex items-center justify-between flex-wrap gap-4">
                        <div className="flex items-center gap-6 text-sm text-muted-foreground flex-wrap gap-y-2">
                          <div className="flex items-center gap-2">
                            <User className="h-4 w-4" />
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

                          <div className="flex items-center gap-2">
                            <MessageSquare className="h-4 w-4" />
                            <span>
                              {ticket.comments.length} comment
                              {ticket.comments.length !== 1 ? "s" : ""}
                            </span>
                          </div>

                          <div className="flex items-center gap-2">
                            <AlertCircle className="h-4 w-4" />
                            <span>
                              {ticket.alerts.length} alert
                              {ticket.alerts.length !== 1 ? "s" : ""}
                            </span>
                          </div>
                        </div>

                        <div className="text-xs text-muted-foreground shrink-0">
                          #{ticket.id}
                        </div>
                      </div>
                    </CardContent>
                  </Card>
                ))}
              </div>

              {totalPages > 1 && (
                <Pagination>
                  <PaginationContent>
                    <PaginationItem>
                      <PaginationPrevious
                        onClick={() =>
                          setCurrentPage(Math.max(1, currentPage - 1))
                        }
                        className={
                          currentPage === 1
                            ? "pointer-events-none opacity-50"
                            : "cursor-pointer"
                        }
                      />
                    </PaginationItem>
                    {Array.from({ length: totalPages }, (_, i) => i + 1).map(
                      (page) => (
                        <PaginationItem key={page}>
                          <PaginationLink
                            onClick={() => setCurrentPage(page)}
                            isActive={currentPage === page}
                            className="cursor-pointer">
                            {page}
                          </PaginationLink>
                        </PaginationItem>
                      )
                    )}
                    <PaginationItem>
                      <PaginationNext
                        onClick={() =>
                          setCurrentPage(Math.min(totalPages, currentPage + 1))
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
              )}
            </>
          )}
        </TabsContent>
      </Tabs>
    </div>
  );
}

import { useQuery } from "@apollo/client";
import { useState, useMemo } from "react";
import { useNavigate } from "react-router-dom";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
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
import { GET_TICKETS } from "@/lib/graphql/queries";
import {
  Ticket,
  TicketStatus,
  TICKET_STATUS_LABELS,
  TICKET_STATUS_COLORS,
  AlertConclusion,
  ALERT_CONCLUSION_LABELS,
} from "@/lib/types";
import { formatRelativeTime } from "@/lib/utils-extended";
import { AlertCircle, MessageSquare, User, Clock } from "lucide-react";

const ITEMS_PER_PAGE = 10;
const ALL_STATUSES: TicketStatus[] = [
  "open",
  "pending",
  "resolved",
  "archived",
];

export default function TicketsPage() {
  const [currentPage, setCurrentPage] = useState(1);
  const [selectedStatuses, setSelectedStatuses] = useState<TicketStatus[]>([]);
  const [activeTab, setActiveTab] = useState<"all" | TicketStatus>("all");

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
      <div>
        <h1 className="text-3xl font-bold tracking-tight">Tickets</h1>
        <p className="text-muted-foreground">
          Manage and track security incidents
        </p>
      </div>

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
                    className="hover:shadow-md hover:border-primary/20 transition-all duration-200 cursor-pointer"
                    onClick={() => handleTicketClick(ticket.id)}>
                    <CardHeader className="pb-0">
                      <div className="flex items-start justify-between gap-4">
                        <div className="flex-1 min-w-0">
                          <div className="flex items-center gap-3 mb-2 flex-wrap">
                            <Badge
                              className={
                                TICKET_STATUS_COLORS[
                                  ticket.status as TicketStatus
                                ]
                              }
                              variant="secondary">
                              {
                                TICKET_STATUS_LABELS[
                                  ticket.status as TicketStatus
                                ]
                              }
                            </Badge>
                            <CardTitle
                              className="text-lg leading-tight break-words min-w-0 flex-1"
                              title={
                                ticket.title ||
                                `Ticket ${ticket.id.slice(0, 8)}`
                              }>
                              {ticket.title ||
                                `Ticket ${ticket.id.slice(0, 8)}`}
                            </CardTitle>
                          </div>

                          {/* Show conclusion and reason for resolved tickets */}
                          {ticket.status === "resolved" && (
                            <div className="mt-3">
                              <div className="flex items-start gap-3 flex-wrap">
                                {ticket.conclusion && (
                                  <Badge
                                    variant="outline"
                                    className="font-normal">
                                    {ALERT_CONCLUSION_LABELS[
                                      ticket.conclusion as AlertConclusion
                                    ] || ticket.conclusion}
                                  </Badge>
                                )}
                                {ticket.reason && (
                                  <div className="text-sm text-muted-foreground leading-relaxed flex-1 min-w-0">
                                    {ticket.reason}
                                  </div>
                                )}
                              </div>
                            </div>
                          )}
                        </div>

                        <div className="text-right text-sm text-muted-foreground shrink-0">
                          <div className="flex items-center gap-1">
                            <Clock className="h-4 w-4" />
                            {formatRelativeTime(ticket.createdAt)}
                          </div>
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

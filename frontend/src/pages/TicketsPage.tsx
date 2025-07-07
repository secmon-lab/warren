import { useQuery } from "@apollo/client";
import { useState } from "react";
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
  const [activeTab, setActiveTab] = useState<"all" | TicketStatus>("open");
  const [createModalOpen, setCreateModalOpen] = useState(false);

  const navigate = useNavigate();

  // Derive selectedStatuses from activeTab. No useState needed for it.
  const selectedStatuses = activeTab === "all" ? [] : [activeTab];

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

  // Backend already sorts by createdAt DESC, no need to sort again
  const tickets: Ticket[] = ticketsData?.tickets?.tickets || [];

  const handleStatusFilter = (status: TicketStatus | "all") => {
    setActiveTab(status);
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
    // Check if this is an authentication/authorization error
    if (ticketsError.message?.includes('Authentication required') ||
        ticketsError.message?.includes('Invalid authentication token') ||
        ticketsError.message?.includes('JSON.parse') ||
        ticketsError.message?.includes('unexpected character')) {
      return (
        <div className="flex items-center justify-center h-64">
          <div className="text-center">
            <div className="text-lg text-red-600 mb-4">
              Authentication required
            </div>
            <div className="text-sm text-muted-foreground mb-4">
              Please log in to access tickets
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
                          {ticket.alertsCount > 0 && (
                            <div className="flex items-center text-sm text-muted-foreground">
                              <AlertCircle className="h-4 w-4 mr-1" />
                              {ticket.alertsCount}
                            </div>
                          )}
                          {ticket.commentsCount > 0 && (
                            <div className="flex items-center text-sm text-muted-foreground">
                              <MessageSquare className="h-4 w-4 mr-1" />
                              {ticket.commentsCount}
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
                              {ticket.commentsCount} comment
                              {ticket.commentsCount !== 1 ? "s" : ""}
                            </span>
                          </div>

                          <div className="flex items-center gap-2">
                            <AlertCircle className="h-4 w-4" />
                            <span>
                              {ticket.alertsCount} alert
                              {ticket.alertsCount !== 1 ? "s" : ""}
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
                    {/* Show page numbers with truncation */}
                    {(() => {
                      const maxVisiblePages = 10;
                      const pageNumbers = [];
                      
                      if (totalPages <= maxVisiblePages) {
                        // Show all pages if total is 10 or less
                        for (let i = 1; i <= totalPages; i++) {
                          pageNumbers.push(i);
                        }
                      } else {
                        // Show truncated pagination for more than 10 pages
                        const startPage = Math.max(1, currentPage - 4);
                        const endPage = Math.min(totalPages, currentPage + 4);
                        
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
                        if (endPage < totalPages) {
                          if (endPage < totalPages - 1) {
                            pageNumbers.push('...');
                          }
                          pageNumbers.push(totalPages);
                        }
                      }
                      
                      return pageNumbers.map((page, index) => (
                        <PaginationItem key={index}>
                          {page === '...' ? (
                            <span className="px-3 py-2 text-sm text-muted-foreground">...</span>
                          ) : (
                            <PaginationLink
                              isActive={page === currentPage}
                              onClick={() => setCurrentPage(page as number)}
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

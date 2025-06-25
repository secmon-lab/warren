import { useState } from "react";
import { useQuery } from "@apollo/client";
import { useNavigate } from "react-router-dom";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
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
import { GET_SIMILAR_TICKETS } from "@/lib/graphql/queries";
import { Ticket, TICKET_STATUS_COLORS, TICKET_STATUS_LABELS } from "@/lib/types";
import { formatRelativeTime } from "@/lib/utils-extended";
import { TicketStatus } from "@/lib/types";
import { Hash, Users } from "lucide-react";

interface SimilarTicketsProps {
  ticketId: string;
}

const ITEMS_PER_PAGE = 5;

export function SimilarTickets({ ticketId }: SimilarTicketsProps) {
  const [threshold, setThreshold] = useState([0.95]);
  const [currentPage, setCurrentPage] = useState(1);
  const navigate = useNavigate();

  const offset = (currentPage - 1) * ITEMS_PER_PAGE;

  const { data, loading, error, refetch } = useQuery(GET_SIMILAR_TICKETS, {
    variables: {
      ticketId,
      threshold: threshold[0],
      offset,
      limit: ITEMS_PER_PAGE,
    },
    fetchPolicy: "cache-and-network",
  });

  const handleThresholdChange = (value: number[]) => {
    setThreshold(value);
    setCurrentPage(1); // Reset to first page when threshold changes
    // Trigger refetch with new threshold
    refetch({
      ticketId,
      threshold: value[0],
      offset: 0,
      limit: ITEMS_PER_PAGE,
    });
  };

  const handleTicketClick = (ticket: Ticket) => {
    navigate(`/tickets/${ticket.id}`);
  };

  const tickets: Ticket[] = data?.similarTickets?.tickets || [];
  const totalCount = data?.similarTickets?.totalCount || 0;
  const totalPages = Math.ceil(totalCount / ITEMS_PER_PAGE);

  const handlePageChange = (page: number) => {
    setCurrentPage(page);
  };

  return (
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
              {threshold[0].toFixed(2)}
            </span>
          </div>
          <Slider
            id="threshold"
            min={0}
            max={1}
            step={0.01}
            value={threshold}
            onValueChange={setThreshold}
            onValueCommit={handleThresholdChange}
            className="w-full"
          />
          <div className="flex justify-between text-xs text-muted-foreground">
            <span>0.0 (Less similar)</span>
            <span>1.0 (More similar)</span>
          </div>
        </div>

        {/* Loading State */}
        {loading && (
          <div className="flex items-center justify-center py-8">
            <div className="text-sm text-muted-foreground">Loading similar tickets...</div>
          </div>
        )}

        {/* Error State */}
        {error && (
          <div className="flex items-center justify-center py-8">
            <div className="text-sm text-red-600">
              Error loading similar tickets: {error.message}
            </div>
          </div>
        )}

        {/* Empty State */}
        {!loading && !error && tickets.length === 0 && (
          <div className="flex items-center justify-center py-8">
            <div className="text-center">
              <Users className="h-8 w-8 mx-auto mb-2 text-muted-foreground" />
              <div className="text-sm text-muted-foreground">
                No similar tickets found with threshold {threshold[0].toFixed(2)}
              </div>
            </div>
          </div>
        )}

        {/* Tickets List */}
        {!loading && !error && tickets.length > 0 && (
          <div className="space-y-3">
            {tickets.map((ticket) => (
              <div
                key={ticket.id}
                className="p-3 border rounded-lg cursor-pointer hover:bg-muted/50 transition-colors"
                onClick={() => handleTicketClick(ticket)}
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
                  <Badge
                    className={TICKET_STATUS_COLORS[ticket.status as TicketStatus]}
                    variant="secondary"
                  >
                    {TICKET_STATUS_LABELS[ticket.status as TicketStatus]}
                  </Badge>
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
        {!loading && !error && totalPages > 1 && (
          <div className="pt-4">
            <Pagination>
              <PaginationContent>
                <PaginationItem>
                  <PaginationPrevious
                    onClick={() => {
                      if (currentPage > 1) handlePageChange(currentPage - 1);
                    }}
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
                    onClick={() => {
                      if (currentPage < totalPages) handlePageChange(currentPage + 1);
                    }}
                    className={
                      currentPage === totalPages
                        ? "pointer-events-none opacity-50"
                        : "cursor-pointer"
                    }
                  />
                </PaginationItem>
              </PaginationContent>
            </Pagination>

            <div className="text-center mt-2">
              <span className="text-xs text-muted-foreground">
                Showing {Math.min((currentPage - 1) * ITEMS_PER_PAGE + 1, totalCount)}-
                {Math.min(currentPage * ITEMS_PER_PAGE, totalCount)} of {totalCount} tickets
              </span>
            </div>
          </div>
        )}
      </CardContent>
    </Card>
  );
}
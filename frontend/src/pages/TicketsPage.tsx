import { useQuery, useMutation } from "@apollo/client";
import { useState, useMemo } from "react";
import { useNavigate } from "react-router-dom";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
} from "@/components/ui/select";
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
import { GET_TICKETS, GET_TAGS, ARCHIVE_TICKETS } from "@/lib/graphql/queries";
import { Ticket, TicketStatus, TagMetadata } from "@/lib/types";
import { AlertCircle, MessageSquare, Plus, Tag, Archive, CircleDot, CheckCircle2, ArchiveIcon, Search, X } from "lucide-react";
import { generateTagColor } from "@/lib/tag-colors";

const ITEMS_PER_PAGE = 10;

type ActiveTab = "active" | "archived";

const ACTIVE_STATUSES: TicketStatus[] = ["open", "resolved"];
const ARCHIVED_STATUSES: TicketStatus[] = ["archived"];

const getStatusIcon = (status: TicketStatus) => {
  switch (status) {
    case "open":
      return <CircleDot className="h-[18px] w-[18px] text-blue-500 shrink-0 mt-0.5" />;
    case "resolved":
      return <CheckCircle2 className="h-[18px] w-[18px] text-green-500 shrink-0 mt-0.5" />;
    case "archived":
      return <ArchiveIcon className="h-[18px] w-[18px] text-gray-400 shrink-0 mt-0.5" />;
    default:
      return <CircleDot className="h-[18px] w-[18px] text-gray-400 shrink-0 mt-0.5" />;
  }
};

const CONCLUSION_CONFIG: Record<string, { label: string; className: string }> = {
  intended: { label: 'Intended', className: 'bg-green-100 text-green-700 border-green-200' },
  unaffected: { label: 'Unaffected', className: 'bg-blue-100 text-blue-700 border-blue-200' },
  false_positive: { label: 'False Positive', className: 'bg-gray-100 text-gray-700 border-gray-200' },
  true_positive: { label: 'True Positive', className: 'bg-red-100 text-red-700 border-red-200' },
  escalated: { label: 'Escalated', className: 'bg-orange-100 text-orange-700 border-orange-200' },
};

export default function TicketsPage() {
  const [currentPage, setCurrentPage] = useState(1);
  const [activeTab, setActiveTab] = useState<ActiveTab>("active");
  const [createModalOpen, setCreateModalOpen] = useState(false);

  // Filter state
  const [statusFilter, setStatusFilter] = useState<"all" | "open" | "resolved">("all");
  const [assigneeFilter, setAssigneeFilter] = useState<string>("all");
  const [keywordFilter, setKeywordFilter] = useState<string>("");
  const [keywordInput, setKeywordInput] = useState<string>("");

  const navigate = useNavigate();

  // Compute the statuses to query based on tab + status filter
  const selectedStatuses = useMemo(() => {
    if (activeTab === "archived") return ARCHIVED_STATUSES;
    if (statusFilter === "open") return ["open"] as TicketStatus[];
    if (statusFilter === "resolved") return ["resolved"] as TicketStatus[];
    return ACTIVE_STATUSES;
  }, [activeTab, statusFilter]);

  const {
    data: ticketsData,
    loading: ticketsLoading,
    error: ticketsError,
    refetch,
  } = useQuery(GET_TICKETS, {
    variables: {
      statuses: selectedStatuses,
      keyword: keywordFilter || undefined,
      assigneeID: assigneeFilter !== "all" ? assigneeFilter : undefined,
      offset: (currentPage - 1) * ITEMS_PER_PAGE,
      limit: ITEMS_PER_PAGE,
    },
  });

  const { data: tagsData } = useQuery(GET_TAGS);
  const tagsByName = new Map((tagsData?.tags || []).map((t: TagMetadata) => [t.name, t]));

  const [archiveTickets, { loading: archiving }] = useMutation(ARCHIVE_TICKETS);

  const tickets: Ticket[] = ticketsData?.tickets?.tickets || [];
  const resolvedTickets = tickets.filter(t => t.status === "resolved");

  // Collect unique assignees from current result set for the dropdown
  const assigneeOptions = useMemo(() => {
    const map = new Map<string, string>();
    for (const t of tickets) {
      if (t.assignee) {
        map.set(t.assignee.id, t.assignee.name);
      }
    }
    return Array.from(map.entries()).map(([id, name]) => ({ id, name }));
  }, [tickets]);

  const handleTabChange = (tab: string) => {
    setActiveTab(tab as ActiveTab);
    setCurrentPage(1);
    setStatusFilter("all");
    setAssigneeFilter("all");
    setKeywordFilter("");
    setKeywordInput("");
  };

  const handleFilterChange = () => {
    setCurrentPage(1);
  };

  const handleStatusFilterChange = (value: string) => {
    setStatusFilter(value as "all" | "open" | "resolved");
    handleFilterChange();
  };

  const handleAssigneeFilterChange = (value: string) => {
    setAssigneeFilter(value);
    handleFilterChange();
  };

  const handleKeywordSearch = () => {
    setKeywordFilter(keywordInput);
    setCurrentPage(1);
  };

  const handleKeywordClear = () => {
    setKeywordInput("");
    setKeywordFilter("");
    setCurrentPage(1);
  };

  const handleArchiveAllResolved = async () => {
    const ids = resolvedTickets.map(t => t.id);
    if (ids.length === 0) return;
    await archiveTickets({ variables: { ids } });
    await refetch();
  };

  if (ticketsLoading) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="text-muted-foreground">Loading tickets...</div>
      </div>
    );
  }

  if (ticketsError) {
    if (ticketsError.message?.includes('Authentication required') ||
        ticketsError.message?.includes('Invalid authentication token') ||
        ticketsError.message?.includes('JSON.parse') ||
        ticketsError.message?.includes('unexpected character')) {
      return (
        <div className="flex items-center justify-center h-64">
          <div className="text-center">
            <div className="text-lg text-red-600 mb-4">Authentication required</div>
            <div className="text-sm text-muted-foreground mb-4">Please log in to access tickets</div>
            <Button onClick={() => window.location.href = '/api/auth/login'} className="flex items-center gap-2">
              Sign In with Slack
            </Button>
          </div>
        </div>
      );
    }
    return (
      <div className="flex items-center justify-center h-64">
        <div className="text-red-600">Error loading tickets: {ticketsError.message}</div>
      </div>
    );
  }

  const totalPages = Math.ceil((ticketsData?.tickets?.totalCount || 0) / ITEMS_PER_PAGE);

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold tracking-tight">Tickets</h1>
          <p className="text-sm text-muted-foreground">Manage and track security incidents</p>
        </div>
        <Button onClick={() => setCreateModalOpen(true)} size="sm" className="flex items-center gap-2">
          <Plus className="h-4 w-4" />
          New Ticket
        </Button>
      </div>

      <CreateTicketModal isOpen={createModalOpen} onClose={() => setCreateModalOpen(false)} />

      <Tabs value={activeTab} onValueChange={handleTabChange}>
        <div className="flex items-center justify-between mb-3">
          <TabsList className="h-8">
            <TabsTrigger value="active" className="text-sm px-3 h-7">Active</TabsTrigger>
            <TabsTrigger value="archived" className="text-sm px-3 h-7">Archived</TabsTrigger>
          </TabsList>

          {activeTab === "active" && resolvedTickets.length > 0 && (
            <Button
              variant="outline"
              size="sm"
              onClick={handleArchiveAllResolved}
              disabled={archiving}
              className="h-8 text-xs flex items-center gap-1.5">
              <Archive className="h-3.5 w-3.5" />
              Archive All Resolved ({resolvedTickets.length})
            </Button>
          )}
        </div>

        {/* Filter bar */}
        <div className="flex items-center gap-2 mb-3">
          {/* Status filter — only shown in Active tab */}
          {activeTab === "active" && (
            <Select value={statusFilter} onValueChange={handleStatusFilterChange}>
              <SelectTrigger className={`h-8 w-36 text-xs transition-colors ${
                statusFilter !== "all"
                  ? "border-blue-500 bg-blue-50 text-blue-700 font-medium"
                  : ""
              }`}>
                {statusFilter === "open" ? (
                  <span className="flex items-center gap-1.5">
                    <CircleDot className="h-3.5 w-3.5 text-blue-500 shrink-0" />
                    Open
                  </span>
                ) : statusFilter === "resolved" ? (
                  <span className="flex items-center gap-1.5">
                    <CheckCircle2 className="h-3.5 w-3.5 text-green-500 shrink-0" />
                    Resolved
                  </span>
                ) : (
                  <span className="text-muted-foreground">All Status</span>
                )}
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">
                  <span className="text-muted-foreground">All Status</span>
                </SelectItem>
                <SelectItem value="open">
                  <span className="flex items-center gap-1.5">
                    <CircleDot className="h-3.5 w-3.5 text-blue-500 shrink-0" />
                    Open
                  </span>
                </SelectItem>
                <SelectItem value="resolved">
                  <span className="flex items-center gap-1.5">
                    <CheckCircle2 className="h-3.5 w-3.5 text-green-500 shrink-0" />
                    Resolved
                  </span>
                </SelectItem>
              </SelectContent>
            </Select>
          )}

          {/* Assignee filter */}
          <Select value={assigneeFilter} onValueChange={handleAssigneeFilterChange}>
            <SelectTrigger className={`h-8 w-44 text-xs transition-colors ${
              assigneeFilter !== "all"
                ? "border-blue-500 bg-blue-50 text-blue-700 font-medium"
                : ""
            }`}>
              {assigneeFilter !== "all" ? (
                <UserWithAvatar userID={assigneeFilter} avatarSize="sm" />
              ) : (
                <span className="text-muted-foreground">All Assignees</span>
              )}
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">All Assignees</SelectItem>
              {assigneeOptions.map(a => (
                <SelectItem key={a.id} value={a.id}>
                  <UserWithAvatar userID={a.id} fallback={a.name} avatarSize="sm" />
                </SelectItem>
              ))}
            </SelectContent>
          </Select>

          {/* Keyword search */}
          <div className="flex items-center gap-1 flex-1 max-w-72">
            <div className="relative flex-1">
              <Search className={`absolute left-2 top-1/2 -translate-y-1/2 h-3.5 w-3.5 pointer-events-none ${
                keywordFilter ? "text-blue-500" : "text-muted-foreground"
              }`} />
              <Input
                className={`h-8 text-xs pl-7 pr-7 transition-colors ${
                  keywordFilter ? "border-blue-500 bg-blue-50" : ""
                }`}
                placeholder="Search keyword..."
                value={keywordInput}
                onChange={e => setKeywordInput(e.target.value)}
                onKeyDown={e => e.key === "Enter" && handleKeywordSearch()}
              />
              {keywordInput && (
                <button
                  onClick={handleKeywordClear}
                  className="absolute right-2 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground">
                  <X className="h-3.5 w-3.5" />
                </button>
              )}
            </div>
            <Button variant="outline" size="sm" className="h-8 px-2 text-xs" onClick={handleKeywordSearch}>
              Search
            </Button>
          </div>

          {/* Clear all filters button — shown when any filter is active */}
          {(statusFilter !== "all" || assigneeFilter !== "all" || keywordFilter) && (
            <button
              onClick={() => {
                setStatusFilter("all");
                setAssigneeFilter("all");
                setKeywordFilter("");
                setKeywordInput("");
                setCurrentPage(1);
              }}
              className="ml-auto flex items-center gap-1 text-xs text-blue-600 hover:text-blue-800 font-medium whitespace-nowrap">
              <X className="h-3 w-3" />
              Clear filters
            </button>
          )}
        </div>

        <TabsContent value={activeTab} className="mt-0">
          {tickets.length === 0 ? (
            <div className="border rounded-md flex items-center justify-center h-24 text-sm text-muted-foreground">
              No tickets found
            </div>
          ) : (
            <>
              <div className="border rounded-md divide-y">
                {tickets.map((ticket) => {
                  const conclusion = ticket.conclusion ? CONCLUSION_CONFIG[ticket.conclusion.toLowerCase()] : null;
                  return (
                    <div
                      key={ticket.id}
                      className="flex items-start gap-3 px-4 py-3.5 hover:bg-muted/40 cursor-pointer transition-colors"
                      onClick={() => navigate(`/tickets/${ticket.id}`)}>
                      {/* Status icon */}
                      {getStatusIcon(ticket.status as TicketStatus)}

                      {/* Main content */}
                      <div className="flex-1 min-w-0">
                        <div className="flex items-start gap-2 flex-wrap">
                          <span className="font-semibold text-[0.9rem] leading-snug break-words">
                            {ticket.isTest && (
                              <span className="text-orange-500 mr-1">[TEST]</span>
                            )}
                            {ticket.title}
                          </span>
                          {/* Tags */}
                          {ticket.tags && ticket.tags.map((tag, i) => {
                            const tagData = tagsByName.get(tag) as TagMetadata | undefined;
                            const colorClass = tagData?.color || generateTagColor(tag);
                            return (
                              <Badge key={i} className={`text-xs px-1.5 py-0 h-5 mt-0.5 ${colorClass}`}>
                                <Tag className="h-2.5 w-2.5 mr-1" />
                                {tag}
                              </Badge>
                            );
                          })}
                        </div>

                        {/* Conclusion line */}
                        {(ticket.status === 'resolved' || ticket.status === 'archived') && conclusion && (
                          <div className="flex items-center gap-2 mt-1.5">
                            <Badge className={`text-xs px-2 py-0.5 h-auto font-medium border ${conclusion.className}`}>
                              {conclusion.label}
                            </Badge>
                            {ticket.reason && (
                              <span className="text-xs text-muted-foreground truncate max-w-lg">
                                {ticket.reason}
                              </span>
                            )}
                          </div>
                        )}

                        {/* Meta row */}
                        <div className="flex items-center gap-3 mt-1.5 text-xs text-muted-foreground">
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
                      </div>

                      {/* Right-side counters */}
                      <div className="flex items-center gap-4 shrink-0 text-xs text-muted-foreground ml-2 mt-0.5">
                        {ticket.alertsCount > 0 && (
                          <span className="flex items-center gap-1">
                            <AlertCircle className="h-3.5 w-3.5" />
                            {ticket.alertsCount}
                          </span>
                        )}
                        {ticket.commentsCount > 0 && (
                          <span className="flex items-center gap-1">
                            <MessageSquare className="h-3.5 w-3.5" />
                            {ticket.commentsCount}
                          </span>
                        )}
                      </div>
                    </div>
                  );
                })}
              </div>

              {totalPages > 1 && (
                <div className="mt-4">
                  <Pagination>
                    <PaginationContent>
                      <PaginationItem>
                        <PaginationPrevious
                          onClick={() => setCurrentPage(Math.max(1, currentPage - 1))}
                          className={currentPage === 1 ? "pointer-events-none opacity-50" : "cursor-pointer"}
                        />
                      </PaginationItem>
                      {(() => {
                        const maxVisiblePages = 10;
                        const pageNumbers: (number | string)[] = [];
                        if (totalPages <= maxVisiblePages) {
                          for (let i = 1; i <= totalPages; i++) pageNumbers.push(i);
                        } else {
                          const startPage = Math.max(1, currentPage - 4);
                          const endPage = Math.min(totalPages, currentPage + 4);
                          if (startPage > 1) {
                            pageNumbers.push(1);
                            if (startPage > 2) pageNumbers.push('...');
                          }
                          for (let i = startPage; i <= endPage; i++) pageNumbers.push(i);
                          if (endPage < totalPages) {
                            if (endPage < totalPages - 1) pageNumbers.push('...');
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
                          onClick={() => setCurrentPage(Math.min(totalPages, currentPage + 1))}
                          className={currentPage === totalPages ? "pointer-events-none opacity-50" : "cursor-pointer"}
                        />
                      </PaginationItem>
                    </PaginationContent>
                  </Pagination>
                </div>
              )}
            </>
          )}
        </TabsContent>
      </Tabs>
    </div>
  );
}

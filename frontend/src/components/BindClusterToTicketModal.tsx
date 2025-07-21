import { useState, memo, useCallback } from "react";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { formatTimeAgo, getSeverityColor } from "@/lib/utils";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
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
  useClusterAlertsQuery, 
  useBindClusterToTicketMutation,
  useGetTicketsQuery 
} from "@/lib/graphql/generated";
import { useToast } from "@/hooks/use-toast";
import { 
  AlertTriangle, 
  Search, 
  Clock, 
  Link2,
  FileText,
  CheckCircle,
  Loader2,
  Ticket
} from "lucide-react";

interface BindClusterToTicketModalProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  clusterId: string;
  clusterSize: number;
  onSuccess?: () => void;
}

const BindClusterToTicketModal = memo(({
  open,
  onOpenChange,
  clusterId,
  clusterSize,
  onSuccess,
}: BindClusterToTicketModalProps) => {
  const [step, setStep] = useState<'select-ticket' | 'select-alerts'>('select-ticket');
  const [selectedTicketId, setSelectedTicketId] = useState<string>("");
  const [selectedAlerts, setSelectedAlerts] = useState<Set<string>>(new Set());
  const [searchKeyword, setSearchKeyword] = useState("");
  const [ticketSearchQuery, setTicketSearchQuery] = useState("");
  const [currentPage, setCurrentPage] = useState(1);
  const [ticketPage, setTicketPage] = useState(1);
  const ITEMS_PER_PAGE = 10;
  const { toast } = useToast();

  // Fetch tickets for selection
  const {
    data: ticketsData,
    loading: ticketsLoading,
    error: ticketsError,
  } = useGetTicketsQuery({
    variables: {
      offset: (ticketPage - 1) * ITEMS_PER_PAGE,
      limit: ITEMS_PER_PAGE,
    },
    skip: !open || step !== 'select-ticket',
  });

  // Fetch cluster alerts for selection
  const {
    data: clusterAlertsData,
    loading: alertsLoading,
    error: alertsError,
    refetch: refetchAlerts,
  } = useClusterAlertsQuery({
    variables: {
      clusterID: clusterId,
      keyword: searchKeyword.trim() || undefined,
      limit: ITEMS_PER_PAGE,
      offset: (currentPage - 1) * ITEMS_PER_PAGE,
    },
    skip: !open || step !== 'select-alerts',
  });

  const [bindClusterToTicket, { loading: bindingToTicket }] = useBindClusterToTicketMutation({
    onCompleted: (data) => {
      toast({
        title: "Alerts Bound to Ticket",
        description: `Successfully bound ${selectedAlerts.size} alerts to ticket "${data.bindClusterToTicket.title}"`,
      });
      onSuccess?.();
      handleClose();
    },
    onError: (error) => {
      toast({
        variant: "destructive",
        title: "Failed to Bind Alerts",
        description: error.message,
      });
    },
  });

  const handleTicketSelect = useCallback((ticketId: string) => {
    setSelectedTicketId(ticketId);
    setStep('select-alerts');
  }, []);

  const handlePageChange = useCallback((page: number) => {
    setCurrentPage(page);
  }, []);

  const handleTicketPageChange = useCallback((page: number) => {
    setTicketPage(page);
  }, []);

  const handleSearch = useCallback((keyword: string) => {
    setSearchKeyword(keyword);
    setCurrentPage(1);
    setSelectedAlerts(new Set());
  }, []);

  const handleSelectAlert = useCallback((alertId: string, checked: boolean) => {
    setSelectedAlerts(prev => {
      const newSelected = new Set(prev);
      if (checked) {
        newSelected.add(alertId);
      } else {
        newSelected.delete(alertId);
      }
      return newSelected;
    });
  }, []);

  const handleSelectAll = useCallback((checked: boolean) => {
    if (checked && clusterAlertsData?.clusterAlerts?.alerts) {
      setSelectedAlerts(prev => {
        const newSelected = new Set(prev);
        clusterAlertsData.clusterAlerts.alerts.forEach((alert) => {
          newSelected.add(alert.id);
        });
        return newSelected;
      });
    } else {
      setSelectedAlerts(new Set());
    }
  }, [clusterAlertsData?.clusterAlerts?.alerts]);

  const handleBindToTicket = useCallback(async () => {
    if (selectedAlerts.size === 0) {
      toast({
        variant: "destructive",
        title: "No Alerts Selected",
        description: "Please select at least one alert to bind to the ticket.",
      });
      return;
    }

    if (!selectedTicketId) {
      toast({
        variant: "destructive",
        title: "No Ticket Selected",
        description: "Please select a ticket first.",
      });
      return;
    }

    try {
      await bindClusterToTicket({
        variables: {
          clusterID: clusterId,
          ticketID: selectedTicketId,
          alertIDs: Array.from(selectedAlerts),
        },
      });
    } catch (error) {
      // Error handling is done in onError callback
    }
  }, [selectedAlerts, selectedTicketId, bindClusterToTicket, clusterId, toast]);

  const handleClose = useCallback(() => {
    setStep('select-ticket');
    setSelectedTicketId("");
    setSelectedAlerts(new Set());
    setSearchKeyword("");
    setTicketSearchQuery("");
    setCurrentPage(1);
    setTicketPage(1);
    onOpenChange(false);
  }, [onOpenChange]);

  const handleBack = useCallback(() => {
    setStep('select-ticket');
    setSelectedAlerts(new Set());
    setSearchKeyword("");
    setCurrentPage(1);
  }, []);


  const getStatusColor = useCallback((status: string) => {
    switch (status.toLowerCase()) {
      case 'open':
        return 'bg-green-100 text-green-800 border-green-200';
      case 'investigating':
        return 'bg-blue-100 text-blue-800 border-blue-200';
      case 'resolved':
        return 'bg-gray-100 text-gray-800 border-gray-200';
      case 'closed':
        return 'bg-gray-100 text-gray-800 border-gray-200';
      default:
        return 'bg-gray-100 text-gray-800 border-gray-200';
    }
  }, []);

  // Filter tickets by search query
  const filteredTickets = ticketsData?.tickets?.tickets?.filter(ticket =>
    ticketSearchQuery.trim() === "" ||
    ticket.title.toLowerCase().includes(ticketSearchQuery.toLowerCase()) ||
    ticket.description.toLowerCase().includes(ticketSearchQuery.toLowerCase())
  ) || [];

  const selectedTicket = ticketsData?.tickets?.tickets?.find(t => t.id === selectedTicketId);

  if (step === 'select-ticket') {
    const totalTickets = ticketsData?.tickets?.totalCount || 0;
    const totalTicketPages = Math.ceil(totalTickets / ITEMS_PER_PAGE);

    return (
      <Dialog open={open} onOpenChange={onOpenChange}>
        <DialogContent className="max-w-3xl max-h-[80vh] flex flex-col">
          <DialogHeader className="flex-shrink-0">
            <DialogTitle className="flex items-center gap-2">
              <Link2 className="h-5 w-5" />
              Bind Cluster <span className="text-blue-600 font-mono font-semibold">{clusterId}</span> to Ticket
            </DialogTitle>
            <DialogDescription>
              Select a ticket to bind alerts from this cluster
            </DialogDescription>
          </DialogHeader>

          {/* Ticket Search */}
          <div className="flex-shrink-0 space-y-3">
            <div className="relative">
              <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 h-4 w-4 text-muted-foreground" />
              <Input
                placeholder="Search tickets by title or description..."
                value={ticketSearchQuery}
                onChange={(e) => setTicketSearchQuery(e.target.value)}
                className="pl-10"
              />
            </div>
          </div>

          {/* Tickets List */}
          <div className="flex-1 overflow-auto">
            {ticketsLoading ? (
              <div className="space-y-3">
                {[...Array(6)].map((_, i) => (
                  <div key={i} className="flex items-center gap-3 p-4 border rounded animate-pulse">
                    <div className="h-4 w-4 bg-gray-200 rounded" />
                    <div className="flex-1 space-y-2">
                      <div className="h-4 bg-gray-200 rounded w-3/4" />
                      <div className="h-3 bg-gray-200 rounded w-1/2" />
                    </div>
                  </div>
                ))}
              </div>
            ) : ticketsError ? (
              <div className="text-center py-8">
                <p className="text-red-600 mb-4">Failed to load tickets: {ticketsError.message}</p>
                <Button onClick={() => window.location.reload()} variant="outline">
                  Retry
                </Button>
              </div>
            ) : filteredTickets.length === 0 ? (
              <div className="text-center py-8">
                <Ticket className="h-12 w-12 mx-auto text-muted-foreground mb-4" />
                <h3 className="text-lg font-medium mb-2">No tickets found</h3>
                <p className="text-muted-foreground">
                  {ticketSearchQuery ? 
                    `No tickets match the search term "${ticketSearchQuery}".` :
                    "No open tickets available."
                  }
                </p>
              </div>
            ) : (
              <div className="space-y-2">
                {filteredTickets.map((ticket) => (
                  <div 
                    key={ticket.id}
                    className="flex items-start gap-3 p-4 border rounded hover:bg-muted/50 transition-colors cursor-pointer"
                    onClick={() => handleTicketSelect(ticket.id)}
                  >
                    <Ticket className="h-4 w-4 text-blue-500 flex-shrink-0 mt-1" />
                    
                    <div className="flex-1 min-w-0">
                      <div className="flex items-start justify-between gap-3 mb-2">
                        <div className="flex-1 min-w-0">
                          <h4 className="font-medium text-sm mb-1 line-clamp-1">
                            {ticket.title}
                          </h4>
                          <p className="text-xs text-muted-foreground line-clamp-2 mb-2">
                            {ticket.description}
                          </p>
                        </div>
                        
                        <div className="flex items-center gap-2 flex-shrink-0">
                          <Badge 
                            variant="secondary" 
                            className={`text-xs ${getStatusColor(ticket.status)}`}
                          >
                            {ticket.status}
                          </Badge>
                          <Badge variant="outline" className="text-xs">
                            {ticket.alertsCount} alerts
                          </Badge>
                        </div>
                      </div>
                      
                      <div className="flex items-center gap-4 text-xs text-muted-foreground">
                        <div className="flex items-center gap-1">
                          <Clock className="h-3 w-3" />
                          <span>{formatTimeAgo(ticket.createdAt)}</span>
                        </div>
                        {ticket.assignee && (
                          <span>Assigned to {ticket.assignee.name}</span>
                        )}
                        <span className="truncate">ID: {ticket.id}</span>
                      </div>
                    </div>
                  </div>
                ))}
              </div>
            )}
          </div>

          {/* Pagination */}
          {totalTicketPages > 1 && (
            <div className="flex-shrink-0 flex justify-center pt-4 border-t">
              <Pagination>
                <PaginationContent>
                  <PaginationItem>
                    <PaginationPrevious
                      href="#"
                      onClick={(e) => {
                        e.preventDefault();
                        if (ticketPage > 1) handleTicketPageChange(ticketPage - 1);
                      }}
                      className={ticketPage <= 1 ? "pointer-events-none opacity-50" : ""}
                    />
                  </PaginationItem>
                  
                  {[...Array(Math.min(5, totalTicketPages))].map((_, i) => {
                    const page = i + 1;
                    return (
                      <PaginationItem key={page}>
                        <PaginationLink
                          href="#"
                          onClick={(e) => {
                            e.preventDefault();
                            handleTicketPageChange(page);
                          }}
                          isActive={ticketPage === page}
                        >
                          {page}
                        </PaginationLink>
                      </PaginationItem>
                    );
                  })}
                  
                  <PaginationItem>
                    <PaginationNext
                      href="#"
                      onClick={(e) => {
                        e.preventDefault();
                        if (ticketPage < totalTicketPages) handleTicketPageChange(ticketPage + 1);
                      }}
                      className={ticketPage >= totalTicketPages ? "pointer-events-none opacity-50" : ""}
                    />
                  </PaginationItem>
                </PaginationContent>
              </Pagination>
            </div>
          )}

          {/* Actions */}
          <div className="flex-shrink-0 flex justify-end gap-2 pt-4 border-t">
            <Button onClick={handleClose} variant="outline">
              Cancel
            </Button>
          </div>
        </DialogContent>
      </Dialog>
    );
  }

  // Step 2: Select alerts
  const totalCount = clusterAlertsData?.clusterAlerts?.totalCount || 0;
  const totalPages = Math.ceil(totalCount / ITEMS_PER_PAGE);
  const alerts = clusterAlertsData?.clusterAlerts?.alerts || [];
  const currentPageAlerts = alerts.map(alert => alert.id);
  const allCurrentPageSelected = currentPageAlerts.length > 0 && 
    currentPageAlerts.every(id => selectedAlerts.has(id));

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-4xl max-h-[90vh] flex flex-col">
        <DialogHeader className="flex-shrink-0">
          <DialogTitle className="flex items-center gap-2">
            <Link2 className="h-5 w-5" />
            Select Alerts to Bind
          </DialogTitle>
          <DialogDescription>
            Select alerts from cluster <span className="text-blue-600 font-mono font-semibold">{clusterId}</span> to bind to ticket "{selectedTicket?.title}"
            {searchKeyword && ` (${totalCount} matching "${searchKeyword}")`}
          </DialogDescription>
        </DialogHeader>

        {/* Selected Ticket Info */}
        {selectedTicket && (
          <div className="flex-shrink-0 p-3 bg-blue-50 border border-blue-200 rounded-lg">
            <div className="flex items-center gap-2 mb-1">
              <Ticket className="h-4 w-4 text-blue-600" />
              <span className="font-medium text-sm">{selectedTicket.title}</span>
              <Badge variant="secondary" className={`text-xs ${getStatusColor(selectedTicket.status)}`}>
                {selectedTicket.status}
              </Badge>
            </div>
            <p className="text-xs text-muted-foreground">
              {selectedTicket.description}
            </p>
          </div>
        )}

        {/* Search and Filter */}
        <div className="flex-shrink-0 space-y-3">
          <div className="flex gap-4">
            <div className="flex-1 relative">
              <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 h-4 w-4 text-muted-foreground" />
              <Input
                placeholder="Filter alerts by data content..."
                value={searchKeyword}
                onChange={(e) => handleSearch(e.target.value)}
                className="pl-10"
              />
            </div>
            <Button onClick={() => refetchAlerts()} variant="outline" size="sm">
              Refresh
            </Button>
          </div>

          <div className="flex justify-between items-center">
            <div className="flex items-center gap-4">
              <div className="text-sm text-muted-foreground">
                {searchKeyword ? `${totalCount} matching alerts` : `${clusterSize} total alerts`}
              </div>
              {alerts.length > 0 && (
                <div className="flex items-center space-x-2">
                  <Checkbox
                    id="select-all"
                    checked={allCurrentPageSelected}
                    onCheckedChange={handleSelectAll}
                  />
                  <Label htmlFor="select-all" className="text-sm">
                    Select all {searchKeyword ? 'filtered' : totalCount} alerts
                  </Label>
                </div>
              )}
            </div>
            <div className="flex items-center gap-2">
              <CheckCircle className="h-4 w-4 text-green-600" />
              <span className="text-sm font-medium">
                {selectedAlerts.size} alert{selectedAlerts.size !== 1 ? 's' : ''} selected
              </span>
            </div>
          </div>
        </div>

        {/* Alerts List */}
        <div className="flex-1 overflow-auto">
          {alertsLoading ? (
            <div className="space-y-3">
              {[...Array(6)].map((_, i) => (
                <div key={i} className="flex items-center gap-3 p-3 border rounded animate-pulse">
                  <div className="h-4 w-4 bg-gray-200 rounded" />
                  <div className="h-4 w-4 bg-gray-200 rounded" />
                  <div className="flex-1 space-y-2">
                    <div className="h-4 bg-gray-200 rounded w-3/4" />
                    <div className="h-3 bg-gray-200 rounded w-1/2" />
                  </div>
                </div>
              ))}
            </div>
          ) : alertsError ? (
            <div className="text-center py-8">
              <p className="text-red-600 mb-4">Failed to load alerts: {alertsError.message}</p>
              <Button onClick={() => refetchAlerts()} variant="outline">
                Retry
              </Button>
            </div>
          ) : alerts.length === 0 ? (
            <div className="text-center py-8">
              <FileText className="h-12 w-12 mx-auto text-muted-foreground mb-4" />
              <h3 className="text-lg font-medium mb-2">No alerts found</h3>
              <p className="text-muted-foreground mb-4">
                {searchKeyword ? 
                  `No alerts match the search term "${searchKeyword}".` :
                  "This cluster appears to be empty."
                }
              </p>
              {searchKeyword && (
                <Button onClick={() => handleSearch("")} variant="outline">
                  Clear Search
                </Button>
              )}
            </div>
          ) : (
            <div className="space-y-2">
              {alerts.map((alert) => (
                <div 
                  key={alert.id} 
                  className={`flex items-start gap-3 p-3 border rounded transition-colors ${
                    selectedAlerts.has(alert.id) ? 'bg-blue-50 border-blue-200' : 'hover:bg-muted/50'
                  }`}
                >
                  <Checkbox
                    checked={selectedAlerts.has(alert.id)}
                    onCheckedChange={(checked) => handleSelectAlert(alert.id, !!checked)}
                    className="mt-1"
                  />
                  <AlertTriangle className="h-4 w-4 text-amber-500 flex-shrink-0 mt-1" />
                  
                  <div className="flex-1 min-w-0">
                    <div className="flex items-start justify-between gap-3 mb-1">
                      <div className="flex-1 min-w-0">
                        <h4 className="font-medium text-sm mb-1 line-clamp-1">
                          {alert.title}
                        </h4>
                        {alert.description && (
                          <p className="text-xs text-muted-foreground line-clamp-1 mb-1">
                            {alert.description}
                          </p>
                        )}
                      </div>
                      
                      <div className="flex items-center gap-2 flex-shrink-0">
                        <Badge 
                          variant="secondary" 
                          className={`text-xs ${getSeverityColor(alert.schema)}`}
                        >
                          {alert.schema}
                        </Badge>
                        {alert.ticket && (
                          <Badge variant="outline" className="text-xs">
                            Already assigned
                          </Badge>
                        )}
                      </div>
                    </div>
                    
                    <div className="flex items-center gap-4 text-xs text-muted-foreground">
                      <div className="flex items-center gap-1">
                        <Clock className="h-3 w-3" />
                        <span>{formatTimeAgo(alert.createdAt)}</span>
                      </div>
                    </div>
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>

        {/* Pagination */}
        {totalPages > 1 && (
          <div className="flex-shrink-0 flex justify-center pt-3 border-t">
            <Pagination>
              <PaginationContent>
                <PaginationItem>
                  <PaginationPrevious
                    href="#"
                    onClick={(e) => {
                      e.preventDefault();
                      if (currentPage > 1) handlePageChange(currentPage - 1);
                    }}
                    className={currentPage <= 1 ? "pointer-events-none opacity-50" : ""}
                  />
                </PaginationItem>
                
                {[...Array(Math.min(5, totalPages))].map((_, i) => {
                  const page = i + 1;
                  return (
                    <PaginationItem key={page}>
                      <PaginationLink
                        href="#"
                        onClick={(e) => {
                          e.preventDefault();
                          handlePageChange(page);
                        }}
                        isActive={currentPage === page}
                      >
                        {page}
                      </PaginationLink>
                    </PaginationItem>
                  );
                })}
                
                <PaginationItem>
                  <PaginationNext
                    href="#"
                    onClick={(e) => {
                      e.preventDefault();
                      if (currentPage < totalPages) handlePageChange(currentPage + 1);
                    }}
                    className={currentPage >= totalPages ? "pointer-events-none opacity-50" : ""}
                  />
                </PaginationItem>
              </PaginationContent>
            </Pagination>
          </div>
        )}

        {/* Actions */}
        <div className="flex-shrink-0 flex justify-between pt-4 border-t">
          <Button onClick={handleBack} variant="outline" disabled={bindingToTicket}>
            Back
          </Button>
          <div className="flex gap-2">
            <Button onClick={handleClose} variant="outline" disabled={bindingToTicket}>
              Cancel
            </Button>
            <Button 
              onClick={handleBindToTicket} 
              disabled={selectedAlerts.size === 0 || bindingToTicket}
            >
              {bindingToTicket ? (
                <>
                  <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                  Binding...
                </>
              ) : (
                <>
                  <Link2 className="h-4 w-4 mr-2" />
                  Bind {selectedAlerts.size} Alert{selectedAlerts.size !== 1 ? 's' : ''}
                </>
              )}
            </Button>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  );
});

BindClusterToTicketModal.displayName = 'BindClusterToTicketModal';

export default BindClusterToTicketModal;
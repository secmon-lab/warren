import { useState, memo, useCallback } from "react";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { Checkbox } from "@/components/ui/checkbox";
import {
  Pagination,
  PaginationContent,
  PaginationItem,
  PaginationLink,
  PaginationNext,
  PaginationPrevious,
} from "@/components/ui/pagination";
import { useClusterAlertsQuery, useCreateTicketFromClusterMutation } from "@/lib/graphql/generated";
import { useToast } from "@/hooks/use-toast";
import { 
  AlertTriangle, 
  Search, 
  Clock, 
  Plus,
  FileText,
  CheckCircle,
  Loader2
} from "lucide-react";

interface CreateTicketFromClusterModalProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  clusterId: string;
  clusterSize: number;
  onSuccess?: () => void;
}

const CreateTicketFromClusterModal = memo(({
  open,
  onOpenChange,
  clusterId,
  clusterSize,
  onSuccess,
}: CreateTicketFromClusterModalProps) => {
  const [currentPage, setCurrentPage] = useState(1);
  const [searchKeyword, setSearchKeyword] = useState("");
  const [selectedAlerts, setSelectedAlerts] = useState<Set<string>>(new Set());
  const [ticketTitle, setTicketTitle] = useState("");
  const [ticketDescription, setTicketDescription] = useState("");
  const ITEMS_PER_PAGE = 10;
  const { toast } = useToast();

  const {
    data: clusterAlertsData,
    loading: alertsLoading,
    error: alertsError,
    refetch,
  } = useClusterAlertsQuery({
    variables: {
      clusterID: clusterId,
      keyword: searchKeyword.trim() || undefined,
      limit: ITEMS_PER_PAGE,
      offset: (currentPage - 1) * ITEMS_PER_PAGE,
    },
    skip: !open,
  });

  const [createTicketFromCluster, { loading: creatingTicket }] = useCreateTicketFromClusterMutation({
    onCompleted: (data) => {
      toast({
        title: "Ticket Created",
        description: `Successfully created ticket "${data.createTicketFromCluster.title}"`,
      });
      onSuccess?.();
      handleClose();
    },
    onError: (error) => {
      toast({
        variant: "destructive",
        title: "Failed to Create Ticket",
        description: error.message,
      });
    },
  });

  const handlePageChange = useCallback((page: number) => {
    setCurrentPage(page);
  }, []);

  const handleSearch = useCallback((keyword: string) => {
    setSearchKeyword(keyword);
    setCurrentPage(1);
    setSelectedAlerts(new Set()); // Clear selection when searching
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

  const handleCreateTicket = useCallback(async () => {
    if (selectedAlerts.size === 0) {
      toast({
        variant: "destructive",
        title: "No Alerts Selected",
        description: "Please select at least one alert to create a ticket.",
      });
      return;
    }

    // Title is optional - will be auto-generated if not provided

    try {
      await createTicketFromCluster({
        variables: {
          clusterID: clusterId,
          alertIDs: Array.from(selectedAlerts),
          title: ticketTitle.trim() || undefined, // Send undefined if empty
          description: ticketDescription.trim() || undefined,
        },
      });
    } catch (error) {
      // Error handling is done in onError callback
    }
  }, [selectedAlerts, ticketTitle, ticketDescription, createTicketFromCluster, clusterId, toast]);

  const handleClose = useCallback(() => {
    setCurrentPage(1);
    setSearchKeyword("");
    setSelectedAlerts(new Set());
    setTicketTitle("");
    setTicketDescription("");
    onOpenChange(false);
  }, [onOpenChange]);

  const formatTimeAgo = useCallback((dateString: string) => {
    const date = new Date(dateString);
    const now = new Date();
    const diffInHours = Math.floor((now.getTime() - date.getTime()) / (1000 * 60 * 60));
    
    if (diffInHours < 1) return "Just now";
    if (diffInHours < 24) return `${diffInHours}h ago`;
    const diffInDays = Math.floor(diffInHours / 24);
    if (diffInDays < 7) return `${diffInDays}d ago`;
    return date.toLocaleDateString();
  }, []);

  const getSeverityColor = useCallback((schema: string) => {
    const lowerSchema = schema.toLowerCase();
    if (lowerSchema.includes('critical') || lowerSchema.includes('high')) {
      return 'bg-red-100 text-red-800 border-red-200';
    }
    if (lowerSchema.includes('medium') || lowerSchema.includes('warning')) {
      return 'bg-yellow-100 text-yellow-800 border-yellow-200';
    }
    if (lowerSchema.includes('low') || lowerSchema.includes('info')) {
      return 'bg-blue-100 text-blue-800 border-blue-200';
    }
    return 'bg-gray-100 text-gray-800 border-gray-200';
  }, []);

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
            <Plus className="h-5 w-5" />
            Create Ticket from Cluster <span className="text-blue-600 font-mono font-semibold">{clusterId}</span>
          </DialogTitle>
          <DialogDescription>
            Select alerts from this cluster to include in the new ticket
            {searchKeyword && ` (${totalCount} matching "${searchKeyword}")`}
          </DialogDescription>
        </DialogHeader>

        {/* Ticket Details */}
        <div className="flex-shrink-0 space-y-4 border-b pb-4">
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            <div className="space-y-2">
              <Label htmlFor="ticket-title">Ticket Title</Label>
              <Input
                id="ticket-title"
                placeholder="Auto-generated if empty"
                value={ticketTitle}
                onChange={(e) => setTicketTitle(e.target.value)}
              />
            </div>
            <div className="space-y-2">
              <Label>Selected Alerts</Label>
              <div className="flex items-center gap-2 h-10 px-3 border rounded-md bg-muted/50">
                <CheckCircle className="h-4 w-4 text-green-600" />
                <span className="text-sm font-medium">
                  {selectedAlerts.size} alert{selectedAlerts.size !== 1 ? 's' : ''} selected
                </span>
              </div>
            </div>
          </div>
          <div className="space-y-2">
            <Label htmlFor="ticket-description">Description (Optional)</Label>
            <Textarea
              id="ticket-description"
              placeholder="Enter ticket description..."
              value={ticketDescription}
              onChange={(e) => setTicketDescription(e.target.value)}
              rows={3}
            />
          </div>
        </div>

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
            <Button onClick={() => refetch()} variant="outline" size="sm">
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
              <Button onClick={() => refetch()} variant="outline">
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
                            Assigned
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
        <div className="flex-shrink-0 flex justify-end gap-2 pt-4 border-t">
          <Button onClick={handleClose} variant="outline" disabled={creatingTicket}>
            Cancel
          </Button>
          <Button 
            onClick={handleCreateTicket} 
            disabled={selectedAlerts.size === 0 || creatingTicket}
          >
            {creatingTicket ? (
              <>
                <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                Creating...
              </>
            ) : (
              <>
                <Plus className="h-4 w-4 mr-2" />
                Create Ticket ({selectedAlerts.size} alert{selectedAlerts.size !== 1 ? 's' : ''})
              </>
            )}
          </Button>
        </div>
      </DialogContent>
    </Dialog>
  );
});

CreateTicketFromClusterModal.displayName = 'CreateTicketFromClusterModal';

export default CreateTicketFromClusterModal;
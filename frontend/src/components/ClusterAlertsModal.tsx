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
import {
  Pagination,
  PaginationContent,
  PaginationItem,
  PaginationLink,
  PaginationNext,
  PaginationPrevious,
} from "@/components/ui/pagination";
import { useClusterAlertsQuery } from "@/lib/graphql/generated";
import { 
  AlertTriangle, 
  Search, 
  Clock, 
  Plus, 
  Link2,
  FileText
} from "lucide-react";
import { Link } from "react-router-dom";

interface ClusterAlertsModalProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  clusterId: string;
  clusterSize: number;
  onCreateTicket: (clusterId: string) => void;
  onBindToTicket: (clusterId: string) => void;
}

const ClusterAlertsModal = memo(({
  open,
  onOpenChange,
  clusterId,
  clusterSize,
  onCreateTicket,
  onBindToTicket,
}: ClusterAlertsModalProps) => {
  const [currentPage, setCurrentPage] = useState(1);
  const [searchKeyword, setSearchKeyword] = useState("");
  const ITEMS_PER_PAGE = 20;

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
    skip: !open, // Only fetch when modal is open
  });

  const handlePageChange = useCallback((page: number) => {
    setCurrentPage(page);
  }, []);

  const handleSearch = useCallback((keyword: string) => {
    setSearchKeyword(keyword);
    setCurrentPage(1); // Reset to first page when searching
  }, []);

  const handleCreateTicket = useCallback(() => {
    onCreateTicket(clusterId);
  }, [clusterId, onCreateTicket]);

  const handleBindToTicket = useCallback(() => {
    onBindToTicket(clusterId);
  }, [clusterId, onBindToTicket]);


  const totalCount = clusterAlertsData?.clusterAlerts?.totalCount || 0;
  const totalPages = Math.ceil(totalCount / ITEMS_PER_PAGE);
  const alerts = clusterAlertsData?.clusterAlerts?.alerts || [];

  // Reset page when modal opens
  const handleOpenChange = (newOpen: boolean) => {
    if (newOpen) {
      setCurrentPage(1);
      setSearchKeyword("");
    }
    onOpenChange(newOpen);
  };

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent className="max-w-4xl max-h-[80vh] flex flex-col">
        <DialogHeader className="flex-shrink-0">
          <DialogTitle className="flex items-center gap-2">
            <AlertTriangle className="h-5 w-5" />
            Cluster <span className="text-blue-600 font-mono font-semibold">{clusterId}</span> Details
          </DialogTitle>
          <DialogDescription>
            {clusterSize} alerts in this cluster
            {searchKeyword && ` (${totalCount} matching "${searchKeyword}")`}
          </DialogDescription>
        </DialogHeader>

        {/* Search and Actions */}
        <div className="flex-shrink-0 space-y-4">
          <div className="flex gap-4">
            <div className="flex-1 relative">
              <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 h-4 w-4 text-muted-foreground" />
              <Input
                placeholder="Search alerts by data content..."
                value={searchKeyword}
                onChange={(e) => handleSearch(e.target.value)}
                className="pl-10"
              />
            </div>
            <Button onClick={() => refetch()} variant="outline">
              Refresh
            </Button>
          </div>

          <div className="flex justify-between items-center">
            <div className="text-sm text-muted-foreground">
              {searchKeyword ? `${totalCount} matching alerts` : `${clusterSize} total alerts`}
            </div>
            <div className="flex gap-2">
              <Button
                onClick={handleCreateTicket}
                size="sm"
                disabled={alerts.length === 0}
              >
                <Plus className="h-4 w-4 mr-2" />
                Create Ticket
              </Button>
              <Button
                onClick={handleBindToTicket}
                size="sm"
                variant="outline"
                disabled={alerts.length === 0}
              >
                <Link2 className="h-4 w-4 mr-2" />
                Bind to Ticket
              </Button>
            </div>
          </div>
        </div>

        {/* Alerts List */}
        <div className="flex-1 overflow-auto">
          {alertsLoading ? (
            <div className="space-y-3">
              {[...Array(6)].map((_, i) => (
                <div key={i} className="flex items-center gap-3 p-4 border rounded animate-pulse">
                  <div className="h-4 w-4 bg-gray-200 rounded" />
                  <div className="flex-1 space-y-2">
                    <div className="h-4 bg-gray-200 rounded w-3/4" />
                    <div className="h-3 bg-gray-200 rounded w-1/2" />
                  </div>
                  <div className="h-6 w-16 bg-gray-200 rounded" />
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
            <div className="space-y-3">
              {alerts.map((alert) => (
                <Link 
                  key={alert.id} 
                  to={`/alerts/${alert.id}`}
                  className="block"
                >
                  <div className="flex items-start gap-3 p-4 border rounded hover:bg-muted/50 transition-colors cursor-pointer">
                    <AlertTriangle className="h-4 w-4 text-amber-500 flex-shrink-0 mt-1" />
                    
                    <div className="flex-1 min-w-0">
                      <div className="flex items-start justify-between gap-3 mb-2">
                        <div className="flex-1 min-w-0">
                          <h4 className="font-medium text-sm mb-1 line-clamp-1 hover:text-blue-600 transition-colors">
                            {alert.title}
                          </h4>
                          {alert.description && (
                            <p className="text-xs text-muted-foreground line-clamp-2 mb-2">
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
                              Ticket #{alert.ticket.id}
                            </Badge>
                          )}
                        </div>
                      </div>
                      
                      <div className="flex items-center gap-4 text-xs text-muted-foreground">
                        <div className="flex items-center gap-1">
                          <Clock className="h-3 w-3" />
                          <span>{formatTimeAgo(alert.createdAt)}</span>
                        </div>
                        <span className="truncate">ID: {alert.id}</span>
                      </div>
                    </div>
                  </div>
                </Link>
              ))}
            </div>
          )}
        </div>

        {/* Pagination */}
        {totalPages > 1 && (
          <div className="flex-shrink-0 flex justify-center pt-4 border-t">
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
      </DialogContent>
    </Dialog>
  );
});

ClusterAlertsModal.displayName = 'ClusterAlertsModal';

export default ClusterAlertsModal;
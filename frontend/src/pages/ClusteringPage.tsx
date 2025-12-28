import { useState, useMemo, useCallback, memo } from "react";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Slider } from "@/components/ui/slider";
import { Label } from "@/components/ui/label";
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from "@/components/ui/collapsible";
import {
  Pagination,
  PaginationContent,
  PaginationItem,
  PaginationLink,
  PaginationNext,
  PaginationPrevious,
} from "@/components/ui/pagination";
import { useAlertClustersQuery } from "@/lib/graphql/generated";
import { AlertTriangle, Search, Settings, ChevronDown, RefreshCw } from "lucide-react";
import ClusterCard from "@/components/ClusterCard";
import ClusterAlertsModal from "@/components/ClusterAlertsModal";
import CreateTicketFromClusterModal from "@/components/CreateTicketFromClusterModal";
import BindClusterToTicketModal from "@/components/BindClusterToTicketModal";

const ClusteringPage = memo(() => {
  const [currentPage, setCurrentPage] = useState(1);
  const [searchQuery, setSearchQuery] = useState("");
  const [minClusterSize, setMinClusterSize] = useState("1");
  const [eps, setEps] = useState("0.15");
  const [minSamples, setMinSamples] = useState("2");
  const [tempEps, setTempEps] = useState("0.15"); // Temporary value while sliding
  const [tempMinSamples, setTempMinSamples] = useState("2"); // Temporary value while sliding
  const [showAdvanced, setShowAdvanced] = useState(false);
  const [selectedCluster, setSelectedCluster] = useState<{id: string, size: number} | null>(null);
  const [createTicketModal, setCreateTicketModal] = useState<{id: string, size: number} | null>(null);
  const [bindTicketModal, setBindTicketModal] = useState<{id: string, size: number} | null>(null);
  const ITEMS_PER_PAGE = 20;

  const {
    data: clustersData,
    loading: clustersLoading,
    error: clustersError,
    refetch: refetchClusters,
  } = useAlertClustersQuery({
    variables: {
      offset: (currentPage - 1) * ITEMS_PER_PAGE,
      limit: ITEMS_PER_PAGE,
      minClusterSize: parseInt(minClusterSize),
      eps: parseFloat(eps),
      minSamples: parseInt(minSamples),
      keyword: searchQuery.trim() || undefined,
    },
    pollInterval: 30000, // Poll every 30 seconds for updates
  });

  // Get clusters directly from backend (already filtered)
  const clusters = useMemo(() => {
    return clustersData?.alertClusters?.clusters || [];
  }, [clustersData?.alertClusters?.clusters]);

  // Calculate pagination values from backend totalCount
  const totalClusters = clustersData?.alertClusters?.totalCount || 0;
  const totalPages = Math.ceil(totalClusters / ITEMS_PER_PAGE);
  // Backend already handles pagination, so we use clusters directly
  const paginatedClusters = clusters;

  const handlePageChange = useCallback((page: number) => {
    setCurrentPage(page);
    // Backend will handle fetching the correct page
  }, []);

  const handleRefresh = useCallback(() => {
    refetchClusters();
  }, [refetchClusters]);

  const handleCreateTicket = useCallback((clusterId: string) => {
    const cluster = clusters.find(c => c.id === clusterId);
    if (cluster) {
      setCreateTicketModal({ id: clusterId, size: cluster.size });
    }
  }, [clusters]);

  const handleBindToTicket = useCallback((clusterId: string) => {
    const cluster = clusters.find(c => c.id === clusterId);
    if (cluster) {
      setBindTicketModal({ id: clusterId, size: cluster.size });
    }
  }, [clusters]);

  const handleViewDetails = useCallback((clusterId: string) => {
    const cluster = clusters.find(c => c.id === clusterId);
    if (cluster) {
      setSelectedCluster({ id: clusterId, size: cluster.size });
    }
  }, [clusters]);

  if (clustersLoading) {
    return (
      <div className="space-y-6 p-6">
        <div className="flex items-center gap-2">
          <AlertTriangle className="h-6 w-6" />
          <h1 className="text-2xl font-bold">Alert Clusters</h1>
        </div>
        <div className="grid gap-4">
          {[...Array(6)].map((_, i) => (
            <Card key={i} className="animate-pulse">
              <CardHeader>
                <div className="h-4 bg-gray-200 rounded w-1/3"></div>
              </CardHeader>
              <CardContent>
                <div className="space-y-2">
                  <div className="h-3 bg-gray-200 rounded w-full"></div>
                  <div className="h-3 bg-gray-200 rounded w-2/3"></div>
                </div>
              </CardContent>
            </Card>
          ))}
        </div>
      </div>
    );
  }

  if (clustersError) {
    return (
      <div className="space-y-6 p-6">
        <div className="flex items-center gap-2">
          <AlertTriangle className="h-6 w-6" />
          <h1 className="text-2xl font-bold">Alert Clusters</h1>
        </div>
        <Card className="border-red-200 bg-red-50">
          <CardHeader>
            <CardTitle className="text-red-700">Error Loading Clusters</CardTitle>
          </CardHeader>
          <CardContent>
            <p className="text-red-600 mb-4">{clustersError.message}</p>
            <Button onClick={handleRefresh} variant="outline">
              Try Again
            </Button>
          </CardContent>
        </Card>
      </div>
    );
  }

  const computedAt = clustersData?.alertClusters?.computedAt;
  const parameters = clustersData?.alertClusters?.parameters;
  const noiseCount = clustersData?.alertClusters?.noiseAlerts?.length || 0;

  return (
    <div className="space-y-6 p-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          <AlertTriangle className="h-6 w-6" />
          <h1 className="text-2xl font-bold">Alert Clusters</h1>
        </div>
        <div className="flex items-center gap-2">
          <Button onClick={handleRefresh} variant="outline" size="sm">
            <RefreshCw className="h-4 w-4 mr-2" />
            Refresh
          </Button>
        </div>
      </div>

      {/* Clustering Info */}
      {computedAt && (
        <Card>
          <CardContent className="pt-6">
            <div className="grid grid-cols-1 md:grid-cols-4 gap-4 text-sm">
              <div>
                <p className="text-muted-foreground">Computed At</p>
                <p className="font-medium">{new Date(computedAt).toISOString().split('T')[0].replace(/-/g, '/')} {new Date(computedAt).toISOString().split('T')[1].split('.')[0]}</p>
              </div>
              <div>
                <p className="text-muted-foreground">Total Clusters</p>
                <p className="font-medium">{totalClusters}</p>
              </div>
              <div>
                <p className="text-muted-foreground">Noise Alerts</p>
                <p className="font-medium">{noiseCount}</p>
              </div>
              <div>
                <p className="text-muted-foreground">Parameters</p>
                <p className="font-medium">
                  eps: {parameters?.eps}, min: {parameters?.minSamples}
                </p>
              </div>
            </div>
          </CardContent>
        </Card>
      )}

      {/* Filters */}
      <Card>
        <CardContent className="pt-6">
          <div className="flex flex-col md:flex-row gap-4">
            <div className="flex-1">
              <div className="relative">
                <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 h-4 w-4 text-muted-foreground" />
                <Input
                  placeholder="Search clusters by alert data or keywords..."
                  value={searchQuery}
                  onChange={(e) => {
                    setSearchQuery(e.target.value);
                    setCurrentPage(1); // Reset to first page when searching
                  }}
                  className="pl-10"
                />
              </div>
            </div>
            <div className="w-full md:w-48">
              <Select value={minClusterSize} onValueChange={(value) => {
                setMinClusterSize(value);
                setCurrentPage(1); // Reset to first page when changing filter
              }}>
                <SelectTrigger>
                  <SelectValue placeholder="Min cluster size" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="1">Any size</SelectItem>
                  <SelectItem value="2">2+ alerts</SelectItem>
                  <SelectItem value="3">3+ alerts</SelectItem>
                  <SelectItem value="5">5+ alerts</SelectItem>
                  <SelectItem value="10">10+ alerts</SelectItem>
                </SelectContent>
              </Select>
            </div>
          </div>
          
          {/* Advanced Parameters */}
          <Collapsible open={showAdvanced} onOpenChange={setShowAdvanced} className="mt-4">
            <CollapsibleTrigger asChild>
              <Button variant="outline" size="sm" className="gap-2">
                <Settings className="h-4 w-4" />
                Advanced Parameters
                <ChevronDown className={`h-4 w-4 transition-transform ${showAdvanced ? 'rotate-180' : ''}`} />
              </Button>
            </CollapsibleTrigger>
            <CollapsibleContent className="mt-4">
              <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
                <div className="space-y-2">
                  <Label htmlFor="eps-slider">
                    Epsilon (eps): {tempEps}
                  </Label>
                  <Slider
                    id="eps-slider"
                    min={0.01}
                    max={0.5}
                    step={0.01}
                    value={[parseFloat(tempEps)]}
                    onValueChange={(value) => {
                      setTempEps(value[0].toFixed(2)); // Update temporary value with 2 decimal places
                    }}
                    onValueCommit={(value) => {
                      setEps(value[0].toFixed(2)); // Apply value when sliding ends
                      setCurrentPage(1); // Reset to first page when parameters change
                    }}
                    className="w-full"
                  />
                </div>
                
                <div className="space-y-2">
                  <Label htmlFor="minsamples-slider">
                    Min Samples: {tempMinSamples}
                  </Label>
                  <Slider
                    id="minsamples-slider"
                    min={2}
                    max={10}
                    step={1}
                    value={[parseInt(tempMinSamples)]}
                    onValueChange={(value) => {
                      setTempMinSamples(value[0].toString()); // Update temporary value
                    }}
                    onValueCommit={(value) => {
                      setMinSamples(value[0].toString()); // Apply value when sliding ends
                      setCurrentPage(1); // Reset to first page when parameters change
                    }}
                    className="w-full"
                  />
                </div>
              </div>
              
              <div className="mt-4 p-3 bg-muted/50 rounded-lg">
                <p className="text-sm text-muted-foreground">
                  <strong>Tip:</strong> Start with default values (eps=0.15, minSamples=2). 
                  Decrease eps for more clusters, increase for fewer clusters. 
                  Adjust minSamples based on your minimum cluster size preference.
                </p>
              </div>
            </CollapsibleContent>
          </Collapsible>
        </CardContent>
      </Card>

      {/* Clusters Grid */}
      {paginatedClusters.length > 0 ? (
        <div className="grid gap-4">
          {paginatedClusters.map((cluster) => (
            <ClusterCard
              key={cluster.id}
              cluster={cluster}
              onCreateTicket={handleCreateTicket}
              onBindToTicket={handleBindToTicket}
              onViewDetails={handleViewDetails}
            />
          ))}
        </div>
      ) : (
        <Card>
          <CardContent className="py-12 text-center">
            <AlertTriangle className="h-12 w-12 mx-auto text-muted-foreground mb-4" />
            <h3 className="text-lg font-medium mb-2">No clusters found</h3>
            <p className="text-muted-foreground mb-4">
              {searchQuery ? 
                "No clusters match your search criteria. Try adjusting your filters." :
                "No alert clusters are available. Clusters are computed automatically from unbound alerts."
              }
            </p>
            {searchQuery && (
              <Button 
                onClick={() => setSearchQuery("")} 
                variant="outline"
              >
                Clear Search
              </Button>
            )}
          </CardContent>
        </Card>
      )}

      {/* Pagination */}
      {totalPages > 1 && (
        <div className="flex justify-center">
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

      {/* Cluster Details Modal */}
      {selectedCluster && (
        <ClusterAlertsModal
          open={!!selectedCluster}
          onOpenChange={(open) => !open && setSelectedCluster(null)}
          clusterId={selectedCluster.id}
          clusterSize={selectedCluster.size}
          onCreateTicket={(clusterId: string) => {
            setCreateTicketModal({ id: clusterId, size: selectedCluster?.size || 0 });
          }}
          onBindToTicket={(clusterId: string) => {
            setBindTicketModal({ id: clusterId, size: selectedCluster?.size || 0 });
          }}
        />
      )}

      {/* Create Ticket Modal */}
      {createTicketModal && (
        <CreateTicketFromClusterModal
          open={!!createTicketModal}
          onOpenChange={(open) => !open && setCreateTicketModal(null)}
          clusterId={createTicketModal.id}
          clusterSize={createTicketModal.size}
          onSuccess={() => {
            refetchClusters();
          }}
        />
      )}

      {/* Bind Ticket Modal */}
      {bindTicketModal && (
        <BindClusterToTicketModal
          open={!!bindTicketModal}
          onOpenChange={(open) => !open && setBindTicketModal(null)}
          clusterId={bindTicketModal.id}
          clusterSize={bindTicketModal.size}
          onSuccess={() => {
            refetchClusters();
          }}
        />
      )}
    </div>
  );
});

ClusteringPage.displayName = 'ClusteringPage';

export default ClusteringPage;
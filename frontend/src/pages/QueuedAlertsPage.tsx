import { useQuery, useMutation, useLazyQuery } from "@apollo/client";
import { useState, useCallback, useEffect, useRef, useMemo } from "react";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import { Input } from "@/components/ui/input";
import {
  Pagination,
  PaginationContent,
  PaginationItem,
  PaginationLink,
  PaginationNext,
  PaginationPrevious,
} from "@/components/ui/pagination";
import {
  GET_QUEUED_ALERTS,
  REPROCESS_QUEUED_ALERT,
  DISCARD_QUEUED_ALERTS,
  GET_REPROCESS_JOB,
} from "@/lib/graphql/queries";
import { Search, Trash2, RefreshCw, Loader2, ChevronDown, ChevronRight } from "lucide-react";
import { useConfirm } from "@/hooks/use-confirm";
import { useToast } from "@/hooks/use-toast";

interface QueuedAlert {
  id: string;
  schema: string;
  title: string;
  data: string;
  createdAt: string;
}

interface QueuedAlertsData {
  queuedAlerts: {
    alerts: QueuedAlert[];
    totalCount: number;
  };
}

export default function QueuedAlertsPage() {
  const confirm = useConfirm();
  const { toast } = useToast();
  const [currentPage, setCurrentPage] = useState(1);
  const [selectedIds, setSelectedIds] = useState<Set<string>>(new Set());
  const [searchKeyword, setSearchKeyword] = useState("");
  const [appliedKeyword, setAppliedKeyword] = useState("");
  const [reprocessingJobId, setReprocessingJobId] = useState<string | null>(null);
  const [expandedIds, setExpandedIds] = useState<Set<string>>(new Set());
  const ITEMS_PER_PAGE = 20;

  const pollIntervalRef = useRef<ReturnType<typeof setInterval> | null>(null);

  const {
    data,
    loading,
    error,
    refetch,
  } = useQuery<QueuedAlertsData>(GET_QUEUED_ALERTS, {
    variables: {
      keyword: appliedKeyword || undefined,
      offset: (currentPage - 1) * ITEMS_PER_PAGE,
      limit: ITEMS_PER_PAGE,
    },
  });

  const [pollJob] = useLazyQuery(GET_REPROCESS_JOB, {
    fetchPolicy: "network-only",
  });

  const [reprocessAlert] = useMutation(REPROCESS_QUEUED_ALERT, {
    onCompleted: (result) => {
      const jobId = result.reprocessQueuedAlert.id;
      setReprocessingJobId(jobId);
      toast({ title: "Reprocessing started", description: "Alert is being reprocessed in the background." });

      // Start polling
      pollIntervalRef.current = setInterval(async () => {
        const { data: jobData } = await pollJob({ variables: { id: jobId } });
        if (jobData?.reprocessJob) {
          const status = jobData.reprocessJob.status;
          if (status === "COMPLETED") {
            clearInterval(pollIntervalRef.current!);
            pollIntervalRef.current = null;
            setReprocessingJobId(null);
            toast({ title: "Reprocessing completed", description: "Alert has been successfully reprocessed." });
            refetch();
          } else if (status === "FAILED") {
            clearInterval(pollIntervalRef.current!);
            pollIntervalRef.current = null;
            setReprocessingJobId(null);
            toast({ title: "Reprocessing failed", description: jobData.reprocessJob.error || "Unknown error", variant: "destructive" });
            refetch();
          }
        }
      }, 2000);
    },
    onError: (err) => {
      toast({ title: "Failed to start reprocessing", description: err.message, variant: "destructive" });
    },
  });

  const [discardAlerts, { loading: discarding }] = useMutation(DISCARD_QUEUED_ALERTS, {
    onCompleted: () => {
      toast({ title: "Alerts discarded", description: `${selectedIds.size} alert(s) have been discarded.` });
      setSelectedIds(new Set());
      refetch();
    },
    onError: (err) => {
      toast({ title: "Failed to discard alerts", description: err.message, variant: "destructive" });
    },
  });

  useEffect(() => {
    return () => {
      if (pollIntervalRef.current) {
        clearInterval(pollIntervalRef.current);
      }
    };
  }, []);

  const alerts = data?.queuedAlerts?.alerts || [];
  const totalCount = data?.queuedAlerts?.totalCount || 0;
  const totalPages = Math.ceil(totalCount / ITEMS_PER_PAGE);

  const handleSearch = useCallback(() => {
    setAppliedKeyword(searchKeyword);
    setCurrentPage(1);
    setSelectedIds(new Set());
  }, [searchKeyword]);

  const handleKeyDown = useCallback(
    (e: React.KeyboardEvent) => {
      if (e.key === "Enter") handleSearch();
    },
    [handleSearch]
  );

  const handlePageChange = (page: number) => {
    setCurrentPage(page);
    setSelectedIds(new Set());
  };

  const toggleSelect = useCallback((id: string) => {
    setSelectedIds((prev) => {
      const next = new Set(prev);
      if (next.has(id)) next.delete(id);
      else next.add(id);
      return next;
    });
  }, []);

  const toggleSelectAll = useCallback(() => {
    if (selectedIds.size === alerts.length) {
      setSelectedIds(new Set());
    } else {
      setSelectedIds(new Set(alerts.map((a) => a.id)));
    }
  }, [selectedIds.size, alerts]);

  const handleDiscard = useCallback(async () => {
    const count = selectedIds.size;
    const confirmed = await confirm({
      title: "Discard Queued Alerts",
      description: `Are you sure you want to discard ${count} queued alert(s)? They will be permanently deleted.`,
      confirmText: "Discard",
      cancelText: "Cancel",
      variant: "destructive",
    });
    if (!confirmed) return;
    discardAlerts({ variables: { ids: Array.from(selectedIds) } });
  }, [selectedIds, confirm, discardAlerts]);

  const handleReprocess = useCallback(
    async (id: string) => {
      const confirmed = await confirm({
        title: "Reprocess Alert",
        description: "This will send the alert through the processing pipeline. Continue?",
        confirmText: "Reprocess",
        cancelText: "Cancel",
      });
      if (!confirmed) return;
      reprocessAlert({ variables: { id } });
    },
    [confirm, reprocessAlert]
  );

  const formatDate = (dateString: string) => {
    const date = new Date(dateString);
    const datePart = date.toISOString().split("T")[0].replace(/-/g, "/");
    const timePart = date.toISOString().split("T")[1].split(".")[0];
    return `${datePart} ${timePart}`;
  };

  const toggleExpand = useCallback((id: string) => {
    setExpandedIds((prev) => {
      const next = new Set(prev);
      if (next.has(id)) next.delete(id);
      else next.add(id);
      return next;
    });
  }, []);

  const getDataPreview = useMemo(() => (data: string): string => {
    try {
      const parsed = JSON.parse(data);
      const flat = JSON.stringify(parsed);
      return flat.length > 120 ? flat.slice(0, 120) + "..." : flat;
    } catch {
      return data.length > 120 ? data.slice(0, 120) + "..." : data;
    }
  }, []);

  if (loading) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="text-lg">Loading queued alerts...</div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="text-lg text-red-600">
          Error loading queued alerts: {error.message}
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-4">
      <div>
        <h1 className="text-3xl font-bold tracking-tight">Queued Alerts</h1>
        <p className="text-muted-foreground">
          Alerts throttled by the circuit breaker. Reprocess or discard them.
        </p>
      </div>

      {/* Search bar */}
      <div className="flex gap-2">
        <div className="relative flex-1 max-w-sm">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
          <Input
            placeholder="Search by keyword..."
            value={searchKeyword}
            onChange={(e) => setSearchKeyword(e.target.value)}
            onKeyDown={handleKeyDown}
            className="pl-9"
          />
        </div>
        <Button variant="outline" onClick={handleSearch}>
          Search
        </Button>
      </div>

      {/* Action bar */}
      {selectedIds.size > 0 && (
        <div className="flex items-center gap-3 px-4 py-2 bg-muted/50 border rounded-lg">
          <span className="text-sm font-medium">{selectedIds.size} selected</span>
          <div className="flex gap-2 ml-auto">
            <Button
              variant="destructive"
              size="sm"
              onClick={handleDiscard}
              disabled={discarding}
            >
              <Trash2 className="h-4 w-4 mr-1" />
              {discarding ? "Discarding..." : "Discard"}
            </Button>
          </div>
        </div>
      )}

      {/* Total count */}
      <div className="text-sm text-muted-foreground">
        {totalCount} queued alert{totalCount !== 1 ? "s" : ""}
      </div>

      {alerts.length === 0 ? (
        <div className="bg-card text-card-foreground rounded-xl border shadow-sm">
          <div className="flex items-center justify-center h-32 px-6">
            <p className="text-muted-foreground">No queued alerts</p>
          </div>
        </div>
      ) : (
        <div className="space-y-2">
          {/* Select all */}
          <div className="flex items-center gap-3 px-4 py-2">
            <Checkbox
              checked={alerts.length > 0 && selectedIds.size === alerts.length}
              onCheckedChange={toggleSelectAll}
            />
            <span className="text-sm text-muted-foreground">Select all on this page</span>
          </div>

          {alerts.map((qa) => {
            const isExpanded = expandedIds.has(qa.id);
            return (
              <div
                key={qa.id}
                className="bg-card text-card-foreground rounded-xl border shadow-sm overflow-hidden"
              >
                <div className="px-4 py-3">
                  <div className="flex items-start gap-3">
                    <div className="pt-0.5" onClick={(e) => e.stopPropagation()}>
                      <Checkbox
                        checked={selectedIds.has(qa.id)}
                        onCheckedChange={() => toggleSelect(qa.id)}
                      />
                    </div>
                    <div
                      className="flex-1 min-w-0 cursor-pointer"
                      onClick={() => toggleExpand(qa.id)}
                    >
                      <div className="flex items-center gap-2 mb-1">
                        {isExpanded ? (
                          <ChevronDown className="h-4 w-4 text-muted-foreground flex-shrink-0" />
                        ) : (
                          <ChevronRight className="h-4 w-4 text-muted-foreground flex-shrink-0" />
                        )}
                        <Badge variant="outline" className="text-xs">
                          {qa.schema}
                        </Badge>
                        <span className="text-xs text-muted-foreground ml-auto">
                          {formatDate(qa.createdAt)}
                        </span>
                      </div>
                      {!isExpanded && (
                        <p className="text-sm text-muted-foreground font-mono truncate pl-8">
                          {getDataPreview(qa.data)}
                        </p>
                      )}
                    </div>
                    <div className="flex-shrink-0">
                      <Button
                        variant="outline"
                        size="sm"
                        onClick={(e) => {
                          e.stopPropagation();
                          handleReprocess(qa.id);
                        }}
                        disabled={reprocessingJobId !== null}
                      >
                        {reprocessingJobId !== null ? (
                          <Loader2 className="h-4 w-4 mr-1 animate-spin" />
                        ) : (
                          <RefreshCw className="h-4 w-4 mr-1" />
                        )}
                        Reprocess
                      </Button>
                    </div>
                  </div>
                  {isExpanded && (
                    <div className="mt-2 ml-8">
                      <pre className="text-xs bg-muted p-3 rounded-lg overflow-auto max-h-96 whitespace-pre-wrap break-all">
                        {(() => {
                          try {
                            return JSON.stringify(JSON.parse(qa.data), null, 2);
                          } catch {
                            return qa.data;
                          }
                        })()}
                      </pre>
                    </div>
                  )}
                </div>
              </div>
            );
          })}
        </div>
      )}

      {/* Pagination */}
      {totalPages > 1 && (
        <div className="flex justify-center mt-6">
          <Pagination>
            <PaginationContent>
              <PaginationItem>
                <PaginationPrevious
                  onClick={() => handlePageChange(Math.max(1, currentPage - 1))}
                  className={currentPage === 1 ? "pointer-events-none opacity-50" : "cursor-pointer"}
                />
              </PaginationItem>
              {Array.from({ length: Math.min(totalPages, 10) }, (_, i) => i + 1).map((page) => (
                <PaginationItem key={page}>
                  <PaginationLink
                    isActive={page === currentPage}
                    onClick={() => handlePageChange(page)}
                    className="cursor-pointer"
                  >
                    {page}
                  </PaginationLink>
                </PaginationItem>
              ))}
              <PaginationItem>
                <PaginationNext
                  onClick={() => handlePageChange(Math.min(totalPages, currentPage + 1))}
                  className={currentPage === totalPages ? "pointer-events-none opacity-50" : "cursor-pointer"}
                />
              </PaginationItem>
            </PaginationContent>
          </Pagination>
        </div>
      )}
    </div>
  );
}

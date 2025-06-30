import { useState, useEffect, useCallback } from "react";
import { useQuery, useMutation } from "@apollo/client";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Slider } from "@/components/ui/slider";
import { Checkbox } from "@/components/ui/checkbox";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Badge } from "@/components/ui/badge";
import { GET_UNBOUND_ALERTS, BIND_ALERTS_TO_TICKET, GET_TICKET } from "@/lib/graphql/queries";
import { Alert } from "@/lib/types";
import { AlertCircle, Search, X } from "lucide-react";
import { useSuccessToast, useErrorToast } from "@/hooks/use-toast";

interface SalvageModalProps {
  isOpen: boolean;
  onClose: () => void;
  ticketId: string;
}

interface UnboundAlertsData {
  unboundAlerts: {
    alerts: Alert[];
    totalCount: number;
  };
}

export function SalvageModal({ isOpen, onClose, ticketId }: SalvageModalProps) {
  const [threshold, setThreshold] = useState([0.95]);
  const [keyword, setKeyword] = useState("");
  const [selectedAlerts, setSelectedAlerts] = useState<Set<string>>(new Set());
  const [debouncedKeyword, setDebouncedKeyword] = useState("");
  
  const successToast = useSuccessToast();
  const errorToast = useErrorToast();

  // Debounce keyword input
  useEffect(() => {
    const timer = setTimeout(() => {
      setDebouncedKeyword(keyword);
    }, 500);

    return () => clearTimeout(timer);
  }, [keyword]);

  const { data: alertsData, loading: alertsLoading, refetch } = useQuery<UnboundAlertsData>(
    GET_UNBOUND_ALERTS,
    {
      variables: {
        threshold: threshold[0],
        keyword: debouncedKeyword || undefined,
        ticketId,
        offset: 0,
        limit: 50,
      },
      skip: !isOpen,
    }
  );

  const [bindAlerts, { loading: bindingLoading }] = useMutation(BIND_ALERTS_TO_TICKET, {
    refetchQueries: [{ query: GET_TICKET, variables: { id: ticketId } }],
  });

  // Auto-refresh when threshold changes
  useEffect(() => {
    if (isOpen) {
      refetch({
        threshold: threshold[0],
        keyword: debouncedKeyword || undefined,
        ticketId,
        offset: 0,
        limit: 50,
      });
    }
  }, [threshold, debouncedKeyword, isOpen, refetch, ticketId]);

  const handleThresholdChange = useCallback((value: number[]) => {
    setThreshold(value);
  }, []);

  const handleSelectAlert = useCallback((alertId: string, checked: boolean) => {
    setSelectedAlerts(prev => {
      const newSet = new Set(prev);
      if (checked) {
        newSet.add(alertId);
      } else {
        newSet.delete(alertId);
      }
      return newSet;
    });
  }, []);

  const handleSelectAll = useCallback((checked: boolean) => {
    if (checked) {
      const allAlertIds = alertsData?.unboundAlerts.alerts.map(alert => alert.id) || [];
      setSelectedAlerts(new Set(allAlertIds));
    } else {
      setSelectedAlerts(new Set());
    }
  }, [alertsData?.unboundAlerts.alerts]);

  const handleSubmit = async () => {
    if (selectedAlerts.size === 0) {
      errorToast("Please select at least one alert to bind.");
      return;
    }

    try {
      await bindAlerts({
        variables: {
          ticketId,
          alertIds: Array.from(selectedAlerts),
        },
      });
      
      successToast(`Successfully bound ${selectedAlerts.size} alert(s) to the ticket.`);
      setSelectedAlerts(new Set());
      onClose();
    } catch (error) {
      errorToast(`Failed to bind alerts: ${error}`);
    }
  };

  const handleClose = () => {
    setSelectedAlerts(new Set());
    setKeyword("");
    setThreshold([0.95]);
    onClose();
  };

  const alerts = alertsData?.unboundAlerts.alerts || [];
  const totalCount = alertsData?.unboundAlerts.totalCount || 0;
  const isAllSelected = selectedAlerts.size > 0 && selectedAlerts.size === alerts.length;
  const isIndeterminate = selectedAlerts.size > 0 && selectedAlerts.size < alerts.length;

  return (
    <Dialog open={isOpen} onOpenChange={handleClose}>
      <DialogContent className="max-w-4xl max-h-[90vh] flex flex-col">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <AlertCircle className="h-5 w-5" />
            Salvage Alerts
          </DialogTitle>
        </DialogHeader>

        <div className="flex-1 flex flex-col gap-4 overflow-hidden">
          {/* Controls */}
          <div className="space-y-4 flex-shrink-0">
            {/* Similarity Threshold */}
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
                onValueChange={handleThresholdChange}
                className="w-full"
              />
              <div className="flex justify-between text-xs text-muted-foreground">
                <span>0.0 (Less similar)</span>
                <span>1.0 (More similar)</span>
              </div>
            </div>

            {/* Keyword Filter */}
            <div className="space-y-2">
              <Label htmlFor="keyword" className="text-sm font-medium">
                Keyword Filter
              </Label>
              <div className="relative">
                <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 h-4 w-4 text-muted-foreground" />
                <Input
                  id="keyword"
                  placeholder="Filter alerts by keyword in data..."
                  value={keyword}
                  onChange={(e) => setKeyword(e.target.value)}
                  className="pl-10"
                />
                {keyword && (
                  <Button
                    variant="ghost"
                    size="sm"
                    className="absolute right-2 top-1/2 transform -translate-y-1/2 h-6 w-6 p-0"
                    onClick={() => setKeyword("")}
                  >
                    <X className="h-4 w-4" />
                  </Button>
                )}
              </div>
            </div>
          </div>

          {/* Results */}
          <div className="flex-1 flex flex-col overflow-hidden">
            <div className="flex items-center justify-between mb-4">
              <div className="flex items-center gap-4">
                <h3 className="text-lg font-semibold">
                  Matching Alerts ({totalCount})
                </h3>
                {alerts.length > 0 && (
                  <div className="flex items-center gap-2">
                    <Checkbox
                      id="select-all"
                      checked={isAllSelected}
                      ref={(el: HTMLButtonElement | null) => {
                        if (el) (el as any).indeterminate = isIndeterminate;
                      }}
                      onCheckedChange={handleSelectAll}
                    />
                    <Label htmlFor="select-all" className="text-sm">
                      Select All ({alerts.length})
                    </Label>
                  </div>
                )}
              </div>
              {selectedAlerts.size > 0 && (
                <Badge variant="secondary">
                  {selectedAlerts.size} selected
                </Badge>
              )}
            </div>

            {alertsLoading ? (
              <div className="flex items-center justify-center py-8">
                <div className="text-sm text-muted-foreground">Loading alerts...</div>
              </div>
            ) : alerts.length === 0 ? (
              <div className="flex items-center justify-center py-8">
                <div className="text-center">
                  <AlertCircle className="h-8 w-8 mx-auto mb-2 text-muted-foreground" />
                  <div className="text-sm text-muted-foreground">
                    No alerts found with the current filters
                  </div>
                </div>
              </div>
            ) : (
              <ScrollArea className="flex-1">
                <div className="space-y-3 pr-4">
                  {alerts.map((alert) => (
                    <div
                      key={alert.id}
                      className="border rounded-lg p-4 hover:bg-muted/50 transition-colors"
                    >
                      <div className="flex items-start gap-3">
                        <Checkbox
                          id={`alert-${alert.id}`}
                          checked={selectedAlerts.has(alert.id)}
                          onCheckedChange={(checked: boolean) =>
                            handleSelectAlert(alert.id, checked)
                          }
                          className="mt-1"
                        />
                        <div className="flex-1 min-w-0">
                          <div className="flex items-center gap-2 mb-2">
                            <Badge variant="outline" className="text-xs">
                              {alert.schema}
                            </Badge>
                            <span className="text-xs text-muted-foreground">
                              {new Date(alert.createdAt).toLocaleString()}
                            </span>
                          </div>
                          <h4 className="font-medium text-sm mb-1 line-clamp-2">
                            {alert.title}
                          </h4>
                          {alert.description && (
                            <p className="text-xs text-muted-foreground line-clamp-2 mb-2">
                              {alert.description}
                            </p>
                          )}
                          <div className="flex flex-wrap gap-1">
                            {alert.attributes.slice(0, 3).map((attr, index) => (
                              <Badge key={index} variant="secondary" className="text-xs">
                                {attr.key}: {attr.value}
                              </Badge>
                            ))}
                            {alert.attributes.length > 3 && (
                              <Badge variant="outline" className="text-xs">
                                +{alert.attributes.length - 3} more
                              </Badge>
                            )}
                          </div>
                        </div>
                      </div>
                    </div>
                  ))}
                </div>
              </ScrollArea>
            )}
          </div>

          {/* Actions */}
          <div className="flex justify-between pt-4 border-t">
            <Button variant="outline" onClick={handleClose}>
              Cancel
            </Button>
            <Button
              onClick={handleSubmit}
              disabled={selectedAlerts.size === 0 || bindingLoading}
              className="min-w-[120px]"
            >
              {bindingLoading ? "Binding..." : `Bind ${selectedAlerts.size} Alert(s)`}
            </Button>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  );
}
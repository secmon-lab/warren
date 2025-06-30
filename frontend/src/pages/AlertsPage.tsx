import { useQuery } from "@apollo/client";
import { useState, useMemo } from "react";
import { useNavigate } from "react-router-dom";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Pagination,
  PaginationContent,
  PaginationItem,
  PaginationLink,
  PaginationNext,
  PaginationPrevious,
} from "@/components/ui/pagination";
import { GET_ALERTS } from "@/lib/graphql/queries";
import { Alert } from "@/lib/types";
import { AlertTriangle } from "lucide-react";

interface AlertsData {
  alerts: {
    alerts: Alert[];
    totalCount: number;
  };
}

export default function AlertsPage() {
  const navigate = useNavigate();
  const [currentPage, setCurrentPage] = useState(1);
  const ITEMS_PER_PAGE = 10;

  const {
    data: alertsData,
    loading: alertsLoading,
    error: alertsError,
  } = useQuery<AlertsData>(GET_ALERTS, {
    variables: {
      offset: (currentPage - 1) * ITEMS_PER_PAGE,
      limit: ITEMS_PER_PAGE,
    },
  });

  // Sort alerts by createdAt in descending order (newest first)
  const sortedAlerts: Alert[] = useMemo(() => {
    if (!alertsData?.alerts?.alerts) return [];
    
    return [...alertsData.alerts.alerts].sort((a, b) => 
      new Date(b.createdAt).getTime() - new Date(a.createdAt).getTime()
    );
  }, [alertsData?.alerts?.alerts]);

  // Calculate pagination values
  const totalCount = alertsData?.alerts?.totalCount || 0;
  const totalPages = Math.ceil(totalCount / ITEMS_PER_PAGE);

  const handleAlertClick = (alertId: string) => {
    navigate(`/alerts/${alertId}`);
  };

  const formatDate = (dateString: string) => {
    return new Date(dateString).toLocaleDateString("ja-JP", {
      year: "numeric",
      month: "short",
      day: "numeric",
      hour: "2-digit",
      minute: "2-digit",
    });
  };

  if (alertsLoading) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="text-lg">Loading alerts...</div>
      </div>
    );
  }

  if (alertsError) {
    // Check if this is an authentication/authorization error
    if (alertsError.message?.includes('Authentication required') ||
        alertsError.message?.includes('Invalid authentication token') ||
        alertsError.message?.includes('JSON.parse') ||
        alertsError.message?.includes('unexpected character')) {
      return (
        <div className="flex items-center justify-center h-64">
          <div className="text-center">
            <div className="text-lg text-red-600 mb-4">
              Authentication required
            </div>
            <div className="text-sm text-muted-foreground mb-4">
              Please log in to access alerts
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
          Error loading alerts: {alertsError.message}
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold tracking-tight">Unbound Alerts</h1>
          <p className="text-muted-foreground">
            Monitor and manage unassigned security alerts
          </p>
        </div>
      </div>

      {sortedAlerts.length === 0 ? (
        <div className="bg-card text-card-foreground rounded-xl border shadow-sm">
          <div className="flex items-center justify-center h-32 px-6">
            <p className="text-muted-foreground">No alerts found</p>
          </div>
        </div>
      ) : (
        <div className="space-y-2">
          {sortedAlerts.map((alert: Alert) => (
            <div
              key={alert.id}
              className="bg-card text-card-foreground rounded-xl border shadow-sm cursor-pointer hover:shadow-md transition-shadow"
              onClick={() => handleAlertClick(alert.id)}
            >
              <div className="px-4 py-6">
                <div className="flex flex-col">
                  <div className="flex items-center gap-2 mb-1">
                    <AlertTriangle className="h-4 w-4 text-orange-500" />
                    <Badge variant="outline" className="text-xs">
                      {alert.schema}
                    </Badge>
                    {alert.ticket && (
                      <Badge variant="secondary" className="text-xs">
                        Assigned
                      </Badge>
                    )}
                    <span className="text-xs text-muted-foreground ml-auto">
                      {formatDate(alert.createdAt)}
                    </span>
                  </div>
                  
                  <div className="flex items-start justify-between">
                    <div className="flex-1 min-w-0">
                      <h3 className="text-base font-semibold mb-1 line-clamp-1">
                        {alert.title}
                      </h3>
                      {alert.description && (
                        <p className="text-sm text-muted-foreground line-clamp-2 mb-1">
                          {alert.description}
                        </p>
                      )}
                      
                      {/* Alert Attributes */}
                      <div className="flex flex-wrap gap-1">
                        {alert.attributes && alert.attributes.length > 0 && (
                          <>
                            {alert.attributes.slice(0, 3).map((attr, index: number) => (
                              <Badge 
                                key={index} 
                                variant={attr.auto ? "secondary" : "default"}
                                className="text-xs"
                              >
                                {attr.key}: {attr.value}
                              </Badge>
                            ))}
                            {alert.attributes.length > 3 && (
                              <Badge variant="outline" className="text-xs">
                                +{alert.attributes.length - 3} more
                              </Badge>
                            )}
                          </>
                        )}
                        {alert.ticket && (
                          <Badge variant="outline" className="text-xs ml-auto">
                            Ticket #{alert.ticket.id.slice(-8)}
                          </Badge>
                        )}
                      </div>
                    </div>
                  </div>
                </div>
              </div>
            </div>
          ))}
        </div>
      )}

      {/* Pagination */}
      {totalPages > 1 && (
        <div className="flex justify-center">
          <Pagination>
            <PaginationContent>
              <PaginationItem>
                <PaginationPrevious
                  onClick={() => setCurrentPage(Math.max(1, currentPage - 1))}
                  className={currentPage === 1 ? "pointer-events-none opacity-50" : "cursor-pointer"}
                />
              </PaginationItem>
              {Array.from({ length: totalPages }, (_, i) => i + 1).map((page) => (
                <PaginationItem key={page}>
                  <PaginationLink
                    onClick={() => setCurrentPage(page)}
                    isActive={currentPage === page}
                    className="cursor-pointer"
                  >
                    {page}
                  </PaginationLink>
                </PaginationItem>
              ))}
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
    </div>
  );
} 
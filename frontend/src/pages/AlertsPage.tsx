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
import { GET_ALERTS, GET_TAGS } from "@/lib/graphql/queries";
import { Alert, TagMetadata } from "@/lib/types";
import { AlertTriangle, Tag } from "lucide-react";
import { generateTagColor } from "@/lib/tag-colors";

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

  const { data: tagsData } = useQuery(GET_TAGS);
  const tagsByName = new Map((tagsData?.tags || []).map((t: TagMetadata) => [t.name, t]));

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
          <h1 className="text-3xl font-bold tracking-tight">New Alerts</h1>
          <p className="text-muted-foreground">
            Monitor and manage new security alerts
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
              className="bg-card text-card-foreground rounded-xl border shadow-sm cursor-pointer hover:shadow-md transition-shadow overflow-hidden"
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
                    <div className="flex-1 min-w-0 pr-2">
                      <h3 className="text-base font-semibold mb-1 line-clamp-1">
                        {alert.title}
                      </h3>
                      {alert.description && (
                        <p className="text-sm text-muted-foreground line-clamp-2 mb-1">
                          {alert.description}
                        </p>
                      )}
                      
                      {/* Alert Tags and Attributes */}
                      <div className="flex flex-wrap gap-1 max-w-full mb-1">
                        {alert.tags && alert.tags.length > 0 && (
                          <>
                            {alert.tags.map((tag, index) => {
                              const tagData = tagsByName.get(tag) as TagMetadata | undefined;
                              const colorClass = tagData?.color || generateTagColor(tag);
                              return (
                                <Badge 
                                  key={`tag-${index}`}
                                  variant="secondary"
                                  className={`text-xs ${colorClass}`}
                                >
                                  <Tag className="h-3 w-3 mr-1" />
                                  {tag}
                                </Badge>
                              );
                            })}
                          </>
                        )}
                      </div>
                      
                      {/* Alert Attributes */}
                      <div className="flex flex-wrap gap-1 max-w-full">
                        {alert.attributes && alert.attributes.length > 0 && (
                          <>
                            {alert.attributes.slice(0, 3).map((attr, index: number) => {
                              const displayText = `${attr.key}: ${attr.value}`;
                              const isLong = displayText.length > 40;
                              const truncatedText = isLong ? `${displayText.substring(0, 37)}...` : displayText;
                              
                              return (
                                <Badge 
                                  key={index} 
                                  variant={attr.auto ? "outline" : "default"}
                                  className="text-xs max-w-[280px] truncate inline-block"
                                  title={displayText}
                                >
                                  {truncatedText}
                                </Badge>
                              );
                            })}
                            {alert.attributes.length > 3 && (
                              <Badge variant="outline" className="text-xs">
                                +{alert.attributes.length - 3} more
                              </Badge>
                            )}
                          </>
                        )}
                        {alert.ticket && (
                          <Badge variant="outline" className="text-xs">
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
        <div className="flex justify-center mt-6">
          <Pagination>
            <PaginationContent>
              <PaginationItem>
                <PaginationPrevious
                  onClick={() => setCurrentPage(Math.max(1, currentPage - 1))}
                  className={currentPage === 1 ? "pointer-events-none opacity-50" : "cursor-pointer"}
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
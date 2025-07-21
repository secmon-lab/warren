import { useState, memo, useCallback } from "react";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from "@/components/ui/collapsible";
import { useClusterAlertsQuery } from "@/lib/graphql/generated";
import { 
  ChevronDown, 
  ChevronRight, 
  AlertTriangle, 
  Clock, 
  Plus, 
  Link2,
  Tag,
  Users
} from "lucide-react";

interface ClusterCardProps {
  cluster: {
    id: string;
    size: number;
    keywords?: string[] | null;
    createdAt: string;
    centerAlert?: {
      id: string;
      title: string;
      description?: string | null;
      schema: string;
      createdAt: string;
    } | null;
  };
  onCreateTicket: (clusterId: string) => void;
  onBindToTicket: (clusterId: string) => void;
  onViewDetails: (clusterId: string) => void;
}

const ClusterCard = memo(({ 
  cluster, 
  onCreateTicket, 
  onBindToTicket, 
  onViewDetails 
}: ClusterCardProps) => {
  const [isExpanded, setIsExpanded] = useState(false);
  
  const {
    data: clusterAlertsData,
    loading: alertsLoading,
    error: alertsError,
  } = useClusterAlertsQuery({
    variables: {
      clusterID: cluster.id,
      limit: 10, // Show first 10 alerts when expanded
      offset: 0,
    },
    skip: !isExpanded, // Only fetch when expanded
  });

  const handleCreateTicket = useCallback(() => {
    onCreateTicket(cluster.id);
  }, [cluster.id, onCreateTicket]);

  const handleBindToTicket = useCallback(() => {
    onBindToTicket(cluster.id);
  }, [cluster.id, onBindToTicket]);

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
    // Simple heuristic based on schema name
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

  return (
    <Card className="transition-all duration-200 hover:shadow-md">
      <Collapsible open={isExpanded} onOpenChange={setIsExpanded}>
        <CardHeader className="pb-3">
          <div className="flex items-start justify-between">
            <div className="flex-1">
              <CollapsibleTrigger asChild>
                <div className="flex items-center gap-2 cursor-pointer group">
                  {isExpanded ? (
                    <ChevronDown className="h-4 w-4 text-muted-foreground group-hover:text-foreground transition-colors" />
                  ) : (
                    <ChevronRight className="h-4 w-4 text-muted-foreground group-hover:text-foreground transition-colors" />
                  )}
                  <CardTitle className="text-lg group-hover:text-primary transition-colors">
                    Cluster <span className="text-blue-600 font-mono font-semibold">{cluster.id}</span>
                  </CardTitle>
                </div>
              </CollapsibleTrigger>
              
              {/* Cluster Info */}
              <div className="flex items-center gap-4 mt-2 text-sm text-muted-foreground">
                <div className="flex items-center gap-1">
                  <Users className="h-4 w-4" />
                  <span>{cluster.size} alerts</span>
                </div>
                <div className="flex items-center gap-1">
                  <Clock className="h-4 w-4" />
                  <span>{formatTimeAgo(cluster.createdAt)}</span>
                </div>
              </div>
            </div>

            {/* Action Buttons */}
            <div className="flex items-center gap-2 ml-4">
              <Button
                onClick={handleCreateTicket}
                size="sm"
                variant="outline"
                className="text-xs"
              >
                <Plus className="h-3 w-3 mr-1" />
                Create Ticket
              </Button>
              <Button
                onClick={handleBindToTicket}
                size="sm"
                variant="outline"
                className="text-xs"
              >
                <Link2 className="h-3 w-3 mr-1" />
                Bind to Ticket
              </Button>
            </div>
          </div>

          {/* Center Alert Summary */}
          {cluster.centerAlert && (
            <div className="mt-3 p-3 bg-muted/50 rounded-lg">
              <div className="flex items-start justify-between gap-3">
                <div className="flex-1 min-w-0">
                  <div className="flex items-center gap-2 mb-1">
                    <AlertTriangle className="h-4 w-4 text-amber-500 flex-shrink-0" />
                    <p className="font-medium text-sm truncate">
                      {cluster.centerAlert.title}
                    </p>
                  </div>
                  {cluster.centerAlert.description && (
                    <p className="text-xs text-muted-foreground line-clamp-2">
                      {cluster.centerAlert.description}
                    </p>
                  )}
                </div>
                <Badge 
                  variant="secondary" 
                  className={`text-xs ${getSeverityColor(cluster.centerAlert.schema)}`}
                >
                  {cluster.centerAlert.schema}
                </Badge>
              </div>
            </div>
          )}

          {/* Keywords */}
          {cluster.keywords && cluster.keywords.length > 0 && (
            <div className="flex items-center gap-2 mt-3">
              <Tag className="h-3 w-3 text-muted-foreground" />
              <div className="flex flex-wrap gap-1">
                {cluster.keywords.slice(0, 5).map((keyword, index) => (
                  <Badge key={index} variant="outline" className="text-xs">
                    {keyword}
                  </Badge>
                ))}
                {cluster.keywords.length > 5 && (
                  <Badge variant="outline" className="text-xs">
                    +{cluster.keywords.length - 5} more
                  </Badge>
                )}
              </div>
            </div>
          )}
        </CardHeader>

        <CollapsibleContent>
          <CardContent className="pt-0">
            <div className="border-t pt-4">
              <div className="flex items-center justify-between mb-3">
                <h4 className="font-medium text-sm">Cluster Alerts</h4>
                <Button
                  onClick={() => onViewDetails(cluster.id)}
                  variant="link"
                  size="sm"
                  className="text-xs h-auto p-0"
                >
                  View all {cluster.size} alerts →
                </Button>
              </div>

              {alertsLoading ? (
                <div className="space-y-2">
                  {[...Array(3)].map((_, i) => (
                    <div key={i} className="flex items-center gap-3 p-2 border rounded animate-pulse">
                      <div className="h-4 w-4 bg-gray-200 rounded" />
                      <div className="flex-1 space-y-1">
                        <div className="h-3 bg-gray-200 rounded w-3/4" />
                        <div className="h-2 bg-gray-200 rounded w-1/2" />
                      </div>
                    </div>
                  ))}
                </div>
              ) : alertsError ? (
                <p className="text-sm text-red-600">Failed to load alerts</p>
              ) : (
                <div className="space-y-2">
                  {clusterAlertsData?.clusterAlerts?.alerts?.slice(0, 5).map((alert) => (
                    <div key={alert.id} className="flex items-center gap-3 p-2 border rounded hover:bg-muted/50 transition-colors">
                      <AlertTriangle className="h-4 w-4 text-amber-500 flex-shrink-0" />
                      <div className="flex-1 min-w-0">
                        <p className="text-sm font-medium truncate">{alert.title}</p>
                        <p className="text-xs text-muted-foreground">
                          {formatTimeAgo(alert.createdAt)}
                        </p>
                      </div>
                      {alert.ticket && (
                        <Badge variant="secondary" className="text-xs">
                          Ticket #{alert.ticket.id}
                        </Badge>
                      )}
                    </div>
                  ))}
                  
                  {clusterAlertsData?.clusterAlerts?.totalCount && 
                   clusterAlertsData.clusterAlerts.totalCount > 5 && (
                    <div className="text-center pt-2">
                      <Button
                        onClick={() => onViewDetails(cluster.id)}
                        variant="outline"
                        size="sm"
                        className="text-xs"
                      >
                        View {clusterAlertsData.clusterAlerts.totalCount - 5} more alerts
                      </Button>
                    </div>
                  )}
                </div>
              )}
            </div>
          </CardContent>
        </CollapsibleContent>
      </Collapsible>
    </Card>
  );
});

ClusterCard.displayName = 'ClusterCard';

export default ClusterCard;
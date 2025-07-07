import { useState } from "react";
import { useNavigate } from "react-router-dom";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { CreateTicketModal } from "@/components/CreateTicketModal";
import { 
  useGetDashboardQuery, 
  useGetActivitiesQuery,
  type GetActivitiesQuery
} from "@/lib/graphql/generated";
import { 
  AlertTriangle, 
  Ticket, 
  Plus, 
  Clock, 
  User, 
  ArrowRight,
  ChevronLeft,
  ChevronRight,
  Activity,
  AlertCircle,
  MessageSquare,
  Link,
  Layers
} from "lucide-react";
import { formatDistanceToNow } from "date-fns";
import { useUserName } from "@/components/ui/user-name";
import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar";
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@/components/ui/tooltip";

// UserAvatar component
function UserAvatar({ userID }: { userID: string }) {
  const { name, isLoading } = useUserName(userID);
  
  const displayName = name || userID || "Unknown User";
  
  if (isLoading) {
    return (
      <div className="w-10 h-10 bg-muted rounded-full animate-pulse" />
    );
  }
  
  return (
    <TooltipProvider>
      <Tooltip>
        <TooltipTrigger asChild>
          <Avatar className="w-10 h-10 cursor-help">
            <AvatarImage src={`/api/user/${userID}/icon`} alt={displayName} />
            <AvatarFallback className="text-sm font-medium">
              {displayName.charAt(0).toUpperCase()}
            </AvatarFallback>
          </Avatar>
        </TooltipTrigger>
        <TooltipContent>
          <p>{displayName}</p>
          <p className="text-xs text-muted-foreground">ID: {userID}</p>
        </TooltipContent>
      </Tooltip>
    </TooltipProvider>
  );
}

export default function Dashboard() {
  const [isCreateTicketOpen, setIsCreateTicketOpen] = useState(false);
  const [activitiesPage, setActivitiesPage] = useState(0);
  const navigate = useNavigate();
  
  const { data: dashboardData, loading: dashboardLoading } = useGetDashboardQuery({
    fetchPolicy: "cache-and-network",
  });

  const { data: activitiesData, loading: activitiesLoading } = useGetActivitiesQuery({
    variables: { offset: activitiesPage * 10, limit: 10 },
    fetchPolicy: "cache-and-network",
  });

  const handleTicketClick = (ticketId: string) => {
    navigate(`/tickets/${ticketId}`);
  };

  const handleAlertClick = (alertId: string) => {
    navigate(`/alerts/${alertId}`);
  };

  const handleActivityClick = (activity: GetActivitiesQuery['activities']['activities'][0]) => {
    // Priority: Ticket > Alert
    if (activity.ticketID) {
      navigate(`/tickets/${activity.ticketID}`);
    } else if (activity.alertID) {
      navigate(`/alerts/${activity.alertID}`);
    }
  };

  const getActivityIcon = (type: string) => {
    switch (type) {
      case "ticket_created":
        return <Ticket className="h-4 w-4 text-blue-600" />;
      case "ticket_updated":
        return <Ticket className="h-4 w-4 text-orange-600" />;
      case "ticket_status_changed":
        return <Clock className="h-4 w-4 text-amber-600" />;
      case "comment_added":
        return <MessageSquare className="h-4 w-4 text-purple-600" />;
      case "alert_bound":
        return <Link className="h-4 w-4 text-green-600" />;
      case "alerts_bulk_bound":
        return <Layers className="h-4 w-4 text-green-600" />;
      default:
        return <Activity className="h-4 w-4 text-gray-600" />;
    }
  };

  const getActivityTitle = (type: string) => {
    switch (type) {
      case "ticket_created":
        return "Ticket Created";
      case "ticket_updated":
        return "Ticket Updated";
      case "ticket_status_changed":
        return "Status Changed";
      case "comment_added":
        return "Comment Added";
      case "alert_bound":
        return "Alert Bound";
      case "alerts_bulk_bound":
        return "Alerts Bulk Bound";
      default:
        return "Activity";
    }
  };

  return (
    <div className="space-y-8">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold tracking-tight">Dashboard</h1>
          <p className="text-muted-foreground">
            Security monitoring and ticket management system
          </p>
        </div>
        <Button onClick={() => setIsCreateTicketOpen(true)} className="flex items-center gap-2">
          <Plus className="h-4 w-4" />
          Create Ticket
        </Button>
      </div>

      {/* Main Content - Left 2/3 for Tickets/Alerts, Right 1/3 for Activity */}
      <div className="grid gap-8 lg:grid-cols-3 lg:grid-rows-1">
        {/* Left Section - Tickets and Alerts in 2 rows */}
        <div className="lg:col-span-2 space-y-8">
          {/* Open Tickets Section */}
          <Card className="border-blue-200 bg-gradient-to-br from-blue-50 to-white">
            <CardHeader className="pb-4">
              <div className="flex items-center justify-between">
                <div className="flex items-center gap-3">
                  <div className="p-2 bg-blue-100 rounded-lg">
                    <Ticket className="h-6 w-6 text-blue-600" />
                  </div>
                  <div>
                    <CardTitle className="text-lg font-semibold text-blue-900">Open Tickets</CardTitle>
                    <p className="text-sm text-blue-600">
                      {dashboardLoading ? "..." : dashboardData?.dashboard.openTicketsCount || 0} unresolved tickets
                    </p>
                  </div>
                </div>
                <Button 
                  variant="ghost" 
                  size="sm" 
                  onClick={() => navigate('/tickets')}
                  className="text-blue-600 hover:text-blue-700 hover:bg-blue-100"
                >
                  View All
                  <ArrowRight className="h-4 w-4 ml-1" />
                </Button>
              </div>
            </CardHeader>
            <CardContent className="space-y-3">
              {dashboardLoading ? (
                <div className="text-sm text-muted-foreground">Loading...</div>
              ) : dashboardData?.dashboard?.openTickets && dashboardData.dashboard.openTickets.length > 0 ? (
                <>
                  {dashboardData.dashboard.openTickets.slice(0, 5).map((ticket) => (
                    <div
                      key={ticket.id}
                      className="p-3 bg-white border border-blue-100 rounded-lg cursor-pointer hover:border-blue-200 hover:shadow-sm transition-all"
                      onClick={() => handleTicketClick(ticket.id)}
                    >
                      <div className="space-y-2">
                        <div className="flex items-start justify-between">
                          <p className="font-medium text-sm truncate text-blue-900 pr-2">{ticket.title}</p>
                          <Badge variant="outline" className="text-xs border-blue-200 text-blue-700 flex-shrink-0">
                            {ticket.status}
                          </Badge>
                        </div>
                        {/* Add ticket description */}
                        {ticket.description && (
                          <p className="text-xs text-gray-500 line-clamp-2">
                            {ticket.description}
                          </p>
                        )}
                        <div className="flex items-center justify-between text-xs">
                          {ticket.assignee && (
                            <div className="flex items-center gap-1 text-blue-600">
                              <User className="h-3 w-3" />
                              <span className="truncate">{ticket.assignee.name}</span>
                            </div>
                          )}
                          <span className="text-blue-500 flex-shrink-0">
                            {formatDistanceToNow(new Date(ticket.createdAt), { addSuffix: true })}
                          </span>
                        </div>
                      </div>
                    </div>
                  ))}
                </>
              ) : (
                <div className="text-center py-8">
                  <Ticket className="h-12 w-12 text-blue-300 mx-auto mb-3" />
                  <p className="text-sm text-blue-600">No open tickets</p>
                </div>
              )}
            </CardContent>
          </Card>

          {/* New Alerts Section */}
          <Card className="border-amber-200 bg-gradient-to-br from-amber-50 to-white">
            <CardHeader className="pb-4">
              <div className="flex items-center justify-between">
                <div className="flex items-center gap-3">
                  <div className="p-2 bg-amber-100 rounded-lg">
                    <AlertTriangle className="h-6 w-6 text-amber-600" />
                  </div>
                  <div>
                    <CardTitle className="text-lg font-semibold text-amber-900">New Alerts</CardTitle>
                    <p className="text-sm text-amber-600">
                      {dashboardLoading ? "..." : dashboardData?.dashboard.unboundAlertsCount || 0} alerts need attention
                    </p>
                  </div>
                </div>
                <Button 
                  variant="ghost" 
                  size="sm" 
                  onClick={() => navigate('/alerts')}
                  className="text-amber-600 hover:text-amber-700 hover:bg-amber-100"
                >
                  View All
                  <ArrowRight className="h-4 w-4 ml-1" />
                </Button>
              </div>
            </CardHeader>
            <CardContent className="space-y-3">
              {dashboardLoading ? (
                <div className="text-sm text-muted-foreground">Loading...</div>
              ) : dashboardData?.dashboard?.unboundAlerts && dashboardData.dashboard.unboundAlerts.length > 0 ? (
                <>
                  {dashboardData.dashboard.unboundAlerts.slice(0, 5).map((alert) => (
                    <div
                      key={alert.id}
                      className="p-3 bg-white border border-amber-100 rounded-lg cursor-pointer hover:border-amber-200 hover:shadow-sm transition-all"
                      onClick={() => handleAlertClick(alert.id)}
                    >
                      <div className="space-y-2">
                        <p className="font-medium text-sm truncate text-amber-900">{alert.title}</p>
                        <p className="text-xs text-amber-600 line-clamp-2">
                          {alert.description}
                        </p>
                        <div className="flex justify-end">
                          <span className="text-xs text-amber-500">
                            {formatDistanceToNow(new Date(alert.createdAt), { addSuffix: true })}
                          </span>
                        </div>
                      </div>
                    </div>
                  ))}
                </>
              ) : (
                <div className="text-center py-8">
                  <AlertCircle className="h-12 w-12 text-amber-300 mx-auto mb-3" />
                  <p className="text-sm text-amber-600">No new alerts</p>
                </div>
              )}
            </CardContent>
          </Card>
        </div>

        {/* Right Section - Activity Feed using full width within the column */}
        <div className="lg:col-span-1">
          <Card className="border-green-200 bg-gradient-to-br from-green-50 to-white h-full">
            <CardHeader className="pb-4">
              <div className="flex items-center gap-3">
                <div className="p-2 bg-green-100 rounded-lg">
                  <Activity className="h-6 w-6 text-green-600" />
                </div>
                <div>
                  <CardTitle className="text-lg font-semibold text-green-900">Activity Feed</CardTitle>
                  <p className="text-sm text-green-600">Recent system activities</p>
                </div>
              </div>
            </CardHeader>
            <CardContent className="space-y-3">
              {activitiesLoading ? (
                <div className="text-sm text-muted-foreground">Loading...</div>
              ) : activitiesData?.activities?.activities && activitiesData.activities.activities.length > 0 ? (
                <>
                  <div className="space-y-3">
                    {activitiesData.activities.activities.map((activity) => {
                      return (
                        <div
                          key={activity.id}
                          className="p-3 bg-white border border-green-100 rounded-lg cursor-pointer hover:border-green-200 hover:shadow-md transition-all duration-200 w-full"
                          onClick={() => handleActivityClick(activity)}
                        >
                          <div className="flex items-start gap-3">
                            {/* User Avatar - Left */}
                            <div className="flex-shrink-0">
                              <UserAvatar userID={activity.userID || ''} />
                            </div>
                            
                            {/* Event Info - Right */}
                            <div className="flex-1 min-w-0">
                              {/* Time */}
                              <div className="text-xs text-gray-500 mb-2">
                                {formatDistanceToNow(new Date(activity.createdAt), { addSuffix: true })}
                              </div>
                              
                              {/* Action with icon */}
                              <div className="flex items-center gap-2 mb-2">
                                {getActivityIcon(activity.type)}
                                <span className="text-sm text-gray-800">
                                  {activity.type === 'ticket_status_changed' && activity.metadata ? 
                                    (() => {
                                      try {
                                        const metadata = JSON.parse(activity.metadata);
                                        return `${metadata.old_status} → ${metadata.new_status}`;
                                      } catch {
                                        return getActivityTitle(activity.type);
                                      }
                                    })() : 
                                    getActivityTitle(activity.type)
                                  }
                                </span>
                              </div>
                              
                              {/* Ticket/Alert info */}
                              {(activity.ticket || activity.alert) && (
                                <div className="mb-2">
                                  <p className="text-sm font-medium text-gray-900 mb-1 leading-tight">
                                    {activity.ticket?.title || activity.alert?.title}
                                  </p>
                                  {/* Show ticket/alert description */}
                                  <p className="text-xs text-gray-500 line-clamp-2">
                                    {(() => {
                                      const description = activity.ticket?.description || activity.alert?.description;
                                      return description && description.length > 50 
                                        ? `${description.substring(0, 50)}...`
                                        : description || '';
                                    })()}
                                  </p>
                                </div>
                              )}
                              
                              {/* Action link */}
                              {(activity.ticketID || activity.alertID) && (
                                <div className="flex items-center gap-1 text-xs text-green-600">
                                  <span>View details</span>
                                  <ArrowRight className="h-3 w-3" />
                                </div>
                              )}
                            </div>
                          </div>
                        </div>
                      );
                    })}
                  </div>
                  
                  {/* Pagination */}
                  {activitiesData.activities.totalCount > 10 && (
                    <div className="flex justify-between items-center pt-4 border-t border-green-100">
                      <Button
                        variant="outline"
                        size="sm"
                        onClick={() => setActivitiesPage(Math.max(0, activitiesPage - 1))}
                        disabled={activitiesPage === 0}
                        className="text-green-600 border-green-200 hover:bg-green-50"
                      >
                        <ChevronLeft className="h-4 w-4" />
                      </Button>
                      <span className="text-xs text-green-600 px-2">
                        {activitiesPage + 1}/{Math.ceil(activitiesData.activities.totalCount / 10)}
                      </span>
                      <Button
                        variant="outline"
                        size="sm"
                        onClick={() => setActivitiesPage(activitiesPage + 1)}
                        disabled={(activitiesPage + 1) * 10 >= activitiesData.activities.totalCount}
                        className="text-green-600 border-green-200 hover:bg-green-50"
                      >
                        <ChevronRight className="h-4 w-4" />
                      </Button>
                    </div>
                  )}
                </>
              ) : (
                <div className="text-center py-8">
                  <Activity className="h-12 w-12 text-green-300 mx-auto mb-3" />
                  <p className="text-sm text-green-600">No recent activities</p>
                </div>
              )}
            </CardContent>
          </Card>
        </div>
      </div>

      <CreateTicketModal
        isOpen={isCreateTicketOpen}
        onClose={() => setIsCreateTicketOpen(false)}
      />
    </div>
  );
}

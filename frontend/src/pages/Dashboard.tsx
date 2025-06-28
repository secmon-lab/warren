import { useQuery } from "@apollo/client";
import { useState } from "react";
import { useNavigate } from "react-router-dom";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { CreateTicketModal } from "@/components/CreateTicketModal";
import { GET_DASHBOARD, GET_ACTIVITIES } from "@/lib/graphql/queries";
import { 
  AlertTriangle, 
  Ticket, 
  Plus, 
  Clock, 
  User, 
  MessageCircle,
  ArrowRight,
  ChevronLeft,
  ChevronRight,
  Activity,
  AlertCircle
} from "lucide-react";
import { formatDistanceToNow } from "date-fns";

export default function Dashboard() {
  const [isCreateTicketOpen, setIsCreateTicketOpen] = useState(false);
  const [activitiesPage, setActivitiesPage] = useState(0);
  const navigate = useNavigate();
  
  const { data: dashboardData, loading: dashboardLoading } = useQuery(GET_DASHBOARD, {
    fetchPolicy: "cache-and-network",
  });

  const { data: activitiesData, loading: activitiesLoading } = useQuery(GET_ACTIVITIES, {
    variables: { offset: activitiesPage * 10, limit: 10 },
    fetchPolicy: "cache-and-network",
  });

  const handleTicketClick = (ticketId: string) => {
    navigate(`/tickets/${ticketId}`);
  };

  const handleAlertClick = (alertId: string) => {
    navigate(`/alerts/${alertId}`);
  };

  const handleActivityClick = (activity: any) => {
    // Priority: Ticket > Alert
    if (activity.ticketID) {
      navigate(`/tickets/${activity.ticketID}`);
    } else if (activity.alertID) {
      navigate(`/alerts/${activity.alertID}`);
    }
  };

  const formatActivityDescription = (activity: any) => {
    switch (activity.type) {
      case "ticket_created":
        return "Created new ticket";
      case "ticket_status_changed":
        return "Changed ticket status";
      case "comment_added":
        return "Added comment";
      case "alert_bound":
        return "Bound alert to ticket";
      case "alerts_bulk_bound":
        return "Bulk bound alerts to ticket";
      default:
        return activity.description;
    }
  };

  const getActivityIcon = (type: string) => {
    switch (type) {
      case "ticket_created":
        return <Ticket className="h-4 w-4 text-blue-500" />;
      case "comment_added":
        return <MessageCircle className="h-4 w-4 text-green-500" />;
      case "alert_bound":
      case "alerts_bulk_bound":
        return <AlertTriangle className="h-4 w-4 text-yellow-500" />;
      default:
        return <Clock className="h-4 w-4 text-gray-500" />;
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
        <Button onClick={() => setIsCreateTicketOpen(true)} className="gap-2 bg-blue-600 hover:bg-blue-700">
          <Plus className="h-4 w-4" />
          Create Ticket
        </Button>
      </div>

      {/* Top Section - Tickets and Alerts in 2 columns */}
      <div className="grid gap-8 lg:grid-cols-2">
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
            ) : dashboardData?.dashboard.openTickets?.length > 0 ? (
              <>
                {dashboardData.dashboard.openTickets.slice(0, 5).map((ticket: any) => (
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

        {/* Unbound Alerts Section */}
        <Card className="border-amber-200 bg-gradient-to-br from-amber-50 to-white">
          <CardHeader className="pb-4">
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-3">
                <div className="p-2 bg-amber-100 rounded-lg">
                  <AlertTriangle className="h-6 w-6 text-amber-600" />
                </div>
                <div>
                  <CardTitle className="text-lg font-semibold text-amber-900">Unbound Alerts</CardTitle>
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
            ) : dashboardData?.dashboard.unboundAlerts?.length > 0 ? (
              <>
                {dashboardData.dashboard.unboundAlerts.slice(0, 5).map((alert: any) => (
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
                <p className="text-sm text-amber-600">No unbound alerts</p>
              </div>
            )}
          </CardContent>
        </Card>
      </div>

      {/* Bottom Section - Activity Feed taking full width */}
      <Card className="border-green-200 bg-gradient-to-br from-green-50 to-white">
        <CardHeader className="pb-4">
          <div className="flex items-center gap-3">
            <div className="p-2 bg-green-100 rounded-lg">
              <Activity className="h-6 w-6 text-green-600" />
            </div>
            <div>
              <CardTitle className="text-xl font-semibold text-green-900">Activity Feed</CardTitle>
              <p className="text-sm text-green-600">Recent system activities</p>
            </div>
          </div>
        </CardHeader>
        <CardContent className="space-y-4">
          {activitiesLoading ? (
            <div className="text-sm text-muted-foreground">Loading...</div>
          ) : activitiesData?.activities?.activities?.length > 0 ? (
            <>
              <div className="grid gap-3 md:grid-cols-2 lg:grid-cols-3">
                {activitiesData.activities.activities.map((activity: any) => (
                  <div
                    key={activity.id}
                    className="p-4 bg-white border border-green-100 rounded-lg cursor-pointer hover:border-green-200 hover:shadow-sm transition-all"
                    onClick={() => handleActivityClick(activity)}
                  >
                    <div className="flex items-start gap-3">
                      <div className="mt-0.5 flex-shrink-0">
                        {getActivityIcon(activity.type)}
                      </div>
                      <div className="flex-1 min-w-0">
                        <p className="text-sm font-medium text-green-900 truncate">{activity.title}</p>
                        <p className="text-xs text-green-600 mt-1">
                          {formatActivityDescription(activity)}
                        </p>
                        <p className="text-xs text-green-500 mt-1">
                          {formatDistanceToNow(new Date(activity.createdAt), { addSuffix: true })}
                        </p>
                      </div>
                    </div>
                  </div>
                ))}
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
                    <ChevronLeft className="h-4 w-4 mr-1" />
                    Previous
                  </Button>
                  <span className="text-sm text-green-600 px-4">
                    Page {activitiesPage + 1} of {Math.ceil(activitiesData.activities.totalCount / 10)}
                  </span>
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => setActivitiesPage(activitiesPage + 1)}
                    disabled={(activitiesPage + 1) * 10 >= activitiesData.activities.totalCount}
                    className="text-green-600 border-green-200 hover:bg-green-50"
                  >
                    Next
                    <ChevronRight className="h-4 w-4 ml-1" />
                  </Button>
                </div>
              )}
            </>
          ) : (
            <div className="text-center py-12">
              <Activity className="h-16 w-16 text-green-300 mx-auto mb-4" />
              <p className="text-lg font-medium text-green-700 mb-2">No recent activities</p>
              <p className="text-sm text-green-600">
                Create tickets or bind alerts to see activities here
              </p>
            </div>
          )}
        </CardContent>
      </Card>

      <CreateTicketModal
        isOpen={isCreateTicketOpen}
        onClose={() => setIsCreateTicketOpen(false)}
      />
    </div>
  );
}

import { useQuery } from "@apollo/client";
import { useState } from "react";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { CreateTicketModal } from "@/components/CreateTicketModal";
import { GET_DASHBOARD } from "@/lib/graphql/queries";
import { AlertTriangle, Ticket, Plus } from "lucide-react";

export default function Dashboard() {
  const [isCreateTicketOpen, setIsCreateTicketOpen] = useState(false);
  
  const { data: dashboardData, loading: dashboardLoading } = useQuery(GET_DASHBOARD, {
    fetchPolicy: "cache-and-network",
  });

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold tracking-tight">ダッシュボード</h1>
          <p className="text-muted-foreground">
            セキュリティ監視とチケット管理システム
          </p>
        </div>
        <Button onClick={() => setIsCreateTicketOpen(true)} className="gap-2">
          <Plus className="h-4 w-4" />
          チケット作成
        </Button>
      </div>

      <div className="grid gap-6 md:grid-cols-2 lg:grid-cols-4">
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">オープンチケット</CardTitle>
            <Ticket className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">
              {dashboardLoading ? "..." : dashboardData?.dashboard.openTicketsCount || 0}
            </div>
            <p className="text-xs text-muted-foreground">未解決のチケット数</p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">未紐づけアラート</CardTitle>
            <AlertTriangle className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">
              {dashboardLoading ? "..." : dashboardData?.dashboard.unboundAlertsCount || 0}
            </div>
            <p className="text-xs text-muted-foreground">チケット未紐づけのアラート数</p>
          </CardContent>
        </Card>
      </div>

      <CreateTicketModal
        isOpen={isCreateTicketOpen}
        onClose={() => setIsCreateTicketOpen(false)}
      />
    </div>
  );
}

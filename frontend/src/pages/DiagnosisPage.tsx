import { useState } from "react";
import { useNavigate } from "react-router-dom";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Stethoscope, Play, Loader2 } from "lucide-react";
import {
  Pagination,
  PaginationContent,
  PaginationItem,
  PaginationLink,
  PaginationNext,
  PaginationPrevious,
} from "@/components/ui/pagination";
import {
  useGetDiagnosesQuery,
  useRunDiagnosisMutation,
} from "@/lib/graphql/generated";
import { useToast } from "@/hooks/use-toast";

const ITEMS_PER_PAGE = 20;

function statusBadgeVariant(status: string): "default" | "secondary" | "destructive" | "outline" {
  switch (status) {
    case "pending":
      return "secondary";
    case "healthy":
      return "outline";
    case "fixed":
      return "default";
    case "partially_fixed":
      return "destructive";
    default:
      return "outline";
  }
}

function statusLabel(status: string): string {
  switch (status) {
    case "pending":
      return "Pending";
    case "healthy":
      return "Healthy";
    case "fixed":
      return "Fixed";
    case "partially_fixed":
      return "Partially Fixed";
    default:
      return status;
  }
}

export default function DiagnosisPage() {
  const navigate = useNavigate();
  const { toast } = useToast();
  const [currentPage, setCurrentPage] = useState(1);

  const { data, loading, error, refetch } = useGetDiagnosesQuery({
    variables: {
      offset: (currentPage - 1) * ITEMS_PER_PAGE,
      limit: ITEMS_PER_PAGE,
    },
  });

  const [runDiagnosis, { loading: running }] = useRunDiagnosisMutation({
    onCompleted: (result) => {
      toast({ title: "Diagnosis started", description: `ID: ${result.runDiagnosis.id}` });
      refetch();
      navigate(`/diagnosis/${result.runDiagnosis.id}`);
    },
    onError: (err) => {
      toast({ title: "Failed to run diagnosis", description: err.message, variant: "destructive" });
    },
  });

  if (error) {
    return (
      <div className="flex items-center justify-center h-64">
        <p className="text-red-500">Error loading diagnoses: {error.message}</p>
      </div>
    );
  }

  const diagnoses = data?.diagnoses?.diagnoses ?? [];
  const totalCount = data?.diagnoses?.totalCount ?? 0;
  const totalPages = Math.ceil(totalCount / ITEMS_PER_PAGE);

  return (
    <div className="container mx-auto p-6">
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-3xl font-bold flex items-center gap-2">
            <Stethoscope className="h-8 w-8" />
            Diagnosis
          </h1>
          <p className="text-muted-foreground mt-2">
            Detect and fix data inconsistencies in the database
          </p>
        </div>
        <Button onClick={() => runDiagnosis()} disabled={running}>
          {running ? (
            <Loader2 className="mr-2 h-4 w-4 animate-spin" />
          ) : (
            <Play className="mr-2 h-4 w-4" />
          )}
          Run Check
        </Button>
      </div>

      {loading ? (
        <div className="space-y-4">
          {[...Array(5)].map((_, i) => (
            <Card key={i} className="animate-pulse">
              <CardHeader>
                <div className="h-5 bg-gray-200 rounded w-1/3" />
              </CardHeader>
              <CardContent>
                <div className="h-4 bg-gray-200 rounded w-1/4" />
              </CardContent>
            </Card>
          ))}
        </div>
      ) : diagnoses.length === 0 ? (
        <Card>
          <CardContent className="py-12">
            <p className="text-center text-muted-foreground">
              No diagnoses yet. Click "Run Check" to start.
            </p>
          </CardContent>
        </Card>
      ) : (
        <div className="space-y-4">
          {diagnoses.map((diag) => (
            <Card
              key={diag.id}
              className="cursor-pointer hover:shadow-md transition-shadow"
              onClick={() => navigate(`/diagnosis/${diag.id}`)}
            >
              <CardHeader className="pb-2">
                <CardTitle className="flex items-center justify-between text-base">
                  <span className="font-mono text-sm text-muted-foreground">{diag.id}</span>
                  <Badge variant={statusBadgeVariant(diag.status)}>
                    {statusLabel(diag.status)}
                  </Badge>
                </CardTitle>
              </CardHeader>
              <CardContent>
                <div className="flex gap-6 text-sm">
                  <span>Total: <strong>{diag.totalCount}</strong></span>
                  <span className="text-yellow-600">Pending: <strong>{diag.pendingCount}</strong></span>
                  <span className="text-green-600">Fixed: <strong>{diag.fixedCount}</strong></span>
                  <span className="text-red-600">Failed: <strong>{diag.failedCount}</strong></span>
                </div>
                <p className="text-xs text-muted-foreground mt-2">
                  Created: {diag.createdAt} &nbsp;·&nbsp; Updated: {diag.updatedAt}
                </p>
              </CardContent>
            </Card>
          ))}
        </div>
      )}

      {!loading && totalPages > 1 && (
        <div className="mt-6">
          <Pagination>
            <PaginationContent>
              <PaginationItem>
                <PaginationPrevious
                  onClick={() => setCurrentPage((p) => Math.max(1, p - 1))}
                  className={currentPage === 1 ? "pointer-events-none opacity-50" : "cursor-pointer"}
                />
              </PaginationItem>
              {[...Array(Math.min(5, totalPages))].map((_, i) => {
                const pageNum = i + 1 + Math.max(0, currentPage - 3);
                if (pageNum > totalPages) return null;
                return (
                  <PaginationItem key={pageNum}>
                    <PaginationLink
                      onClick={() => setCurrentPage(pageNum)}
                      isActive={currentPage === pageNum}
                      className="cursor-pointer"
                    >
                      {pageNum}
                    </PaginationLink>
                  </PaginationItem>
                );
              })}
              <PaginationItem>
                <PaginationNext
                  onClick={() => setCurrentPage((p) => Math.min(totalPages, p + 1))}
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

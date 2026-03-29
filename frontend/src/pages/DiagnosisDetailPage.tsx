import { useState } from "react";
import { useParams, useNavigate } from "react-router-dom";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { ArrowLeft, Wrench, Loader2, Stethoscope } from "lucide-react";
import {
  Pagination,
  PaginationContent,
  PaginationItem,
  PaginationLink,
  PaginationNext,
  PaginationPrevious,
} from "@/components/ui/pagination";
import {
  useGetDiagnosisQuery,
  useGetDiagnosisIssuesQuery,
  useFixDiagnosisMutation,
} from "@/lib/graphql/generated";
import { useToast } from "@/hooks/use-toast";

const ITEMS_PER_PAGE = 50;

const KNOWN_RULE_IDS = [
  "missing_alert_embedding",
  "missing_ticket_embedding",
  "legacy_alert_status",
  "legacy_ticket_status",
  "binding_mismatch",
  "orphaned_tag_id",
  "missing_alert_metadata",
  "legacy_knowledge",
];

function issueStatusBadgeVariant(status: string): "default" | "secondary" | "destructive" | "outline" {
  switch (status) {
    case "pending":
      return "secondary";
    case "fixed":
      return "default";
    case "failed":
      return "destructive";
    default:
      return "outline";
  }
}

function diagnosisStatusBadgeVariant(status: string): "default" | "secondary" | "destructive" | "outline" {
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

function diagnosisStatusLabel(status: string): string {
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

export default function DiagnosisDetailPage() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const { toast } = useToast();
  const [currentPage, setCurrentPage] = useState(1);
  const [statusFilter, setStatusFilter] = useState<string>("all");
  const [ruleFilter, setRuleFilter] = useState<string>("all");

  const { data: diagnosisData, loading: diagnosisLoading, refetch: refetchDiagnosis } =
    useGetDiagnosisQuery({
      variables: { id: id! },
      skip: !id,
    });

  const { data: issuesData, loading: issuesLoading, refetch: refetchIssues } =
    useGetDiagnosisIssuesQuery({
      variables: {
        diagnosisID: id!,
        offset: (currentPage - 1) * ITEMS_PER_PAGE,
        limit: ITEMS_PER_PAGE,
        status: statusFilter !== "all" ? statusFilter : undefined,
        ruleID: ruleFilter !== "all" ? ruleFilter : undefined,
      },
      skip: !id,
    });

  const [fixDiagnosis, { loading: fixing }] = useFixDiagnosisMutation({
    onCompleted: () => {
      toast({ title: "Fix completed", description: "All pending issues have been processed." });
      refetchDiagnosis();
      refetchIssues();
    },
    onError: (err) => {
      toast({ title: "Fix failed", description: err.message, variant: "destructive" });
    },
  });

  const diagnosis = diagnosisData?.diagnosis;
  const issues = issuesData?.diagnosisIssues?.issues ?? [];
  const totalCount = issuesData?.diagnosisIssues?.totalCount ?? 0;
  const totalPages = Math.ceil(totalCount / ITEMS_PER_PAGE);

  const canFix = diagnosis?.status === "pending" || diagnosis?.status === "partially_fixed";

  function handleStatusChange(v: string) {
    setStatusFilter(v);
    setCurrentPage(1);
  }

  function handleRuleChange(v: string) {
    setRuleFilter(v);
    setCurrentPage(1);
  }

  return (
    <div className="container mx-auto p-6">
      <div className="flex items-center gap-3 mb-6">
        <Button variant="ghost" size="sm" onClick={() => navigate("/diagnosis")}>
          <ArrowLeft className="h-4 w-4 mr-1" />
          Back
        </Button>
        <h1 className="text-2xl font-bold flex items-center gap-2">
          <Stethoscope className="h-6 w-6" />
          Diagnosis Detail
        </h1>
      </div>

      {/* Diagnosis summary card */}
      {diagnosisLoading ? (
        <Card className="mb-6 animate-pulse">
          <CardHeader>
            <div className="h-5 bg-gray-200 rounded w-1/3" />
          </CardHeader>
          <CardContent>
            <div className="h-4 bg-gray-200 rounded w-1/4" />
          </CardContent>
        </Card>
      ) : diagnosis ? (
        <Card className="mb-6">
          <CardHeader className="pb-2">
            <CardTitle className="flex items-center justify-between">
              <span className="font-mono text-sm text-muted-foreground">{diagnosis.id}</span>
              <div className="flex items-center gap-3">
                <Badge variant={diagnosisStatusBadgeVariant(diagnosis.status)}>
                  {diagnosisStatusLabel(diagnosis.status)}
                </Badge>
                {canFix && (
                  <Button
                    size="sm"
                    onClick={() => fixDiagnosis({ variables: { id: id! } })}
                    disabled={fixing}
                  >
                    {fixing ? (
                      <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                    ) : (
                      <Wrench className="mr-2 h-4 w-4" />
                    )}
                    Fix All
                  </Button>
                )}
              </div>
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="flex gap-6 text-sm">
              <span>Total: <strong>{diagnosis.totalCount}</strong></span>
              <span className="text-yellow-600">Pending: <strong>{diagnosis.pendingCount}</strong></span>
              <span className="text-green-600">Fixed: <strong>{diagnosis.fixedCount}</strong></span>
              <span className="text-red-600">Failed: <strong>{diagnosis.failedCount}</strong></span>
            </div>
            <p className="text-xs text-muted-foreground mt-2">
              Created: {diagnosis.createdAt} &nbsp;·&nbsp; Updated: {diagnosis.updatedAt}
            </p>
          </CardContent>
        </Card>
      ) : (
        <Card className="mb-6">
          <CardContent className="py-8">
            <p className="text-center text-muted-foreground">Diagnosis not found.</p>
          </CardContent>
        </Card>
      )}

      {/* Filters — applied server-side so they cover all pages */}
      <div className="flex gap-4 mb-4">
        <Select value={statusFilter} onValueChange={handleStatusChange}>
          <SelectTrigger className="w-40">
            <SelectValue placeholder="Status" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">All Statuses</SelectItem>
            <SelectItem value="pending">Pending</SelectItem>
            <SelectItem value="fixed">Fixed</SelectItem>
            <SelectItem value="failed">Failed</SelectItem>
          </SelectContent>
        </Select>

        <Select value={ruleFilter} onValueChange={handleRuleChange}>
          <SelectTrigger className="w-56">
            <SelectValue placeholder="Rule" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">All Rules</SelectItem>
            {KNOWN_RULE_IDS.map((ruleID) => (
              <SelectItem key={ruleID} value={ruleID}>
                {ruleID}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>

      {/* Issues list */}
      {issuesLoading ? (
        <div className="space-y-3">
          {[...Array(5)].map((_, i) => (
            <Card key={i} className="animate-pulse">
              <CardContent className="py-4">
                <div className="h-4 bg-gray-200 rounded w-3/4" />
              </CardContent>
            </Card>
          ))}
        </div>
      ) : issues.length === 0 ? (
        <Card>
          <CardContent className="py-12">
            <p className="text-center text-muted-foreground">
              {totalCount === 0 && statusFilter === "all" && ruleFilter === "all"
                ? "No issues found for this diagnosis."
                : "No issues match the current filters."}
            </p>
          </CardContent>
        </Card>
      ) : (
        <div className="border rounded-lg divide-y">
          {issues.map((issue) => (
            <div key={issue.id} className="flex items-center gap-3 px-4 py-2 hover:bg-muted/40 text-sm">
              <Badge variant="outline" className="text-xs shrink-0 font-mono">
                {issue.ruleID}
              </Badge>
              <span className="font-mono text-xs text-muted-foreground shrink-0 hidden sm:block">
                {issue.targetID.slice(0, 8)}…
              </span>
              <span className="flex-1 min-w-0 truncate text-muted-foreground" title={issue.description}>
                {issue.description}
                {issue.failReason && (
                  <span className="text-red-500 ml-2">({issue.failReason})</span>
                )}
              </span>
              <span className="text-xs text-muted-foreground shrink-0 hidden md:block">
                {issue.fixedAt ?? issue.createdAt}
              </span>
              <Badge variant={issueStatusBadgeVariant(issue.status)} className="shrink-0 text-xs">
                {issue.status}
              </Badge>
            </div>
          ))}
        </div>
      )}

      {/* Pagination */}
      {!issuesLoading && totalPages > 1 && (
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

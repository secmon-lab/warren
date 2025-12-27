import { useState } from "react";
import { useParams, useNavigate } from "react-router-dom";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { ArrowLeft, Brain, Search } from "lucide-react";
import {
  Pagination,
  PaginationContent,
  PaginationItem,
  PaginationLink,
  PaginationNext,
  PaginationPrevious,
} from "@/components/ui/pagination";
import {
  useListAgentMemoriesQuery,
  MemorySortField,
  SortOrder,
} from "@/lib/graphql/generated";
import { calculateFinalScore } from "@/utils/memoryScore";

export default function AgentMemoriesPage() {
  const { agentId } = useParams<{ agentId: string }>();
  const navigate = useNavigate();
  const [currentPage, setCurrentPage] = useState(1);
  const [searchQuery, setSearchQuery] = useState("");
  const [sortBy, setSortBy] = useState<MemorySortField>("CREATED_AT");
  const [sortOrder, setSortOrder] = useState<SortOrder>("DESC");
  const ITEMS_PER_PAGE = 20;

  const { data, loading, error } = useListAgentMemoriesQuery({
    variables: {
      agentID: agentId!,
      offset: (currentPage - 1) * ITEMS_PER_PAGE,
      limit: ITEMS_PER_PAGE,
      sortBy,
      sortOrder,
      keyword: searchQuery || undefined,
    },
    skip: !agentId,
  });

  if (!agentId) {
    return (
      <div className="flex items-center justify-center h-screen">
        <p className="text-red-500">Agent ID is required</p>
      </div>
    );
  }

  if (error) {
    return (
      <div className="flex items-center justify-center h-screen">
        <p className="text-red-500">Error loading memories: {error.message}</p>
      </div>
    );
  }

  const memories = data?.listAgentMemories?.memories || [];
  const totalCount = data?.listAgentMemories?.totalCount || 0;
  const totalPages = Math.ceil(totalCount / ITEMS_PER_PAGE);

  // Calculate FinalScore for each memory
  const memoriesWithScore = memories.map((mem) => ({
    ...mem,
    finalScore: calculateFinalScore(mem.score, mem.lastUsedAt).finalScore,
  }));

  return (
    <div className="container mx-auto p-6">
      {/* Header */}
      <div className="mb-6">
        <Button
          variant="ghost"
          className="mb-4"
          onClick={() => navigate("/memory")}
        >
          <ArrowLeft className="mr-2 h-4 w-4" />
          Back to Agents
        </Button>
        <h1 className="text-3xl font-bold flex items-center gap-2">
          <Brain className="h-8 w-8" />
          {agentId} Memories
        </h1>
        <p className="text-muted-foreground mt-2">
          {totalCount} {totalCount === 1 ? "memory" : "memories"} found
        </p>
      </div>

      {/* Filters */}
      <div className="mb-6 space-y-4">
        <div className="relative">
          <Search className="absolute left-3 top-3 h-4 w-4 text-muted-foreground" />
          <Input
            placeholder="Search in claims and queries..."
            value={searchQuery}
            onChange={(e) => {
              setSearchQuery(e.target.value);
              setCurrentPage(1);
            }}
            className="pl-9"
          />
        </div>

        <div className="flex gap-4">
          <Select
            value={sortBy}
            onValueChange={(value) => setSortBy(value as MemorySortField)}
          >
            <SelectTrigger className="w-[200px]">
              <SelectValue placeholder="Sort by" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value={"SCORE"}>Score</SelectItem>
              <SelectItem value={"CREATED_AT"}>Created At</SelectItem>
              <SelectItem value={"LAST_USED_AT"}>Last Used At</SelectItem>
            </SelectContent>
          </Select>

          <Select
            value={sortOrder}
            onValueChange={(value) => setSortOrder(value as SortOrder)}
          >
            <SelectTrigger className="w-[150px]">
              <SelectValue placeholder="Order" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value={"DESC"}>Descending</SelectItem>
              <SelectItem value={"ASC"}>Ascending</SelectItem>
            </SelectContent>
          </Select>
        </div>
      </div>

      {/* Memory Cards */}
      {loading ? (
        <div className="space-y-4">
          {[...Array(5)].map((_, i) => (
            <Card key={i} className="animate-pulse">
              <CardHeader>
                <div className="h-6 bg-gray-200 rounded w-3/4"></div>
              </CardHeader>
              <CardContent>
                <div className="h-4 bg-gray-200 rounded w-full mb-2"></div>
                <div className="h-4 bg-gray-200 rounded w-1/2"></div>
              </CardContent>
            </Card>
          ))}
        </div>
      ) : memoriesWithScore.length === 0 ? (
        <Card>
          <CardContent className="py-8">
            <p className="text-center text-muted-foreground">
              {searchQuery ? "No memories found matching your search" : "No memories found"}
            </p>
          </CardContent>
        </Card>
      ) : (
        <div className="space-y-4">
          {memoriesWithScore.map((mem) => (
            <Card
              key={mem.id}
              className="cursor-pointer hover:shadow-lg transition-shadow"
              onClick={() => navigate(`/memory/${agentId}/${mem.id}`)}
            >
              <CardHeader>
                <CardTitle className="text-lg">{mem.query}</CardTitle>
              </CardHeader>
              <CardContent>
                <p className="text-sm text-muted-foreground mb-4 line-clamp-2">
                  {mem.claim}
                </p>
                <div className="flex flex-wrap gap-4 text-xs text-muted-foreground">
                  <span>Score: <strong className={mem.score >= 0 ? "text-green-600" : "text-red-600"}>{mem.score.toFixed(1)}</strong></span>
                  <span>Final Score: <strong>{mem.finalScore.toFixed(3)}</strong></span>
                  <span>Created: {new Date(mem.createdAt).toLocaleDateString()}</span>
                  {mem.lastUsedAt && (
                    <span>Last Used: {new Date(mem.lastUsedAt).toLocaleDateString()}</span>
                  )}
                </div>
              </CardContent>
            </Card>
          ))}
        </div>
      )}

      {/* Pagination */}
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

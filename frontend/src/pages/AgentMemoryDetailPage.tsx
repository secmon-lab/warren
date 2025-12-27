import { useParams, useNavigate } from "react-router-dom";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { ArrowLeft, Brain } from "lucide-react";
import { useGetAgentMemoryQuery } from "@/lib/graphql/generated";
import { calculateFinalScore } from "@/utils/memoryScore";

export default function AgentMemoryDetailPage() {
  const { agentId, memoryId } = useParams<{ agentId: string; memoryId: string }>();
  const navigate = useNavigate();

  const { data, loading, error } = useGetAgentMemoryQuery({
    variables: {
      agentID: agentId!,
      memoryID: memoryId!,
    },
    skip: !agentId || !memoryId,
  });

  if (!agentId || !memoryId) {
    return (
      <div className="flex items-center justify-center h-screen">
        <p className="text-red-500">Agent ID and Memory ID are required</p>
      </div>
    );
  }

  if (loading) {
    return (
      <div className="container mx-auto p-6">
        <div className="animate-pulse space-y-4">
          <div className="h-8 bg-gray-200 rounded w-1/4"></div>
          <div className="h-64 bg-gray-200 rounded"></div>
        </div>
      </div>
    );
  }

  if (error || !data?.getAgentMemory) {
    return (
      <div className="flex items-center justify-center h-screen">
        <p className="text-red-500">
          {error ? `Error loading memory: ${error.message}` : "Memory not found"}
        </p>
      </div>
    );
  }

  const memory = data.getAgentMemory;
  const breakdown = calculateFinalScore(memory.score, memory.lastUsedAt);

  return (
    <div className="container mx-auto p-6 max-w-4xl">
      {/* Header */}
      <div className="mb-6">
        <Button
          variant="ghost"
          className="mb-4"
          onClick={() => navigate(`/memory/${agentId}`)}
        >
          <ArrowLeft className="mr-2 h-4 w-4" />
          Back to {agentId} Memories
        </Button>
        <h1 className="text-3xl font-bold flex items-center gap-2">
          <Brain className="h-8 w-8" />
          Memory Detail
        </h1>
      </div>

      {/* Memory Details */}
      <div className="space-y-6">
        {/* Basic Info */}
        <Card>
          <CardHeader>
            <CardTitle>Basic Information</CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            <div>
              <label className="text-sm font-medium text-muted-foreground">Memory ID</label>
              <p className="text-sm font-mono">{memory.id}</p>
            </div>
            <div>
              <label className="text-sm font-medium text-muted-foreground">Agent ID</label>
              <p className="text-sm">{memory.agentID}</p>
            </div>
            <div>
              <label className="text-sm font-medium text-muted-foreground">Query</label>
              <p className="text-sm">{memory.query}</p>
            </div>
            <div>
              <label className="text-sm font-medium text-muted-foreground">Claim</label>
              <p className="text-sm whitespace-pre-wrap">{memory.claim}</p>
            </div>
          </CardContent>
        </Card>

        {/* Scores */}
        <Card>
          <CardHeader>
            <CardTitle>Scoring Information</CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            <div>
              <label className="text-sm font-medium text-muted-foreground">Quality Score</label>
              <div className="flex items-center gap-2">
                <Badge variant={memory.score >= 0 ? "default" : "destructive"}>
                  {memory.score.toFixed(2)}
                </Badge>
                <span className="text-xs text-muted-foreground">(-10.0 to +10.0)</span>
              </div>
            </div>

            <div>
              <label className="text-sm font-medium text-muted-foreground">Final Score (Selection)</label>
              <div className="flex items-center gap-2">
                <Badge>{breakdown.finalScore.toFixed(4)}</Badge>
                <span className="text-xs text-muted-foreground">(0.0 to 1.0)</span>
              </div>
            </div>

            {/* Score Breakdown */}
            <div className="mt-4 p-4 bg-muted rounded-lg space-y-3">
              <p className="text-sm font-medium">Score Breakdown</p>
              <div className="space-y-2">
                <div className="flex justify-between items-center">
                  <span className="text-xs text-muted-foreground">Similarity (50%)</span>
                  <span className="text-xs font-mono">{breakdown.similarity.toFixed(4)}</span>
                </div>
                <div className="w-full bg-gray-200 rounded-full h-2">
                  <div
                    className="bg-blue-500 h-2 rounded-full"
                    style={{ width: `${breakdown.similarity * 100}%` }}
                  ></div>
                </div>

                <div className="flex justify-between items-center">
                  <span className="text-xs text-muted-foreground">Quality (30%)</span>
                  <span className="text-xs font-mono">{breakdown.quality.toFixed(4)}</span>
                </div>
                <div className="w-full bg-gray-200 rounded-full h-2">
                  <div
                    className="bg-green-500 h-2 rounded-full"
                    style={{ width: `${breakdown.quality * 100}%` }}
                  ></div>
                </div>

                <div className="flex justify-between items-center">
                  <span className="text-xs text-muted-foreground">Recency (20%)</span>
                  <span className="text-xs font-mono">{breakdown.recency.toFixed(4)}</span>
                </div>
                <div className="w-full bg-gray-200 rounded-full h-2">
                  <div
                    className="bg-purple-500 h-2 rounded-full"
                    style={{ width: `${breakdown.recency * 100}%` }}
                  ></div>
                </div>
              </div>
            </div>
          </CardContent>
        </Card>

        {/* Timestamps */}
        <Card>
          <CardHeader>
            <CardTitle>Timestamps</CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            <div>
              <label className="text-sm font-medium text-muted-foreground">Created At</label>
              <p className="text-sm">{new Date(memory.createdAt).toISOString().split('T')[0].replace(/-/g, '/')} {new Date(memory.createdAt).toISOString().split('T')[1].split('.')[0]}</p>
            </div>
            <div>
              <label className="text-sm font-medium text-muted-foreground">Last Used At</label>
              <p className="text-sm">
                {memory.lastUsedAt
                  ? `${new Date(memory.lastUsedAt).toISOString().split('T')[0].replace(/-/g, '/')} ${new Date(memory.lastUsedAt).toISOString().split('T')[1].split('.')[0]}`
                  : "Never used"}
              </p>
            </div>
          </CardContent>
        </Card>
      </div>
    </div>
  );
}

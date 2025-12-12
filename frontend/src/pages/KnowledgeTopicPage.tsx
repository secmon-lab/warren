import { useQuery } from "@apollo/client";
import { useParams, useNavigate } from "react-router-dom";
import { useState } from "react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { GET_KNOWLEDGES_BY_TOPIC } from "@/lib/graphql/queries";
import { Knowledge } from "@/lib/types";
import {
  ChevronDown,
  ChevronRight,
  ArrowLeft,
  User,
  Calendar,
  FileText,
} from "lucide-react";
import { formatDistanceToNow } from "date-fns";

export default function KnowledgeTopicPage() {
  const { topic } = useParams<{ topic: string }>();
  const navigate = useNavigate();
  const [expandedItems, setExpandedItems] = useState<Set<string>>(new Set());

  const { data, loading, error } = useQuery(GET_KNOWLEDGES_BY_TOPIC, {
    variables: { topic },
    skip: !topic,
  });

  const toggleItem = (slug: string) => {
    const newExpanded = new Set(expandedItems);
    if (newExpanded.has(slug)) {
      newExpanded.delete(slug);
    } else {
      newExpanded.add(slug);
    }
    setExpandedItems(newExpanded);
  };

  if (!topic) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="text-lg text-red-600">Topic parameter is missing</div>
      </div>
    );
  }

  if (loading) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="text-lg">Loading knowledge...</div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="text-lg text-red-600">
          Error loading knowledge: {error.message}
        </div>
      </div>
    );
  }

  const knowledges: Knowledge[] = data?.knowledgesByTopic || [];

  return (
    <div className="space-y-6">
      <div className="flex items-center gap-4">
        <Button
          variant="ghost"
          size="sm"
          onClick={() => navigate("/knowledge")}
          className="flex items-center gap-2">
          <ArrowLeft className="h-4 w-4" />
          Back to Topics
        </Button>
      </div>

      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold tracking-tight">
            Knowledge: {topic}
          </h1>
          <p className="text-muted-foreground">
            {knowledges.length} knowledge item
            {knowledges.length !== 1 ? "s" : ""}
          </p>
        </div>
      </div>

      {knowledges.length === 0 ? (
        <Card>
          <CardContent className="flex items-center justify-center h-32">
            <p className="text-muted-foreground">
              No knowledge found for this topic
            </p>
          </CardContent>
        </Card>
      ) : (
        <div className="space-y-3">
          {knowledges.map((knowledge) => {
            const isExpanded = expandedItems.has(knowledge.slug);
            return (
              <Card key={knowledge.slug}>
                <CardHeader>
                  <div className="flex items-start justify-between">
                    <div className="flex-1">
                      <div className="flex items-center gap-2">
                        <Button
                          variant="ghost"
                          size="sm"
                          onClick={() => toggleItem(knowledge.slug)}
                          className="p-0 h-6 w-6">
                          {isExpanded ? (
                            <ChevronDown className="h-4 w-4" />
                          ) : (
                            <ChevronRight className="h-4 w-4" />
                          )}
                        </Button>
                        <CardTitle
                          className="text-lg cursor-pointer"
                          onClick={() => toggleItem(knowledge.slug)}>
                          {knowledge.name}
                        </CardTitle>
                        <Badge
                          variant={
                            knowledge.state === "active"
                              ? "default"
                              : "secondary"
                          }>
                          {knowledge.state}
                        </Badge>
                      </div>
                      <div className="ml-8 mt-2 space-y-1">
                        <div className="text-xs text-muted-foreground">
                          Slug: {knowledge.slug}
                        </div>
                        <div className="flex items-center gap-4 text-sm text-muted-foreground">
                          <div className="flex items-center gap-1">
                            <User className="h-3 w-3" />
                            <span>{knowledge.author.name}</span>
                          </div>
                          <div className="flex items-center gap-1">
                            <Calendar className="h-3 w-3" />
                            <span>
                              Updated{" "}
                              {formatDistanceToNow(
                                new Date(knowledge.updatedAt),
                                {
                                  addSuffix: true,
                                }
                              )}
                            </span>
                          </div>
                        </div>
                      </div>
                    </div>
                  </div>
                </CardHeader>
                {isExpanded && (
                  <CardContent>
                    <div className="space-y-3">
                      <div>
                        <h4 className="text-sm font-semibold mb-2 flex items-center gap-2">
                          <FileText className="h-4 w-4" />
                          Content
                        </h4>
                        <div className="bg-muted p-4 rounded-md border">
                          <pre className="whitespace-pre-wrap text-sm font-mono">
                            {knowledge.content}
                          </pre>
                        </div>
                      </div>
                      <div className="text-xs text-muted-foreground space-y-1">
                        <div>
                          <span className="font-medium">Commit ID:</span>{" "}
                          <span className="font-mono">
                            {knowledge.commitID.substring(0, 16)}...
                          </span>
                        </div>
                        <div>
                          <span className="font-medium">Created:</span>{" "}
                          {formatDistanceToNow(new Date(knowledge.createdAt), {
                            addSuffix: true,
                          })}
                        </div>
                      </div>
                    </div>
                  </CardContent>
                )}
              </Card>
            );
          })}
        </div>
      )}
    </div>
  );
}


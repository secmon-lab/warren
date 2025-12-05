import { useQuery } from "@apollo/client";
import { useNavigate } from "react-router-dom";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { GET_KNOWLEDGE_TOPICS } from "@/lib/graphql/queries";
import { TopicSummary } from "@/lib/types";
import { BookOpen, FileText } from "lucide-react";

export default function KnowledgePage() {
  const navigate = useNavigate();

  const { data, loading, error } = useQuery(GET_KNOWLEDGE_TOPICS);

  if (loading) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="text-lg">Loading knowledge topics...</div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="text-lg text-red-600">
          Error loading knowledge topics: {error.message}
        </div>
      </div>
    );
  }

  const topics: TopicSummary[] = data?.knowledgeTopics || [];

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold tracking-tight">Knowledge Base</h1>
          <p className="text-muted-foreground">
            Browse knowledge organized by topics
          </p>
        </div>
      </div>

      {topics.length === 0 ? (
        <Card>
          <CardContent className="flex items-center justify-center h-32">
            <p className="text-muted-foreground">No knowledge topics found</p>
          </CardContent>
        </Card>
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
          {topics.map((topicSummary) => (
            <Card
              key={topicSummary.topic}
              className="cursor-pointer hover:shadow-md transition-shadow"
              onClick={() => navigate(`/knowledge/${topicSummary.topic}`)}>
              <CardHeader>
                <div className="flex items-start justify-between">
                  <div className="flex items-center gap-2">
                    <BookOpen className="h-5 w-5 text-primary" />
                    <CardTitle className="text-lg">
                      {topicSummary.topic}
                    </CardTitle>
                  </div>
                  <Badge variant="secondary" className="ml-2">
                    {topicSummary.count}
                  </Badge>
                </div>
              </CardHeader>
              <CardContent>
                <div className="flex items-center text-sm text-muted-foreground">
                  <FileText className="h-4 w-4 mr-1" />
                  <span>
                    {topicSummary.count} knowledge{" "}
                    {topicSummary.count !== 1 ? "items" : "item"}
                  </span>
                </div>
              </CardContent>
            </Card>
          ))}
        </div>
      )}
    </div>
  );
}


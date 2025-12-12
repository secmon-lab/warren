import { useQuery, useMutation } from "@apollo/client";
import { useParams, useNavigate } from "react-router-dom";
import { useState } from "react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import {
  GET_KNOWLEDGES_BY_TOPIC,
  ARCHIVE_KNOWLEDGE,
} from "@/lib/graphql/queries";
import { Knowledge } from "@/lib/types";
import { KnowledgeEditDialog } from "@/components/KnowledgeEditDialog";
import {
  ChevronDown,
  ChevronRight,
  ArrowLeft,
  Calendar,
  FileText,
  Plus,
  Edit,
  Archive,
} from "lucide-react";
import { formatDistanceToNow } from "date-fns";

export default function KnowledgeTopicPage() {
  const { topic } = useParams<{ topic: string }>();
  const navigate = useNavigate();
  const [expandedItems, setExpandedItems] = useState<Set<string>>(new Set());
  const [editDialogOpen, setEditDialogOpen] = useState(false);
  const [editMode, setEditMode] = useState<"create" | "edit">("create");
  const [selectedKnowledge, setSelectedKnowledge] = useState<
    Knowledge | undefined
  >(undefined);
  const [archiveDialogOpen, setArchiveDialogOpen] = useState(false);
  const [knowledgeToArchive, setKnowledgeToArchive] = useState<
    Knowledge | undefined
  >(undefined);

  const { data, loading, error, refetch } = useQuery(GET_KNOWLEDGES_BY_TOPIC, {
    variables: { topic },
    skip: !topic,
  });

  const [archiveKnowledge, { loading: archiving }] = useMutation(
    ARCHIVE_KNOWLEDGE,
    {
      onCompleted: () => {
        refetch();
        setArchiveDialogOpen(false);
        setKnowledgeToArchive(undefined);
      },
      onError: (error) => {
        console.error("Failed to archive knowledge:", error);
        alert(`Failed to archive: ${error.message}`);
      },
    }
  );

  const toggleItem = (slug: string) => {
    const newExpanded = new Set(expandedItems);
    if (newExpanded.has(slug)) {
      newExpanded.delete(slug);
    } else {
      newExpanded.add(slug);
    }
    setExpandedItems(newExpanded);
  };

  const handleCreateClick = () => {
    setEditMode("create");
    setSelectedKnowledge(undefined);
    setEditDialogOpen(true);
  };

  const handleEditClick = (knowledge: Knowledge) => {
    setEditMode("edit");
    setSelectedKnowledge(knowledge);
    setEditDialogOpen(true);
  };

  const handleArchiveClick = (knowledge: Knowledge) => {
    setKnowledgeToArchive(knowledge);
    setArchiveDialogOpen(true);
  };

  const handleArchiveConfirm = async () => {
    if (!knowledgeToArchive) return;
    await archiveKnowledge({
      variables: {
        topic: knowledgeToArchive.topic,
        slug: knowledgeToArchive.slug,
      },
    });
  };

  const handleDialogSuccess = () => {
    refetch();
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
        <Button onClick={handleCreateClick} className="flex items-center gap-2">
          <Plus className="h-4 w-4" />
          Add Knowledge
        </Button>
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
                            <Avatar className="h-4 w-4">
                              {knowledge.author.icon && (
                                <AvatarImage
                                  src={knowledge.author.icon}
                                  alt={knowledge.author.name}
                                />
                              )}
                              <AvatarFallback className="text-[10px]">
                                {knowledge.author.name.charAt(0).toUpperCase()}
                              </AvatarFallback>
                            </Avatar>
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
                    <div className="flex items-center gap-2">
                      <Button
                        variant="outline"
                        size="sm"
                        onClick={() => handleEditClick(knowledge)}
                        className="flex items-center gap-1">
                        <Edit className="h-3 w-3" />
                        Edit
                      </Button>
                      <Button
                        variant="outline"
                        size="sm"
                        onClick={() => handleArchiveClick(knowledge)}
                        className="flex items-center gap-1 text-orange-600 hover:text-orange-700">
                        <Archive className="h-3 w-3" />
                        Archive
                      </Button>
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

      <KnowledgeEditDialog
        open={editDialogOpen}
        onOpenChange={setEditDialogOpen}
        mode={editMode}
        topic={topic || ""}
        knowledge={selectedKnowledge}
        onSuccess={handleDialogSuccess}
      />

      <AlertDialog open={archiveDialogOpen} onOpenChange={setArchiveDialogOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Archive Knowledge</AlertDialogTitle>
            <AlertDialogDescription>
              Are you sure you want to archive "{knowledgeToArchive?.name}"?
              This will remove it from the active knowledge list, but the data
              will be preserved.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction
              onClick={handleArchiveConfirm}
              disabled={archiving}
              className="bg-orange-600 hover:bg-orange-700">
              {archiving ? "Archiving..." : "Archive"}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
}


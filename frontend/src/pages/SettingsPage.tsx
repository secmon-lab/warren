import { useState } from "react";
import { useQuery, useMutation } from "@apollo/client";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Input } from "@/components/ui/input";
import { useToast } from "@/hooks/use-toast";
import { GET_TAGS, CREATE_TAG, DELETE_TAG } from "@/lib/graphql/queries";
import { generateTagColor } from "@/lib/tag-colors";
import { TagMetadata } from "@/lib/types";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
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
import { Settings, Tag, Trash2, Plus, Clock, Calendar, Edit } from "lucide-react";
import { formatRelativeTime } from "@/lib/utils-extended";
import EditTagModal from "@/components/EditTagModal";


export default function SettingsPage() {
  const { toast } = useToast();
  const [newTagName, setNewTagName] = useState("");
  const [tagToDelete, setTagToDelete] = useState<string | null>(null);
  const [tagToEdit, setTagToEdit] = useState<TagMetadata | null>(null);
  const [isEditModalOpen, setIsEditModalOpen] = useState(false);

  const { data: tagsData, loading: tagsLoading, refetch: refetchTags } = useQuery(GET_TAGS);
  
  const [createTag, { loading: createLoading }] = useMutation(CREATE_TAG, {
    onCompleted: () => {
      toast({
        title: "Tag Created",
        description: "The tag has been created successfully.",
      });
      setNewTagName("");
      refetchTags();
    },
    onError: (error) => {
      toast({
        title: "Error",
        description: `Failed to create tag: ${error.message}`,
        variant: "destructive",
      });
    },
  });

  const [deleteTag, { loading: deleteLoading }] = useMutation(DELETE_TAG, {
    onCompleted: () => {
      toast({
        title: "Tag Deleted",
        description: "The tag has been deleted successfully.",
      });
      refetchTags();
    },
    onError: (error) => {
      toast({
        title: "Error",
        description: `Failed to delete tag: ${error.message}`,
        variant: "destructive",
      });
    },
  });

  const handleCreateTag = async () => {
    if (!newTagName.trim()) return;

    try {
      await createTag({
        variables: { name: newTagName.trim() },
      });
    } catch (error) {
      console.error("Error creating tag:", error);
    }
  };

  const handleDeleteTag = async () => {
    if (!tagToDelete) return;

    try {
      await deleteTag({
        variables: { name: tagToDelete },
      });
      setTagToDelete(null);
    } catch (error) {
      console.error("Error deleting tag:", error);
    }
  };

  const handleEditTag = (tag: TagMetadata) => {
    setTagToEdit(tag);
    setIsEditModalOpen(true);
  };

  const handleEditModalClose = () => {
    setIsEditModalOpen(false);
    setTagToEdit(null);
  };

  const handleEditSuccess = () => {
    refetchTags();
  };

  const tags: TagMetadata[] = tagsData?.tags || [];

  return (
    <div className="space-y-6">
      {/* Header */}
      <div>
        <h1 className="text-3xl font-bold flex items-center gap-2">
          <Settings className="h-8 w-8" />
          Settings
        </h1>
        <p className="text-muted-foreground mt-1">
          Manage your Warren instance settings
        </p>
      </div>

      <Tabs defaultValue="tags" className="space-y-4">
        <TabsList>
          <TabsTrigger value="tags">Tags</TabsTrigger>
          {/* Add more tabs here as needed */}
        </TabsList>

        <TabsContent value="tags" className="space-y-4">
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <Tag className="h-5 w-5" />
                Tag Management
              </CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              {/* Create new tag */}
              <div className="flex gap-2">
                <Input
                  placeholder="Enter new tag name..."
                  value={newTagName}
                  onChange={(e) => setNewTagName(e.target.value)}
                  onKeyDown={(e) => {
                    if (e.key === "Enter") {
                      handleCreateTag();
                    }
                  }}
                  disabled={createLoading}
                />
                <Button
                  onClick={handleCreateTag}
                  disabled={!newTagName.trim() || createLoading}
                  className="flex items-center gap-2"
                >
                  <Plus className="h-4 w-4" />
                  Create Tag
                </Button>
              </div>

              {/* Tags list */}
              {tagsLoading ? (
                <div className="text-center py-8 text-muted-foreground">
                  Loading tags...
                </div>
              ) : tags.length === 0 ? (
                <div className="text-center py-8">
                  <Tag className="h-12 w-12 mx-auto mb-3 text-muted-foreground" />
                  <p className="text-muted-foreground">No tags created yet</p>
                  <p className="text-sm text-muted-foreground mt-1">
                    Create your first tag to start organizing alerts and tickets
                  </p>
                </div>
              ) : (
                <div className="space-y-2">
                  {tags.map((tag) => (
                    <div
                      key={tag.id}
                      className="flex items-start justify-between p-3 rounded-lg border bg-card hover:bg-muted/50 transition-colors"
                    >
                      <div className="flex-1 space-y-2">
                        <div className="flex items-center gap-3">
                          <Badge 
                            variant="secondary" 
                            className={`text-sm ${tag.color || generateTagColor(tag.name)}`}
                          >
                            {tag.name}
                          </Badge>
                          <div className="flex items-center gap-4 text-xs text-muted-foreground">
                            <span className="flex items-center gap-1">
                              <Calendar className="h-3 w-3" />
                              Created {formatRelativeTime(tag.createdAt)}
                            </span>
                            {tag.updatedAt !== tag.createdAt && (
                              <span className="flex items-center gap-1">
                                <Clock className="h-3 w-3" />
                                Updated {formatRelativeTime(tag.updatedAt)}
                              </span>
                            )}
                          </div>
                        </div>
                        {tag.description && (
                          <p className="text-sm text-muted-foreground ml-1">
                            {tag.description}
                          </p>
                        )}
                      </div>
                      <div className="flex items-center gap-1">
                        <Button
                          variant="ghost"
                          size="sm"
                          onClick={() => handleEditTag(tag)}
                          className="text-muted-foreground hover:text-foreground"
                        >
                          <Edit className="h-4 w-4" />
                        </Button>
                        <Button
                          variant="ghost"
                          size="sm"
                          onClick={() => setTagToDelete(tag.name)}
                          className="text-destructive hover:text-destructive"
                        >
                          <Trash2 className="h-4 w-4" />
                        </Button>
                      </div>
                    </div>
                  ))}
                </div>
              )}
            </CardContent>
          </Card>
        </TabsContent>
      </Tabs>

      {/* Delete confirmation dialog */}
      <AlertDialog
        open={!!tagToDelete}
        onOpenChange={(open) => !open && setTagToDelete(null)}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Delete Tag</AlertDialogTitle>
            <AlertDialogDescription>
              Are you sure you want to delete the tag "{tagToDelete}"? This action
              cannot be undone. The tag will be removed from all alerts and tickets.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction
              onClick={handleDeleteTag}
              disabled={deleteLoading}
              className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
            >
              Delete
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      {/* Edit tag modal */}
      <EditTagModal
        tag={tagToEdit}
        isOpen={isEditModalOpen}
        onClose={handleEditModalClose}
        onSuccess={handleEditSuccess}
      />
    </div>
  );
}
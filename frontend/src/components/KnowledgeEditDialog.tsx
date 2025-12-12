import { useState, useEffect } from "react";
import { useMutation } from "@apollo/client";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { CREATE_KNOWLEDGE, UPDATE_KNOWLEDGE } from "@/lib/graphql/queries";
import { Knowledge } from "@/lib/types";
import { Loader2 } from "lucide-react";

interface KnowledgeEditDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  mode: "create" | "edit";
  topic: string;
  knowledge?: Knowledge;
  onSuccess: () => void;
}

export function KnowledgeEditDialog({
  open,
  onOpenChange,
  mode,
  topic,
  knowledge,
  onSuccess,
}: KnowledgeEditDialogProps) {
  const [formData, setFormData] = useState({
    topic: topic,
    slug: "",
    name: "",
    content: "",
  });
  const [errors, setErrors] = useState<Record<string, string>>({});

  const [createKnowledge, { loading: creating }] = useMutation(
    CREATE_KNOWLEDGE,
    {
      onCompleted: () => {
        onSuccess();
        onOpenChange(false);
        resetForm();
      },
      onError: (error) => {
        setErrors({ submit: error.message });
      },
    }
  );

  const [updateKnowledge, { loading: updating }] = useMutation(
    UPDATE_KNOWLEDGE,
    {
      onCompleted: () => {
        onSuccess();
        onOpenChange(false);
        resetForm();
      },
      onError: (error) => {
        setErrors({ submit: error.message });
      },
    }
  );

  const loading = creating || updating;

  useEffect(() => {
    if (mode === "edit" && knowledge) {
      setFormData({
        topic: knowledge.topic,
        slug: knowledge.slug,
        name: knowledge.name,
        content: knowledge.content,
      });
    } else if (mode === "create") {
      setFormData({
        topic: topic,
        slug: "",
        name: "",
        content: "",
      });
    }
    setErrors({});
  }, [mode, knowledge, topic, open]);

  const resetForm = () => {
    setFormData({
      topic: topic,
      slug: "",
      name: "",
      content: "",
    });
    setErrors({});
  };

  const validateForm = () => {
    const newErrors: Record<string, string> = {};

    if (mode === "create" && !formData.slug.trim()) {
      newErrors.slug = "Slug is required";
    }
    if (!formData.name.trim()) {
      newErrors.name = "Name is required";
    }
    if (formData.name.length > 100) {
      newErrors.name = "Name must be 100 characters or less";
    }
    if (!formData.content.trim()) {
      newErrors.content = "Content is required";
    }

    // Check for Firestore forbidden characters
    const forbiddenChars = ["/", "\\"];
    const forbiddenSequences = ["__"];

    if (mode === "create") {
      for (const char of forbiddenChars) {
        if (formData.slug.includes(char)) {
          newErrors.slug = `Slug cannot contain '${char}'`;
          break;
        }
      }
      for (const seq of forbiddenSequences) {
        if (formData.slug.includes(seq)) {
          newErrors.slug = `Slug cannot contain '${seq}'`;
          break;
        }
      }
      if (formData.slug.startsWith(".") || formData.slug.endsWith(".")) {
        newErrors.slug = "Slug cannot start or end with '.'";
      }
    }

    setErrors(newErrors);
    return Object.keys(newErrors).length === 0;
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();

    if (!validateForm()) {
      return;
    }

    const input = {
      topic: formData.topic,
      slug: formData.slug,
      name: formData.name,
      content: formData.content,
    };

    if (mode === "create") {
      await createKnowledge({ variables: { input } });
    } else {
      await updateKnowledge({ variables: { input } });
    }
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-2xl max-h-[90vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle>
            {mode === "create" ? "Create Knowledge" : "Edit Knowledge"}
          </DialogTitle>
          <DialogDescription>
            {mode === "create"
              ? "Add a new knowledge item to this topic."
              : "Update the knowledge item."}
          </DialogDescription>
        </DialogHeader>

        <form onSubmit={handleSubmit} className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="topic">Topic</Label>
            <Input
              id="topic"
              value={formData.topic}
              disabled
              className="bg-muted"
            />
          </div>

          <div className="space-y-2">
            <Label htmlFor="slug">
              Slug {mode === "create" && <span className="text-red-500">*</span>}
            </Label>
            <Input
              id="slug"
              value={formData.slug}
              onChange={(e) =>
                setFormData({ ...formData, slug: e.target.value })
              }
              disabled={mode === "edit"}
              className={mode === "edit" ? "bg-muted" : ""}
              placeholder="unique-identifier"
            />
            {mode === "edit" && (
              <p className="text-xs text-muted-foreground">
                Slug cannot be changed after creation
              </p>
            )}
            {errors.slug && (
              <p className="text-sm text-red-600">{errors.slug}</p>
            )}
          </div>

          <div className="space-y-2">
            <Label htmlFor="name">
              Name <span className="text-red-500">*</span>
            </Label>
            <Input
              id="name"
              value={formData.name}
              onChange={(e) =>
                setFormData({ ...formData, name: e.target.value })
              }
              placeholder="Short descriptive name"
              maxLength={100}
            />
            <p className="text-xs text-muted-foreground">
              {formData.name.length}/100 characters
            </p>
            {errors.name && (
              <p className="text-sm text-red-600">{errors.name}</p>
            )}
          </div>

          <div className="space-y-2">
            <Label htmlFor="content">
              Content <span className="text-red-500">*</span>
            </Label>
            <Textarea
              id="content"
              value={formData.content}
              onChange={(e) =>
                setFormData({ ...formData, content: e.target.value })
              }
              placeholder="Knowledge content..."
              rows={10}
              className="font-mono text-sm"
            />
            {errors.content && (
              <p className="text-sm text-red-600">{errors.content}</p>
            )}
          </div>

          {errors.submit && (
            <div className="bg-red-50 border border-red-200 text-red-700 px-4 py-3 rounded">
              {errors.submit}
            </div>
          )}

          <DialogFooter>
            <Button
              type="button"
              variant="outline"
              onClick={() => onOpenChange(false)}
              disabled={loading}>
              Cancel
            </Button>
            <Button type="submit" disabled={loading}>
              {loading && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
              {mode === "create" ? "Create" : "Update"}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}

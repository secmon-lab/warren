import { useState, useEffect } from "react";
import { useQuery, useMutation } from "@apollo/client";
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
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Badge } from "@/components/ui/badge";
import { useToast } from "@/hooks/use-toast";
import { UPDATE_TAG, GET_AVAILABLE_TAG_COLOR_NAMES } from "@/lib/graphql/queries";
import { TagMetadata } from "@/lib/types";
import { colorClassToName, colorNameToClass } from "@/lib/tag-colors";

interface EditTagModalProps {
  tag: TagMetadata | null;
  isOpen: boolean;
  onClose: () => void;
  onSuccess: () => void;
}

export default function EditTagModal({ tag, isOpen, onClose, onSuccess }: EditTagModalProps) {
  const { toast } = useToast();
  const [name, setName] = useState("");
  const [description, setDescription] = useState("");
  const [color, setColor] = useState("");

  const { data: colorNamesData } = useQuery(GET_AVAILABLE_TAG_COLOR_NAMES);
  const availableColorNames = colorNamesData?.availableTagColorNames || [];

  const [updateTag, { loading: updateLoading }] = useMutation(UPDATE_TAG, {
    onCompleted: () => {
      toast({
        title: "Tag Updated",
        description: "The tag has been updated successfully.",
      });
      onSuccess();
      onClose();
    },
    onError: (error) => {
      toast({
        title: "Error",
        description: `Failed to update tag: ${error.message}`,
        variant: "destructive",
      });
    },
  });

  useEffect(() => {
    if (tag && isOpen) {
      setName(tag.name);
      setDescription(tag.description || "");
      setColor(colorClassToName(tag.color));
    }
  }, [tag, isOpen]);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!tag || !name.trim() || !color) return;

    await updateTag({
      variables: {
        input: {
          id: tag.id,
          name: name.trim(),
          color, // Send color name to backend
          description: description.trim() || undefined,
        },
      },
    });
  };

  const handleClose = () => {
    setName("");
    setDescription("");
    setColor("");
    onClose();
  };

  if (!tag) return null;

  return (
    <Dialog open={isOpen} onOpenChange={handleClose}>
      <DialogContent className="sm:max-w-[425px]">
        <form onSubmit={handleSubmit}>
          <DialogHeader>
            <DialogTitle>Edit Tag</DialogTitle>
            <DialogDescription>
              Update the name, color, and description for this tag.
            </DialogDescription>
          </DialogHeader>
          
          <div className="grid gap-4 py-4">
            <div className="grid grid-cols-4 items-center gap-4">
              <Label htmlFor="name" className="text-right">
                Name
              </Label>
              <Input
                id="name"
                value={name}
                onChange={(e) => setName(e.target.value)}
                className="col-span-3"
                placeholder="Enter tag name..."
                required
              />
            </div>
            
            <div className="grid grid-cols-4 items-center gap-4">
              <Label htmlFor="color" className="text-right">
                Color
              </Label>
              <div className="col-span-3 space-y-2">
                <Select value={color} onValueChange={setColor} required>
                  <SelectTrigger>
                    <SelectValue placeholder="Select a color" />
                  </SelectTrigger>
                  <SelectContent className="max-h-48">
                    {availableColorNames.map((colorName: string) => (
                      <SelectItem key={colorName} value={colorName}>
                        <div className="flex items-center gap-2">
                          <Badge className={colorNameToClass(colorName)} variant="secondary">
                            Sample
                          </Badge>
                          <span className="text-sm capitalize">
                            {colorName}
                          </span>
                        </div>
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
                {color && (
                  <div className="flex items-center gap-2">
                    <span className="text-sm text-muted-foreground">Preview:</span>
                    <Badge className={colorNameToClass(color)} variant="secondary">
                      {name || "Tag Name"}
                    </Badge>
                  </div>
                )}
              </div>
            </div>
            
            <div className="grid grid-cols-4 items-start gap-4">
              <Label htmlFor="description" className="text-right pt-2">
                Description
              </Label>
              <Textarea
                id="description"
                value={description}
                onChange={(e) => setDescription(e.target.value)}
                className="col-span-3 min-h-[160px]"
                placeholder="Optional description for this tag..."
              />
            </div>
          </div>
          
          <DialogFooter>
            <Button
              type="button"
              variant="outline"
              onClick={handleClose}
              disabled={updateLoading}
            >
              Cancel
            </Button>
            <Button
              type="submit"
              disabled={!name.trim() || !color || updateLoading}
            >
              {updateLoading ? "Updating..." : "Update Tag"}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
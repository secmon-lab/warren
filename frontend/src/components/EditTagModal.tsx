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

  // Helper function to convert Tailwind class to color name
  const colorClassToName = (colorClass: string): string => {
    const colorMap: Record<string, string> = {
      "bg-red-100 text-red-800": "red",
      "bg-orange-100 text-orange-800": "orange",
      "bg-amber-100 text-amber-800": "amber",
      "bg-yellow-100 text-yellow-800": "yellow",
      "bg-lime-100 text-lime-800": "lime",
      "bg-green-100 text-green-800": "green",
      "bg-emerald-100 text-emerald-800": "emerald",
      "bg-teal-100 text-teal-800": "teal",
      "bg-cyan-100 text-cyan-800": "cyan",
      "bg-sky-100 text-sky-800": "sky",
      "bg-blue-100 text-blue-800": "blue",
      "bg-indigo-100 text-indigo-800": "indigo",
      "bg-violet-100 text-violet-800": "violet",
      "bg-purple-100 text-purple-800": "purple",
      "bg-fuchsia-100 text-fuchsia-800": "fuchsia",
      "bg-pink-100 text-pink-800": "pink",
      "bg-rose-100 text-rose-800": "rose",
      "bg-slate-100 text-slate-800": "slate",
      "bg-gray-100 text-gray-800": "gray",
      "bg-zinc-100 text-zinc-800": "zinc",
    };
    return colorMap[colorClass] || "gray";
  };

  // Helper function to convert color name to Tailwind class
  const colorNameToClass = (colorName: string): string => {
    const classMap: Record<string, string> = {
      red: "bg-red-100 text-red-800",
      orange: "bg-orange-100 text-orange-800",
      amber: "bg-amber-100 text-amber-800",
      yellow: "bg-yellow-100 text-yellow-800",
      lime: "bg-lime-100 text-lime-800",
      green: "bg-green-100 text-green-800",
      emerald: "bg-emerald-100 text-emerald-800",
      teal: "bg-teal-100 text-teal-800",
      cyan: "bg-cyan-100 text-cyan-800",
      sky: "bg-sky-100 text-sky-800",
      blue: "bg-blue-100 text-blue-800",
      indigo: "bg-indigo-100 text-indigo-800",
      violet: "bg-violet-100 text-violet-800",
      purple: "bg-purple-100 text-purple-800",
      fuchsia: "bg-fuchsia-100 text-fuchsia-800",
      pink: "bg-pink-100 text-pink-800",
      rose: "bg-rose-100 text-rose-800",
      slate: "bg-slate-100 text-slate-800",
      gray: "bg-gray-100 text-gray-800",
      zinc: "bg-zinc-100 text-zinc-800",
    };
    return classMap[colorName] || "bg-gray-100 text-gray-800";
  };

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

    try {
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
    } catch (error) {
      console.error("Error updating tag:", error);
    }
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
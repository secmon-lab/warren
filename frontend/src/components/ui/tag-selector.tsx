import React, { useState, useRef, useEffect } from "react";
import { useMutation, useQuery } from "@apollo/client";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { X, Plus } from "lucide-react";
import { GET_TAGS, CREATE_TAG } from "@/lib/graphql/queries";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";
import { cn } from "@/lib/utils";
import { TagMetadata } from "@/lib/types";
import { TagObject } from "@/lib/graphql/generated";

interface TagSelectorProps {
  selectedTags: string[];
  onTagsChange: (tags: string[]) => void;
  // Optional prop to pass existing tagObjects for efficiency
  existingTagObjects?: TagObject[];
  disabled?: boolean;
  className?: string;
}

export function TagSelector({
  selectedTags,
  onTagsChange,
  disabled = false,
  className,
}: TagSelectorProps) {
  const [isOpen, setIsOpen] = useState(false);
  const [searchValue, setSearchValue] = useState("");
  const inputRef = useRef<HTMLInputElement>(null);

  const { data: tagsData, refetch: refetchTags } = useQuery(GET_TAGS);
  const [createTag, { loading: createLoading }] = useMutation(CREATE_TAG);

  const allTags: TagMetadata[] = tagsData?.tags || [];
  const tagsByName = new Map(allTags.map((t: TagMetadata) => [t.name, t]));

  // Filter tags based on search - optimized with Set for O(1) lookup
  const selectedTagsSet = new Set(selectedTags);
  const filteredTags = allTags.filter(
    (tag: TagMetadata) =>
      tag.name.toLowerCase().includes(searchValue.toLowerCase()) &&
      !selectedTagsSet.has(tag.name)
  );

  const handleAddTag = (tag: string) => {
    if (!selectedTagsSet.has(tag)) {
      onTagsChange([...selectedTags, tag]);
    }
    setSearchValue("");
  };

  const handleRemoveTag = (tag: string) => {
    onTagsChange(selectedTags.filter((t) => t !== tag));
  };

  const handleCreateTag = async () => {
    if (!searchValue.trim()) return;
    
    try {
      await createTag({
        variables: { name: searchValue.trim() },
      });
      await refetchTags();
      handleAddTag(searchValue.trim());
    } catch (error) {
      console.error("Failed to create tag:", error);
    }
  };

  const handleKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
    if (e.key === "Enter") {
      e.preventDefault();
      if (filteredTags.length > 0) {
        handleAddTag(filteredTags[0].name);
      } else if (showCreateOption) {
        handleCreateTag();
      }
    }
  };

  useEffect(() => {
    if (isOpen && inputRef.current) {
      inputRef.current.focus();
    }
  }, [isOpen]);

  const showCreateOption = 
    searchValue.trim() && 
    !allTags.some((tag: TagMetadata) => tag.name.toLowerCase() === searchValue.toLowerCase().trim());

  return (
    <div className={cn("space-y-2", className)}>
      <div className="flex flex-wrap gap-2">
        {selectedTags.map((tag) => {
          // Try to get tag data from TagMetadata first (which has color)
          const tagData = tagsByName.get(tag);
          
          // Use color from full tag metadata if available, otherwise use default
          const colorClass = tagData?.color || "bg-gray-100 text-gray-800";
          return (
            <Badge key={tag} className={`text-sm ${colorClass}`}>
              {tag}
              {!disabled && (
                <button
                  onClick={() => handleRemoveTag(tag)}
                  className="ml-1 hover:text-destructive"
                >
                  <X className="h-3 w-3" />
                </button>
              )}
            </Badge>
          );
        })}
        {!disabled && (
          <Popover open={isOpen} onOpenChange={setIsOpen}>
            <PopoverTrigger asChild>
              <Button
                variant="outline"
                size="sm"
                className="h-7 px-2"
              >
                <Plus className="h-3 w-3 mr-1" />
                Add Tag
              </Button>
            </PopoverTrigger>
            <PopoverContent className="w-64 p-2" align="start">
              <div className="space-y-2">
                <Input
                  ref={inputRef}
                  placeholder="Search or create tag..."
                  value={searchValue}
                  onChange={(e) => setSearchValue(e.target.value)}
                  onKeyDown={handleKeyDown}
                  className="h-8"
                />
                <div className="max-h-48 overflow-y-auto">
                  {filteredTags.length > 0 && (
                    <div className="space-y-1">
                      {filteredTags.map((tag: TagMetadata) => (
                        <button
                          key={tag.name}
                          onClick={() => handleAddTag(tag.name)}
                          className="w-full text-left px-2 py-1 text-sm hover:bg-accent rounded"
                        >
                          {tag.name}
                        </button>
                      ))}
                    </div>
                  )}
                  {showCreateOption && (
                    <button
                      onClick={handleCreateTag}
                      disabled={createLoading}
                      className="w-full text-left px-2 py-1 text-sm hover:bg-accent rounded text-muted-foreground"
                    >
                      {createLoading ? (
                        "Creating..."
                      ) : (
                        <>
                          <Plus className="inline h-3 w-3 mr-1" />
                          Create "{searchValue.trim()}"
                        </>
                      )}
                    </button>
                  )}
                  {!filteredTags.length && !showCreateOption && searchValue && (
                    <p className="text-sm text-muted-foreground px-2 py-1">
                      No tags found
                    </p>
                  )}
                </div>
              </div>
            </PopoverContent>
          </Popover>
        )}
      </div>
    </div>
  );
}
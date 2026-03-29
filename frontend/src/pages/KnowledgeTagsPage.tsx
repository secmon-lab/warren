import { useState } from "react";
import { useQuery, useMutation } from "@apollo/client";
import {
  GET_KNOWLEDGE_TAGS,
  CREATE_KNOWLEDGE_TAG,
  UPDATE_KNOWLEDGE_TAG,
  DELETE_KNOWLEDGE_TAG,
} from "@/lib/graphql/queries";

export default function KnowledgeTagsPage() {
  const { data, loading, refetch } = useQuery(GET_KNOWLEDGE_TAGS);
  const tags = data?.knowledgeTags || [];

  const [newName, setNewName] = useState("");
  const [newDesc, setNewDesc] = useState("");
  const [editingId, setEditingId] = useState<string | null>(null);
  const [editName, setEditName] = useState("");
  const [editDesc, setEditDesc] = useState("");

  const [createTag] = useMutation(CREATE_KNOWLEDGE_TAG);
  const [updateTag] = useMutation(UPDATE_KNOWLEDGE_TAG);
  const [deleteTag] = useMutation(DELETE_KNOWLEDGE_TAG);

  const handleCreate = async () => {
    if (!newName) return;
    await createTag({
      variables: {
        input: { name: newName.toLowerCase().trim(), description: newDesc },
      },
    });
    setNewName("");
    setNewDesc("");
    refetch();
  };

  const handleUpdate = async (id: string) => {
    await updateTag({
      variables: {
        input: { id, name: editName, description: editDesc },
      },
    });
    setEditingId(null);
    refetch();
  };

  const handleDelete = async (id: string) => {
    if (!confirm("Delete this tag?")) return;
    await deleteTag({ variables: { id } });
    refetch();
  };

  const startEditing = (tag: { id: string; name: string; description: string }) => {
    setEditingId(tag.id);
    setEditName(tag.name);
    setEditDesc(tag.description);
  };

  if (loading) return <p className="text-sm text-gray-500">Loading...</p>;

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-bold">Knowledge Tags</h1>

      {/* Create new tag */}
      <div className="rounded-lg border p-4">
        <h2 className="text-sm font-semibold">Create New Tag</h2>
        <div className="mt-2 flex gap-2">
          <input
            type="text"
            value={newName}
            onChange={(e) => setNewName(e.target.value)}
            className="rounded-md border border-gray-300 px-3 py-1 text-sm"
            placeholder="Tag name"
          />
          <input
            type="text"
            value={newDesc}
            onChange={(e) => setNewDesc(e.target.value)}
            className="flex-1 rounded-md border border-gray-300 px-3 py-1 text-sm"
            placeholder="Description (optional)"
          />
          <button
            onClick={handleCreate}
            className="rounded-md bg-blue-600 px-4 py-1 text-sm font-medium text-white hover:bg-blue-700"
          >
            Create
          </button>
        </div>
      </div>

      {/* Tag list */}
      <div className="space-y-2">
        {tags.map((tag: { id: string; name: string; description: string; createdAt: string }) => (
          <div key={tag.id} className="flex items-center justify-between rounded border p-3">
            {editingId === tag.id ? (
              <div className="flex flex-1 gap-2">
                <input
                  type="text"
                  value={editName}
                  onChange={(e) => setEditName(e.target.value)}
                  className="rounded-md border border-gray-300 px-2 py-1 text-sm"
                />
                <input
                  type="text"
                  value={editDesc}
                  onChange={(e) => setEditDesc(e.target.value)}
                  className="flex-1 rounded-md border border-gray-300 px-2 py-1 text-sm"
                />
                <button
                  onClick={() => handleUpdate(tag.id)}
                  className="text-sm text-blue-600 hover:text-blue-800"
                >
                  Save
                </button>
                <button
                  onClick={() => setEditingId(null)}
                  className="text-sm text-gray-500 hover:text-gray-700"
                >
                  Cancel
                </button>
              </div>
            ) : (
              <>
                <div>
                  <span className="font-medium">{tag.name}</span>
                  {tag.description && (
                    <span className="ml-2 text-sm text-gray-500">{tag.description}</span>
                  )}
                </div>
                <div className="flex gap-2">
                  <button
                    onClick={() => startEditing(tag)}
                    className="text-sm text-blue-600 hover:text-blue-800"
                  >
                    Edit
                  </button>
                  <button
                    onClick={() => handleDelete(tag.id)}
                    className="text-sm text-red-500 hover:text-red-700"
                  >
                    Delete
                  </button>
                </div>
              </>
            )}
          </div>
        ))}
        {tags.length === 0 && (
          <p className="text-sm text-gray-500">No tags yet.</p>
        )}
      </div>
    </div>
  );
}
